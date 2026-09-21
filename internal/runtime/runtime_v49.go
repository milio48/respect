//go:build !v132

package runtime

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	blink "github.com/epkgs/blink"
	"respect-app/assets"
	"respect-app/internal/payload"
)

// User-Agent modern untuk kompatibilitas web Miniblink 49
const defaultModernUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\x00"

// Run menampilkan jendela Miniblink 49 sesuai konfigurasi payload.
func Run(cfg *payload.Config) {
	cfg.Defaults()

	app := blink.NewApp()
	defer app.Exit()

	// Injeksi boot script untuk menyetel bahasa navigator ke Indonesia/Inggris
	app.AddBootScript(`
try {
	Object.defineProperty(navigator, 'language', { get: () => 'id-ID' });
	Object.defineProperty(navigator, 'languages', { get: () => ['id-ID', 'id', 'en-US', 'en'] });
} catch (e) {}
`)

	view := app.CreateWebWindowPopup(blink.WithWebWindowSize(int32(cfg.Width), int32(cfg.Height)))
	if len(assets.RespectIcon) > 0 {
		view.Window.SetIconFromBytes(assets.RespectIcon)
	}
	view.Window.SetTitle(cfg.Title)
	view.Window.MoveToCenter()

	// 1. Injeksi User-Agent modern ke webview Miniblink
	uaBytes := []byte(defaultModernUA)
	_, _, _ = app.CallFunc("wkeSetUserAgent", uintptr(view.GetWindowHandle()), uintptr(unsafe.Pointer(&uaBytes[0])))

	// 2. Isolasi penyimpanan (Dynamic Sandbox per-aplikasi) agar tidak mengotori working directory
	appName := "default"
	if exePath, err := os.Executable(); err == nil {
		base := filepath.Base(exePath)
		appName = strings.TrimSuffix(base, filepath.Ext(base))
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = os.TempDir()
	}
	appDir := filepath.Join(localAppData, "respect", "apps", appName)
	_ = os.MkdirAll(filepath.Join(appDir, "storage"), 0755)

	cookiePath := append([]byte(filepath.Join(appDir, "cookies.dat")), 0)
	_, _, _ = app.CallFunc("wkeSetCookieJarFullPath", uintptr(view.GetWindowHandle()), uintptr(unsafe.Pointer(&cookiePath[0])))

	storagePath := append([]byte(filepath.Join(appDir, "storage")), 0)
	_, _, _ = app.CallFunc("wkeSetLocalStorageFullPath", uintptr(view.GetWindowHandle()), uintptr(unsafe.Pointer(&storagePath[0])))

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
		// Simpan HTML string ke temp file untuk mendukung dokumen single-file besar
		h := sha256.Sum256([]byte(cfg.Source))
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("respect_app_%x.html", h[:8]))
		if err := os.WriteFile(tmpFile, []byte(cfg.Source), 0644); err == nil {
			view.LoadURL("file:///" + filepath.ToSlash(tmpFile))
		} else {
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
<body><h2>respect-lite.exe — Kesalahan</h2><p>%s</p></body>
</html>`, msg))
	view.LoadURL("data:text/html;charset=utf-8," + escaped)
}
