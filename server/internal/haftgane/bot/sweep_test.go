package bot

import (
	"context"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func policyBatch(t testing.TB, g *rules.Game, levels [rules.MaxSeats]Level, entropy uint64) (rules.Batch, [rules.MaxSeats]Decision) {
	t.Helper()
	b := rules.Batch{Point: g.Point().ID(), Picks: []rules.Pick{}}
	var decisions [rules.MaxSeats]Decision
	for seat := rules.Seat(0); seat < rules.Seat(g.State.Seats); seat++ {
		if !g.Point().Required.Has(seat) {
			continue
		}
		cs := g.Legal(seat)
		d, err := Choose(context.Background(), levels[seat], g.Observe(seat), cs, rng.MixSeed(entropy, uint64(seat)), DefaultWork())
		if err != nil || !d.Complete {
			t.Fatal("policy failed", g.Point(), seat, levels[seat], err)
		}
		found := false
		for _, c := range cs {
			if c.Key() == d.Key {
				found = true
			}
		}
		if !found {
			t.Fatal("policy returned unoffered key", d.Key)
		}
		if d.Counters.Quotes > MaxQuoteCalls || d.Counters.Ranked > rules.MaxTurnChoices || d.Counters.Previews > rules.MaxTurnChoices || d.Counters.Iterations != 1 {
			t.Fatal("work bound", d)
		}
		decisions[seat] = d
		b.Picks = append(b.Picks, rules.Pick{Seat: seat, Key: d.Key})
	}
	return b, decisions
}
func TestAllCountsAllPointsLegal(t *testing.T) {
	var points [rules.NumPointKinds]int
	games, picks := 0, 0
	for n := uint8(3); n <= 7; n++ {
		for side := rules.SideRule(0); side < rules.NumSideRules; side++ {
			for focal := Random; focal < NumLevels; focal++ {
				for seed := uint64(0); seed < 6; seed++ {
					g, err := rules.NewGame(rules.Setup{Seed: 0x6d170001 + seed, Seats: n, Options: rules.Options{Sides: side, HideAtEnd: seed%2 == 1}})
					if err != nil {
						t.Fatal(err)
					}
					var levels [rules.MaxSeats]Level
					for s := uint8(0); s < n; s++ {
						levels[s] = Level((int(s) + int(focal)) % int(NumLevels))
					}
					for g.State.Phase != rules.PhaseOver {
						if g.State.Points >= rules.MaxPoints {
							t.Fatal("nonterminating game")
						}
						points[g.Point().Kind]++
						b, _ := policyBatch(t, &g, levels, rng.MixSeed(0x706f6c696379, seed))
						picks += len(b.Picks)
						g, _, err = g.Resolve(b)
						if err != nil {
							t.Fatal(err)
						}
						if err := g.CheckInvariants(); err != nil {
							t.Fatal(err)
						}
					}
					games++
				}
			}
		}
	}
	for kind := rules.PointKind(1); kind < rules.NumPointKinds; kind++ {
		if points[kind] == 0 {
			t.Fatal("unreached decision phase", kind)
		}
	}
	t.Logf("%d complete games, %d legal choices, point kinds %v", games, picks, points)
}

func assertBotPair(t *testing.T, a, b *rules.Game, seat rules.Seat) {
	t.Helper()
	oa, ob := a.Observe(seat), b.Observe(seat)
	ca, cb := a.Legal(seat), b.Legal(seat)
	if oa != ob || !reflect.DeepEqual(ca, cb) {
		t.Fatal("pair changed permitted inputs")
	}
	for _, level := range []Level{Random, Greedy, Apprentice} {
		da, ea := Choose(context.Background(), level, oa, ca, 0x68756464656e, DefaultWork())
		db, eb := Choose(context.Background(), level, ob, cb, 0x68756464656e, DefaultWork())
		if ea != nil || eb != nil || da != db {
			t.Fatal("forbidden information changed decision", level, ea, eb, da, db)
		}
		ra, sa, ea := Rank(context.Background(), oa, ca, nil)
		rb, sb, eb := Rank(context.Background(), ob, cb, nil)
		if ea != nil || eb != nil || sa != sb || !reflect.DeepEqual(ra, rb) {
			t.Fatal("forbidden information changed ranking/features")
		}
	}
}
func TestForbiddenInformationPairs(t *testing.T) {
	pairs := [5]int{}
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 12; seed++ {
			setup := rules.Setup{Seed: seed, Seats: n, Options: rules.Options{Sides: rules.SidesRandom}}
			a, err := rules.NewGame(setup)
			if err != nil {
				t.Fatal(err)
			}
			// A future packet exchange changes real future hands, not public facts.
			sc := rules.ScenarioFromGame(a)
			if sc.State.Deals[1][1][0] != sc.State.Deals[1][2][0] {
				sc.State.Deals[1][1][0], sc.State.Deals[1][2][0] = sc.State.Deals[1][2][0], sc.State.Deals[1][1][0]
				b, err := rules.NewGameFromScenario(seed, a.State.Opts, sc)
				if err != nil {
					t.Fatal(err)
				}
				assertBotPair(t, &a, &b, 0)
				pairs[0]++
			}
			// Exchange a dealt guild with an excluded one, conserving inventory.
			sc = rules.ScenarioFromGame(a)
			swapped := false
			for seat := uint8(0); seat < n && !swapped; seat++ {
				for card, id := range sc.State.Deals[2][seat] {
					if id >= 71 {
						sc.State.BoxGuilds[0], sc.State.Deals[2][seat][card] = id, sc.State.BoxGuilds[0]
						swapped = true
						break
					}
				}
			}
			if !swapped {
				t.Fatal("fixture has no dealt guild")
			}
			b, err := rules.NewGameFromScenario(seed, a.State.Opts, sc)
			if err != nil {
				t.Fatal(err)
			}
			assertBotPair(t, &a, &b, 0)
			pairs[4]++
			// Opponent hands exchange a physical copy, maintaining all supply/size.
			sc = rules.ScenarioFromGame(a)
			if sc.State.Hands[1][0] != sc.State.Hands[2][0] {
				sc.State.Hands[1][0], sc.State.Hands[2][0] = sc.State.Hands[2][0], sc.State.Hands[1][0]
				// At the initial point these hands are the entered materialized deal.
				sc.Record.Packets[0] = sc.State.Hands
				b, err := rules.NewGameFromScenario(seed^0xdeadbeef, a.State.Opts, sc)
				if err != nil {
					t.Fatal(err)
				}
				assertBotPair(t, &a, &b, 0)
				pairs[1]++
			}
			// Replay two different hidden sales or burials from the same full deal.
			for _, kind := range []rules.ActionKind{rules.ActSell, rules.ActStage} {
				source := a
				// Starting production cannot afford a stage. Reach a real later
				// turn before comparing two hidden carriers for the same stage.
				if kind == rules.ActStage {
					levels := [rules.MaxSeats]Level{Greedy, Greedy, Greedy, Greedy, Greedy, Greedy, Greedy}
					for source.State.Age == 1 && source.State.Turn < 6 {
						hasStage := false
						for _, c := range source.Legal(1) {
							hasStage = hasStage || c.Kind == rules.ActStage
						}
						if hasStage {
							break
						}
						batch, _ := policyBatch(t, &source, levels, seed)
						source, _, err = source.Resolve(batch)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				cs := source.Legal(1)
				var variants []rules.Choice
				for _, c := range cs {
					if c.Kind == kind && (len(variants) == 0 || c.Design != variants[0].Design) {
						if len(variants) == 0 || c.Left == variants[0].Left && c.Right == variants[0].Right {
							variants = append(variants, c)
						}
					}
					if len(variants) == 2 {
						break
					}
				}
				if len(variants) != 2 {
					continue
				}
				batches := [2]rules.Batch{}
				for k := range 2 {
					batch := rules.Batch{Point: source.Point().ID()}
					for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
						key := (rules.Action{Kind: rules.ActSell, Design: source.State.Hands[seat][0]}).Key()
						if seat == 1 {
							key = variants[k].Key()
						}
						batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: key})
					}
					batches[k] = batch
				}
				ga, _, err := source.Resolve(batches[0])
				if err != nil {
					t.Fatal(err)
				}
				gb, _, err := source.Resolve(batches[1])
				if err != nil {
					t.Fatal(err)
				}
				if ga.State.Phase != rules.PhaseTurn {
					continue
				}
				assertBotPair(t, &ga, &gb, 0)
				if kind == rules.ActSell {
					pairs[2]++
				} else {
					pairs[3]++
				}
			}
		}
	}
	for kind, count := range pairs {
		if count == 0 {
			t.Fatal("missing hidden pair", kind)
		}
	}
	t.Logf("equal bot decisions/rankings/features/counters across pairs: future %d, opponent hand/root seed %d, hidden sale %d, hidden tuck %d, excluded guild %d", pairs[0], pairs[1], pairs[2], pairs[3], pairs[4])
}

