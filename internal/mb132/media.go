package mb132

// media.go — Pemutar media native memakai Windows Media Foundation (MFPlay)
// dan fallback WinMM MCI (fitur bawaan Windows).
//
// Engine Miniblink tidak memiliki Chromium media pipeline internal, sehingga
// pemutaran media didelegasikan ke subsistem native Windows:
//   - Audio: Media Foundation / MCI (MP3, WAV, AAC, M4A, OGG/WMA)
//   - Video: Media Foundation merender ke child-window HWND overlay di atas webview
//            (Mendukung MP4 [H.264 + AAC], WMV, AVI via DirectX Video Acceleration)
//
// Kode ini murni memakai API bawaan Windows (mfplay.dll & winmm.dll),
// 0 KB overhead, tanpa membengkakkan ukuran binary, dan tanpa dependensi luar.

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modMfplay                = syscall.NewLazyDLL("mfplay.dll")
	procMFPCreateMediaPlayer = modMfplay.NewProc("MFPCreateMediaPlayer")

	winmmMedia             = syscall.NewLazyDLL("winmm.dll")
	procMciSendStringW     = winmmMedia.NewProc("mciSendStringW")
	procMciGetErrorStringW = winmmMedia.NewProc("mciGetErrorStringW")

	procCreateWindowExW = user32.NewProc("CreateWindowExW")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
)

var guidMFP100ns [16]byte // GUID_NULL = 100ns position type

type mfpPropVariant struct {
	vt         uint16
	wReserved1 uint16
	wReserved2 uint16
	wReserved3 uint16
	hVal       int64
	padding    int64 // Pad struct to exact 24-byte PROPVARIANT in 64-bit Windows
}

type mfpRect struct {
	Left, Top, Right, Bottom int32
}

const (
	wsChild           = 0x40000000
	wsVisible         = 0x10000000
	swHide            = 0
	swShowNoActivate  = 4
	swpAsyncWindowPos = 0x4000
)

func mfpCall(obj uintptr, index int, args ...uintptr) uintptr {
	if obj == 0 {
		return 0x80004005 // E_FAIL
	}
	vtbl := *(**uintptr)(unsafe.Pointer(obj))
	method := *(*uintptr)(unsafe.Add(unsafe.Pointer(vtbl), uintptr(index)*unsafe.Sizeof(uintptr(0))))
	allArgs := append([]uintptr{obj}, args...)
	r, _, _ := syscall.SyscallN(method, allArgs...)
	return r
}


type mediaSession struct {
	player   uintptr // IMFPMediaPlayer (MFPlay) jika aktif
	alias    string  // MCI alias jika fallback ke MCI
	video    bool
	hwnd     uintptr
	path     string
	tempFile bool
	lengthMs int64
	backend  string
}

var (
	mediaMu       sync.Mutex
	mediaSessions = map[string]*mediaSession{}
	mediaCounter  int
)

func mciSend(cmd string) (string, uint32) {
	cmdW, err := syscall.UTF16FromString(cmd)
	if err != nil || len(cmdW) == 0 {
		return "", 1
	}
	buf := make([]uint16, 4096)
	r, _, _ := procMciSendStringW.Call(
		uintptr(unsafe.Pointer(&cmdW[0])),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	return strings.TrimSpace(syscall.UTF16ToString(buf)), uint32(r)
}

func mciErrorString(code uint32) string {
	if procMciGetErrorStringW.Find() != nil {
		return ""
	}
	buf := make([]uint16, 512)
	procMciGetErrorStringW.Call(uintptr(code), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func mediaTempDir() string {
	dir := filepath.Join(os.TempDir(), "respect_media")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func writeMediaTemp(name string, data []byte) (string, error) {
	h := sha1.Sum(data)
	ext := filepath.Ext(name)
	p := filepath.Join(mediaTempDir(), hex.EncodeToString(h[:8])+ext)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return "", err
	}
	return p, nil
}

// resolveMediaSource memetakan src (http/https/file) menjadi path lokal.
func (wv *WebView) resolveMediaSource(src string) (string, bool, error) {
	u, err := url.Parse(src)
	if err != nil {
		return "", false, fmt.Errorf("URL tidak valid: %s", src)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		host := strings.ToLower(u.Hostname())
		if (host == "app" || host == "app.local") && len(wv.virtualFiles) > 0 {
			rel := strings.TrimPrefix(u.Path, "/")
			if rel == "" {
				rel = "index.html"
			}
			if data, ok := wv.virtualFiles[rel]; ok {
				p, e := writeMediaTemp(rel, data)
				return p, true, e
			}
			return "", false, fmt.Errorf("berkas virtual tidak ditemukan: %s", rel)
		}
		return downloadMedia(src)
	case "file":
		p := u.Path
		if p == "" {
			p = strings.TrimPrefix(src, "file://")
		}
		p = filepath.FromSlash(strings.TrimPrefix(p, "/"))
		return p, false, nil
	case "blob", "data":
		return "", false, fmt.Errorf("skema %s tidak didukung lewat jalur berkas", u.Scheme)
	default:
		if src != "" && !strings.Contains(src, "://") {
			return filepath.FromSlash(src), false, nil
		}
		return "", false, fmt.Errorf("skema sumber tidak didukung: %s", src)
	}
}

func downloadMedia(rawURL string) (string, bool, error) {
	base := rawURL
	if idx := strings.IndexAny(base, "?#"); idx != -1 {
		base = base[:idx]
	}
	ext := filepath.Ext(base)
	if ext == "" {
		ext = ".bin"
	}
	h := sha1.Sum([]byte(rawURL))
	target := filepath.Join(mediaTempDir(), hex.EncodeToString(h[:8])+ext)
	if fi, err := os.Stat(target); err == nil && fi.Size() > 0 {
		return target, true, nil
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", false, fmt.Errorf("gagal mengunduh media: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("unduh media gagal, status %d", resp.StatusCode)
	}

	tmp := target + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", false, err
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 1<<30)); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", false, fmt.Errorf("gagal menulis media: %w", err)
	}
	_ = f.Close()
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return "", false, err
	}
	return target, true, nil
}

