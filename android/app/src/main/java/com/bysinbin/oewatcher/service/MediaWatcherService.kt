package com.bysinbin.oewatcher.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.media.AudioManager
import android.media.MediaMetadata
import android.media.session.MediaController
import android.media.session.MediaSession
import android.media.session.MediaSessionManager
import android.media.session.PlaybackState
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log
import android.view.KeyEvent
import android.widget.Toast
import androidx.core.app.NotificationCompat
import com.bysinbin.oewatcher.MainActivity
import com.bysinbin.oewatcher.data.MediaRepository
import com.bysinbin.oewatcher.engine.BlacklistMatcher
import com.bysinbin.oewatcher.model.MediaTrackInfo
import com.bysinbin.oewatcher.model.SkipEvent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class MediaWatcherService : NotificationListenerService() {

    private lateinit var mediaSessionManager: MediaSessionManager
    private val activeControllers = mutableMapOf<String, MediaController>()
    private val mainHandler = Handler(Looper.getMainLooper())
    private var lastSkippedKey = ""
    private var lastSkipTimestamp = 0L

    private val sessionsListener = MediaSessionManager.OnActiveSessionsChangedListener { controllers ->
        updateActiveSessions(controllers)
    }

    override fun onCreate() {
        super.onCreate()
        MediaRepository.init(applicationContext)
        mediaSessionManager = getSystemService(Context.MEDIA_SESSION_SERVICE) as MediaSessionManager
        createNotificationChannels()

        kotlinx.coroutines.CoroutineScope(kotlinx.coroutines.Dispatchers.Main).launch {
            MediaRepository.playerActions.collect { action ->
                when (action) {
                    "SKIP_NEXT" -> {
                        val controller = activeControllers.values.firstOrNull()
                        try { controller?.transportControls?.skipToNext() } catch (_: Exception) {}
                        sendMediaNextKeyEvent()
                    }
                    "PLAY_PAUSE" -> {
                        val controller = activeControllers.values.firstOrNull()
                        val state = controller?.playbackState?.state
                        try {
                            if (state == PlaybackState.STATE_PLAYING) {
                                controller.transportControls.pause()
                            } else {
                                controller?.transportControls?.play() ?: sendMediaPlayPauseKeyEvent()
                            }
                        } catch (_: Exception) {
                            sendMediaPlayPauseKeyEvent()
                        }
                    }
                }
            }
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForegroundSticky()
        refreshSessionsAndNotifications()
        return START_STICKY
    }

    override fun onListenerConnected() {
        super.onListenerConnected()
        Log.i(TAG, "🟢 OE Watcher Medya Dinleyici Servisi Bağlandı!")
        MediaRepository.setServiceRunning(true)
        MediaRepository.setPermissionGranted(true)

        startForegroundSticky()

        try {
            val component = ComponentName(this, MediaWatcherService::class.java)
            try { mediaSessionManager.removeOnActiveSessionsChangedListener(sessionsListener) } catch (_: Exception) {}
            mediaSessionManager.addOnActiveSessionsChangedListener(sessionsListener, component)
        } catch (e: Exception) {
            Log.e(TAG, "Sessions listener kayıt hatası: ${e.message}")
        }

        refreshSessionsAndNotifications()
    }

    private fun refreshSessionsAndNotifications() {
        try {
            val component = ComponentName(this, MediaWatcherService::class.java)
            val controllers = mediaSessionManager.getActiveSessions(component)
            if (!controllers.isNullOrEmpty()) {
                updateActiveSessions(controllers)
            }
        } catch (e: Exception) {
            Log.d(TAG, "Aktif oturumları alma hatası: ${e.message}")
        }

        try {
            val activeNotifs = activeNotifications
            if (activeNotifs != null) {
                for (sbn in activeNotifs) {
                    inspectMediaNotification(sbn)
                }
            }
        } catch (e: Exception) {
            Log.d(TAG, "Bildirimleri tarama hatası: ${e.message}")
        }
    }

    override fun onListenerDisconnected() {
        super.onListenerDisconnected()
        Log.w(TAG, "🔴 OE Watcher Medya Dinleyici Servisi Bağlantısı Kesildi.")
        MediaRepository.setServiceRunning(false)
        try {
            mediaSessionManager.removeOnActiveSessionsChangedListener(sessionsListener)
        } catch (_: Exception) {}
        activeControllers.clear()

        // Attempt instant auto-rebind
        try {
            val component = ComponentName(this, MediaWatcherService::class.java)
            requestRebind(component)
        } catch (_: Exception) {}
    }

    private fun startForegroundSticky() {
        try {
            val openAppIntent = Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP
            }
            val pendingIntent = PendingIntent.getActivity(
                this, 0, openAppIntent,
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )

            val notification = NotificationCompat.Builder(this, CHANNEL_STICKY_ID)
                .setSmallIcon(android.R.drawable.ic_media_play)
                .setContentTitle("OE Watcher Devrede")
                .setContentText("Medya akışları arka planda canlı filtreleniyor")
                .setPriority(NotificationCompat.PRIORITY_MIN)
                .setCategory(Notification.CATEGORY_SERVICE)
                .setOngoing(true)
                .setContentIntent(pendingIntent)
                .build()

            startForeground(STICKY_NOTIFICATION_ID, notification)
        } catch (e: Exception) {
            Log.e(TAG, "startForeground hatası: ${e.message}")
        }
    }

    private fun updateActiveSessions(controllers: List<MediaController>?) {
        if (controllers == null) return
        for (controller in controllers) {
            val pkg = controller.packageName ?: continue
            activeControllers[pkg] = controller

            controller.registerCallback(object : MediaController.Callback() {
                override fun onMetadataChanged(metadata: MediaMetadata?) {
                    handleTrackUpdate(controller, metadata, controller.playbackState)
                }

                override fun onPlaybackStateChanged(state: PlaybackState?) {
                    handleTrackUpdate(controller, controller.metadata, state)
                }
            }, mainHandler)

            // Initial inspect
            handleTrackUpdate(controller, controller.metadata, controller.playbackState)
        }
    }

    private fun handleTrackUpdate(
        controller: MediaController,
        metadata: MediaMetadata?,
        playbackState: PlaybackState?
    ) {
        val isPlaying = playbackState?.state == PlaybackState.STATE_PLAYING ||
                playbackState?.state == PlaybackState.STATE_BUFFERING

        val title = metadata?.getString(MediaMetadata.METADATA_KEY_TITLE)
            ?: metadata?.getString(MediaMetadata.METADATA_KEY_DISPLAY_TITLE)
            ?: metadata?.description?.title?.toString()
            ?: ""

        val artist = metadata?.getString(MediaMetadata.METADATA_KEY_ARTIST)
            ?: metadata?.getString(MediaMetadata.METADATA_KEY_DISPLAY_SUBTITLE)
            ?: metadata?.description?.subtitle?.toString()
            ?: metadata?.getString(MediaMetadata.METADATA_KEY_ALBUM_ARTIST)
            ?: metadata?.getString(MediaMetadata.METADATA_KEY_AUTHOR)
            ?: ""

        val album = metadata?.getString(MediaMetadata.METADATA_KEY_ALBUM)
            ?: metadata?.description?.description?.toString()
            ?: ""

        val pkg = controller.packageName ?: ""
        val appName = getFriendlyAppName(pkg)

        if (title.isBlank() && artist.isBlank()) return

        Log.d(TAG, "Oynatılan Parça [${appName}]: $artist - $title (isPlaying=$isPlaying)")

        val trackInfo = MediaTrackInfo(
            app = appName,
            packageName = pkg,
            title = title,
            artist = artist,
            album = album,
            isPlaying = isPlaying
        )

        MediaRepository.updateCurrentTrack(trackInfo)

        if (isPlaying) {
            evaluateBlacklist(controller, trackInfo, null)
        }
    }

    private fun evaluateBlacklist(
        controller: MediaController?,
        track: MediaTrackInfo,
        fallbackNotification: StatusBarNotification?
    ) {
        val initialMatch = BlacklistMatcher.match(track.artist, track.title)
        val albumMatch = if (initialMatch == null && track.album.isNotBlank()) {
            BlacklistMatcher.match(track.album, track.title)
        } else null

        val match = initialMatch ?: albumMatch ?: return

        val now = System.currentTimeMillis()
        val trackKey = "${track.packageName}:::${track.artist}:::${track.title}"

        // Anti-bounce guard: prevent skipping multiple times within 2 seconds for the exact same track
        if (trackKey == lastSkippedKey && (now - lastSkipTimestamp) < 2000) {
            return
        }

        lastSkippedKey = trackKey
        lastSkipTimestamp = now

        Log.w(TAG, "🚨 Kara Liste Tespiti: '${match.personName}' tespit edildi -> ${track.artist} - ${track.title}")

        if (MediaRepository.autoSkipEnabled.value) {
            // Step 1: TransportControls.skipToNext()
            try {
                controller?.transportControls?.skipToNext()
                Log.i(TAG, "✅ transportControls.skipToNext() başarıyla gönderildi!")
            } catch (e: Exception) {
                Log.e(TAG, "transportControls.skipToNext() hatası: ${e.message}")
            }

            // Step 2: Try Notification Actions (Next Button)
            fallbackNotification?.let { sbn ->
                triggerNotificationNextAction(sbn)
            }

            // Step 3: Broadcast KEYCODE_MEDIA_NEXT as universal fallback
            sendMediaNextKeyEvent()

            // Step 4: Record history event
            val event = SkipEvent(
                app = track.app,
                artist = track.artist.ifBlank { "Bilinmeyen Sanatçı" },
                title = track.title.ifBlank { "İsimsiz Parça" },
                matchedPerson = match.personName,
                success = true
            )
            MediaRepository.recordSkip(event)

            // Step 5: Visual Toast HUD & Notification
            mainHandler.post {
                Toast.makeText(
                    applicationContext,
                    "⏩ OE Watcher: ${match.personName} otomatik geçildi!",
                    Toast.LENGTH_SHORT
                ).show()
            }
            showSkippedNotification(match.personName, track.title)
        }
    }

    override fun onNotificationPosted(sbn: StatusBarNotification?) {
        super.onNotificationPosted(sbn)
        if (sbn != null) {
            inspectMediaNotification(sbn)
        }
    }

    private fun inspectMediaNotification(sbn: StatusBarNotification) {
        val pkg = sbn.packageName ?: return

        // Inspect media notifications
        val isMediaPackage = pkg.contains("spotify") ||
                pkg.contains("music") ||
                pkg.contains("youtube") ||
                pkg.contains("deezer") ||
                pkg.contains("fizy") ||
                pkg.contains("apple") ||
                pkg.contains("tidal") ||
                pkg.contains("soundcloud") ||
                sbn.notification.extras?.containsKey(Notification.EXTRA_MEDIA_SESSION) == true

        if (!isMediaPackage) return

        val extras = sbn.notification.extras ?: return
        val title = extras.getCharSequence(Notification.EXTRA_TITLE)?.toString() ?: ""
        val text = extras.getCharSequence(Notification.EXTRA_TEXT)?.toString() ?: ""
        val subText = extras.getCharSequence(Notification.EXTRA_SUB_TEXT)?.toString() ?: ""

        // Extract MediaController from token
        @Suppress("DEPRECATION")
        val token = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            extras.getParcelable(Notification.EXTRA_MEDIA_SESSION, MediaSession.Token::class.java)
        } else {
            extras.getParcelable(Notification.EXTRA_MEDIA_SESSION) as? MediaSession.Token
        }

        val controller = if (token != null) {
            try { MediaController(applicationContext, token) } catch (_: Exception) { null }
        } else {
            activeControllers[pkg] ?: try {
                val component = ComponentName(this, MediaWatcherService::class.java)
                mediaSessionManager.getActiveSessions(component).firstOrNull { it.packageName == pkg }
            } catch (_: Exception) { null }
        }

        if (controller != null) {
            activeControllers[pkg] = controller
            controller.registerCallback(object : MediaController.Callback() {
                override fun onMetadataChanged(metadata: MediaMetadata?) {
                    handleTrackUpdate(controller, metadata, controller.playbackState)
                }
                override fun onPlaybackStateChanged(state: PlaybackState?) {
                    handleTrackUpdate(controller, controller.metadata, state)
                }
            }, mainHandler)
            handleTrackUpdate(controller, controller.metadata, controller.playbackState)
        }

        // Also direct-evaluate notification contents
        if (title.isNotBlank() || text.isNotBlank()) {
            val candidateArtist = text.ifBlank { subText }
            val match = BlacklistMatcher.match(candidateArtist, title)
                ?: BlacklistMatcher.match(subText, title)
            if (match != null) {
                val track = MediaTrackInfo(
                    app = getFriendlyAppName(pkg),
                    packageName = pkg,
                    title = title,
                    artist = candidateArtist,
                    isPlaying = true
                )
                evaluateBlacklist(controller, track, sbn)
            }
        }
    }

    private fun triggerNotificationNextAction(sbn: StatusBarNotification) {
        val actions = sbn.notification.actions ?: return
        for (action in actions) {
            val titleStr = action.title?.toString()?.lowercase() ?: ""
            if (titleStr.contains("next") ||
                titleStr.contains("sonraki") ||
                titleStr.contains("iler") ||
                titleStr.contains("skip") ||
                titleStr.contains("atla") ||
                titleStr.contains("gec") ||
                titleStr.contains("geç")
            ) {
                try {
                    action.actionIntent.send()
                    Log.i(TAG, "✅ Bildirim 'Sonraki' aksiyonu başarıyla tetiklendi!")
                    return
                } catch (e: Exception) {
                    Log.e(TAG, "Bildirim aksiyon hatası: ${e.message}")
                }
            }
        }
    }

    private fun sendMediaNextKeyEvent() {
        try {
            val audioManager = getSystemService(Context.AUDIO_SERVICE) as? AudioManager ?: return
            val down = KeyEvent(KeyEvent.ACTION_DOWN, KeyEvent.KEYCODE_MEDIA_NEXT)
            val up = KeyEvent(KeyEvent.ACTION_UP, KeyEvent.KEYCODE_MEDIA_NEXT)
            audioManager.dispatchMediaKeyEvent(down)
            audioManager.dispatchMediaKeyEvent(up)
        } catch (e: Exception) {
            Log.e(TAG, "Media keyevent hatası: ${e.message}")
        }
    }

    private fun sendMediaPlayPauseKeyEvent() {
        try {
            val audioManager = getSystemService(Context.AUDIO_SERVICE) as? AudioManager ?: return
            val down = KeyEvent(KeyEvent.ACTION_DOWN, KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE)
            val up = KeyEvent(KeyEvent.ACTION_UP, KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE)
            audioManager.dispatchMediaKeyEvent(down)
            audioManager.dispatchMediaKeyEvent(up)
        } catch (e: Exception) {
            Log.e(TAG, "PlayPause keyevent hatası: ${e.message}")
        }
    }

    private fun getFriendlyAppName(pkg: String): String {
        return when {
            pkg.contains("spotify") -> "Spotify"
            pkg.contains("youtube.music") -> "YouTube Music"
            pkg.contains("youtube") -> "YouTube"
            pkg.contains("apple.android.music") -> "Apple Music"
            pkg.contains("deezer") -> "Deezer"
            pkg.contains("fizy") -> "Fizy"
            pkg.contains("soundcloud") -> "SoundCloud"
            else -> {
                try {
                    val pm = packageManager
                    val appInfo = pm.getApplicationInfo(pkg, 0)
                    pm.getApplicationLabel(appInfo).toString()
                } catch (_: Exception) {
                    pkg.substringAfterLast('.')
                }
            }
        }
    }

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val manager = getSystemService(NotificationManager::class.java) ?: return

            // 1. Sticky background status channel
            val stickyChannel = NotificationChannel(
                CHANNEL_STICKY_ID,
                "OE Watcher Servis Durumu",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Uygulamanın arka planda canlı kalmasını sağlayan sistem bildirimi"
                setShowBadge(false)
            }
            manager.createNotificationChannel(stickyChannel)

            // 2. Heads-up skip notification channel
            val skipChannel = NotificationChannel(
                CHANNEL_SKIP_ID,
                "Atlanan Şarkı Bildirimleri",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Kara listedeki şarkılar atlandığında çıkan bilgilendirme rozeti"
            }
            manager.createNotificationChannel(skipChannel)
        }
    }

    private fun showSkippedNotification(personName: String, trackTitle: String) {
        try {
            val notification = NotificationCompat.Builder(this, CHANNEL_SKIP_ID)
                .setSmallIcon(android.R.drawable.ic_media_next)
                .setContentTitle("⏩ Şarkı Otomatik Geçildi")
                .setContentText("Kara listedeki $personName atlandı ($trackTitle)")
                .setPriority(NotificationCompat.PRIORITY_HIGH)
                .setAutoCancel(true)
                .setTimeoutAfter(4000)
                .build()

            val manager = getSystemService(NotificationManager::class.java)
            manager?.notify(SKIP_NOTIFICATION_ID, notification)
        } catch (_: Exception) {}
    }

    companion object {
        private const val TAG = "MediaWatcherService"
        private const val CHANNEL_STICKY_ID = "oe_watcher_persistent"
        private const val CHANNEL_SKIP_ID = "oe_watcher_skips"
        private const val STICKY_NOTIFICATION_ID = 1000
        private const val SKIP_NOTIFICATION_ID = 1001
    }
}
