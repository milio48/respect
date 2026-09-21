package main

import (
	"os"

	blink "github.com/epkgs/blink"
)

func main() {
	// 1. Buat aplikasi (bukan blink.New())
	app := blink.NewApp()
	defer app.Exit()

	// 2. Buat jendela browser
	view := app.CreateWebWindowPopup()

	// 3. Atur judul dan posisi jendela
	view.Window.SetTitle("respect.exe")
	view.Window.MoveToCenter()

	// 4. Muat konten HTML atau URL
	// Untuk file lokal, gunakan format file:///C:/path/to/file.html
	// Untuk string HTML langsung, gunakan view.LoadHtml("<h1>Hello</h1>")
	view.LoadURL("https://www.example.com")

	// 5. Tampilkan jendela
	view.ShowWindow()

	// 6. Pastikan aplikasi keluar saat jendela ditutup
	view.OnDestroy(func() {
		os.Exit(0)
	})

	// 7. Jalankan message loop agar aplikasi tetap berjalan
	app.KeepRunning()
}
