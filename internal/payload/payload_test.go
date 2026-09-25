package payload

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"respect-app/internal/cryptopayload"
	"respect-app/internal/tarball"
)

func TestPayloadV1(t *testing.T) {
	dummyExe := []byte("MZ_DUMMY_EXECUTABLE_CONTENT_V1")

	cfg := Config{
		Mode:    "url",
		Source:  "https://example.com",
		Title:   "Test App V1",
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

	var fullFile []byte
	fullFile = append(fullFile, dummyExe...)
	fullFile = append(fullFile, cfgBytes...)
	fullFile = append(fullFile, lenBuf...)
	fullFile = append(fullFile, []byte(MagicV1)...)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test_v1.exe")
	if err := os.WriteFile(tmpPath, fullFile, 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	p, err := ReadPayloadFrom(tmpPath)
	if err != nil {
		t.Fatalf("expected payload, got error: %v", err)
	}

	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}
	if p.Config.Title != cfg.Title {
		t.Errorf("title mismatch: %s != %s", p.Config.Title, cfg.Title)
	}

	// Test StripTrailer V1
	stripped := StripTrailer(fullFile)
	if string(stripped) != string(dummyExe) {
		t.Fatalf("stripped V1 mismatch")
	}
}

func TestPayloadV2(t *testing.T) {
	dummyExe := []byte("MZ_DUMMY_EXECUTABLE_CONTENT_V2")

	cfg := Config{
		Mode:    "app",
		Source:  "index.html",
		Title:   "Test App V2",
		Width:   1280,
		Height:  800,
		OutName: "test_v2.exe",
	}
	cfg.Defaults()

	cfgBytes, _ := json.Marshal(cfg)
	files := map[string][]byte{
		"respect.json":     cfgBytes,
		"index.html":       []byte("<h1>Hello V2</h1>"),
		"assets/style.css": []byte("body { color: blue; }"),
	}

	tarBytes, err := tarball.Pack(files)
	if err != nil {
		t.Fatalf("tarball.Pack gagal: %v", err)
	}

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		t.Fatalf("zstd encoder error: %v", err)
	}
	zstdBlob := enc.EncodeAll(tarBytes, nil)
	enc.Close()

	peSample := dummyExe
	if len(peSample) > 4096 {
		peSample = peSample[:4096]
	}
	encryptedBlob, err := cryptopayload.Encrypt(zstdBlob, peSample, int64(len(dummyExe)))
	if err != nil {
		t.Fatalf("cryptopayload.Encrypt gagal: %v", err)
	}

	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(encryptedBlob)))

	var fullFile []byte
	fullFile = append(fullFile, dummyExe...)
	fullFile = append(fullFile, encryptedBlob...)
	fullFile = append(fullFile, lenBuf...)
	fullFile = append(fullFile, []byte(MagicV2)...)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test_v2.exe")
	if err := os.WriteFile(tmpPath, fullFile, 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	p, err := ReadPayloadFrom(tmpPath)
	if err != nil {
		t.Fatalf("expected payload V2, got error: %v", err)
	}

	if p.Version != 2 {
		t.Errorf("expected version 2, got %d", p.Version)
	}
	if p.Config.Title != "Test App V2" {
		t.Errorf("title mismatch: %s != Test App V2", p.Config.Title)
	}
	if len(p.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(p.Files))
	}
	if string(p.Files["index.html"]) != "<h1>Hello V2</h1>" {
		t.Errorf("index.html content mismatch: %s", string(p.Files["index.html"]))
	}

	// Test StripTrailer V2
	stripped := StripTrailer(fullFile)
	if string(stripped) != string(dummyExe) {
		t.Fatalf("stripped V2 mismatch")
	}
}
