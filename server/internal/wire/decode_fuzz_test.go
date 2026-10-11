package wire

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// FuzzDecodeClient feeds arbitrary bytes to DecodeClient, the one gate every
// client frame passes before the server acts on it. It must never panic; a
// refusal must wrap ErrBadFrame; and a frame it accepts must be one of the
// client frame types whose "t" names it, must pass the same checks again
// once re-encoded (so validation does not depend on how the JSON was
// spelled), and must decode back to the same value.
//
// The seeds are the frames of TestDecodeClientAccepts and a few near
// misses. The nightly CI job runs this target (.github/workflows/ci.yml);
// a crasher lands in testdata/fuzz/FuzzDecodeClient, and committing it turns
// it into a regression test.
func FuzzDecodeClient(f *testing.F) {
	tok := "AAAAAAAAAAAAAAAAAAAAAA" // 22 characters, the shape of a token
	seeds := []string{
		`{"t":"hello","v":1,"token":"` + tok + `","name":"Sara","clientId":"c1"}`,
		`{"t":"create","mode":"online","seat":"random","names":["Sara"]}`,
		`{"t":"create","mode":"ai","bot":"hard","seat":0,"names":["Sara"]}`,
		`{"t":"create","mode":"hotseat","names":["Sara","Kian"]}`,
		`{"t":"join","code":"K7QX2M"}`,
		`{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":"construct:baths"}`,
		`{"t":"act","code":"K7QX2M","seq":1,"base":0,"key":"construct:baths","seat":1}`,
		`{"t":"undo","code":"K7QX2M","seq":2,"base":1}`,
		`{"t":"resync","code":"K7QX2M"}`,
		`{"t":"leave","code":"K7QX2M"}`,
		`{"t":"rematch","code":"K7QX2M"}`,
		`{"t":"join","code":"K7QX2M"} {}`,
		`{"t":"act","code":"K7QX2M","seq":-1,"base":0,"key":"x"}`,
		`{"t":"hello","v":1,"token":"` + tok + `","name":"a\u202eb","clientId":"c1"}`,
		`{"t":"nonsense"}`,
		`[]`,
		``,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		m, err := DecodeClient(raw)
		if err != nil {
			if !errors.Is(err, ErrBadFrame) {
				t.Fatalf("refusal %v does not wrap ErrBadFrame", err)
			}
			if m != nil {
				t.Fatalf("a refused frame also returned %T", m)
			}
			return
		}
		if m == nil {
			t.Fatalf("an accepted frame returned no message")
		}
		want, err := peekType(raw)
		if err != nil {
			t.Fatalf("an accepted frame has no readable type: %v", err)
		}
		types := map[string]reflect.Type{
			"hello":   reflect.TypeFor[Hello](),
			"create":  reflect.TypeFor[Create](),
			"join":    reflect.TypeFor[Join](),
			"act":     reflect.TypeFor[Act](),
			"undo":    reflect.TypeFor[Undo](),
			"resync":  reflect.TypeFor[Resync](),
			"leave":   reflect.TypeFor[Leave](),
			"rematch": reflect.TypeFor[Rematch](),
		}
		if got := reflect.TypeOf(m); types[want] != got {
			t.Fatalf("frame %q decoded as %v", want, got)
		}

		again, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("re-encoding %#v: %v", m, err)
		}
		back, err := DecodeClient(again)
		if err != nil {
			t.Fatalf("the re-encoded frame %s is refused: %v", again, err)
		}
		if !reflect.DeepEqual(back, m) {
			t.Fatalf("round trip changed the frame:\n got %#v\nwant %#v", back, m)
		}
	})
}
