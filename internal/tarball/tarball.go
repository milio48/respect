package tarball

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
)

// Daftar ekstensi file berbahaya yang ditolak keras demi keamanan
var blockedExtensions = map[string]bool{
	".exe": true,
	".dll": true,
	".bat": true,
	".cmd": true,
	".ps1": true,
	".vbs": true,
	".scr": true,
	".msi": true,
	".com": true,
	".pif": true,
	".cpl": true,
}

// Pack mengemas kumpulan file in-memory (map path -> bytes) ke dalam format tar.
func Pack(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for rawPath, data := range files {
		cleanPath := sanitizePath(rawPath)
		if cleanPath == "" || cleanPath == "." {
			continue
		}

		ext := strings.ToLower(filepath.Ext(cleanPath))
		if blockedExtensions[ext] {
			return nil, fmt.Errorf("file terlarang terdeteksi: %s (%s)", cleanPath, ext)
		}

		hdr := &tar.Header{
			Name:     cleanPath,
			Mode:     0644,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("gagal menulis header tar untuk %s: %w", cleanPath, err)
		}

		if _, err := tw.Write(data); err != nil {
			return nil, fmt.Errorf("gagal menulis data tar untuk %s: %w", cleanPath, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("gagal menutup tar writer: %w", err)
	}

	return buf.Bytes(), nil
}

// UnpackToMemory membaca stream tar langsung ke memori (map[path][]byte)
// dengan proteksi keamanan ketat (zip-slip traversal & extension filtering).
func UnpackToMemory(r io.Reader) (map[string][]byte, error) {
	tr := tar.NewReader(r)
	files := make(map[string][]byte)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gagal membaca entri tar: %w", err)
		}

		// Hanya izinkan file reguler, tolak symlink, hardlink, fifo, dev
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}

		cleanPath := sanitizePath(hdr.Name)
		if cleanPath == "" || cleanPath == "." {
			continue
		}

		// Tolak keras jika mencoba path traversal (zip-slip)
		if strings.Contains(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") || strings.HasPrefix(cleanPath, "\\") {
			return nil, fmt.Errorf("keamanan terlanggar: zip-slip path traversal terdeteksi (%s)", hdr.Name)
		}

		ext := strings.ToLower(filepath.Ext(cleanPath))
		if blockedExtensions[ext] {
			return nil, fmt.Errorf("keamanan terlanggar: file biner/executable ditolak (%s)", cleanPath)
		}

		var data bytes.Buffer
		if _, err := io.Copy(&data, tr); err != nil {
			return nil, fmt.Errorf("gagal membaca konten file %s: %w", cleanPath, err)
		}

		files[cleanPath] = data.Bytes()
	}

	if len(files) == 0 {
		return nil, errors.New("arsip tar kosong atau tidak memiliki file valid")
	}

	return files, nil
}

// sanitizePath membersihkan path file dan mengubah semua separator ke forward slash (/).
func sanitizePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	p = path.Clean(p)
	if p == "." {
		return ""
	}
	return p
}

// DetectMIME menentukan MIME type berdasarkan ekstensi file untuk kebutuhan web browser.
func DetectMIME(name string) string {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".wasm":
		return "application/wasm"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogg":
		return "audio/ogg"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
