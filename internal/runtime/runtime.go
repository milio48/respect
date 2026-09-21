package runtime

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	blink "github.com/epkgs/blink"
	"respect-app/assets"
	"respect-app/internal/payload"
)

// Run menampilkan jendela Miniblink sesuai konfigurasi payload.
func Run(cfg *payload.Config) {
	cfg.Defaults()

	app := blink.NewApp()
	defer app.Exit()

	view := app.CreateWebWindowPopup(blink.WithWebWindowSize(int32(cfg.Width), int32(cfg.Height)))
	if len(assets.RespectIcon) > 0 {
		view.Window.SetIconFromBytes(assets.RespectIcon)
	}
	view.Window.SetTitle(cfg.Title)
	view.Window.MoveToCenter()


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
			loadErrorHTML(view, "Gagal memuat path file: "+err.Error())
		}

	case "html":
		// Simpan HTML string ke temp file untuk mendukung dokumen single-file besar tanpa limitasi data URI
		h := sha256.Sum256([]byte(cfg.Source))
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("respect_app_%x.html", h[:8]))
		if err := os.WriteFile(tmpFile, []byte(cfg.Source), 0644); err == nil {
			view.LoadURL("file:///" + filepath.ToSlash(tmpFile))
		} else {
			// Fallback ke data URI jika penulisan temp file gagal
			dataURI := "data:text/html;charset=utf-8," + url.PathEscape(cfg.Source)
			view.LoadURL(dataURI)
		}

	default:
		loadErrorHTML(view, "Mode tidak dikenal: "+cfg.Mode)
	}

	view.ShowWindow()

	view.OnDestroy(func() {
		os.Exit(0)
	})

	app.KeepRunning()
}

func loadErrorHTML(view *blink.View, msg string) {
	escaped := url.PathEscape(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Error</title>
<style>body{font-family:system-ui,sans-serif;padding:32px;background:#1a1a1a;color:#ff5555;text-align:center;}</style>
</head>
<body><h2>respect.exe — Kesalahan</h2><p>%s</p></body>
</html>`, msg))
	view.LoadURL("data:text/html;charset=utf-8," + escaped)
}
