# Medya ve İçerik Filtresi (OE Watcher) 🎵🛡️🚫

**OE Watcher**, hem masaüstünde çalan müzikleri denetleyen hem de web tarayıcınızda gezinirken istemediğiniz kişilere ait paylaşımları, tweetleri ve videoları otomatik olarak gizleyen çift katmanlı, profesyonel bir filtreleme ekosistemidir.

* **macOS**, **Windows** ve **Linux** desteği (Saf Go).
* **Google Chrome**, **Edge**, **Brave** ve **Mozilla Firefox** eklentileri ile anlık web içerik sansürleme.
* Arka planda sessizce çalışır, **`config.json`** veya modern **Web Kontrol Paneli** ile yönetilir ve **yerel ağ şifresi** ile korunur.

---

## 🌟 İki Temel Yetenek

### 1. 🛡️ Akıllı Web İçerik Filtreleme ve Gizleme (Content Shielding)
İnternette gezinirken (Twitter/X, Instagram, YouTube, Ekşi Sözlük, haber portalları, bloglar), kara listedeki kişilerin geçtiği tweetleri, video kartlarını, haber paragraflarını veya profilleri anında tespit eder:
* **Flu (Blur) Kalkanı:** İçeriği görmeden önce otomatik olarak bulanıklaştırır ve gizler.
* **Onay Mekanizması:** Kartın üzerine şık bir kalkan yerleştirerek *"⚠️ [Kişi Adı] hakkında içerik tespit edildi. Bu içeriği görmek istiyor musunuz?"* diye sorar.
* **👁️ İçeriği Göster:** İncelemek isterseniz tek tıkla içeriği açabilirsiniz.
* **Çevrimdışı ve Canlı Koruma:** Masaüstü uygulaması açıkken yeni eklediğiniz kişiler anında tarayıcıya yansır; masaüstü kapalıyken bile 570+ kişilik yerleşik önbelleğiyle koruma sürer.

### 2. 🎵 Canlı Medya ve Müzik Takipçisi (Auto-Skip)
Bilgisayarınızda (**Spotify**, **Apple Music**, **YouTube**, **Brave**, **Safari**, **Edge**, **VLC**) çalan şarkıları milisaniyelik taramayla izler:
* Kara listedeki sanatçı, prodüktör, besteci veya söz yazarı çaldığında şarkıyı donanımsal medya tuşlarıyla anında atlar (Skip).
* İsteğe bağlı olarak **"Şarkıyı Geç"**, **"1 Saat Ertele"** veya **"Dinlemeye Devam"** seçenekleri sunan işletim sistemi onay penceresi açar.

---

## 🌍 Platform Desteği ve Entegrasyonlar

| Özellik | macOS | Windows (10 / 11) | Linux (GNOME, KDE vb.) |
| :--- | :--- | :--- | :--- |
| **Web İçerik Gizleme** | Chrome, Firefox, Brave, Arc, Edge | Chrome, Firefox, Edge, Brave, Opera | Chrome, Firefox, Chromium, Brave |
| **İzlenen Medyalar** | Spotify, Apple Music, Chrome, Brave, Arc, Safari | Spotify Desktop, Chrome, Edge, GSMTC Medyaları | Spotify, Chromium, Firefox, VLC, MPRIS2 Oynatıcılar |
| **Şarkı Geçişi (Skip)** | Donanımsal Medya Tuşu (19) + MediaRemote | Donanımsal Medya Tuşu (`VK_MEDIA_NEXT_TRACK`) | `playerctl`, D-Bus MPRIS2, `xdotool` |
| **Kullanıcı Arayüzü** | Üst Menü Çubuğu (`🎧 OE`) + Canlı Web Paneli | Canlı Web Kontrol Paneli (`:8080`) | Canlı Web Kontrol Paneli (`:8080`) |
| **Uyarı & Dialog** | macOS 3 Butonlu Etkileşimli Pencere | Windows Forms / PowerShell Uyarı Penceresi | `zenity` / `kdialog` / `notify-send` |
| **Otomatik Başlatma** | macOS LaunchAgent (`-install-service`) | Windows Başlangıç Kaydı (`-install-service`) | Linux systemd user service (`-install-service`) |
| **Veri Konumu** | `~/Library/Application Support/OEWatcher` | `%APPDATA%\OEWatcher` | `~/.config/OEWatcher` |

---

## 🛡️ Akıllı Web İçerik Gizleme Nasıl Çalışır?

