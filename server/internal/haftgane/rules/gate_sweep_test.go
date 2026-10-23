package rules

import (
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// These are acceptance policies, not the later shipping bots. Random chooses
// an offered key uniformly. Simple is deterministic and reads only its own
// offer plus public city/point facts; it develops production, then stages and
// points, and sells when no worthwhile construction is offered.
func gateChoice(g *Game, seat Seat, policy string, r *rng.RNG) Choice {
	choices := g.Legal(seat)
	if policy == "random" {
		return choices[r.IntN(len(choices))]
	}
	best, chosen := -1<<30, choices[0]
	for _, c := range choices {
		value := -int(c.Total) * 3
		switch c.Kind {
		case ActSide:
			value += int(c.Side)
		case ActSell:
			value += 1
		case ActBuild, ActBuildDiscarded:
			d, _ := catalog.DesignByID(c.Design)
			switch d.Color {
			case catalog.Brown, catalog.Grey:
				value += 30 - int(g.State.Age)*5
			case catalog.Blue:
				value += 13
			case catalog.Green:
				value += 15
			case catalog.Yellow:
				value += 12
			case catalog.Red:
				value += 11
			case catalog.Purple:
				value += 16
			}
			if c.Free != FreeNone {
				value += 4
			}
		case ActStage:
			value += 20
			if g.State.Cities[seat].Board == catalog.BoardHalikarnassos || g.State.Cities[seat].Board == catalog.BoardBabylon || g.State.Cities[seat].Board == catalog.BoardOlympia {
				value += 12
			}
		}
		if value > best {
			best, chosen = value, c
		}
	}
	return chosen
}

type gateCounts struct {
	Games, Points                                                        int
	Point                                                                [NumPointKinds]int
	Action                                                               [NumActionKinds]int
	Free                                                                 [FreeDiscard + 1]int
	Effect                                                               [catalog.NumEffectKinds]int
	Condition                                                            [catalog.NumFreeConditions]int
	Events                                                               [NumEventKinds]int
	Stages                                                               [catalog.NumBoards + 1][3][MaxStages + 1]int
	Last                                                                 [NumActionKinds]int
	EmptySkip, DuplicateSkip                                             int
	MaxChoices, MaxPayments, MaxEvents, MaxPoints, MaxBuildings, MaxPile int
}

func (c *gateCounts) placement(st *State, seat Seat, a Action) {
	c.Action[a.Kind]++
	var effects []catalog.Effect
	switch a.Kind {
	case ActStage:
		city := st.Cities[seat]
		board, _ := catalog.BoardByID(city.Board)
		c.Stages[city.Board][city.Side][city.Stages+1]++
		effects = board.Sides[city.Side-1].Stages[city.Stages].Effects
	case ActBuild, ActBuildDiscarded:
		d, _ := catalog.DesignByID(a.Design)
		effects = d.Effects
	}
	for _, e := range effects {
		if e.Kind != catalog.EffNone {
			c.Effect[e.Kind]++
			if e.Kind == catalog.EffFreeBuild {
				c.Condition[e.When]++
			}
		}
	}
}
func TestRulesGateSweep(t *testing.T) {
	seeds := 8
	if text := os.Getenv("HAFTGANE_GATE_GAMES"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 100000 {
			t.Fatal("invalid HAFTGANE_GATE_GAMES", text)
		}
		seeds = n
	}
	var total gateCounts
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for side := SideRule(0); side < NumSideRules; side++ {
			for _, policy := range []string{"random", "simple"} {
				group := gateCounts{}
				for seed := 0; seed < seeds; seed++ {
					setup := Setup{Seats: n, Seed: uint64(seed), Options: Options{Sides: side, Seating: SeatingRule(seed % 2), HideAtEnd: seed%2 == 1}}
					g, err := NewGame(setup)
					if err != nil {
						t.Fatal(err)
					}
					r := rng.NewRNG2(uint64(seed), 0x676174652d706f6c)
					batches := make([]Batch, 0, MaxPoints)
					if err := g.CheckInvariants(); err != nil {
						t.Fatal(err)
					}
					for g.State.Phase != PhaseOver {
						if len(batches) >= MaxPoints {
							t.Fatal("point bound", n, side, policy, seed)
						}
						p := g.Point()
						group.Point[p.Kind]++
						total.Point[p.Kind]++
						b := Batch{Point: p.ID(), Picks: make([]Pick, 0, n)}
						var picks [MaxSeats]Action
						for seat := Seat(0); seat < Seat(n); seat++ {
							cs := g.Legal(seat)
							if !p.Required.Has(seat) {
								if len(cs) != 0 {
									t.Fatal("non-required offer")
								}
								continue
							}
							if len(cs) == 0 || len(cs) > MaxTurnChoices {
								t.Fatal("offer cardinality", len(cs))
							}
							group.MaxChoices = max(group.MaxChoices, len(cs))
							total.MaxChoices = max(total.MaxChoices, len(cs))
							var payments [catalog.NumDesigns + 1][NumActionKinds]int
							for _, ch := range cs {
								parsed, err := ParseKey(ch.Key())
								if err != nil || parsed != ch.Action {
									t.Fatal("offered key is not canonical", err)
								}
								if ch.Kind == ActBuild || ch.Kind == ActStage {
									payments[ch.Design][ch.Kind]++
									count := payments[ch.Design][ch.Kind]
									if count > MaxPaymentOutcomes {
										t.Fatal("too many construction outcomes", count)
									}
									group.MaxPayments = max(group.MaxPayments, count)
									total.MaxPayments = max(total.MaxPayments, count)
								}
							}
							chosen := gateChoice(&g, seat, policy, &r)
							picks[seat] = chosen.Action
							b.Picks = append(b.Picks, Pick{Seat: seat, Key: chosen.Key()})
							group.Free[chosen.Free]++
							total.Free[chosen.Free]++
							group.placement(&g.State, seat, chosen.Action)
							total.placement(&g.State, seat, chosen.Action)
							if p.Kind == PointLastCard {
								group.Last[chosen.Kind]++
								total.Last[chosen.Kind]++
							}
						}
						frozen := g
						next, events, err := g.Resolve(b)
						if err != nil {
							t.Fatal(n, side, policy, seed, p.ID(), err)
						}
						if g != frozen {
							t.Fatal("resolution changed caller")
						}
						if err := next.CheckInvariants(); err != nil {
							t.Fatal(n, side, policy, seed, p.ID(), err)
						}
						checkCoinLedger(t, &g.State, &next.State, &picks, events)
						if len(events) > MaxResolutionEvents {
							t.Fatal("event bound", len(events))
						}
						group.MaxEvents = max(group.MaxEvents, len(events))
						total.MaxEvents = max(total.MaxEvents, len(events))
						for _, e := range events {
							group.Events[e.Kind]++
							total.Events[e.Kind]++
							if e.Kind == EvDiscardBuildNothing {
								pile := 0
								for _, count := range next.State.Discard {
									pile += int(count)
								}
								if pile == 0 {
									group.EmptySkip++
									total.EmptySkip++
								} else {
									group.DuplicateSkip++
									total.DuplicateSkip++
								}
							}
						}
						pile := 0
						for seat := Seat(0); seat < Seat(n); seat++ {
							count := next.State.Cities[seat].Buildings.Count()
							if count > MaxCityBuildings {
								t.Fatal("city building bound", count)
							}
							group.MaxBuildings = max(group.MaxBuildings, count)
							total.MaxBuildings = max(total.MaxBuildings, count)
						}
						for _, count := range next.State.Discard {
							pile += int(count)
						}
						if pile > MaxPileCards {
							t.Fatal("pile bound", pile)
						}
						group.MaxPile = max(group.MaxPile, pile)
						total.MaxPile = max(total.MaxPile, pile)
						batches = append(batches, b)
						g = next
						group.Points++
						total.Points++
					}
					if g.State.Result != referenceResult(&g.State) {
						t.Fatal("independent score oracle", n, side, policy, seed)
					}
					replay, err := Replay(setup, batches)
					if err != nil || replay != g || replay.Fingerprint() != g.Fingerprint() {
						t.Fatal("replay differs", err)
					}
					group.Games++
					total.Games++
					group.Point[PointNone]++
					total.Point[PointNone]++
					group.MaxPoints = max(group.MaxPoints, len(batches))
					total.MaxPoints = max(total.MaxPoints, len(batches))
				}
				t.Logf("N=%d side=%d policy=%s seeds=%d: %+v", n, side, policy, seeds, group)
			}
		}
	}
	if total.Games != 5*int(NumSideRules)*2*seeds {
		t.Fatal("incomplete sweep", total.Games)
	}
	if seeds >= 2000 {
		for kind := PointKind(1); kind < NumPointKinds; kind++ {
			if total.Point[kind] == 0 {
				t.Fatal("unreached point", kind)
			}
		}
		for kind := catalog.EffectKind(1); kind < catalog.NumEffectKinds; kind++ {
			if total.Effect[kind] == 0 {
				t.Fatal("unreached effect", kind)
			}
		}
		for reason := FreeNone; reason <= FreeDiscard; reason++ {
			if total.Free[reason] == 0 {
				t.Fatal("unreached free reason", reason)
			}
		}
		for condition := catalog.FreeCondition(1); condition < catalog.NumFreeConditions; condition++ {
			if total.Condition[condition] == 0 {
				t.Fatal("unreached condition", condition)
			}
		}
		for kind := EventKind(1); kind < NumEventKinds; kind++ {
			if total.Events[kind] == 0 {
				t.Fatal("unreached event", kind)
			}
		}
	}
	t.Logf("FULL COUNTS: %+v", total)
}

