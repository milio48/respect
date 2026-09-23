package builder

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"respect-app/internal/payload"
)

// buildAppFromConfig memproses konfigurasi build dan memanggil BuildSelf (V1) atau BuildSelfV2 (V2).
func buildAppFromConfig(cfg payload.Config) error {
	if strings.TrimSpace(cfg.Source) == "" {
		return errors.New("sumber konten (source) wajib diisi")
	}
	if strings.TrimSpace(cfg.OutName) == "" {
		cfg.OutName = "demo.exe"
	}
	if !strings.HasSuffix(strings.ToLower(cfg.OutName), ".exe") {
		cfg.OutName += ".exe"
	}

	switch cfg.Mode {
	case "app", "dir":
		files, err := collectDirectory(cfg.Source)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return errors.New("direktori kosong atau tidak ada file valid")
		}
		cfg.Source = "index.html"
		return payload.BuildSelfV2(cfg, files)

	case "html":
		// Kemas kode HTML langsung ke Virtual Host V2 in-memory
		files := map[string][]byte{
			"index.html": []byte(cfg.Source),
		}
		cfg.Source = "index.html"
		return payload.BuildSelfV2(cfg, files)

	case "file":
		fi, err := os.Stat(cfg.Source)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			files, err := collectDirectory(cfg.Source)
			if err != nil {
				return err
			}
			cfg.Source = "index.html"
			return payload.BuildSelfV2(cfg, files)
		}

		// Jika single file HTML lokal: kemas file ini beserta seluruh asset di foldernya
		dir := filepath.Dir(cfg.Source)
		baseFile := filepath.Base(cfg.Source)
		files, err := collectDirectory(dir)
		if err != nil {
			files = make(map[string][]byte)
			data, rErr := os.ReadFile(cfg.Source)
			if rErr != nil {
				return rErr
			}
			files[baseFile] = data
		}
		cfg.Source = baseFile
		return payload.BuildSelfV2(cfg, files)

	case "url":
		return payload.BuildSelf(cfg)

	default:
		return payload.BuildSelf(cfg)
	}
}

// collectDirectory mengumpulkan seluruh file dalam folder menjadi map[relative_path][]byte.
func collectDirectory(dirPath string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := filepath.Walk(dirPath, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dirPath, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	return files, err
}
