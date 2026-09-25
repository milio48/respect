//go:build !v132

package builder

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	blink "github.com/epkgs/blink"
	"respect-app/assets"
	"respect-app/internal/mb49"
	"respect-app/internal/payload"
	"respect-app/internal/version"
)

//go:embed static
var static embed.FS

// Run menjalankan antarmuka grafis (GUI) builder respect-lite.exe menggunakan Miniblink 49.
func Run() {
	app, err := mb49.InitApp()
	if err != nil {
		return
	}
	defer app.Exit()

	res, err := fs.Sub(static, "static")
	if err == nil {
		app.Resource.Bind("builder", res)
	}

	view := app.CreateWebWindowPopup(blink.WithWebWindowSize(720, 680))
	if len(assets.RespectIcon) > 0 {
		view.Window.SetIconFromBytes(assets.RespectIcon)
	}
	title := fmt.Sprintf("%s — Standalone EXE Builder", version.BinaryName)
	view.Window.SetTitle(title)
	view.Window.MoveToCenter()

	// Isolasi cookie dan local storage agar tidak mencemari direktori aplikasi
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = os.TempDir()
	}
	appDir := filepath.Join(localAppData, "respect_desktop", "apps", "respect-builder")
	_ = os.MkdirAll(filepath.Join(appDir, "storage"), 0755)
	view.SetCookieJarFullPath(filepath.Join(appDir, "cookie.dat"))
	view.SetLocalStorageFullPath(filepath.Join(appDir, "storage"))

	// Daftarkan IPC handler untuk menerima instruksi build dari frontend JavaScript
	app.IPC.Handle("build-app", func(cfgJSON string) string {
		var cfg payload.Config
		if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
			return errJSON(err)
		}

		if err := buildAppFromConfig(cfg); err != nil {
			return errJSON(err)
		}

		absOut, err := filepath.Abs(cfg.OutName)
		if err != nil {
			absOut = cfg.OutName
		}

		respBytes, _ := json.Marshal(map[string]interface{}{
			"ok":   true,
			"path": absOut,
			"name": filepath.Base(absOut),
		})
		return string(respBytes)
	})

	// Daftarkan IPC handler untuk dialog native
	app.IPC.Handle(cmdPickIcon, func(_ string) string {
		return pickIconJSON(ownerHWND(view))
	})
	app.IPC.Handle(cmdPickFolder, func(_ string) string {
		return pickFolderJSON(ownerHWND(view))
	})
	app.IPC.Handle(cmdPickHTML, func(_ string) string {
		return pickHTMLJSON(ownerHWND(view))
	})

	view.LoadURL("http://builder/index.html")
	view.ShowWindow()

	view.OnDestroy(func() {
		os.Exit(0)
	})

	app.KeepRunning()
}

func errJSON(err error) string {
	b, _ := json.Marshal(map[string]interface{}{
		"ok":    false,
		"error": err.Error(),
	})
	return string(b)
}

// ownerHWND mengembalikan HWND jendela builder, atau 0 bila belum tersedia.
func ownerHWND(view *blink.View) uintptr {
	if view == nil || view.Window == nil {
		return 0
	}
	return uintptr(view.Window.Hwnd)
}
