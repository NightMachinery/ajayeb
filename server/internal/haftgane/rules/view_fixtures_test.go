package rules

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

// These are raw game-body JSON fixtures, ready for the platform's RawMessage
// envelope. They carry no setup, fingerprint, room metadata or display text.
func TestHaftganeViewFixtures(t *testing.T) {
	fixtures := map[string]view.Table{}
	for seed := uint64(0); seed < 100 && len(fixtures) < 7; seed++ {
		g, err := NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SidesChoose}})
		if err != nil {
			t.Fatal(err)
		}
		for {
			switch g.State.Phase {
			case PhaseSides:
				if _, ok := fixtures["sides"]; !ok {
					fixtures["sides"] = g.ViewFor(0)
				}
			case PhaseTurn:
				if _, ok := fixtures["turn-seat"]; !ok {
					fixtures["turn-seat"], fixtures["turn-spectator"] = g.ViewFor(0), g.ViewFor(NoSeat)
				}
			case PhaseDiscardBuild:
				if _, ok := fixtures["discard-seat"]; !ok {
					fixtures["discard-seat"] = g.ViewFor(g.State.Decider)
				}
			case PhaseLastCard:
				if _, ok := fixtures["last-seat"]; !ok {
					fixtures["last-seat"] = g.ViewFor(g.State.Decider)
				}
			case PhaseOver:
				if _, ok := fixtures["over-reveal-spectator"]; !ok {
					fixtures["over-reveal-spectator"] = g.ViewFor(NoSeat)
					hidden, err := NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SidesChoose, HideAtEnd: true}})
					if err != nil {
						t.Fatal(err)
					}
					for hidden.State.Phase != PhaseOver {
						hidden, _, err = hidden.Resolve(observationBatch(hidden, seed))
						if err != nil {
							t.Fatal(err)
						}
					}
					fixtures["over-hidden-seat"] = hidden.ViewFor(0)
				}
			}
			if g.State.Phase == PhaseOver {
				break
			}
			g, _, err = g.Resolve(observationBatch(g, seed))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(fixtures) != 7 {
		t.Fatal("missing fixture phases", len(fixtures))
	}
	for name, v := range fixtures {
		assertViewSlices(t, v)
		encoded, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		encoded = append(encoded, '\n')
		path := filepath.Join("testdata", "views", name+".json")
		if os.Getenv("UPDATE_HAFTGANE_VIEWS") == "1" {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, encoded, 0644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, want) {
			t.Fatalf("%s changed; review and regenerate with UPDATE_HAFTGANE_VIEWS=1 make test-go GO_PKGS=./internal/haftgane/rules GO_RUN=TestHaftganeViewFixtures", path)
		}
		var roundTrip view.Table
		if err := json.Unmarshal(want, &roundTrip); err != nil {
			t.Fatal("platform fixture decode", err)
		}
		again, err := json.Marshal(roundTrip)
		if err != nil || !bytes.Equal(again, bytes.TrimSpace(want)) {
			t.Fatal("fixture DTO round trip", name, err)
		}
		t.Logf("%s %d bytes", name, len(encoded)-1)
	}
}
