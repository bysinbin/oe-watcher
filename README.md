# Medya Filtresi (OE List Watcher) 🎵🚫

macOS üzerinde çalan medyaları (**Spotify**, **Apple Music**, **Google Chrome / YouTube**, **Safari**) anlık olarak izleyen ve verilen OE List API'sindeki kara liste ile eşleştiğinde kullanıcıyı uyararak şarkıyı geçme (skip) imkanı sunan profesyonel **Go (Golang)** uygulaması.

---

## 🚀 Özellikler

1. **Canlı Medya Takibi (macOS):**
   - **Spotify** masaüstü uygulaması
   - **Apple Music** uygulaması
   - **Google Chrome** ve **Safari** aktif sekmeleri (YouTube, Spotify Web vb.)
   - CPU tüketmeyen optimize JXA köprüsü ile anlık durum sorgusu.

2. **Gelişmiş Eşleştirme Motoru:**
   - **Türkçe Karakter Normalizasyonu:** Türkçe `İ/i`, `I/ı`, `ö/o`, `ü/u`, `ş/s`, `ç/c`, `ğ/g` varyasyonlarını sorunsuz eşleştirir.
   - **Parantez & Mahlas (Alias) Ayrıştırma:** Örneğin `"Ceza (Bilgin Özçalkan)"` veya `"Blok3 (Hakan Aydın)"` kayıtlarını hem ana isimden (`Ceza`, `Blok3`) hem de gerçek isimden tanır.
   - **Düet & Çoklu Sanatçı Desteği:** `feat.`, `ft.`, `&`, `,`, `/`, `ve` ile ayrılmış sanatçıları parçalara ayırarak tek tek denetler.

3. **Otomatik İzin Kontrolü (macOS Erişilebilirlik):**
   - Uygulama başlarken macOS Erişilebilirlik (Accessibility) iznini denetler.
   - İzin verilmişse `🔒 macOS Erişilebilirlik (Accessibility) İzni: ETKİN` mesajı verir.
   - İzin eksikse kullanıcıyı uyarır, macOS sistem izin istemini tetikler ve doğrudan **Sistem Ayarları > Gizlilik ve Güvenlik > Erişilebilirlik** sayfasını açar.

4. **Etkileşim & Uyarı Mekanizmaları:**
   - **macOS Yerel Onay Penceresi (Dialog):** Eşleşen sanatçıyı, kategorisini ve konusunu gösterip *"Bu şarkıyı geçmek istiyor musunuz?"* diye sorar. `[Şarkıyı Geç]` seçildiğinde otomatik olarak şarkıyı atlar.
   - **macOS Bildirimleri (Banner):** Şarkı tespit edildiğinde ve geçildiğinde sistem bildirimi ve sesli uyarı (`Ping.aiff`) gönderir.
   - **Otomatik Geçiş Modu (`--auto-skip`):** İstenirse sormadan doğrudan bir sonraki şarkıya geçer.
   - **Zengin Terminal Arayüzü:** Renkli ve detaylı log çıktısı.

4. **Web Kontrol Paneli (Dashboard):**
   - `http://localhost:8080` adresinde çalışan modern, karanlık temalı kontrol paneli.
   - Ekran görüntünüzdeki **KONU //** ve **KATEGORİ //** buton filtrelerinin birebir uyarlaması.
   - Anlık çalan şarkı durumu, manuel şarkı geçiş butonu, 572 kişilik veritabanı arama motoru ve geçmiş kayıtları.

5. **Otomatik Senkronizasyon & Çevrimdışı Önbellek:**
   - Token oluşturur (`/v1/owners/self`), verileri eşitler (`/v1/persons/sync`) ve çeker (`/v1/persons`).
   - Verileri `data/persons.json` içinde önbelleğe alarak internet kesintilerinde veya sonraki açılışlarda anında başlar.

---

## 🛠️ Kurulum ve Derleme

Go yüklü bir macOS ortamında:

```bash
# Proje dizininde
cd /Users/feritetem/Desktop/medya-filter

# Bağımlılıkları kontrol et ve derle
go build -o medya-filter .
```

---

## 💻 Kullanım

### 1. Normal Çalıştırma (Onaylı Mod + Web Panel)
```bash
./medya-filter
```
> Çalan şarkıda bir sanatçı eşleştiğinde ekrana yerel macOS pop-up penceresi gelir ve *"Şarkıyı Geç"* / *"Dinlemeye Devam Et"* seçenekleri sunulur. Ayrıca `http://localhost:8080` adresinden web paneline erişebilirsiniz.

### 2. Otomatik Geçiş Modu (Sormadan Hemen Atla)
```bash
./medya-filter -auto-skip
```

### 3. Belirli Bir Kategoriye Göre Filtreleme
Sadece şarkıcıları veya belirli kategorileri izlemek isterseniz:
```bash
./medya-filter -category "Şarkıcı"
```
*(Desteklenen Kategoriler: `Şarkıcı`, `Oyuncu`, `Yazar`, `Gazeteci`, `Influencer`, `Spor`, `Siyasi`, `Marka`, `Aktivist`, `Din Adamı`, `Diğer`)*

### 4. Tek Bir Sanatçıyı Test Etme (Anlık Sorgu)
```bash
./medya-filter -test "Cem Adrian"
./medya-filter -test "Ceza"
./medya-filter -test "Ahmet Aslan"
```

### 5. Tüm Parametreler

| Bayrak | Tip | Varsayılan | Açıklama |
|---|---|---|---|
| `-auto-skip` | bool | `false` | Onay sormadan şarkıları doğrudan geçer |
| `-no-dialog` | bool | `false` | macOS pop-up onay penceresini kapatır |
| `-no-web` | bool | `false` | Web arayüzünü devre dışı bırakır |
| `-port` | int | `8080` | Web kontrol paneli portu |
| `-category` | string | `HEPSİ` | Yalnızca belirtilen kategoriyi denetler |
| `-interval` | duration | `1.5s` | Medya kontrol aralığı |
| `-sync` | bool | `false` | Başlangıçta API ile zorunlu tam eşitleme yapar |
| `-test` | string | `""` | Verilen sanatçı adını test edip çıkar |

---

## 📁 Proje Mimarisi

```
medya-filter/
├── main.go                # Giriş noktası, CLI argümanları ve sinyal yönetimi
├── data/                  # Yerel JSON önbelleği (persons.json, session.json)
├── pkg/
│   ├── models/            # Person, MediaInfo, MatchResult veri yapıları
│   ├── api/               # OE List API istemcisi (Auth, Sync, Fetch, Cache)
│   ├── matcher/           # Türkçe normalizasyonu, alias ayrıştırma, çoklu filtreleme
│   ├── media/             # macOS Spotify, Apple Music ve Chrome medya dedektörü/kontrolcüsü
│   ├── notifier/          # macOS pop-up dialogları ve bildirimleri
│   └── web/               # Gömülü Web Kontrol Paneli (HTML, CSS, JS)
└── README.md
```
