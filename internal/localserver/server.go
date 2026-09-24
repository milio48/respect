package localserver

import (
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
// dan mengaktifkan Secure Context (isSecureContext: true, crypto.subtle, navigator.clipboard)
// di engine Chromium/Miniblink.
type Server struct {
	Port     int
	listener net.Listener
	httpSrv  *http.Server
	once     sync.Once
}

// Start menginisialisasi dan menjalankan HTTP server in-memory pada port acak loopback 127.0.0.1:0.
func Start(files map[string][]byte) (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("gagal bind listener localserver: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path
		if reqPath == "" || reqPath == "/" {
			reqPath = "index.html"
		}
		cleanPath := strings.TrimPrefix(filepath.ToSlash(reqPath), "/")

		// Cari file di map virtual in-memory
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

		// Tentukan MIME type
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

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	srv := &http.Server{
		Handler: mux,
	}

	s := &Server{
		Port:     port,
		listener: listener,
		httpSrv:  srv,
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	return s, nil
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
