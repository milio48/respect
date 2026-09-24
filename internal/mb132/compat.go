package mb132

// compat.go — Lapisan "akali" keterbatasan engine Miniblink 132.
//
// Strategi: engine tidak menyediakan banyak Web API modern (clipboard, Notification,
// speechSynthesis, fullscreen, dll). Daripada memodifikasi DLL, kita:
//   1. Menyuntikkan JS shim ke setiap script context (window.mbQuery bridge).
//   2. Menjalankan fungsi asli di sisi Go memakai Win32 / API Windows.
//
// Bridge protocol (via window.mbQuery):
//   JS  : mbQuery(1, '{"cmd":"clipboard.read","payload":null}', function(customMsg, response){})
//   Go  : mengembalikan JSON {"ok":true,"data":...} atau {"ok":false,"error":"..."}

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// compatEnabled diaktifkan runtime sebelum CreateWebWindow.
var compatEnabled bool

// EnableCompatShims mengaktifkan JS shim + bridge native.
func EnableCompatShims() { compatEnabled = true }

// -----------------------------------------------------------------------------
// Ring buffer console native (mbOnConsole)
// -----------------------------------------------------------------------------

var (
	consoleMu  sync.Mutex
	consoleBuf []string
)

const consoleBufMax = 500

func recordConsole(level string, msg string, source string, line uintptr) {
	consoleMu.Lock()
	defer consoleMu.Unlock()
	entry := fmt.Sprintf("[%s] %s (%s:%d)", level, msg, source, line)
	consoleBuf = append(consoleBuf, entry)
	if len(consoleBuf) > consoleBufMax {
		consoleBuf = consoleBuf[len(consoleBuf)-consoleBufMax:]
	}
}

func dumpConsole() []string {
	consoleMu.Lock()
	defer consoleMu.Unlock()
	out := make([]string, len(consoleBuf))
	copy(out, consoleBuf)
	return out
}

func consoleLevelName(level uintptr) string {
	switch level {
	case 0:
		return "log"
	case 1:
		return "warn"
	case 2:
		return "error"
	case 3:
		return "debug"
	default:
		return fmt.Sprintf("level%d", level)
	}
}

// installConsoleCapture memasang mbOnConsole agar seluruh pesan console.* dari
// engine ikut terekam di sisi Go (bisa diambil lewat bridge "console.dump").
func (wv *WebView) installConsoleCapture() {
	if procMbOnConsole == nil {
		return
	}
	wv.onConsoleCb = syscall.NewCallback(func(h, param, level, msgPtr, srcPtr, line, stackPtr uintptr) uintptr {
		msg := ptrToUtf8(msgPtr)
		src := ptrToUtf8(srcPtr)
		recordConsole(consoleLevelName(level), msg, src, line)
		return 0
	})
	procMbOnConsole.Call(wv.Handle, wv.onConsoleCb, 0)
}

// -----------------------------------------------------------------------------
// Win32 clipboard
// -----------------------------------------------------------------------------

var (
	procOpenClipboard      = user32.NewProc("OpenClipboard")
	procCloseClipboard     = user32.NewProc("CloseClipboard")
	procEmptyClipboard     = user32.NewProc("EmptyClipboard")
	procSetClipboardData   = user32.NewProc("SetClipboardData")
	procGetClipboardData   = user32.NewProc("GetClipboardData")
	procGlobalAlloc        = kernel32.NewProc("GlobalAlloc")
	procGlobalLock         = kernel32.NewProc("GlobalLock")
	procGlobalUnlock       = kernel32.NewProc("GlobalUnlock")
	procMessageBoxTimeoutW = user32.NewProc("MessageBoxTimeoutW")
	procGetLastInputInfo   = user32.NewProc("GetLastInputInfo")
	procGetTickCount       = kernel32.NewProc("GetTickCount")
	procGetWindowLongPtrW  = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW  = user32.NewProc("SetWindowLongPtrW")
	procGetMonitorInfoW    = user32.NewProc("GetMonitorInfoW")
	procMonitorFromWindow  = user32.NewProc("MonitorFromWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	shell32                = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW      = shell32.NewProc("ShellExecuteW")
)

const (
	cfUnicodeText  = 13
	gmemMoveable   = 0x0002
	wsCaption      = 0x00C00000
	wsThickFrame   = 0x00040000
	monitorDefault = 0x00000002
	swpNoZOrder    = 0x0004
	swpFrameChange = 0x0020
	swpShowWindow  = 0x0040
)

// GWL_STYLE = -16 (direpresentasikan sebagai uintptr two's complement).
var gwlStyle = ^uintptr(15)

type rect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

type lastInputInfo struct {
	CbSize uint32
	DwTime uint32
}

func clipboardRead() string {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer procCloseClipboard.Call()
	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return ""
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)
	// Baca sampai NUL; batasi 8 MB agar aman.
	buf := (*[1 << 22]uint16)(unsafe.Pointer(p))
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(buf[:n])
}

