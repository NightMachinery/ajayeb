package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	hbot "github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

const haFormat = "haftgane-tournament/1"
const haPolicyBase uint64 = 0x706f6c6963797631
const haBootstrapBase uint64 = 0x626f6f7473747261

type haHeader struct {
	Format        string    `json:"format"`
	Game          string    `json:"game"`
	Rules         int       `json:"rules"`
	Catalog       string    `json:"catalog_sha256"`
	Policy        int       `json:"policy"`
	Preset        int       `json:"preset"`
	Work          hbot.Work `json:"work"`
	Temperature   int       `json:"temperature_millivp"`
	Partition     string    `json:"partition"`
	DealBase      uint64    `json:"deal_base"`
	PolicyBase    uint64    `json:"policy_base"`
	BootstrapBase uint64    `json:"bootstrap_base"`
	Derivation    int       `json:"derivation"`
	SideSchedule  string    `json:"side_schedule"`
	Blocks        int       `json:"blocks_per_n"`
	Seats         []int     `json:"seats"`
	Suite         string    `json:"suite"`
}

func haDigest(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	} // Only concrete integer/string rule/report values.
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
func haNewHeader(partition string, blocks int, seats []int, suite string) (haHeader, error) {
	var base uint64
	switch partition {
	case "development":
		base = 0x6d170001
	case "validation":
		base = 0x76616c6964617465
	case "release":
		base = 0x72656c6561736531
	default:
		return haHeader{}, fmt.Errorf("unknown seed partition %q", partition)
	}
	if blocks < 1 || blocks > 10000 || len(seats) < 1 || len(seats) > 5 {
		return haHeader{}, fmt.Errorf("want 1..10000 blocks and distinct counts 3..7")
	}
	ns := append([]int(nil), seats...)
	sort.Ints(ns)
	for i, n := range ns {
		if n < 3 || n > 7 || i > 0 && n == ns[i-1] {
			return haHeader{}, fmt.Errorf("invalid/duplicate seat count %d", n)
		}
	}
	if suite != "release" && suite != "diagnostics" && suite != "all" {
		return haHeader{}, fmt.Errorf("unknown suite %q", suite)
	}
	designs := make([]catalog.Design, 0, catalog.NumDesigns)
	boards := make([]catalog.Board, 0, catalog.NumBoards)
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		d, _ := catalog.DesignByID(id)
		designs = append(designs, d)
	}
	for id := catalog.BoardID(1); id <= catalog.NumBoards; id++ {
		b, _ := catalog.BoardByID(id)
		boards = append(boards, b)
	}
	digest := haDigest(struct {
		Designs []catalog.Design
		Boards  []catalog.Board
	}{designs, boards})
	return haHeader{haFormat, "haftgane", rules.RulesVersion, digest, hbot.PolicyVersion, hbot.PresetVersion, hbot.DefaultWork(), hbot.ApprenticeTemperature, partition, base, haPolicyBase, haBootstrapBase, 1, "day/night/random by block modulo 3", blocks, ns, suite}, nil
}
func haCheckHeader(h haHeader) error {
	want, err := haNewHeader(h.Partition, h.Blocks, h.Seats, h.Suite)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(h, want) {
		return fmt.Errorf("unsupported or mismatched tournament header/version/work/catalog")
	}
	return nil
}
func haDealSeed(h haHeader, n, block int) uint64 {
	return rng.MixSeed(h.DealBase, uint64(block)*5+uint64(n-3))
}
func haInitial(h haHeader, n, block int) (rules.Game, error) {
	g, err := rules.NewGame(rules.Setup{Seed: haDealSeed(h, n, block), Seats: uint8(n), Options: rules.Options{Sides: rules.SidesRandom, HideAtEnd: true}})
	if err != nil {
		return rules.Game{}, err
	}
	sc := rules.ScenarioFromGame(g)
	for s := 0; s < n; s++ {
		if block%3 == 0 {
			sc.State.Cities[s].Side = catalog.Day
		}
		if block%3 == 1 {
			sc.State.Cities[s].Side = catalog.Night
		}
	}
	// Explicit side strata preserve every dealt copy and physical board. No bot
	// gets a side-selection point that could change its paired setup.
	return rules.NewGameFromScenario(g.State.Seed, g.State.Opts, sc)
}
func haPolicyEntropy(h haHeader, n, block int, p rules.Point, seat rules.Seat) uint64 {
	root := rng.MixSeed(h.PolicyBase, uint64(block)*5+uint64(n-3))
	// Logical point coordinates, never accepted-batch count, lineup, arrival
	// order or setup root. An extra ability point cannot shift later tapes.
	return rng.MixSeed(root, uint64(p.Kind)*1024+uint64(p.Age)*128+uint64(p.Turn)*8+uint64(seat))
}

