package payload

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
)

var ErrNoPayload = errors.New("no payload found")

// ReadPayload membaca config dari trailer EXE sendiri.
func ReadPayload() (*Config, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return readPayloadFrom(self)
}

// ReadPayloadFrom membaca config dari file EXE tertentu.
func ReadPayloadFrom(path string) (*Config, error) {
	return readPayloadFrom(path)
}

func readPayloadFrom(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size < TrailerSize {
		return nil, ErrNoPayload
	}

	// Baca 16 byte terakhir (magic)
	magic := make([]byte, 16)
	if _, err := f.ReadAt(magic, size-16); err != nil {
		return nil, err
	}
	if string(magic) != Magic {
		return nil, ErrNoPayload
	}

	// Baca config_len (8 byte sebelum magic)
	lenBuf := make([]byte, 8)
	if _, err := f.ReadAt(lenBuf, size-TrailerSize); err != nil {
		return nil, err
	}
	cfgLen := int64(binary.LittleEndian.Uint64(lenBuf))
	if cfgLen <= 0 || cfgLen > size-TrailerSize {
		return nil, ErrNoPayload
	}

	// Baca config JSON
	cfgBuf := make([]byte, cfgLen)
	cfgStart := size - TrailerSize - cfgLen
	if _, err := f.ReadAt(cfgBuf, cfgStart); err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(cfgBuf, &cfg); err != nil {
		return nil, err
	}
	cfg.Defaults()
	return &cfg, nil
}

// StripTrailer memotong payload lama dari bytes EXE.
// Dipakai supaya self-copy tidak menumpuk payload.
func StripTrailer(b []byte) []byte {
	if len(b) < TrailerSize {
		return b
	}
	magic := b[len(b)-16:]
	if string(magic) != Magic {
		return b
	}
	lenBuf := b[len(b)-TrailerSize : len(b)-16]
	cfgLen := int64(binary.LittleEndian.Uint64(lenBuf))
	if cfgLen <= 0 || cfgLen > int64(len(b))-TrailerSize {
		return b
	}
	return b[:len(b)-TrailerSize-int(cfgLen)]
}
