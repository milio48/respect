//go:build !v132

package builder

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	blink "github.com/epkgs/blink"
	"respect-app/assets"
	"respect-app/internal/payload"
	"respect-app/internal/version"
)

//go:embed static
var static embed.FS

// Run menjalankan antarmuka grafis (GUI) builder respect-lite.exe menggunakan Miniblink 49.
func Run() {
	app := blink.NewApp()
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

	// Daftarkan IPC handler untuk menerima instruksi build dari frontend JavaScript
	app.IPC.Handle("build-app", func(cfgJSON string) string {
		var cfg payload.Config
		if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
			return errJSON(err)
		}

		if strings.TrimSpace(cfg.Source) == "" {
			return errJSON(errors.New("sumber konten (source) wajib diisi"))
		}
		if strings.TrimSpace(cfg.OutName) == "" {
			cfg.OutName = "demo.exe"
		}
		if !strings.HasSuffix(strings.ToLower(cfg.OutName), ".exe") {
			cfg.OutName += ".exe"
		}

		if err := payload.BuildSelf(cfg); err != nil {
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
