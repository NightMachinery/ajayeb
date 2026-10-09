package wire

import (
	"errors"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
)

func validToken(t *testing.T) string {
	t.Helper()
	tok, err := ident.NewToken()
	if err != nil {
		t.Fatalf("ident.NewToken: %v", err)
	}
	return tok
}

func TestDecodeClientAccepts(t *testing.T) {
	tok := validToken(t)
	cases := map[string]string{
		"hello":       `{"t":"hello","v":1,"token":"` + tok + `","name":"Sara","clientId":"c1"}`,
		"create":      `{"t":"create","mode":"online","seat":"random","names":["Sara"]}`,
		"create ai":   `{"t":"create","mode":"ai","bot":"hard","seat":0,"names":["Sara"]}`,
		"join":        `{"t":"join","code":"K7QX2M"}`,
		"act":         `{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":"construct:baths"}`,
		"act hotseat": `{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":"construct:baths","seat":1}`,
		"undo":        `{"t":"undo","code":"K7QX2M","seq":2,"base":1}`,
		"resync":      `{"t":"resync","code":"K7QX2M"}`,
		"leave":       `{"t":"leave","code":"K7QX2M"}`,
		"rematch":     `{"t":"rematch","code":"K7QX2M"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeClient([]byte(raw)); err != nil {
				t.Fatalf("DecodeClient(%s): %v", raw, err)
			}
		})
	}
}

func TestDecodeClientRoundTripsType(t *testing.T) {
	m, err := DecodeClient([]byte(`{"t":"join","code":"K7QX2M"}`))
	if err != nil {
		t.Fatalf("DecodeClient: %v", err)
	}
	join, ok := m.(Join)
	if !ok {
		t.Fatalf("DecodeClient returned %T, want Join", m)
	}
	if join.Code != "K7QX2M" {
		t.Fatalf("Code = %q, want K7QX2M", join.Code)
	}
}

func TestDecodeClientRejectsUnknownType(t *testing.T) {
	_, err := DecodeClient([]byte(`{"t":"nonsense"}`))
	if !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient of an unknown t: err = %v, want ErrBadFrame", err)
	}
}

func TestDecodeClientRejectsMissingType(t *testing.T) {
	_, err := DecodeClient([]byte(`{"code":"K7QX2M"}`))
	if !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient with no t: err = %v, want ErrBadFrame", err)
	}
}

func TestDecodeClientRejectsOversize(t *testing.T) {
	huge := `{"t":"join","code":"` + strings.Repeat("A", MaxClientFrameBytes) + `"}`
	_, err := DecodeClient([]byte(huge))
	if !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient of an oversize frame: err = %v, want ErrBadFrame", err)
	}
}

func TestDecodeClientRejectsUnknownFields(t *testing.T) {
	_, err := DecodeClient([]byte(`{"t":"join","code":"K7QX2M","extra":"field"}`))
	if !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient with an unknown field: err = %v, want ErrBadFrame", err)
	}
}

func TestDecodeClientRejectsTrailingData(t *testing.T) {
	_, err := DecodeClient([]byte(`{"t":"join","code":"K7QX2M"} {"t":"leave","code":"K7QX2M"}`))
	if !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient with trailing data: err = %v, want ErrBadFrame", err)
	}
}

func TestDecodeClientRejectsMalformedJSON(t *testing.T) {
	_, err := DecodeClient([]byte(`{"t":"join"`))
	if !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient of truncated JSON: err = %v, want ErrBadFrame", err)
	}
}

func TestDecodeClientRejectsBadHello(t *testing.T) {
	cases := map[string]string{
		"short token":  `{"t":"hello","v":1,"token":"abc","name":"","clientId":"c1"}`,
		"empty client": `{"t":"hello","v":1,"token":"` + validToken(t) + `","name":"","clientId":""}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
				t.Fatalf("DecodeClient(%s): err = %v, want ErrBadFrame", raw, err)
			}
		})
	}
}

