package mb132

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

var (
	dllOnce sync.Once
	dllErr  error
	dllMod  *syscall.DLL

	procMbInit                 *syscall.Proc
	procMbCreateWebWindow      *syscall.Proc
	procMbDestroyWebView       *syscall.Proc
	procMbSetWindowTitle       *syscall.Proc
	procMbMoveToCenter         *syscall.Proc
	procMbMoveWindow           *syscall.Proc
	procMbShowWindow           *syscall.Proc
	procMbLoadURL              *syscall.Proc
	procMbLoadHtmlWithBaseUrl  *syscall.Proc
	procMbOnJsQuery            *syscall.Proc
	procMbResponseQuery        *syscall.Proc
	procMbOnDestroy            *syscall.Proc
	procMbGetHostHWND          *syscall.Proc
	procMbRunMessageLoop       *syscall.Proc
	procMbExitMessageLoop      *syscall.Proc
	procMbSetLanguage          *syscall.Proc
	procMbSetUserAgent         *syscall.Proc
	procMbSetCookieJarFullPath *syscall.Proc
	procMbSetCookieJarPath     *syscall.Proc
	procMbSetLocalStoragePath  *syscall.Proc
	procMbOnCreateView         *syscall.Proc
	procMbOnNavigation         *syscall.Proc


	// Windows User32 untuk pengaturan icon jendela native
	user32                = syscall.NewLazyDLL("user32.dll")
	procSendMessageW      = user32.NewProc("SendMessageW")
	procCreateIconFromRes = user32.NewProc("CreateIconFromResourceEx")
)

const (
	WM_SETICON = 0x0080
	ICON_SMALL = 0
	ICON_BIG   = 1

	// Modern Chrome User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"
	DefaultLanguage  = "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7"
)

