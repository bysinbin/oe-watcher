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
   - Düet ve `feat.` sanatçı ayrıştırması.
   - Spotify, Apple Music, Google Chrome (YouTube / YT Music) ve Safari tam desteği.
   - Donanımsal medya tuşu simülasyonu ile kusursuz şarkı geçişi.

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
  "category": "HEPSİ"
}
```

* **`api_url`**: Veritabanının çekileceği OE List API adresi.
* **`bind_address`**: `"127.0.0.1"` sayesinde dış ağdan gelen istekler işletim sistemi düzeyinde engellenir.
* **`web_password`**: Web paneline girişte istenecek parola (varsayılan: `admin`).
* **`auto_skip`**: `true` yapılırsa eşleşen şarkılar sormadan hemen atlanır.

---

## 💻 Çalıştırma

### 1. Menü Çubuğu (Tray) Modu (Önerilen)
```bash
./medya-filter
```
> macOS üst çubuğunda `🎧 OE` simgesi belirir. Terminali küçültebilir veya arka planda çalıştırabilirsiniz.

### 2. Sadece Konsol / Terminal Modu
```bash
./medya-filter -cli
```

### 3. Hızlı Test
```bash
./medya-filter -test "Ceza"
./medya-filter -test "Süperstar"
```

---

## 📦 Proje Mimarisi

```
medya-filter/
├── main.go                     # Giriş noktası, bayraklar ve menü çubuğu başlatıcı
├── config.json                 # Yapılandırma dosyası (API, Port, Parola vb.)
├── data/
│   ├── persons.json            # API'den senkronize edilen kişi önbelleği
│   ├── custom_rules.json       # Elle eklenen kişiler, aliaslar ve beyaz liste
│   └── session.json            # API oturum belirteci
├── pkg/
│   ├── config/                 # config.json okuma/yazma motoru
│   ├── tray/                   # macOS Cocoa NSStatusItem (Menu Bar) köprüsü
│   ├── models/                 # Veri modelleri
│   ├── api/                    # OE List API istemcisi
│   ├── matcher/                # Eşleştirme motoru
│   ├── media/                  # Spotify, Apple Music ve Chrome medya denetleyicisi
│   ├── notifier/               # macOS bildirim ve pop-up onay pencereleri
│   └── web/                    # Gömülü Web Paneli (Güvenli Giriş, Dashboard, Kurallar)
└── README.md
```
