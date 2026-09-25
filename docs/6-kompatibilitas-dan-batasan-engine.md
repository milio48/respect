# 🧭 Kompatibilitas Standar Web, Media & Batasan Engine

Dokumen ini menjelaskan secara transparan mengenai **standar web yang didukung, fitur media native, sistem keamanan Secure Context, serta batasan arsitektur engine Miniblink** di dalam Respect Desktop.

---

## 1. Ikhtisar Arsitektur Engine

Respect Desktop menggunakan pendekatan runtime ganda:
- **Respect Modern (`respect.exe`)**: Menggunakan Miniblink Chromium 132 core (64-bit), menghadirkan visual CSS modern, Tailwind CSS v3/v4, nesting, dan JavaScript ES2023+.
- **Respect Lite (`respect-lite.exe`)**: Menggunakan Miniblink 49 core (32-bit & 64-bit), dirancang untuk footprint memori ultra-rendah (~25–40 MB RAM) dan kompatibilitas Windows lawas (XP s.d. 11).

Untuk menjaga ukuran binary tetap berada di kisaran **~8–12 MB** (bukan 150 MB seperti Electron atau CEF), Miniblink membuang beberapa subsistem Chromium yang sangat besar (seperti ANGLE GPU, WebRTC, dan Chromium Media Pipeline). Sebagai gantinya, Respect Desktop menyuntikkan **lapisan shim native berbasis Win32 dan Media Foundation** yang menjembatani browser dengan API bawaan Windows tanpa menambah ukuran file sama sekali (0 KB overhead).

---

## 2. Pemutaran Media Native (Video & Audio)

Secara bawaan, engine embedded Miniblink tidak memiliki media decoder internal. Respect Desktop mengatasi batasan ini dengan mengintegrasikan **Teknik 1: Windows Media Foundation (`mfplay.dll`) & HWND Overlay**.

### A. Format yang Didukung
- **Video**: MP4 (H.264 Video + AAC Audio), WMV, AVI.
  - Video diputar menggunakan akselerasi perangkat keras DirectX (DirectX Video Acceleration / DXVA2) bawaan Windows.
  - Tidak membebani CPU karena decoding dilakukan langsung oleh GPU Windows.
- **Audio**: MP3, WAV, AAC, M4A, OGG, WMA (dengan fallback otomatis ke WinMM MCI).

### B. Penggunaan Tag HTML5 Standar
Pengembang tidak perlu mempelajari sintaks khusus. Cukup gunakan tag `<video>` atau `<audio>` HTML5 biasa:

```html
<!-- Pemutar video lokal atau online -->
<div class="video-container" style="width: 640px; height: 360px; margin: auto;">
  <video src="assets/intro.mp4" controls style="width: 100%; height: 100%;"></video>
</div>
```

Jendela overlay native akan secara otomatis:
1. Menyesuaikan posisi dan ukuran elemen HTML berdasarkan `getBoundingClientRect()`.
2. Mengikuti pergerakan elemen saat halaman di-scroll atau jendela di-resize.
3. Menghilang secara otomatis saat elemen disembunyikan (`display: none`) atau dihapus dari DOM.

### C. Kontrol Penuh via JavaScript
Seluruh API pemutar standar W3C dapat dikontrol lewat skrip JavaScript:

```javascript
const vid = document.querySelector('video');

// 1. Play & Pause
vid.play().then(() => console.log("Sedang memutar"));
vid.pause();

// 2. Seek / Lompat Detik
vid.currentTime = 15.5; // Lompat ke detik 15.5
vid.currentTime += 5;   // Maju 5 detik

// 3. Atur Volume
vid.volume = 0.8; // Nilai antara 0.0 s.d. 1.0

// 4. Informasi Pemutaran
console.log("Durasi total:", vid.duration);
console.log("Posisi:", vid.currentTime);
console.log("Sedang dijeda?", vid.paused);
console.log("Sudah selesai?", vid.ended);

// 5. Event Listener Standar
vid.addEventListener('timeupdate', () => {
  // Berguna untuk menggerakkan custom seek bar slider
});
vid.addEventListener('ended', () => {
  console.log("Video selesai diputar!");
});
```

---

## 3. Mode Runtime: Virtual Host vs Local Server

Respect menyediakan dua pilihan penyajian aplikasi web:

### A. Virtual Host (`http://app`) — Default
- **In-Memory Zero-Socket**: Seluruh asset disajikan langsung dari RAM tanpa membuka port TCP (`127.0.0.1`).
- **Keamanan Total**: Kebal dari scanning port, firewall alert, dan bentrok port dengan aplikasi lain.
- **Batasan**: Browser menganggap origin ini sebagai Non-HTTPS (Non-Secure Context), sehingga fitur seperti Web Crypto API (`crypto.subtle`) dan Service Worker dinonaktifkan oleh standar W3C.

