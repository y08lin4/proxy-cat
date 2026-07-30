# Proxy-Cat Android Shell App

This is a thin Android shell app that wraps the proxy-cat Go binary as a foreground service and displays the Web UI in a WebView.

## How It Works

1. **Binary Extraction**: The Go binary (cross-compiled with `GOOS=android GOARCH=arm64`) is bundled in `app/src/main/assets/`. On first launch, `GoService` copies it to the app's internal `filesDir`.

2. **Foreground Service**: The binary is executed via `ProcessBuilder` with the flags `--headless --no-system-proxy --port 8080`. It runs as a foreground service with a persistent notification, bound to the host `127.0.0.1:8080`.

3. **WebView**: `MainActivity` loads `http://127.0.0.1:8080` in a full-screen WebView with JavaScript and DOM storage enabled. Back navigation is handled for the WebView history stack.

## Building the Go Binary

Before building the APK, cross-compile the Go binary for Android:

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o android/app/src/main/assets/proxy-cat ./cmd/proxy-cat
```

Note: `CGO_ENABLED=0` is required for Android — the NDK provides a different C runtime.

## Building the APK

Open the `android/` directory in Android Studio, or use the Gradle wrapper:

```bash
cd android
./gradlew assembleRelease
```

## Requirements

- Android 8.0+ (API 26)
- Notification permission must be granted (prompted on first launch)

## Permissions

- `INTERNET` — for localhost WebView and (if configured) proxy traffic
- `FOREGROUND_SERVICE` / `FOREGROUND_SERVICE_SPECIAL_USE` — to keep the Go process alive
- `POST_NOTIFICATIONS` — required on Android 13+ for the foreground service notification
