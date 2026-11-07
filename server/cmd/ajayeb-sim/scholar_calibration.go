package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	hbot "github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

type scProfile struct {
	ID   string       `json:"id"`
	Work hbot.AgeWork `json:"work"`
}
type scConfig struct {
	Format     string      `json:"format"`
	Binary     string      `json:"binary_sha256"`
	Catalog    string      `json:"catalog_sha256"`
	Partition  string      `json:"partition"`
	Epoch      int         `json:"epoch"`
	Blocks     int         `json:"blocks_per_n"`
	Seats      []int       `json:"seats"`
	Profiles   []scProfile `json:"profiles"`
	Suite      string      `json:"suite"`
	Frozen     bool        `json:"frozen"`
	GOMAXPROCS int         `json:"gomaxprocs"`
	Load       int         `json:"load_workers"`
}
type scComparison struct {
	Family    string `json:"family"`
	Focal     int    `json:"focal"`
	Offset    int    `json:"offset"`
	Reflected bool   `json:"reflected"`
	Candidate string `json:"candidate"`
	Baseline  string `json:"baseline"`
}

func scPlan(n int, suite string) []scComparison {
	var plan []scComparison
	allG, allS := strings.Repeat("G", n), strings.Repeat("S", n)
	for focal := 0; focal < n; focal++ {
		one := []byte(allG)
		one[focal] = 'S'
		inverse := []byte(allS)
		inverse[focal] = 'G'
		plan = append(plan, scComparison{"main", focal, 0, false, string(one), allG}, scComparison{"inverse", focal, 0, false, allS, string(inverse)})
		offsets := n - 1
		if suite == "screen" {
			offsets = 1
		}
		for offset := 0; offset < offsets; offset++ {
			for _, reflected := range []bool{false, true} {
				base := []byte(allG)
				for d := 1; d < n; d++ {
					distance := d
					if reflected {
						distance = n - d
					}
					if (distance-1+offset)%(n-1)%2 == 0 {
						base[(focal+d)%n] = 'S'
					}
				}
				candidate := append([]byte(nil), base...)
				candidate[focal] = 'S'
				plan = append(plan, scComparison{"mixed", focal, offset, reflected, string(candidate), string(base)})
			}
		}
	}
	return plan
}
func scLineups(plan []scComparison) []string {
	var out []string
	for _, c := range plan {
		for _, s := range []string{c.Candidate, c.Baseline} {
			if !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	}
	slices.Sort(out)
	return out
}
func scHeader(c scConfig) (haHeader, error) {
	h, err := haNewHeader("development", c.Blocks, c.Seats, "all")
	if err != nil {
		return h, err
	}
	var domain uint64
	switch c.Partition {
	case "development":
		domain = 0x7363682d64657631
	case "validation":
		domain = 0x7363682d76616c31
	case "heldout":
		domain = 0x7363682d686f6c31
	default:
		return h, fmt.Errorf("unknown calibration partition")
	}
	h.DealBase = rng.MixSeed(domain, uint64(c.Epoch))
	h.PolicyBase = rng.MixSeed(0x7363682d706f6c31, uint64(c.Epoch))
	return h, nil
}

type scPick struct {
	Seat     int              `json:"seat"`
	Phase    rules.Phase      `json:"phase"`
	Scholar  bool             `json:"scholar"`
	NS       int64            `json:"ns"`
	GapNS    int64            `json:"max_poll_gap_ns"`
	Counters hbot.AgeCounters `json:"counters"`
}
type scGame struct {
	Lineup    string                `json:"lineup"`
	Result    rules.Result          `json:"result"`
	Coins     [rules.MaxSeats]int16 `json:"coins"`
	Batches   []rules.Batch         `json:"batches"`
	Picks     []scPick              `json:"picks"`
	Turns     []int64               `json:"serialized_scholar_service_ns_per_point"`
	WallNS    int64                 `json:"game_compute_ns"`
	Allocated uint64                `json:"allocated_bytes"`
	Allocs    uint64                `json:"allocations"`
	HeapAfter uint64                `json:"heap_after_game_bytes"`
	FinalSHA  string                `json:"final_sha256"`
}
type scBlock struct {
	ConfigSHA   string         `json:"config_sha256"`
	Profile     string         `json:"profile"`
	N           int            `json:"n"`
	Index       int            `json:"index"`
	Comparisons []scComparison `json:"comparisons"`
	Games       []scGame       `json:"games"`
}
type scEnvelope struct {
	Block scBlock `json:"block"`
	SHA   string  `json:"sha256"`
}

type scGapContext struct {
	context.Context
	last time.Time
	gap  time.Duration
}

func (c *scGapContext) Err() error {
	now := time.Now()
	c.gap = max(c.gap, now.Sub(c.last))
	c.last = now
	return c.Context.Err()
}

func scPlay(ctx context.Context, h haHeader, n, index int, start rules.Game, line string, profile scProfile) (scGame, error) {
	out := scGame{Lineup: line, Batches: []rules.Batch{}, Picks: []scPick{}, Turns: []int64{}}
	g := start
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	began := time.Now()
	for g.State.Phase != rules.PhaseOver {
		point := g.Point()
		batch := rules.Batch{Point: point.ID(), Picks: []rules.Pick{}}
		var serial int64
		for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
			if !point.Required.Has(seat) {
				continue
			}
			o, cs := g.Observe(seat), g.Legal(seat)
			entropy := haPolicyEntropy(h, n, index, point, seat)
			pick := scPick{Seat: int(seat), Phase: g.State.Phase, Scholar: line[seat] == 'S'}
			var key string
			t0 := time.Now()
			if pick.Scholar {
				controlled := &scGapContext{Context: ctx, last: t0}
				d, err := hbot.ChooseAge(controlled, o, cs, entropy, profile.Work)
				controlled.gap = max(controlled.gap, time.Since(controlled.last))
				if err != nil || !d.Complete {
					return out, fmt.Errorf("scholar incomplete: %w", err)
				}
				key, pick.Counters, pick.GapNS = d.Key, d.Counters, controlled.gap.Nanoseconds()
			} else {
				d, err := hbot.Choose(ctx, hbot.Greedy, o, cs, entropy, hbot.DefaultWork())
				if err != nil || !d.Complete {
					return out, fmt.Errorf("greedy incomplete: %w", err)
				}
				key = d.Key
			}
			pick.NS = time.Since(t0).Nanoseconds()
			if pick.Scholar {
				serial += pick.NS
			}
			found := false
			for _, c := range cs {
				found = found || c.Key() == key
			}
			if !found {
				return out, fmt.Errorf("unoffered calibration key")
			}
			out.Picks = append(out.Picks, pick)
			batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: key})
		}
		var err error
		g, _, err = g.Resolve(batch)
		if err != nil {
			return out, err
		}
		out.Batches = append(out.Batches, batch)
		if serial > 0 {
			out.Turns = append(out.Turns, serial)
		}
	}
	out.WallNS = time.Since(began).Nanoseconds()
	runtime.ReadMemStats(&after)
	out.Allocated, out.Allocs, out.HeapAfter = after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, after.HeapAlloc
	out.Result, out.FinalSHA = g.State.Result, haDigest(g)
	for seat := 0; seat < n; seat++ {
		out.Coins[seat] = g.State.Cities[seat].Coins
	}
	return out, nil
}
func scRunBlock(ctx context.Context, c scConfig, p scProfile, n, index int) (scBlock, error) {
	h, err := scHeader(c)
	if err != nil {
		return scBlock{}, err
	}
	start, err := haInitial(h, n, index)
	if err != nil {
		return scBlock{}, err
	}
	b := scBlock{ConfigSHA: haDigest(c), Profile: p.ID, N: n, Index: index, Comparisons: scPlan(n, c.Suite), Games: []scGame{}}
	for _, line := range scLineups(b.Comparisons) {
		game, err := scPlay(ctx, h, n, index, start, line, p)
		if err != nil {
			return b, err
		}
		b.Games = append(b.Games, game)
	}
	return b, nil
}
func scValidate(c scConfig, p scProfile, b scBlock) error {
	if b.ConfigSHA != haDigest(c) || b.Profile != p.ID || b.Index < 0 || b.Index >= c.Blocks || !slices.Contains(c.Seats, b.N) || !reflect.DeepEqual(b.Comparisons, scPlan(b.N, c.Suite)) {
		return fmt.Errorf("calibration block identity/plan mismatch")
	}
	lines := scLineups(b.Comparisons)
	if len(b.Games) != len(lines) {
		return fmt.Errorf("missing calibration lineups")
	}
	h, err := scHeader(c)
	if err != nil {
		return err
	}
	start, err := haInitial(h, b.N, b.Index)
	if err != nil {
		return err
	}
	for i, game := range b.Games {
		if game.Lineup != lines[i] {
			return fmt.Errorf("missing/duplicate/changed lineup")
		}
		g := start
		diagnostics := 0
		for _, batch := range game.Batches {
			for _, pick := range batch.Picks {
				if !pick.Seat.Valid(uint8(b.N)) {
					return fmt.Errorf("invalid replay seat")
				}
				if diagnostics >= len(game.Picks) {
					return fmt.Errorf("missing pick diagnostics")
				}
				d := game.Picks[diagnostics]
				diagnostics++
				if d.Seat != int(pick.Seat) || d.Phase != g.State.Phase || d.Scholar != (game.Lineup[pick.Seat] == 'S') || d.NS < 0 || d.GapNS < 0 {
					return fmt.Errorf("invalid pick diagnostics")
				}
				if d.Scholar && (d.Counters.SampleAttempts != int(p.Work.Attempts) || d.Counters.SamplerChecks > int(p.Work.Attempts)*p.Work.SamplerChecks || d.Counters.CompletedSweeps+d.Counters.NoWorldSamples != int(p.Work.Attempts)) {
					return fmt.Errorf("work allowance changed")
				}
			}
			g, _, err = g.Resolve(batch)
			if err != nil {
				return err
			}
		}
		var coins [rules.MaxSeats]int16
		for seat := 0; seat < b.N; seat++ {
			coins[seat] = g.State.Cities[seat].Coins
		}
		if diagnostics != len(game.Picks) || g.State.Phase != rules.PhaseOver || game.Result != g.State.Result || game.Coins != coins || game.FinalSHA != haDigest(g) {
			return fmt.Errorf("calibration replay/final mismatch")
		}
	}
	return nil
}

func scReadJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}
func scPublishJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".scholar-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Link(tmp, path) // Atomic, exclusive publication; never overwrite evidence.
}
func scReadBlock(path string, c scConfig, p scProfile) (scBlock, error) {
	var e scEnvelope
	if err := scReadJSON(path, &e); err != nil {
		return e.Block, err
	}
	if filepath.Base(path) != filepath.Base(scBlockPath(filepath.Dir(path), p, e.Block.N, e.Block.Index)) {
		return e.Block, fmt.Errorf("calibration block filename/identity mismatch")
	}
	if haDigest(e.Block) != e.SHA {
		return e.Block, fmt.Errorf("calibration block digest mismatch")
	}
	return e.Block, scValidate(c, p, e.Block)
}
func scBlockPath(dir string, p scProfile, n, index int) string {
	return filepath.Join(dir, fmt.Sprintf("%s-n%d-b%04d.json", p.ID, n, index))
}

func scProfiles(text string, checks int) ([]scProfile, error) {
	var profiles []scProfile
	for _, spec := range strings.Split(text, ",") {
		parts := strings.Split(spec, "x")
		if len(parts) != 2 {
			return nil, fmt.Errorf("profile must be attemptsxcandidates")
		}
		a, ea := strconv.Atoi(parts[0])
		k, ek := strconv.Atoi(parts[1])
		if ea != nil || ek != nil || a < 0 || a > 65535 || k < 1 || k > rules.MaxTurnChoices {
			return nil, fmt.Errorf("invalid profile %q", spec)
		}
		p := scProfile{fmt.Sprintf("a%d-c%d", a, k), hbot.AgeWork{Attempts: uint16(a), Candidates: uint16(k), SamplerChecks: checks}}
		for _, prior := range profiles {
			if prior.ID == p.ID {
				return nil, fmt.Errorf("duplicate profile")
			}
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}
func scBinarySHA() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
func scLoadWorkers(n int) func() {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var sink atomic.Uint64
	for worker := 0; worker < n; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			x := uint64(1)
			for {
				select {
				case <-stop:
					sink.Add(x)
					return
				default:
					for k := 0; k < 1000; k++ {
						x = x*6364136223846793005 + 1
					}
				}
			}
		}()
	}
	return func() { close(stop); wg.Wait() }
}
func runScholarCalibration(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("scholar-calibrate", stderr)
	mode := fs.String("mode", "run", "run/resume, tally or plan (offline experiment only)")
	partition := fs.String("partition", "development", "development, validation or heldout")
	epoch := fs.Int("epoch", 1, "fresh seed namespace; change after any heldout-guided revision")
	blocks := fs.Int("blocks", 1, "independent blocks per player count")
	counts := fs.String("counts", "3,4,5,6,7", "player counts")
	profilesText := fs.String("profiles", "1x8,2x8", "experimental attempts x candidate cap")
	checks := fs.Int("sampler-checks", rules.MaxSamplerChecks, "deterministic per-attempt sampler bound")
	suite := fs.String("suite", "screen", "screen or all offset/reflection treatments")
	out := fs.String("out", "", "private experiment directory; exclusive files resume after replay validation")
	shard := fs.String("shard", "0/1", "block assignment K/N")
	load := fs.Int("load", 0, "additional continuously computing goroutines")
	frozen := fs.Bool("frozen", false, "predeclared row frozen before heldout (does not enable rooms)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *epoch < 1 || *blocks < 1 || *blocks > 10000 || *checks < 0 || *checks > rules.MaxSamplerChecks || *load < 0 || *load > 4 || (*suite != "screen" && *suite != "all") || (*mode != "plan" && *mode != "run" && *mode != "tally") {
		return fmt.Errorf("invalid calibration flags")
	}
	profiles, err := scProfiles(*profilesText, *checks)
	if err != nil {
		return err
	}
	var seats []int
	for _, s := range strings.Split(*counts, ",") {
		n, err := strconv.Atoi(s)
		if err != nil || n < 3 || n > 7 || slices.Contains(seats, n) {
			return fmt.Errorf("invalid counts")
		}
		seats = append(seats, n)
	}
	slices.Sort(seats)
	binary, err := scBinarySHA()
	if err != nil {
		return err
	}
	h, err := haNewHeader("development", *blocks, seats, "all")
	if err != nil {
		return err
	}
	c := scConfig{"scholar-calibration/1", binary, h.Catalog, *partition, *epoch, *blocks, seats, profiles, *suite, *frozen, runtime.GOMAXPROCS(0), *load}
	if _, err = scHeader(c); err != nil {
		return err
	}
	if c.Partition == "heldout" && (!c.Frozen || c.Blocks < 100 || len(c.Profiles) != 1 || !reflect.DeepEqual(c.Seats, []int{3, 4, 5, 6, 7}) || c.Suite != "all" || c.Profiles[0].Work.Attempts == 0 || *checks != rules.MaxSamplerChecks) {
		return fmt.Errorf("heldout needs one frozen nonzero row, >=100 blocks at every N, all treatments and full sampler bound")
	}
	if *mode == "plan" {
		return json.NewEncoder(stdout).Encode(scPlanSummary(c))
	}
	if *out == "" {
		return fmt.Errorf("missing -out")
	}
	if err = os.MkdirAll(*out, 0700); err != nil {
		return err
	}
	manifest := filepath.Join(*out, "config.json")
	var prior scConfig
	if err = scReadJSON(manifest, &prior); err == nil {
		if !reflect.DeepEqual(prior, c) {
			return fmt.Errorf("experiment identity differs; use a new directory")
		}
	} else if os.IsNotExist(err) {
		if err = scPublishJSON(manifest, c); err != nil {
			if !os.IsExist(err) {
				return err
			}
			// Disjoint CI shards may initialize the shared manifest together.
			if err = scReadJSON(manifest, &prior); err != nil {
				return err
			}
			if !reflect.DeepEqual(prior, c) {
				return fmt.Errorf("experiment identity differs; use a new directory")
			}
		}
	} else {
		return err
	}
	if *mode == "run" {
		parts := strings.Split(*shard, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid shard")
		}
		k, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 != nil || e2 != nil || m < 1 || k < 0 || k >= m {
			return fmt.Errorf("invalid shard")
		}
		stop := scLoadWorkers(c.Load)
		defer stop()
		ordinal := 0
		for _, p := range c.Profiles {
			for _, n := range c.Seats {
				for index := 0; index < c.Blocks; index++ {
					assigned := ordinal%m == k
					ordinal++
					if !assigned {
						continue
					}
					path := scBlockPath(*out, p, n, index)
					if _, err = scReadBlock(path, c, p); err == nil {
						fmt.Fprintf(stderr, "resumed %s N=%d block=%d\n", p.ID, n, index)
						continue
					} else if !os.IsNotExist(err) {
						return err
					}
					t0 := time.Now()
					b, err := scRunBlock(context.Background(), c, p, n, index)
					if err != nil {
						return err
					}
					if err = scValidate(c, p, b); err != nil {
						return err
					}
					if err = scPublishJSON(path, scEnvelope{b, haDigest(b)}); err != nil {
						return err
					}
					fmt.Fprintf(stderr, "completed %s N=%d block=%d games=%d seconds=%.3f\n", p.ID, n, index, len(b.Games), time.Since(t0).Seconds())
				}
			}
		}
	}
	summary, err := scSummarize(*out, c)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(summary)
}
