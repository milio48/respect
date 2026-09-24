//go:build v132

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"respect-app/assets"
	"respect-app/internal/localserver"
	"respect-app/internal/mb132"
	"respect-app/internal/payload"
)

// Run menampilkan jendela Chromium 132 sesuai konfigurasi payload (V1 atau V2).
func Run(p *payload.Payload) {
	goruntime.LockOSThread()
	cfg := &p.Config
	cfg.Defaults()

	view, err := mb132.CreateWebWindow(cfg.Title, cfg.Width, cfg.Height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gagal menginisialisasi jendela runtime: %v\n", err)
		return
	}

	if len(assets.RespectIcon) > 0 {
		_ = view.SetIcon(assets.RespectIcon)
	}

	// 1. Jika payload Versi 2 (In-Memory TAR Virtual Host / In-Memory Local Server)
	if p.Version == 2 && len(p.Files) > 0 {
		entryFile := "index.html"
		if cfg.Source != "" && !strings.HasPrefix(cfg.Source, "http") {
			entryFile = strings.TrimPrefix(cfg.Source, "/")
		}

		// Mode Server: Buka HTTP server in-memory pada 127.0.0.1 untuk mengaktifkan Secure Context (WebCrypto, Clipboard)
		if cfg.ServerMode || cfg.Mode == "server" {
			srv, err := localserver.Start(p.Files)
			if err == nil {
				view.OnDestroy(func() {
					srv.Close()
				})
				serverURL := fmt.Sprintf("http://127.0.0.1:%d/%s", srv.Port, entryFile)
				view.LoadURL(serverURL)
				view.Show()
				mb132.RunMessageLoop()
				return
			}
			// Fallback ke Virtual Host jika start server gagal
		}

		// Mode Default: Virtual Host in-memory (http://app/) - 0 port, 0 disk write
		view.RegisterVirtualHost(p.Files)
		entryURL := "http://app/" + entryFile
		view.LoadURL(entryURL)
		view.Show()
		mb132.RunMessageLoop()
		return
	}

	// 2. Jika payload Versi 1 (JSON tunggal legacy)
	switch cfg.Mode {
	case "url":
		targetURL := strings.TrimSpace(cfg.Source)
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") && !strings.HasPrefix(targetURL, "file://") {
			targetURL = "https://" + targetURL
		}
		// Jika target adalah Google dan belum memiliki parameter bahasa hl=, tambahkan hl=id
		if strings.Contains(targetURL, "google.com") && !strings.Contains(targetURL, "hl=") {
			if strings.Contains(targetURL, "?") {
				targetURL += "&hl=id"
			} else {
				targetURL += "/?hl=id"
			}
		}
		view.LoadURL(targetURL)

	case "file":
		absPath, err := filepath.Abs(cfg.Source)
		if err == nil {
			fileURL := "file:///" + filepath.ToSlash(absPath)
			view.LoadURL(fileURL)
		} else {
			view.LoadHTML(fmt.Sprintf("<h2>Gagal memuat file: %v</h2>", err), "http://localhost/")
		}

	case "html":
		view.LoadHTML(cfg.Source, "http://localhost/")

	default:
		view.LoadHTML(fmt.Sprintf("<h2>Mode tidak dikenal: %s</h2>", cfg.Mode), "http://localhost/")
	}

	view.Show()

	mb132.RunMessageLoop()
}
