package localserver

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// Server mengelola in-memory HTTP server lokal pada 127.0.0.1.
// Server ini menyajikan file langsung dari RAM (tanpa ekstraksi ke hard disk),
// mengaktifkan Secure Context (isSecureContext: true, crypto.subtle, navigator.clipboard),
// dan mengamankan endpoint dari akses luar aplikasi via one-time token + HttpOnly cookie.
type Server struct {
	Port     int
	Token    string
	listener net.Listener
	httpSrv  *http.Server
	once     sync.Once
}

// Start menginisialisasi dan menjalankan HTTP server in-memory pada port acak loopback 127.0.0.1:0
// dengan token acak 128-bit untuk mencegah akses dari luar proses Respect.
func Start(files map[string][]byte) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("gagal bind listener localserver: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path
		if reqPath == "" || reqPath == "/" {
			reqPath = "index.html"
		}
		cleanPath := strings.TrimPrefix(filepath.ToSlash(reqPath), "/")

		// 1. Verifikasi Keamanan: Cek cookie atau query token
		authed := false
		if c, err := r.Cookie("__respect_token__"); err == nil && c.Value == token {
			authed = true
		}

		q := r.URL.Query()
		tokenParam := q.Get("token")
		if tokenParam == token {
			// Set HttpOnly session cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "__respect_token__",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})

			// Jika navigasi halaman utama (HTML), bersihkan token dari URL bar
			// melalui HTTP 302 Redirect agar URL bersih dan Service Worker scope rapi.
			ext := strings.ToLower(filepath.Ext(cleanPath))
			if ext == "" || ext == ".html" || ext == ".htm" {
				q.Del("token")
				cleanURL := "/" + cleanPath
				if len(q) > 0 {
					cleanURL += "?" + q.Encode()
				}
				http.Redirect(w, r, cleanURL, http.StatusFound)
				return
			}
			authed = true
		}

		// Blokir akses tanpa token / cookie sah (mencegah browser luar/malware mengakses port)
		if !authed {
			http.Error(w, "403 Forbidden: Akses tidak diizinkan di luar runtime Respect", http.StatusForbidden)
			return
		}

		// 2. Cari file di map virtual in-memory
		data, found := files[cleanPath]
		if !found {
			// Coba cari alternatif (misal index.html untuk subdirektori)
			if strings.HasSuffix(cleanPath, "/") {
				cleanPath += "index.html"
				data, found = files[cleanPath]
			}
		}

		if !found {
			http.NotFound(w, r)
			return
		}

		// 3. Tentukan MIME type
		ext := strings.ToLower(filepath.Ext(cleanPath))
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			switch ext {
			case ".html", ".htm":
				contentType = "text/html; charset=utf-8"
			case ".js", ".mjs":
				contentType = "application/javascript; charset=utf-8"
			case ".css":
				contentType = "text/css; charset=utf-8"
			case ".json":
				contentType = "application/json; charset=utf-8"
			case ".svg":
				contentType = "image/svg+xml"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".gif":
				contentType = "image/gif"
			case ".ico":
				contentType = "image/x-icon"
			case ".wasm":
				contentType = "application/wasm"
			case ".mp3":
				contentType = "audio/mpeg"
			case ".wav":
				contentType = "audio/wav"
			case ".m4a":
				contentType = "audio/mp4"
			case ".aac":
				contentType = "audio/aac"
			case ".mp4":
				contentType = "video/mp4"
			case ".webm":
				contentType = "video/webm"
			case ".ogg":
				contentType = "audio/ogg"
			case ".woff2":
				contentType = "font/woff2"
			case ".woff":
				contentType = "font/woff"
			case ".ttf":
				contentType = "font/ttf"
			default:
				contentType = http.DetectContentType(data)
			}
		}

		// 4. Header Keamanan
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	srv := &http.Server{
		Handler: mux,
	}

	s := &Server{
		Port:     port,
		Token:    token,
		listener: listener,
		httpSrv:  srv,
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	return s, nil
}

// URL menghasilkan URL awal terproteksi token untuk dimuat oleh WebView.
func (s *Server) URL(entryFile string) string {
	clean := strings.TrimPrefix(filepath.ToSlash(entryFile), "/")
	if clean == "" {
		clean = "index.html"
	}
	return fmt.Sprintf("http://127.0.0.1:%d/%s?token=%s", s.Port, clean, s.Token)
}

// Close menghentikan server dan melepaskan port loopback.
func (s *Server) Close() {
	s.once.Do(func() {
		if s.httpSrv != nil {
			_ = s.httpSrv.Close()
		}
		if s.listener != nil {
			_ = s.listener.Close()
		}
	})
}
