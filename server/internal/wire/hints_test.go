package wire

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// allHints is a settings object with every family, as JSON, with the given
// families switched off.
func allHints(off ...string) string {
	parts := make([]string, 0, len(hintFamilies))
	for _, f := range hintFamilies {
		on := "true"
		for _, o := range off {
			if o == string(f) {
				on = "false"
			}
		}
		parts = append(parts, `"`+string(f)+`":`+on)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func TestDecodeHints(t *testing.T) {
	m, err := DecodeClient([]byte(`{"t":"hints","code":"K7QX2M","seq":3,"hints":` + allHints("prices", "advisor") + `}`))
	if err != nil {
		t.Fatalf("DecodeClient: %v", err)
	}
	h, ok := m.(Hints)
	if !ok {
		t.Fatalf("DecodeClient returned %T, want Hints", m)
	}
	if h.Code != "K7QX2M" || h.Seq != 3 || h.Hints.Prices || h.Hints.Advisor || !h.Hints.Economy || !h.Hints.Recap {
		t.Fatalf("decoded %+v", h)
	}

	c, err := DecodeClient([]byte(`{"t":"create","mode":"hotseat","seat":0,"names":[],"hints":` + allHints("tempo") + `}`))
	if err != nil {
		t.Fatalf("DecodeClient(create with hints): %v", err)
	}
	if cr := c.(Create); cr.Hints == nil || cr.Hints.Tempo || !cr.Hints.Prices {
		t.Fatalf("create hints %+v", cr.Hints)
	}
	c, err = DecodeClient([]byte(`{"t":"create","mode":"hotseat","seat":0,"names":[]}`))
	if err != nil || c.(Create).Hints != nil {
		t.Fatalf("create without hints: %+v, %v", c, err)
	}

	// The advisor's strength: one of the strengths, or left out (empty).
	withStrength := strings.Replace(allHints(), `}`, `,"advisorStrength":"deep"}`, 1)
	m, err = DecodeClient([]byte(`{"t":"hints","code":"K7QX2M","seq":4,"hints":` + withStrength + `}`))
	if err != nil || m.(Hints).Hints.AdvisorStrength != view.AdvisorStrengthDeep {
		t.Fatalf("hints with a strength: %+v, %v", m, err)
	}
	c, err = DecodeClient([]byte(`{"t":"create","mode":"hotseat","seat":0,"names":[],"hints":` + withStrength + `}`))
	if err != nil || c.(Create).Hints.AdvisorStrength != view.AdvisorStrengthDeep {
		t.Fatalf("create with a strength: %+v, %v", c, err)
	}
	if m.(Hints).Hints.AdvisorStrength != view.AdvisorStrengthDeep || h.Hints.AdvisorStrength != "" {
		t.Fatalf("a frame without a strength decodes it as %q", h.Hints.AdvisorStrength)
	}
}

func TestDecodeHintsRejects(t *testing.T) {
	missing := strings.Replace(allHints(), `"advisor":true`, ``, 1)
	missing = strings.Replace(missing, `,}`, `}`, 1)
	cases := map[string]string{
		"missing family":        `{"t":"hints","code":"K7QX2M","seq":1,"hints":` + missing + `}`,
		"null family":           `{"t":"hints","code":"K7QX2M","seq":1,"hints":` + strings.Replace(allHints(), `"recap":true`, `"recap":null`, 1) + `}`,
		"unknown family":        `{"t":"hints","code":"K7QX2M","seq":1,"hints":` + strings.Replace(allHints(), `{`, `{"hints":true,`, 1) + `}`,
		"no hints":              `{"t":"hints","code":"K7QX2M","seq":1}`,
		"null hints":            `{"t":"hints","code":"K7QX2M","seq":1,"hints":null}`,
		"not a bool":            `{"t":"hints","code":"K7QX2M","seq":1,"hints":` + strings.Replace(allHints(), `"recap":true`, `"recap":1`, 1) + `}`,
		"negative seq":          `{"t":"hints","code":"K7QX2M","seq":-1,"hints":` + allHints() + `}`,
		"no code":               `{"t":"hints","code":"","seq":1,"hints":` + allHints() + `}`,
		"extra field":           `{"t":"hints","code":"K7QX2M","seq":1,"base":0,"hints":` + allHints() + `}`,
		"create missing":        `{"t":"create","mode":"online","seat":0,"names":[],"hints":` + missing + `}`,
		"create wrong case":     `{"t":"create","mode":"online","seat":0,"names":[],"hints":` + strings.Replace(allHints(), `"prices"`, `"Prices"`, 1) + `}`,
		"create hints a list":   `{"t":"create","mode":"online","seat":0,"names":[],"hints":[]}`,
		"create hints unknown":  `{"t":"create","mode":"online","seat":0,"names":[],"hints":` + strings.Replace(allHints(), `{`, `{"x":true,`, 1) + `}`,
		"unknown strength":      `{"t":"hints","code":"K7QX2M","seq":1,"hints":` + strings.Replace(allHints(), `}`, `,"advisorStrength":"slow"}`, 1) + `}`,
		"strength not a string": `{"t":"hints","code":"K7QX2M","seq":1,"hints":` + strings.Replace(allHints(), `}`, `,"advisorStrength":2}`, 1) + `}`,
		"create bad strength":   `{"t":"create","mode":"online","seat":0,"names":[],"hints":` + strings.Replace(allHints(), `}`, `,"advisorStrength":"Deep"}`, 1) + `}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeClient([]byte(raw)); !errors.Is(err, ErrBadFrame) {
				t.Fatalf("DecodeClient(%s): err = %v, want ErrBadFrame", raw, err)
			}
		})
	}
}

func TestDiffHints(t *testing.T) {
	var prev view.HintSettings
	prev.Prices, prev.Science = true, true
	next := prev
	SetHint(&next, HintFamilyPrices, false)
	SetHint(&next, HintFamilyAdvisor, true)
	SetHint(&next, HintFamilyEconomy, true)
	on, off := DiffHints(prev, next)
	if len(on) != 2 || on[0] != HintFamilyEconomy || on[1] != HintFamilyAdvisor || len(off) != 1 || off[0] != HintFamilyPrices {
		t.Fatalf("DiffHints: on %v, off %v", on, off)
	}
	on, off = DiffHints(next, next)
	if on == nil || off == nil || len(on)+len(off) != 0 {
		t.Fatalf("DiffHints of equal settings: on %#v, off %#v", on, off)
	}
	if HintOn(next, "nonsense") {
		t.Fatalf("an unknown family is on")
	}
	SetHint(&next, "nonsense", true) // changes nothing, and does not panic
}

// TestStateHintFieldsJSON pins the hint fields of a state frame on the wire:
// room.hints and you.admin always present, and roomLog an array, never null.
func TestStateHintFieldsJSON(t *testing.T) {
	st := NewState("K7QX2M", 3, NewRoomInfo(ModeOnline, RoomStatusLobby, nil, 0), NewYouInfo([]int{0}, false), nil, nil, view.View{})
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	for _, want := range []string{`"hints":{"prices":false,`, `"admin":false`, `"roomLog":[]`} {
		if !strings.Contains(s, want) {
			t.Errorf("frame lacks %s: %s", want, s)
		}
	}
	seat := 1
	st.RoomLog = []RoomLogEntry{{Kind: RoomLogHints, At: 4, Seat: &seat, On: []HintFamily{}, Off: []HintFamily{HintFamilyPrices}}}
	data, err = json.Marshal(st)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if want := `"roomLog":[{"kind":"hints","at":4,"seat":1,"on":[],"off":["prices"]}]`; !strings.Contains(string(data), want) {
		t.Errorf("frame lacks %s: %s", want, data)
	}
}
