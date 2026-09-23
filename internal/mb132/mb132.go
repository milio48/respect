package mb132

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/klauspost/compress/zstd"
	"respect-app/assets"
)

var (
	dllOnce sync.Once
	dllErr  error
	dllMod  *syscall.DLL

	procMbInit                           *syscall.Proc
	procMbUninit                         *syscall.Proc
	procMbCreateWebWindow                *syscall.Proc
	procMbDestroyWebView                 *syscall.Proc
	procMbSetWindowTitle                 *syscall.Proc
	procMbMoveToCenter                   *syscall.Proc
	procMbMoveWindow                     *syscall.Proc
	procMbShowWindow                     *syscall.Proc
	procMbLoadURL                        *syscall.Proc
	procMbLoadHtmlWithBaseUrl            *syscall.Proc
	procMbOnJsQuery                      *syscall.Proc
	procMbResponseQuery                  *syscall.Proc
	procMbOnClose                        *syscall.Proc
	procMbOnDestroy                      *syscall.Proc
	procMbGetHostHWND                    *syscall.Proc
	procMbRunMessageLoop                 *syscall.Proc
	procMbExitMessageLoop                *syscall.Proc
	procMbSetLanguage                    *syscall.Proc
	procMbSetUserAgent                   *syscall.Proc
	procMbSetCookieJarFullPath           *syscall.Proc
	procMbSetCookieJarPath               *syscall.Proc
	procMbSetLocalStoragePath            *syscall.Proc
	procMbOnCreateView                   *syscall.Proc
	procMbOnNavigation                   *syscall.Proc
	procMbSetNavigationToNewWindowEnable *syscall.Proc
	procMbOnLoadUrlBegin                 *syscall.Proc
	procMbNetSetHTTPHeaderFieldUtf8      *syscall.Proc
	procMbOnDidCreateScriptContext       *syscall.Proc
	procMbRunJs                          *syscall.Proc
	procMbEnableHighDPISupport           *syscall.Proc
	procMbOnAlertBox                     *syscall.Proc
	procMbOnConfirmBox                   *syscall.Proc
	procMbOnPromptBox                    *syscall.Proc
	procMbOnTitleChanged                 *syscall.Proc
	procMbOnDownload                     *syscall.Proc
	procMbPopupDownloadMgr               *syscall.Proc

	// Windows User32 & Kernel32 untuk window, dialog, icon, dan proses
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procSendMessageW     = user32.NewProc("SendMessageW")
	procLoadImageW       = user32.NewProc("LoadImageW")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procSetWindowTextW   = user32.NewProc("SetWindowTextW")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

const (
	WM_SETICON = 0x0080
	ICON_SMALL = 0
	ICON_BIG   = 1

	IMAGE_ICON      = 1
	LR_LOADFROMFILE = 0x0010

	// Modern Chrome User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"
	DefaultLanguage  = "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7"
)

// Struktur Exception Handling Windows x64 untuk mencegah crash assertion Miniblink 132
type EXCEPTION_RECORD struct {
	ExceptionCode        uint32
	ExceptionFlags       uint32
	ExceptionRecord      uintptr
	ExceptionAddress     uintptr
	NumberParameters     uint32
	ExceptionInformation [15]uintptr
}

type CONTEXT_AMD64 struct {
	P1Home, P2Home, P3Home, P4Home, P5Home, P6Home uint64
	ContextFlags                                   uint32
	MxCsr                                          uint32
	SegCs, SegDs, SegEs, SegFs, SegGs, SegSs       uint16
	EFlags                                         uint32
	Dr0, Dr1, Dr2, Dr3, Dr6, Dr7                   uint64
	Rax, Rcx, Rdx, Rbx, Rsp, Rbp, Rsi, Rdi         uint64
	R8, R9, R10, R11, R12, R13, R14, R15           uint64
	Rip                                            uint64
}

type EXCEPTION_POINTERS struct {
	ExceptionRecord *EXCEPTION_RECORD
	ContextRecord   *CONTEXT_AMD64
}

var vehOnce sync.Once

// installVehHandler memasang Windows Vectored Exception Handler tingkat kernel
// untuk mencegat STATUS_BREAKPOINT (0x80000003 / int 3) dari Chromium assertions
// sehingga browser tidak force close saat membuka situs-situs berat.
func installVehHandler() {
	vehOnce.Do(func() {
		procAddVeh := kernel32.NewProc("AddVectoredExceptionHandler")

		handler := syscall.NewCallback(func(ep *EXCEPTION_POINTERS) uintptr {
			if ep == nil || ep.ExceptionRecord == nil {
				return 0
			}
			code := ep.ExceptionRecord.ExceptionCode
			if code == 0x40010006 || code == 0xE06D7363 {
				return 0 // OutputDebugString atau C++ exception, biarkan runtime bawaan
			}
			if code == 0x80000003 { // STATUS_BREAKPOINT (DCHECK assertion internal Miniblink/Chromium)
				if ep.ContextRecord != nil {
					ep.ContextRecord.Rip++ // Lewati opcode int 3 (1 byte 0xCC)
				}
				return ^uintptr(0) // EXCEPTION_CONTINUE_EXECUTION (-1)
			}
			return 0 // EXCEPTION_CONTINUE_SEARCH
		})

		procAddVeh.Call(1, handler)
	})
}

// cleanLocalArtifacts menghapus cookies.dat yang sempat tertulis di folder executable/cwd
func cleanLocalArtifacts() {
	self, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(self)
		_ = os.Remove(filepath.Join(dir, "cookies.dat"))
		_ = os.Remove(filepath.Join(dir, "cookies.dat-journal"))
	}
	cwd, err := os.Getwd()
	if err == nil {
		_ = os.Remove(filepath.Join(cwd, "cookies.dat"))
		_ = os.Remove(filepath.Join(cwd, "cookies.dat-journal"))
	}
}

