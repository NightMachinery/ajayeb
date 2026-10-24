package tableroom

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// privateFieldScene is a real seven-seat table opened at a point where one
// seat's private section carries a field the screens asked for.
type privateFieldScene struct {
	seed   uint64
	prefix []BatchRecord
	viewer int
}

// findPrivateFieldScenes plays random-side games, the Babylon and
// Halikarnassos seats preferring their stages, and returns the first point
// where a seat previews a kept card on its sixth turn and the first pile
// search with a card the searching city cannot build.
func findPrivateFieldScenes(t *testing.T) (kept, pile privateFieldScene) {
	t.Helper()
	found := func(s privateFieldScene) bool { return s.prefix != nil }
	for seed := uint64(1); seed < 400 && (!found(kept) || !found(pile)); seed++ {
		su := makeDaySetup(seed, 7, false, false)
		su.Options.Sides = rules.SidesRandom
		g, err := newGame(su)
		if err != nil {
			t.Fatal(err)
		}
		prefix := []BatchRecord{}
		for !pointOver(g.Point()) && (!found(kept) || !found(pile)) {
			st := &g.State
			for s := range int(st.Seats) {
				if found(kept) || st.Phase != rules.PhaseTurn || st.Turn != rules.TurnsPerAge {
					break
				}
				if slices.ContainsFunc(g.ViewFor(rules.Seat(s)).You.Choices, func(c wire.TableChoice) bool { return c.Kept != nil }) {
					kept = privateFieldScene{seed, slices.Clone(prefix), s}
				}
			}
			if !found(pile) && st.Phase == rules.PhaseDiscardBuild && len(g.ViewFor(st.Decider).You.Disabled) > 0 {
				pile = privateFieldScene{seed, slices.Clone(prefix), int(st.Decider)}
			}
			pt := g.Point()
			b := batch{Point: pt.ID, Picks: []pick{}}
			for _, seat := range pt.Required.Seats() {
				cs := g.Legal(seat)
				c := cs[0]
				if board := st.Cities[seat].Board; board == catalog.BoardBabylon || board == catalog.BoardHalikarnassos {
					if i := slices.IndexFunc(cs, func(c rules.Choice) bool { return c.Kind == rules.ActStage }); i >= 0 {
						c = cs[i]
					}
				}
				b.Picks = append(b.Picks, pick{Seat: seat, Key: c.Key()})
			}
			next, _, err := g.Resolve(b)
			if err != nil {
				t.Fatal(err)
			}
			g = next
			prefix = append(prefix, recordBatches([]tabletest.Batch{driverBatch(b)})...)
		}
	}
	if !found(kept) || !found(pile) {
		t.Fatal("no table reached a kept-card preview and a barred pile card", found(kept), found(pile))
	}
	return kept, pile
}

// openScene opens the scene's table and returns every seat's latest frame
// and the truth at that point.
func openScene(t *testing.T, sc privateFieldScene) ([]wire.TableState, *leakcheck.Truth) {
	t.Helper()
	sp := humans(7, sc.seed)
	sp.Sides = "random"
	sp.Prefix = sc.prefix
	h := newTH(t, Config{BotWork: blockBots}, sp)
	a, err := h.p.actor()
	if err != nil {
		t.Fatal(err)
	}
	frames := []wire.TableState{}
	for s := range 7 {
		id := tabletest.SeatClient(s)
		if err := a.call(h.ctx, func() { a.inbound(h.p.clients[id], wire.NewResync(h.p.code)) }); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, h.latest(id))
	}
	su := makeDaySetup(sc.seed, 7, false, false)
	su.Options.Sides = rules.SidesRandom
	truth, err := leakcheck.FromReal(su, engineBatches(sc.prefix))
	if err != nil {
		t.Fatal(err)
	}
	return frames, truth
}

// plantView rebuilds a frame's engine view, lets plant edit it, and checks
// it as seen by viewer.
func plantView(t *testing.T, c *leakcheck.Checker, fr wire.TableState, viewer, moment int, plant func(map[string]any)) []leakcheck.Leak {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(engineViewJSON(t, fr, lastEvents(fr)), &v); err != nil {
		t.Fatal(err)
	}
	plant(v)
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return c.Check(leakcheck.Doc{Kind: leakcheck.View, Viewer: viewer, Moment: moment, Data: data})
}

func hasLeak(leaks []leakcheck.Leak, rule leakcheck.Rule, path string) bool {
	return slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == rule && l.Path == path })
}

