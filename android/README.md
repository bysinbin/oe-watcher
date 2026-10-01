# OE Watcher - Android Native Media Filter & Auto-Skipper

Android için sistem genelinde (**Spotify, YouTube Music, YouTube, Apple Music, Deezer, Tidal, SoundCloud, Tarayıcı vb.**) çalan medyayı arka planda dinleyip kara listedeki sanatçı ve şarkıları **anında otomatik olarak bir sonraki şarkıya atlatan (`skipToNext()`)** yerel Android uygulaması.

---

## 🚀 Özellikler

1. **Evrensel Medya Dinleme (`MediaSessionManager` + `NotificationListenerService`):**
   - Hangi müzik uygulaması olursa olsun Android bildirim ve medya kontrol oturumlarını anında algılar.
   - Arka planda pil tüketmeden, yalnızca yeni bir şarkı başladığında tetiklenir.
2. **Otomatik Atlama (Auto-Skip):**
   - Kara listedeki bir sanatçı veya parça başladığında mikrosaniyeler içinde `transportControls.skipToNext()` komutu gönderir.
   - Bildirim olarak hangi sanatçı ve şarkının engellendiğini haber verir.
3. **Türkçe Karakter & Regex Desteği:**
   - `ı/i`, `ş/s`, `ç/c`, `ğ/g`, `ü/u`, `ö/o` gibi Türkçe karakter farklarını ve varyasyonları otomatik tolere eden normalizasyon motoru.
4. **Çevrimdışı & Bağımsız (Offline-First):**
   - `default_persons.json` veritabanı uygulamanın içine gömülüdür (`assets/default_persons.json`).
   - Herhangi bir sunucu veya bilgisayar bağlantısına ihtiyaç duymaz, tamamen telefonunuzda çalışır.
5. **Modern Jetpack Compose Dark UI:**
   - Koyu kırmızı / arduvaz şık karanlık tema.
   - Şu an çalan şarkı ve uygulama kartı.
   - Otomatik atlama açma/kapama anahtarı.
   - İstatistikler ve engelleme geçmişi akışı.

---

## 📲 Telefona Kurulum

Hazır derlenen APK dosyası şu dizindedir:
```
android/app/build/outputs/apk/debug/app-debug.apk
```

### Yöntem A: USB Kablosu & ADB ile Tek Komutla Kurulum
Telefonunuzda **Geliştirici Seçenekleri** ve **USB Hata Ayıklama** açıksa:
```bash
/Users/feritetem/Library/Android/sdk/platform-tools/adb install -r /Users/feritetem/Desktop/medya-filter/android/app/build/outputs/apk/debug/app-debug.apk
```

### Yöntem B: Dosyayı Telefona Gönderip Yükleme
1. `app-debug.apk` dosyasını WhatsApp, Telegram, Google Drive, Quick Share veya USB ile telefonunuza aktarın.
2. Telefonda dosyaya tıklayıp **Yükle** deyin (Bilinmeyen kaynaklardan yüklemeye izin verin).

---

## ⚙️ İlk Açılış ve İzin Verme

1. **OE Watcher** uygulamasını açın.
2. Ana ekranda beliren **"Bildirim & Medya İzni Gerekli"** uyarısındaki **"İzin Ver"** butonuna dokunun.
3. Açılan Android sistem ayarlarında **OE Watcher**'ı bulun ve izni **Açık** konuma getirin.
4. Artık Spotify, YouTube Music veya herhangi bir oynatıcıdan müzik açtığınızda uygulama arka planda sessizce koruma sağlayacaktır!