// spawnDetachedCleanup menjalankan proses cmd independen di background
// dengan delay kecil untuk membersihkan cookies.dat jika libcurl menuliskannya di detik akhir proses
func spawnDetachedCleanup() {
	self, err := os.Executable()
	if err != nil {
		return
	}
	cookieFile := filepath.Join(filepath.Dir(self), "cookies.dat")

	cmdStr := fmt.Sprintf("ping 127.0.0.1 -n 1 >nul & del /f /q \"%s\" >nul 2>&1", cookieFile)
	cmd := exec.Command("cmd.exe", "/C", cmdStr)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	_ = cmd.Start()
}

// ptrToUtf8 membaca string UTF-8 yang ditunjuk oleh pointer memori C
func ptrToUtf8(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	p := (*byte)(unsafe.Pointer(ptr))
	slice := unsafe.Slice(p, 10*1024*1024)
	var n int
	for n = 0; n < len(slice); n++ {
		if slice[n] == 0 {
			break
		}
	}
	return string(slice[:n])
}

// GetAppSandboxDir mengembalikan path folder penyimpanan sandbox unik untuk aplikasi ini
// di %LOCALAPPDATA%\respect\apps\<appName>\ sehingga sesi antar aplikasi tidak saling bentrok.
func GetAppSandboxDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	appName := "default"
	self, err := os.Executable()
	if err == nil {
		name := strings.TrimSuffix(filepath.Base(self), filepath.Ext(self))
		if name != "" {
			appName = name
		}
	}
	// Normalisasi nama folder
	appName = strings.ToLower(strings.TrimSpace(appName))
	var clean strings.Builder
	for _, r := range appName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			clean.WriteRune(r)
		} else {
			clean.WriteRune('_')
		}
	}
	appName = clean.String()
	if appName == "" {
		appName = "default"
	}

	appDir := filepath.Join(base, "respect", "apps", appName)
	_ = os.MkdirAll(appDir, 0755)
	return appDir
}