func (s *mediaSession) close() {
	if s == nil {
		return
	}
	if s.player != 0 {
		mfpCall(s.player, 34) // Shutdown
		mfpCall(s.player, 2)  // Release
		s.player = 0
	}
	if s.alias != "" {
		mciSend("stop " + s.alias)
		mciSend("close " + s.alias)
		s.alias = ""
	}
	if s.hwnd != 0 {
		procDestroyWindow.Call(s.hwnd)
		s.hwnd = 0
	}
	if s.tempFile && s.path != "" {
		_ = os.Remove(s.path)
	}
}

func (wv *WebView) mediaGet(handle string) *mediaSession {
	mediaMu.Lock()
	defer mediaMu.Unlock()
	return mediaSessions[handle]
}

func (wv *WebView) mediaLoad(handle, src string, video bool) (map[string]interface{}, error) {
	if handle == "" {
		return nil, fmt.Errorf("handle media kosong")
	}
	path, temp, err := wv.resolveMediaSource(src)
	if err != nil {
		return nil, err
	}
	return wv.mediaOpenSession(handle, path, temp, video)
}

func mediaExtFromMime(mime string) string {
	m := strings.ToLower(mime)
	switch {
	case strings.Contains(m, "wav"):
		return ".wav"
	case strings.Contains(m, "mpeg"), strings.Contains(m, "mp3"):
		return ".mp3"
	case strings.Contains(m, "mp4"), strings.Contains(m, "m4a"), strings.Contains(m, "aac"):
		return ".mp4"
	case strings.Contains(m, "ogg"):
		return ".ogg"
	case strings.Contains(m, "webm"):
		return ".webm"
	case strings.Contains(m, "avi"):
		return ".avi"
	default:
		return ".bin"
	}
}

// mediaLoadData menerima media inline (base64 dari data: URI atau file lokal via blob:).
func (wv *WebView) mediaLoadData(handle, name, mime, b64 string, video bool) (map[string]interface{}, error) {
	if handle == "" {
		return nil, fmt.Errorf("handle media kosong")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		raw2, err2 := base64.RawStdEncoding.DecodeString(strings.TrimSpace(b64))
		if err2 != nil {
			return nil, fmt.Errorf("data media base64 tidak valid")
		}
		raw = raw2
	}
	ext := filepath.Ext(name)
	if ext == "" {
		ext = mediaExtFromMime(mime)
	}
	path, err := writeMediaTemp("inline"+ext, raw)
	if err != nil {
		return nil, err
	}
	return wv.mediaOpenSession(handle, path, true, video)
}

