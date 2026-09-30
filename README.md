# Medya Filtresi (OE List Watcher) 🎵🚫

macOS üzerinde çalan medyaları (**Spotify**, **Apple Music**, **Google Chrome / YouTube**, **Safari**) anlık olarak izleyen ve verilen OE List API'sindeki kara liste ile eşleştiğinde kullanıcıyı uyararak şarkıyı geçme (skip) imkanı sunan profesyonel **Go (Golang)** uygulaması.

Artık **macOS Menü Çubuğu (Menu Bar / Status Bar)** üzerinden sessizce çalışır, **`config.json`** ile yönetilir ve **yerel ağ şifresi** ile korunur.

---

## 🚀 Öne Çıkan Özellikler

1. **macOS Menü Çubuğu (Tray / Bildirim Alanı):**
   - Terminali açık tutmanıza gerek kalmadan macOS üst menü çubuğunda (`🎧 OE`) yerleşir.
   - Menüden tek tıkla: Web Panelini açma, şarkıyı hemen geçme, otomatik geçişi açıp kapatma ve API senkronizasyonu yapma.
   - Kara listeden bir şarkı çaldığında menü çubuğu simgesi anında `🚨 OE [UYARI]` durumuna geçer.

2. **Merkezi Yapılandırma (`config.json`):**
   - API URL'i, web portu, ağ bağlantı adresi ve web arayüzü şifresi `config.json` üzerinden kolayca değiştirilebilir.
   - Değişiklikler anında yürürlüğe girer.

3. **Çift Katmanlı Ağ ve Parola Güvenliği:**
   - **IP İzolasyonu (`bind_address: "127.0.0.1"`):** Varsayılan olarak yalnızca kendi bilgisayarınızdan erişilebilir. Aynı Wi-Fi/yerel ağdaki hiç kimse IP adresinizi yazarak panele erişemez.
   - **Parola Koruması (`web_password`):** Web kontrol paneli parola ile korunur. Arayüze giren kullanıcılar `config.json` içindeki parolayı girmeden şarkıları göremez veya kuralları değiştiremez.

4. **Özel Kurallar Yönetimi:**
   - **➕ Elle Kişi Ekleme:** API'de yer almayan kişileri kara listeye dahil etme.
   - **🔀 Alias (Mahlas) Eşleştirme:** Listeden kişiyi seçerek veya takma ad girerek (`Süperstar` ➔ `Ajda Pekkan`) hedef kişiyle bağlama.
   - **🛡️ Exclude List (Beyaz Liste):** Kara listede olsa dahi muaf tutulacak sanatçı veya şarkıları belirleme.

5. **Gelişmiş Eşleştirme & Canlı Medya Takibi:**
   - Türkçe karakter normalizasyonu (`İ/i`, `ö/o`, `ü/u`, `ş/s`, `ç/c`, `ğ/g`).
   - Levenshtein tabanlı Bulanık Eşleştirme (Fuzzy Match).
   - Düet ve `feat.` sanatçı ayrıştırması.
   - **YouTube Açıklama & Katkıda Bulunanlar Taraması:** YouTube DOM/açıklama alanındaki `Söz:`, `Müzik:`, `Beat:`, `Prod:`, `Aranje:`, `Düet:` satırlarını tarayarak prodüktör veya söz yazarlarını yakalar.
   - Spotify, Apple Music, Google Chrome, Brave, Arc Browser ve Safari tam desteği.
   - Donanımsal medya tuşu simülasyonu ile kusursuz şarkı geçişi.

6. **0ms Darwin Notification Center (Sıfır Ek CPU):**
   - Spotify ve Apple Music şarkı geçişlerinde macOS Cocoa `NSDistributedNotificationCenter` üzerinden 0 milisaniye gecikmeyle uyanır.

7. **Server-Sent Events (SSE) ile Anlık Web Paneli:**
   - `/api/events` akışı sayesinde şarkı değişimleri, ertelemeler ve ayarlar web paneline anında (0ms) canlı yansır.

8. **Etkileşimli 3 Butonlu macOS Dialog Penceresi:**
   - Eşleşme olduğunda: **"Şarkıyı Geç"**, **"1 Saat Ertele (Snooze)"** ve **"Dinlemeye Devam"** seçenekleri sunulur.

9. **Otomatik Arka Plan API Senkronizasyonu:**
   - `auto_sync_hours: 12` ayarı ile arka planda sessizce yeni kara liste kayıtlarını günceller.

10. **Bağımsız macOS `.app` Paketi (`OE Watcher.app`):**
    - `./build_app.sh` komutuyla Dock'ta yer kaplamayan, doğrudan `/Applications` içine atılabilen bağımsız `.app` paketi üretilir.

---

## ⚙️ Yapılandırma (`config.json`)

Uygulama ilk çalıştığında otomatik olarak aşağıdaki `config.json` dosyasını oluşturur:

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
* **`enable_dialog`**: `true` olduğunda 3 seçenekli ("Şarkıyı Geç", "1 Saat Ertele", "Dinlemeye Devam") macOS penceresi açılır.
* **`auto_sync_hours`**: Belirtilen saat aralığında (varsayılan: 12) API'den otomatik veri tazelemesi yapar.
* **`fuzzy_matching` & `fuzzy_threshold`**: Harf hatalarını toleransla yakalar (%85 benzerlik).

---

## 💻 Çalıştırma ve Kurulum

### 1. Bağımsız `.app` Paketi Oluşturma (Önerilen)
```bash
./build_app.sh
```
> Bu komut `dist/OE Watcher.app` paketini üretir. Çift tıklayarak açabilir veya `/Applications` (Uygulamalar) klasörüne taşıyabilirsiniz. Menü çubuğuna (`🎧 OE`) yerleşir ve Dock'ta görünmez.

### 2. Doğrudan Çalıştırma
```bash
./medya-filter
```

### 3. Mac Başlangıcında Otomatik Başlatma (LaunchAgent)
```bash
./medya-filter -install-service    # Mac açılışında arka planda başlasın
./medya-filter -uninstall-service  # Başlangıç servisini kaldır
```

### 4. Sadece Konsol / Terminal Modu
```bash
./medya-filter -cli
```

### 5. Hızlı Test
```bash
./medya-filter -test "Ceza"
./medya-filter -test "Süperstar"
```

---

## 📦 Proje Mimarisi

```
medya-filter/
├── main.go                     # Giriş noktası, bayraklar ve menü çubuğu başlatıcı
├── build_app.sh                # Bağımsız .app paketi derleme betiği
├── config.json                 # Yapılandırma dosyası (API, Port, Parola vb.)
├── data/
│   ├── persons.json            # API'den senkronize edilen kişi önbelleği
│   ├── custom_rules.json       # Elle eklenen kişiler, aliaslar ve beyaz liste
│   └── session.json            # API oturum belirteci
├── pkg/
│   ├── config/                 # config.json okuma/yazma motoru
│   ├── tray/                   # macOS Cocoa NSStatusItem & Darwin Notification Center
│   ├── models/                 # Veri modelleri (MediaInfo, MatchResult vb.)
│   ├── api/                    # OE List API istemcisi
│   ├── matcher/                # Eşleştirme motoru (Exact, Fuzzy, Katkıda Bulunanlar)
│   ├── media/                  # Spotify, Apple Music, Chrome, Brave, Arc, Safari izleyicisi
│   ├── notifier/               # macOS bildirim ve 3 butonlu etkileşimli dialog
│   └── web/                    # Gömülü Web Paneli (SSE, Canlı Dashboard, Ayarlar)
└── README.md
```

