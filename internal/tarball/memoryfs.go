package tarball

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

// MemoryFS mengimplementasikan io/fs.FS, io/fs.ReadFileFS, dan io/fs.StatFS
// sepenuhnya di atas map memori murni (RAM).
type MemoryFS struct {
	files map[string][]byte
}

// NewMemoryFS membuat instance baru MemoryFS dari map file in-memory.
func NewMemoryFS(files map[string][]byte) *MemoryFS {
	normalized := make(map[string][]byte, len(files))
	for k, v := range files {
		clean := sanitizePath(k)
		if clean != "" {
			normalized[clean] = v
		}
	}
	return &MemoryFS{files: normalized}
}

// Files mengembalikan map file mentah yang tersimpan di memori.
func (m *MemoryFS) Files() map[string][]byte {
	return m.files
}

// Open membuka file untuk dibaca sesuai kontrak io/fs.FS.
func (m *MemoryFS) Open(name string) (fs.File, error) {
	name = sanitizePath(name)
	if name == "" || name == "." {
		return &memDir{name: ".", modTime: time.Now()}, nil
	}

	data, ok := m.files[name]
	if !ok {
		// Cek apakah path ini adalah sebuah direktori
		prefix := name + "/"
		isDir := false
		for k := range m.files {
			if strings.HasPrefix(k, prefix) {
				isDir = true
				break
			}
		}
		if isDir {
			return &memDir{name: path.Base(name), modTime: time.Now()}, nil
		}
		return nil, fs.ErrNotExist
	}

	return &memFile{
		name:    path.Base(name),
		reader:  bytes.NewReader(data),
		size:    int64(len(data)),
		modTime: time.Now(),
	}, nil
}

// ReadFile membaca seluruh isi file sekaligus sesuai kontrak io/fs.ReadFileFS.
func (m *MemoryFS) ReadFile(name string) ([]byte, error) {
	name = sanitizePath(name)
	data, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

// Stat mengambil metadata file sesuai kontrak io/fs.StatFS.
func (m *MemoryFS) Stat(name string) (fs.FileInfo, error) {
	f, err := m.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}

// --- memFile (Implementasi fs.File untuk data biner) ---

type memFile struct {
	name    string
	reader  *bytes.Reader
	size    int64
	modTime time.Time
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	return &memFileInfo{
		name:    f.name,
		size:    f.size,
		mode:    0644,
		modTime: f.modTime,
		isDir:   false,
	}, nil
}

func (f *memFile) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

func (f *memFile) Close() error {
	return nil
}

// --- memDir (Implementasi fs.ReadDirFile untuk direktori virtual) ---

type memDir struct {
	name    string
	modTime time.Time
}

func (d *memDir) Stat() (fs.FileInfo, error) {
	return &memFileInfo{
		name:    d.name,
		size:    0,
		mode:    os.ModeDir | 0755,
		modTime: d.modTime,
		isDir:   true,
	}, nil
}

func (d *memDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: fs.ErrInvalid}
}

func (d *memDir) Close() error {
	return nil
}

func (d *memDir) ReadDir(n int) ([]fs.DirEntry, error) {
	return nil, io.EOF
}

// --- memFileInfo (Implementasi fs.FileInfo) ---

type memFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *memFileInfo) Name() string       { return fi.name }
func (fi *memFileInfo) Size() int64         { return fi.size }
func (fi *memFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *memFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *memFileInfo) IsDir() bool        { return fi.isDir }
func (fi *memFileInfo) Sys() interface{}   { return nil }
