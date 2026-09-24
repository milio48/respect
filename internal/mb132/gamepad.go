package mb132

// gamepad.go — Shim navigator.getGamepads() memakai XInput (Windows).
// Pure syscall, tanpa dependency eksternal.

import (
	"sync"
	"syscall"
	"unsafe"
)

type xinputGamepad struct {
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

type xinputState struct {
	PacketNumber uint32
	Gamepad      xinputGamepad
}

var (
	xinputOnce     sync.Once
	xinputGetState *syscall.LazyProc
)

func loadXInput() {
	xinputOnce.Do(func() {
		for _, name := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
			dll := syscall.NewLazyDLL(name)
			if err := dll.Load(); err != nil {
				continue
			}
			p := dll.NewProc("XInputGetState")
			if err := p.Find(); err != nil {
				continue
			}
			xinputGetState = p
			return
		}
	})
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func axis(v int16) float64 {
	f := float64(v) / 32767.0
	if f < -1 {
		f = -1
	}
	if f > 1 {
		f = 1
	}
	return f
}

// pollGamepads membaca state 4 slot XInput dan mengembalikan struktur ala Web Gamepad.
func pollGamepads() []map[string]interface{} {
	loadXInput()
	out := make([]map[string]interface{}, 4)
	for i := 0; i < 4; i++ {
		gp := map[string]interface{}{
			"index":     i,
			"connected": false,
			"id":        "XInput Controller",
			"mapping":   "standard",
			"axes":      []float64{0, 0, 0, 0},
			"buttons":   []map[string]interface{}{},
		}
		if xinputGetState != nil {
			var st xinputState
			r, _, _ := xinputGetState.Call(uintptr(i), uintptr(unsafe.Pointer(&st)))
			if r == 0 {
				gp["connected"] = true
				gp["axes"] = []float64{
					axis(st.Gamepad.ThumbLX),
					axis(st.Gamepad.ThumbLY),
					axis(st.Gamepad.ThumbRX),
					axis(st.Gamepad.ThumbRY),
				}
				btns := make([]map[string]interface{}, 17)
				for b := 0; b < 16; b++ {
					pressed := st.Gamepad.Buttons&(1<<uint(b)) != 0
					btns[b] = map[string]interface{}{"pressed": pressed, "value": boolToFloat(pressed)}
				}
				btns[6] = map[string]interface{}{"pressed": st.Gamepad.LeftTrigger > 30, "value": float64(st.Gamepad.LeftTrigger) / 255.0}
				btns[7] = map[string]interface{}{"pressed": st.Gamepad.RightTrigger > 30, "value": float64(st.Gamepad.RightTrigger) / 255.0}
				gp["buttons"] = btns
			}
		}
		out[i] = gp
	}
	return out
}