// findDLL mencari path DLL Miniblink 132 (blink.dll atau mb132_x64.dll).
// Jika tidak ditemukan di folder lokal (mode slim dev), maka akan mengekstrak dari embedded assets (mode release).
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
			if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
				return p, nil
			}
		}
	}

	// Jika mode release (single file .exe mandiri dengan engine tertanam)
	if len(assets.BlinkDLLZst) > 0 {
		return ensureExtractedDLL()
	}

	return "", errors.New("file Miniblink 132 DLL (blink.dll atau mb132_x64.dll) tidak ditemukan di folder aplikasi")
}

const expectedBlinkDLLSize = 68962816

// ensureExtractedDLL mendekompresi dan mengekstrak embedded blink.dll.zst ke cache lokal sistem (%LocalAppData%\respect\engine)
func ensureExtractedDLL() (string, error) {
	baseDir := os.Getenv("LOCALAPPDATA")
	if baseDir == "" {
		baseDir = filepath.Join(os.TempDir(), "respect")
	}
	engineDir := filepath.Join(baseDir, "respect", "engine")
	targetPath := filepath.Join(engineDir, "blink.dll")

	// Jika file cache sudah ada dengan ukuran sama persis (68.96 MB), gunakan langsung (start instan 0 ms)
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() == expectedBlinkDLLSize {
		return targetPath, nil
	}

	if err := os.MkdirAll(engineDir, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori cache engine (%s): %w", engineDir, err)
	}

	zr, err := zstd.NewReader(bytes.NewReader(assets.BlinkDLLZst))
	if err != nil {
		return "", fmt.Errorf("gagal inisialisasi dekompresi zstd: %w", err)
	}
	defer zr.Close()

	tmpPath := targetPath + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		tmpFile, err = os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			if fi, sErr := os.Stat(targetPath); sErr == nil && fi.Size() == expectedBlinkDLLSize {
				return targetPath, nil
			}
			return "", fmt.Errorf("gagal membuka target DLL (%s): %w", targetPath, err)
		}
		if _, err := io.Copy(tmpFile, zr); err != nil {
			_ = tmpFile.Close()
			return "", fmt.Errorf("gagal mendekompresi blink.dll: %w", err)
		}
		_ = tmpFile.Close()
		return targetPath, nil
	}

	if _, err := io.Copy(tmpFile, zr); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("gagal mendekompresi blink.dll: %w", err)
	}
	_ = tmpFile.Close()

	_ = os.Remove(targetPath)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		if fi, sErr := os.Stat(targetPath); sErr == nil && fi.Size() == expectedBlinkDLLSize {
			_ = os.Remove(tmpPath)
			return targetPath, nil
		}
		return tmpPath, nil
	}

	return targetPath, nil
}