// TestPrivateViewFieldCanaries: the sixth-turn kept-card preview and the
// pile search's reasons pass the oracle in every seat's real frame, and a
// planted wrong-viewer copy of each fails: a kept card the viewer does not
// hold, a dropped preview, a pile reason in a seat that is not searching,
// a pile reason naming a card outside the pile, and a dropped reason.
func TestPrivateViewFieldCanaries(t *testing.T) {
	keptScene, pileScene := findPrivateFieldScenes(t)
	t.Run("kept preview", func(t *testing.T) {
		frames, truth := openScene(t, keptScene)
		c := leakcheck.New(truth, frameConfig)
		moment := len(keptScene.prefix)
		for s, fr := range frames {
			if leaks := plantView(t, c, fr, s, moment, func(map[string]any) {}); len(leaks) != 0 {
				t.Fatalf("seat %d's clean view: %v", s, leaks)
			}
		}
		me := keptScene.viewer
		fr := frames[me]
		i := slices.IndexFunc(fr.View.You.Choices, func(c wire.TableChoice) bool { return c.Kept != nil })
		if i < 0 {
			t.Fatal("the scene's viewer previews no kept card")
		}
		neighbour := (me + 1) % 7
		hands := truth.Moments[moment].Hands
		only := func(a, b int) string {
			for _, id := range hands[a] {
				if !slices.Contains(hands[b], id) {
					return id
				}
			}
			t.Fatalf("seat %d holds no card seat %d lacks", a, b)
			return ""
		}
		foreign, mine := only(neighbour, me), only(me, neighbour)
		if l := plantView(t, c, fr, me, moment, func(v map[string]any) {
			v["you"].(map[string]any)["choices"].([]any)[i].(map[string]any)["kept"] = foreign
		}); !hasLeak(l, leakcheck.HiddenCard, "choice.kept") {
			t.Errorf("a neighbour's card as the kept preview: %v", l)
		}
		if l := plantView(t, c, frames[neighbour], neighbour, moment, func(v map[string]any) {
			v["you"].(map[string]any)["choices"].([]any)[0].(map[string]any)["kept"] = mine
		}); !hasLeak(l, leakcheck.HiddenCard, "choice.kept") {
			t.Errorf("the viewer's preview copied to its neighbour: %v", l)
		}
		if l := plantView(t, c, fr, me, moment, func(v map[string]any) {
			delete(v["you"].(map[string]any)["choices"].([]any)[i].(map[string]any), "kept")
		}); !hasLeak(l, leakcheck.Mismatch, "choice.kept") {
			t.Errorf("a dropped preview: %v", l)
		}
	})
	t.Run("pile reasons", func(t *testing.T) {
		frames, truth := openScene(t, pileScene)
		c := leakcheck.New(truth, frameConfig)
		moment := len(pileScene.prefix)
		for s, fr := range frames {
			if leaks := plantView(t, c, fr, s, moment, func(map[string]any) {}); len(leaks) != 0 {
				t.Fatalf("seat %d's clean view: %v", s, leaks)
			}
		}
		me := pileScene.viewer
		fr := frames[me]
		if len(fr.View.You.Disabled) == 0 || fr.View.You.Disabled[0].Kind != "build-discarded" {
			t.Fatal("the searching seat gets no pile reason", fr.View.You.Disabled)
		}
		reason := fr.View.You.Disabled[0]
		other := (me + 1) % 7
		if len(frames[other].View.You.Disabled) != 0 {
			t.Fatal("a seat that is not searching got a reason", frames[other].View.You.Disabled)
		}
		if l := plantView(t, c, frames[other], other, moment, func(v map[string]any) {
			v["you"].(map[string]any)["disabled"] = []any{map[string]any{"kind": reason.Kind, "design": reason.Design, "stage": 0, "reason": reason.Reason, "cheapest": 0, "shortBy": 0}}
		}); !hasLeak(l, leakcheck.Mismatch, "disabled") {
			t.Errorf("the pile reason copied to a seat that is not searching: %v", l)
		}
		outside := truth.Slugs[slices.IndexFunc(truth.Slugs, func(id string) bool { return !slices.Contains(truth.Moments[moment].Pile, id) })]
		if l := plantView(t, c, fr, me, moment, func(v map[string]any) {
			v["you"].(map[string]any)["disabled"].([]any)[0].(map[string]any)["design"] = outside
		}); !hasLeak(l, leakcheck.HiddenCard, "disabled.design") {
			t.Errorf("a card outside the pile as a pile reason: %v", l)
		}
		if l := plantView(t, c, fr, me, moment, func(v map[string]any) {
			v["you"].(map[string]any)["disabled"] = []any{}
		}); !hasLeak(l, leakcheck.Mismatch, "disabled") {
			t.Errorf("a dropped pile reason: %v", l)
		}
	})
}
