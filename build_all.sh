#!/usr/bin/env bash
set -e

# ====================================================================
#  OE WATCHER // Multi-Platform Build Script (macOS, Windows, Linux)
# ====================================================================

echo "🚀 OE Watcher Çoklu Platform Derleme Başlatılıyor..."

# 1. Build macOS Native .app bundle & .dmg installer
if [ "$(uname)" = "Darwin" ]; then
    echo ""
    echo "🍏 [1/3] macOS .app paketi hazırlanıyor..."
    ./build_app.sh
    echo ""
    echo "💿 macOS .dmg yükleyici paketi hazırlanıyor..."
    ./build_dmg.sh
fi

# 2. Build Linux Binaries (AMD64 & ARM64)
echo ""
echo "🐧 [2/3] Linux ikili dosyaları derleniyor..."
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/oe-watcher-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/oe-watcher-linux-arm64 .
chmod +x dist/oe-watcher-linux-amd64 dist/oe-watcher-linux-arm64
echo "✅ Linux AMD64: dist/oe-watcher-linux-amd64"
echo "✅ Linux ARM64: dist/oe-watcher-linux-arm64"

# 3. Build Windows Executables (AMD64 & ARM64)
echo ""
echo "🪟 [3/3] Windows çalıştırılabilir dosyaları (.exe) derleniyor..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/oe-watcher-windows-amd64.exe .
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/oe-watcher-windows-arm64.exe .
echo "✅ Windows AMD64: dist/oe-watcher-windows-amd64.exe"
echo "✅ Windows ARM64: dist/oe-watcher-windows-arm64.exe"

# 4. Package Browser Extensions (Chrome & Firefox)
echo ""
echo "🌐 [4/4] Tarayıcı eklentileri (Chrome & Firefox) paketleniyor..."
./build_extensions.sh

echo ""
echo "===================================================================="
echo "🎉 TÜM PLATFORMLAR VE EKLENTİLER İÇİN DERLEME TAMAMLANDI!"
echo "📁 Çıktı Klasörü: dist/"
echo "===================================================================="
ls -lh dist/