// Init memuat DLL dan menginisialisasi engine Miniblink 132
func Init() error {
	dllOnce.Do(func() {
		// Bersihkan sisa cookies lama sebelum engine diinisialisasi
		cleanLocalArtifacts()

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

		// Pasang proteksi crash assertion Windows VEH
		installVehHandler()

		procMbInit, dllErr = dllMod.FindProc("mbInit")
		if dllErr != nil {
			return
		}
		procMbUninit, _ = dllMod.FindProc("mbUninit")
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

		// Prosedur pendukung & callback (opsional agar tidak menggugurkan Init jika ada variasi build DLL)
		procMbSetWindowTitle, _ = dllMod.FindProc("mbSetWindowTitle")
		procMbMoveToCenter, _ = dllMod.FindProc("mbMoveToCenter")
		procMbMoveWindow, _ = dllMod.FindProc("mbMoveWindow")
		procMbOnJsQuery, _ = dllMod.FindProc("mbOnJsQuery")
		procMbResponseQuery, _ = dllMod.FindProc("mbResponseQuery")
		procMbOnClose, _ = dllMod.FindProc("mbOnClose")
		procMbOnDestroy, _ = dllMod.FindProc("mbOnDestroy")
		procMbSetLanguage, _ = dllMod.FindProc("mbSetLanguage")
		procMbSetUserAgent, _ = dllMod.FindProc("mbSetUserAgent")
		procMbSetCookieJarFullPath, _ = dllMod.FindProc("mbSetCookieJarFullPath")
		procMbSetCookieJarPath, _ = dllMod.FindProc("mbSetCookieJarPath")
		procMbSetLocalStoragePath, _ = dllMod.FindProc("mbSetLocalStorageFullPath")
		procMbOnCreateView, _ = dllMod.FindProc("mbOnCreateView")
		procMbOnNavigation, _ = dllMod.FindProc("mbOnNavigation")
		procMbSetNavigationToNewWindowEnable, _ = dllMod.FindProc("mbSetNavigationToNewWindowEnable")
		procMbOnLoadUrlBegin, _ = dllMod.FindProc("mbOnLoadUrlBegin")
		procMbNetSetHTTPHeaderFieldUtf8, _ = dllMod.FindProc("mbNetSetHTTPHeaderFieldUtf8")
		procMbOnDidCreateScriptContext, _ = dllMod.FindProc("mbOnDidCreateScriptContext")
		procMbRunJs, _ = dllMod.FindProc("mbRunJs")
		procMbEnableHighDPISupport, _ = dllMod.FindProc("mbEnableHighDPISupport")
		procMbOnAlertBox, _ = dllMod.FindProc("mbOnAlertBox")
		procMbOnConfirmBox, _ = dllMod.FindProc("mbOnConfirmBox")
		procMbOnPromptBox, _ = dllMod.FindProc("mbOnPromptBox")
		procMbOnTitleChanged, _ = dllMod.FindProc("mbOnTitleChanged")
		procMbOnDownload, _ = dllMod.FindProc("mbOnDownload")
		procMbPopupDownloadMgr, _ = dllMod.FindProc("mbPopupDownloadMgr")

		// Panggil mbInit terlebih dahulu agar Chromium AtExitManager terinisialisasi
		procMbInit.Call(0)
	})

	return dllErr
}

