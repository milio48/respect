# Lifecycle & Arsitektur Respect Desktop

Dokumentasi teknis siklus hidup eksekusi (*runtime lifecycle*), pemisahan alur internal antara **`respect.exe` (Chromium 132 Modern)** dan **`respect-lite.exe` (Miniblink 49 Lite)**, serta mekanisme kompilasi mandiri (*self-replicating binary*).

---

## 1. Titik Percabangan Arsitektur (The Divergence Point)

Meskipun `respect.exe` dan `respect-lite.exe` berbagi fungsi utama yang sama (sebagai Builder GUI dan Runner aplikasi mandiri), **alur kerja keduanya bercabang total di tingkat kompilasi dan runtime engine**:

```
                              [ main.go Entry Point ]
                                         |
                       +-----------------+-----------------+
                       | 1. attachConsole() (bila ada CLI) |
                       | 2. Parsing Flag CLI (--build, -v) |
                       | 3. payload.ReadPayload()          |
                       +-----------------+-----------------+
                                         |
                +------------------------+------------------------+
                |                                                 |
         (Ada Trailer)                                    (Tanpa Trailer)
                |                                                 |
                v                                                 v
        [ Mode Runner ]                                   [ Mode Builder ]
                |                                                 |
    +-----------+-----------+                         +-----------+-----------+
    |                       |                         |                       |
    v                       v                         v                       v
[//go:build v132]   [//go:build !v132]        [//go:build v132]   [//go:build !v132]
  respect.exe        respect-lite.exe           respect.exe        respect-lite.exe
 (Modern v132)          (Lite v49)             (Modern v132)          (Lite v49)
```

### Mekanisme Pemisahan:
1. **Entry Point Bersama (`main.go`)**:
   Hanya menangani parsing CLI flags (`--build`, `--version`, dll.) dan pemeriksaan *payload trailer* di ujung file executable.
2. **Pemisahan via Go Build Tags**:
   - `respect.exe` dikompilasi dengan tag `v132` (atau `v132,embed132`). Mengaktifkan:
     - `internal/mb132/` (Custom pure Go DLL loader & Chromium binding)
     - `internal/builder/builder_v132.go`
     - `internal/runtime/runtime_v132.go`
   - `respect-lite.exe` dikompilasi dengan tag default `!v132`. Mengaktifkan:
     - `github.com/epkgs/blink` (Wrapper Miniblink 49 pihak ketiga)
     - `internal/builder/builder_v49.go`
     - `internal/runtime/runtime_v49.go`

---

## 2. Matriks Perbandingan Komprehensif