type haComparison struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Focal     int                        `json:"focal"`
	Offset    int                        `json:"offset"`
	Reflected bool                       `json:"reflected"`
	Candidate [rules.MaxSeats]hbot.Level `json:"candidate"`
	Baseline  [rules.MaxSeats]hbot.Level `json:"baseline"`
}

func haPlan(n int, suite string) []haComparison {
	out := []haComparison{}
	add := func(name string, focal, offset int, reflected bool, candidate, baseline [rules.MaxSeats]hbot.Level) {
		out = append(out, haComparison{fmt.Sprintf("%s/%d/%d/%t", name, focal, offset, reflected), name, focal, offset, reflected, candidate, baseline})
	}
	homogeneous := func(level hbot.Level) (lineup [rules.MaxSeats]hbot.Level) {
		for s := 0; s < n; s++ {
			lineup[s] = level
		}
		return lineup
	}
	if suite == "release" || suite == "all" {
		for focal := 0; focal < n; focal++ {
			for _, level := range []hbot.Level{hbot.Greedy, hbot.Apprentice} {
				base := homogeneous(hbot.Random)
				candidate := base
				candidate[focal] = level
				add(level.String()+"-random", focal, 0, false, candidate, base)
			}
			for offset := 0; offset < n-1; offset++ {
				for _, reflected := range []bool{false, true} {
					base := homogeneous(hbot.Random)
					for d := 1; d < n; d++ {
						distance := d
						if reflected {
							distance = n - d
						}
						base[(focal+d)%n] = [3]hbot.Level{hbot.Greedy, hbot.Apprentice, hbot.Random}[(distance-1+offset)%(n-1)%3]
					}
					candidate := base
					base[focal] = hbot.Apprentice
					candidate[focal] = hbot.Greedy
					add("mixed-greedy-apprentice", focal, offset, reflected, candidate, base)
				}
			}
		}
	}
	if suite == "diagnostics" || suite == "all" {
		for focal := 0; focal < n; focal++ {
			for _, level := range []hbot.Level{hbot.Greedy, hbot.Apprentice} {
				candidate := homogeneous(level)
				base := candidate
				base[focal] = hbot.Random
				add("inverse-"+level.String(), focal, 0, false, candidate, base)
			}
			for level := hbot.Random; level < hbot.NumLevels; level++ {
				line := homogeneous(level)
				add("selfplay-"+level.String(), focal, 0, false, line, line)
			}
		}
	}
	return out
}
func haLineupKey(line [rules.MaxSeats]hbot.Level, n int) string {
	names := make([]string, 0, n)
	for _, l := range line[:n] {
		names = append(names, l.String())
	}
	return strings.Join(names, "/")
}

