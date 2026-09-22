//go:build !windows

package filedialog

import "errors"

var errUnsupported = errors.New("dialog file hanya tersedia di Windows")

// PickIcon tidak didukung di luar Windows.
func PickIcon(owner uintptr) (string, error) {
	return "", errUnsupported
}

// PickFile tidak didukung di luar Windows.
func PickFile(owner uintptr, title, filter string) (string, error) {
	return "", errUnsupported
}