func (wv *WebView) mediaOpenSession(handle, path string, temp, video bool) (map[string]interface{}, error) {
	mediaMu.Lock()
	if old := mediaSessions[handle]; old != nil {
		old.close()
		delete(mediaSessions, handle)
	}
	mediaCounter++
	mediaMu.Unlock()

	var overlayHwnd uintptr
	if video {
		overlayHwnd = createOverlayWindow(wv.hwnd)
	}

	// 1. Coba gunakan Windows Media Foundation (mfplay.dll)
	// Media Foundation natively mendukung MP4 (H.264/AAC), MP3, WAV, WMV, AVI
	if procMFPCreateMediaPlayer.Find() == nil {
		urlW, err := syscall.UTF16PtrFromString(path)
		if err == nil {
			var player uintptr
			hr, _, _ := procMFPCreateMediaPlayer.Call(
				0, 0, 0, 0, overlayHwnd,
				uintptr(unsafe.Pointer(&player)),
			)
			if hr == 0 && player != 0 {
				var item uintptr
				// CreateMediaItemFromURL synchronously (fSync = 1)
				r := mfpCall(player, 14, uintptr(unsafe.Pointer(urlW)), 1, 0, uintptr(unsafe.Pointer(&item)))
				if r == 0 && item != 0 {
					var durPV mfpPropVariant
					mfpCall(item, 13, uintptr(unsafe.Pointer(&guidMFP100ns[0])), uintptr(unsafe.Pointer(&durPV)))
					lengthMs := durPV.hVal / 10000

					mfpCall(player, 16, item) // SetMediaItem
					mfpCall(item, 2)          // Release item

					sess := &mediaSession{
						player:   player,
						video:    video,
						hwnd:     overlayHwnd,
						path:     path,
						tempFile: temp,
						lengthMs: lengthMs,
						backend:  "mfplay",
					}

					mediaMu.Lock()
					mediaSessions[handle] = sess
					mediaMu.Unlock()

					return map[string]interface{}{"duration": lengthMs, "backend": "mfplay"}, nil
				}
				// Gagal load item, shutdown player
				mfpCall(player, 34)
				mfpCall(player, 2)
			}
		}
	}

	// 2. Fallback ke WinMM MCI jika Media Foundation tidak tersedia
	alias := fmt.Sprintf("respect_media_%d", mediaCounter)
	attempts := []string{
		fmt.Sprintf("open \"%s\" alias %s", path, alias),
		fmt.Sprintf("open \"%s\" type mpegvideo alias %s", path, alias),
		fmt.Sprintf("open \"%s\" type waveaudio alias %s", path, alias),
	}
	var opened bool
	var lastErr uint32
	for _, cmd := range attempts {
		if _, code := mciSend(cmd); code == 0 {
			opened = true
			break
		} else {
			lastErr = code
		}
	}
	if !opened {
		if overlayHwnd != 0 {
			procDestroyWindow.Call(overlayHwnd)
		}
		errText := mciErrorString(lastErr)
		if temp {
			_ = os.Remove(path)
		}
		if errText == "" {
			errText = "format tidak didukung codec sistem Windows"
		}
		return nil, fmt.Errorf("gagal membuka media: %s (code %d)", errText, lastErr)
	}
	mciSend("set " + alias + " time format milliseconds")

	var lengthMs int64
	if v, code := mciSend("status " + alias + " length"); code == 0 {
		lengthMs, _ = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	}

	sess := &mediaSession{
		alias:    alias,
		video:    video,
		hwnd:     overlayHwnd,
		path:     path,
		tempFile: temp,
		lengthMs: lengthMs,
		backend:  "mci",
	}
	mediaMu.Lock()
	mediaSessions[handle] = sess
	mediaMu.Unlock()

	return map[string]interface{}{"duration": lengthMs, "backend": "mci"}, nil
}

func (wv *WebView) mediaPlay(handle string) (map[string]interface{}, error) {
	sess := wv.mediaGet(handle)
	if sess == nil {
		return nil, fmt.Errorf("media belum dimuat")
	}

	if sess.player != 0 {
		if sess.video && sess.hwnd != 0 {
			procShowWindow.Call(sess.hwnd, swShowNoActivate)
		}
		r := mfpCall(sess.player, 3) // IMFPMediaPlayer::Play
		if r != 0 {
			return nil, fmt.Errorf("Media Foundation play gagal (0x%x)", uint32(r))
		}
		return map[string]interface{}{"ok": true}, nil
	}

	cmd := "play " + sess.alias
	if sess.video && sess.hwnd != 0 {
		cmd = fmt.Sprintf("play %s window %d", sess.alias, sess.hwnd)
	}
	if _, code := mciSend(cmd); code != 0 {
		return nil, fmt.Errorf("MCI play gagal (code %d)", code)
	}
	return map[string]interface{}{"ok": true}, nil
}

