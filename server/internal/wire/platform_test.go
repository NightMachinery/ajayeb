package wire

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

var platformExamples = []string{
	`{"t":"table-pick","code":"REGAAA","seq":0,"point":{"id":"age-1-turn-1","revision":0},"key":"sell:clay-pool","confirm":false}`,
	`{"t":"table-undo","code":"REGAAA","seq":1,"base":0}`,
	`{"t":"table-seat","code":"REGAAA","seq":2,"seat":6,"control":"human","userId":"spectator"}`,
	`{"t":"table-seat","code":"REGAAA","seq":3,"seat":0,"control":"bot","bot":"easy"}`,
	`{"t":"table-role","code":"REGAAA","seq":4,"op":"promote","userId":"spectator"}`,
	`{"t":"table-role","code":"REGAAA","seq":5,"op":"step-down"}`,
	`{"t":"table-resign","code":"REGAAA","seq":6,"confirm":true}`,
	`{"t":"lobby-create","seatCount":7,"options":{"seating":"random","allowUndo":true,"finalReveal":false}}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":7,"op":"seats","seatCount":2}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":8,"op":"options","options":{"firstPlayer":0}}`,
	`{"t":"lobby-create","seatCount":3,"options":{"sides":"choose"}}`,
	`{"t":"lobby-create","seatCount":3,"options":{"sides":"day"}}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":8,"op":"options","options":{"sides":"random"}}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":9,"op":"add-bot","seat":6,"bot":"hard"}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":10,"op":"remove-bot","seat":0}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":11,"op":"seat","seat":0,"userId":"spectator"}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":12,"op":"remove-human","userId":"spectator"}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":13,"op":"release"}`,
	`{"t":"lobby-edit","code":"REGAAA","seq":14,"op":"open-reservation","seat":3}`,
	`{"t":"lobby-role","code":"REGAAA","seq":15,"op":"demote","userId":"spectator"}`,
	`{"t":"lobby-start","code":"REGAAA","seq":16,"fillBots":false}`,
	`{"t":"link","code":"REGAAA","seat":6}`,
	`{"t":"relink","code":"REGAAA","seat":6}`,
}

func TestPlatformStrictDecoders(t *testing.T) {
	for _, raw := range platformExamples {
		t.Run(raw, func(t *testing.T) {
			m, err := DecodeClientVersion([]byte(raw), PlatformProtocolVersion)
			if err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeClientVersion(b, PlatformProtocolVersion)
			if err != nil || !reflect.DeepEqual(m, again) {
				t.Fatalf("roundtrip %s: %v", b, err)
			}
			if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
				t.Fatal("new request accepted by v1 decoder")
			}
			// Exercise every required field, not just the nonzero semantic values.
			var fields map[string]json.RawMessage
			if err := json.Unmarshal([]byte(raw), &fields); err != nil {
				t.Fatal(err)
			}
			typ := reflect.TypeOf(m)
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				key := strings.Split(f.Tag.Get("json"), ",")[0]
				if f.Type.Kind() == reflect.Pointer || ((reflect.TypeOf(m) == reflect.TypeFor[PlatformLink]() || reflect.TypeOf(m) == reflect.TypeFor[PlatformRelink]()) && key == "seat") {
					continue
				}
				value := fields[key]
				delete(fields, key)
				bad, _ := json.Marshal(fields)
				if _, err := DecodeClientVersion(bad, 2); err == nil {
					t.Errorf("missing %s accepted", key)
				}
				fields[key] = json.RawMessage("null")
				bad, _ = json.Marshal(fields)
				if _, err := DecodeClientVersion(bad, 2); err == nil {
					t.Errorf("null %s accepted", key)
				}
				fields[key] = value
			}
			for _, bad := range []string{strings.TrimSuffix(raw, "}") + `,"unknown":1}`, raw + `{}`, strings.TrimSuffix(raw, "}") + `,"t":"table-pick"}`} {
				if _, err := DecodeClientVersion([]byte(bad), 2); !errors.Is(err, ErrBadFrame) {
					t.Errorf("accepted %s: %v", bad, err)
				}
			}
		})
	}
}

