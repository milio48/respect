package winprint

import (
	"errors"
	"math"
	"syscall"
	"unsafe"
)

var (
	comdlg32           = syscall.NewLazyDLL("comdlg32.dll")
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	user32             = syscall.NewLazyDLL("user32.dll")

	procPrintDlgW      = comdlg32.NewProc("PrintDlgW")
	procGetDeviceCaps  = gdi32.NewProc("GetDeviceCaps")
	procDeleteDC       = gdi32.NewProc("DeleteDC")
	procStartDocW      = gdi32.NewProc("StartDocW")
	procEndDoc         = gdi32.NewProc("EndDoc")
	procStartPage      = gdi32.NewProc("StartPage")
	procEndPage        = gdi32.NewProc("EndPage")
	procSetStretchMode = gdi32.NewProc("SetStretchBltMode")
	procStretchBlt     = gdi32.NewProc("StretchBlt")
	procCreateCompDC   = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompBm   = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject   = gdi32.NewProc("SelectObject")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procGetDC          = user32.NewProc("GetDC")
	procReleaseDC      = user32.NewProc("ReleaseDC")
	procGetClientRect  = user32.NewProc("GetClientRect")
	procPrintWindow    = user32.NewProc("PrintWindow")
)

type PRINTDLGW struct {
	lStructSize         uint32
	hwndOwner           uintptr
	hDevMode            uintptr
	hDevNames           uintptr
	hDC                 uintptr
	Flags               uint32
	nFromPage           uint16
	nToPage             uint16
	nMinPage            uint16
	nMaxPage            uint16
	nCopies             uint16
	hInstance           uintptr
	lCustData           uintptr
	lpfnPrintHook       uintptr
	lpfnSetupHook       uintptr
	lpPrintTemplateName uintptr
	lpSetupTemplateName uintptr
	hPrintTemplate      uintptr
	hSetupTemplate      uintptr
}

