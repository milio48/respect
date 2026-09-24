package mb132

// media.go — Pemutar media native memakai WinMM MCI (fitur bawaan Windows).
//
// Engine Miniblink tidak punya media pipeline, jadi playback dialihkan ke MCI:
//   Audio:  mciSendString("open ... alias X; play X")
//   Video:  MCI merender ke child-window overlay di atas webview (play X window HWND)
//
// MCI memakai codec sistem Windows (bukan DLL aplikasi pihak ketiga).
// Format yang didukung bergantung pada device codec Windows; MP3/WAV/AVI/WMV/MPG
// umumnya jalan. MP4/H.264 TIDAK didukung MCI — untuk itu diperlukan dekoder
// open-source (mis. ffmpeg), yang tidak dibundel di sini.

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
	winmmMedia             = syscall.NewLazyDLL("winmm.dll")
	procMciSendStringW     = winmmMedia.NewProc("mciSendStringW")
	procMciGetErrorStringW = winmmMedia.NewProc("mciGetErrorStringW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
)

const (
	wsChild   = 0x40000000
	wsVisible = 0x10000000
)

type mediaSession struct {
	alias    string
	video    bool
	hwnd     uintptr
	path     string
	tempFile bool
	lengthMs int64
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
	if s.alias != "" {
		mciSend("stop " + s.alias)
		mciSend("close " + s.alias)
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
	alias := fmt.Sprintf("respect_media_%d", mediaCounter)
	mediaMu.Unlock()

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
		errText := mciErrorString(lastErr)
		if temp {
			_ = os.Remove(path)
		}
		if errText == "" {
			errText = "format tidak didukung codec Windows"
		}
		if video {
			return nil, fmt.Errorf("video tidak bisa diputar embedded (%s, code %d). MCI/Windows tidak mendukung format ini; dibutuhkan dekoder open-source (ffmpeg).", errText, lastErr)
		}
		return nil, fmt.Errorf("MCI gagal membuka media: %s (code %d)", errText, lastErr)
	}
	mciSend("set " + alias + " time format milliseconds")

	var lengthMs int64
	if v, code := mciSend("status " + alias + " length"); code == 0 {
		lengthMs, _ = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	}

	sess := &mediaSession{alias: alias, video: video, path: path, tempFile: temp, lengthMs: lengthMs}
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
	cmd := "play " + sess.alias
	if sess.video {
		mediaMu.Lock()
		if sess.hwnd == 0 {
			sess.hwnd = createOverlayWindow(wv.hwnd)
		}
		hwnd := sess.hwnd
		mediaMu.Unlock()
		if hwnd != 0 {
			cmd = fmt.Sprintf("play %s window %d", sess.alias, hwnd)
		}
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
	mediaMu.Lock()
	if sess.hwnd == 0 {
		sess.hwnd = createOverlayWindow(wv.hwnd)
	}
	hwnd := sess.hwnd
	mediaMu.Unlock()
	if hwnd == 0 {
		return fmt.Errorf("gagal membuat overlay video")
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoZOrder|swpShowWindow|0x0010)
	return nil
}
