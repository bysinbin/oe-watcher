#!/usr/bin/env bash
set -e

# ====================================================================
#  OE WATCHER // Android Native APK Build Script
# ====================================================================

echo "🤖 OE Watcher Android APK Derleme Başlatılıyor..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ANDROID_DIR="$SCRIPT_DIR/android"
DIST_DIR="$SCRIPT_DIR/dist"

if [ ! -d "$ANDROID_DIR" ]; then
    echo "❌ Hata: 'android' dizini bulunamadı!"
    exit 1
fi

mkdir -p "$DIST_DIR"

cd "$ANDROID_DIR"
chmod +x ./gradlew

echo "📦 Gradle Release derlemesi çalıştırılıyor..."
./gradlew assembleRelease --no-daemon

APK_SRC="$ANDROID_DIR/app/build/outputs/apk/release/app-release.apk"
if [ ! -f "$APK_SRC" ]; then
    APK_SRC="$ANDROID_DIR/app/build/outputs/apk/release/app-release-signed.apk"
fi
if [ ! -f "$APK_SRC" ]; then
    APK_SRC="$ANDROID_DIR/app/build/outputs/apk/debug/app-debug.apk"
fi

if [ -f "$APK_SRC" ]; then
    cp "$APK_SRC" "$DIST_DIR/oe-watcher-android.apk"
    echo ""
    echo "===================================================================="
    echo "🎉 ANDROID APK BAŞARIYLA OLUŞTURULDU!"
    echo "📁 Dosya: $DIST_DIR/oe-watcher-android.apk"
    ls -lh "$DIST_DIR/oe-watcher-android.apk"
    echo "===================================================================="
else
    echo "❌ Hata: APK dosyası bulunamadı!"
    exit 1
fi
