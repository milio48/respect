package payload

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
	"respect-app/internal/tarball"
)

var ErrNoPayload = errors.New("no payload found")

// ReadPayload membaca data payload (V1 atau V2) dari trailer EXE sendiri.
func ReadPayload() (*Payload, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return ReadPayloadFrom(self)
}

// ReadPayloadFrom membaca data payload (V1 atau V2) dari file EXE tertentu.
func ReadPayloadFrom(path string) (*Payload, error) {
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

	// 1. Baca 16 byte terakhir (magic)
	magic := make([]byte, 16)
	if _, err := f.ReadAt(magic, size-16); err != nil {
		return nil, err
	}

	magicStr := string(magic)
	var version int
	if magicStr == MagicV2 {
		version = 2
	} else if magicStr == MagicV1 {
		version = 1
	} else {
		return nil, ErrNoPayload
	}

	// 2. Baca payload_len (8 byte sebelum magic)
	lenBuf := make([]byte, 8)
	if _, err := f.ReadAt(lenBuf, size-TrailerSize); err != nil {
		return nil, err
	}
	payloadLen := int64(binary.LittleEndian.Uint64(lenBuf))
	if payloadLen <= 0 || payloadLen > size-TrailerSize {
		return nil, ErrNoPayload
	}

	// 3. Baca blob payload mentah
	payloadBuf := make([]byte, payloadLen)
	payloadStart := size - TrailerSize - payloadLen
	if _, err := f.ReadAt(payloadBuf, payloadStart); err != nil {
		return nil, err
	}

	// 4. Proses berdasarkan versi
	if version == 2 {
		// V2: zstd(TAR)
		zr, err := zstd.NewReader(bytes.NewReader(payloadBuf))
		if err != nil {
			return nil, fmt.Errorf("gagal inisialisasi dekompresi zstd trailer: %w", err)
		}
		defer zr.Close()

		files, err := tarball.UnpackToMemory(zr)
		if err != nil {
			return nil, fmt.Errorf("gagal mengekstrak arsip tar payload: %w", err)
		}

		// Ambil respect.json untuk konfigurasi jendela
		var cfg Config
		if cfgJSON, ok := files["respect.json"]; ok {
			if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
				return nil, fmt.Errorf("gagal parsing respect.json dari payload: %w", err)
			}
		}
		cfg.Defaults()

		return &Payload{
			Version: 2,
			Config:  cfg,
			Files:   files,
		}, nil
	}

	// V1: JSON Config mentah
	var cfg Config
	if err := json.Unmarshal(payloadBuf, &cfg); err != nil {
		return nil, err
	}
	cfg.Defaults()

	return &Payload{
		Version: 1,
		Config:  cfg,
		Files:   nil,
	}, nil
}

// StripTrailer memotong payload lama (V1 atau V2) dari bytes EXE.
// Dipakai supaya self-copy tidak menumpuk payload berulang.
func StripTrailer(b []byte) []byte {
	if len(b) < TrailerSize {
		return b
	}
	magic := string(b[len(b)-16:])
	if magic != MagicV1 && magic != MagicV2 {
		return b
	}
	lenBuf := b[len(b)-TrailerSize : len(b)-16]
	payloadLen := int64(binary.LittleEndian.Uint64(lenBuf))
	if payloadLen <= 0 || payloadLen > int64(len(b))-TrailerSize {
		return b
	}
	return b[:len(b)-TrailerSize-int(payloadLen)]
}
