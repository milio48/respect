//go:build !v132

package runtime

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"unsafe"

	blink "github.com/epkgs/blink"
	"respect-app/assets"
	"respect-app/internal/localserver"
	"respect-app/internal/payload"
	"respect-app/internal/tarball"
	"respect-app/internal/winprint"
)

// User-Agent modern untuk kompatibilitas web Miniblink 49
const defaultModernUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\x00"

// Run menampilkan jendela Miniblink 49 sesuai konfigurasi payload (V1 atau V2).
func Run(p *payload.Payload) {
	goruntime.LockOSThread()
	cfg := &p.Config
	cfg.Defaults()

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	engineDir := filepath.Join(localAppData, "respect_desktop", "engine_v49")
	_ = os.MkdirAll(engineDir, 0755)

	// Pastikan Miniblink memuat miniblink_49.dll dari AppData terisolasi agar tidak mengotori %TEMP%
	// dan tidak bentrok dengan blink.dll (Chromium 132 Modern) jika berada di direktori yang sama
	app := blink.NewApp(
		blink.WithTempPath(engineDir),
		blink.WithDllFile("miniblink_49.dll"),
	)
	defer app.Exit()

	// Daftarkan handler IPC standar untuk runtime Lite
	app.IPC.Handle("respect_stress_ping", func(arg string) string {
		return `{"ok":true,"pong":"respect-lite"}`
	})
	app.IPC.Handle("ping", func(_ string) string {
		return `{"ok":true,"engine":"v49"}`
	})

	var currentHWND uintptr
	app.IPC.Handle("page.print", func(payloadJSON string) string {
		var p struct {
			Title string `json:"title"`
		}
		_ = json.Unmarshal([]byte(payloadJSON), &p)
		if p.Title == "" {
			p.Title = cfg.Title
		}
		if p.Title == "" {
			p.Title = "Respect Document"
		}
		printed, err := winprint.PrintWindowContent(currentHWND, p.Title)
		if err != nil {
			return fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
		}
		return fmt.Sprintf(`{"ok":true,"printed":%v}`, printed)
	})

	// Injeksi boot script untuk menyetel bahasa navigator ke Indonesia/Inggris & dukungan cetak (ES5 murni)
	app.AddBootScript(`
try {
	Object.defineProperty(navigator, 'language', { get: function () { return 'id-ID'; } });
	Object.defineProperty(navigator, 'languages', { get: function () { return ['id-ID', 'id', 'en-US', 'en']; } });
} catch (e) {}

try {
	window.print = function () {
		try { window.dispatchEvent(new Event('beforeprint')); } catch (e) {}
		var title = document.title || 'Respect Document';
		var payload = JSON.stringify({ title: title });
		if (window.ipc && typeof window.ipc.invoke === 'function') {
			return window.ipc.invoke('page.print', payload).then(function (res) {
				try { window.dispatchEvent(new Event('afterprint')); } catch (e) {}
				return res;
			}).catch(function (err) {
				try { window.dispatchEvent(new Event('afterprint')); } catch (e) {}
				throw err;
			});
		}
		return Promise.resolve();
	};

	window.addEventListener('keydown', function (e) {
		if ((e.ctrlKey || e.metaKey) && (e.key === 'p' || e.key === 'P' || e.keyCode === 80)) {
			e.preventDefault();
			window.print();
		}
	});

	window.respect = window.respect || {};
	window.respect.print = function () { return window.print(); };
	window.respect.printElement = function (target) {
		var el = typeof target === 'string' ? document.querySelector(target) : target;
		if (!el) {
			console.warn('[Respect] Element not found for print:', target);
			return Promise.reject(new Error('Element not found'));
		}
		var styleId = '__respect_print_element_style__';
		var style = document.getElementById(styleId);
		if (!style) {
			style = document.createElement('style');
			style.id = styleId;
			style.textContent =
				'@media print {' +
				'  body * { visibility: hidden !important; }' +
				'  .__respect_print_target__, .__respect_print_target__ * { visibility: visible !important; }' +
				'  .__respect_print_target__ { position: absolute !important; left: 0 !important; top: 0 !important; width: 100% !important; margin: 0 !important; }' +
				'}';
			document.head.appendChild(style);
		}
		el.classList.add('__respect_print_target__');
		var cleanup = function () {
			el.classList.remove('__respect_print_target__');
			window.removeEventListener('afterprint', cleanup);
		};
		window.addEventListener('afterprint', cleanup);
		setTimeout(cleanup, 3000);
		return window.print();
	};
} catch (e) {}
`)

	view := app.CreateWebWindowPopup(blink.WithWebWindowSize(int32(cfg.Width), int32(cfg.Height)))
	if view == nil || view.Hwnd == 0 || view.GetWindowHandle() == 0 {
		return
	}
	if len(assets.RespectIcon) > 0 {
		view.Window.SetIconFromBytes(assets.RespectIcon)
	}
	view.Window.SetTitle(cfg.Title)
	view.Window.MoveToCenter()
	if view.Window != nil {
		currentHWND = uintptr(view.Window.Hwnd)
	}

	// 1. Injeksi User-Agent modern ke webview Miniblink
	uaBytes := []byte(defaultModernUA)
	_, _, _ = app.CallFunc("wkeSetUserAgent", uintptr(view.GetWindowHandle()), uintptr(unsafe.Pointer(&uaBytes[0])))

	// 2. Isolasi penyimpanan (Dynamic Sandbox per-aplikasi) agar tidak mengotori working directory
	appName := "default"
	if exePath, err := os.Executable(); err == nil {
		base := filepath.Base(exePath)
		appName = strings.TrimSuffix(base, filepath.Ext(base))
	}
	appDir := filepath.Join(localAppData, "respect_desktop", "apps", appName)
	_ = os.MkdirAll(filepath.Join(appDir, "storage"), 0755)

	view.SetCookieJarFullPath(filepath.Join(appDir, "cookie.dat"))
	view.SetLocalStorageFullPath(filepath.Join(appDir, "storage"))

	// 3. Jika payload Versi 2 (In-Memory TAR Virtual Host atau In-Memory Local Server)
	if p.Version == 2 && len(p.Files) > 0 {
		entryFile := "index.html"
		if cfg.Source != "" && !strings.HasPrefix(cfg.Source, "http") {
			entryFile = strings.TrimPrefix(cfg.Source, "/")
		}

		if cfg.ServerMode || cfg.Mode == "server" {
			srv, err := localserver.Start(p.Files)
			if err == nil {
				view.LoadURL(srv.URL(entryFile))
				view.ShowWindow()
				view.OnDestroy(func() {
					srv.Close()
					os.Exit(0)
				})
				app.KeepRunning()
				return
			}
		}

		memFS := tarball.NewMemoryFS(p.Files)
		_ = app.Resource.Bind("app", memFS)
		entryURL := "http://app/" + entryFile
		view.LoadURL(entryURL)
		view.ShowWindow()
		view.OnDestroy(func() {
			os.Exit(0)
		})
		app.KeepRunning()
		return
	}

	// 4. Jika payload Versi 1 (JSON tunggal legacy)
	switch cfg.Mode {
	case "url":
		targetURL := strings.TrimSpace(cfg.Source)
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") && !strings.HasPrefix(targetURL, "file://") {
			targetURL = "https://" + targetURL
		}
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
			loadErrorHTML(view, cfg.Title, "Gagal memuat path file: "+err.Error())
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
		loadErrorHTML(view, cfg.Title, "Mode tidak dikenal: "+cfg.Mode)
	}

	view.ShowWindow()

	view.OnDestroy(func() {
		os.Exit(0)
	})

	app.KeepRunning()
}

func loadErrorHTML(view *blink.View, title, msg string) {
	if title == "" {
		title = "Respect Lite"
	}
	escaped := url.PathEscape(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Error</title>
<style>body{font-family:system-ui,sans-serif;padding:32px;background:#1a1a1a;color:#ff5555;text-align:center;}</style>
</head>
<body><h2>%s — Kesalahan</h2><p>%s</p></body>
</html>`, title, msg))
	view.LoadURL("data:text/html;charset=utf-8," + escaped)
}

