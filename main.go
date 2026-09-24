package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"

	"respect-app/internal/builder"
	"respect-app/internal/icon"
	"respect-app/internal/payload"
	"respect-app/internal/runtime"
	"respect-app/internal/version"
)

func init() {
	// Hubungkan icon & metadata injector ke payload builder
	payload.IconInjector = icon.InjectIcon
	payload.DefaultIconInjector = icon.InjectDefaultIcon
	payload.MetadataApplier = func(targetExe string, cfg payload.Config) error {
		return icon.ApplyMetadata(targetExe, icon.Metadata{
			ProductName:     cfg.Title,
			FileDescription: cfg.Title + " Desktop Application",
			CompanyName:     cfg.Company,
			LegalCopyright:  cfg.Copyright,
			Version:         cfg.AppVersion,
			IconPath:        cfg.IconPath,
		})
	}
}

// attachConsole menghubungkan stdout/stderr ke terminal pemanggil di Windows (jika ada)
func attachConsole() {
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole := modkernel32.NewProc("AttachConsole")
	// ATTACH_PARENT_PROCESS = -1 (^uintptr(0))
	procAttachConsole.Call(^uintptr(0))
}

func main() {
	// Kunci OS thread utama agar seluruh panggilan GUI Win32 dan Chromium/Miniblink
	// tetap terikat pada satu thread (mencegah zombie process akibat Go thread preemption)
	goruntime.LockOSThread()

	// Jika terdapat argumen CLI (misal --build atau --help), pasang console output
	if len(os.Args) > 1 {
		attachConsole()
	}

	// 1. Parsing CLI flags untuk mode build baris perintah
	versionFlag := flag.Bool("version", false, "Tampilkan informasi versi aplikasi")
	flag.BoolVar(versionFlag, "v", false, "Tampilkan informasi versi aplikasi (shorthand)")
	buildFlag := flag.Bool("build", false, "Bangun file EXE baru dari konfigurasi")
	mode := flag.String("mode", "url", "Mode tampilan: url | html | file | app | server")
	serverFlag := flag.Bool("server", false, "Jalankan via in-memory local HTTP server (127.0.0.1) untuk mengaktifkan Secure Context (WebCrypto & Clipboard)")
	source := flag.String("source", "", "Sumber konten: URL / kode HTML / path file lokal")
	dirFlag := flag.String("dir", "", "Path direktori frontend web untuk dikemas ke dalam EXE (mode virtual host V2)")
	out := flag.String("out", "demo.exe", "Nama output file EXE")
	title := flag.String("title", "respect.exe", "Judul jendela aplikasi")
	width := flag.Int("width", 800, "Lebar jendela aplikasi")
	height := flag.Int("height", 600, "Tinggi jendela aplikasi")
	iconPath := flag.String("icon", "", "Path file icon .ico (opsional)")
	appVer := flag.String("app-version", "1.0.0", "Nomor versi aplikasi (default: 1.0.0)")
	company := flag.String("company", "", "Nama perusahaan atau pengembang aplikasi")
	copyright := flag.String("copyright", "", "Hak cipta aplikasi (contoh: Copyright © 2026 Developer)")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version.String())
		return
	}

	if *buildFlag {
		outName := strings.TrimSpace(*out)
		if !strings.HasSuffix(strings.ToLower(outName), ".exe") {
			outName += ".exe"
		}

		// A. Mode Direktori (V2: In-Memory TAR Virtual Host atau Local Server)
		targetDir := strings.TrimSpace(*dirFlag)
		if targetDir != "" || *mode == "app" || *mode == "dir" || *mode == "server" || *serverFlag {
			if targetDir == "" {
				targetDir = strings.TrimSpace(*source)
			}
			if targetDir == "" {
				fmt.Fprintln(os.Stderr, "error: parameter --dir atau --source direktori wajib diisi untuk mode app/dir/server")
				os.Exit(1)
			}

			files := make(map[string][]byte)
			err := filepath.Walk(targetDir, func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				rel, err := filepath.Rel(targetDir, p)
				if err != nil {
					return err
				}
				data, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				files[rel] = data
				return nil
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "error membaca folder:", err)
				os.Exit(1)
			}
			if len(files) == 0 {
				fmt.Fprintln(os.Stderr, "error: direktori kosong atau tidak ada file valid")
				os.Exit(1)
			}

			appMode := "app"
			if *mode == "server" || *serverFlag {
				appMode = "server"
			}

			cfg := payload.Config{
				Mode:       appMode,
				Source:     "index.html",
				Title:      *title,
				Width:      *width,
				Height:     *height,
				IconPath:   *iconPath,
				OutName:    outName,
				AppVersion: *appVer,
				Company:    *company,
				Copyright:  *copyright,
				ServerMode: *serverFlag || *mode == "server",
			}

			if err := payload.BuildSelfV2(cfg, files); err != nil {
				fmt.Fprintln(os.Stderr, "build v2 error:", err)
				os.Exit(1)
			}

			fmt.Println("OK:", cfg.OutName)
			return
		}

		// B. Mode HTML String (V2 Virtual Host In-Memory)
		if *mode == "html" {
			htmlCode := strings.TrimSpace(*source)
			if htmlCode == "" {
				fmt.Fprintln(os.Stderr, "error: parameter --source kode HTML wajib diisi untuk mode html")
				os.Exit(1)
			}
			if !strings.HasPrefix(strings.ToLower(htmlCode), "<!doctype") && !strings.HasPrefix(strings.ToLower(htmlCode), "<html") {
				htmlCode = fmt.Sprintf("<!DOCTYPE html>\n<html>\n<head>\n  <meta charset=\"utf-8\">\n  <title>%s</title>\n</head>\n<body>\n%s\n</body>\n</html>", *title, htmlCode)
			}
			files := map[string][]byte{
				"index.html": []byte(htmlCode),
			}
			cfg := payload.Config{
				Mode:       "html",
				Source:     "index.html",
				Title:      *title,
				Width:      *width,
				Height:     *height,
				IconPath:   *iconPath,
				OutName:    outName,
				AppVersion: *appVer,
				Company:    *company,
				Copyright:  *copyright,
			}
			if err := payload.BuildSelfV2(cfg, files); err != nil {
				fmt.Fprintln(os.Stderr, "build v2 error:", err)
				os.Exit(1)
			}
			fmt.Println("OK:", cfg.OutName)
			return
		}

		// C. Mode Single File / URL (V1 Legacy)
		if strings.TrimSpace(*source) == "" {
			fmt.Fprintln(os.Stderr, "error: parameter --source atau --dir wajib diisi")
			os.Exit(1)
		}

		cfg := payload.Config{
			Mode:       *mode,
			Source:     *source,
			Title:      *title,
			Width:      *width,
			Height:     *height,
			IconPath:   *iconPath,
			OutName:    outName,
			AppVersion: *appVer,
			Company:    *company,
			Copyright:  *copyright,
		}

		if err := payload.BuildSelf(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "build error:", err)
			os.Exit(1)
		}

		fmt.Println("OK:", cfg.OutName)
		return
	}

	// 2. Cek apakah binary ini sendiri memiliki payload trailer (Runtime Mode V1 atau V2)
	if p, err := payload.ReadPayload(); err == nil {
		if *serverFlag || *mode == "server" {
			p.Config.ServerMode = true
		}
		runtime.Run(p)
		return
	}

	// 3. Jika tidak ada payload trailer, jalankan Builder UI (Builder Mode)
	builder.Run()
}
