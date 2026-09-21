package icon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"respect-app/assets"
)

// InjectIcon mengganti icon pada targetExe memakai rcedit.exe
// yang diekstrak ke folder temporary Windows.
func InjectIcon(targetExe, iconPath string) error {
	if len(assets.Rcedit) == 0 {
		return fmt.Errorf("embedded rcedit.exe kosong")
	}

	if _, err := os.Stat(targetExe); err != nil {
		return fmt.Errorf("file target exe tidak ditemukan: %w", err)
	}
	if _, err := os.Stat(iconPath); err != nil {
		return fmt.Errorf("file icon tidak ditemukan: %w", err)
	}

	rceditPath := filepath.Join(os.TempDir(), "respect_rcedit.exe")
	if err := os.WriteFile(rceditPath, assets.Rcedit, 0755); err != nil {
		return fmt.Errorf("gagal mengekstrak rcedit: %w", err)
	}

	targetAbs, err := filepath.Abs(targetExe)
	if err == nil {
		targetExe = targetAbs
	}
	iconAbs, err := filepath.Abs(iconPath)
	if err == nil {
		iconPath = iconAbs
	}

	cmd := exec.Command(rceditPath, targetExe, "--set-icon", iconPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rcedit error (%w): %s", err, string(output))
	}

	return nil
}