Tarayıcı eklentisi ([extension/](file:///Users/feritetem/Desktop/medya-filter/extension) ve [extension-firefox/](file:///Users/feritetem/Desktop/medya-filter/extension-firefox)), gezinme deneyiminizi bozmadan sayfa içerisindeki istenmeyen ögeleri hedefler:

1. **Hedef Odaklı Kalkanlama:** Sayfanın tamamını değil, yalnızca kara listedeki ismin geçtiği tweet'i (`article[data-testid="tweet"]`), YouTube videosunu (`ytd-rich-item-renderer`), haber kutusunu (`.news-item`) veya arama sonucunu kalkanlar.
2. **Cam Efekti (Glassmorphism Overlay):** Kalkanlanan ögenin üzerine zarif, karanlık temalı bir bilgi kutusu yerleştirilir:
   ```
   ┌──────────────────────────────────────────────────────────────┐
   │ ⚠️ [Kişi Adı] hakkında içerik tespit edildi                 │
   │ Bu içeriği görmek istiyor musunuz?                           │
   │                                                              │
   │ [👁️ İçeriği Göster]    [❌ Gizle]    [🛡️ Bu Sayfayı Muaf Tut] │
   └──────────────────────────────────────────────────────────────┘
   ```
3. **Akıllı Karakter Eşleştirme:** Büyük/küçük harf, Türkçe karakterler (`İ/i`, `ş/s`, `ğ/g`) ve takma adlar (Alias) otomatik çözümlenir.
4. **Beyaz Liste / Muafiyet:** Belirli bir GitHub reposunu, web sitesini veya sayfayı (örn. `github.com/bysinbin/oe-watcher`) tek tıkla muaf tutabilirsiniz.

---

## 🚀 Öne Çıkan Özellikler

1. **Çoklu Platform (macOS, Windows, Linux) Desteği:**
   - Go'nun sıfır harici C bağımlılığı prensibiyle Windows ve Linux üzerinde saf Go ile derlenir ve anında çalışır.
   - macOS'ta yerel Cocoa Menü Çubuğu (`🎧 OE`) uygulamasını sunar.

2. **Canlı Web Kontrol Paneli (SSE Destekli):**
   - Şarkı değişimleri, ertelemeler ve filtre durumları `http://127.0.0.1:8080` adresindeki modern web arayüzüne anında (0ms) yansır.
   - Kara listedeki kişileri arayabilir, yeni kişi veya takma ad (alias) ekleyebilir, beyaz liste (exclude list) tanımlayabilirsiniz.

3. **Merkezi Yapılandırma (`config.json`):**
   - API URL'i, web portu, IP bağlantı adresi ve web arayüzü şifresi kolayca yönetilir.

4. **Çift Katmanlı Ağ ve Parola Güvenliği:**
   - **IP İzolasyonu (`bind_address: "127.0.0.1"`):** Yalnızca kendi bilgisayarınızdan erişilebilir.
   - **Parola Koruması (`web_password`):** Web kontrol paneli parola ile korunur (varsayılan: `admin`).

5. **Özel Kurallar Yönetimi:**
   - **➕ Elle Kişi Ekleme:** API'de yer almayan kişileri kara listeye dahil etme.
   - **🔀 Alias (Mahlas) Eşleştirme:** Takma ad girerek (`Süperstar` ➔ `Ajda Pekkan`) hedef kişiyle bağlama.
   - **🛡️ Exclude List (Beyaz Liste):** URL veya isim tanımlayarak (`https://github.com/bysinbin/oe-watcher`, `Beşiktaş JK` vb.) muaf tutma.

6. **Gelişmiş Eşleştirme & Katkıda Bulunanlar Taraması:**
   - Türkçe karakter normalizasyonu ve Levenshtein Bulanık Eşleştirme (%85 tolerans).
   - YouTube açıklaması ve şarkı künyesindeki prodüktör, aranjör, söz/müzik yazarlarını tarar.

---

## 🌐 Tarayıcı Eklentileri Kurulumu

### 🦊 Mozilla Firefox Kurulumu:
1. Firefox'ta adres çubuğuna `about:debugging#/runtime/this-firefox` yazın.
2. **"Geçici Eklenti Yükle..."** *(Load Temporary Add-on...)* butonuna tıklayın.
3. Projedeki **`extension-firefox/manifest.json`** dosyasını seçin.
4. Eklenti anında araç çubuğunuza eklenir ve koruma başlar.

*(Alternatif olarak `./build_extensions.sh` ile oluşturulan `dist/oe-watcher-firefox.xpi` dosyasını doğrudan Firefox'a sürükleyip bırakabilirsiniz).*

---

### 🌐 Google Chrome / Edge / Brave / Opera Kurulumu:
1. Tarayıcınızda `chrome://extensions` (Edge için `edge://extensions`) adresine gidin.
2. Sağ üstteki **"Geliştirici Modu"**nu (Developer mode) açın.
3. **"Paketlenmemiş Öğe Yükle"** (Load unpacked) butonuna tıklayıp projedeki **`extension`** klasörünü seçin.

---

### 📦 Eklentileri Paketleme (Web Store / AMO):
```bash
./build_extensions.sh
```
> Çıktılar:
> - `dist/oe-watcher-chrome.zip` (Chrome Web Store için)
> - `dist/oe-watcher-firefox.xpi` ve `dist/oe-watcher-firefox.zip` (Firefox AMO için)

---

## ⚙️ Yapılandırma (`config.json`)

```json
{
  "api_url": "https://oelist-api.frax.tr/v1",
  "web_port": 8080,
  "bind_address": "127.0.0.1",
  "web_password": "admin",
  "auto_skip": false,
  "enable_dialog": true,
  "poll_interval_ms": 1500,
  "category": "HEPSİ",
  "fuzzy_matching": true,
  "fuzzy_threshold": 0.85,
  "auto_sync_hours": 12
}
```

* **`api_url`**: Veritabanının çekileceği OE List API adresi.
* **`bind_address`**: `"127.0.0.1"` sayesinde dış ağdan gelen istekler işletim sistemi düzeyinde engellenir.
* **`web_password`**: Web paneline girişte istenecek parola (varsayılan: `admin`).
* **`auto_skip`**: `true` yapılırsa eşleşen şarkılar sormadan hemen atlanır.
* **`enable_dialog`**: `true` olduğunda 3 seçenekli ("Şarkıyı Geç", "1 Saat Ertele", "Dinlemeye Devam") onay penceresi açılır.
* **`auto_sync_hours`**: Belirtilen saat aralığında (varsayılan: 12 saat) arka planda sessizce yeni kara liste kayıtlarını tazeler.

---

## 💻 Masaüstü Uygulaması Derleme ve Kurulum

### 1. Tüm Platformları ve Eklentileri Tek Seferde Derleme
```bash
./build_all.sh
```
> Çıktılar doğrudan `dist/` klasörüne toplanır:
> - `dist/OE Watcher.app` (macOS arm64 Uygulama Paketi)
> - `dist/oe-watcher-windows-amd64.exe` (Windows 64-bit)
> - `dist/oe-watcher-windows-arm64.exe` (Windows ARM64)
> - `dist/oe-watcher-linux-amd64` (Linux 64-bit)
> - `dist/oe-watcher-linux-arm64` (Linux ARM64 / Raspberry Pi)
> - `dist/oe-watcher-chrome.zip` (Chrome Eklenti Paketi)
> - `dist/oe-watcher-firefox.xpi` (Firefox Eklenti Paketi)

---

### 2. Platforma Özel Derleme

#### macOS:
```bash
./build_app.sh
```
> `dist/OE Watcher.app` oluşturulur ve kod imzası uygulanır. `/Applications` klasörüne taşıyıp çift tıklayarak çalıştırabilirsiniz.

#### Windows:
```cmd
build_windows.bat
REM veya:
go build -o dist/oe-watcher.exe .
```

#### Linux:
```bash
./build_linux.sh
REM veya:
go build -o dist/oe-watcher .
```

---

### 3. Bilgisayar Açılışında Otomatik Başlatma

İşletim sisteminize göre tek komutla otomatik başlatma servisini kurabilirsiniz:

```bash
# macOS (LaunchAgent):
./dist/OE\ Watcher.app/Contents/MacOS/OE\ Watcher -install-service

# Windows (Registry Startup):
dist\oe-watcher.exe -install-service

# Linux (systemd user service & autostart desktop):
./dist/oe-watcher -install-service
```

Kaldırmak için:
```bash
./oe-watcher -uninstall-service
```

---

## 📦 Proje Mimarisi

```
medya-filter/
├── main.go                     # Giriş noktası, bayraklar ve çoklu platform servis yöneticisi
├── build_all.sh                # macOS, Windows, Linux ve Eklenti tam derleme betiği
├── build_app.sh                # Bağımsız macOS .app paketi derleme ve kod imzalama
├── build_windows.bat           # Windows yerel derleme betiği
├── build_linux.sh              # Linux yerel derleme betiği
├── build_extensions.sh         # Chrome ve Firefox eklentilerini paketleme betiği
├── config.json                 # Yapılandırma dosyası (API, Port, Parola vb.)
├── extension/                  # Google Chrome / Chromium tabanlı tarayıcı eklentisi
├── extension-firefox/          # Mozilla Firefox (Gecko MV3) eklentisi
├── data/
│   ├── persons.json            # API'den senkronize edilen kişi önbelleği
│   ├── custom_rules.json       # Elle eklenen kişiler, aliaslar ve beyaz liste
│   └── session.json            # API oturum belirteci
├── pkg/
│   ├── config/                 # Platformlar arası config ve veri yolu yöneticisi
│   ├── tray/                   # macOS Cocoa Menü Çubuğu ve evrensel döngü
│   ├── models/                 # Veri modelleri (MediaInfo, MatchResult vb.)
│   ├── api/                    # OE List API istemcisi
│   ├── matcher/                # Eşleştirme motoru (Exact, Fuzzy, Katkıda Bulunanlar)
│   ├── media/                  # macOS (JXA/MediaRemote), Windows (GSMTC/Win32), Linux (MPRIS2)
│   ├── notifier/               # macOS, Windows ve Linux yerel bildirim ve onay pencereleri
│   ├── permission/             # macOS Accessibility TCC ve evrensel izin yöneticisi
│   ├── history/                # Tespit ve şarkı geçiş geçmişi yöneticisi
│   ├── rules/                  # Özel kurallar, takma ad ve exclude motoru
│   └── web/                    # Gömülü Web Paneli (SSE, Canlı Dashboard, Ayarlar)
└── README.md
```
