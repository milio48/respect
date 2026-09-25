package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

func compressFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("file sumber tidak ditemukan (%s): %w", src, err)
	}

	if dstInfo, err := os.Stat(dst); err == nil {
		if dstInfo.ModTime().After(srcInfo.ModTime()) && dstInfo.Size() > 0 {
			fmt.Printf("File %s sudah mutakhir (%.2f MB)\n", dst, float64(dstInfo.Size())/(1024*1024))
			return nil
		}
	}

	fmt.Printf("Mengompres %s (%.2f MB) -> %s dengan zstd...\n", src, float64(srcInfo.Size())/(1024*1024), dst)
	t0 := time.Now()

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("gagal membuka sumber: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("gagal membuat direktori tujuan: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("gagal membuat tujuan: %w", err)
	}
	defer out.Close()

	enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return fmt.Errorf("gagal menginisialisasi zstd writer: %w", err)
	}

	if _, err := io.Copy(enc, in); err != nil {
		_ = enc.Close()
		return fmt.Errorf("gagal kompresi: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("gagal menutup encoder: %w", err)
	}

	dstInfo, _ := out.Stat()
	ratio := (1.0 - float64(dstInfo.Size())/float64(srcInfo.Size())) * 100.0
	fmt.Printf("Selesai dalam %v! Ukuran: %.2f MB (Hemat %.1f%%)\n", time.Since(t0), float64(dstInfo.Size())/(1024*1024), ratio)
	return nil
}

func findBlinkModuleDir() string {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/epkgs/blink")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return strings.TrimSpace(out.String())
	}
	return ""
}

func compressModern() error {
	src := "assets/blink.dll"
	dst := "assets/blink.dll.zst"
	if _, err := os.Stat(src); err != nil {
		fmt.Printf("Skip Modern: %s tidak ditemukan\n", src)
		return nil
	}
	return compressFile(src, dst)
}

func compressLite() error {
	moduleDir := findBlinkModuleDir()

	// 1. Lite x64
	srcX64 := ""
	if _, err := os.Stat("assets/miniblink_49_x64.dll"); err == nil {
		srcX64 = "assets/miniblink_49_x64.dll"
	} else if moduleDir != "" {
		cand := filepath.Join(moduleDir, "internal", "miniblink", "release", "x64", "miniblink_4975_x64.dll")
		if _, err := os.Stat(cand); err == nil {
			srcX64 = cand
		}
	}
	if srcX64 == "" {
		cand := filepath.Join(os.Getenv("LOCALAPPDATA"), "respect_desktop", "engine_v49", "miniblink_4975_x64", "miniblink_49.dll")
		if _, err := os.Stat(cand); err == nil {
			srcX64 = cand
		}
	}

	if srcX64 != "" {
		if err := compressFile(srcX64, "assets/miniblink_49_x64.dll.zst"); err != nil {
			return err
		}
	} else {
		fmt.Println("Peringatan: Source DLL Miniblink x64 tidak ditemukan.")
	}

	// 2. Lite x86 (32-bit)
	srcX86 := ""
	if _, err := os.Stat("assets/miniblink_49_x86.dll"); err == nil {
		srcX86 = "assets/miniblink_49_x86.dll"
	} else if moduleDir != "" {
		cand := filepath.Join(moduleDir, "internal", "miniblink", "release", "x32", "miniblink_4975_x32.dll")
		if _, err := os.Stat(cand); err == nil {
			srcX86 = cand
		}
	}

	if srcX86 != "" {
		if err := compressFile(srcX86, "assets/miniblink_49_x86.dll.zst"); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	target := "modern"
	if len(os.Args) > 1 {
		target = strings.ToLower(strings.TrimSpace(os.Args[1]))
	}

	// Dukung custom source & destination file jika parameter kedua diisi
	if len(os.Args) > 2 {
		src := os.Args[1]
		dst := os.Args[2]
		if err := compressFile(src, dst); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	switch target {
	case "lite", "v49":
		if err := compressLite(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "all":
		if err := compressModern(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := compressLite(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default: // "modern" atau "v132"
		if err := compressModern(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
