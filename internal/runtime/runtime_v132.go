//go:build v132

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"respect-app/assets"
	"respect-app/internal/mb132"
	"respect-app/internal/payload"
)

// Run menampilkan jendela Chromium 132 sesuai konfigurasi payload (V1 atau V2).
func Run(p *payload.Payload) {
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

	// 1. Jika payload Versi 2 (In-Memory TAR Virtual Host)
	if p.Version == 2 && len(p.Files) > 0 {
		view.RegisterVirtualHost(p.Files)
		entryURL := "http://app/index.html"
		if cfg.Source != "" && !strings.HasPrefix(cfg.Source, "http") {
			entryURL = "http://app/" + strings.TrimPrefix(cfg.Source, "/")
		}
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
