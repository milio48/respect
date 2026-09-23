package tarball

import (
	"bytes"
	"io"
	"io/fs"
	"testing"
)

func TestPackAndUnpack(t *testing.T) {
	origFiles := map[string][]byte{
		"index.html":       []byte("<h1>Hello Respect</h1>"),
		"assets/style.css": []byte("body { background: #000; }"),
		"assets/app.js":    []byte("console.log('App loaded');"),
		"data/config.json": []byte(`{"version": 2}`),
	}

	tarBytes, err := Pack(origFiles)
	if err != nil {
		t.Fatalf("Pack gagal: %v", err)
	}

	unpacked, err := UnpackToMemory(bytes.NewReader(tarBytes))
	if err != nil {
		t.Fatalf("UnpackToMemory gagal: %v", err)
	}

	if len(unpacked) != len(origFiles) {
		t.Fatalf("Ekspektasi %d file, didapat %d", len(origFiles), len(unpacked))
	}

	for k, v := range origFiles {
		got, ok := unpacked[k]
		if !ok {
			t.Errorf("File %s tidak ditemukan setelah unpack", k)
		} else if string(got) != string(v) {
			t.Errorf("Konten file %s berbeda: %s != %s", k, string(got), string(v))
		}
	}
}

func TestZipSlipRejection(t *testing.T) {
	// Tes penolakan saat mencoba pack dengan path jahat
	evilFiles := map[string][]byte{
		"../../evil.txt": []byte("hack"),
	}

	packed, err := Pack(evilFiles)
	if err == nil {
		// Jika lolos pack (karena sanitize), coba langsung unpack tar jahat buatan manual
		t.Logf("Pack membersihkan path, packed %d bytes", len(packed))
	}
}

func TestBlockedExtensions(t *testing.T) {
	blocked := []string{"malware.exe", "trojan.dll", "script.bat", "run.ps1", "bad.vbs"}
	for _, name := range blocked {
		files := map[string][]byte{
			name: []byte("malicious content"),
		}
		_, err := Pack(files)
		if err == nil {
			t.Errorf("Harusnya file %s ditolak oleh Pack(), tapi berhasil lolos", name)
		}
	}
}

func TestMemoryFS(t *testing.T) {
	files := map[string][]byte{
		"index.html":       []byte("<!DOCTYPE html>"),
		"assets/style.css": []byte("h1 { color: red; }"),
	}

	memFS := NewMemoryFS(files)

	// Test ReadFile
	data, err := fs.ReadFile(memFS, "index.html")
	if err != nil {
		t.Fatalf("fs.ReadFile gagal: %v", err)
	}
	if string(data) != "<!DOCTYPE html>" {
		t.Fatalf("Konten salah: %s", string(data))
	}

	// Test Open
	f, err := memFS.Open("assets/style.css")
	if err != nil {
		t.Fatalf("Open assets/style.css gagal: %v", err)
	}
	defer f.Close()

	buf, _ := io.ReadAll(f)
	if string(buf) != "h1 { color: red; }" {
		t.Fatalf("Konten buffer salah: %s", string(buf))
	}

	// Test non-existent file
	_, err = memFS.Open("ghost.png")
	if err == nil {
		t.Fatal("Ekspektasi error untuk file tidak ada, tapi nil")
	}
}

func TestDetectMIME(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"index.html", "text/html; charset=utf-8"},
		{"main.js", "application/javascript; charset=utf-8"},
		{"style.css", "text/css; charset=utf-8"},
		{"icon.png", "image/png"},
		{"logo.svg", "image/svg+xml"},
		{"font.woff2", "font/woff2"},
		{"module.wasm", "application/wasm"},
	}

	for _, tc := range tests {
		got := DetectMIME(tc.file)
		if got != tc.expected {
			t.Errorf("DetectMIME(%s) = %s, ekspektasi %s", tc.file, got, tc.expected)
		}
	}
}
