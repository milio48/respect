package cryptopayload

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

// Mask bytes terobfuskasi untuk salting kunci (terdistribusi di machine code)
var saltSeed = [...]byte{
	0x9A, 0x4B, 0x1F, 0x7C, 0xD2, 0x83, 0x5E, 0x21,
	0x6B, 0xF4, 0x08, 0x3D, 0xC7, 0xAA, 0x55, 0x19,
	0x82, 0x3E, 0x91, 0x4C, 0xFA, 0x73, 0x2B, 0x06,
	0xE8, 0x15, 0x64, 0xBC, 0x30, 0xDF, 0x49, 0x77,
}

// Additional Authenticated Data (AAD) untuk mengikat ciphertext ke magic header V2
var aadV2 = []byte("RESPECTv2\x00\x00\x00\x00\x00\x00\x00")

// DeriveKey meracik kunci pseudorandom 32-byte untuk AES-256
// dari cuplikan PE header biner, ukuran file dasar, dan operasi XOR masking terselubung.
func DeriveKey(peSample []byte, totalSize int64) [32]byte {
	limit := len(peSample)
	if limit > 4096 {
		limit = 4096 // 4 KB pertama berisi PE header konstan
	}
	sample := peSample[:limit]

	mac := hmac.New(sha256.New, saltSeed[:])
	mac.Write(sample)

	// Tambahkan entropy ukuran biner
	var lenBuf [8]byte
	u := uint64(totalSize)
	for i := 0; i < 8; i++ {
		lenBuf[i] = byte(u >> (i * 8))
	}
	mac.Write(lenBuf[:])

	var key [32]byte
	copy(key[:], mac.Sum(nil))
	return key
}

// Encrypt mengenkripsi data plaintext menggunakan AES-256-GCM.
// Menghasilkan blob berformat: [12-byte Nonce] + [Ciphertext + 16-byte Auth Tag].
func Encrypt(plaintext []byte, peSample []byte, totalSize int64) ([]byte, error) {
	key := DeriveKey(peSample, totalSize)
	defer zeroBytes(key[:])

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi cipher AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("gagal membuat nonce acak: %w", err)
	}

	// Seal menempelkan ciphertext + tag di belakang nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, aadV2)
	return ciphertext, nil
}

// Decrypt mendekripsi ciphertext yang dihasilkan Encrypt menggunakan AES-256-GCM.
func Decrypt(encryptedBlob []byte, peSample []byte, totalSize int64) ([]byte, error) {
	key := DeriveKey(peSample, totalSize)
	defer zeroBytes(key[:])

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi cipher AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedBlob) < nonceSize+gcm.Overhead() {
		return nil, errors.New("data terenkripsi tidak valid: ukuran terlalu pendek")
	}

	nonce := encryptedBlob[:nonceSize]
	ciphertext := encryptedBlob[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aadV2)
	if err != nil {
		return nil, fmt.Errorf("otentikasi dekripsi payload gagal (data korup atau kunci tidak cocok): %w", err)
	}

	return plaintext, nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
