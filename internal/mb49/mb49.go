//go:build !v132

package mb49

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"

	blink "github.com/epkgs/blink"
	"github.com/klauspost/compress/zstd"
	"respect-app/assets"
)

// InitApp menginisialisasi epkgs/blink dengan memastikan DLL Miniblink 49
// telah terdekompresi ke folder %LOCALAPPDATA%\respect_desktop\engine_v49\
func InitApp() (*blink.Blink, error) {
	arch := "x64"
	expectedSize := int64(42510848)
	if goruntime.GOARCH == "386" {
		arch = "x32"
		expectedSize = int64(35473920)
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	engineDir := filepath.Join(localAppData, "respect_desktop", "engine_v49")
	subDir := filepath.Join(engineDir, fmt.Sprintf("miniblink_4975_%s", arch))
	_ = os.MkdirAll(subDir, 0755)

	targetDLL := filepath.Join(subDir, "miniblink_49.dll")

	// 1. Cek apakah DLL sudah ada di cache lokal dengan ukuran yang sesuai
	needExtract := true
	if fi, err := os.Stat(targetDLL); err == nil && fi.Size() == expectedSize {
		needExtract = false
	}

	// 2. Jika belum ada dan aset kompresi ZSTD tersedia, dekompresi sekarang
	if needExtract && len(assets.Miniblink49DLLZst) > 0 {
		zr, err := zstd.NewReader(bytes.NewReader(assets.Miniblink49DLLZst))
		if err != nil {
			return nil, fmt.Errorf("gagal inisialisasi zstd reader untuk miniblink_49: %w", err)
		}
		defer zr.Close()

		pid := os.Getpid()
		tmpPath := fmt.Sprintf("%s.tmp.%d", targetDLL, pid)
		tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			// Jika gagal buka file baru, cek lagi apakah proses lain sudah selesai mengekstraknya
			if fi, sErr := os.Stat(targetDLL); sErr == nil && fi.Size() == expectedSize {
				needExtract = false
			} else {
				return nil, fmt.Errorf("gagal membuat file temp DLL (%s): %w", tmpPath, err)
			}
		}

		if needExtract {
			if _, err := io.Copy(tmpFile, zr); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpPath)
				return nil, fmt.Errorf("gagal mendekompresi DLL (%s): %w", targetDLL, err)
			}
			_ = tmpFile.Close()

			_ = os.Remove(targetDLL)
			if err := os.Rename(tmpPath, targetDLL); err != nil {
				if fi, sErr := os.Stat(targetDLL); sErr == nil && fi.Size() == expectedSize {
					_ = os.Remove(tmpPath)
				} else {
					return nil, fmt.Errorf("gagal meletakkan DLL Miniblink (%s): %w", targetDLL, err)
				}
			}
		}
	}

	// 3. Inisialisasi blink dengan memuat langsung targetDLL dari path absolut
	app := blink.NewApp(
		blink.WithTempPath(engineDir),
		blink.WithDllFile(targetDLL),
	)
	return app, nil
}
