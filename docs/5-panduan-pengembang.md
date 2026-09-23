# 🛠️ Panduan Pengembang & Kompilasi Source Code

Dokumentasi ini ditujukan bagi para pengembang (*developers*) dan kontributor yang ingin memahami struktur kode, mengompilasi binary Respect langsung dari source code (*clean build*), serta memanfaatkan pipeline automasi CI/CD.

---

## 1. Prasyarat Sistem & Alat (Tooling)

Sebelum melakukan kompilasi, pastikan lingkungan pengembangan Windows Anda telah memenuhi syarat berikut:

- **Sistem Operasi**: Windows 10 atau Windows 11 (64-bit).
- **Go Compiler**: [Go 1.25+](https://go.dev/dl/) terpasang dan terdaftar di `PATH`.
- **PowerShell**: Versi 5.1 atau PowerShell 7 Core.
- **Git**: Terpasang untuk *cloning* repositori.
- **(Opsional) CGO**: Proyek Respect dirancang menggunakan pure Go runtime & syscall loader untuk Chromium 132 (`internal/mb132`), sehingga **tidak memerlukan compiler GCC/MinGW (CGO diaktifkan/dinonaktifkan tetap berfungsi)**.

---

## 2. Mengambil Source Code

```powershell
# Clone repositori Respect
git clone https://github.com/milio48/respect.git
cd respect
```

---

## 3. Kompilasi Otomatis via `scripts/build.ps1`

Cara termudah dan paling konsisten untuk mengompilasi seluruh varian binary adalah menggunakan skrip otomasi PowerShell yang telah disediakan di folder `scripts/`:

```powershell
# Menampilkan bantuan parameter skrip
Get-Help .\scripts\build.ps1 -Detailed
```

### Opsi Target Build

| Perintah PowerShell | Output Executable | Mode Engine | Deskripsi |
| :--- | :--- | :--- | :--- |
| `.\scripts\build.ps1 -Target modern` | `dist/respect/respect.exe` | **Slim (Dev)** | Kompilasi kilat (~2 detik); menyalin `blink.dll` ke samping binary. |
| `.\scripts\build.ps1 -Target modern -Embed` | `dist/respect/respect.exe` | **Standalone Single-File** | Engine Chromium 132 dikompresi ZSTD (level 19) dan ditanamkan ke dalam `.exe`. |
| `.\scripts\build.ps1 -Target lite` | `dist/respect-lite/respect-lite.exe` | **Standalone 64-bit** | Engine Miniblink 49 tertanam (x64). |
| `.\scripts\build.ps1 -Target lite-x86` | `dist/respect-lite/respect-lite-x86.exe` | **Standalone 32-bit** | Engine Miniblink 49 tertanam untuk sistem lawas/32-bit (x86). |
| `.\scripts\build.ps1 -Target all -Embed` | Semua varian di atas | **Full Standalone** | Membangun seluruh varian untuk persiapan rilis resmi. |

### Parameter Tambahan Skrip `build.ps1`

- **`-Embed`**: Menanamkan engine Chromium ke dalam file executable (menggunakan tag Go `embed132`). Wajib untuk distribusi ke pengguna akhir agar berformat single-file.
- **`-Version <string>`**: Menyuntikkan string nomor versi ke `respect-app/internal/version.AppVersion` (contoh: `-Version 1.2.0`). Jika diabaikan, nomor versi dibaca dari git tag aktif atau default `dev`.

---

## 4. Kompilasi Manual Menggunakan Perintah Go

Jika Anda lebih memilih menjalankan perintah `go build` secara langsung dari terminal:

### 🌟 Respect Modern (Chromium 132)

#### A. Mode Single-File Mandiri (Sama seperti Rilis Resmi):
```powershell
go build -tags "v132,embed132" -ldflags="-s -w -H windowsgui -X respect-app/internal/version.AppVersion=1.0.0" -o respect.exe .
```

#### B. Mode Slim (Pengembangan Cepat):
```powershell
# Hasil binary (~3.4 MB) membutuhkan blink.dll di folder yang sama
go build -tags v132 -ldflags="-s -w -H windowsgui" -o respect.exe .
```

### 🪶 Respect Lite (Miniblink 49)

#### A. Arsitektur 64-bit (x64):
```powershell
$env:GOARCH = "amd64"
go build -ldflags="-s -w -H windowsgui -X respect-app/internal/version.AppVersion=1.0.0" -o respect-lite.exe .
```

#### B. Arsitektur 32-bit (x86 untuk Windows XP s.d. 11):
```powershell
$env:GOARCH = "386"
go build -ldflags="-s -w -H windowsgui -X respect-app/internal/version.AppVersion=1.0.0" -o respect-lite-x86.exe .
```

> **Penjelasan Linker Flags (`-ldflags`):**
> - `-s`: Membuang simbol debug (*symbol table*).
> - `-w`: Membuang informasi debugging DWARF (menghemat ~25% ukuran binary).
> - `-H windowsgui`: Mengompilasi sebagai aplikasi GUI Windows (mencegah jendela Command Prompt hitam muncul saat aplikasi dibuka oleh user).
> - `-X`: Menyuntikkan string variabel versi pada saat kompilasi (*compile-time injection*).

---

## 5. Struktur Pohon Direktori Repositori

```
respect/
├── .github/
│   └── workflows/
│       └── release.yml          # Pipeline CI/CD GitHub Actions untuk rilis biner otomatis
├── assets/
│   ├── icon-full.png            # Aset grafis dokumentasi (Modern)
│   ├── icon-lite.png            # Aset grafis dokumentasi (Lite)
│   ├── respect-full.ico         # Icon Windows PE resmi untuk Respect Modern
│   ├── respect-lite.ico         # Icon Windows PE resmi untuk Respect Lite
│   ├── rcedit.exe               # Utilitas stamping resource PE Win32 (embedded)
│   └── workflow-thumbnail.jpg   # Banner alur kerja Respect
├── docs/                        # Dokumentasi Markdown resmi untuk GitHub Pages
│   ├── index.md                 # Halaman utama / daftar isi dokumentasi
│   ├── 1-panduan-penggunaan.md  # Panduan GUI Builder & CLI lengkap
│   ├── 2-perbandingan-edisi.md  # Komparasi Modern vs Lite & 88 Standar Web
│   ├── 3-arsitektur-virtual-host.md # Virtual Host in-memory & enkripsi V2
│   ├── 4-lifecycle.md           # Siklus hidup internal, thread loop & divergence
│   └── 5-panduan-pengembang.md  # Dokumen panduan kompilasi ini
├── internal/
│   ├── builder/                 # Subsystem GUI Builder
│   │   ├── builder_v132.go      # Adapter GUI untuk engine Chromium 132
│   │   ├── builder_v49.go       # Adapter GUI untuk engine Miniblink 49
│   │   └── static/              # Frontend Web Builder mandiri (HTML/CSS/JS)
│   │       ├── index.html       # UI form, dialog folder/file, dual-channel IPC
│   │       └── style.css        # Desain glassmorphism & responsive layout
│   ├── icon/
│   │   └── icon.go              # Ekstraksi dan injeksi icon PE via embedded rcedit
│   ├── mb132/                   # Wrapper murni Go untuk engine Chromium 132
│   │   ├── mb132.go             # Syscall DLL loader, VEH crash handler, hooks
│   │   ├── embed.go             # Direct embed assets.BlinkDLLZst (tag: embed132)
│   │   └── noembed.go           # Stub mode slim tanpa embedding (tag: !embed132)
│   ├── payload/                 # Engine Trailer Binary V1 & V2
│   │   ├── payload.go           # Parser magic header, strip trailer, self-replicating
│   │   ├── crypto.go            # Enkripsi & dekripsi AES-256-GCM + Zstandard
│   │   └── types.go             # Definisi skema konfigurasi aplikasi
│   ├── runtime/                 # Subsystem Runner Aplikasi Mandiri
│   │   ├── runtime_v132.go      # In-Memory Virtual Host & hook Chromium 132
│   │   └── runtime_v49.go       # Runner Miniblink 49
│   └── version/                 # Sentralisasi versi aplikasi & engine
│       ├── version_v132.go      # Metadata versi Respect Modern
│       └── version_v49.go       # Metadata versi Respect Lite
├── scripts/
│   ├── build.ps1                # Skrip utama kompilasi multi-target
│   └── verify_cookies_and_zombie.ps1 # Skrip verifikasi isolasi sandbox & process leak
├── go.mod                       # Definisi modul Go
├── go.sum                       # Checksum ketergantungan modul
├── lifecycle.md                 # Salinan arsitektur teknis inti di root
├── main.go                      # Entry point: CLI parser, trailer check, app routing
└── README.md                    # Ringkasan cepat & panduan pengguna di root
```

---

## 6. Pipeline CI/CD GitHub Actions

Penerbitan rilis baru dikelola sepenuhnya secara otomatis oleh GitHub Actions melalui file `.github/workflows/release.yml`:

1. **Pemicu (Trigger)**:
   Mendorong tag git baru berformat `v*` (contoh: `v1.2.0`):
   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```
2. **Proses Eksekusi di Cloud Runner Windows**:
   - Memasang Go 1.25.x.
   - Menjalankan kompilasi tiga varian binary mandiri (`respect.exe`, `respect-lite.exe`, dan `respect-lite-x86.exe`).
   - Menyuntikkan icon PE resmi dan metadata versi ke header executable.
   - Memvalidasi integritas file.
3. **Penerbitan Aset**:
   Aset biner langsung diunggah ke halaman **GitHub Releases** tanpa membungkusnya dalam file zip, sehingga pengguna dapat langsung mengunduh 1 file `.exe` dan menjalankannya seketika.

---

## 7. Melakukan Pengujian & Verifikasi

Sebelum mengirimkan *Pull Request* atau membuat rilis:

```powershell
# 1. Jalankan Go vet untuk memeriksa validitas kode
go vet ./...

# 2. Uji kompilasi kedua edisi
.\scripts\build.ps1 -Target modern
.\scripts\build.ps1 -Target lite

# 3. Jalankan skrip verifikasi sandbox dan eliminasi zombie process
powershell -ExecutionPolicy Bypass -File .\scripts\verify_cookies_and_zombie.ps1
```

---

[⬅️ Sebelumnya: 4. Lifecycle & Runtime Internal](4-lifecycle.md) | [Kembali ke Daftar Isi 🏠](index.md)
