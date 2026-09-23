# ⚡ Dokumentasi Resmi Respect Desktop

Selamat datang di pusat dokumentasi resmi **Respect Desktop**.

Respect Desktop adalah utilitas Windows modern yang dirancang untuk mengubah website, aplikasi web (React, Vue, Svelte, Vite), atau file HTML menjadi aplikasi desktop (`.exe`) mandiri siap pakai — **tanpa perlu instalasi server, runtime Node.js, atau coding tambahan**.

---

## 📚 Daftar Isi Dokumentasi

Dokumentasi ini dibagi menjadi beberapa bab terstruktur untuk memudahkan navigasi:

### [1. Panduan Penggunaan Lengkap](1-panduan-penggunaan.md)
Pelajari cara membangun aplikasi desktop menggunakan dua metode:
- **GUI Desktop Builder**: Menggunakan antarmuka visual grafis, memilih mode sumber (URL, Folder App V2, HTML Inline, File HTML Lokal), memilih file/folder dengan dialog native Windows, dan fitur *Drag & Drop*.
- **Baris Perintah (CLI)**: Automasi pembuatan binary `.exe` melalui terminal Command Prompt, PowerShell, atau skrip CI/CD dengan parameter lengkap.

### [2. Perbandingan Edisi & Standar Web](2-perbandingan-edisi.md)
Ketahui perbedaan mendalam antara dua edisi Respect Desktop:
- **Respect Modern (`respect.exe`)**: Didukung oleh engine Chromium 132 untuk visual web kaya, animasi mutakhir, Tailwind CSS, dan standar ES2023+.
- **Respect Lite (`respect-lite.exe`)**: Didukung oleh Miniblink 49 untuk konsumsi RAM super ringan (~25–40 MB) dan kompatibilitas sistem lawas (Windows XP s.d. 11).
- **Hasil Riset 88 Standar Web**: Matriks pengujian menyeluruh terhadap JavaScript, CSS, HTML5, dan Web APIs.

### [3. Arsitektur In-Memory Virtual Host](3-arsitektur-virtual-host.md)
Mekanisme runtime canggih untuk menyajikan aplikasi frontend web:
- **Zero-Disk Extraction**: File web tidak pernah diekstrak ke hard drive (%TEMP%).
- **Zero-Network Socket**: Tidak ada port TCP lokal yang dibuka (`127.0.0.1`), 100% aman dari bentrok port atau pencegatan proxy/VPN.
- **Enkripsi Payload Trailer V2**: Perlindungan anti-reverse engineering menggunakan AES-256-GCM terotentikasi dan Zstandard.

### [4. Lifecycle & Alur Eksekusi Internal](4-lifecycle.md)
Dokumentasi teknis lengkap mengenai:
- Titik percabangan arsitektur antara edisi Modern dan Lite.
- Pipeline eksekusi berdampingan (*Side-by-side execution*).
- Penanganan memori, Win32 message loop, isolasi sandbox cookies/LocalStorage, dan mekanisme *self-replicating binary*.

### [5. Panduan Pengembang & Kompilasi](5-panduan-pengembang.md)
Panduan bagi developer yang ingin mengompilasi Respect langsung dari source code:
- Prasyarat sistem (Go 1.25+, Windows 10/11).
- Skrip otomasi PowerShell `build.ps1` (Mode Slim dev vs Mode Embed rilis).
- Kompilasi manual menggunakan Go build tags (`v132`, `embed132`).
- Struktur repository dan pipeline rilis otomatis GitHub Actions.

---

## 🔗 Tautan Eksternal & Kredit Komunitas

Respect Desktop dapat terwujud berkat karya luar biasa dari komunitas open source:
- [Miniblink 49 C++ Core Engine](https://github.com/weolar/miniblink49/) oleh **Weolar**.
- [epkgs/blink Go Binding](https://deepwiki.com/epkgs/blink).
- Repositori Utama Respect: [https://github.com/milio48/respect](https://github.com/milio48/respect).
