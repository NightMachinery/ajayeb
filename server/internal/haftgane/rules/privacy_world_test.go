package rules

// Whole-game privacy checks adapted from the independent views review.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func privacySweepMix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// A policy unlike the slice's: biased to stages and sales so that piles,
// looks, kept cards and anonymous removals are frequent.
func privacySweepBatch(g Game, seed uint64, step int) Batch {
	p := g.Point()
	b := Batch{Point: p.ID()}
	for s := Seat(0); s < Seat(g.State.Seats); s++ {
		if !p.Required.Has(s) {
			continue
		}
		cs := g.Legal(s)
		h := privacySweepMix(seed*1000003 + uint64(step)*131 + uint64(s))
		c := cs[h%uint64(len(cs))]
		switch (h >> 20) % 5 {
		case 0, 1:
			for _, x := range cs {
				if x.Kind == ActStage {
					c = x
					break
				}
			}
		case 2:
			for _, x := range cs {
				if x.Kind == ActSell {
					c = x
					break
				}
			}
		}
		b.Picks = append(b.Picks, Pick{Seat: s, Key: c.Key()})
	}
	return b
}

type privacySweepRun struct {
	games   []Game
	batches []Batch
}

func privacySweepPlay(t *testing.T, setup Setup, seed uint64) privacySweepRun {
	t.Helper()
	g, err := NewGame(setup)
	if err != nil {
		t.Fatal(err)
	}
	run := privacySweepRun{games: []Game{g}}
	for step := 0; g.Point().Kind != PointNone; step++ {
		b := privacySweepBatch(g, seed, step)
		next, _, err := g.Resolve(b)
		if err != nil {
			t.Fatal(err)
		}
		run.batches = append(run.batches, b)
		run.games = append(run.games, next)
		g = next
	}
	return run
}

func privacySweepViewers(n uint8) []Seat {
	out := []Seat{}
	for s := Seat(0); s < Seat(n); s++ {
		out = append(out, s)
	}
	return append(out, NoSeat)
}

// knows reports whether viewer has ever held, seen or been offered id.
func privacySweepKnows(g *Game, v Seat, id catalog.DesignID) bool {
	if !v.Valid(g.State.Seats) {
		return false
	}
	o := g.Observe(v)
	for _, x := range o.Now.Hand {
		if x == id {
			return true
		}
	}
	for _, x := range o.Now.Tucked {
		if x == id {
			return true
		}
	}
	if o.Now.Kept == id || o.Now.Offer[id] > 0 {
		return true
	}
	for a := range o.Know.Seen {
		for t := range o.Know.Seen[a] {
			for _, x := range o.Know.Seen[a][t] {
				if x == id {
					return true
				}
			}
			if o.Know.Picks[a][t] == id {
				return true
			}
		}
	}
	for _, l := range o.Know.PileLooks {
		if l.Count[id] > 0 {
			return true
		}
	}
	return false
}

func privacySweepSubstitute(b Batch, from, to catalog.DesignID) Batch {
	out := Batch{Point: b.Point, Picks: make([]Pick, len(b.Picks))}
	copy(out.Picks, b.Picks)
	fromSlug, toSlug := designSlug(from), designSlug(to)
	for i, p := range out.Picks {
		a, err := ParseKey(p.Key)
		if err == nil && a.Design == from {
			out.Picks[i].Key = strings.Replace(p.Key, ":"+fromSlug, ":"+toSlug, 1)
		}
	}
	return out
}

func privacySweepSwapIn(arr []catalog.DesignID, from, to catalog.DesignID) bool {
	for i, x := range arr {
		if x == from {
			arr[i] = to
			return true
		}
	}
	return false
}

