package rules

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

func observationBatch(g Game, seed uint64) Batch {
	p := g.Point()
	b := Batch{Point: p.ID(), Picks: make([]Pick, 0, MaxSeats)}
	for s := Seat(0); s < Seat(g.State.Seats); s++ {
		if !p.Required.Has(s) {
			continue
		}
		cs := g.Legal(s)
		choice := cs[(int(seed)*5+int(g.State.Points)*3+int(s)*7)%len(cs)]
		if seed%2 == 0 {
			for _, c := range cs {
				if c.Kind == ActStage {
					choice = c
					break
				}
			}
		}
		b.Picks = append(b.Picks, Pick{Seat: s, Key: choice.Key()})
	}
	return b
}

// Section 10: replayed history is exactly the stream seen live, for every
// entitlement, including Age I and private abilities. End reveal never changes
// historical event filtering. No initial events are returned by NewGame.
func TestHistoryEqualsLiveFilteredStreams(t *testing.T) {
	looks, skipped, lastCards := 0, 0, 0
	for _, n := range []uint8{3, 7} {
		for seed := uint64(0); seed < 24; seed++ {
			for _, hide := range []bool{false, true} {
				g, err := NewGame(Setup{Seats: n, Seed: seed, Options: Options{Sides: SideRule(seed % 3), HideAtEnd: hide}})
				if err != nil {
					t.Fatal(err)
				}
				var live [MaxSeats + 1][]view.HistoryEntry
				for i := 0; i <= int(n); i++ {
					viewer := Seat(i)
					if i == int(n) {
						viewer = NoSeat
					}
					initial := []view.Event{}
					if g.State.Opts.Sides != SidesChoose {
						initial = FilterEvents([]Event{{Kind: EvAgeStart, Age: 1}}, viewer)
					}
					live[i] = []view.HistoryEntry{{Point: HistoryStartPoint, Events: initial}}
				}
				for {
					for i := 0; i <= int(n); i++ {
						viewer := Seat(i)
						if i == int(n) {
							viewer = NoSeat
						}
						got, err := g.History(viewer)
						if err != nil || !reflect.DeepEqual(got, live[i]) {
							t.Fatal("history differs from live", n, seed, hide, g.Point().ID(), viewer, err, got, live[i])
						}
					}
					if g.State.Phase == PhaseOver {
						break
					}
					before := g
					p := g.Point()
					b := observationBatch(g, seed)
					next, events, err := g.Resolve(b)
					if err != nil {
						t.Fatal(err)
					}
					for i := 0; i <= int(n); i++ {
						viewer := Seat(i)
						if i == int(n) {
							viewer = NoSeat
						}
						live[i] = append(live[i], view.HistoryEntry{Point: p.ID(), Events: FilterEvents(events, viewer)})
					}
					if p.Kind == PointLastCard {
						lastCards++
					}
					for _, pick := range b.Picks {
						a, err := ParseKey(pick.Key)
						if err != nil {
							t.Fatal(err)
						}
						k := next.Observe(pick.Seat).Know
						if p.Kind == PointTurn || p.Kind == PointLastCard {
							hand := before.State.Hands[pick.Seat]
							if p.Kind == PointLastCard {
								hand = [HandSize]catalog.DesignID{before.State.Cities[pick.Seat].Kept}
							}
							if k.Seen[p.Age-1][p.Turn-1] != hand || k.Picks[p.Age-1][p.Turn-1] != a.Design {
								t.Fatal("notebook did not remember exactly own sighting", p.ID(), pick.Seat)
							}
						}
						if a.Kind == ActStage && stageHasEffect(&next.State.Cities[pick.Seat], next.State.Cities[pick.Seat].Stages, catalog.EffBuildFromDiscard) {
							looks++
							found := false
							for _, look := range k.PileLooks {
								if look.Age == p.Age && look.Turn == p.Turn {
									found = true
									if look.Count != next.State.Discard {
										t.Fatal("pile look is not step-six offer")
									}
								}
							}
							if !found {
								t.Fatal("missing owner's pile look")
							}
							other := pick.Seat.Left(n)
							for _, look := range next.Observe(other).Know.PileLooks {
								if look.Age == p.Age && look.Turn == p.Turn {
									t.Fatal("pile look given to non-owner")
								}
							}
							for _, ev := range events {
								if ev.Kind == EvDiscardBuildNothing && ev.Seat == pick.Seat {
									skipped++
								}
							}
							changed := next.Observe(pick.Seat)
							oldHash := changed.Fingerprint()
							for i := range changed.Know.PileLooks {
								if changed.Know.PileLooks[i].Age == p.Age && changed.Know.PileLooks[i].Turn == p.Turn {
									changed.Know.PileLooks[i] = PileLook{}
								}
							}
							if oldHash == changed.Fingerprint() {
								t.Fatal("pile look absent from observation identity")
							}
						}
					}
					g = next
				}
			}
		}
	}
	if looks == 0 || lastCards == 0 {
		t.Fatal("ability history paths not exercised", looks, skipped, lastCards)
	}
	t.Logf("live/history equality: 96 complete games, %d pile looks (%d skipped), %d last cards", looks, skipped, lastCards)
}

