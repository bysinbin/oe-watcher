#!/usr/bin/env bash
set -e

# ====================================================================
#  OE WATCHER // macOS .app Paketi Oluşturucu (Build Script)
# ====================================================================

APP_NAME="OE Watcher"
BUNDLE_DIR="dist/${APP_NAME}.app"
CONTENTS_DIR="${BUNDLE_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

echo "🔨 [1/5] Eski derleme kalıntıları temizleniyor..."
rm -rf "dist"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"

echo "⚙️ [2/5] macOS Menü Çubuğu (Tray & CGO) ikili dosyası derleniyor..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o "${MACOS_DIR}/${APP_NAME}" .
chmod +x "${MACOS_DIR}/${APP_NAME}"

echo "📝 [3/5] Info.plist yapılandırma dosyası hazırlanıyor..."
cat << 'EOF' > "${CONTENTS_DIR}/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>OE Watcher</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.bysinbin.oe-watcher</string>
    <key>CFBundleName</key>
    <string>OE Watcher</string>
    <key>CFBundleDisplayName</key>
    <string>OE Watcher</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.2.0</string>
    <key>CFBundleVersion</key>
    <string>1.2.0</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSAppleEventsUsageDescription</key>
    <string>OE Watcher çalan müzikleri tespit etmek ve şarkı geçişlerini kontrol etmek için bu izne ihtiyaç duyar.</string>
    <key>NSAccessibilityUsageDescription</key>
    <string>OE Watcher müzik çaları kontrol etmek ve şarkı geçişi yapabilmek için erişilebilirlik iznine ihtiyaç duyar.</string>
</dict>
</plist>
EOF

echo "🎨 [4/6] macOS Uygulama İkonu (AppIcon.icns) üretiliyor..."
ICONSET_DIR="/tmp/OEWatcher.iconset"
rm -rf "${ICONSET_DIR}"
mkdir -p "${ICONSET_DIR}"

python3 -c "
import zlib, struct

def make_base_icon():
    width = 1024
    height = 1024
    raw = bytearray()
    for y in range(height):
        raw.append(0)
        for x in range(width):
            dx = (x - 512) / 512.0
            dy = (y - 512) / 512.0
            r_dist = (dx*dx + dy*dy)**0.5
            
            # Squircle badge
            if abs(dx)**4 + abs(dy)**4 < 0.68:
                # Premium crimson/dark slate gradient
                grad = 1.0 - (dy * 0.4 + dx * 0.2)
                r = int(min(255, max(0, 220 * grad)))
                g = int(min(255, max(0, 35 * grad)))
                b = int(min(255, max(0, 65 * grad + 20)))
                
                # Headphone geometry
                in_arc = 0.32 < r_dist < 0.46 and dy < 0.12
                in_cup_l = abs(dx + 0.38) < 0.08 and abs(dy - 0.08) < 0.20
                in_cup_r = abs(dx - 0.38) < 0.08 and abs(dy - 0.08) < 0.20
                
                if in_arc or in_cup_l or in_cup_r:
                    r, g, b = 255, 255, 255
                    
                raw.extend([r, g, b, 255])
            else:
                raw.extend([0, 0, 0, 0])

    def chunk(tag, data):
        return struct.pack('>I', len(data)) + tag + data + struct.pack('>I', zlib.crc32(tag + data) & 0xffffffff)

    png = b'\x89PNG\r\n\x1a\n'
    png += chunk(b'IHDR', struct.pack('>IIBBBBB', width, height, 8, 6, 0, 0, 0))
    png += chunk(b'IDAT', zlib.compress(bytes(raw), 6))
    png += chunk(b'IEND', b'')
    with open('/tmp/oe_icon_1024.png', 'wb') as f:
        f.write(png)

make_base_icon()
"

sips -z 16 16     /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_16x16.png" > /dev/null
sips -z 32 32     /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_16x16@2x.png" > /dev/null
sips -z 32 32     /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_32x32.png" > /dev/null
sips -z 64 64     /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_32x32@2x.png" > /dev/null
sips -z 128 128   /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_128x128.png" > /dev/null
sips -z 256 256   /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_128x128@2x.png" > /dev/null
sips -z 256 256   /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_256x256.png" > /dev/null
sips -z 512 512   /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_256x256@2x.png" > /dev/null
sips -z 512 512   /tmp/oe_icon_1024.png --out "${ICONSET_DIR}/icon_512x512.png" > /dev/null
cp /tmp/oe_icon_1024.png "${ICONSET_DIR}/icon_512x512@2x.png"

iconutil -c icns "${ICONSET_DIR}" -o "${RESOURCES_DIR}/AppIcon.icns"
rm -rf "${ICONSET_DIR}" /tmp/oe_icon_1024.png

echo "📦 [5/6] Yapılandırma ve veri dosyaları kopyalanıyor..."
cp config.json "${RESOURCES_DIR}/" || true
cp -r data "${RESOURCES_DIR}/" || true
cp config.json "dist/" || true
cp -r data "dist/" || true

echo "🔏 [6/6] macOS Uygulama Paketi Kod İmzası (Ad-hoc Codesign) uygulanıyor..."
# Finder ve sistem kalıntı özniteliklerini temizle (FinderInfo detritus hatasını engeller)
xattr -cr "${BUNDLE_DIR}"
# Bundle ID ile ad-hoc kod imzası uygula (Erişilebilirlik izninin kalıcı olmasını sağlar)
codesign --force --deep --sign - --identifier "com.bysinbin.oe-watcher" "${BUNDLE_DIR}"
codesign -vvv --deep --strict "${BUNDLE_DIR}"

echo ""
echo "===================================================================="
echo "🎉 BAŞARILI: Bağımsız macOS Paketi Hazırlandı ve İmzalandı!"
echo "📁 Paket Konumu: dist/OE Watcher.app"
echo "===================================================================="
echo "💡 İpucu: 'dist/OE Watcher.app' dosyasını doğrudan /Applications klasörüne"
echo "          sürükleyip bırakabilir veya çift tıklayarak çalıştırabilirsiniz."
echo "          Uygulama menü çubuğuna (🎧 OE) yerleşecek ve Dock'ta yer kaplamayacaktır."
echo "===================================================================="
