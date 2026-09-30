#!/usr/bin/env bash
set -e

# ====================================================================
#  OE WATCHER // Linux Native Build Script
# ====================================================================

echo "🐧 OE Watcher Linux paketi derleniyor..."
mkdir -p dist
go build -ldflags="-s -w" -o dist/oe-watcher .
chmod +x dist/oe-watcher

cp config.json dist/ || true
cp -r data dist/ || true

echo ""
echo "===================================================================="
echo "🎉 BAŞARILI: Linux binary hazırlandı: dist/oe-watcher"
echo "Çalıştırmak için: ./dist/oe-watcher"
echo "Servis olarak kurmak için: ./dist/oe-watcher -install-service"
echo "===================================================================="
