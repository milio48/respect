# 🛡️ Arsitektur In-Memory Virtual Host & Enkripsi Payload

Salah satu inovasi terbesar pada Respect Desktop adalah arsitektur **In-Memory Virtual Host (Trailer V2)**. Pendekatan ini memungkinkan Respect mengemas ribuan file frontend web (HTML, JS, CSS, aset gambar, font) ke dalam satu file `.exe` mandiri tanpa pernah mengekstrak file ke disk dan tanpa membuka port jaringan lokal.

---

## 1. Komparasi Tiga Pendekatan Desktop Web

| Parameter | ❌ 1. File Protocol (`file:///`) | ⚠️ 2. Disk Local Server (Tradisional) | 🌟 3. In-Memory Virtual Host (`http://app/`) | 🚀 4. In-Memory Local Server (`--server`) |
| :--- | :--- | :--- | :--- | :--- |
| **Kinerja & Kecepatan** | Terbatas (I/O lambat) | Sedang (I/O disk) | **Maksimal (0 ms dari RAM)** | **Sangat Cepat (RAM + Loopback)** |
| **ES Modules (`import/export`)** | ❌ Diblokir CORS Chromium | ✅ Berfungsi | **✅ Berfungsi sempurna** | **✅ Berfungsi sempurna** |
| **Fetch API & Relative URL** | ❌ Diblokir (`null` origin) | ✅ Berfungsi | **✅ Berfungsi sempurna** | **✅ Berfungsi sempurna** |
| **Secure Context (`crypto.subtle`)** | ❌ `isSecureContext: false` | ✅ `isSecureContext: true` | ⚠️ `isSecureContext: false` | **✅ `isSecureContext: true`** |
| **Ekstraksi ke Hard Drive** | Menulis file ke `%TEMP%` | **Wajib ekstrak** ribuan file | **NOL EKSTRAKSI (100% di RAM)** | **NOL EKSTRAKSI (100% di RAM)** |
| **Port Jaringan TCP** | Nol | Membuka Port tetap di OS | **NOL PORT (Tidak ada socket)** | **Port Acak Loopback 127.0.0.1:0** |
| **Isolasi Keamanan** | Terisolasi | Rentan port scanner lokal | **100% Terisolasi di Respect** | **Terikat ketat ke 127.0.0.1** |
| **Bentrok VPN / Proxy** | Tidak | Rentan bentrok proxy | **100% Kebal (Bebas proxy)** | **100% Kebal (Bypass localhost)** |

> **Fleksibilitas Respect:** Anda dapat menggunakan mode default **In-Memory Virtual Host (`http://app/`)** untuk isolasi 0-port maksimal, atau menyalakan flag `--server` (atau `--mode server`) jika aplikasi Anda memerlukan fitur modern yang mewajibkan **Secure Context** (seperti `crypto.subtle`, `crypto.randomUUID()`, dan `navigator.clipboard`) tanpa mengorbankan prinsip zero-disk file footprint.

---

## 2. Struktur Binary Payload Trailer V2

Format binary Respect menggunakan teknik *PE Overlay Trailer* yang diperluas untuk mendukung arsip multi-file terkompresi Zstandard level 19 dan terenkripsi AES-256-GCM:

```
+-------------------------------------------------------------------------+
|                  respect.exe / respect-lite.exe                         |
|  - Windows PE Base Executable (Go Runtime + Browser Wrapper)            |
|  - Embedded Engine Assets (blink.dll.zst, icon, rcedit.exe)             |
+-------------------------------------------------------------------------+
|  ENCRYPTED PAYLOAD BLOB (AES-256-GCM Terotentikasi):                    |
|  ├── 12-byte Random Nonce                                               |
|  ├── Ciphertext TAR Archive (Zstandard level 19):                       |
|  │   ├── respect.json     <-- Konfigurasi & title aplikasi              |
|  │   ├── index.html       <-- Entry point HTML                          |
|  │   └── assets/          <-- Bundle JS, CSS, Font, Image               |
|  └── 16-byte GCM Authentication Tag                                     |
+-------------------------------------------------------------------------+
|  Panjang Payload Encrypted (8-byte Little Endian uint64)                 |
+-------------------------------------------------------------------------+
|  Magic Identifier (17-byte): "RESPECT_PAYLOAD_V2"                       |
+-------------------------------------------------------------------------+
```