func privacySweepJSON(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// P1: an Age III guild that is never built, swapped with a box guild before
// the game. Every viewer who never sees the guild must get identical views,
// observations, statistics, histories and offers, at every position.
func TestWholeGameGuildSwapNoninterference(t *testing.T) {
	pairs, positions, frames, skipped, diverged := 0, 0, 0, 0, 0
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 6; seed++ {
			for _, hide := range []bool{false, true} {
				setup := Setup{Seats: n, Seed: 0xabcdef00 + seed*7919, Options: Options{Sides: SideRule(seed % 3), HideAtEnd: hide}}
				a := privacySweepPlay(t, setup, seed)
				a0 := a.games[0]
				final := a.games[len(a.games)-1]
				var dealt []catalog.DesignID
				for _, row := range a0.State.Deals[2] {
					for _, id := range row {
						if d, ok := catalog.DesignByID(id); ok && d.Color == catalog.Purple {
							dealt = append(dealt, id)
						}
					}
				}
				tried := 0
				for _, gid := range dealt {
					built := false
					for s := 0; s < int(n); s++ {
						if final.State.Cities[s].Buildings.Has(gid) {
							built = true
						}
					}
					if built || tried >= 2 {
						continue
					}
					for _, hid := range a0.State.BoxGuilds {
						if hid == 0 || tried >= 2 {
							continue
						}
						tried++
						pairs++
						sc := ScenarioFromGame(a0)
						swapped := false
						for s := range sc.State.Deals[2] {
							if privacySweepSwapIn(sc.State.Deals[2][s][:], gid, hid) {
								swapped = true
								break
							}
						}
						if !swapped || !privacySweepSwapIn(sc.State.BoxGuilds[:], hid, gid) {
							t.Fatal("swap failed")
						}
						b0, err := NewGameFromScenario(a0.State.Seed, a0.State.Opts, sc)
						if err != nil {
							t.Fatal("swapped setup rejected", err)
						}
						bGames := []Game{b0}
						bg := b0
						for i, ba := range a.batches {
							bb := privacySweepSubstitute(ba, gid, hid)
							next, _, err := bg.Resolve(bb)
							if err != nil || next.Point().ID() != a.games[i+1].Point().ID() {
								diverged++
								break
							}
							bg = next
							bGames = append(bGames, bg)
						}
						for i := range bGames {
							ga, gb := a.games[i], bGames[i]
							if ga.State.Phase == PhaseOver && !hide {
								continue // the end reveal legitimately shows the box
							}
							positions++
							for _, v := range privacySweepViewers(n) {
								if privacySweepKnows(&ga, v, gid) || privacySweepKnows(&gb, v, hid) {
									skipped++
									continue
								}
								frames++
								where := fmt.Sprintf("n%d seed%d hide%v %s viewer %d (guild %s vs %s)", n, seed, hide, ga.Point().ID(), v, designSlug(gid), designSlug(hid))
								va, vb := privacySweepJSON(t, ga.ViewFor(v)), privacySweepJSON(t, gb.ViewFor(v))
								if !bytes.Equal(va, vb) {
									t.Errorf("LEAK view: %s\nA %s\nB %s", where, va, vb)
									return
								}
								oa, ob := ga.Observe(v), gb.Observe(v)
								if oa != ob || oa.Fingerprint() != ob.Fingerprint() || oa.Statistics() != ob.Statistics() {
									t.Errorf("LEAK observation: %s", where)
									return
								}
								ha, errA := ga.History(v)
								hb, errB := gb.History(v)
								if errA != nil || errB != nil || !bytes.Equal(privacySweepJSON(t, ha), privacySweepJSON(t, hb)) {
									t.Errorf("LEAK history: %s %v %v", where, errA, errB)
									return
								}
								if v.Valid(n) && !bytes.Equal(privacySweepJSON(t, ga.Legal(v)), privacySweepJSON(t, gb.Legal(v))) {
									t.Errorf("LEAK offer: %s", where)
									return
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("%d guild-swap pairs, %d positions, %d viewer frames compared equal, %d viewer frames skipped (viewer saw the guild), %d replays diverged publicly", pairs, positions, frames, skipped, diverged)
	if frames == 0 || pairs == 0 || diverged != 0 {
		t.Fatal("whole-game noninterference coverage", pairs, frames, diverged)
	}
}

// P2: notebook bounds against the actual future and present, computed with
// independent rotation arithmetic, for every seat at every position.
func TestNotebookAgainstActualWorld(t *testing.T) {
	deductions, returning, circulating := 0, 0, 0
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 40; seed++ {
			run := privacySweepPlay(t, Setup{Seats: n, Seed: 0x51ed + seed, Options: Options{Sides: SideRule(seed % 3)}}, seed+100)
			for _, g := range run.games {
				st := &g.State
				if st.Age == 0 || st.Phase == PhaseOver {
					continue
				}
				for v := Seat(0); v < Seat(n); v++ {
					o := g.Observe(v)
					s := o.Statistics()
					for _, d := range s.Deductions[:s.NumDeductions] {
						deductions++
						kind := g.Record.Plays[d.Age-1][d.Turn-1][d.Seat].Kind
						if g.Record.Picks[d.Age-1][d.Turn-1][d.Seat] != d.Design || (d.Seat != v.Left(n) && d.Seat != v.Right(n)) || kind == ActBuild {
							t.Fatalf("false neighbour deduction %+v viewer %d at %s", d, v, g.Point().ID())
						}
					}
					resolved := int(st.Turn) - 1
					if st.Phase == PhaseDiscardBuild {
						resolved = int(st.Turn)
					}
					if st.Phase == PhaseLastCard {
						resolved = 6
					}
					if resolved >= 6 {
						continue
					}
					current := resolved + 1
					if st.Phase == PhaseDiscardBuild {
						current = resolved
					}
					dir := 1
					if st.Age == 2 {
						dir = -1
					}
					shift := func(seat, k int) int {
						return ((seat+dir*k)%int(n) + int(n)) % int(n)
					}
					var seenOrigin [MaxSeats]bool
					for turn := 1; turn <= resolved && turn < 6; turn++ {
						seenOrigin[shift(int(v), -(turn-1))] = true
					}
					var actualRet [catalog.NumDesigns + 1]int
					var actualCol [catalog.NumColors]int
					for origin := 0; origin < int(n); origin++ {
						if !seenOrigin[origin] {
							continue
						}
						holder := shift(origin, current-1)
						for _, id := range st.Hands[holder] {
							if id != 0 {
								d, _ := catalog.DesignByID(id)
								actualCol[d.Color]++
							}
						}
						dist := 0
						for h := holder; ; {
							h = shift(h, 1)
							dist++
							if h == int(v) {
								break
							}
						}
						r := current + dist
						if n > 5 || r > 6 {
							continue
						}
						for _, f := range run.games {
							if f.State.Age == st.Age && f.State.Phase == PhaseTurn && int(f.State.Turn) == r {
								for _, id := range f.State.Hands[v] {
									if id != 0 {
										actualRet[id]++
									}
								}
								break
							}
						}
					}
					for colour, b := range s.Circulating {
						circulating++
						if int(b.Certain) > actualCol[colour] || int(b.AtMost) < actualCol[colour] {
							t.Fatalf("circulating bound %+v excludes actual %d colour %d viewer %d at %s", b, actualCol[colour], colour, v, g.Point().ID())
						}
					}
					for id, b := range s.Returning {
						returning++
						if int(b.Certain) > actualRet[id] || int(b.AtMost) < actualRet[id] {
							t.Fatalf("returning bound %+v excludes actual %d of %s viewer %d n%d at %s", b, actualRet[id], designSlug(catalog.DesignID(id)), v, n, g.Point().ID())
						}
					}
				}
			}
		}
	}
	if deductions == 0 || returning == 0 || circulating == 0 {
		t.Fatal("missing notebook paths", deductions, returning, circulating)
	}
	t.Logf("deductions checked %d, returning bounds %d, circulating bounds %d", deductions, returning, circulating)
}
