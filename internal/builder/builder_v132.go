//go:build v132

package builder

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"respect-app/assets"
	"respect-app/internal/mb132"
	"respect-app/internal/payload"
	"respect-app/internal/version"
)

//go:embed static/index.html
var indexHTML string

// Run menjalankan antarmuka grafis (GUI) builder respect.exe menggunakan Chromium 132.
func Run() {
	runtime.LockOSThread()
	title := fmt.Sprintf("%s — Standalone EXE Builder", version.BinaryName)
	view, err := mb132.CreateWebWindow(title, 720, 680)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gagal menginisialisasi jendela builder: %v\n", err)
		return
	}

	if len(assets.RespectIcon) > 0 {
		_ = view.SetIcon(assets.RespectIcon)
	}

	// Daftarkan handler query JavaScript (window.mbQuery)
	view.HandleQuery(func(cfgJSON string) string {
		switch parseCommand(cfgJSON) {
		case cmdPickIcon:
			return pickIconJSON(view.HostHWND())
		case cmdPickFolder:
			return pickFolderJSON(view.HostHWND())
		case cmdPickHTML:
			return pickHTMLJSON(view.HostHWND())
		}

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

	// Muat kode HTML builder secara native
	view.LoadHTML(indexHTML, "http://builder/")
	view.Show()

	mb132.RunMessageLoop()
}

func errJSON(err error) string {
	b, _ := json.Marshal(map[string]interface{}{
		"ok":    false,
		"error": err.Error(),
	})
	return string(b)
}