| Dimensi Arsitektur | 🌟 Respect Modern (`respect.exe`) | 🪶 Respect Lite (`respect-lite.exe`) |
| :--- | :--- | :--- |
| **Engine Inti** | **Chromium 132** (Blink modern 2025/2026) | **Miniblink 49** (Fork WebKit ringan) |
| **Target Arsitektur CPU** | **64-bit (x64)** | **64-bit (x64)** & **32-bit (x86)** |
| **Dukungan OS Windows** | Windows 7 SP1, 8, 10, 11 (64-bit) | Windows XP SP3 s.d. 11 (32-bit & 64-bit) |
| **Packaging Engine DLL** | Terkompresi **Zstandard (ZSTD)** level 19 (`blink.dll.zst`, ~23.36 MB) | Terkompresi **Zstandard (ZSTD)** level 19 (`miniblink_49_x64.dll.zst`, ~14.23 MB / `miniblink_49_x86.dll.zst`, ~12.05 MB) |
| **Ukuran Binary Standalone** | **~31.56 MB** | **~32.45 MB** (x64) / **~30.06 MB** (x86) |
| **Mekanisme Pemuatan DLL** | Lazy extract ke `%LocalAppData%\respect_desktop\engine\` dengan validasi ukuran byte (Start instan **0 ms** saat cache ada) | Lazy extract terisolasi ke `%LocalAppData%\respect_desktop\engine_v49\` via `internal/mb49` (Start instan **0 ms** saat cache ada) |
| **Penanganan Crash Engine** | **Windows Vectored Exception Handler (VEH)** mencegat `STATUS_BREAKPOINT` (0x80000003) | Runtime bawaan WebKit |
| **Protokol IPC Frontend** | Native Chromium `window.mbQuery(1, payload, cb)` | Miniblink Channel `window.ipc.invoke(channel, payload)` |
| **Penanganan Popup / New Tab** | Intersepsi native via `procMbOnCreateView` (redirect ke jendela aktif) | Miniblink popup window default |
| **Dialog Interaktif (JS)** | Dialihkan ke Win32 `MessageBoxW` native (`alert`, `confirm`, `prompt`) | Dialog bawaan engine browser |
| **Injeksi Jaringan (HTTP)** | Intersepsi `procMbOnLoadUrlBegin` + `procMbNetSetHTTPHeaderFieldUtf8` | Spoofing User-Agent global via `wkeSetUserAgent` |
| **Event / Message Loop** | Chromium RunLoop (`procMbRunMessageLoop`) | Win32 Message Loop (`win.GetMessage` & `DispatchMessage`) |
| **Pembersihan Artifacts** | `cleanLocalArtifacts()` & background detached `cmd del cookies.dat` | `os.Exit(0)` standar Win32 |
| **Pemuatan HTML Lokal** | Direct in-memory load via `procMbLoadHtmlWithBaseUrl` | Penulisan file sementara di `%TEMP%\respect_app_<hash>.html` |
| **Kompatibilitas Web** | ES2023+, CSS Grid, Flexbox, Tailwind CSS, WebGL, WASM | ES6 Dasar, CSS3 Ringan (Tidak mendukung CSS Grid & CSS Variables) |
| **Konsumsi Memori (RAM)** | ~80–120 MB | **~25–40 MB** (Sangat hemat resource) |

---

## 3. Alur Eksekusi Berdampingan (Side-by-Side Lifecycle Pipeline)

Diagram di bawah ini memperlihatkan perbedaan mendasar di setiap tahapan eksekusi:

```
TAHAP                   RESPECT MODERN (respect.exe)             RESPECT LITE (respect-lite.exe)
─────                   ────────────────────────────             ───────────────────────────────

1. STARTUP              os.Executable()                          os.Executable()
                        attachConsole() jika argumen CLI         attachConsole() jika argumen CLI
                        payload.ReadPayload()                    payload.ReadPayload()

2. DETEKSI MODE         Ada Trailer -> Mode Runner               Ada Trailer -> Mode Runner
                        Tanpa Trailer -> Mode Builder            Tanpa Trailer -> Mode Builder

3. ENGINE BOOT          mb132.Init() (Lazy on-demand)            blink.NewApp()
                        ├── 1. Cek local .\blink.dll             ├── Ekstrak raw DLL ke %TEMP%
                        ├── 2. Cek cache %LocalAppData%\...      ├── LoadLibraryW di background
                        │      (Validasi 68.962.816 bytes)       └── win.SetProcessDPIAware()
                        └── 3. Decompress ZSTD (~0.15 dtk)
                        syscall.LoadDLL("blink.dll")
                        installVehHandler() (VEH 0x80000003)
                        procMbInit.Call(0)

4. WINDOWING &          procMbCreateWebWindow(POPUP)             app.CreateWebWindowPopup()
   HOOK SUBSYSTEM       ├── SetUserAgent(Chrome 132)             ├── app.AddBootScript(lang: id-ID)
                        ├── SetLanguage(id-ID, en-US)            ├── wkeSetUserAgent(Chrome 120)
                        ├── SetIsolatedStorage(apps/<name>)      ├── wkeSetCookieJarFullPath(...)
                        ├── Hook onCreateView (popup redirect)   └── wkeSetLocalStorageFullPath(...)
                        ├── Hook JS alert/confirm/prompt
                        ├── Hook TitleChanged (SetWindowTextW)
                        ├── Hook Download (PopupDownloadMgr)
                        ├── Hook OnLoadUrlBegin (HTTP Headers)
                        └── Hook OnDidCreateScriptContext
                            (Injeksi Smooth Wheel & Lang)