func (wv *WebView) mediaSimple(handle, action string) (map[string]interface{}, error) {
	sess := wv.mediaGet(handle)
	if sess == nil {
		return nil, fmt.Errorf("media belum dimuat")
	}

	if sess.player != 0 {
		switch action {
		case "pause":
			mfpCall(sess.player, 4) // Pause
		case "resume":
			mfpCall(sess.player, 3) // Play
		case "stop":
			mfpCall(sess.player, 5) // Stop
			if sess.video && sess.hwnd != 0 {
				procShowWindow.Call(sess.hwnd, swHide)
			}
		}
		return map[string]interface{}{"ok": true}, nil
	}

	if _, code := mciSend(action + " " + sess.alias); code != 0 {
		if action == "resume" {
			if _, c2 := mciSend("play " + sess.alias); c2 == 0 {
				return map[string]interface{}{"ok": true}, nil
			}
		}
		return nil, fmt.Errorf("MCI %s gagal (code %d)", action, code)
	}
	return map[string]interface{}{"ok": true}, nil
}

func (wv *WebView) mediaSeek(handle string, ms int64) error {
	sess := wv.mediaGet(handle)
	if sess == nil {
		return fmt.Errorf("media belum dimuat")
	}

	if sess.player != 0 {
		var pv mfpPropVariant
		pv.vt = 20 // VT_I8
		pv.hVal = ms * 10000
		r := mfpCall(sess.player, 7, uintptr(unsafe.Pointer(&guidMFP100ns[0])), uintptr(unsafe.Pointer(&pv)))
		if r != 0 {
			return fmt.Errorf("Media Foundation seek gagal (0x%x)", uint32(r))
		}
		return nil
	}

	if _, code := mciSend(fmt.Sprintf("seek %s to %d", sess.alias, ms)); code != 0 {
		return fmt.Errorf("MCI seek gagal (code %d)", code)
	}
	return nil
}

func (wv *WebView) mediaVolume(handle string, volume float64) error {
	sess := wv.mediaGet(handle)
	if sess == nil {
		return fmt.Errorf("media belum dimuat")
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}

	if sess.player != 0 {
		vol32 := float32(volume)
		mfpCall(sess.player, 20, uintptr(*(*uint32)(unsafe.Pointer(&vol32))))
		return nil
	}

	if _, code := mciSend(fmt.Sprintf("setaudio %s volume to %d", sess.alias, int(volume*1000))); code != 0 {
		return fmt.Errorf("MCI volume gagal (code %d)", code)
	}
	return nil
}

func (wv *WebView) mediaStatus(handle string) (map[string]interface{}, error) {
	sess := wv.mediaGet(handle)
	if sess == nil {
		return nil, fmt.Errorf("media belum dimuat")
	}

	if sess.player != 0 {
		var posPV mfpPropVariant
		rPos := mfpCall(sess.player, 8, uintptr(unsafe.Pointer(&guidMFP100ns[0])), uintptr(unsafe.Pointer(&posPV)))
		pos := int64(0)
		if rPos == 0 && posPV.hVal > 0 {
			pos = posPV.hVal / 10000
		}

		var state uint32
		mfpCall(sess.player, 13, uintptr(unsafe.Pointer(&state)))
		mode := "stopped"
		switch state {
		case 3: // MFP_MEDIAPLAYER_STATE_PLAYING
			mode = "playing"
		case 2: // MFP_MEDIAPLAYER_STATE_PAUSED
			mode = "paused"
		case 1: // MFP_MEDIAPLAYER_STATE_STOPPED
			mode = "stopped"
		}
		return map[string]interface{}{"position": pos, "duration": sess.lengthMs, "mode": mode}, nil
	}

	pos := int64(0)
	if v, code := mciSend("status " + sess.alias + " position"); code == 0 {
		pos, _ = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	}
	length := sess.lengthMs
	if v, code := mciSend("status " + sess.alias + " length"); code == 0 {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			length = n
		}
	}
	mode := ""
	if v, code := mciSend("status " + sess.alias + " mode"); code == 0 {
		mode = strings.ToLower(strings.TrimSpace(v))
	}
	return map[string]interface{}{"position": pos, "duration": length, "mode": mode}, nil
}

func (wv *WebView) mediaDispose(handle string) {
	mediaMu.Lock()
	sess := mediaSessions[handle]
	delete(mediaSessions, handle)
	mediaMu.Unlock()
	if sess != nil {
		sess.close()
	}
}

func createOverlayWindow(parent uintptr) uintptr {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	cls, _ := syscall.UTF16PtrFromString("STATIC")
	title, _ := syscall.UTF16PtrFromString("RespectMediaOverlay")
	h, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsChild|wsVisible),
		0, 0, 16, 16,
		parent, 0, hInst, 0,
	)
	return h
}

func (wv *WebView) mediaRect(handle string, x, y, w, h int) error {
	sess := wv.mediaGet(handle)
	if sess == nil || !sess.video {
		return nil
	}
	if sess.hwnd == 0 {
		return nil
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	procSetWindowPos.Call(
		sess.hwnd, 0,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		swpNoZOrder|swpShowWindow|0x0010|swpAsyncWindowPos,
	)
	return nil
}