func clipboardWrite(text string) error {
	u16, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	size := uintptr(len(u16) * 2)
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("GlobalAlloc gagal")
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return fmt.Errorf("GlobalLock gagal")
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u16))
	copy(dst, u16)
	procGlobalUnlock.Call(h)

	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return fmt.Errorf("OpenClipboard gagal")
	}
	procEmptyClipboard.Call()
	if sr, _, _ := procSetClipboardData.Call(cfUnicodeText, h); sr == 0 {
		procCloseClipboard.Call()
		return fmt.Errorf("SetClipboardData gagal")
	}
	procCloseClipboard.Call()
	return nil
}

// -----------------------------------------------------------------------------
// Fullscreen (borderless monitor fill)
// -----------------------------------------------------------------------------

func (wv *WebView) setFullscreen(on bool) bool {
	if wv.hwnd == 0 {
		return false
	}
	if on {
		if wv.fsActive {
			return true
		}
		var wr rect
		procGetWindowRect.Call(wv.hwnd, uintptr(unsafe.Pointer(&wr)))
		style, _, _ := procGetWindowLongPtrW.Call(wv.hwnd, gwlStyle)
		wv.fsStyle = style
		wv.fsRect = wr

		hm, _, _ := procMonitorFromWindow.Call(wv.hwnd, monitorDefault)
		if hm == 0 {
			return false
		}
		var mi monitorInfo
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		mr, _, _ := procGetMonitorInfoW.Call(hm, uintptr(unsafe.Pointer(&mi)))
		if mr == 0 {
			return false
		}
		newStyle := style &^ uintptr(wsCaption|wsThickFrame)
		procSetWindowLongPtrW.Call(wv.hwnd, gwlStyle, newStyle)
		procSetWindowPos.Call(
			wv.hwnd, 0,
			uintptr(mi.RcMonitor.Left), uintptr(mi.RcMonitor.Top),
			uintptr(mi.RcMonitor.Right-mi.RcMonitor.Left), uintptr(mi.RcMonitor.Bottom-mi.RcMonitor.Top),
			swpNoZOrder|swpFrameChange|swpShowWindow,
		)
		wv.fsActive = true
		return true
	}

	if !wv.fsActive {
		return true
	}
	procSetWindowLongPtrW.Call(wv.hwnd, gwlStyle, wv.fsStyle)
	procSetWindowPos.Call(
		wv.hwnd, 0,
		uintptr(wv.fsRect.Left), uintptr(wv.fsRect.Top),
		uintptr(wv.fsRect.Right-wv.fsRect.Left), uintptr(wv.fsRect.Bottom-wv.fsRect.Top),
		swpNoZOrder|swpFrameChange|swpShowWindow,
	)
	wv.fsActive = false
	return true
}

// -----------------------------------------------------------------------------
// Notification (native message box dengan auto-timeout)
// -----------------------------------------------------------------------------

func (wv *WebView) notify(title, body string) {
	if title == "" {
		title = wv.title
	}
	if title == "" {
		title = "Notification"
	}
	bodyW, err1 := syscall.UTF16PtrFromString(body)
	titleW, err2 := syscall.UTF16PtrFromString(title)
	if err1 != nil || err2 != nil {
		return
	}
	hwnd := wv.hwnd
	go func() {
		if procMessageBoxTimeoutW.Find() == nil {
			// MB_OK | MB_ICONINFORMATION | MB_TOPMOST, timeout 5 detik
			procMessageBoxTimeoutW.Call(hwnd, uintptr(unsafe.Pointer(bodyW)), uintptr(unsafe.Pointer(titleW)), 0x40|0x40000, 0, 5000)
			return
		}
		procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(bodyW)), uintptr(unsafe.Pointer(titleW)), 0x40|0x40000)
	}()
}

// -----------------------------------------------------------------------------
// Text-to-Speech (Windows SAPI via PowerShell)
// -----------------------------------------------------------------------------

var (
	speakMu  sync.Mutex
	speakCmd *exec.Cmd
)

