//go:build embed132

package assets

import _ "embed"

//go:embed blink.dll
var BlinkDLL []byte
