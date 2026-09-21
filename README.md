# ⚡ Respect Desktop

**Respect Desktop** adalah alat praktis untuk mengubah situs Web, aplikasi web lokal, atau kode HTML menjadi aplikasi desktop Windows mandiri (`.exe`) siap pakai tanpa perlu instalasi server atau coding tambahan.

---

## 🎯 Panduan Memilih Edisi (Untuk Pengguna)

Respect tersedia dalam dua edisi rilis siap pakai:

| Fitur | 🌟 **Respect Modern (`respect.exe`)** | 🪶 **Respect Lite (`respect-lite.exe`)** |
|---|---|---|
| **Kelebihan Utama** | Performa tinggi, mendukung web modern & grafik berat | File tunggal portabel, tanpa ketergantungan file lain |
| **Bentuk Aplikasi** | File `respect.exe` + didampingi `blink.dll` | **Satu file `.exe` mandiri** (langsung jalan) |
| **Ukuran Download** | ~3.3 MB (exe) + ~69 MB (dll) | ~57 MB (semua sudah terintegrasi di dalam) |
| **Engine Web** | Chromium 132 (Terbaru & Cepat) | Miniblink 49 (Ringan & Hemat Memori) |
| **Dukungan Windows** | Windows 7, 8, 10, 11 (64-bit) | Windows XP, 7, 8, 10, 11 (32-bit & 64-bit) |
| **Rekomendasi Untuk** | Website modern, dashboard SPA, grafik WebGL, SaaS | Web sederhana, aplikasi kasir (POS), laptop lama/spesifikasi rendah |

> **💡 Rekomendasi:** 
> - Pilih **Respect Modern** jika Anda ingin membuka website masa kini dengan tampilan terbaik.
> - Pilih **Respect Lite** jika Anda ingin file tunggal yang sangat mudah dipindah ke flashdisk atau dijalankan di komputer kantor/kasir dengan Windows lama.

---

## 🚀 Cara Penggunaan (End-User)

### 1. Menggunakan Tampilan Grafis (GUI Builder) — Paling Mudah!

1. Unduh edisi yang Anda inginkan dari halaman [Releases](../../releases).
2. Klik ganda pada **`respect.exe`** (atau **`respect-lite.exe`**).
3. Jendela antarmuka builder akan terbuka:
   - Pilih mode: **URL Website**, **HTML Inline**, atau **File HTML Lokal**.
   - Masukkan alamat situs (misalnya `https://app.anda.com`) atau pilih file HTML lokal Anda.
   - Atur nama file aplikasi (misal `AplikasiSaya.exe`) dan judul jendela yang diinginkan.
   - *(Opsional)* Pilih file icon `.ico` kustom Anda.
4. Klik tombol **Build Standalone EXE**.
5. Selesai! File `.exe` baru Anda langsung siap dijalankan dan dibagikan.

---

### 2. Menggunakan Baris Perintah (CLI) — Untuk Otomasi

Jika Anda terbiasa dengan Command Prompt atau PowerShell, Anda dapat membuat aplikasi secara instan tanpa membuka jendela antarmuka:

```powershell
# Cek versi dan varian engine
.\respect.exe --version

# Buat aplikasi dari URL website
.\respect.exe --build --source "https://google.com" --out "GoogleApp.exe" --title "Google Desktop"

# Buat aplikasi dari file HTML lokal dengan ukuran jendela kustom
.\respect.exe --build --source "C:\proyek\index.html" --mode file --width 1280 --height 800 --out "Dashboard.exe"

# Buat aplikasi dengan menyematkan icon kustom (.ico)
.\respect.exe --build --source "https://my-app.com" --icon "app-icon.ico" --out "MyApp.exe"
```

<details>
<summary>📋 <b>Daftar Parameter Perintah CLI Lengkap</b></summary>

