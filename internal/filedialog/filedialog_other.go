//go:build !windows

package filedialog

import "errors"

var errUnsupported = errors.New("dialog file hanya tersedia di Windows")

// PickIcon tidak didukung di luar Windows.
func PickIcon(owner uintptr) (string, error) {
	return "", errUnsupported
}

// PickFolder tidak didukung di luar Windows.
func PickFolder(owner uintptr, title string) (string, error) {
	return "", errUnsupported
}

// PickHTML tidak didukung di luar Windows.
func PickHTML(owner uintptr) (string, error) {
	return "", errUnsupported
}
