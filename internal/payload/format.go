package payload

const (
	// Magic string versi 1 (JSON tunggal lama).
	MagicV1 = "RESPECTv1\x00\x00\x00\x00\x00\x00\x00"

	// Magic string versi 2 (TAR + Zstandard multi-file).
	MagicV2 = "RESPECTv2\x00\x00\x00\x00\x00\x00\x00"

	// Alias Magic untuk backward compatibility V1
	Magic = MagicV1

	// Ukuran magic (16 bytes) + uint64 payload_len (8 bytes).
	TrailerSize = 16 + 8
)

// Payload adalah data lengkap yang diekstrak dari trailer executable.
type Payload struct {
	Version int               `json:"version"` // 1 atau 2
	Config  Config            `json:"config"`
	Files   map[string][]byte `json:"-"` // Berisi file-file in-memory (hanya terisi jika Version == 2)
}

// Config adalah metadata yang di-append ke EXE.
type Config struct {
	Mode       string `json:"mode"`   // "url" | "html" | "file" | "app"
	Source     string `json:"source"` // URL / HTML string / path file / entry point
	Title      string `json:"title,omitempty"`
	Width      int    `json:"width,omitempty"`  // default 800
	Height     int    `json:"height,omitempty"` // default 600
	IconPath   string `json:"icon_path,omitempty"`
	OutName    string `json:"out_name,omitempty"`
	AppVersion string `json:"app_version,omitempty"` // e.g. "1.0.0"
	Company    string `json:"company,omitempty"`     // e.g. "PT Solusi Digital"
	Copyright  string `json:"copyright,omitempty"`   // e.g. "Copyright © 2026 PT Solusi Digital"
	ServerMode bool   `json:"server_mode,omitempty"` // Gunakan in-memory HTTP server lokal (127.0.0.1) untuk Secure Context & WebCrypto
}

// Defaults mengisi nilai default kalau kosong.
func (c *Config) Defaults() {
	if c.Width == 0 {
		c.Width = 800
	}
	if c.Height == 0 {
		c.Height = 600
	}
	if c.Title == "" {
		c.Title = "respect.exe"
	}
	if c.AppVersion == "" {
		c.AppVersion = "1.0.0"
	}
	if c.Copyright == "" && c.Title != "" && c.Title != "respect.exe" {
		c.Copyright = "Copyright © 2026 " + c.Title
	}
}
