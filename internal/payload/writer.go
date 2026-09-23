package payload

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
	"respect-app/internal/cryptopayload"
	"respect-app/internal/tarball"
)

// MetadataApplier adalah fungsi untuk menginjeksi icon dan metadata PE ke base EXE.
var MetadataApplier func(targetExe string, cfg Config) error

// IconInjector adalah fungsi opsional untuk menginjeksi icon ke base EXE sebelum trailer ditempel.
var IconInjector func(targetExe, iconPath string) error

// DefaultIconInjector menyuntikkan icon bawaan Respect jika user tidak menyertakan icon kustom.
var DefaultIconInjector func(targetExe string) error

// prepareBaseEXE mempersiapkan file base executable sebelum ditempel trailer payload.
func prepareBaseEXE(cfg Config) error {
	if cfg.OutName == "" {
		return errors.New("out_name wajib diisi")
	}

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

	// Injeksi metadata PE dan icon sebelum trailer ditempel
	if MetadataApplier != nil {
		if err := MetadataApplier(cfg.OutName, cfg); err != nil {
			return err
		}
	} else {
		if cfg.IconPath != "" && IconInjector != nil {
			if err := IconInjector(cfg.OutName, cfg.IconPath); err != nil {
				return err
			}
		} else if DefaultIconInjector != nil {
			_ = DefaultIconInjector(cfg.OutName)
		}
	}

	return nil
}

// appendTrailer menempelkan data blob + 8-byte panjang + magic string ke ujung executable.
func appendTrailer(targetExe string, blob []byte, magic string) error {
	out, err := os.OpenFile(targetExe, os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(blob); err != nil {
		return err
	}

	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(blob)))
	if _, err := out.Write(lenBuf); err != nil {
		return err
	}
	if _, err := out.Write([]byte(magic)); err != nil {
		return err
	}

	return nil
}

// BuildSelf menghasilkan file EXE baru (format V1 - JSON tunggal).
func BuildSelf(cfg Config) error {
	cfg.Defaults()
	if err := prepareBaseEXE(cfg); err != nil {
		return err
	}

	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return appendTrailer(cfg.OutName, cfgBytes, MagicV1)
}

// BuildSelfV2 menghasilkan file EXE baru (format V2 - TAR + Zstandard in-memory multi-file).
func BuildSelfV2(cfg Config, files map[string][]byte) error {
	cfg.Defaults()

	if files == nil {
		files = make(map[string][]byte)
	}

	// 1. Sisipkan respect.json ke dalam arsip TAR
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("gagal serialize config respect.json: %w", err)
	}
	files["respect.json"] = cfgBytes

	// 2. Kemas seluruh file ke dalam stream TAR
	tarBytes, err := tarball.Pack(files)
	if err != nil {
		return fmt.Errorf("gagal mengemas arsip tar: %w", err)
	}

	// 3. Kompresi stream TAR dengan Zstandard level tertinggi
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return fmt.Errorf("gagal inisialisasi zstd encoder: %w", err)
	}
	zstdBlob := enc.EncodeAll(tarBytes, nil)
	enc.Close()

	// 4. Siapkan base executable (cloning + rcedit stamping)
	if err := prepareBaseEXE(cfg); err != nil {
		return err
	}

	// 5. Baca sample PE header dan ukuran total base executable untuk derived key
	targetFile, err := os.Open(cfg.OutName)
	if err != nil {
		return fmt.Errorf("gagal membuka target exe untuk enkripsi: %w", err)
	}
	stat, err := targetFile.Stat()
	if err != nil {
		targetFile.Close()
		return fmt.Errorf("gagal stat target exe: %w", err)
	}
	baseExeSize := stat.Size()
	peSample := make([]byte, 4096)
	n, _ := targetFile.ReadAt(peSample, 0)
	targetFile.Close()
	peSample = peSample[:n]

	// 6. Enkripsi zstdBlob dengan AES-256-GCM
	encryptedBlob, err := cryptopayload.Encrypt(zstdBlob, peSample, baseExeSize)
	if err != nil {
		return fmt.Errorf("gagal mengenkripsi payload v2: %w", err)
	}

	// 7. Tempelkan trailer V2 terenkripsi
	return appendTrailer(cfg.OutName, encryptedBlob, MagicV2)
}
