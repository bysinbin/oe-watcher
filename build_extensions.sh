#!/usr/bin/env bash
set -e

# ====================================================================
#  OE WATCHER // Browser Extensions Packaging Script
# ====================================================================

echo "📦 Tarayıcı Eklentileri Paketleniyor..."
mkdir -p dist

# 1. Chrome / Chromium / Edge / Brave Extension (.zip)
echo "🌐 [1/2] Google Chrome / Edge / Brave eklentisi paketleniyor..."
(cd extension && zip -q -r -FS ../dist/oe-watcher-chrome.zip . -x ".*" -x "__MACOSX" -x "README.md")
echo "✅ Chrome Paketi: dist/oe-watcher-chrome.zip"

# 2. Mozilla Firefox Extension (.xpi & .zip)
echo "🦊 [2/2] Mozilla Firefox eklentisi (.xpi / .zip) paketleniyor..."
(cd extension-firefox && zip -q -r -FS ../dist/oe-watcher-firefox.zip . -x ".*" -x "__MACOSX" -x "README.md")
cp dist/oe-watcher-firefox.zip dist/oe-watcher-firefox.xpi
echo "✅ Firefox Paketi: dist/oe-watcher-firefox.xpi"
echo "✅ Firefox Arşivi: dist/oe-watcher-firefox.zip"

echo ""
echo "===================================================================="
echo "🎉 Eklenti paketleme tamamlandı!"
echo "📁 Çıktılar:"
echo "   • dist/oe-watcher-chrome.zip  (Chrome Web Store / Manuel Yükleme)"
echo "   • dist/oe-watcher-firefox.xpi (Firefox Add-on / about:debugging)"
echo "===================================================================="
