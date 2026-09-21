package payload

const (
	// Magic 16 bytes di ujung file.
	Magic = "RESPECTv1\x00\x00\x00\x00\x00\x00\x00"

	// Ukuran magic (16 bytes) + uint64 config_len (8 bytes).
	TrailerSize = 16 + 8
)

// Config adalah metadata yang di-append ke EXE.
type Config struct {
	Mode     string `json:"mode"`               // "url" | "html" | "file"
	Source   string `json:"source"`             // URL / HTML string / path file
	Title    string `json:"title,omitempty"`
	Width    int    `json:"width,omitempty"`    // default 800
	Height   int    `json:"height,omitempty"`   // default 600
	IconPath string `json:"icon_path,omitempty"`
	OutName  string `json:"out_name,omitempty"`
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
}
