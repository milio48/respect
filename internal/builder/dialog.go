package builder

import (
	"encoding/json"
	"strings"

	"respect-app/internal/filedialog"
)

// cmdPickIcon adalah perintah frontend untuk membuka dialog pilih file icon.
const cmdPickIcon = "pick-icon"

// pickIconFn memisahkan pemanggilan dialog native agar bisa diganti saat pengujian.
var pickIconFn = filedialog.PickIcon

// requestCommand hanya membaca field "cmd" dari payload frontend sehingga
// payload build yang lama tetap 100% kompatibel.
type requestCommand struct {
	Cmd string `json:"cmd"`
}

func parseCommand(reqJSON string) string {
	var probe requestCommand
	if err := json.Unmarshal([]byte(reqJSON), &probe); err != nil {
		return ""
	}
	return strings.TrimSpace(probe.Cmd)
}

// pickIconJSON membuka dialog pilih icon native dan mengembalikan respons JSON
// yang siap dipakai frontend. Owner adalah HWND jendela builder.
func pickIconJSON(owner uintptr) string {
	path, err := pickIconFn(owner)
	if err != nil {
		return errJSON(err)
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"ok":        true,
		"path":      path,
		"cancelled": path == "",
	})
	return string(resp)
}
