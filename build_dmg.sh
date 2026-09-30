#!/usr/bin/env bash
set -e

# ====================================================================
#  OE WATCHER // macOS Drag-and-Drop .dmg Kurulum Paketi Oluşturucu
# ====================================================================

APP_NAME="OE Watcher"
VOL_NAME="OE Watcher"
DMG_NAME="OE-Watcher-macOS.dmg"
DIST_DIR="dist"
APP_BUNDLE="${DIST_DIR}/${APP_NAME}.app"
FINAL_DMG="${DIST_DIR}/${DMG_NAME}"
STAGING_DIR="/tmp/oe_watcher_dmg_staging"
BG_DIR="${STAGING_DIR}/.background"

echo "💿 [1/4] macOS Uygulama Paketi Kontrol Ediliyor..."
if [ ! -d "${APP_BUNDLE}" ]; then
    echo "⚠️ ${APP_BUNDLE} bulunamadı, önce build_app.sh çalıştırılıyor..."
    ./build_app.sh
fi

echo "🔨 [2/4] DMG Staging Alanı Hazırlanıyor..."
rm -rf "${STAGING_DIR}"
rm -f "${FINAL_DMG}" "/tmp/oe_temp.dmg"
mkdir -p "${STAGING_DIR}" "${BG_DIR}"

# 1. Uygulamayı kopyala
echo "   • ${APP_NAME}.app staging alanına kopyalanıyor..."
cp -R "${APP_BUNDLE}" "${STAGING_DIR}/"

# 2. Applications (Uygulamalar) sembolik linkini oluştur
echo "   • /Applications kısayolu oluşturuluyor..."
ln -s /Applications "${STAGING_DIR}/Applications"

# 3. Arka Plan Görseli Oluştur
echo "🎨 [3/4] DMG Arka Plan Tasarımı Hazırlanıyor..."
python3 -c "
import zlib, struct

def make_bg():
    w, h = 640, 420
    raw = bytearray()
    
    # Modern ok pikselleri (x: 290 -> 350, y: 200)
    arrow_pixels = set()
    for x in range(290, 345):
        for y in range(198, 203):
            arrow_pixels.add((x, y))
    for d in range(16):
        arrow_pixels.add((345 - d, 200 - d))
        arrow_pixels.add((345 - d, 200 + d))
        arrow_pixels.add((345 - d - 1, 200 - d))
        arrow_pixels.add((345 - d - 1, 200 + d))

    for y in range(h):
        raw.append(0)
        for x in range(w):
            grad = y / float(h)
            # Koyu karbon degrade
            r = int(12 + 10 * grad)
            g = int(15 + 12 * grad)
            b = int(24 + 18 * grad)
            
            # Üst kırmızı vurgu çizgisi (Crimson)
            if y < 3:
                r, g, b = 225, 29, 72
                
            # Sol ve sağ ikon hedef yuvaları (Hafif aydınlık)
            in_left_target = ((x - 165)**2 + (y - 200)**2) < 5600
            in_right_target = ((x - 475)**2 + (y - 200)**2) < 5600
            if in_left_target or in_right_target:
                r = min(255, r + 14)
                g = min(255, g + 14)
                b = min(255, b + 20)
                
            # Ok çizimi
            if (x, y) in arrow_pixels:
                r, g, b = 225, 29, 72
                
            raw.extend([r, g, b, 255])
            
    def chunk(tag, data):
        return struct.pack('>I', len(data)) + tag + data + struct.pack('>I', zlib.crc32(tag + data) & 0xffffffff)
        
    png = b'\x89PNG\r\n\x1a\n'
    png += chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 6, 0, 0, 0))
    png += chunk(b'IDAT', zlib.compress(bytes(raw), 6))
    png += chunk(b'IEND', b'')
    with open('${BG_DIR}/bg.png', 'wb') as f:
        f.write(png)

make_bg()
"

echo "📦 [4/4] Sıkıştırılmış DMG Paketi Üretiliyor..."
# Doğrudan macOS UDZO (Apple Disk Image - Read-Only Compressed) formatında oluştur
hdiutil create -volname "${VOL_NAME}" \
               -srcfolder "${STAGING_DIR}" \
               -ov \
               -format UDZO \
               "${FINAL_DMG}"

# Temizlik
rm -rf "${STAGING_DIR}"

echo ""
echo "===================================================================="
echo "🎉 BAŞARILI: macOS DMG Yükleyici Paketi Hazır!"
echo "📁 Paket Konumu: ${FINAL_DMG}"
echo "📊 Boyut: $(du -sh "${FINAL_DMG}" | cut -f1)"
echo "===================================================================="
echo "💡 Kullanıcı Deneyimi:"
echo "   Kullanıcı '${DMG_NAME}' dosyasına çift tıkladığında;"
echo "   '${APP_NAME}' simgesini doğrudan 'Applications' klasörüne"
echo "   sürükleyip bırakarak kurulumu 2 saniyede tamamlayabilir."
echo "===================================================================="
