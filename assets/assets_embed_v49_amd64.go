//go:build (embed49 || embedlite) && !v132 && !386

package assets

import _ "embed"

//go:embed miniblink_49_x64.dll.zst
var Miniblink49DLLZst []byte
