package winprint

import (
	"testing"
)

func TestCalculatePrintRect(t *testing.T) {
	// A4 600 DPI: 4961 x 7016
	printerW := 4961
	printerH := 7016
	dpiX := 600
	dpiY := 600

	srcW := 800
	srcH := 600

	rect := CalculatePrintRect(printerW, printerH, dpiX, dpiY, srcW, srcH)

	if rect.W <= 0 || rect.H <= 0 {
		t.Fatalf("Dimensi cetak tidak valid: %+v", rect)
	}
	if rect.X < 0 || rect.Y < 0 {
		t.Fatalf("Koordinat cetak tidak boleh negatif: %+v", rect)
	}
	if rect.X+rect.W > printerW {
		t.Fatalf("Lebar cetak melebihi batas kertas: %d > %d", rect.X+rect.W, printerW)
	}
	if rect.Y+rect.H > printerH {
		t.Fatalf("Tinggi cetak melebihi batas kertas: %d > %d", rect.Y+rect.H, printerH)
	}

	// Cek aspek rasio harus terjaga (800 / 600 = 1.333...)
	aspectSrc := float64(srcW) / float64(srcH)
	aspectDest := float64(rect.W) / float64(rect.H)
	diff := aspectDest - aspectSrc
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("Aspek rasio berubah: src %f vs dest %f", aspectSrc, aspectDest)
	}
}

func TestHasPrinters(t *testing.T) {
	// Memastikan fungsi HasPrinters berjalan tanpa panik
	_ = HasPrinters()
}
