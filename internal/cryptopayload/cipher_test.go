package cryptopayload

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	peSample := []byte("DUMMY_EXECUTABLE_BASE_BINARY_MACHINE_CODE_PE_HEADER_1234567890")
	totalSize := int64(1048576) // 1 MB
	secretData := []byte("<html><body><h1>Rahasia Perusahaan Tidak Boleh Bocor</h1></body></html>")

	encrypted, err := Encrypt(secretData, peSample, totalSize)
	if err != nil {
		t.Fatalf("Encrypt gagal: %v", err)
	}

	// Verifikasi overhead tepat 28 byte (12 byte Nonce + 16 byte Tag)
	expectedLen := len(secretData) + 12 + 16
	if len(encrypted) != expectedLen {
		t.Fatalf("Overhead salah: panjang enkripsi %d, ekspektasi %d (tambahan %d byte)", len(encrypted), expectedLen, len(encrypted)-len(secretData))
	}

	decrypted, err := Decrypt(encrypted, peSample, totalSize)
	if err != nil {
		t.Fatalf("Decrypt gagal: %v", err)
	}

	if !bytes.Equal(decrypted, secretData) {
		t.Fatalf("Data didekripsi tidak cocok dengan aslinya: %s", string(decrypted))
	}
}

func TestNonceUniqueness(t *testing.T) {
	peSample := []byte("DUMMY_EXECUTABLE")
	totalSize := int64(1048576)
	secretData := []byte("SAME_DATA")

	enc1, _ := Encrypt(secretData, peSample, totalSize)
	enc2, _ := Encrypt(secretData, peSample, totalSize)

	if bytes.Equal(enc1, enc2) {
		t.Fatal("Dua enkripsi dari data yang sama menghasilkan ciphertext yang identik (Nonce tidak acak!)")
	}
}

func TestTamperingProtection(t *testing.T) {
	peSample := []byte("DUMMY_EXECUTABLE")
	totalSize := int64(1048576)
	secretData := []byte("DATA_PENTING")

	encrypted, err := Encrypt(secretData, peSample, totalSize)
	if err != nil {
		t.Fatalf("Encrypt gagal: %v", err)
	}

	// Ubah 1 bit di tengah ciphertext
	corrupted := append([]byte(nil), encrypted...)
	corrupted[15] ^= 0x01

	_, err = Decrypt(corrupted, peSample, totalSize)
	if err == nil {
		t.Fatal("Ekspektasi dekripsi gagal saat data di-tamper, tapi lolos tanpa error!")
	}
}

func TestKeyMismatchProtection(t *testing.T) {
	peSampleA := []byte("EXECUTABLE_ORIGINAL_VERSION_A")
	peSampleB := []byte("EXECUTABLE_MODIFIED_VERSION_B")
	totalSize := int64(1048576)
	secretData := []byte("DATA_RAHASIA")

	encrypted, _ := Encrypt(secretData, peSampleA, totalSize)

	// Coba dekripsi menggunakan binary yang berbeda
	_, err := Decrypt(encrypted, peSampleB, totalSize)
	if err == nil {
		t.Fatal("Ekspektasi dekripsi gagal saat kunci binary berbeda, tapi lolos!")
	}
}
