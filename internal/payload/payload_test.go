package payload

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStripTrailerAndReadPayload(t *testing.T) {
	dummyExe := []byte("MZ_DUMMY_EXECUTABLE_CONTENT_1234567890")

	cfg := Config{
		Mode:    "url",
		Source:  "https://example.com",
		Title:   "Test App",
		Width:   1024,
		Height:  768,
		OutName: "test.exe",
	}
	cfg.Defaults()

	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(cfgBytes)))

	// Gabungkan dummy EXE + cfgBytes + lenBuf + Magic
	var fullFile []byte
	fullFile = append(fullFile, dummyExe...)
	fullFile = append(fullFile, cfgBytes...)
	fullFile = append(fullFile, lenBuf...)
	fullFile = append(fullFile, []byte(Magic)...)

	// Simpan ke file temp
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test_payload.exe")
	if err := os.WriteFile(tmpPath, fullFile, 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Test ReadPayloadFrom
	readCfg, err := ReadPayloadFrom(tmpPath)
	if err != nil {
		t.Fatalf("expected payload, got error: %v", err)
	}

	if readCfg.Mode != cfg.Mode || readCfg.Source != cfg.Source || readCfg.Title != cfg.Title {
		t.Fatalf("payload mismatch: %+v != %+v", readCfg, cfg)
	}
	if readCfg.Width != 1024 || readCfg.Height != 768 {
		t.Fatalf("dimensions mismatch: width=%d height=%d", readCfg.Width, readCfg.Height)
	}

	// Test StripTrailer
	stripped := StripTrailer(fullFile)
	if string(stripped) != string(dummyExe) {
		t.Fatalf("stripped content mismatch: got %d bytes, expected %d bytes", len(stripped), len(dummyExe))
	}

	// Test StripTrailer on clean file without trailer
	strippedClean := StripTrailer(dummyExe)
	if string(strippedClean) != string(dummyExe) {
		t.Fatalf("strippedClean mismatch")
	}
}

func TestReadActualDemoExe(t *testing.T) {
	// Cek apakah demo.exe ada di root project
	demoPath := filepath.Join("..", "..", "demo.exe")
	if _, err := os.Stat(demoPath); os.IsNotExist(err) {
		t.Skip("demo.exe belum dibangun, lewati pengujian ini")
	}

	cfg, err := ReadPayloadFrom(demoPath)
	if err != nil {
		t.Fatalf("gagal membaca payload dari demo.exe: %v", err)
	}

	if cfg.Mode != "url" {
		t.Errorf("expected mode 'url', got '%s'", cfg.Mode)
	}
	if cfg.Source != "https://example.com" {
		t.Errorf("expected source 'https://example.com', got '%s'", cfg.Source)
	}
	if cfg.Title != "Demo App" {
		t.Errorf("expected title 'Demo App', got '%s'", cfg.Title)
	}
}
