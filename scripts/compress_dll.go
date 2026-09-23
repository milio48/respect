package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/klauspost/compress/zstd"
)

func main() {
	src := "assets/blink.dll"
	dst := "assets/blink.dll.zst"

	if len(os.Args) > 1 {
		src = os.Args[1]
	}
	if len(os.Args) > 2 {
		dst = os.Args[2]
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "File sumber tidak ditemukan: %v\n", err)
		os.Exit(1)
	}

	if dstInfo, err := os.Stat(dst); err == nil {
		if dstInfo.ModTime().After(srcInfo.ModTime()) && dstInfo.Size() > 0 {
			fmt.Printf("File %s sudah mutakhir (%.2f MB)\n", dst, float64(dstInfo.Size())/(1024*1024))
			return
		}
	}

	fmt.Printf("Mengompres %s (%.2f MB) -> %s dengan zstd...\n", src, float64(srcInfo.Size())/(1024*1024), dst)
	t0 := time.Now()

	in, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gagal membuka sumber: %v\n", err)
		os.Exit(1)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gagal membuat tujuan: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gagal menginisialisasi zstd writer: %v\n", err)
		os.Exit(1)
	}

	if _, err := io.Copy(enc, in); err != nil {
		_ = enc.Close()
		fmt.Fprintf(os.Stderr, "Gagal kompresi: %v\n", err)
		os.Exit(1)
	}
	if err := enc.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Gagal menutup encoder: %v\n", err)
		os.Exit(1)
	}

	dstInfo, _ := out.Stat()
	ratio := (1.0 - float64(dstInfo.Size())/float64(srcInfo.Size())) * 100.0
	fmt.Printf("Selesai dalam %v! Ukuran: %.2f MB (Hemat %.1f%%)\n", time.Since(t0), float64(dstInfo.Size())/(1024*1024), ratio)
}