### Kompatibilitas Mundur (*Backward Compatibility*):
- Binary membaca 17-byte terakhir file executable untuk mendeteksi tanda pengenal (*Magic String*).
- Jika berakhiran `RESPECT_PAYLOAD_V1`: menggunakan parser JSON tunggal legacy.
- Jika berakhiran `RESPECT_PAYLOAD_V2`: menggunakan pipeline dekripsi AES-256-GCM dan streaming dekompresi ZSTD TAR in-memory.

---

## 3. Perlindungan Anti-Reverse Engineering (AES-256-GCM)

Kode sumber frontend web (JavaScript logika bisnis, endpoint API rahasia, token otentikasi) yang dikemas di dalam binary executable rentan terbongkar jika disimpan dalam teks mentah (*plaintext*). Respect menerapkan skema proteksi biner:

### A. Penurunan Kunci Dinamis (*Dynamic Key Derivation*)
Kunci enkripsi **tidak pernah disimpan secara hardcoded** di dalam file binary. Kunci 256-bit diturunkan saat runtime melalui fungsi `DeriveKey`:
```go
DeriveKey(peSample []byte, totalSize int64)
```
1. **Sampling PE Header**: Mengambil 4.096 byte pertama dari PE header biner aplikasi itu sendiri.
2. **Ukuran File Fisik**: Menggabungkan ukuran file executable pasca-injeksi metadata `rcedit`.
3. **Obfuscated HMAC-SHA256**: Menggabungkan sampling biner dengan *seed salt* terobfuskasi (XOR-masked) untuk menghasilkan kunci AES-256 yang unik bagi setiap file executable yang dibangun.

### B. Otentikasi Integritas Data
Enkripsi menggunakan mode **Galois/Counter Mode (GCM)**:
- **Tampering Protection**: Jika ada pihak ketiga yang mencoba memodifikasi atau mengutak-atik byte payload di dalam file `.exe`, verifikasi GCM Tag akan gagal secara kriptografis dan eksekusi langsung dihentikan.
- **Minimal Overhead**: Enkripsi AES-256-GCM hanya menambahkan tepat **28 byte** (12-byte Nonce + 16-byte Auth Tag) ke ukuran file akhir.

---

## 4. Alur Kerja Virtual Host di Kedua Engine

### A. Respect Modern (Chromium 132)
1. Seluruh file hasil ekstraksi TAR dimuat ke dalam `map[string][]byte` di memori RAM Go.
2. Callback native `mbOnLoadUrlBegin` mencegat setiap request yang diawali `http://app/*` atau `http://app.local/*`.
3. MIME type ditentukan secara otomatis via `tarball.DetectMIME(relPath)` dan dikirim ke Chromium melalui `mbNetSetMIMEType`.
4. Isi file disuntikkan langsung ke job jaringan Chromium melalui fungsi C `mbNetSetData` tanpa menyentuh soket jaringan.
5. Setiap request 404 (seperti `favicon.ico`) dicegat langsung di memori untuk mencegah kebocoran request ke stack jaringan eksternal.

### B. Respect Lite (Miniblink 49)
1. File in-memory dibungkus ke dalam abstraksi filesystem Go `tarball.NewMemoryFS(files)` yang mengimplementasikan antarmuka standar `io/fs.FS`.
2. Engine Miniblink 49 mengikat filesystem memori tersebut menggunakan fungsi `app.Resource.Bind("app", memFS)`.
3. Seluruh request ke domain virtual `http://app/index.html` dilayani secara instan dari RAM.

---

[⬅️ Sebelumnya: 2. Perbandingan Edisi](2-perbandingan-edisi.md) | [Daftar Isi](index.md) | [Selanjutnya: 4. Lifecycle & Alur Internal ➡️](4-lifecycle.md)