5. IPC REGISTRATION     Mode Builder:                            Mode Builder:
                        view.HandleQuery(func(json) -> resp)     app.IPC.Handle("build-app", ...)
                        Backend memproses window.mbQuery         app.IPC.Handle("pick-icon", ...)

6. CONTENT RENDERING    Mode Runner:                             Mode Runner:
                        - URL  : view.LoadURL(url)               - URL  : view.LoadURL(url)
                        - File : view.LoadURL("file:///...")     - File : view.LoadURL("file:///...")
                        - HTML : view.LoadHTML(html, base)       - HTML : Simpan ke %TEMP%\*.html
                        Mode Builder:                                     lalu load file:/// (bypas limit)
                        view.LoadHTML(indexHTML, base)           Mode Builder:
                                                                 app.Resource.Bind("builder", static)
                                                                 view.LoadURL("http://builder/index.html")

7. RUNTIME LOOP         procMbRunMessageLoop.Call()              app.KeepRunning()
                        (Chromium Native Event Loop)             (Win32 GetMessage / DispatchMessage)

8. SHUTDOWN &           procMbExitMessageLoop.Call()             view.OnDestroy(func() { os.Exit(0) })
   CLEANUP              cleanLocalArtifacts()
                        spawnDetachedCleanup() (del cookies.dat)
```

---

## 4. Rincian Fase Eksekusi & Perbedaan Teknis

### Fase 1: Startup & Trailer Parsing (Sama di Kedua Varian)
1. Binary membaca dirinya sendiri via `os.Executable()`.
2. Binary membaca byte paling belakang file (16 byte terakhir) untuk memeriksa magic header:
   - **`RESPECT_PAYLOAD_V2` (Format Baru: In-Memory TAR + Zstandard + AES-256-GCM)**:
     - Membaca sample PE header (4KB) dan base file size untuk menurunkan kunci AES-256 via HMAC-SHA256 (Dynamic Key Derivation).
     - Otentikasi dan dekripsi ciphertext AES-256-GCM langsung di RAM (Zero-Leak).
     - Streaming dekompresi ZSTD langsung ke memori.
     - Unpack TAR in-memory ke `map[string][]byte`.
     - Konfigurasi `respect.json` diekstrak untuk inisialisasi window.
     - Masuk ke **Mode Runner In-Memory Virtual Host (`http://app/`)** (0 bytes ke disk, 0 port jaringan).
   - **`RESPECT_PAYLOAD_V1` (Format Legacy: JSON Tunggal)**:
     - Unmarshal JSON config langsung dan masuk ke **Mode Runner Legacy**.
   - **Tanpa Trailer**:
     - Binary bertindak sebagai executable master **Builder**.
     - Memeriksa flag CLI `--build` (mode headless/terminal) atau tanpa argumen (buka GUI Builder via `builder.Run()`).

---

### Fase 2: Engine Bootstrapping & Memory Loading (Sangat Berbeda)

#### 🌟 Respect Modern (`internal/mb132/mb132.go`):
- Menggunakan strategi **3-Tier Engine Discovery**:
  1. *Local Directory*: Memeriksa apakah `./blink.dll` atau `./mb132_x64.dll` ada di folder lokal (digunakan pada mode pengembangan lokal/slim agar build cepat tanpa embedding).
  2. *System Engine Cache*: Memeriksa `%LocalAppData%\respect_desktop\engine\blink.dll`. Jika file ada dan ukurannya persis **68.962.816 byte**, engine langsung di-load via `syscall.LoadDLL`. Waktu inisialisasi: **0 ms**.
  3. *ZSTD Decompression*: Jika file cache belum ada (first run), mendekompresi `assets.BlinkDLLZst` (23.36 MB) menggunakan decoder streaming `github.com/klauspost/compress/zstd` ke folder cache sistem. Waktu ekstraksi: **~0.15 detik**.