func TestHistoryPartialScenarioFailsClosed(t *testing.T) {
	g := abilityScenario(t, 2, 3, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night}}, [MaxSeats][]catalog.DesignID{0: {39}})
	if events, err := g.History(0); err == nil || events == nil || len(events) != 0 {
		t.Fatal("incomplete history invented events", err, events)
	}
	if g.Observe(0).Know != (Knowledge{}) {
		t.Fatal("incomplete history invented memories")
	}
	if g.Observe(0).Now.Hand != g.State.Hands[0] {
		t.Fatal("partial history blocked current private facts")
	}
}

// Ruling 3: an empty pile still creates private knowledge of its complete
// multiset. Reach the skip from a full initial deal, so history is replayable.
func TestHistorySkippedPileLook(t *testing.T) {
	for seed := uint64(0); seed < 1000; seed++ {
		g, err := NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SidesChoose}})
		if err != nil {
			t.Fatal(err)
		}
		sc := ScenarioFromGame(g)
		for seat := 0; seat < 7; seat++ {
			if sc.State.Cities[seat].Board == catalog.BoardHalikarnassos {
				sc.State.Cities[0].Board, sc.State.Cities[seat].Board = sc.State.Cities[seat].Board, sc.State.Cities[0].Board
			}
		}
		for seat := 1; seat < 7; seat++ {
			if sc.State.Cities[seat].Board == catalog.BoardOlympia {
				sc.State.Cities[1].Board, sc.State.Cities[seat].Board = sc.State.Cities[seat].Board, sc.State.Cities[1].Board
			}
		}
		g, err = NewGameFromScenario(seed, g.State.Opts, sc)
		if err != nil {
			t.Fatal(err)
		}
		side := Batch{Point: g.Point().ID()}
		for seat := Seat(0); seat < 7; seat++ {
			s := catalog.Day
			if seat == 0 {
				s = catalog.Night
			}
			side.Picks = append(side.Picks, Pick{Seat: seat, Key: (Action{Kind: ActSide, Side: s}).Key()})
		}
		g, _, err = g.Resolve(side)
		if err != nil {
			t.Fatal(err)
		}
		possible := true
		var events []Event
		for turn := 1; turn <= 2; turn++ {
			b := Batch{Point: g.Point().ID()}
			for seat := Seat(0); seat < 7; seat++ {
				var chosen Choice
				for _, c := range g.Legal(seat) {
					if seat == 0 {
						if turn == 1 && c.Kind == ActBuild && c.Design == 2 || turn == 2 && c.Kind == ActStage {
							chosen = c
							break
						}
					} else if c.Kind == ActBuild {
						chosen = c
						break
					}
				}
				if chosen.Kind == ActNone {
					possible = false
					break
				}
				b.Picks = append(b.Picks, Pick{Seat: seat, Key: chosen.Key()})
			}
			if !possible {
				break
			}
			g, events, err = g.Resolve(b)
			if err != nil {
				t.Fatal(err)
			}
		}
		if !possible {
			continue
		}
		skipped := false
		for _, e := range events {
			if e.Kind == EvDiscardBuildNothing && e.Seat == 0 {
				skipped = true
			}
		}
		if !skipped {
			t.Fatal("empty pile did not skip")
		}
		o := g.Observe(0)
		if o.Know.PileLooks[0] != (PileLook{Age: 1, Turn: 2}) || o.Know.PileLooks[1] != (PileLook{}) {
			t.Fatal("empty opportunity not remembered", o.Know.PileLooks)
		}
		if g.Observe(1).Know.PileLooks != ([MaxPileLooks]PileLook{}) {
			t.Fatal("skipped private look leaked")
		}
		h, err := g.History(0)
		if err != nil || len(h) != 4 {
			t.Fatal("skipped look history not replayable", err, h)
		}
		t.Logf("reachable empty-pile skip at seed %d", seed)
		return
	}
	t.Fatal("no reachable empty-pile skip fixture")
}

// The platform's batch counter is the History index for every side option.
// The initial entry is batch zero, before any accepted choice batch.
func TestHistoryBatchZeroContract(t *testing.T) {
	for side := SideRule(0); side < NumSideRules; side++ {
		g, err := NewGame(Setup{Seed: 83, Seats: 3, Options: Options{Sides: side}})
		if err != nil {
			t.Fatal(err)
		}
		for {
			for _, viewer := range []Seat{0, 1, 2, NoSeat} {
				entries, err := g.History(viewer)
				if err != nil || len(entries) != int(g.State.Points)+1 {
					t.Fatalf("side %d batch %d history length %d: %v", side, g.State.Points, len(entries), err)
				}
				if entries[0].Point != HistoryStartPoint || entries[0].Events == nil {
					t.Fatal("missing canonical batch zero", side, entries[0])
				}
				initial := []view.Event{}
				if side != SidesChoose {
					initial = FilterEvents([]Event{{Kind: EvAgeStart, Age: 1}}, viewer)
				}
				if !reflect.DeepEqual(entries[0].Events, initial) {
					t.Fatal("batch zero events", side, entries[0].Events, initial)
				}
			}
			if g.State.Phase == PhaseOver {
				break
			}
			g, _, err = g.Resolve(salesBatch(g))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
