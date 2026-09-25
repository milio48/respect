# 🎯 Perbandingan Edisi & Standar Web

Respect Desktop dirilis dalam dua edisi utama yang didesain untuk kebutuhan yang berbeda: **Respect Modern** dan **Respect Lite**. Kedua edisi ini hadir dalam bentuk satu file executable (`.exe`) mandiri tanpa memerlukan instalasi tambahan.

---

## 1. Matriks Perbandingan Ringkas

| Dimensi Fitur | 🌟 Respect Modern (`respect.exe`) | 🪶 Respect Lite (`respect-lite.exe`) |
| :--- | :--- | :--- |
| **Engine Inti** | **Chromium 132** (Blink Modern 2025/2026) | **Miniblink 49** (Fork WebKit Ringan) |
| **Ukuran File Binary** | **~31–32 MB** *(Engine ZSTD tertanam, x64)* | **~32 MB** *(x64)* / **~30 MB** *(x86 32-bit, ZSTD tertanam)* |
| **Konsumsi Memori (RAM)** | ~80–120 MB (Sesuai beban tab Chromium) | **~25–40 MB (Sangat hemat dan enteng)** |
| **Dukungan OS Windows** | Windows 7 SP1, 8, 10, 11 (64-bit) | Windows XP SP3, 7, 8, 10, 11 (32-bit & 64-bit) |
| **Kompatibilitas CSS** | **100% Dukungan Modern** (CSS Grid, Variables, Tailwind CSS) | CSS3 Dasar (Tanpa CSS Grid, tanpa CSS Variables) |
| **Kompatibilitas JavaScript** | **ES2023+ Lengkap** (ES Modules, Optional Chaining) | ES6 Dasar |
| **Kecepatan Start Awal** | **0 ms** (Instan setelah dekompresi cache awal) | **Instan** |
| **Karakter Penggunaan** | Dashboard modern, aplikasi SaaS, visual interaktif | Aplikasi kasir (POS), utilitas kantor, PC berspesifikasi rendah |

---

## 2. Hasil Riset 88 Standar Web Modern

Untuk memberikan gambaran yang akurat mengenai batas kemampuan masing-masing engine, telah dilakukan pengujian komparasi mendalam terhadap **88 fitur standar web modern**:

| Kategori Pengujian | 🪶 Respect Lite (v49) | 🌟 Respect Modern (v132) | Catatan Penting |
| :--- | :---: | :---: | :--- |
| **JavaScript & ECMAScript** | **19 / 24 (79%)** | **24 / 24 (100%)** | Modern lulus penuh standar ES2020+ (Optional Chaining `?.`, Nullish Coalescing `??`, ES Modules `import/export`, `WeakRef`, `BigInt`, `Promise.allSettled`). |
| **CSS Modern & Layout** | **6 / 22 (27%)** | **22 / 22 (100%)** | **Perbedaan Terbesar:** Lite tidak mendukung CSS Grid, CSS Variables (`var(--color)`), `backdrop-filter`, `gap`, `aspect-ratio`, `:has()`. Modern mendukung 100% framework modern seperti Tailwind CSS, Bootstrap 5, dan Material UI. |
| **HTML5 Core APIs** | **18 / 32 (56%)** | **21 / 32 (66%)** | Keduanya mendukung Canvas 2D, Web Audio, SVG, Web Workers, SessionStorage, dan LocalStorage. |
| **PWA & OS Integrations** | **1 / 10 (10%)** | **3 / 10 (30%)** | Batasan arsitektur webview desktop: API tingkat sistem operasi seperti Push Notifications, Background Sync, dan Web Bluetooth tidak diekspos secara default. |
| **TOTAL SKOR KOMPATIBILITAS** | **44 / 88 (50.0%)** | **70 / 88 (79.5%)** | **Modern unggul mutlak pada rendering grafis visual dan fleksibilitas framework frontend.** |

---

## 3. Rincian Fitur Utama yang Didukung

### A. Respect Modern (Chromium 132)
- **Tailwind CSS & Utility-First Frameworks**: Mendukung penuh seluruh kelas utility, arbitrary values, dan pseudo-selectors.
- **Modern Single-Page Applications (SPA)**: Mendukung build modern dari Vite, Next.js (Static Export), Nuxt (Static), SvelteKit, dan React Router v6+.
- **Grafik Interaktif & Visualisasi Data**: Mendukung Chart.js, ECharts, ApexCharts, Canvas 2D animasi, dan Three.js (WebGL).
- **Akselerasi Scrolling Natural**: Dilengkapi injeksi hook akselerasi mouse wheel yang membuat pergerakan scroll terasa mulus dan responsif di Windows.

### B. Respect Lite (Miniblink 49)
- **Kompatibilitas Komputer Lawas**: Dapat dijalankan di mesin kasir (POS), pabrik, atau komputer kantor lama berbasis Windows 7, Windows 8, bahkan Windows XP SP3.
- **Konsumsi Memori Minimalis**: Hanya membutuhkan ~25–40 MB RAM, menjadikannya pilihan sempurna untuk aplikasi latar belakang (*background utility*) atau perangkat dengan kapasitas RAM terbatas (2 GB - 4 GB).
- **Dukungan Arsitektur 32-bit (x86)**: Menyediakan binary khusus `respect-lite-x86.exe` untuk sistem operasi Windows 32-bit.

---

## 4. Panduan Pengambilan Keputusan: Pilih yang Mana?

```
                     Apakah aplikasi Anda menggunakan
                  Tailwind CSS, React, Vue, atau Vite?
                                 |
                 +---------------+---------------+
                 |                               |
                YA                             TIDAK
                 |                               |
                 v                               v
         [ Gunakan Modern ]            Apakah aplikasi harus
         (respect.exe v132)           berjalan di Windows XP /
                                       komputer RAM < 4 GB?
                                                 |
                                 +---------------+---------------+
                                 |                               |
                                YA                             TIDAK
                                 |                               |
                                 v                               v
                         [ Gunakan Lite ]                [ Rekomendasi:  ]
                       (respect-lite.exe)                [ Modern v132   ]
```

- **Pilih `respect.exe` (Modern)** jika prioritas Anda adalah:
  1. Tampilan visual modern, animasi yang mulus, dan desain responsif.
  2. Mengemas framework JavaScript terkini (Vite, React, Vue, Svelte).
  3. Membuka website pihak ketiga modern (Google Apps, Notion, Canva, WhatsApp Web, ChatGPT).

- **Pilih `respect-lite.exe` (Lite)** jika prioritas Anda adalah:
  1. Konsumsi RAM yang serendah mungkin (di bawah 50 MB).
  2. Harus berjalan di sistem operasi lawas (Windows XP / Windows 7 32-bit).
  3. Aplikasi kasir toko (POS), inventaris gudang, atau sistem pelaporan lokal sederhana.

---

[⬅️ Sebelumnya: 1. Panduan Penggunaan](1-panduan-penggunaan.md) | [Daftar Isi](index.md) | [Selanjutnya: 3. Arsitektur Virtual Host ➡️](3-arsitektur-virtual-host.md)
