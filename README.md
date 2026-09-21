# ⚡ Respect Desktop

**Respect Desktop** adalah aplikasi Windows praktis untuk mengubah situs website, web app, atau file HTML menjadi aplikasi desktop (`.exe`) mandiri siap pakai — **hanya dengan beberapa klik, tanpa perlu install server, Node.js, atau coding tambahan!**

---

## 🎯 Pilih Edisi yang Tepat untuk Anda

Respect siap pakai langsung setelah diunduh (tidak perlu proses install):

| Pilihan | 🌟 **Respect Modern (`respect.exe`)** | 🪶 **Respect Lite (`respect-lite.exe`)** |
|---|---|---|
| **Kelebihan** | Mendukung web modern, tampilan mulus & animasi kaya | **File tunggal mandiri**, langsung jalan tanpa file lain |
| **Isi File** | File `respect.exe` didampingi file `blink.dll` | **Hanya 1 file `.exe`** (sangat praktis) |
| **Engine Browser** | Chromium 132 (Terbaru & Cepat) | Miniblink 49 (Sangat Ringan & Hemat RAM) |
| **Kompatibilitas** | Windows 7, 8, 10, 11 (64-bit) | Windows XP, 7, 8, 10, 11 (32-bit & 64-bit) |
| **Paling Cocok Untuk** | Dashboard modern, aplikasi SaaS, grafik interaktif | Aplikasi kasir (POS), laptop lawas, utilitas kantor |

> **💡 Panduan Cepat:**
> - Jika ingin tampilan web terbaik dan modern: **Unduh Respect Modern**.
> - Jika ingin file tunggal yang ringkas untuk dibawa di flashdisk atau komputer lama: **Unduh Respect Lite**.

---

## 🚀 Cara Menggunakan (Sangat Mudah!)

1. **Unduh aplikasi** dari halaman [GitHub Releases](../../releases).
2. **Buka aplikasi**:
   - Untuk Respect Modern: Ekstrak zip, lalu klik dua kali **`respect.exe`**.
   - Untuk Respect Lite: Cukup klik dua kali **`respect-lite.exe`**.
3. **Isi formulir pembuatan**:
   - Masukkan alamat web (contoh: `https://aplikasisaya.com`) atau pilih file HTML lokal Anda.
   - Beri nama aplikasi Anda (contoh: `AplikasiSaya.exe`).
   - *(Opsional)* Pilih gambar icon `.ico` Anda sendiri.
4. Klik tombol hijau **Build Standalone EXE**.
5. **Selesai!** File `.exe` buatan Anda langsung jadi di folder yang sama dan siap digunakan atau dibagikan ke siapa saja.

---

## 💡 Fitur Lanjutan (Opsional)

<details>
<summary>💻 <b>Klik di sini jika ingin menggunakan Baris Perintah (CLI) untuk Otomasi</b></summary>

Bagi Anda yang ingin membuat file `.exe` secara otomatis melalui script Command Prompt atau PowerShell:

```powershell
# Cek versi aplikasi
.\respect.exe --version

# Buat aplikasi langsung dari URL website
.\respect.exe --build --source "https://google.com" --out "GoogleDesktop.exe" --title "Google"

# Buat aplikasi dari file HTML lokal dengan ukuran jendela tertentu
.\respect.exe --build --source "C:\proyek\index.html" --mode file --width 1280 --height 800 --out "Dashboard.exe"

# Buat aplikasi dengan icon kustom (.ico)
.\respect.exe --build --source "https://my-app.com" --icon "icon.ico" --out "MyApp.exe"
```

### Parameter CLI
- `--build` : Mengaktifkan pembuatan file executable dari terminal
- `--source` : URL website atau path file HTML
- `--out` : Nama file output (default: `demo.exe`)
- `--title` : Judul jendela aplikasi
- `--width` / `--height` : Ukuran jendela awal aplikasi
- `--icon` : Path file icon `.ico` (opsional)
- `--version` : Cek versi engine dan edisi yang aktif

</details>

---

## 🛠️ Area Pengembang (Developer & Source Code)

<details>
<summary>🔧 <b>Klik di sini untuk Panduan Kompilasi dari Source Code & Kontribusi</b></summary>

### Prasyarat
- Sistem Operasi: **Windows 10/11 (64-bit)**
- **Go 1.25+**
- PowerShell 5.1+

---

### Cara Kompilasi (Build) Lokal

#### 1. Menggunakan Skrip PowerShell Otomatis
```powershell
# Bangun kedua edisi sekaligus ke folder dist/
.\scripts\build.ps1 -Target all

# Bangun hanya Respect Modern (respect.exe + blink.dll)
.\scripts\build.ps1 -Target modern

# Bangun hanya Respect Lite (respect-lite.exe single file)
.\scripts\build.ps1 -Target lite
```

#### 2. Menggunakan Perintah Go Manual
```powershell
# Respect Modern (Chromium 132)
go build -tags v132 -ldflags="-s -w -H windowsgui" -o respect.exe .

# Respect Lite (Miniblink 49)
go build -ldflags="-s -w -H windowsgui" -o respect-lite.exe .
```

---

### Struktur Repository

```
respect/
├── .github/workflows/   # CI/CD GitHub Actions untuk rilis Windows otomatis
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

### Otomasi Rilis GitHub Actions
Workflow rilis tersedia di `.github/workflows/release.yml`. Ketika tag versi dibuat (misal `git tag v1.0.0 && git push origin v1.0.0`), GitHub Actions akan:
1. Mengompilasi `respect.exe` dan `respect-lite.exe` di mesin Windows runner.
2. Menyuntikkan icon aplikasi resmi Respect.
3. Mengarsipkan bundel `.zip` untuk rilis.
4. Menerbitkan aset secara otomatis ke GitHub Releases.

</details>

---

## 📄 Lisensi
Didistribusikan untuk mempermudah distribusi aplikasi web berbasis desktop di platform Windows.