type DOCINFOW struct {
	cbSize      int32
	lpszDocName *uint16
	lpszOutput  *uint16
	lpszDatType *uint16
	fwType      uint32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

const (
	PD_RETURNDC       = 0x00000100
	PD_RETURNDEFAULT  = 0x00000400
	PD_NOSELECTION    = 0x00000004
	PD_NOPAGENUMS     = 0x00000008

	HORZRES     = 8
	VERTRES     = 10
	LOGPIXELSX  = 88
	LOGPIXELSY  = 90
	HALFTONE    = 4
	SRCCOPY     = 0x00CC0020
)

// PrintRect merepresentasikan koordinat dan ukuran cetak yang telah diskalakan.
type PrintRect struct {
	X, Y, W, H int
}

// CalculatePrintRect menghitung dimensi dan posisi cetak agar proporsional di tengah halaman dengan margin.
func CalculatePrintRect(printerW, printerH, dpiX, dpiY, srcW, srcH int) PrintRect {
	if printerW <= 0 || printerH <= 0 || srcW <= 0 || srcH <= 0 {
		return PrintRect{W: srcW, H: srcH}
	}

	// Margin 10mm (sekitar 0.394 inci)
	marginX := int(float64(dpiX) * 0.394)
	marginY := int(float64(dpiY) * 0.394)

	availW := printerW - 2*marginX
	availH := printerH - 2*marginY
	if availW <= 0 {
		availW = printerW
		marginX = 0
	}
	if availH <= 0 {
		availH = printerH
		marginY = 0
	}

	scaleX := float64(availW) / float64(srcW)
	scaleY := float64(availH) / float64(srcH)
	scale := math.Min(scaleX, scaleY)

	destW := int(float64(srcW) * scale)
	destH := int(float64(srcH) * scale)
	destX := marginX + (availW-destW)/2
	destY := marginY

	return PrintRect{
		X: destX,
		Y: destY,
		W: destW,
		H: destH,
	}
}

// HasPrinters memeriksa apakah sistem memiliki printer yang terpasang / aktif.
func HasPrinters() bool {
	var pd PRINTDLGW
	pd.lStructSize = uint32(unsafe.Sizeof(pd))
	pd.Flags = PD_RETURNDC | PD_RETURNDEFAULT

	r, _, _ := procPrintDlgW.Call(uintptr(unsafe.Pointer(&pd)))
	if r != 0 && pd.hDC != 0 {
		procDeleteDC.Call(pd.hDC)
		return true
	}
	return false
}

// PrintWindowContent membuka dialog printer native Windows dan mencetak konten jendela hwnd.
// Mengembalikan (true, nil) jika berhasil dicetak, (false, nil) jika user membatalkan (Cancel),
// atau (false, err) jika terjadi kesalahan teknis.
func PrintWindowContent(hwnd uintptr, title string) (bool, error) {
	if hwnd == 0 {
		return false, errors.New("invalid window handle (HWND 0)")
	}

	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	srcW := int(rc.Right - rc.Left)
	srcH := int(rc.Bottom - rc.Top)
	if srcW <= 0 || srcH <= 0 {
		return false, errors.New("ukuran area jendela tidak valid")
	}

	// 1. Tampilkan Windows Native Print Dialog (modal terhadap jendela aplikasi)
	var pd PRINTDLGW
	pd.lStructSize = uint32(unsafe.Sizeof(pd))
	pd.hwndOwner = hwnd
	pd.Flags = PD_RETURNDC | PD_NOSELECTION | PD_NOPAGENUMS

	r1, _, _ := procPrintDlgW.Call(uintptr(unsafe.Pointer(&pd)))
	if r1 == 0 || pd.hDC == 0 {
		// User membatalkan dialog (Cancel)
		return false, nil
	}
	defer procDeleteDC.Call(pd.hDC)

	// 2. Tangkap tampilan jendela ke memori bitmap (GDI Compatible Bitmap)
	hdcScreen, _, _ := procGetDC.Call(hwnd)
	if hdcScreen == 0 {
		return false, errors.New("gagal mendapatkan DC layar")
	}
	defer procReleaseDC.Call(hwnd, hdcScreen)

	hdcMem, _, _ := procCreateCompDC.Call(hdcScreen)
	if hdcMem == 0 {
		return false, errors.New("gagal membuat compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hbm, _, _ := procCreateCompBm.Call(hdcScreen, uintptr(srcW), uintptr(srcH))
	if hbm == 0 {
		return false, errors.New("gagal membuat compatible bitmap")
	}
	defer procDeleteObject.Call(hbm)

	oldBm, _, _ := procSelectObject.Call(hdcMem, hbm)
	defer procSelectObject.Call(hdcMem, oldBm)

	// PW_RENDERFULLCONTENT = 2
	pwRet, _, _ := procPrintWindow.Call(hwnd, hdcMem, 2)
	if pwRet == 0 {
		// Fallback PW_CLIENTONLY = 1
		procPrintWindow.Call(hwnd, hdcMem, 1)
	}

	// 3. Baca resolusi kertas printer terpilih
	pw, _, _ := procGetDeviceCaps.Call(pd.hDC, HORZRES)
	ph, _, _ := procGetDeviceCaps.Call(pd.hDC, VERTRES)
	dpiX, _, _ := procGetDeviceCaps.Call(pd.hDC, LOGPIXELSX)
	dpiY, _, _ := procGetDeviceCaps.Call(pd.hDC, LOGPIXELSY)

	rect := CalculatePrintRect(int(pw), int(ph), int(dpiX), int(dpiY), srcW, srcH)

	// 4. Kirim job cetak ke printer DC dengan interpolasi kualitas tinggi HALFTONE
	if title == "" {
		title = "Respect Desktop Document"
	}
	docNamePtr, _ := syscall.UTF16PtrFromString(title)

	var di DOCINFOW
	di.cbSize = int32(unsafe.Sizeof(di))
	di.lpszDocName = docNamePtr

	jobId, _, err := procStartDocW.Call(pd.hDC, uintptr(unsafe.Pointer(&di)))
	if int32(jobId) <= 0 {
		return false, err
	}

	procStartPage.Call(pd.hDC)
	procSetStretchMode.Call(pd.hDC, HALFTONE)
	procStretchBlt.Call(
		pd.hDC,
		uintptr(rect.X),
		uintptr(rect.Y),
		uintptr(rect.W),
		uintptr(rect.H),
		hdcMem,
		0,
		0,
		uintptr(srcW),
		uintptr(srcH),
		SRCCOPY,
	)
	procEndPage.Call(pd.hDC)
	procEndDoc.Call(pd.hDC)

	return true, nil
}