func TestDecodeClientRejectsBadCreate(t *testing.T) {
	cases := map[string]string{
		"unknown mode":      `{"t":"create","mode":"weird","seat":"random","names":[]}`,
		"ai without bot":    `{"t":"create","mode":"ai","seat":"random","names":[]}`,
		"online with bot":   `{"t":"create","mode":"online","bot":"hard","seat":"random","names":[]}`,
		"unknown bot level": `{"t":"create","mode":"ai","bot":"extreme","seat":0,"names":[]}`,
		"bad seat string":   `{"t":"create","mode":"online","seat":"left","names":[]}`,
		"bad seat number":   `{"t":"create","mode":"online","seat":2,"names":[]}`,
		"too many names":    `{"t":"create","mode":"online","seat":"random","names":["A","B","C"]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
				t.Fatalf("DecodeClient(%s): err = %v, want ErrBadFrame", raw, err)
			}
		})
	}
}

func TestDecodeClientRejectsEmptyCode(t *testing.T) {
	for _, raw := range []string{
		`{"t":"join","code":""}`,
		`{"t":"leave","code":""}`,
		`{"t":"resync","code":""}`,
		`{"t":"rematch","code":""}`,
		`{"t":"act","code":"","seq":1,"base":0,"key":"construct:baths"}`,
		`{"t":"undo","code":"","seq":1,"base":0}`,
	} {
		if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
			t.Fatalf("DecodeClient(%s): err = %v, want ErrBadFrame", raw, err)
		}
	}
}

func TestDecodeClientRejectsBadAct(t *testing.T) {
	cases := map[string]string{
		"empty key":     `{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":""}`,
		"negative seq":  `{"t":"act","code":"K7QX2M","seq":-1,"base":0,"key":"construct:baths"}`,
		"negative base": `{"t":"act","code":"K7QX2M","seq":1,"base":-1,"key":"construct:baths"}`,
		"bad seat":      `{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":"construct:baths","seat":2}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
				t.Fatalf("DecodeClient(%s): err = %v, want ErrBadFrame", raw, err)
			}
		})
	}
}

func TestSeatChoiceJSONRoundTrip(t *testing.T) {
	data, err := SeatRandom.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `"random"` {
		t.Fatalf("SeatRandom marshaled as %s, want \"random\"", data)
	}

	for _, seat := range []int{0, 1} {
		data, err := SeatNumber(seat).MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}
		var c SeatChoice
		if err := c.UnmarshalJSON(data); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", data, err)
		}
		got, ok := c.Seat()
		if !ok || got != seat {
			t.Fatalf("round trip of seat %d: got (%d, %v)", seat, got, ok)
		}
	}

	for _, bad := range []string{`"left"`, `2`, `1.5`, `null`, `true`} {
		var c SeatChoice
		if err := c.UnmarshalJSON([]byte(bad)); err == nil {
			t.Errorf("UnmarshalJSON(%s): want error, got nil", bad)
		}
	}
}

func TestSeatChoiceRandomFlag(t *testing.T) {
	if !SeatRandom.Random() {
		t.Errorf("SeatRandom.Random() = false, want true")
	}
	if SeatNumber(0).Random() {
		t.Errorf("SeatNumber(0).Random() = true, want false")
	}
}

func TestDecodeClientRejectsControlCharacters(t *testing.T) {
	raw := `{"t":"hello","v":1,"token":"` + validToken(t) + `","name":"Sa\u0007ra","clientId":"c1"}`
	if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
		t.Fatalf("DecodeClient with a control character in name: err = %v, want ErrBadFrame", err)
	}
}

func TestDiscriminatorHelper(t *testing.T) {
	if got, ok := discriminator("'hello'"); !ok || got != "hello" {
		t.Errorf("discriminator(%q) = (%q, %v), want (hello, true)", "'hello'", got, ok)
	}
	for _, bad := range []string{"hello", "'hello", "hello'", "", "'"} {
		if _, ok := discriminator(bad); ok {
			t.Errorf("discriminator(%q): want ok=false", bad)
		}
	}
}

func TestDecodeClientRejectsOverlongFields(t *testing.T) {
	cases := map[string]string{
		"name":     `{"t":"hello","v":1,"token":"` + validToken(t) + `","name":"` + strings.Repeat("x", maxNameLen+1) + `","clientId":"c1"}`,
		"clientId": `{"t":"hello","v":1,"token":"` + validToken(t) + `","name":"","clientId":"` + strings.Repeat("x", maxClientIDLen+1) + `"}`,
		"key":      `{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":"` + strings.Repeat("x", maxActionKeyLen+1) + `"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
				t.Fatalf("DecodeClient(%s): err = %v, want ErrBadFrame", raw, err)
			}
		})
	}
}