func TestRandomHierarchyAndBoundedRanking(t *testing.T) {
	o := testObservation(2, 3)
	o.Now.Hand = [rules.HandSize]catalog.DesignID{39, 39, 40}
	o.Now.Cities[0].Coins = 500
	cs := []rules.Choice{testBuild(39, 0, 0), testBuild(39, 1, 0), testBuild(39, 0, 1), {Action: rules.Action{Kind: rules.ActSell, Design: 39}}, testBuild(40, 0, 0)}
	var designs [2]int
	var kinds [2]int
	for seed := uint64(0); seed < 3000; seed++ {
		d, err := Choose(context.Background(), Random, o, cs, seed, DefaultWork())
		if err != nil {
			t.Fatal(err)
		}
		a, err := rules.ParseKey(d.Key)
		if err != nil {
			t.Fatal(err)
		}
		if a.Design == 39 {
			designs[0]++
			if a.Kind == rules.ActBuild {
				kinds[0]++
			} else {
				kinds[1]++
			}
		} else {
			designs[1]++
		}
	}
	// Multiplicity is 2:1, then kinds 1:1 despite three build payments.
	if designs[0] < 1800 || designs[0] > 2200 || kinds[0] < 850 || kinds[0] > 1150 {
		t.Fatal("random distribution weights payment variants", designs, kinds)
	}
	full := make([]rules.Choice, 0, rules.MaxTurnChoices)
	for i := 0; i < rules.MaxTurnChoices; i++ {
		full = append(full, testBuild(60, uint8(i%15), uint8(i/15)))
	}
	ranked, counters, err := Rank(context.Background(), o, full, nil)
	if err != nil || len(ranked) != rules.MaxTurnChoices || counters.Ranked != rules.MaxTurnChoices || counters.Quotes > MaxQuoteCalls {
		t.Fatal("ranking bound", len(ranked), counters, err)
	}
	t.Log("random multiplicities/kinds", designs, kinds, "max rank counters", counters)
}