// A bounded playout consumes only fuzz bytes and offered keys. Every accepted
// transition checks conservation, the state-only core, the coin ledger and a
// complete replay. A malformed-key probe must fail without mutating values.
func FuzzPlayout(f *testing.F) {
	for _, data := range [][]byte{{}, {0}, {1, 2, 3, 4}, {255, 254, 128, 64, 32, 16, 8, 4, 2, 1}, []byte("Babylon-Halikarnassos-Olympia")} {
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		at := 0
		nextByte := func() byte {
			if len(data) == 0 {
				return 0
			}
			b := data[at%len(data)]
			at++
			return b
		}
		seed := uint64(0)
		for i := 0; i < 8; i++ {
			seed |= uint64(nextByte()) << uint(8*i)
		}
		setup := Setup{Seed: seed, Seats: 3 + nextByte()%5, Options: Options{Sides: SideRule(nextByte() % 3), Seating: SeatingRule(nextByte() % 2), HideAtEnd: nextByte()%2 == 1}}
		g, err := NewGame(setup)
		if err != nil {
			t.Fatal(err)
		}
		batches := make([]Batch, 0, MaxPoints)
		for g.State.Phase != PhaseOver {
			if len(batches) >= MaxPoints {
				t.Fatal("nonterminating game")
			}
			b := Batch{Point: g.Point().ID(), Picks: make([]Pick, 0, MaxSeats)}
			var picks [MaxSeats]Action
			for seat := Seat(0); seat < Seat(setup.Seats); seat++ {
				if !g.Point().Required.Has(seat) {
					continue
				}
				cs := g.Legal(seat)
				if len(cs) == 0 || len(cs) > MaxTurnChoices {
					t.Fatal("invalid offer cardinality")
				}
				c := cs[int(nextByte())%len(cs)]
				picks[seat] = c.Action
				b.Picks = append(b.Picks, Pick{Seat: seat, Key: c.Key()})
			}
			if nextByte()%8 == 0 {
				bad := Batch{Point: b.Point, Picks: append([]Pick{}, b.Picks...)}
				bad.Picks[0].Key += "!"
				before := g
				refused, events, err := g.Resolve(bad)
				if err == nil || g != before || refused != before || len(events) != 0 {
					t.Fatal("bad-key refusal changed values")
				}
			}
			core, coreEvents, err := ResolveState(&g.State, &picks, nil)
			if err != nil {
				t.Fatal(err)
			}
			played, events, err := g.Resolve(b)
			if err != nil || played.State != core || !reflect.DeepEqual(events, coreEvents) {
				t.Fatal("core/wrapper differs", err)
			}
			if err := played.CheckInvariants(); err != nil {
				t.Fatal(err)
			}
			checkCoinLedger(t, &g.State, &played.State, &picks, events)
			batches = append(batches, b)
			g = played
		}
		replay, err := Replay(setup, batches)
		if err != nil || replay != g {
			t.Fatal("fuzz replay differs", err)
		}
	})
}