func TestPlatformInvalidValues(t *testing.T) {
	for _, raw := range []string{
		`{"t":"table-pick","code":"REGAAA","seq":-1,"point":{"id":"sides","revision":0},"key":"x","confirm":false}`,
		`{"t":"table-pick","code":"REGAAA","seq":0,"point":{"id":"sides"},"key":"x","confirm":false}`,
		`{"t":"table-pick","code":"REGAAA","seq":0,"point":{"id":"sides","revision":-1},"key":"x","confirm":false}`,
		`{"t":"table-pick","code":"REGAAA","seq":0,"point":{"id":"sides","revision":0,"seat":1},"key":"x","confirm":false}`,
		`{"t":"table-pick","code":"REGAAA","seq":0,"point":{"id":"sides","id":"sides","revision":0},"key":"x","confirm":false}`,
		`{"t":"table-undo","code":"REGAAA","seq":0,"base":-1}`,
		`{"t":"table-resign","code":"REGAAA","seq":0,"confirm":false}`,
		`{"t":"table-seat","code":"REGAAA","seq":0,"seat":7,"control":"bot","bot":"easy"}`,
		`{"t":"table-seat","code":"REGAAA","seq":0,"seat":0,"control":"human","bot":"easy"}`,
		`{"t":"table-seat","code":"REGAAA","seq":0,"seat":0,"control":"bot","bot":"unknown"}`,
		`{"t":"table-role","code":"REGAAA","seq":0,"op":"step-down","userId":"other"}`,
		`{"t":"table-role","code":"REGAAA","seq":0,"op":"promote"}`,
		`{"t":"table-role","code":"REGAAA","seq":0,"op":"unknown"}`,
		`{"t":"lobby-create","seatCount":1,"options":{}}`,
		`{"t":"lobby-create","seatCount":8,"options":{}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"firstPlayer":2}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"seating":"arbitrary"}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"sides":""}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"sides":"night"}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"sides":null}}`,
		`{"t":"lobby-edit","code":"REGAAA","seq":0,"op":"options","options":{"sides":"unknown"}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"allowUndo":null}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"AllowUndo":true}}`,
		`{"t":"lobby-create","seatCount":3,"options":{"seed":123}}`,
		`{"t":"lobby-edit","code":"REGAAA","seq":0,"op":"seats","seatCount":3,"seat":0}`,
		`{"t":"lobby-edit","code":"REGAAA","seq":0,"op":"add-bot","seat":0}`,
		`{"t":"lobby-edit","code":"REGAAA","seq":0,"op":"release","userId":"other"}`,
		`{"t":"lobby-start","code":"","seq":0,"fillBots":true}`,
		`{"t":"table-state","code":"REGAAA"}`,
		`{"t":"link","code":"REGAAA","seat":7}`,
	} {
		if _, err := DecodeClientVersion([]byte(raw), 2); !errors.Is(err, ErrBadFrame) {
			t.Errorf("accepted %s: %v", raw, err)
		}
	}
	tooLong := strings.Replace(platformExamples[0], "sell:clay-pool", strings.Repeat("x", MaxClientFrameBytes), 1)
	if _, err := DecodeClientVersion([]byte(tooLong), 2); !errors.Is(err, ErrBadFrame) {
		t.Fatal("accepted oversized platform frame")
	}
	if _, err := DecodeClientVersion([]byte(platformExamples[0]), 3); !errors.Is(err, ErrBadFrame) {
		t.Fatal("accepted unnegotiated version")
	}
}

func TestPlatformLeavesDuelRequestsUnchanged(t *testing.T) {
	for _, m := range clientMessageTypes {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		v1, err1 := DecodeClientVersion(b, 1)
		v2, err2 := DecodeClientVersion(b, 2)
		if (err1 == nil) != (err2 == nil) || !reflect.DeepEqual(v1, v2) {
			t.Fatalf("Duel decode differs: %s: %v / %v", b, err1, err2)
		}
	}
	// Historical link seat default is part of the one Duel request contract.
	for _, raw := range []string{`{"t":"link","code":"REGAAA"}`, `{"t":"relink","code":"REGAAA","seat":0}`} {
		a, e := DecodeClientVersion([]byte(raw), 1)
		b, f := DecodeClientVersion([]byte(raw), 2)
		if e != nil || f != nil || !reflect.DeepEqual(a, b) {
			t.Fatal("Duel link defaults changed")
		}
	}
}

func TestPlatformServerEnvelopes(t *testing.T) {
	frames := []ServerMessage{NewTableState("REGAAA"), NewLobbyState("REGAAA"), NewTableReject(2, "decision-closed"), NewLobbyReject(3, "not-admin"), PlatformLinked{T: "linked", Code: "REGAAA", Seat: 6, Claim: "private"}}
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
			walk(typ.Elem())
			return
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
			if !f.IsExported() || name == "" || (f.Type.Kind() == reflect.Pointer) != (opts == "omitempty") {
				t.Errorf("invalid DTO field %s.%s", typ, f.Name)
			}
			walk(f.Type)
		}
	}
	for _, m := range frames {
		walk(reflect.TypeOf(m))
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "null") {
			t.Fatalf("nil slice in %s", b)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}
		want, ok := discriminator(reflect.TypeOf(m).Field(0).Tag.Get("tstype"))
		if !ok || string(raw["t"]) != `"`+want+`"` {
			t.Fatalf("bad discriminator: %s", b)
		}
	}
	// The table envelope preserves exactly these fields around the real engine DTOs.
	for typ, want := range map[reflect.Type]string{
		reflect.TypeFor[TableView]():  "phase age turn pass pile seats you result reveal canDownload",
		reflect.TypeFor[TableEvent](): "kind seat other age turn play design stage side left right printed delta source outcome pass",
	} {
		var got []string
		for i := 0; i < typ.NumField(); i++ {
			name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
			got = append(got, name)
		}
		if strings.Join(got, " ") != want {
			t.Fatalf("%s fields %v, want %s", typ, got, want)
		}
	}
}

func TestSharedLobbyDuelHintOptionsDecoder(t *testing.T) {
	hints, err := json.Marshal(view.HintSettings{Prices: true, AdvisorStrength: "twoStage"})
	if err != nil {
		t.Fatal(err)
	}
	base := `{"t":"lobby-create","seatCount":2,"options":{"hints":` + string(hints) + `}}`
	message, err := DecodeClientVersion([]byte(base), PlatformProtocolVersion)
	if err != nil || message.(LobbyCreate).Options.Hints == nil {
		t.Fatalf("complete hints refused: %v", err)
	}
	cases := []string{
		strings.Replace(base, `"counting":false,`, "", 1),
		strings.Replace(base, `"prices":true`, `"prices":null`, 1),
		strings.Replace(base, `"prices":true`, `"prices":true,"prices":false`, 1),
		strings.Replace(base, `"prices":true`, `"unknown":true`, 1),
		strings.Replace(base, `"twoStage"`, `"unbounded"`, 1),
	}
	for _, raw := range cases {
		if _, err := DecodeClientVersion([]byte(raw), PlatformProtocolVersion); err == nil {
			t.Fatalf("invalid hints accepted: %s", raw)
		}
	}
}
