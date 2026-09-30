# OE Watcher // Mozilla Firefox Eklentisi 🦊🛡️

Bu eklenti, Mozilla Firefox üzerinde gezinirken (Twitter/X, Instagram, YouTube, haber portalları ve web siteleri) metinlerde veya videolarda kara listedeki isimler geçtiğinde içeriği otomatik olarak filtreler, flu (blur) hale getirir ve görmeden önce onayınızı sorar.

OE Watcher yerel servisi (`http://127.0.0.1:8080`) ile canlı senkronize çalışır.

---

## 🚀 Firefox'a Yükleme Adımları (Geliştirici / Geçici Eklenti)

1. **Firefox** tarayıcınızı açın.
2. Adres çubuğuna **`about:debugging#/runtime/this-firefox`** yazıp Enter'a basın.
3. Sayfadaki **"Geçici Eklenti Yükle..."** (Load Temporary Add-on...) butonuna tıklayın.
4. Dosya seçici penceresinde bu klasördeki **`manifest.json`** dosyasını seçin.
5. Eklenti anında Firefox araç çubuğuna (`🎧 OE Watcher`) eklenecektir!

---

## 📦 Dağıtım Paketi (.xpi / .zip) Oluşturma

Tüm eklenti dosyalarını imzalamak veya dağıtmak için `.xpi` paketi oluşturabilirsiniz:

```bash
./build_extensions.sh
```

Bu komut:
- `dist/oe-watcher-firefox.xpi` (Firefox eklenti paketi)
- `dist/oe-watcher-chrome.zip` (Chrome/Edge eklenti paketi)

dosyalarını otomatik olarak oluşturur.

---

## ⚙️ Özellikler

* **Akıllı Bulanıklaştırma (Smart Shielding):** Kara listedeki kişi ismi geçtiğinde kartı, tweet'i veya video bloğunu gizler.
* **İçeriği Göster Butonu:** İncelemek istediğinizde *"👁️ İçeriği Göster"* butonuna basarak tek tıkla açabilirsiniz.
* **Beyaz Liste / Muafiyet:** *"🛡️ Bu Sayfayı Muaf Tut"* seçeneği ile istediğiniz URL'yi filtre dışı bırakabilirsiniz.
* **Yerel API Senkronizasyonu:** OE Watcher masaüstü uygulamasında yaptığınız özel kural ve kişi eklemeleri anında Firefox eklentisine aktarılır.
* **Çevrimdışı Koruma:** Masaüstü uygulaması kapalı olsa bile eklenti dahili 570+ kişilik önbelleğiyle korumaya devam eder.