func speakText(text string, rate, pitch, volume float64) error {
	speakMu.Lock()
	if speakCmd != nil && speakCmd.Process != nil {
		_ = speakCmd.Process.Kill()
	}
	safe := strings.ReplaceAll(text, "'", "''")
	rateInt := int((rate - 1.0) * 5.0)
	if rateInt < -10 {
		rateInt = -10
	}
	if rateInt > 10 {
		rateInt = 10
	}
	volInt := int(volume * 100)
	if volInt < 0 {
		volInt = 0
	}
	if volInt > 100 {
		volInt = 100
	}
	_ = pitch // SAPI SpVoice tidak memakai pitch langsung; dipertahankan untuk kompatibilitas API.
	ps := fmt.Sprintf(
		"Add-Type -AssemblyName System.Speech; $s=New-Object System.Speech.Synthesis.SpeechSynthesizer; $s.Rate=%d; $s.Volume=%d; $s.Speak('%s')",
		rateInt, volInt, safe,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	err := cmd.Start()
	if err != nil {
		speakMu.Unlock()
		return err
	}
	speakCmd = cmd
	speakMu.Unlock()
	go func() { _ = cmd.Wait() }()
	return nil
}

func stopSpeak() {
	speakMu.Lock()
	defer speakMu.Unlock()
	if speakCmd != nil && speakCmd.Process != nil {
		_ = speakCmd.Process.Kill()
	}
	speakCmd = nil
}

// -----------------------------------------------------------------------------
// Utilities: idle time, open external, storage estimate
// -----------------------------------------------------------------------------

func idleTimeMs() uint32 {
	var lii lastInputInfo
	lii.CbSize = uint32(unsafe.Sizeof(lii))
	if r, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii))); r == 0 {
		return 0
	}
	t, _, _ := procGetTickCount.Call()
	return uint32(t) - lii.DwTime
}

func openExternal(url string) error {
	op, _ := syscall.UTF16PtrFromString("open")
	urlW, err := syscall.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	r, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(op)), uintptr(unsafe.Pointer(urlW)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("ShellExecute gagal (kode %d)", r)
	}
	return nil
}

func storageEstimate() map[string]interface{} {
	dir := GetAppSandboxDir()
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return map[string]interface{}{
		"quota": int64(2) << 30, // 2 GB kuota virtual
		"usage": total,
	}
}

// -----------------------------------------------------------------------------
// Bridge dispatcher
// -----------------------------------------------------------------------------

type compatRequest struct {
	Cmd     string          `json:"cmd"`
	Payload json.RawMessage `json:"payload"`
}

