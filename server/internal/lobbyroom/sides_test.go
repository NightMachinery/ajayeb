package lobbyroom

import (
	"encoding/json"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Missing sides is the effective day default, including lobby records
// written before this additive field existed. All choices survive a cold
// load and reach the actual table's first engine point through Start.
func TestLobbySidesStartAndRestore(t *testing.T) {
	for _, rule := range []string{"", "day", "random", "choose"} {
		t.Run(rule, func(t *testing.T) {
			f := newFixture(t, nil)
			k := f.conn(user(0), "c0", "P0")
			opts := wire.LobbyOptions{Seating: ptr("lobby")}
			if rule != "" {
				opts.Sides = ptr(rule)
			}
			code := f.create(k, 3, opts)
			f.restart()
			f.join(k, code)
			restored := k.lobby("restored lobby", any2)
			if rule == "" && restored.Options.Sides != nil {
				t.Fatal("old absent value changed")
			}
			if rule != "" && (restored.Options.Sides == nil || *restored.Options.Sides != rule) {
				t.Fatal("side rule lost on restore")
			}
			f.send(k, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
			st := k.table("started real table", func(wire.TableState) bool { return true })
			r := f.tableRecord(code)
			want := rule
			if want == "" {
				want = "day"
			}
			if r.Setup.Sides != want {
				t.Fatalf("setup side rule %q, want %q", r.Setup.Sides, want)
			}
			if rule == "choose" {
				if st.Point.Kind != "sides" || len(st.View.You.Choices) != 2 || len(st.View.You.Hand) != 0 {
					t.Fatal("missing side point")
				}
			} else if st.Point.Kind != "turn" || len(st.View.You.Hand) != 7 {
				t.Fatal("day/random should open Age I")
			}
			if st.History == nil || len(*st.History) != 1 || (*st.History)[0].Batch != 0 {
				t.Fatal("initial history numbering")
			}
		})
	}
}

func TestLobbySidesValidationAndCloning(t *testing.T) {
	for _, rule := range []string{"choose", "day", "random"} {
		o := wire.LobbyOptions{Sides: ptr(rule)}
		c := cloneOptions(o)
		*o.Sides = "invalid"
		if c.Sides == o.Sides || *c.Sides != rule || !validOptions(c) {
			t.Fatal("side pointer not cloned")
		}
	}
	for _, rule := range []string{"", "night", "invalid"} {
		if validOptions(wire.LobbyOptions{Sides: ptr(rule)}) {
			t.Fatal("invalid side rule accepted", rule)
		}
	}
	// Explicitly decode old option bytes, rather than assuming a nil field
	// represents a compatible stored value.
	var old wire.LobbyOptions
	if err := json.Unmarshal([]byte(`{"seating":"lobby"}`), &old); err != nil || !validOptions(old) || old.Sides != nil {
		t.Fatal("old lobby options no longer load", err)
	}
}