- Memasang **Windows Vectored Exception Handler (VEH)** via kernel32 `AddVectoredExceptionHandler`. Menangkap dan memotong kode `STATUS_BREAKPOINT` (0x80000003 / `int 3`) yang sering dipicu oleh assertion `DCHECK` Chromium saat merender halaman web modern yang kompleks, sehingga window tidak pernah crash atau force close secara misterius.

#### 🪶 Respect Lite (`internal/mb49/mb49.go` & `github.com/epkgs/blink`):
- Pemuatan engine dikelola terpusat oleh package `internal/mb49`:
  1. *System Engine Cache*: Memeriksa `%LocalAppData%\respect_desktop\engine_v49\miniblink_4975_<arch>\miniblink_49.dll`. Jika file sudah ada dan ukurannya sesuai (42.510.848 byte untuk x64, 35.473.920 byte untuk x86), file langsung dimuat via `blink.WithDllFile(targetDLL)`. Waktu inisialisasi: **0 ms**.
  2. *ZSTD Decompression*: Jika file cache belum ada (first run), mendekompresi `assets.Miniblink49DLLZst` (14.23 MB x64 / 12.05 MB x86) ke folder cache sistem secara atomik. Waktu ekstraksi: **~0.05 detik**.
- Menggunakan build tag `slim` bawaan `epkgs/blink` sehingga compiler Go tidak lagi menanamkan DLL mentah berukuran 42.5 MB ke dalam executable.
- Memanggil `win.SetProcessDPIAware()` agar rendering font dan elemen antarmuka tidak buram pada layar High DPI.

---

### Fase 3: Windowing, Subsystem Hooking & Keamanan

#### 🌟 Respect Modern:
Chromium 132 mengintegrasikan 7 hook native tingkat rendah di `internal/mb132/mb132.go`:
1. **Popup & Tab Redirect (`procMbOnCreateView`)**:
   Halaman web modern sering membuka jendela otentikasi (Google OAuth, Facebook Login) atau tautan luar dengan `target="_blank"` atau `window.open`. Callback ini menangkap URL target dan memuatnya langsung di jendela aktif, mencegah hilangnya sesi login akibat pop-up yang terblokir.
2. **Native Win32 Dialogs (`mbOnAlertBox`, `mbOnConfirmBox`, `mbOnPromptBox`)**:
   Memotong panggilan JavaScript bawaan dan menggantikannya dengan kotak dialog native Windows `MessageBoxW` (dengan ikon `MB_ICONINFORMATION` atau `MB_ICONQUESTION`).
3. **Dynamic Title Sync (`procMbOnTitleChanged`)**:
   Setiap kali dokumen web mengubah tag `<title>`, callback native memperbarui teks titlebar jendela Win32 melalui `SetWindowTextW`.
4. **Download Manager Hook (`procMbOnDownload`)**:
   Memanggil `procMbPopupDownloadMgr` untuk memunculkan dialog penyimpanan berkas (*Save As*) bawaan Windows secara otomatis ketika user mengklik file unduhan.
5. **Network Header Injection (`procMbOnLoadUrlBegin` & `procMbNetSetHTTPHeaderFieldUtf8`)**:
   Menyuntikkan header HTTP `Accept-Language: id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7` dan `User-Agent: Chrome/132.0.0.0` ke seluruh request jaringan sebelum keluar ke internet. Menjamin Google tidak pernah mengembalikan hasil pencarian berbahasa Cina.
6. **V8 Context Script Injection (`procMbOnDidCreateScriptContext`)**:
   Menyuntikkan skrip JS mikro saat engine V8 dibuat untuk menerapkan akselerasi *smooth mouse wheel scrolling* yang natural dan menstandarkan properti `navigator.language`.