// findDLL mencari path DLL Miniblink 132 (blink.dll atau mb132_x64.dll)
func findDLL() (string, error) {
	self, err := os.Executable()
	var searchDirs []string
	if err == nil {
		searchDirs = append(searchDirs, filepath.Dir(self))
	}
	cwd, err := os.Getwd()
	if err == nil {
		searchDirs = append(searchDirs, cwd)
	}

	dllNames := []string{"blink.dll", "mb132_x64.dll"}
	for _, dir := range searchDirs {
		for _, name := range dllNames {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", errors.New("file Miniblink 132 DLL (blink.dll atau mb132_x64.dll) tidak ditemukan di folder aplikasi")
}

// Init memuat DLL dan menginisialisasi engine Miniblink 132
func Init() error {
	dllOnce.Do(func() {
		dllPath, err := findDLL()
		if err != nil {
			dllErr = err
			return
		}

		dllMod, err = syscall.LoadDLL(dllPath)
		if err != nil {
			dllErr = fmt.Errorf("gagal memuat DLL (%s): %w", dllPath, err)
			return
		}

		procMbInit, dllErr = dllMod.FindProc("mbInit")
		if dllErr != nil {
			return
		}
		procMbCreateWebWindow, dllErr = dllMod.FindProc("mbCreateWebWindow")
		if dllErr != nil {
			return
		}
		procMbDestroyWebView, dllErr = dllMod.FindProc("mbDestroyWebView")
		if dllErr != nil {
			return
		}
		procMbSetWindowTitle, dllErr = dllMod.FindProc("mbSetWindowTitle")
		if dllErr != nil {
			return
		}
		procMbMoveToCenter, dllErr = dllMod.FindProc("mbMoveToCenter")
		if dllErr != nil {
			return
		}
		procMbMoveWindow, dllErr = dllMod.FindProc("mbMoveWindow")
		if dllErr != nil {
			return
		}
		procMbShowWindow, dllErr = dllMod.FindProc("mbShowWindow")
		if dllErr != nil {
			return
		}
		procMbLoadURL, dllErr = dllMod.FindProc("mbLoadURL")
		if dllErr != nil {
			return
		}
		procMbLoadHtmlWithBaseUrl, dllErr = dllMod.FindProc("mbLoadHtmlWithBaseUrl")
		if dllErr != nil {
			return
		}
		procMbOnJsQuery, dllErr = dllMod.FindProc("mbOnJsQuery")
		if dllErr != nil {
			return
		}
		procMbResponseQuery, dllErr = dllMod.FindProc("mbResponseQuery")
		if dllErr != nil {
			return
		}
		procMbOnDestroy, dllErr = dllMod.FindProc("mbOnDestroy")
		if dllErr != nil {
			return
		}
		procMbGetHostHWND, dllErr = dllMod.FindProc("mbGetHostHWND")
		if dllErr != nil {
			return
		}
		procMbRunMessageLoop, dllErr = dllMod.FindProc("mbRunMessageLoop")
		if dllErr != nil {
			return
		}
		procMbExitMessageLoop, dllErr = dllMod.FindProc("mbExitMessageLoop")
		if dllErr != nil {
			return
		}
		procMbSetLanguage, dllErr = dllMod.FindProc("mbSetLanguage")
		if dllErr != nil {
			return
		}
		procMbSetUserAgent, dllErr = dllMod.FindProc("mbSetUserAgent")
		if dllErr != nil {
			return
		}
		procMbSetCookieJarFullPath, dllErr = dllMod.FindProc("mbSetCookieJarFullPath")
		if dllErr != nil {
			return
		}
		procMbSetCookieJarPath, _ = dllMod.FindProc("mbSetCookieJarPath")
		procMbSetLocalStoragePath, dllErr = dllMod.FindProc("mbSetLocalStorageFullPath")
		if dllErr != nil {
			return
		}
		procMbOnCreateView, dllErr = dllMod.FindProc("mbOnCreateView")
		if dllErr != nil {
			return
		}
		procMbOnNavigation, dllErr = dllMod.FindProc("mbOnNavigation")
		if dllErr != nil {
			return
		}

		// Panggil mbInit dengan settings default NULL
		procMbInit.Call(0)

		// Bersihkan file cookies.dat lokal jika ada sisa eksekusi sebelumnya
		self, err := os.Executable()
		if err == nil {
			_ = os.Remove(filepath.Join(filepath.Dir(self), "cookies.dat"))
		}
	})

	return dllErr
}

// WebView merepresentasikan satu jendela browser Miniblink 132
type WebView struct {
	Handle uintptr
	hwnd   uintptr

	onDestroyCb    uintptr
	onDestroyUser  func()

	onQueryCb      uintptr
	onQueryUser    func(req string) string

	onCreateViewCb uintptr
}

// CreateWebWindow membuat jendela webview popup baru
func CreateWebWindow(title string, width, height int) (*WebView, error) {
	if err := Init(); err != nil {
		return nil, err
	}

	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	// mbCreateWebWindow: type=0 (MB_WINDOW_TYPE_POPUP), parent=0, x=0, y=0, w, h
	handle, _, err := procMbCreateWebWindow.Call(0, 0, 0, 0, uintptr(width), uintptr(height))
	if handle == 0 {
		return nil, fmt.Errorf("mbCreateWebWindow gagal: %w", err)
	}

	hwnd, _, _ := procMbGetHostHWND.Call(handle)

	wv := &WebView{
		Handle: handle,
		hwnd:   hwnd,
	}

	// 1. Set User Agent & Bahasa default (menghindari bahasa Cina & tampilan jadul)
	wv.SetUserAgent(DefaultUserAgent)
	wv.SetLanguage(DefaultLanguage)

	// 2. Isolasi penyimpanan (Cookie & LocalStorage) ke folder AppData/Temp agar folder aplikasi tetap bersih
	wv.SetIsolatedStorage()

	// 3. Pasang handler pembukaan jendela baru (target="_blank" / popup) agar tidak force close
	wv.onCreateViewCb = syscall.NewCallback(func(h uintptr, param uintptr, navType int32, urlPtr *byte, feat uintptr) uintptr {
		if urlPtr != nil {
			slice := unsafe.Slice(urlPtr, 2048)
			var n int
			for n = 0; n < len(slice) && slice[n] != 0; n++ {
			}
			newURL := string(slice[:n])
			if newURL != "" {
				wv.LoadURL(newURL)
			}
		}
		return 0
	})
	procMbOnCreateView.Call(wv.Handle, wv.onCreateViewCb, 0)

	if title != "" {
		wv.SetTitle(title)
	}
	wv.MoveToCenter()

	// Pasang callback OnDestroy standar
	wv.onDestroyCb = syscall.NewCallback(func(v, p1, p2 uintptr) uintptr {
		if wv.onDestroyUser != nil {
			wv.onDestroyUser()
		}
		ExitMessageLoop()

		// Bersihkan file cookies.dat lokal jika tertulis di folder executable
		self, err := os.Executable()
		if err == nil {
			_ = os.Remove(filepath.Join(filepath.Dir(self), "cookies.dat"))
		}
		return 0
	})
	procMbOnDestroy.Call(wv.Handle, wv.onDestroyCb, 0)

	return wv, nil
}

// SetTitle mengatur judul jendela
func (v *WebView) SetTitle(title string) {
	utf8Str := []byte(title + "\x00")
	procMbSetWindowTitle.Call(v.Handle, uintptr(unsafe.Pointer(&utf8Str[0])))
}

// SetUserAgent mengatur string User-Agent
func (v *WebView) SetUserAgent(ua string) {
	if procMbSetUserAgent != nil {
		utf8Str := []byte(ua + "\x00")
		procMbSetUserAgent.Call(v.Handle, uintptr(unsafe.Pointer(&utf8Str[0])))
	}
}

// SetLanguage mengatur locale dan bahasa accept HTTP
func (v *WebView) SetLanguage(lang string) {
	if procMbSetLanguage != nil {
		utf8Str := []byte(lang + "\x00")
		procMbSetLanguage.Call(v.Handle, uintptr(unsafe.Pointer(&utf8Str[0])))
	}
}

// SetIsolatedStorage memindahkan penyimpanan Cookie dan LocalStorage ke LocalAppData
// agar folder instalasi aplikasi tetap bersih tanpa file cookie/LocalStorage liar
func (v *WebView) SetIsolatedStorage() {
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	cacheDir := filepath.Join(appData, "respect", "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cookieFile, err1 := syscall.UTF16PtrFromString(filepath.Join(cacheDir, "cookies.dat"))
	storageDir, err2 := syscall.UTF16PtrFromString(cacheDir)

	if err1 == nil && procMbSetCookieJarFullPath != nil {
		procMbSetCookieJarFullPath.Call(v.Handle, uintptr(unsafe.Pointer(cookieFile)))
	}
	if err2 == nil && procMbSetCookieJarPath != nil {
		procMbSetCookieJarPath.Call(v.Handle, uintptr(unsafe.Pointer(storageDir)))
	}
	if err2 == nil && procMbSetLocalStoragePath != nil {
		procMbSetLocalStoragePath.Call(v.Handle, uintptr(unsafe.Pointer(storageDir)))
	}
}

// MoveToCenter memposisikan jendela di tengah layar
func (v *WebView) MoveToCenter() {
	procMbMoveToCenter.Call(v.Handle)
}

// Show menampilkan jendela
func (v *WebView) Show() {
	procMbShowWindow.Call(v.Handle, 1)
}

// LoadURL memuat halaman web berdasarkan URL
func (v *WebView) LoadURL(url string) {
	utf8Str := []byte(url + "\x00")
	procMbLoadURL.Call(v.Handle, uintptr(unsafe.Pointer(&utf8Str[0])))
}

// LoadHTML memuat string HTML langsung dengan baseURL
func (v *WebView) LoadHTML(html string, baseURL string) {
	if baseURL == "" {
		baseURL = "http://localhost/"
	}
	htmlBytes := []byte(html + "\x00")
	baseBytes := []byte(baseURL + "\x00")
	procMbLoadHtmlWithBaseUrl.Call(v.Handle, uintptr(unsafe.Pointer(&htmlBytes[0])), uintptr(unsafe.Pointer(&baseBytes[0])))
}

// OnDestroy mendaftarkan fungsi yang dipanggil saat jendela ditutup
func (v *WebView) OnDestroy(cb func()) {
	v.onDestroyUser = cb
}

// HandleQuery mendaftarkan fungsi Go untuk merespons query JavaScript (window.mbQuery)
func (v *WebView) HandleQuery(handler func(req string) string) {
	v.onQueryUser = handler

	v.onQueryCb = syscall.NewCallback(func(handle uintptr, param uintptr, es uintptr, queryId int64, customMsg int32, reqPtr *byte) uintptr {
		var reqStr string
		if reqPtr != nil {
			slice := unsafe.Slice(reqPtr, 10*1024*1024)
			var n int
			for n = 0; n < len(slice); n++ {
				if slice[n] == 0 {
					break
				}
			}
			reqStr = string(slice[:n])
		}

		var respStr string
		if v.onQueryUser != nil {
			respStr = v.onQueryUser(reqStr)
		}

		respBytes := []byte(respStr + "\x00")
		procMbResponseQuery.Call(handle, uintptr(queryId), uintptr(customMsg), uintptr(unsafe.Pointer(&respBytes[0])))
		return 0
	})

	procMbOnJsQuery.Call(v.Handle, v.onQueryCb, 0)
}

// SetIcon memasang icon jendela dari byte data .ico
func (v *WebView) SetIcon(iconBytes []byte) error {
	if v.hwnd == 0 || len(iconBytes) == 0 {
		return nil
	}

	if len(iconBytes) < 22 {
		return nil
	}

	hIcon, _, _ := procCreateIconFromRes.Call(
		uintptr(unsafe.Pointer(&iconBytes[22])),
		uintptr(len(iconBytes)-22),
		1, // isIcon = TRUE
		0x00030000,
		0, 0,
		0,
	)

	if hIcon != 0 {
		procSendMessageW.Call(v.hwnd, WM_SETICON, ICON_BIG, hIcon)
		procSendMessageW.Call(v.hwnd, WM_SETICON, ICON_SMALL, hIcon)
	}

	return nil
}

// RunMessageLoop menjalankan Windows message loop
func RunMessageLoop() {
	if procMbRunMessageLoop != nil {
		procMbRunMessageLoop.Call()
	}
}

// ExitMessageLoop menghentikan message loop
func ExitMessageLoop() {
	if procMbExitMessageLoop != nil {
		procMbExitMessageLoop.Call()
	}
}
