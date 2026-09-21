package payload

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// IconInjector adalah fungsi opsional untuk menginjeksi icon ke base EXE sebelum trailer ditempel.
var IconInjector func(targetExe, iconPath string) error

// DefaultIconInjector menyuntikkan icon bawaan Respect jika user tidak menyertakan icon kustom.
var DefaultIconInjector func(targetExe string) error

// BuildSelf menghasilkan file EXE baru berdasarkan EXE ini sendiri,
// dengan config di-append di ujungnya (trailer format).
func BuildSelf(cfg Config) error {
	if cfg.OutName == "" {
		return errors.New("out_name wajib diisi")
	}
	cfg.Defaults()

	self, err := os.Executable()
	if err != nil {
		return err
	}

	// Hindari overwrite file EXE yang sedang aktif berjalan
	selfAbs, err1 := filepath.Abs(self)
	outAbs, err2 := filepath.Abs(cfg.OutName)
	if err1 == nil && err2 == nil && selfAbs == outAbs {
		return errors.New("out_name tidak boleh sama dengan file EXE yang sedang berjalan")
	}

	selfBytes, err := os.ReadFile(self)
	if err != nil {
		return err
	}

	// Potong payload lama (kalau ada) supaya tidak menumpuk
	selfBytes = StripTrailer(selfBytes)

	// Serialize config
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	// Pastikan direktori tujuan tersedia
	outDir := filepath.Dir(cfg.OutName)
	if outDir != "" && outDir != "." {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}
	}

	// Tulis base EXE
	if err := os.WriteFile(cfg.OutName, selfBytes, 0755); err != nil {
		return err
	}

	// Injeksi icon jika ada sebelum trailer ditempel
	if cfg.IconPath != "" && IconInjector != nil {
		if err := IconInjector(cfg.OutName, cfg.IconPath); err != nil {
			return err
		}
	} else if DefaultIconInjector != nil {
		// Jika tidak ada icon kustom, suntikkan icon resmi Respect secara otomatis
		_ = DefaultIconInjector(cfg.OutName)
	}

	// Buka file dalam mode append untuk menempelkan trailer payload
	out, err := os.OpenFile(cfg.OutName, os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(cfgBytes); err != nil {
		return err
	}

	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(cfgBytes)))
	if _, err := out.Write(lenBuf); err != nil {
		return err
	}
	if _, err := out.Write([]byte(Magic)); err != nil {
		return err
	}

	return nil
}
