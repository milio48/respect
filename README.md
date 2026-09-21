# ⚡ Respect Desktop

Aplikasi desktop Windows yang mengubah halaman Web, HTML lokal, atau kode HTML inline menjadi file executable (`.exe`) mandiri tanpa proses recompile.

Dilengkapi dengan antarmuka GUI Builder interaktif dan antarmuka CLI otomatis.

---

## 🚀 Dua Edisi Resmi

Respect hadir dalam dua edisi yang dirancang untuk kebutuhan berbeda:

| Fitur | 🌟 **Respect Modern (`respect.exe`)** | 🪶 **Respect Lite (`respect-lite.exe`)** |
|---|---|---|
| **Versi Engine** | Chromium 132 (Modern Miniblink Pure Go) | Miniblink 49 (WKE / Chromium 49 Legacy) |
| **Bentuk Distribusi** | Slim binary (~3.5 MB) + `blink.dll` (~69 MB) | **Single-file standalone** (~57 MB, DLL ter-embed) |
| **Dukungan OS** | Windows 7, 10, 11 (64-bit) | Windows XP, 7, 8, 10, 11 (32-bit & 64-bit) |
| **Standar Web** | ES2024, CSS Modern, WebGL, HTTP/2 & 3 | HTML5 dasar, ES6 awal, CSS dasar |
| **Sandboxing** | Dynamic Sandbox (`%LOCALAPPDATA%\respect\apps\<app>\`) | Dynamic Sandbox (`%LOCALAPPDATA%\respect\apps\<app>\`) |
| **Cocok Untuk** | Dashboard SPA modern, grafik interaktif, Web modern | Utilitas internal, mesin kasir, kompatibilitas OS lawas |

---

## 🛠️ Cara Kompilasi (Build)

### Menggunakan Skrip Otomatis (Direkomendasikan)
Gunakan skrip PowerShell terintegrasi:

```powershell
# Membangun kedua varian sekaligus ke folder dist/
.\scripts\build.ps1 -Target all

# Hanya varian Modern (respect.exe)
.\scripts\build.ps1 -Target modern

# Hanya varian Lite (respect-lite.exe)
.\scripts\build.ps1 -Target lite
```

### Menggunakan Go CLI Langsung

#### 1. Respect Modern (v132)
```powershell
go build -tags v132 -ldflags="-s -w -H windowsgui" -o respect.exe .
```
*Catatan: Pastikan `blink.dll` diletakkan di samping `respect.exe`.*

#### 2. Respect Lite (v49)
```powershell
go build -ldflags="-s -w -H windowsgui" -o respect-lite.exe .
```

---

## 📖 Penggunaan

### 1. Mode Antarmuka Grafis (GUI Builder)
Cukup jalankan `respect.exe` atau `respect-lite.exe` tanpa parameter untuk membuka antarmuka GUI Builder:
```powershell
.\respect.exe
```

### 2. Mode Baris Perintah (CLI)

```powershell
# Periksa versi aplikasi dan engine
.\respect.exe --version
.\respect-lite.exe --version

# Bangun aplikasi dari URL website
.\respect.exe --build --source "https://google.com" --out "GoogleApp.exe" --title "Google"

# Bangun aplikasi dari file HTML lokal dengan ukuran jendela kustom
.\respect.exe --build --source "C:\path\ke\index.html" --mode file --width 1280 --height 800 --out "MyApp.exe"

# Bangun aplikasi dengan menyertakan icon kustom (.ico)
.\respect.exe --build --source "https://my-dashboard.com" --icon "icon.ico" --out "Dashboard.exe"
```

### Parameter CLI
- `--version`, `-v` : Menampilkan nomor versi dan detail engine
- `--build` : Mengaktifkan mode pembuatan executable
- `--mode` : Mode konten (`url`, `file`, `html`) — *default: `url`*
- `--source` : Alamat URL, path file HTML lokal, atau kode HTML inline
- `--out` : Nama file output EXE tujuan — *default: `demo.exe`*
- `--title` : Judul jendela aplikasi — *default: `respect.exe`*
- `--width` : Lebar jendela dalam piksel — *default: `800`*
- `--height` : Tinggi jendela dalam piksel — *default: `600`*
- `--icon` : Path file icon `.ico` (jika tidak diisi, icon resmi Respect disuntikkan otomatis)

---

## 🏗️ Struktur Repository

```
respect/
├── .github/workflows/   # Workflow GitHub Actions untuk automated Windows release
├── assets/              # Icon resmi Respect (.ico) dan rcedit.exe
├── internal/
│   ├── builder/         # GUI Builder (dual implementation: builder_v132.go & builder_v49.go)
│   │   └── static/      # Shared HTML/CSS/JS Builder dengan auto-detect engine & dual-IPC
│   ├── icon/            # Injeksi icon default dan kustom via rcedit
│   ├── mb132/           # Wrapper pure Go Miniblink 132 (VEH crash handler, dynamic sandbox)
│   ├── payload/         # Shared RESPECTv1 trailer engine (agnostik engine, 100% kompatibel)
│   ├── runtime/         # Runtime runner (dual implementation: runtime_v132.go & runtime_v49.go)
│   └── version/         # Sistem versioning sentral (version_v132.go & version_v49.go)
├── scripts/             # Skrip automasi build Windows (build.ps1)
└── main.go              # Shared CLI & application entry point
```

---

## 📄 Lisensi
Hak Cipta © 2026. Didistribusikan untuk pengembangan aplikasi desktop ringan berbasis web di lingkungan Windows.