// WebView merepresentasikan satu jendela browser Miniblink 132
type WebView struct {
	Handle uintptr
	hwnd   uintptr
	title  string

	onCloseCb     uintptr
	onDestroyCb   uintptr
	onDestroyUser func()

	onQueryCb   uintptr
	onQueryUser func(req string) string

	onLoadBeginCb uintptr
	onScriptCtxCb uintptr

	onAlertBoxCb     uintptr
	onConfirmBoxCb   uintptr
	onPromptBoxCb    uintptr
	onTitleChangedCb uintptr
	onCreateViewCb   uintptr
	onDownloadCb     uintptr
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
		title:  title,
	}

	// 1. Set User Agent & Bahasa default
	wv.SetUserAgent(DefaultUserAgent)
	wv.SetLanguage(DefaultLanguage)

	// 2. Isolasi penyimpanan (Cookie & LocalStorage) ke folder AppData sandbox unik per nama aplikasi
	wv.SetIsolatedStorage()

	// 3. Tangani pembuatan popup/tab baru (target="_blank" dan window.open)
	// Alihkan navigasi langsung ke jendela aktif agar alur OAuth/popup login berjalan mulus tanpa silent drop
	if procMbOnCreateView != nil {
		wv.onCreateViewCb = syscall.NewCallback(func(h, param, navType, urlPtr, winFeatures uintptr) uintptr {
			var targetURL string
			if urlPtr != 0 {
				targetURL = ptrToUtf8(urlPtr)
			}
			if targetURL != "" && targetURL != "about:blank" {
				wv.LoadURL(targetURL)
			}
			return 0
		})
		procMbOnCreateView.Call(wv.Handle, wv.onCreateViewCb, 0)
	}

	// 4. Handler dialog interaktif JavaScript (Alert, Confirm, Prompt)
	if procMbOnAlertBox != nil {
		wv.onAlertBoxCb = syscall.NewCallback(func(h, param, msgPtr uintptr) uintptr {
			var msg string
			if msgPtr != 0 {
				msg = ptrToUtf8(msgPtr)
			}
			msgW, _ := syscall.UTF16PtrFromString(msg)
			t := wv.title
			if t == "" {
				t = "Alert"
			}
			titleW, _ := syscall.UTF16PtrFromString(t)
			procMessageBoxW.Call(wv.hwnd, uintptr(unsafe.Pointer(msgW)), uintptr(unsafe.Pointer(titleW)), 0x40) // MB_OK | MB_ICONINFORMATION
			return 0
		})
		procMbOnAlertBox.Call(wv.Handle, wv.onAlertBoxCb, 0)
	}

	if procMbOnConfirmBox != nil {
		wv.onConfirmBoxCb = syscall.NewCallback(func(h, param, msgPtr uintptr) uintptr {
			var msg string
			if msgPtr != 0 {
				msg = ptrToUtf8(msgPtr)
			}
			msgW, _ := syscall.UTF16PtrFromString(msg)
			t := wv.title
			if t == "" {
				t = "Confirm"
			}
			titleW, _ := syscall.UTF16PtrFromString(t)
			ret, _, _ := procMessageBoxW.Call(wv.hwnd, uintptr(unsafe.Pointer(msgW)), uintptr(unsafe.Pointer(titleW)), 0x01|0x20) // MB_OKCANCEL | MB_ICONQUESTION
			if ret == 1 {                                                                                                         // IDOK
				return 1
			}
			return 0
		})
		procMbOnConfirmBox.Call(wv.Handle, wv.onConfirmBoxCb, 0)
	}

	if procMbOnPromptBox != nil {
		wv.onPromptBoxCb = syscall.NewCallback(func(h, param, msgPtr, defPtr, resPtr uintptr) uintptr {
			if resPtr != 0 {
				*(*int32)(unsafe.Pointer(resPtr)) = 0 // batalkan prompt dengan aman, kembalikan null ke JS tanpa hang
			}
			return 0
		})
		procMbOnPromptBox.Call(wv.Handle, wv.onPromptBoxCb, 0)
	}

	// 5. Handler Title Changed: perbarui judul window native saat <title> halaman web berubah
	if procMbOnTitleChanged != nil {
		wv.onTitleChangedCb = syscall.NewCallback(func(h, param, titlePtr uintptr) uintptr {
			if titlePtr != 0 {
				newTitle := ptrToUtf8(titlePtr)
				if newTitle != "" {
					wv.title = newTitle
					tW, err := syscall.UTF16PtrFromString(newTitle)
					if err == nil && wv.hwnd != 0 {
						procSetWindowTextW.Call(wv.hwnd, uintptr(unsafe.Pointer(tW)))
					}
				}
			}
			return 0
		})
		procMbOnTitleChanged.Call(wv.Handle, wv.onTitleChangedCb, 0)
	}

	// 6. Handler Download: buka dialog unduhan Save As standar Windows secara otomatis
	if procMbOnDownload != nil {
		wv.onDownloadCb = syscall.NewCallback(func(h, param, frameId, urlPtr, downloadJob uintptr) uintptr {
			if procMbPopupDownloadMgr != nil {
				r, _, _ := procMbPopupDownloadMgr.Call(h, urlPtr, downloadJob)
				return r
			}
			return 0
		})
		procMbOnDownload.Call(wv.Handle, wv.onDownloadCb, 0)
	}

	// 7. Injeksi HTTP Header (Accept-Language dan User-Agent) ke setiap request jaringan
	// Menjamin Google dan website lain tidak pernah mendeteksi bahasa Cina
	if procMbOnLoadUrlBegin != nil && procMbNetSetHTTPHeaderFieldUtf8 != nil {
		langKey := []byte("Accept-Language\x00")
		langVal := []byte(DefaultLanguage + "\x00")
		uaKey := []byte("User-Agent\x00")
		uaVal := []byte(DefaultUserAgent + "\x00")

		wv.onLoadBeginCb = syscall.NewCallback(func(h, param, urlPtr, jobPtr uintptr) uintptr {
			procMbNetSetHTTPHeaderFieldUtf8.Call(jobPtr, uintptr(unsafe.Pointer(&langKey[0])), uintptr(unsafe.Pointer(&langVal[0])), 0)
			procMbNetSetHTTPHeaderFieldUtf8.Call(jobPtr, uintptr(unsafe.Pointer(&uaKey[0])), uintptr(unsafe.Pointer(&uaVal[0])), 0)
			return 0 // 0 = lanjutkan request secara normal
		})
		procMbOnLoadUrlBegin.Call(wv.Handle, wv.onLoadBeginCb, 0)
	}

	// 8. Injeksi skrip context V8: override navigator.language dan terapkan akselerasi mouse wheel yang natural
	if procMbOnDidCreateScriptContext != nil && procMbRunJs != nil {
		preloadScript := []byte(`
try {
    Object.defineProperty(navigator, 'language', { get: () => 'id-ID', configurable: true });
    Object.defineProperty(navigator, 'languages', { get: () => ['id-ID', 'id', 'en-US', 'en'], configurable: true });
} catch(e) {}
try {
    if (!window.__respect_wheel_hooked) {
        window.__respect_wheel_hooked = true;
        function getScrollParent(el) {
            if (!el || el === document) return document.scrollingElement || document.documentElement;
            const isScroll = (node) => {
                if (!node || node.nodeType !== 1) return false;
                const s = window.getComputedStyle(node);
                return (s.overflowY === 'auto' || s.overflowY === 'scroll') && node.scrollHeight > node.clientHeight;
            };
            while (el && el !== document.body && el !== document.documentElement) {
                if (isScroll(el)) return el;
                el = el.parentElement;
            }
            return document.scrollingElement || document.documentElement;
        }
        window.addEventListener('wheel', function(e) {
            if (e.ctrlKey) return;
            const scroller = getScrollParent(e.target);
            if (!scroller) return;
            // Dampen delta agar pergerakan scroll terasa natural dan responsif (55% dari delta mentah Windows)
            scroller.scrollTop += e.deltaY * 0.55;
            if (e.deltaX) scroller.scrollLeft += e.deltaX * 0.55;
            e.preventDefault();
        }, { passive: false });
    }
} catch(e) {}
` + "\x00")

		wv.onScriptCtxCb = syscall.NewCallback(func(h, param, frameId, ctx, extGroup, worldId uintptr) uintptr {
			procMbRunJs.Call(h, frameId, uintptr(unsafe.Pointer(&preloadScript[0])), 0, 0, 0, 0)
			return 0
		})
		procMbOnDidCreateScriptContext.Call(wv.Handle, wv.onScriptCtxCb, 0)
	}

	if title != "" {
		wv.SetTitle(title)
	}
	wv.MoveToCenter()

	// 9. Pasang handler OnClose agar saat tombol X titlebar diklik,
	// message loop segera dihentikan sehingga proses langsung keluar dari Task Manager
	if procMbOnClose != nil {
		wv.onCloseCb = syscall.NewCallback(func(h, param, unuse uintptr) uintptr {
			if wv.onDestroyUser != nil {
				wv.onDestroyUser()
			}
			ExitMessageLoop()
			return 0
		})
		procMbOnClose.Call(wv.Handle, wv.onCloseCb, 0)
	}

	// Pasang callback OnDestroy standar
	wv.onDestroyCb = syscall.NewCallback(func(v, p1, p2 uintptr) uintptr {
		if wv.onDestroyUser != nil {
			wv.onDestroyUser()
		}
		ExitMessageLoop()
		return 0
	})
	procMbOnDestroy.Call(wv.Handle, wv.onDestroyCb, 0)

	return wv, nil
}

