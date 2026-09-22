package version

import "fmt"

var (
	// AppVersion adalah versi rilis Respect (dapat di-override via go -ldflags "-X ...")
	AppVersion = "1.0.0"
)

// String mengembalikan deskripsi lengkap versi aplikasi
func String() string {
	return fmt.Sprintf("%s v%s-%s (Engine: %s)", BinaryName, AppVersion, Edition, EngineName)
}

// GetInfo mengembalikan informasi versi dalam bentuk map untuk IPC/UI
func GetInfo() map[string]string {
	return map[string]string{
		"version":    AppVersion,
		"edition":    Edition,
		"binary":     BinaryName,
		"engine":     EngineName,
		"variant":    Variant,
		"fullString": String(),
	}
}
