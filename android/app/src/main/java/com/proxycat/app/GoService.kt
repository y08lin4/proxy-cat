package com.proxycat.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.os.Binder
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import java.io.File
import java.io.FileOutputStream

class GoService : Service() {

    companion object {
        private const val TAG = "GoService"
        private const val CHANNEL_ID = "proxy-cat"
        private const val NOTIFICATION_ID = 1
        private const val BINARY_NAME = "proxy-cat"
        private const val PORT = 8080
    }

    inner class LocalBinder : Binder() {
        fun getService(): GoService = this@GoService
    }

    private val binder = LocalBinder()
    private var process: Process? = null

    override fun onBind(intent: Intent?): IBinder = binder

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val notification = buildNotification()
        startForeground(NOTIFICATION_ID, notification)
        startBinary()
        return START_STICKY
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Proxy-Cat Service",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Keeps the proxy-cat Go backend running"
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildNotification(): Notification {
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Proxy-Cat")
            .setContentText("Proxy running on port $PORT")
            .setSmallIcon(android.R.drawable.ic_menu_manage)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }

    private fun startBinary() {
        Thread {
            try {
                val binaryFile = extractBinary()
                makeExecutable(binaryFile)
                val command = listOf(
                    binaryFile.absolutePath,
                    "--headless",
                    "--no-system-proxy",
                    "--port", PORT.toString()
                )
                Log.i(TAG, "Starting: ${command.joinToString(" ")}")
                val processBuilder = ProcessBuilder(command)
                    .directory(filesDir)
                    .redirectErrorStream(true)
                process = processBuilder.start()

                // Log stdout/stderr
                process?.inputStream?.bufferedReader()?.use { reader ->
                    reader.lineSequence().forEach { line ->
                        Log.i(TAG, "[proxy-cat] $line")
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to start proxy-cat binary", e)
            }
        }.start()
    }

    private fun extractBinary(): File {
        val binaryFile = File(filesDir, BINARY_NAME)
        if (binaryFile.exists() && binaryFile.canExecute()) {
            Log.i(TAG, "Binary already extracted at ${binaryFile.absolutePath}")
            return binaryFile
        }

        Log.i(TAG, "Extracting binary from assets…")
        assets.open(BINARY_NAME).use { input ->
            FileOutputStream(binaryFile).use { output ->
                input.copyTo(output)
            }
        }
        return binaryFile
    }

    private fun makeExecutable(file: File) {
        if (!file.setExecutable(true)) {
            Log.w(TAG, "setExecutable returned false for ${file.absolutePath}")
        }
        check(file.canExecute()) { "Binary is not executable: ${file.absolutePath}" }
        Log.i(TAG, "Binary ready: ${file.absolutePath}")
    }

    override fun onDestroy() {
        process?.destroy()
        process = null
        super.onDestroy()
    }
}
