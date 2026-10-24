package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

// The gallery's special-decision samples (web/src/lib/dev/gallery): for each
// table size and each special decision, the deciding seat's view and the
// view of the next seat, both from ViewFor at one point of a legal, seeded
// game. Nothing is constructed by hand, so the samples carry every view
// field, the kept preview and the discard search's reasons included.
var galleryCasesPath = filepath.Join("..", "..", "..", "..", "web", "src", "lib", "dev", "gallery", "haftgane-special-cases.json")

// galleryKinds is the order the gallery lists the decisions in.
var galleryKinds = []string{"side", "sixth", "discard", "kept", "last", "first-colour", "first-turn", "last-turn"}

type galleryCase struct {
	Kind    string     `json:"kind"`
	Seats   int        `json:"seats"`
	View    view.Table `json:"view"`
	Waiting view.Table `json:"waiting"`
}

// galleryKind reports whether seat s, seeing v, faces the decision want. A
// turn point counts only when the view shows what makes it special: a kept
// preview, or a free build of that kind. A discard search counts only when
// the pile holds a card the city may not build, so its reason shows.
func galleryKind(g Game, s Seat, v view.Table, want string) bool {
	p := g.Point()
	you := v.You
	switch want {
	case "side":
		return g.State.Phase == PhaseSides
	case "sixth":
		return g.State.Phase == PhaseTurn && g.State.Turn == 6 && slices.ContainsFunc(you.Choices, func(c view.Choice) bool { return c.Kept != nil })
	case "discard":
		return g.State.Phase == PhaseDiscardBuild && g.State.Decider == s && len(you.Disabled) > 0 && len(you.Choices) > 0
	case "kept":
		return g.State.Phase == PhaseDiscardBuild && !p.Required.Has(s) && you.Kept != ""
	case "last":
		return g.State.Phase == PhaseLastCard && g.State.Decider == s
	}
	return g.State.Phase == PhaseTurn && slices.ContainsFunc(you.Choices, func(c view.Choice) bool { return c.Free == want })
}

// waitingSeat is the first seat after s that the point does not wait for,
// or simply the next seat when every seat decides.
func waitingSeat(g Game, s Seat) Seat {
	n := Seat(g.State.Seats)
	for i := Seat(1); i < n; i++ {
		if w := (s + i) % n; !g.Point().Required.Has(w) {
			return w
		}
	}
	return (s + 1) % n
}

// galleryCases plays seeded games with sides chosen, preferring stages on
// even seeds so wonder abilities come up, and keeps the first point per
// table size and decision, scanning seats in order.
func galleryCases(t *testing.T) []galleryCase {
	t.Helper()
	var out []galleryCase
	for _, n := range []uint8{3, 5, 7} {
		found := map[string]galleryCase{}
		for seed := uint64(0); seed < 400 && len(found) < len(galleryKinds); seed++ {
			g, err := NewGame(Setup{Seats: n, Seed: seed, Options: Options{Sides: SidesChoose}})
			if err != nil {
				t.Fatal(err)
			}
			for g.State.Phase != PhaseOver {
				for s := Seat(0); s < Seat(n); s++ {
					v := g.ViewFor(s)
					for _, k := range galleryKinds {
						if _, ok := found[k]; ok || !galleryKind(g, s, v, k) {
							continue
						}
						found[k] = galleryCase{Kind: k, Seats: int(n), View: v, Waiting: g.ViewFor(waitingSeat(g, s))}
					}
				}
				if g, _, err = g.Resolve(observationBatch(g, seed)); err != nil {
					t.Fatal(err)
				}
			}
		}
		for _, k := range galleryKinds {
			c, ok := found[k]
			if !ok {
				t.Fatalf("no %s decision in %d-seat games", k, n)
			}
			out = append(out, c)
		}
	}
	return out
}

// TestGallerySpecialCases keeps the gallery's samples current: any change
// to the view fails it until they are regenerated. The comparison is by
// value, because Prettier formats the committed file.
func TestGallerySpecialCases(t *testing.T) {
	cases := galleryCases(t)
	for _, c := range cases {
		assertViewSlices(t, c.View)
		assertViewSlices(t, c.Waiting)
	}
	got, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("UPDATE_HAFTGANE_GALLERY") == "1" {
		if err := os.WriteFile(galleryCasesPath, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	committed, err := os.ReadFile(galleryCasesPath)
	if err != nil {
		t.Fatal(err)
	}
	var want, have any
	if err := json.Unmarshal(committed, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &have); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(have, want) {
		t.Fatal("the gallery's special-decision samples changed; regenerate with UPDATE_HAFTGANE_GALLERY=1 make test-go GO_PKGS=./internal/haftgane/rules GO_RUN=TestGallerySpecialCases, then run Prettier on", galleryCasesPath)
	}
}
