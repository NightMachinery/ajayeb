package tableroom

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Each contaminated received frame must trigger its named detector and path.
// Opening, remembered mid-game, finished and spectator views first pass clean.
func TestRealViewExtraCanaries(t *testing.T) {
	const seed = 2029
	su := makeDaySetup(seed, 7, false, false)
	g, err := newGame(su)
	if err != nil {
		t.Fatal(err)
	}
	prefix := []BatchRecord{}
	var middle []BatchRecord
	for !pointOver(g.Point()) {
		pt := g.Point()
		b := batch{Point: pt.ID, Picks: []pick{}}
		for _, seat := range pt.Required.Seats() {
			cs := g.Legal(seat)
			b.Picks = append(b.Picks, pick{Seat: seat, Key: cs[0].Key()})
		}
		next, _, err := g.Resolve(b)
		if err != nil {
			t.Fatal(err)
		}
		g = next
		prefix = append(prefix, recordBatches([]tabletest.Batch{driverBatch(b)})...)
		if len(prefix) == 3 {
			middle = slices.Clone(prefix)
		}
	}
	scenarios := []struct {
		name   string
		prefix []BatchRecord
	}{{"opening", nil}, {"mid-game", middle}, {"finished", prefix}}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			sp := humans(7, seed)
			sp.Prefix = scenario.prefix
			h := newTH(t, Config{}, sp)
			a, err := h.p.actor()
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range []tabletest.ClientID{tabletest.SeatClient(0), tabletest.SpectatorClient} {
				if err := a.call(h.ctx, func() { a.inbound(h.p.clients[id], wire.NewResync(h.p.code)) }); err != nil {
					t.Fatal(err)
				}
			}
			h.p.drain()
			fr := h.latest(tabletest.SeatClient(0))
			truth, err := leakcheck.FromReal(su, engineBatches(scenario.prefix))
			if err != nil {
				t.Fatal(err)
			}
			checker := leakcheck.New(truth, frameConfig)
			moment := len(scenario.prefix)
			base := engineViewJSON(t, fr, lastEvents(fr))
			clean := func(data []byte, viewer int) {
				t.Helper()
				if got := checker.Check(leakcheck.Doc{Kind: leakcheck.View, Viewer: viewer, Moment: moment, Data: data}); len(got) > 0 {
					t.Fatal(got)
				}
			}
			clean(base, 0)
			own := truth.Moments[moment].Hands[0]
			foreign := ""
			for _, id := range truth.Slugs {
				if !slices.Contains(own, id) {
					foreign = id
					break
				}
			}
			type canary struct {
				name  string
				rule  leakcheck.Rule
				path  string
				plant func(map[string]any)
			}
			cases := []canary{}
			switch scenario.name {
			case "opening":
				cases = append(cases,
					canary{"remembered opponent hand", leakcheck.Mismatch, "you.seen", func(v map[string]any) {
						v["you"].(map[string]any)["notebook"].(map[string]any)["seen"] = []any{map[string]any{"point": "age-1-turn-1", "hand": []string{foreign}, "played": foreign, "action": "sell"}}
					}},
					canary{"foreign pile look", leakcheck.Mismatch, "notebook.pileLooks", func(v map[string]any) {
						v["you"].(map[string]any)["notebook"].(map[string]any)["pileLooks"] = []any{map[string]any{"point": "age-1-turn-1-discard-build", "cards": []any{map[string]any{"design": foreign, "count": 1}}}}
					}},
					canary{"foreign statistical deduction", leakcheck.Mismatch, "notebook.statistics", func(v map[string]any) {
						v["you"].(map[string]any)["notebook"].(map[string]any)["statistics"].(map[string]any)["neighbourPicks"] = []any{map[string]any{"point": "age-1-turn-1", "seat": 1, "design": foreign}}
					}},
					canary{"foreign witness value", leakcheck.Mismatch, "choice.witness", func(v map[string]any) {
						v["you"].(map[string]any)["choices"].([]any)[0].(map[string]any)["witness"] = []any{map[string]any{"resource": foreign, "source": "own-fixed", "producer": 0, "price": 0}}
					}},
					canary{"forged science counts", leakcheck.Mismatch, "science", func(v map[string]any) {
						v["seats"].([]any)[0].(map[string]any)["science"].(map[string]any)["counts"] = []int{9, 9, 9}
					}},
					canary{"unknown nested hash", leakcheck.UnknownField, "", func(v map[string]any) {
						v["you"].(map[string]any)["notebook"].(map[string]any)["observationHash"] = truth.Secrets[1].Value
					}},
					canary{"foreign disabled design", leakcheck.HiddenCard, "disabled.design", func(v map[string]any) {
						v["you"].(map[string]any)["disabled"].([]any)[0].(map[string]any)["design"] = foreign
					}},
					canary{"forged disabled cheapest", leakcheck.Mismatch, "disabled", func(v map[string]any) {
						d := v["you"].(map[string]any)["disabled"].([]any)[0].(map[string]any)
						d["cheapest"] = d["cheapest"].(float64) + 1
					}},
					canary{"forged disabled shortfall", leakcheck.Mismatch, "disabled", func(v map[string]any) {
						d := v["you"].(map[string]any)["disabled"].([]any)[0].(map[string]any)
						d["shortBy"] = d["shortBy"].(float64) + 1
					}},
					canary{"forged neighbour unit", leakcheck.Mismatch, "neighbours", func(v map[string]any) {
						u := v["you"].(map[string]any)["neighbours"].([]any)[0].(map[string]any)["units"].([]any)[0].(map[string]any)
						u["count"] = u["count"].(float64) + 1
					}},
				)
			case "mid-game":
				if fr.View.You == nil || len(fr.View.You.Notebook.Seen) == 0 {
					t.Fatal("mid-game notebook is empty")
				}
				seen := fr.View.You.Notebook.Seen[0].Hand
				for _, id := range truth.Slugs {
					if !slices.Contains(seen, id) {
						foreign = id
						break
					}
				}
				cases = append(cases, canary{"foreign card in remembered hand", leakcheck.HiddenCard, "you.seen[0].hand", func(v map[string]any) {
					s := v["you"].(map[string]any)["notebook"].(map[string]any)["seen"].([]any)[0].(map[string]any)
					s["hand"] = append(s["hand"].([]any), foreign)
				}})
			case "finished":
				if fr.View.Reveal == nil || len(fr.View.Reveal.Leftovers) == 0 {
					t.Fatal("finished reveal has no leftovers")
				}
				cases = append(cases, canary{"forged revealed leftover", leakcheck.Mismatch, "reveal.leftovers", func(v map[string]any) {
					v["reveal"].(map[string]any)["leftovers"].([]any)[0].(map[string]any)["age"] = 9
				}})
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					var v map[string]any
					if err := json.Unmarshal(base, &v); err != nil {
						t.Fatal(err)
					}
					tc.plant(v)
					data, err := json.Marshal(v)
					if err != nil {
						t.Fatal(err)
					}
					got := checker.Check(leakcheck.Doc{Kind: leakcheck.View, Viewer: 0, Moment: moment, Data: data})
					if !slices.ContainsFunc(got, func(l leakcheck.Leak) bool { return l.Rule == tc.rule && l.Path == tc.path }) {
						t.Fatalf("want %s at %q, got %v", tc.rule, tc.path, got)
					}
				})
			}
			t.Run("spectator private section", func(t *testing.T) {
				watcher := h.latest(tabletest.SpectatorClient)
				data := engineViewJSON(t, watcher, lastEvents(watcher))
				clean(data, -1)
				var v, seat map[string]any
				if err := json.Unmarshal(data, &v); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(base, &seat); err != nil {
					t.Fatal(err)
				}
				v["you"] = seat["you"]
				data, err = json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				got := checker.Check(leakcheck.Doc{Kind: leakcheck.View, Viewer: -1, Moment: moment, Data: data})
				if !slices.ContainsFunc(got, func(l leakcheck.Leak) bool { return l.Rule == leakcheck.ForeignOffer && l.Path == "you" }) {
					t.Fatalf("want foreign private section at you, got %v", got)
				}
			})
		})
	}
}

func driverBatch(b rules.Batch) tabletest.Batch {
	out := tabletest.Batch{Point: b.Point, Picks: []tabletest.Pick{}}
	for _, p := range b.Picks {
		out.Picks = append(out.Picks, tabletest.Pick{Seat: int(p.Seat), Key: p.Key})
	}
	return out
}
