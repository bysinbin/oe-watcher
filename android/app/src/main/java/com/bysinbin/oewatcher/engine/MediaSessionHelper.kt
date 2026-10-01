package com.bysinbin.oewatcher.engine

import android.content.ComponentName
import android.content.Context
import android.media.MediaMetadata
import android.media.session.MediaSessionManager
import android.media.session.PlaybackState
import android.util.Log
import com.bysinbin.oewatcher.data.MediaRepository
import com.bysinbin.oewatcher.model.MediaTrackInfo
import com.bysinbin.oewatcher.model.SkipEvent
import com.bysinbin.oewatcher.service.MediaWatcherService

object MediaSessionHelper {
    private const val TAG = "MediaSessionHelper"
    private var lastSkippedKey = ""
    private var lastSkipTimestamp = 0L

    fun inspectActiveMedia(context: Context) {
        try {
            val mediaSessionManager = context.getSystemService(Context.MEDIA_SESSION_SERVICE) as? MediaSessionManager ?: return
            val component = ComponentName(context, MediaWatcherService::class.java)
            val controllers = mediaSessionManager.getActiveSessions(component)
            if (!controllers.isNullOrEmpty()) {
                val active = controllers.firstOrNull { it.playbackState?.state == PlaybackState.STATE_PLAYING }
                    ?: controllers.firstOrNull { it.playbackState?.state == PlaybackState.STATE_BUFFERING }
                    ?: controllers.firstOrNull { it.metadata != null }
                    ?: controllers.first()

                val metadata = active.metadata
                val playbackState = active.playbackState
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

                val pkg = active.packageName ?: ""
                val appName = when {
                    pkg.contains("youtube.music") -> "YouTube Music"
                    pkg.contains("spotify") -> "Spotify"
                    pkg.contains("deezer") -> "Deezer"
                    pkg.contains("apple") -> "Apple Music"
                    pkg.contains("fizy") -> "Fizy"
                    pkg.contains("soundcloud") -> "SoundCloud"
                    else -> pkg.substringAfterLast('.').replaceFirstChar { it.uppercase() }
                }

                if (title.isNotBlank() || artist.isNotBlank()) {
                    val trackInfo = MediaTrackInfo(
                        app = appName,
                        packageName = pkg,
                        title = title,
                        artist = artist,
                        album = album,
                        isPlaying = isPlaying
                    )
                    Log.i(TAG, "⚡ Anında Çalan Parça Algılandı: $artist - $title")
                    MediaRepository.updateCurrentTrack(trackInfo)
                    MediaRepository.setServiceRunning(true)

                    // Auto-skip immediate check on app open / foreground poll
                    if (isPlaying && MediaRepository.autoSkipEnabled.value) {
                        val initialMatch = BlacklistMatcher.match(trackInfo.artist, trackInfo.title)
                        val albumMatch = if (initialMatch == null && trackInfo.album.isNotBlank()) {
                            BlacklistMatcher.match(trackInfo.album, trackInfo.title)
                        } else null
                        val match = initialMatch ?: albumMatch

                        if (match != null) {
                            val now = System.currentTimeMillis()
                            val trackKey = "${trackInfo.packageName}:::${trackInfo.artist}:::${trackInfo.title}"
                            if (trackKey != lastSkippedKey || (now - lastSkipTimestamp) > 2000) {
                                lastSkippedKey = trackKey
                                lastSkipTimestamp = now
                                try {
                                    active.transportControls?.skipToNext()
                                    Log.w(TAG, "🚨 [MediaSessionHelper] Kara listedeki sanatçı anında atlandı: ${match.personName}")
                                    val event = SkipEvent(
                                        app = trackInfo.app,
                                        artist = trackInfo.artist,
                                        title = trackInfo.title,
                                        matchedPerson = match.personName,
                                        success = true
                                    )
                                    MediaRepository.recordSkip(event)
                                } catch (e: Exception) {
                                    Log.e(TAG, "Skip error: ${e.message}")
                                }
                            }
                        }
                    }
                }
            }
        } catch (e: Exception) {
            Log.d(TAG, "Aktif medya oturumu sorgu: ${e.message}")
        }
    }
}