7. **Isolasi Sandbox (`procMbSetCookieJarFullPath` & `procMbSetLocalStorageFullPath`)**:
   Mengarahkan seluruh cookie dan LocalStorage ke `%LocalAppData%\respect_desktop\apps\<appName>\`.

#### 🪶 Respect Lite:
- Menggunakan API bawaan Miniblink 49 via `app.CallFunc`:
  1. `wkeSetUserAgent`: Menyetel string User-Agent Chrome 120 agar situs web tidak menolak engine WebKit lama.
  2. `wkeSetCookieJarFullPath` & `wkeSetLocalStorageFullPath`: Menyetel jalur file cookie dan storage ke folder sandbox per-aplikasi.
  3. `app.AddBootScript`: Menyuntikkan script override `navigator.language` ke context global browser.

---

### Fase 4: Protokol IPC (Inter-Process Communication)

Frontend builder (`static/index.html`) menggunakan arsitektur adaptif (*dual-channel abstraction*) untuk mendeteksi varian engine secara otomatis:

```javascript
function invokeBuild(cfg) {
  var payload = JSON.stringify(cfg);

  // 1. Jalur Respect Modern (v132)
  if (typeof window.mbQuery === 'function') {
    return new Promise(function (resolve) {
      window.mbQuery(1, payload, function (customMsg, response) {
        resolve(response || customMsg);
      });
    });
  }

  // 2. Jalur Respect Lite (v49)
  if (typeof window.ipc !== 'undefined' && window.ipc && typeof window.ipc.invoke === 'function') {
    return window.ipc.invoke('build-app', payload);
  }
}
```

- **Modern IPC Flow**:
  `window.mbQuery` $\to$ C DLL Callback `procMbOnJsQuery` $\to$ Go Handler `view.HandleQuery` $\to$ Eksekusi `payload.BuildSelf` $\to$ Balasan dikirim via `procMbResponseQuery.Call(handle, queryId, 0, resultPtr)`.
- **Lite IPC Flow**:
  `window.ipc.invoke("build-app")` $\to$ Message dispatching Win32 internal `epkgs/blink` $\to$ Go Handler `app.IPC.Handle("build-app")` $\to$ Eksekusi `payload.BuildSelf` $\to$ Return JSON string langsung ke promise JS.

---

### Fase 5: Penanganan Konten & Dynamic Sandbox

Struktur direktori runtime pengguna diisolasi penuh:

```
%LocalAppData%\respect_desktop\
├── engine\
│   └── blink.dll                 <-- Cache engine Chromium 132 (dibagi bersama)
├── engine_v49\
│   └── miniblink_4975_x64\
│       └── miniblink_49.dll      <-- Cache engine Miniblink 49 (ZSTD dekompresi)
└── apps\
    ├── kasir-toko\               <-- Sandbox Aplikasi A (berdasarkan nama EXE)
    │   ├── cookies.dat
    │   └── storage\
    └── dashboard-admin\          <-- Sandbox Aplikasi B
        ├── cookies.dat
        └── storage\
```

#### Perbedaan Pemuatan Konten HTML:
- **Respect Modern**:
  Mampu memuat string HTML langsung ke engine Chromium via `procMbLoadHtmlWithBaseUrl`. Mendukung CSS Variables, modern flexbox, CSS Grid, ES2023, Canvas, dan WebGL tanpa kendala ukuran dokumen.
- **Respect Lite**:
  Miniblink 49 memiliki batasan panjang URL data URI (`data:text/html,...`). Oleh karena itu, pada mode HTML, Respect Lite menuliskan string HTML ke file sementara `%TEMP%\respect_app_<hash>.html`, lalu membukanya via URL lokal `file:///`.

---

### Fase 6: Event Loop & Lifecycle Teardown

- **Respect Modern**:
  - Digerakkan oleh Chromium Message Loop (`procMbRunMessageLoop.Call()`).
  - Saat jendela ditutup, memanggil `procMbExitMessageLoop.Call()`.
  - Menjalankan fungsi `cleanLocalArtifacts()` untuk menghapus `cookies.dat` yang mungkin sempat tertulis di folder kerja.
  - Memanggil `spawnDetachedCleanup()`: menjalankan proses background tersembunyi `cmd.exe /C ping 127.0.0.1 -n 1 >nul & del /f /q cookies.dat` untuk menghapus file sisa libcurl yang masih terkunci selama beberapa milidetik setelah proses Go ditutup.
