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

// Run menampilkan jendela Miniblink 132 sesuai konfigurasi payload.
func Run(cfg *payload.Config) {
	cfg.Defaults()

	view, err := mb132.CreateWebWindow(cfg.Title, cfg.Width, cfg.Height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gagal menginisialisasi jendela runtime: %v\n", err)
		return
	}

	if len(assets.RespectIcon) > 0 {
		_ = view.SetIcon(assets.RespectIcon)
	}

	switch cfg.Mode {
	case "url":
		targetURL := strings.TrimSpace(cfg.Source)
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") && !strings.HasPrefix(targetURL, "file://") {
			targetURL = "https://" + targetURL
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