type compatResponse struct {
	Ok    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func okJSON(data interface{}) string {
	b, _ := json.Marshal(compatResponse{Ok: true, Data: data})
	return string(b)
}

func errJSON(msg string) string {
	b, _ := json.Marshal(compatResponse{Ok: false, Error: msg})
	return string(b)
}

// installCompat memasang handler bridge (window.mbQuery) untuk aksi native.
func (wv *WebView) installCompat() {
	wv.HandleQuery(func(req string) string { return wv.handleCompatQuery(req) })
}

func (wv *WebView) handleCompatQuery(req string) string {
	var m compatRequest
	if err := json.Unmarshal([]byte(req), &m); err != nil {
		return errJSON("request tidak valid: " + err.Error())
	}

	switch m.Cmd {
	case "clipboard.read":
		return okJSON(map[string]string{"text": clipboardRead()})

	case "clipboard.write":
		var p struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		if err := clipboardWrite(p.Text); err != nil {
			return errJSON(err.Error())
		}
		return okJSON(true)

	case "fullscreen.set":
		var p struct {
			On bool `json:"on"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		return okJSON(wv.setFullscreen(p.On))

	case "notify":
		var p struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		wv.notify(p.Title, p.Body)
		return okJSON(true)

	case "speak":
		var p struct {
			Text   string  `json:"text"`
			Rate   float64 `json:"rate"`
			Pitch  float64 `json:"pitch"`
			Volume float64 `json:"volume"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		if p.Rate == 0 {
			p.Rate = 1
		}
		if p.Volume == 0 {
			p.Volume = 1
		}
		if err := speakText(p.Text, p.Rate, p.Pitch, p.Volume); err != nil {
			return errJSON(err.Error())
		}
		return okJSON(true)

	case "speak.cancel":
		stopSpeak()
		return okJSON(true)

	case "openExternal":
		var p struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		if err := openExternal(p.URL); err != nil {
			return errJSON(err.Error())
		}
		return okJSON(true)

	case "idle.time":
		return okJSON(map[string]uint32{"idleMs": idleTimeMs()})

	case "storage.estimate":
		return okJSON(storageEstimate())

	case "console.dump":
		return okJSON(dumpConsole())

	case "media.load":
		var p struct {
			Handle string `json:"handle"`
			Src    string `json:"src"`
			Video  bool   `json:"video"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		d, err := wv.mediaLoad(p.Handle, p.Src, p.Video)
		if err != nil {
			return errJSON(err.Error())
		}
		return okJSON(d)

	case "media.loadData":
		var p struct {
			Handle string `json:"handle"`
			Name   string `json:"name"`
			Mime   string `json:"mime"`
			Data   string `json:"data"`
			Video  bool   `json:"video"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		d, err := wv.mediaLoadData(p.Handle, p.Name, p.Mime, p.Data, p.Video)
		if err != nil {
			return errJSON(err.Error())
		}
		return okJSON(d)

	case "media.play":
		var p struct {
			Handle string `json:"handle"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		d, err := wv.mediaPlay(p.Handle)
		if err != nil {
			return errJSON(err.Error())
		}
		return okJSON(d)

	case "media.pause", "media.resume", "media.stop":
		var p struct {
			Handle string `json:"handle"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		action := strings.TrimPrefix(m.Cmd, "media.")
		d, err := wv.mediaSimple(p.Handle, action)
		if err != nil {
			return errJSON(err.Error())
		}
		return okJSON(d)

	case "media.seek":
		var p struct {
			Handle string `json:"handle"`
			Ms     int64  `json:"ms"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		if err := wv.mediaSeek(p.Handle, p.Ms); err != nil {
			return errJSON(err.Error())
		}
		return okJSON(true)

	case "media.volume":
		var p struct {
			Handle string  `json:"handle"`
			Volume float64 `json:"volume"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		if err := wv.mediaVolume(p.Handle, p.Volume); err != nil {
			return errJSON(err.Error())
		}
		return okJSON(true)

	case "media.status":
		var p struct {
			Handle string `json:"handle"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		d, err := wv.mediaStatus(p.Handle)
		if err != nil {
			return errJSON(err.Error())
		}
		return okJSON(d)

	case "media.rect":
		var p struct {
			Handle string `json:"handle"`
			X      int    `json:"x"`
			Y      int    `json:"y"`
			W      int    `json:"w"`
			H      int    `json:"h"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		if err := wv.mediaRect(p.Handle, p.X, p.Y, p.W, p.H); err != nil {
			return errJSON(err.Error())
		}
		return okJSON(true)

	case "media.dispose":
		var p struct {
			Handle string `json:"handle"`
		}
		_ = json.Unmarshal(m.Payload, &p)
		wv.mediaDispose(p.Handle)
		return okJSON(true)

	case "media.info":
		return okJSON(map[string]interface{}{
			"mci":   true,
			"vlc":   false,
			"audio": "MP3/WAV via MCI (codec Windows)",
			"video": "hanya format yang didukung codec Windows (AVI/WMV/MPG)",
			"hint":  "MP4/H.264 butuh dekoder open-source (ffmpeg); belum dibundel.",
		})

	default:
		return errJSON("perintah bridge tidak dikenal: " + m.Cmd)
	}
}

// compatPreloadJS mengembalikan skrip shim yang disuntikkan pada setiap script
// context. Shim ini mem-polyfill API yang absen dan meneruskan aksi ke Go via mbQuery.
func compatPreloadJS() string {
	return `
(function () {
  try {
    if (window.__respect_compat) return;
    window.__respect_compat = true;

    function invoke(cmd, payload) {
      return new Promise(function (resolve, reject) {
        if (typeof window.mbQuery !== 'function') { reject(new Error('respect bridge unavailable')); return; }
        var request = JSON.stringify({ cmd: cmd, payload: payload === undefined ? null : payload });
        var settled = false;
        try {
          window.mbQuery(1, request, function (customMsg, response) {
            if (settled) return; settled = true;
            try {
              var obj = JSON.parse(response);
              if (obj && obj.ok) resolve(obj.data);
              else reject(new Error((obj && obj.error) ? obj.error : 'bridge error'));
            } catch (e) { reject(new Error('bad bridge response')); }
          });
        } catch (e) { reject(e); }
        setTimeout(function () { if (!settled) { settled = true; reject(new Error('bridge timeout')); } }, 15000);
      });
    }
    window.respectInvoke = invoke;
    window.respectNativeConsole = function () { return invoke('console.dump'); };

    try { if (typeof navigator.vibrate !== 'function') navigator.vibrate = function () { return true; }; } catch (e) {}

    try {
      if (!navigator.clipboard || typeof navigator.clipboard.writeText !== 'function' || typeof navigator.clipboard.readText !== 'function') {
        var clip = (navigator.clipboard && typeof navigator.clipboard === 'object') ? navigator.clipboard : {};
        clip.writeText = function (t) { return invoke('clipboard.write', { text: String(t == null ? '' : t) }); };
        clip.readText = function () { return invoke('clipboard.read').then(function (d) { return (d && d.text) ? d.text : ''; }); };
        try { Object.defineProperty(navigator, 'clipboard', { get: function () { return clip; }, configurable: true }); }
        catch (e) { try { navigator.clipboard = clip; } catch (e2) {} }
      }
    } catch (e) {}

    try {
      if (!navigator.permissions) {
        Object.defineProperty(navigator, 'permissions', { configurable: true, value: { query: function (d) { return Promise.resolve({ state: 'granted', name: (d && d.name) || 'unknown', onchange: null }); } } });
      }
    } catch (e) {}

    try {
      if (typeof window.Notification === 'undefined') {
        var RNotify = function (title, opts) {
          this.title = title || '';
          this.body = (opts && opts.body) || '';
          this.onclick = null; this.onclose = null; this.onerror = null;
          invoke('notify', { title: String(this.title), body: String(this.body) });
        };
        RNotify.permission = 'granted';
        RNotify.requestPermission = function (cb) { if (cb) cb('granted'); return Promise.resolve('granted'); };
        RNotify.prototype.close = function () {};
        window.Notification = RNotify;
      } else if (Notification.permission === 'default') {
        Notification.requestPermission = function (cb) { if (cb) cb('granted'); return Promise.resolve('granted'); };
      }
    } catch (e) {}

    try {
      if (typeof navigator.share !== 'function') {
        navigator.share = function (data) {
          var text = (data && (data.text || data.title || data.url)) || JSON.stringify(data || {});
          return invoke('clipboard.write', { text: String(text) });
        };
      }
    } catch (e) {}

    try {
      if (typeof navigator.sendBeacon !== 'function') {
        navigator.sendBeacon = function (url, data) {
          try { fetch(url, { method: 'POST', body: data, keepalive: true }); return true; } catch (e) { return false; }
        };
      }
    } catch (e) {}

    try {
      var st = navigator.storage;
      if (!st) {
        st = {};
        try { Object.defineProperty(navigator, 'storage', { get: function () { return st; }, configurable: true }); }
        catch (e) { try { navigator.storage = st; } catch (e2) {} }
      }
      if (typeof st.estimate !== 'function') st.estimate = function () { return invoke('storage.estimate'); };
      if (typeof st.persist !== 'function') st.persist = function () { return Promise.resolve(true); };
      if (typeof st.persisted !== 'function') st.persisted = function () { return Promise.resolve(true); };
    } catch (e) {}

    try {
      if (typeof window.caches === 'undefined') {
        var _stores = {};
        function _Cache(name) { this._name = name; this._m = _stores[name] || (_stores[name] = {}); }
        _Cache.prototype.put = function (req, resp) { var k = (typeof req === 'string') ? req : (req && req.url) || String(req); this._m[k] = resp; return Promise.resolve(); };
        _Cache.prototype.match = function (req) { var k = (typeof req === 'string') ? req : (req && req.url) || String(req); return Promise.resolve(this._m[k]); };
        _Cache.prototype.matchAll = function () { var m = this._m; return Promise.resolve(Object.keys(m).map(function (k) { return m[k]; })); };
        _Cache.prototype['delete'] = function (req) { var k = (typeof req === 'string') ? req : (req && req.url) || String(req); delete this._m[k]; return Promise.resolve(true); };
        _Cache.prototype.keys = function () { var m = this._m; return Promise.resolve(Object.keys(m)); };
        var _caches = {
          open: function (name) { return Promise.resolve(new _Cache(name || 'default')); },
          has: function (name) { return Promise.resolve(!!_stores[name]); },
          'delete': function (name) { delete _stores[name]; return Promise.resolve(true); },
          keys: function () { return Promise.resolve(Object.keys(_stores)); },
          match: function (req) {
            var k = (typeof req === 'string') ? req : (req && req.url) || String(req);
            for (var n in _stores) { if (_stores[n][k]) return Promise.resolve(_stores[n][k]); }
            return Promise.resolve(undefined);
          }
        };
        try { Object.defineProperty(window, 'caches', { get: function () { return _caches; }, configurable: true }); }
        catch (e) { try { window.caches = _caches; } catch (e2) {} }
      }
    } catch (e) {}

    try {
      if (!('speechSynthesis' in window) || typeof window.speechSynthesis.speak !== 'function') {
        var synth = {
          speaking: false, pending: false, paused: false,
          speak: function (u) {
            synth.speaking = true;
            try { if (u.onstart) u.onstart({}); } catch (e) {}
            invoke('speak', { text: u.text || '', rate: u.rate || 1, pitch: u.pitch || 1, volume: (u.volume == null ? 1 : u.volume) })
              .then(function () { synth.speaking = false; try { if (u.onend) u.onend({}); } catch (e) {} },
                    function (err) { synth.speaking = false; try { if (u.onerror) u.onerror({ error: String((err && err.message) || err) }); } catch (e) {} });
          },
          cancel: function () { synth.speaking = false; invoke('speak.cancel', {}); },
          pause: function () { synth.paused = true; }, resume: function () { synth.paused = false; },
          getVoices: function () { return [{ name: 'Respect Native Voice', lang: 'id-ID', localService: true, default: true }]; }
        };
        try { Object.defineProperty(window, 'speechSynthesis', { get: function () { return synth; }, configurable: true }); }
        catch (e) { try { window.speechSynthesis = synth; } catch (e2) {} }
        if (typeof window.SpeechSynthesisUtterance === 'undefined') {
          window.SpeechSynthesisUtterance = function (text) { this.text = text || ''; this.rate = 1; this.pitch = 1; this.volume = 1; };
        }
      }
    } catch (e) {}

    try {
      var fsEl = null;
      function fireFs() {
        try { document.dispatchEvent(new Event('fullscreenchange')); } catch (e) {}
        try { document.dispatchEvent(new Event('webkitfullscreenchange')); } catch (e) {}
      }
      function reqFs() {
        var self = this;
        return invoke('fullscreen.set', { on: true }).then(function () { fsEl = self || document.documentElement; fireFs(); });
      }
      try { Element.prototype.requestFullscreen = reqFs; } catch (e) {}
      try { document.documentElement.requestFullscreen = function () { return reqFs.call(document.documentElement); }; } catch (e) {}
      try { document.exitFullscreen = function () { return invoke('fullscreen.set', { on: false }).then(function () { fsEl = null; fireFs(); }); }; } catch (e) {}
      try { Object.defineProperty(document, 'fullscreenElement', { get: function () { return fsEl; }, configurable: true }); } catch (e) {}
      try { Object.defineProperty(document, 'fullscreenEnabled', { get: function () { return true; }, configurable: true }); } catch (e) {}
    } catch (e) {}

    try {
      if (typeof window.IdleDetector === 'undefined') {
        window.IdleDetector = function () { this.userState = 'active'; this.screenState = 'unlocked'; };
        window.IdleDetector.requestPermission = function () { return Promise.resolve('granted'); };
        window.IdleDetector.prototype.start = function () { return invoke('idle.time'); };
      }
    } catch (e) {}

    try {
      if (typeof navigator.getGamepads !== 'function') {
        var _gp = [null, null, null, null];
        function _refreshGamepads() {
          invoke('gamepad.poll').then(function (list) {
            if (!list) return;
            for (var i = 0; i < list.length && i < 4; i++) {
              var c = list[i];
              if (c && c.connected) {
                var prev = _gp[i];
                _gp[i] = { index: i, id: c.id, connected: true, mapping: 'standard', axes: c.axes, buttons: c.buttons, timestamp: Date.now() };
                if (!prev) { try { window.dispatchEvent(new Event('gamepadconnected')); } catch (e) {} }
              } else if (_gp[i]) {
                _gp[i] = null;
                try { window.dispatchEvent(new Event('gamepaddisconnected')); } catch (e) {}
              }
            }
          }).catch(function () {});
        }
        setInterval(_refreshGamepads, 120);
        _refreshGamepads();
        navigator.getGamepads = function () { return _gp; };
      }
    } catch (e) {}

    try {
      if (typeof HTMLMediaElement !== 'undefined' && typeof window.mbQuery === 'function') {
        // ---- Pemutar media native (WinMM MCI) via bridge ----
        var _respectMediaCount = 0;
        function _mHandle(el) { if (!el.__respect_handle) { _respectMediaCount++; el.__respect_handle = 'm' + _respectMediaCount; } return el.__respect_handle; }
        function _mFire(el, t) { try { var ev = new Event(t); ev.__respectSynthetic = true; el.dispatchEvent(ev); } catch (e) {} }
        function _mState(el) {
          if (!el.__respect_state) { el.__respect_state = { playing: false, ended: false, pos: 0, dur: 0, timer: null, vol: 1, ready: 0 }; }
          return el.__respect_state;
        }
        function _mStopPoll(el) { var s = el.__respect_state; if (s && s.timer) { clearInterval(s.timer); s.timer = null; } }
        function _mSendRect(el) {
          if (!el.__respect_handle || String(el.tagName).toUpperCase() !== 'VIDEO') return;
          try {
            var r = el.getBoundingClientRect();
            if (r.width < 1 || r.height < 1) return;
            invoke('media.rect', { handle: el.__respect_handle, x: Math.round(r.left), y: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height) });
          } catch (e) {}
        }
        function _mStartPoll(el) {
          _mStopPoll(el);
          var s = _mState(el);
          s.timer = setInterval(function () {
            if (!el.__respect_handle) return;
            _mSendRect(el);
            invoke('media.status', { handle: el.__respect_handle }).then(function (st) {
              if (!st) return;
              s.pos = st.position || 0; s.dur = st.duration || s.dur;
              _mFire(el, 'timeupdate');
              if (st.mode === 'stopped' && s.dur > 0 && s.pos >= s.dur - 250) {
                if (!s.ended) { s.ended = true; s.playing = false; _mStopPoll(el); _mFire(el, 'ended'); }
              } else if (st.mode === 'paused') {
                s.playing = false;
              }
            }).catch(function () {});
          }, 300);
        }
        function _mSrc(el) {
          if (el.__respect_src) return el.__respect_src;
          try { var a = el.getAttribute && el.getAttribute('src'); if (a) return a; } catch (e) {}
          try {
            var s = el.getElementsByTagName('source');
            if (s && s.length) return s[0].__respect_src || s[0].getAttribute('src') || '';
          } catch (e) {}
          return '';
        }

        // Netralkan src native (HTML-parsed) agar engine TIDAK mencoba memuat media
        // dan memicu event 'error' native palsu yang menutupi error asli.
        function _mNeutralize(el) {
          try {
            if (!el || !el.tagName) return;
            var tag = String(el.tagName).toUpperCase();
            if (tag !== 'AUDIO' && tag !== 'VIDEO') return;
            var s = el.getAttribute('src');
            if (s && !el.__respect_src) el.__respect_src = s;
            if (s) el.removeAttribute('src');
            el.__respect_managed = true;
            var srcs = el.getElementsByTagName('source');
            for (var i = 0; i < srcs.length; i++) {
              var ss = srcs[i].getAttribute('src');
              if (ss && !el.__respect_src) el.__respect_src = ss;
              if (ss) srcs[i].removeAttribute('src');
            }
          } catch (e) {}
        }
        function _mNeutralizeAll(root) {
          try {
            var r = root || document;
            var a = r.getElementsByTagName('audio');
            for (var i = 0; i < a.length; i++) _mNeutralize(a[i]);
            var v = r.getElementsByTagName('video');
            for (var j = 0; j < v.length; j++) _mNeutralize(v[j]);
          } catch (e) {}
        }

        // Simpan referensi Blob/File dari URL.createObjectURL agar blob: bisa dibaca ulang.
        try {
          if (!window.__respectBlobs && typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function') {
            window.__respectBlobs = {};
            var _origCOU = URL.createObjectURL;
            URL.createObjectURL = function (obj) {
              var u = _origCOU.call(URL, obj);
              try { window.__respectBlobs[u] = obj; } catch (e) {}
              return u;
            };
          }
        } catch (e) {}

        function _mPlaySource(el, handle, isVideo, src) {
          try {
            if (src.indexOf('blob:') === 0 && window.__respectBlobs && window.__respectBlobs[src]) {
              var blob = window.__respectBlobs[src];
              return new Promise(function (resolve, reject) {
                var fr = new FileReader();
                fr.onload = function () {
                  var res = String(fr.result || '');
                  var idx = res.indexOf(',');
                  var meta = res.substring(0, idx);
                  var b64 = res.substring(idx + 1);
                  var mime = '';
                  var mm = meta.match(/data:([^;]+)/);
                  if (mm) mime = mm[1];
                  resolve(invoke('media.loadData', { handle: handle, name: (blob.name || 'media'), mime: mime, data: b64, video: isVideo }));
                };
                fr.onerror = function () { reject(new Error('gagal membaca blob media')); };
                fr.readAsDataURL(blob);
              });
            }
            if (src.indexOf('data:') === 0) {
              var i2 = src.indexOf(',');
              if (i2 !== -1) {
                var meta2 = src.substring(5, i2);
                var b64b = src.substring(i2 + 1);
                if (/;base64/i.test(meta2)) {
                  var mime2 = meta2.replace(/;base64/i, '');
                  return invoke('media.loadData', { handle: handle, name: '', mime: mime2, data: b64b, video: isVideo });
                }
              }
            }
          } catch (e) {}
          return invoke('media.load', { handle: handle, src: src, video: isVideo });
        }

        try {
          Object.defineProperty(HTMLMediaElement.prototype, 'src', {
            configurable: true,
            get: function () { return this.__respect_src || this.getAttribute('src') || ''; },
            set: function (v) {
              this.__respect_managed = true;
              this.__respect_src = String(v == null ? '' : v);
              // JANGAN setAttribute('src'): cegah engine memuat media (pasti gagal) dan
              // memicu error native palsu. Hanya simpan di properti internal.
              try { this.setAttribute('data-respect-src', this.__respect_src); } catch (e) {}
            }
          });
          if (typeof HTMLSourceElement !== 'undefined') {
            Object.defineProperty(HTMLSourceElement.prototype, 'src', {
              configurable: true,
              get: function () { return this.__respect_src || this.getAttribute('src') || ''; },
              set: function (v) {
                this.__respect_src = String(v == null ? '' : v);
                try {
                  var p = this.parentNode;
                  if (p && (String(p.tagName).toUpperCase() === 'AUDIO' || String(p.tagName).toUpperCase() === 'VIDEO')) {
                    p.__respect_src = this.__respect_src;
                    p.__respect_managed = true;
                  }
                } catch (e) {}
              }
            });
          }
          Object.defineProperty(HTMLMediaElement.prototype, 'currentSrc', { configurable: true, get: function () { return _mSrc(this); } });
          Object.defineProperty(HTMLMediaElement.prototype, 'readyState', { configurable: true, get: function () { return _mState(this).ready; } });
          Object.defineProperty(HTMLMediaElement.prototype, 'duration', { configurable: true, get: function () { var s = _mState(this); return s.dur ? s.dur / 1000 : NaN; } });
          Object.defineProperty(HTMLMediaElement.prototype, 'currentTime', {
            configurable: true,
            get: function () { return _mState(this).pos / 1000; },
            set: function (v) { var el = this; var s = _mState(el); s.pos = Math.max(0, Math.round((Number(v) || 0) * 1000)); if (el.__respect_handle) invoke('media.seek', { handle: el.__respect_handle, ms: s.pos }).catch(function () {}); _mFire(el, 'timeupdate'); }
          });
          Object.defineProperty(HTMLMediaElement.prototype, 'paused', { configurable: true, get: function () { return !_mState(this).playing; } });
          Object.defineProperty(HTMLMediaElement.prototype, 'ended', { configurable: true, get: function () { return !!_mState(this).ended; } });
          Object.defineProperty(HTMLMediaElement.prototype, 'volume', {
            configurable: true,
            get: function () { return _mState(this).vol; },
            set: function (v) { var el = this; var s = _mState(el); s.vol = Math.max(0, Math.min(1, Number(v) || 0)); if (el.__respect_handle) invoke('media.volume', { handle: el.__respect_handle, volume: s.vol }).catch(function () {}); }
          });
        } catch (e) {}

        HTMLMediaElement.prototype.canPlayType = function (type) {
          var t = String(type || '').toLowerCase();
          if (t.indexOf('mpegurl') !== -1 || t.indexOf('m3u') !== -1) return ''; // HLS tidak didukung (butuh MSE)
          if (t.indexOf('mp4') !== -1 || t.indexOf('avc1') !== -1 || t.indexOf('h264') !== -1 || t.indexOf('audio/mpeg') !== -1 || t.indexOf('mp3') !== -1 || t.indexOf('aac') !== -1 || t.indexOf('wav') !== -1 || t.indexOf('wave') !== -1 || t.indexOf('ogg') !== -1 || t.indexOf('avi') !== -1 || t.indexOf('mpeg') !== -1) return 'probably';
          return '';
        };

        HTMLMediaElement.prototype.load = function () { _mFire(this, 'loadstart'); };

        HTMLMediaElement.prototype.play = function () {
          var el = this;
          var s = _mState(el);
          var src = _mSrc(el);
          if (!src) { return Promise.reject(new Error('sumber media kosong')); }
          var isVideo = String(el.tagName).toUpperCase() === 'VIDEO';
          var handle = _mHandle(el);
          el.__respect_src = src;
          return _mPlaySource(el, handle, isVideo, src).then(function (info) {
            s.dur = (info && info.duration) ? info.duration : 0;
            s.ended = false;
            s.ready = 4;
            _mFire(el, 'loadstart');
            _mFire(el, 'loadedmetadata');
            _mFire(el, 'durationchange');
            _mFire(el, 'canplay');
            return invoke('media.play', { handle: handle });
          }).then(function () {
            s.playing = true;
            _mFire(el, 'play');
            _mFire(el, 'playing');
            _mStartPoll(el);
            _mSendRect(el);
          }).catch(function (err) {
            s.playing = false;
            s.ready = 0;
            try { el.error = { code: 4, MEDIA_ERR_SRC_NOT_SUPPORTED: 4, message: String((err && err.message) || err) }; } catch (e) {}
            _mFire(el, 'error');
            throw err;
          });
        };

        HTMLMediaElement.prototype.pause = function () {
          var el = this; var s = _mState(el);
          if (el.__respect_handle) { invoke('media.pause', { handle: el.__respect_handle }).catch(function () {}); }
          s.playing = false; _mStopPoll(el); _mFire(el, 'pause');
        };

        try {
          window.addEventListener('scroll', function () {
            var vids = document.getElementsByTagName('video');
            for (var i = 0; i < vids.length; i++) { if (vids[i].__respect_handle) _mSendRect(vids[i]); }
          }, true);
          window.addEventListener('resize', function () {
            var vids = document.getElementsByTagName('video');
            for (var i = 0; i < vids.length; i++) { if (vids[i].__respect_handle) _mSendRect(vids[i]); }
          });
        } catch (e) {}

        // Sembunyikan event 'error' native palsu dari elemen media yang kita kelola.
        try {
          document.addEventListener('error', function (ev) {
            try {
              var t = ev.target;
              if (t && t.__respect_managed && !ev.__respectSynthetic) { ev.stopPropagation(); ev.preventDefault(); }
            } catch (e) {}
          }, true);
        } catch (e) {}

        // Netralkan src native (HTML-parsed) sekarang & untuk node baru.
        try {
          if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', function () { _mNeutralizeAll(document); });
          } else {
            _mNeutralizeAll(document);
          }
          if (typeof MutationObserver === 'function' && document.documentElement) {
            new MutationObserver(function (mut) {
              for (var i = 0; i < mut.length; i++) {
                var nodes = mut[i].addedNodes || [];
                for (var j = 0; j < nodes.length; j++) _mNeutralize(nodes[j]);
              }
            }).observe(document.documentElement, { childList: true, subtree: true });
          }
        } catch (e) {}
      }
    } catch (e) {}

  } catch (e) { /* shim tidak boleh menggagalkan halaman */ }
})();
`
}
