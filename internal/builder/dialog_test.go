package builder

import (
	"encoding/json"
	"errors"
	"testing"
)

// pickIconResponse adalah bentuk respons yang dikirim ke frontend.
type pickIconResponse struct {
	OK        bool   `json:"ok"`
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
	Error     string `json:"error"`
}

func decodePickIconResponse(t *testing.T, resp string) pickIconResponse {
	t.Helper()

	var res pickIconResponse
	if err := json.Unmarshal([]byte(resp), &res); err != nil {
		t.Fatalf("respons bukan JSON valid: %v (%s)", err, resp)
	}
	return res
}

func TestParseCommand(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"perintah pick-icon", `{"cmd":"pick-icon"}`, cmdPickIcon},
		{"perintah dengan spasi", `{"cmd":" pick-icon "}`, cmdPickIcon},
		{"payload build lama tanpa cmd", `{"mode":"url","source":"https://example.com","out_name":"demo.exe"}`, ""},
		{"payload kosong", ``, ""},
		{"bukan json", `bukan-json`, ""},
	}

	for _, tc := range cases {
		if got := parseCommand(tc.input); got != tc.want {
			t.Errorf("%s: parseCommand(%q) = %q, mau %q", tc.name, tc.input, got, tc.want)
		}
	}
}

func TestPickIconJSONSukses(t *testing.T) {
	original := pickIconFn
	defer func() { pickIconFn = original }()

	var gotOwner uintptr
	pickIconFn = func(owner uintptr) (string, error) {
		gotOwner = owner
		return `C:\icons\app.ico`, nil
	}

	res := decodePickIconResponse(t, pickIconJSON(4242))

	if gotOwner != 4242 {
		t.Errorf("owner HWND tidak diteruskan: dapat %d", gotOwner)
	}
	if !res.OK || res.Cancelled {
		t.Errorf("status tidak sesuai: %+v", res)
	}
	if res.Path != `C:\icons\app.ico` {
		t.Errorf("path tidak sesuai: %q", res.Path)
	}
	if res.Error != "" {
		t.Errorf("error seharusnya kosong: %q", res.Error)
	}
}

func TestPickIconJSONDibatalkan(t *testing.T) {
	original := pickIconFn
	defer func() { pickIconFn = original }()

	pickIconFn = func(owner uintptr) (string, error) {
		return "", nil
	}

	res := decodePickIconResponse(t, pickIconJSON(0))

	if !res.OK || !res.Cancelled {
		t.Errorf("pembatalan harus ok=true dan cancelled=true: %+v", res)
	}
	if res.Path != "" {
		t.Errorf("path harus kosong saat dibatalkan: %q", res.Path)
	}
}

func TestPickIconJSONGagal(t *testing.T) {
	original := pickIconFn
	defer func() { pickIconFn = original }()

	pickIconFn = func(owner uintptr) (string, error) {
		return "", errors.New("dialog tidak tersedia")
	}

	res := decodePickIconResponse(t, pickIconJSON(0))

	if res.OK {
		t.Errorf("kegagalan harus ok=false: %+v", res)
	}
	if res.Error != "dialog tidak tersedia" {
		t.Errorf("pesan error tidak diteruskan: %q", res.Error)
	}
}