type haGame struct {
	ID        string                     `json:"id"`
	Lineup    [rules.MaxSeats]hbot.Level `json:"lineup"`
	Result    rules.Result               `json:"result"`
	Coins     [rules.MaxSeats]int16      `json:"coins"`
	FinalHash string                     `json:"final_sha256"`
	Batches   []rules.Batch              `json:"batches"`
	// One tuple per accepted pick in batch/seat order:
	// seat, point kind, active ns, iterations, ranked, previews, quotes, draws,
	// fallback reason (0 normal). Keys are already present in Batches.
	Diagnostics [][]int64 `json:"pick_diagnostics"`
}
type haBlock struct {
	N           int                             `json:"n"`
	Index       int                             `json:"block"`
	Seed        uint64                          `json:"seed"`
	InitialHash string                          `json:"initial_sha256"`
	Boards      [rules.MaxSeats]catalog.BoardID `json:"boards"`
	Sides       [rules.MaxSeats]catalog.Side    `json:"sides"`
	Comparisons []haComparison                  `json:"comparisons"`
	Games       []haGame                        `json:"games"`
}

func haPlay(ctx context.Context, h haHeader, n, block int, start rules.Game, line [rules.MaxSeats]hbot.Level) (haGame, error) {
	out := haGame{ID: haLineupKey(line, n), Lineup: line, Batches: []rules.Batch{}, Diagnostics: [][]int64{}}
	g := start
	for g.State.Phase != rules.PhaseOver {
		if len(out.Batches) >= rules.MaxPoints {
			return haGame{}, fmt.Errorf("nonterminating tournament game")
		}
		point := g.Point()
		batch := rules.Batch{Point: point.ID(), Picks: []rules.Pick{}}
		for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
			if !point.Required.Has(seat) {
				continue
			}
			offers := g.Legal(seat)
			observation := g.Observe(seat)
			t0 := time.Now()
			d, err := hbot.Choose(ctx, line[seat], observation, offers, haPolicyEntropy(h, n, block, point, seat), h.Work)
			ns := time.Since(t0).Nanoseconds()
			if err != nil || !d.Complete {
				return haGame{}, fmt.Errorf("policy %s at %s seat %d incomplete: %w", line[seat], point.ID(), seat, err)
			}
			found := false
			for _, c := range offers {
				found = found || c.Key() == d.Key
			}
			if !found {
				return haGame{}, fmt.Errorf("policy returned unoffered key %q", d.Key)
			}
			c := d.Counters
			out.Diagnostics = append(out.Diagnostics, []int64{int64(seat), int64(point.Kind), ns, int64(c.Iterations), int64(c.Ranked), int64(c.Previews), int64(c.Quotes), int64(c.RandomDraws), 0})
			batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: d.Key})
		}
		var err error
		g, _, err = g.Resolve(batch)
		if err != nil {
			return haGame{}, err
		}
		out.Batches = append(out.Batches, batch)
	}
	out.Result = g.State.Result
	out.FinalHash = haDigest(g)
	for s := 0; s < n; s++ {
		out.Coins[s] = g.State.Cities[s].Coins
	}
	return out, nil
}
func haRunBlock(ctx context.Context, h haHeader, n, block int) (haBlock, error) {
	start, err := haInitial(h, n, block)
	if err != nil {
		return haBlock{}, err
	}
	out := haBlock{N: n, Index: block, Seed: start.State.Seed, InitialHash: haDigest(start), Comparisons: haPlan(n, h.Suite), Games: []haGame{}}
	for s := 0; s < n; s++ {
		out.Boards[s] = start.State.Cities[s].Board
		out.Sides[s] = start.State.Cities[s].Side
	}
	seen := map[string]bool{}
	for _, comparison := range out.Comparisons {
		for _, line := range [2][rules.MaxSeats]hbot.Level{comparison.Candidate, comparison.Baseline} {
			key := haLineupKey(line, n)
			if seen[key] {
				continue
			}
			seen[key] = true
			game, err := haPlay(ctx, h, n, block, start, line)
			if err != nil {
				return haBlock{}, err
			}
			out.Games = append(out.Games, game)
		}
	}
	sort.Slice(out.Games, func(i, j int) bool { return out.Games[i].ID < out.Games[j].ID })
	return out, nil
}
func haValidateBlock(h haHeader, b haBlock) error {
	if b.N < 3 || b.N > 7 || b.Index < 0 || b.Index >= h.Blocks {
		return fmt.Errorf("invalid block count/index")
	}
	has := false
	for _, n := range h.Seats {
		has = has || n == b.N
	}
	if !has {
		return fmt.Errorf("unexpected block player count")
	}
	start, err := haInitial(h, b.N, b.Index)
	if err != nil {
		return err
	}
	if b.Seed != start.State.Seed || b.InitialHash != haDigest(start) {
		return fmt.Errorf("block setup identity mismatch")
	}
	var boards [rules.MaxSeats]catalog.BoardID
	var sides [rules.MaxSeats]catalog.Side
	for s := 0; s < b.N; s++ {
		boards[s] = start.State.Cities[s].Board
		sides[s] = start.State.Cities[s].Side
	}
	if b.Boards != boards || b.Sides != sides {
		return fmt.Errorf("block board/side assignment mismatch")
	}
	plan := haPlan(b.N, h.Suite)
	if !reflect.DeepEqual(b.Comparisons, plan) {
		return fmt.Errorf("missing, duplicate or changed comparison/rotation/lineup")
	}
	expected := map[string][rules.MaxSeats]hbot.Level{}
	for _, c := range plan {
		for _, line := range [2][rules.MaxSeats]hbot.Level{c.Candidate, c.Baseline} {
			expected[haLineupKey(line, b.N)] = line
		}
	}
	seen := map[string]bool{}
	for _, game := range b.Games {
		line, ok := expected[game.ID]
		if !ok || game.Lineup != line {
			return fmt.Errorf("unexpected game lineup %q", game.ID)
		}
		if seen[game.ID] {
			return fmt.Errorf("duplicate game %q", game.ID)
		}
		seen[game.ID] = true
		g := start
		pick := 0
		if len(game.Batches) > rules.MaxPoints {
			return fmt.Errorf("too many game batches")
		}
		for _, batch := range game.Batches {
			p := g.Point()
			last := rules.NoSeat
			for i, pk := range batch.Picks {
				if !pk.Seat.Valid(uint8(b.N)) || i > 0 && pk.Seat <= last || !p.Required.Has(pk.Seat) {
					return fmt.Errorf("noncanonical game picks")
				}
				last = pk.Seat
				if pick >= len(game.Diagnostics) {
					return fmt.Errorf("missing pick diagnostic")
				}
				d := game.Diagnostics[pick]
				pick++
				if len(d) != 9 || d[0] != int64(pk.Seat) || d[1] != int64(p.Kind) || d[2] < 0 || d[3] != 1 || d[8] != 0 {
					return fmt.Errorf("invalid/incomplete pick diagnostic")
				}
				offers := g.Legal(pk.Seat)
				ranked, draws := int64(len(offers)), int64(0)
				if line[pk.Seat] == hbot.Random {
					ranked = 0
					draws = 1
					if g.State.Phase == rules.PhaseTurn {
						draws = 3
					}
				} else if line[pk.Seat] == hbot.Apprentice && len(offers) > 1 {
					draws = 1
				}
				if d[4] != ranked || d[5] != ranked || d[6] < 0 || d[6] > int64(hbot.MaxQuoteCalls) || line[pk.Seat] == hbot.Random && d[6] != 0 || d[7] != draws {
					return fmt.Errorf("inconsistent work diagnostic")
				}
			}
			g, _, err = g.Resolve(batch)
			if err != nil {
				return fmt.Errorf("game replay: %w", err)
			}
		}
		if pick != len(game.Diagnostics) {
			return fmt.Errorf("extra pick diagnostic")
		}
		var coins [rules.MaxSeats]int16
		for s := 0; s < b.N; s++ {
			coins[s] = g.State.Cities[s].Coins
		}
		if g.State.Phase != rules.PhaseOver || !game.Result.Over || game.Result != g.State.Result || coins != game.Coins || game.FinalHash != haDigest(g) {
			return fmt.Errorf("game terminal result/hash mismatch")
		}
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("missing game lineup")
	}
	return nil
}