- **Respect Lite**:
  - Digerakkan oleh Win32 Message Loop standar di `app.KeepRunning()` (`win.GetMessage`, `win.TranslateMessage`, `win.DispatchMessage`).
  - Menutup proses secara langsung saat jendela utama dihancurkan: `view.OnDestroy(func() { os.Exit(0) })`.

---

## 5. Mekanisme Replikasi Diri (`BuildSelf`)

Ketika user membangun aplikasi mandiri (baik via GUI Builder maupun CLI `--build`), **tidak diperlukan compiler Go eksternal**. Binary yang sedang berjalan mengkloning dirinya sendiri:

```
[ respect.exe / respect-lite.exe yang aktif ]
                    │
                    ▼
1. Baca binary diri sendiri: os.Executable()
                    │
                    ▼
2. Potong trailer lama jika ada: payload.StripTrailer()
   (Mencegah penumpukan metadata berulang)
                    │
                    ▼
3. Serialize Config baru -> JSON bytes
                    │
                    ▼
4. Injeksi Trailer ke akhir file:
   [Raw Executable Bytes] + [JSON Config] + [Header: RESPECT_PAYLOAD_V1 + Offset 8-byte]
                    │
                    ▼
5. Stamping Icon & Metadata PE (Opsional via rcedit.exe embedded):
   - Ekstrak rcedit.exe ke temporary directory
   - Ganti RT_ICON dan string PE (Product Name, Company, Version)
   - Hapus temporary rcedit.exe
                    │
                    ▼
[ Output: EXE Mandiri Baru Selesai ]
```

*Prinsip Turunan (Inheritance):*
- Jika dibuild dari `respect.exe` $\to$ menghasilkan executable Chromium 132 (~27.16 MB).
- Jika dibuild dari `respect-lite.exe` $\to$ menghasilkan executable Miniblink 49 (~57.73 MB).
- Jika dibuild dari `respect-lite-x86.exe` $\to$ menghasilkan executable 32-bit untuk Windows XP s.d. 11 (~50.78 MB).

---

## 6. Pipeline Kompilasi Master (`scripts/build.ps1`)

Seluruh varian binary dikompilasi melalui satu titik komando `scripts/build.ps1`:

| Target Komando | Output File | Mode Engine | Ukuran Standar | Keterangan & Tujuan |
| :--- | :--- | :--- | :--- | :--- |
| `.\scripts\build.ps1 -Target modern` | `dist/respect/respect.exe` | Slim (No Embed) | **~3.4 MB** | Mode dev cepat; membutuhkan `blink.dll` di folder yang sama. |
| `.\scripts\build.ps1 -Target modern -Embed` | `dist/respect/respect.exe` | Standalone ZSTD | **~27.2 MB** | Single-file mandiri dengan engine Chromium 132 terkompresi. |
| `.\scripts\build.ps1 -Target lite` | `dist/respect-lite/respect-lite.exe` | Standalone 64-bit | **~57.7 MB** | Single-file mandiri engine Miniblink 49 (x64). |
| `.\scripts\build.ps1 -Target lite-x86` | `dist/respect-lite/respect-lite-x86.exe` | Standalone 32-bit | **~50.8 MB** | Single-file mandiri engine Miniblink 49 (x86) untuk PC lawas / XP. |
| `.\scripts\build.ps1 -Target all -Embed` | Ketiga file di atas | Standalone Release | Sesuai di atas | Dipicu otomatis oleh GitHub Actions Workflow saat release tag git `v*`. |

---

[⬅️ Sebelumnya: 3. Arsitektur Virtual Host](3-arsitektur-virtual-host.md) | [Daftar Isi](index.md) | [Selanjutnya: 5. Panduan Pengembang ➡️](5-panduan-pengembang.md)
