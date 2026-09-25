//go:build windows

// Package filedialog menyediakan dialog pilih file native Windows (comdlg32)
// tanpa dependensi eksternal, sehingga bisa dipakai oleh engine Miniblink 132
// maupun Miniblink 49.
package filedialog

import (
	"errors"
	"fmt"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	comdlg32                 = syscall.NewLazyDLL("comdlg32.dll")
	user32                   = syscall.NewLazyDLL("user32.dll")
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	shell32                  = syscall.NewLazyDLL("shell32.dll")
	ole32                    = syscall.NewLazyDLL("ole32.dll")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")
	procGetWindowThreadPID   = user32.NewProc("GetWindowThreadProcessId")
	procGetCurrentThreadID   = kernel32.NewProc("GetCurrentThreadId")
	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
)

const (
	OFN_NOCHANGEDIR   = 0x00000008
	OFN_PATHMUSTEXIST = 0x00000800
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_EXPLORER      = 0x00080000

	BIF_RETURNONLYFSDIRS = 0x00000001
	BIF_NEWDIALOGSTYLE   = 0x00000040
)

// iconFilter adalah daftar filter dialog untuk file icon (.ico).
const iconFilter = "File Icon (*.ico)\x00*.ico\x00Semua File (*.*)\x00*.*\x00\x00"

// htmlFilter adalah daftar filter dialog untuk file HTML.
const htmlFilter = "File HTML (*.html;*.htm)\x00*.html;*.htm\x00Semua File (*.*)\x00*.*\x00\x00"

// browseInfoW adalah cerminan struktur BROWSEINFOW milik Windows untuk dialog pilih folder.
type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

// PickFolder membuka dialog pemilih folder native Windows.
func PickFolder(owner uintptr, title string) (string, error) {
	if err := procSHBrowseForFolderW.Find(); err != nil {
		return "", errors.New("dialog folder Windows tidak tersedia: " + err.Error())
	}

	if title == "" {
		title = "Pilih folder project web (dist / build)"
	}
	titleUTF16 := toUTF16(title)
	bi := browseInfoW{
		hwndOwner: safeOwner(owner),
		lpszTitle: &titleUTF16[0],
		ulFlags:   BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE,
	}

	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", nil // Pengguna membatalkan dialog
	}
	defer procCoTaskMemFree.Call(pidl)

	pathBuf := make([]uint16, 4096)
	procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))

	return syscall.UTF16ToString(pathBuf), nil
}

// PickHTML membuka dialog "Open" untuk memilih file .html / .htm.
func PickHTML(owner uintptr) (string, error) {
	return PickFile(owner, "Pilih file HTML lokal", htmlFilter)
}

// openFileNameW adalah cerminan struktur OPENFILENAMEW milik Windows.
type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

// PickIcon membuka dialog "Open" untuk memilih file icon .ico.
// Owner adalah HWND jendela pemanggil (boleh 0 bila tidak tersedia).
// String kosong dikembalikan tanpa error bila pengguna membatalkan dialog.
func PickIcon(owner uintptr) (string, error) {
	return PickFile(owner, "Pilih file icon (.ico)", iconFilter)
}

// toUTF16 mengubah string menjadi UTF-16 + terminator NUL.
// Tidak memakai syscall.UTF16FromString karena fungsi itu menolak string yang
// memuat NUL, padahal filter dialog Windows wajib memakai pemisah NUL.
func toUTF16(s string) []uint16 {
	return append(utf16.Encode([]rune(s)), 0)
}

// safeOwner hanya mengembalikan owner bila HWND-nya dimiliki thread pemanggil.
//
// Dialog modal dengan owner lintas-thread akan menggantung: proses pembuatan
// dialog menunggu thread pemilik jendela, sementara thread itu menunggu handler
// selesai. Ini terbukti pada Miniblink 49, yang menjalankan handler IPC di
// thread job terpisah dari thread jendela builder.
func safeOwner(owner uintptr) uintptr {
	if owner == 0 {
		return 0
	}

	var pid uint32
	ownerThread, _, _ := procGetWindowThreadPID.Call(owner, uintptr(unsafe.Pointer(&pid)))
	currentThread, _, _ := procGetCurrentThreadID.Call()
	if ownerThread != currentThread {
		return 0
	}

	return owner
}

// PickFile membuka dialog "Open" native Windows dan mengembalikan path terpilih.
func PickFile(owner uintptr, title, filter string) (string, error) {
	if err := procGetOpenFileNameW.Find(); err != nil {
		return "", errors.New("dialog file Windows tidak tersedia: " + err.Error())
	}

	filterUTF16 := toUTF16(filter)
	titleUTF16 := toUTF16(title)

	buf := make([]uint16, 4096)

	ofn := openFileNameW{
		hwndOwner:    safeOwner(owner),
		lpstrFilter:  &filterUTF16[0],
		nFilterIndex: 1,
		lpstrFile:    &buf[0],
		nMaxFile:     uint32(len(buf)),
		lpstrTitle:   &titleUTF16[0],
		flags:        OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))

	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		// CommDlgExtendedError() == 0 berarti pengguna menekan Cancel.
		// Kode selain itu menandakan kegagalan nyata (mis. COM belum diinisialisasi).
		if extErr, _, _ := procCommDlgExtendedError.Call(); extErr != 0 {
			return "", fmt.Errorf("dialog file gagal dibuka (CommDlgExtendedError=0x%X)", extErr)
		}
		return "", nil
	}

	return syscall.UTF16ToString(buf), nil
}