| Parameter | Deskripsi | Default |
|---|---|---|
| `--version`, `-v` | Menampilkan nomor versi dan identitas engine | - |
| `--build` | Mengaktifkan mode pembuatan file executable | - |
| `--mode` | Jenis konten: `url`, `file`, atau `html` | `url` |
| `--source` | Alamat URL, lokasi path file HTML, atau kode HTML inline | *(Wajib)* |
| `--out` | Nama file `.exe` yang akan dihasilkan | `demo.exe` |
| `--title` | Judul yang tampil pada title bar jendela aplikasi | `respect.exe` |
| `--width` | Lebar jendela aplikasi saat pertama kali dibuka (piksel) | `800` |
| `--height` | Tinggi jendela aplikasi saat pertama kali dibuka (piksel) | `600` |
| `--icon` | Lokasi file icon `.ico` (jika kosong, icon default Respect digunakan) | `""` |

</details>

---

## 💻 Area Pengembang (Developer & Source Build)

<details>
<summary>🛠️ <b>Klik di sini untuk melihat Panduan Kompilasi & Arsitektur Kode</b></summary>

### Prasyarat Pengembangan
- Sistem Operasi: **Windows 10/11 (64-bit)**
- **Go 1.25+** terinstal di sistem
- PowerShell 5.1+

---

### Cara Kompilasi dari Source Code

#### Opsi 1: Menggunakan Skrip Otomatis (Direkomendasikan)
Tersedia skrip PowerShell untuk mengompilasi kedua varian dan menyuntikkan icon secara otomatis:

```powershell
# Bangun kedua edisi sekaligus ke folder dist/
.\scripts\build.ps1 -Target all

# Bangun hanya Respect Modern (respect.exe)
.\scripts\build.ps1 -Target modern

# Bangun hanya Respect Lite (respect-lite.exe)
.\scripts\build.ps1 -Target lite
```

#### Opsi 2: Menggunakan Perintah Go Manual

```powershell
# 1. Kompilasi Respect Modern (Chromium 132)
go build -tags v132 -ldflags="-s -w -H windowsgui" -o respect.exe .

# 2. Kompilasi Respect Lite (Miniblink 49)
go build -ldflags="-s -w -H windowsgui" -o respect-lite.exe .
```

---

### Struktur Repository

```
respect/
├── .github/workflows/   # GitHub Actions untuk automated Windows build & release
├── assets/              # Icon resmi (.ico) dan alat injeksi binary (rcedit.exe)
├── internal/
│   ├── builder/         # UI Builder GUI (builder_v132.go & builder_v49.go)
│   │   └── static/      # Frontend Builder (HTML/CSS/JS mandiri, dual-IPC)
│   ├── icon/            # Injeksi icon aplikasi via Windows PE Resource
│   ├── mb132/           # Wrapper Miniblink 132 pure Go (VEH crash handler & sandbox)
│   ├── payload/         # Shared trailer engine format RESPECTv1 (100% kompatibel)
│   ├── runtime/         # Runtime runner (runtime_v132.go & runtime_v49.go)
│   └── version/         # Sentralisasi versioning (version_v132.go & version_v49.go)
├── scripts/             # Skrip automasi build lokal (build.ps1)
└── main.go              # Shared CLI parser & application entry point
```

---

### Rilis Otomatis (GitHub Actions)
Repository ini telah dilengkapi dengan workflow CI/CD di `.github/workflows/release.yml`. Saat tag rilis dibuat (misal `git tag v1.0.0 && git push origin v1.0.0`), GitHub Actions akan secara otomatis:
1. Mengompilasi `respect.exe` dan `respect-lite.exe` pada Windows runner.
2. Menyuntikkan icon aplikasi resmi.
3. Mengarsipkan bundel `.zip` untuk varian Modern dan Lite.
4. Menerbitkan rilis baru pada tab GitHub Releases.

</details>

---

## 📄 Lisensi
Didistribusikan untuk mempermudah distribusi aplikasi web berbasis desktop di platform Windows.