// SetTitle mengatur judul jendela
func (v *WebView) SetTitle(title string) {
	v.title = title
	if procMbSetWindowTitle != nil {
		utf8Str := []byte(title + "\x00")
		procMbSetWindowTitle.Call(v.Handle, uintptr(unsafe.Pointer(&utf8Str[0])))
	}
	if v.hwnd != 0 {
		tW, err := syscall.UTF16PtrFromString(title)
		if err == nil {
			procSetWindowTextW.Call(v.hwnd, uintptr(unsafe.Pointer(tW)))
		}
	}
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

// SetIsolatedStorage memindahkan penyimpanan Cookie dan LocalStorage ke sandbox aplikasi di LocalAppData
// agar folder instalasi aplikasi tetap bersih dan sesi antar aplikasi Respect tidak saling bertabrakan.
func (v *WebView) SetIsolatedStorage() {
	appDir := GetAppSandboxDir()
	cacheDir := filepath.Join(appDir, "cache")
	storageDir := filepath.Join(appDir, "storage")
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.MkdirAll(storageDir, 0755)

	cookieFile, err1 := syscall.UTF16PtrFromString(filepath.Join(cacheDir, "cookies.dat"))
	storagePath, err2 := syscall.UTF16PtrFromString(storageDir)

	if err1 == nil && procMbSetCookieJarFullPath != nil {
		procMbSetCookieJarFullPath.Call(v.Handle, uintptr(unsafe.Pointer(cookieFile)))
	}
	if err2 == nil && procMbSetCookieJarPath != nil {
		procMbSetCookieJarPath.Call(v.Handle, uintptr(unsafe.Pointer(storagePath)))
	}
	if err2 == nil && procMbSetLocalStoragePath != nil {
		procMbSetLocalStoragePath.Call(v.Handle, uintptr(unsafe.Pointer(storagePath)))
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

// HostHWND mengembalikan handle jendela host (HWND) milik webview.
// Dipakai sebagai owner dialog native agar dialog tetap modal terhadap builder.
func (v *WebView) HostHWND() uintptr {
	return v.hwnd
}

// HandleQuery mendaftarkan fungsi Go untuk merespons query JavaScript (window.mbQuery)
func (v *WebView) HandleQuery(handler func(req string) string) {
	v.onQueryUser = handler

	v.onQueryCb = syscall.NewCallback(func(handle uintptr, param uintptr, es uintptr, queryId int64, customMsg int32, reqPtr *byte) uintptr {
		reqStr := ptrToUtf8(uintptr(unsafe.Pointer(reqPtr)))

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

// SetIcon memasang icon jendela dari byte data .ico atau resource PE bawaan
func (v *WebView) SetIcon(iconBytes []byte) error {
	if v.hwnd == 0 {
		return nil
	}

	var hIcon uintptr

	// 1. Coba muat dari file icon temporer jika iconBytes disediakan
	if len(iconBytes) > 0 {
		tempIco := filepath.Join(os.TempDir(), "respect_win_icon.ico")
		if err := os.WriteFile(tempIco, iconBytes, 0644); err == nil {
			icoPathW, errW := syscall.UTF16PtrFromString(tempIco)
			if errW == nil {
				hIcon, _, _ = procLoadImageW.Call(
					0,
					uintptr(unsafe.Pointer(icoPathW)),
					IMAGE_ICON,
					0, 0,
					LR_LOADFROMFILE,
				)
			}
		}
	}

	// 2. Jika gagal atau tidak ada bytes, muat langsung dari PE Resource file EXE saat ini (Resource ID 1)
	if hIcon == 0 {
		hModule, _, _ := procGetModuleHandleW.Call(0)
		if hModule != 0 {
			hIcon, _, _ = procLoadIconW.Call(hModule, uintptr(1))
		}
	}

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

	// Setelah message loop berhenti, uninitialization dan bersihkan cookies
	if procMbUninit != nil {
		procMbUninit.Call()
	}
	cleanLocalArtifacts()
	spawnDetachedCleanup()
	os.Exit(0)
}

// ExitMessageLoop menghentikan message loop
func ExitMessageLoop() {
	if procMbExitMessageLoop != nil {
		procMbExitMessageLoop.Call()
	}
	cleanLocalArtifacts()
	spawnDetachedCleanup()
}
