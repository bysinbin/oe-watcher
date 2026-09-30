# 🛡️ OE Watcher // Chrome Akıllı İçerik Filtresi Uzantısı

Bu Chrome uzantısı, internette (X / Twitter, YouTube, Ekşi Sözlük, haber siteleri ve bloglarda) gezinirken **kara listedeki sanatçı veya kişilerin geçtiği içerikleri otomatik olarak tespit eder**.

İçeriği doğrudan silmek yerine şık bir cam efektiyle (blur) buzlar ve size sorar:
> **"🛡️ OE Watcher: Bu içerikte kara listedeki [Kişi Adı] geçiyor. Bu içeriği görmek istiyor musunuz?"**
> `[ 👁️ İçeriği Göster ]`  `[ ✕ Gizli Tut ]`

---

## 🚀 1 Dakikada Kurulum

1. **Google Chrome** tarayıcınızı açın.
2. Adres çubuğuna şunu yazıp Enter'a basın:
   ```
   chrome://extensions/
   ```
3. Sağ üst köşedeki **"Geliştirici modu" (Developer mode)** anahtarını açık konuma getirin.
4. Sol üstte beliren **"Paketlenmemiş öğe yükle" (Load unpacked)** butonuna tıklayın.
5. Açılan dosya seçim penceresinden şu klasörü seçin:
   ```
   /Users/feritetem/Desktop/medya-filter/extension
   ```
6. **Tebrikler!** Uzantı anında yüklenecektir. Chrome araç çubuğundaki yapboz simgesinden uzantıyı sabitleyebilirsiniz.

---

## ✨ Özellikler

* **Akıllı DOM Taraması:** X/Twitter tweetleri, YouTube video başlık ve kartları, haber paragrafları ve blog yazılarını algılar.
* **Sonsuz Kaydırma (Infinite Scroll) Desteği:** Sayfayı aşağı kaydırdıkça yeni gelen gönderileri performans kaybı olmadan anında yakalar.
* **Türkçe Harf Uyumluluğu:** İ/i, I/ı, Ş/ş, Ğ/ğ gibi Türkçe karakterleri kusursuz eşleştirir.
* **Kelime Sınırı Güvenliği:** "Cem" ismi "acemi" kelimesiyle, "Ali" ismi "kalibre" ile yanlışlıkla tetiklenmez.
* **Yerel OE Watcher Entegrasyonu:** Masaüstünüzde çalışan `OE Watcher` uygulamasından kara listeyi, aliasları ve beyaz listeyi otomatik çeker.
* **Kullanıcı Kontrolü:** "İçeriği Göster" butonuna bastığınızda buzlama o içerik için kalkar; isterseniz "Yeniden Gizle" ile kapatabilirsiniz.