### B. Local Server (`--server` / `http://127.0.0.1:<port>`)
- **Mengaktifkan Secure Context**: Menyediakan akses penuh ke:
  - `isSecureContext = true`
  - `crypto.subtle` (SHA-256, HMAC, AES-GCM, RSA)
  - `crypto.randomUUID()`
  - Service Worker & PWA Caching
- **Sistem Keamanan One-Time Token & HttpOnly Cookie**:
  Untuk mencegah aplikasi lain atau browser eksternal di komputer pengguna mengakses server lokal Respect:
  1. Saat aplikasi dibuka, engine memuat URL unik dengan token acak 128-bit: `http://127.0.0.1:<port>/?token=<random-token>`.
  2. Server lokal langsung memverifikasi token, menyetel cookie `respect_auth` dengan flag `HttpOnly; SameSite=Strict`, dan melakukan redirect 302 ke halaman utama.
  3. Setiap permintaan selanjutnya wajib memiliki cookie tersebut. Permintaan tanpa cookie atau dari luar aplikasi akan langsung ditolak dengan status **HTTP 403 Forbidden**.

---

## 4. Matriks Kompatibilitas Fitur

| Kategori | Fitur Web | Status di Respect Desktop | Solusi / Catatan |
|---|---|---|---|
| **Layout & Visual** | HTML5 Semantic, Canvas 2D | ✅ Didukung 100% | Skor 19/22 pada suite capability |
| | CSS Modern (Flexbox, Grid, Nesting, `:has`) | ✅ Didukung 100% | Skor 32/32 pada suite capability |
| | Tailwind CSS (v3 / v4) | ✅ Didukung 100% | Berjalan mulus di edisi Modern |
| **Media** | Video MP4 (H.264 + AAC) | ✅ Didukung | Akselerasi DirectX via Media Foundation |
| | Audio (MP3, WAV, AAC, M4A) | ✅ Didukung | Native Media Foundation & MCI |
| | Streaming Adaptif (MSE / HLS / DASH) | ❌ Tidak Didukung | Butuh Chromium Media Pipeline penuh |
| | DRM / Proteksi Konten (EME/Widevine) | ❌ Tidak Didukung | Tidak bisa memutar Netflix/Spotify Web |
| **Grafis Lanjutan** | WebGL 1.0, 2.0, WebGPU | ❌ Tidak Didukung | Arsitektur engine tanpa GPU Process / ANGLE |
| **JavaScript & API**| ES Modules (`import`, `type=module`) | ✅ Didukung 100% | Didukung di Virtual Host maupun Server |
| | WebAssembly (`.wasm`) | ✅ Didukung 100% | Kompilasi instan via V8 |
| | IndexedDB, LocalStorage, Cookies | ✅ Didukung 100% | Persisten di folder `%APPDATA%` |
| | Text-to-Speech (`speechSynthesis`) | ✅ Didukung | Diakali via Windows SAPI native |
| | Clipboard (`navigator.clipboard`) | ✅ Didukung | Diakali via Win32 API clipboard |
| | Gamepad API (Xbox / XInput) | ✅ Didukung | Diakali via XInput Win32 |
| | Notifikasi (`Notification`) | ✅ Didukung | Diteruskan ke native MessageBox |
| | Web Crypto API (`crypto.subtle`) | ✅ Didukung | Wajib mengaktifkan mode `--server` |
| **Komunikasi & Hardware** | WebRTC Video Call P2P | ❌ Tidak Didukung | Membutuhkan stack C++ WebRTC penuh |
| | Kamera & Mic (`getUserMedia`) | ❌ Tidak Didukung | Pipeline media input tidak ada di engine |
| | Geolocation API | ❌ Dinonaktifkan | Sengaja tidak memakai estimasi IP palsu |

---

## 5. Panduan Praktis Pemilihan Fitur untuk Pengembang

1. **Aplikasi Dashboard, Point of Sale (POS), Kasir, Sistem ERP**:
   - Gunakan mode default (**Virtual Host**). Performa paling instan, memori hemat, dan tidak memicu prompt firewall Windows.
2. **Aplikasi Pemutar Media, E-Learning, Presentasi Video/Audio**:
   - Gunakan format file `.mp4` (H.264/AAC) untuk video dan `.mp3` atau `.wav` untuk audio.
   - Gunakan tag `<video>` biasa dengan kontrol CSS dan JavaScript standar.
3. **Aplikasi Enkripsi Data, Password Vault, SQLite WebAssembly**:
   - Gunakan flag `--server` saat build agar Web Crypto API (`crypto.subtle`) aktif dalam Secure Context.
4. **Jika Anda Membutuhkan 3D Canvas (Three.js) atau Video Call (WebRTC)**:
   - Respect Desktop **bukan** pilihan yang tepat untuk kebutuhan game 3D WebGL atau video call WebRTC karena ketiadaan GPU process internal. Untuk kebutuhan tersebut, pengembang disarankan menggunakan runtime browser penuh seperti WebView2 atau Electron.
