package icon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"respect-app/assets"
)

// Metadata mendefinisikan informasi PE resource yang akan disuntikkan ke EXE.
type Metadata struct {
	ProductName     string
	FileDescription string
	CompanyName     string
	LegalCopyright  string
	Version         string // Contoh: "1.0.0"
	IconPath        string // Path file .ico eksternal (opsional)
}

// ensureRcedit mengekstrak embedded rcedit.exe ke folder temporary jika belum ada.
func ensureRcedit() (string, error) {
	if len(assets.Rcedit) == 0 {
		return "", fmt.Errorf("embedded rcedit.exe kosong")
	}
	rceditPath := filepath.Join(os.TempDir(), "respect_rcedit.exe")
	if _, err := os.Stat(rceditPath); err != nil {
		if err := os.WriteFile(rceditPath, assets.Rcedit, 0755); err != nil {
			return "", fmt.Errorf("gagal mengekstrak rcedit: %w", err)
		}
	}
	return rceditPath, nil
}

// ApplyMetadata menyuntikkan icon, versi, dan metadata PE ke targetExe menggunakan rcedit.
func ApplyMetadata(targetExe string, meta Metadata) error {
	rceditPath, err := ensureRcedit()
	if err != nil {
		return err
	}

	if _, err := os.Stat(targetExe); err != nil {
		return fmt.Errorf("file target exe tidak ditemukan: %w", err)
	}

	targetAbs, err := filepath.Abs(targetExe)
	if err == nil {
		targetExe = targetAbs
	}

	args := []string{targetExe}

	// 1. Injeksi Icon
	iconPath := meta.IconPath
	if iconPath != "" {
		if _, err := os.Stat(iconPath); err == nil {
			if iconAbs, err := filepath.Abs(iconPath); err == nil {
				iconPath = iconAbs
			}
			args = append(args, "--set-icon", iconPath)
		}
	} else if len(assets.RespectIcon) > 0 {
		// Gunakan icon default bawaan Respect
		tempIco := filepath.Join(os.TempDir(), "respect_default.ico")
		if err := os.WriteFile(tempIco, assets.RespectIcon, 0644); err == nil {
			args = append(args, "--set-icon", tempIco)
		}
	}

	// 2. Versi Aplikasi & Versi File (Format X.Y.Z dan X.Y.Z.W)
	ver := strings.TrimSpace(meta.Version)
	if ver == "" {
		ver = "1.0.0"
	}
	fileVer := ver
	parts := strings.Split(fileVer, ".")
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	fileVer = strings.Join(parts, ".")

	args = append(args, "--set-product-version", ver)
	args = append(args, "--set-file-version", fileVer)

	// 3. String Metadata Windows PE
	prodName := strings.TrimSpace(meta.ProductName)
	if prodName == "" {
		prodName = strings.TrimSuffix(filepath.Base(targetExe), filepath.Ext(targetExe))
	}
	args = append(args, "--set-version-string", "ProductName", prodName)

	fileDesc := strings.TrimSpace(meta.FileDescription)
	if fileDesc == "" {
		fileDesc = prodName + " Desktop Application"
	}
	args = append(args, "--set-version-string", "FileDescription", fileDesc)

	if strings.TrimSpace(meta.CompanyName) != "" {
		args = append(args, "--set-version-string", "CompanyName", strings.TrimSpace(meta.CompanyName))
	}
	if strings.TrimSpace(meta.LegalCopyright) != "" {
		args = append(args, "--set-version-string", "LegalCopyright", strings.TrimSpace(meta.LegalCopyright))
	}

	cmd := exec.Command(rceditPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rcedit error (%w): %s", err, string(output))
	}

	return nil
}

// InjectIcon mengganti icon pada targetExe memakai rcedit.exe
func InjectIcon(targetExe, iconPath string) error {
	return ApplyMetadata(targetExe, Metadata{
		IconPath: iconPath,
	})
}

// InjectDefaultIcon menyuntikkan icon default Respect ke targetExe
func InjectDefaultIcon(targetExe string) error {
	return ApplyMetadata(targetExe, Metadata{})
}
