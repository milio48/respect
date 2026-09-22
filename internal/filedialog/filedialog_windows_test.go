//go:build windows

package filedialog

import (
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

var (
	procCreateWindowExW = user32.NewProc("CreateWindowExW")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
)

// newTestWindow membuat jendela sederhana pada thread pemanggil.
// Mengembalikan 0 bila jendela tidak bisa dibuat.
func newTestWindow() uintptr {
	class, err := syscall.UTF16PtrFromString("STATIC")
	if err != nil {
		return 0
	}
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(class)),
		0, 0, 0, 0, 10, 10, 0, 0, 0, 0,
	)
	return hwnd
}

// TestSafeOwnerThreadSama memastikan owner milik thread sendiri tetap dipakai
// supaya dialog tetap modal terhadap jendela builder.
func TestSafeOwnerThreadSama(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd := newTestWindow()
	if hwnd == 0 {
		t.Skip("tidak bisa membuat jendela uji di lingkungan ini")
	}
	defer procDestroyWindow.Call(hwnd)

	if got := safeOwner(hwnd); got != hwnd {
		t.Errorf("safeOwner(jendela thread sendiri) = %d, mau %d", got, hwnd)
	}
}

// TestSafeOwnerLintasThread adalah regresi bug dialog lite: owner dari thread
// lain harus diabaikan, kalau tidak dialog tidak akan pernah muncul dan
// pemanggil menggantung.
func TestSafeOwnerLintasThread(t *testing.T) {
	hwndCh := make(chan uintptr, 1)
	hold := make(chan struct{})

	go func() {
		runtime.LockOSThread()
		hwnd := newTestWindow()
		hwndCh <- hwnd
		<-hold
		if hwnd != 0 {
			procDestroyWindow.Call(hwnd)
		}
	}()

	hwnd := <-hwndCh
	if hwnd == 0 {
		close(hold)
		t.Skip("tidak bisa membuat jendela uji di lingkungan ini")
	}

	if got := safeOwner(hwnd); got != 0 {
		t.Errorf("safeOwner(jendela thread lain) = %d, mau 0", got)
	}

	close(hold)
}

// TestSafeOwnerNol memastikan tidak ada owner berarti tidak ada masalah.
func TestSafeOwnerNol(t *testing.T) {
	if got := safeOwner(0); got != 0 {
		t.Errorf("safeOwner(0) = %d, mau 0", got)
	}
}

// TestToUTF16MengizinkanPemisahNUL memastikan konversi tidak menolak NUL.
// Regresi: syscall.UTF16FromString mengembalikan EINVAL untuk string ber-NUL,
// sehingga filter dialog gagal sebelum dialog sempat dibuka.
func TestToUTF16MengizinkanPemisahNUL(t *testing.T) {
	got := toUTF16("a\x00b")
	want := []uint16{'a', 0, 'b', 0}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("toUTF16 = %v, mau %v", got, want)
	}
}

// TestIconFilterFormat memastikan filter dialog berbentuk pasangan
// deskripsi/pola dan diakhiri pemisah ganda sesuai aturan OPENFILENAMEW.
func TestIconFilterFormat(t *testing.T) {
	parts := strings.Split(iconFilter, "\x00")

	if len(parts) < 4 {
		t.Fatalf("filter terlalu pendek: %q", iconFilter)
	}
	if parts[len(parts)-1] != "" || parts[len(parts)-2] != "" {
		t.Fatalf("filter harus diakhiri dua NUL: %q", iconFilter)
	}

	tokens := parts[:len(parts)-2]
	if len(tokens)%2 != 0 {
		t.Fatalf("jumlah token deskripsi/pola harus genap: %v", tokens)
	}
	if tokens[0] != "File Icon (*.ico)" || tokens[1] != "*.ico" {
		t.Fatalf("pasangan filter pertama tidak sesuai: %v", tokens[:2])
	}
}

// TestLayoutOpenFileNameW memastikan struktur hasil tulis-tangan cocok dengan
// OPENFILENAMEW milik Windows x64 (sizeof = 152 byte). Bila layout meleset,
// GetOpenFileNameW akan langsung gagal tanpa menampilkan dialog.
func TestLayoutOpenFileNameW(t *testing.T) {
	var ofn openFileNameW

	if got, want := unsafe.Sizeof(ofn), uintptr(152); got != want {
		t.Fatalf("sizeof(openFileNameW) = %d, mau %d", got, want)
	}

	offsets := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"lStructSize", unsafe.Offsetof(ofn.lStructSize), 0},
		{"hwndOwner", unsafe.Offsetof(ofn.hwndOwner), 8},
		{"hInstance", unsafe.Offsetof(ofn.hInstance), 16},
		{"lpstrFilter", unsafe.Offsetof(ofn.lpstrFilter), 24},
		{"nFilterIndex", unsafe.Offsetof(ofn.nFilterIndex), 44},
		{"lpstrFile", unsafe.Offsetof(ofn.lpstrFile), 48},
		{"nMaxFile", unsafe.Offsetof(ofn.nMaxFile), 56},
		{"lpstrFileTitle", unsafe.Offsetof(ofn.lpstrFileTitle), 64},
		{"nMaxFileTitle", unsafe.Offsetof(ofn.nMaxFileTitle), 72},
		{"lpstrInitialDir", unsafe.Offsetof(ofn.lpstrInitialDir), 80},
		{"lpstrTitle", unsafe.Offsetof(ofn.lpstrTitle), 88},
		{"flags", unsafe.Offsetof(ofn.flags), 96},
		{"nFileOffset", unsafe.Offsetof(ofn.nFileOffset), 100},
		{"nFileExtension", unsafe.Offsetof(ofn.nFileExtension), 102},
		{"lpstrDefExt", unsafe.Offsetof(ofn.lpstrDefExt), 104},
		{"lCustData", unsafe.Offsetof(ofn.lCustData), 112},
		{"lpfnHook", unsafe.Offsetof(ofn.lpfnHook), 120},
		{"lpTemplateName", unsafe.Offsetof(ofn.lpTemplateName), 128},
		{"pvReserved", unsafe.Offsetof(ofn.pvReserved), 136},
		{"dwReserved", unsafe.Offsetof(ofn.dwReserved), 144},
		{"flagsEx", unsafe.Offsetof(ofn.flagsEx), 148},
	}

	for _, o := range offsets {
		if o.got != o.want {
			t.Errorf("offset %s = %d, mau %d", o.name, o.got, o.want)
		}
	}
}
