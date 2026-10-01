package com.bysinbin.oewatcher

import android.content.ComponentName
import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.service.notification.NotificationListenerService
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.lifecycle.lifecycleScope
import com.bysinbin.oewatcher.data.MediaRepository
import com.bysinbin.oewatcher.engine.MediaSessionHelper
import com.bysinbin.oewatcher.service.MediaWatcherService
import com.bysinbin.oewatcher.theme.OEWatcherTheme
import com.bysinbin.oewatcher.ui.main.MainScreen
import com.bysinbin.oewatcher.ui.main.checkNotificationPermission
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        MediaRepository.init(applicationContext)

        ensureServiceAndCheckMedia()

        // Background poller while app is active in foreground
        lifecycleScope.launch {
            while (isActive) {
                if (checkNotificationPermission(this@MainActivity)) {
                    MediaSessionHelper.inspectActiveMedia(applicationContext)
                }
                delay(1500)
            }
        }

        enableEdgeToEdge()
        setContent {
            OEWatcherTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    MainScreen()
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        ensureServiceAndCheckMedia()
    }

    private fun ensureServiceAndCheckMedia() {
        val hasPerm = checkNotificationPermission(this)
        MediaRepository.setPermissionGranted(hasPerm)

        if (hasPerm) {
            val component = ComponentName(this, MediaWatcherService::class.java)

            // 1. Force requestRebind
            try {
                NotificationListenerService.requestRebind(component)
            } catch (_: Exception) {}

            // 2. Start foreground service explicitly (bypasses Xiaomi background restriction)
            try {
                val intent = Intent(this, MediaWatcherService::class.java)
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    startForegroundService(intent)
                } else {
                    startService(intent)
                }
            } catch (_: Exception) {}

            // 3. Immediately inspect active media
            MediaSessionHelper.inspectActiveMedia(applicationContext)
        }
    }
}
