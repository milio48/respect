package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"respect-app/internal/builder"
	"respect-app/internal/icon"
	"respect-app/internal/payload"
	"respect-app/internal/runtime"
)

func init() {
	// Hubungkan icon injector ke payload builder
	payload.IconInjector = icon.InjectIcon
}

// attachConsole menghubungkan stdout/stderr ke terminal pemanggil di Windows (jika ada)
func attachConsole() {
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole := modkernel32.NewProc("AttachConsole")
	// ATTACH_PARENT_PROCESS = -1 (^uintptr(0))
	procAttachConsole.Call(^uintptr(0))
}

func main() {
	// Jika terdapat argumen CLI (misal --build atau --help), pasang console output
	if len(os.Args) > 1 {
		attachConsole()
	}

	// 1. Parsing CLI flags untuk mode build baris perintah
	buildFlag := flag.Bool("build", false, "Bangun file EXE baru dari konfigurasi")
	mode := flag.String("mode", "url", "Mode tampilan: url | html | file")
	source := flag.String("source", "", "Sumber konten: URL / kode HTML / path file lokal")
	out := flag.String("out", "demo.exe", "Nama output file EXE")
	title := flag.String("title", "respect.exe", "Judul jendela aplikasi")
	width := flag.Int("width", 800, "Lebar jendela aplikasi")
	height := flag.Int("height", 600, "Tinggi jendela aplikasi")
	iconPath := flag.String("icon", "", "Path file icon .ico (opsional)")
	flag.Parse()

	if *buildFlag {
		if strings.TrimSpace(*source) == "" {
			fmt.Fprintln(os.Stderr, "error: parameter --source wajib diisi")
			os.Exit(1)
		}

		outName := strings.TrimSpace(*out)
		if !strings.HasSuffix(strings.ToLower(outName), ".exe") {
			outName += ".exe"
		}

		cfg := payload.Config{
			Mode:     *mode,
			Source:   *source,
			Title:    *title,
			Width:    *width,
			Height:   *height,
			IconPath: *iconPath,
			OutName:  outName,
		}

		if err := payload.BuildSelf(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "build error:", err)
			os.Exit(1)
		}

		fmt.Println("OK:", cfg.OutName)
		return
	}

	// 2. Cek apakah binary ini sendiri memiliki payload trailer (Runtime Mode)
	if cfg, err := payload.ReadPayload(); err == nil {
		runtime.Run(cfg)
		return
	}

	// 3. Jika tidak ada payload trailer, jalankan Builder UI (Builder Mode)
	builder.Run()
}
