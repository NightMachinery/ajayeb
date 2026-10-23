package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type haAssignment struct {
	Index int `json:"index"`
	Count int `json:"count"`
}
type haFooter struct {
	Blocks   int    `json:"blocks"`
	SHA256   string `json:"blocks_sha256"`
	PeakHeap uint64 `json:"peak_heap_bytes_sampled_after_block"`
}
type haFrame struct {
	Kind       string        `json:"kind"`
	Header     *haHeader     `json:"header,omitempty"`
	Assignment *haAssignment `json:"assignment,omitempty"`
	Block      *haBlock      `json:"block,omitempty"`
	Footer     *haFooter     `json:"footer,omitempty"`
}

func haBlockKey(n, index int) string { return fmt.Sprintf("%d/%d", n, index) }
func (s haAssignment) validate(h haHeader) error {
	if s.Count < 1 || s.Count > len(h.Seats)*h.Blocks || s.Index < 0 || s.Index >= s.Count {
		return fmt.Errorf("invalid shard assignment")
	}
	return nil
}
func (s haAssignment) has(h haHeader, n, index int) bool {
	for ordinal, count := range h.Seats {
		if count == n {
			return (ordinal*h.Blocks+index)%s.Count == s.Index
		}
	}
	return false
}
func haFrameBytes(frame haFrame) ([]byte, error) {
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
func haDecodeFrame(data []byte) (haFrame, error) {
	var f haFrame
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&f); err != nil {
		return f, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return f, fmt.Errorf("trailing frame data")
	}
	return f, nil
}

type haOutput struct {
	file       *os.File
	buffer     *bufio.Writer
	compressed *gzip.Writer
	writer     io.Writer
}

func haOpenOutput(path string) (*haOutput, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	b := bufio.NewWriterSize(f, 65536)
	out := &haOutput{file: f, buffer: b, writer: b}
	if strings.HasSuffix(path, ".gz") {
		out.compressed = gzip.NewWriter(b)
		out.writer = out.compressed
	}
	return out, nil
}
func (o *haOutput) close() error {
	var err error
	if o.compressed != nil {
		err = o.compressed.Close()
	}
	err = errors.Join(err, o.buffer.Flush(), o.file.Close())
	return err
}
func haWriteTournament(ctx context.Context, h haHeader, assignment haAssignment, path string, progress io.Writer) (haSummary, error) {
	out, err := haOpenOutput(path)
	if err != nil {
		return haSummary{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = out.close()
		}
	}()
	data, err := haFrameBytes(haFrame{Kind: "header", Header: &h, Assignment: &assignment})
	if err != nil {
		return haSummary{}, err
	}
	if _, err = out.writer.Write(data); err != nil {
		return haSummary{}, err
	}
	digest := sha256.New()
	a := haNewAccumulator(h)
	for _, n := range h.Seats {
		for block := 0; block < h.Blocks; block++ {
			if !assignment.has(h, n, block) {
				continue
			}
			b, err := haRunBlock(ctx, h, n, block)
			if err != nil {
				return haSummary{}, err
			}
			data, err := haFrameBytes(haFrame{Kind: "block", Block: &b})
			if err != nil {
				return haSummary{}, err
			}
			if _, err = out.writer.Write(data); err != nil {
				return haSummary{}, err
			}
			_, _ = digest.Write(data)
			a.add(b)
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			a.peakHeap = max(a.peakHeap, mem.HeapAlloc)
			fmt.Fprintf(progress, "completed N=%d block=%d unique-games=%d\n", n, block, len(b.Games))
		}
	}
	footer := haFooter{a.blocks, fmt.Sprintf("%x", digest.Sum(nil)), a.peakHeap}
	data, err = haFrameBytes(haFrame{Kind: "footer", Footer: &footer})
	if err != nil {
		return haSummary{}, err
	}
	if _, err = out.writer.Write(data); err != nil {
		return haSummary{}, err
	}
	err = out.close()
	closed = true
	if err != nil {
		return haSummary{}, err
	}
	if assignment.Count > 1 {
		return haSummary{Header: h, Complete: false, Blocks: a.blocks, Games: a.games, Picks: a.picks, PeakHeapBytes: a.peakHeap}, nil
	}
	return a.finish(), nil
}

// haMerge reads one block at a time, replays its accepted keys, and releases
// its game data after adding the bounded metric tallies. It never averages an
// incomplete block or treats its rotations as independent bootstrap samples.
func haMerge(paths []string) (haSummary, error) {
	if len(paths) == 0 {
		return haSummary{}, fmt.Errorf("missing shard files")
	}
	var header *haHeader
	var accumulator *haAccumulator
	shardCount := 0
	shards := map[int]bool{}
	blocks := map[string]bool{}
	for _, path := range paths {
		if err := func() error {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			var reader io.Reader = file
			if strings.HasSuffix(path, ".gz") {
				zipped, err := gzip.NewReader(file)
				if err != nil {
					return err
				}
				defer zipped.Close()
				reader = zipped
			}
			scanner := bufio.NewScanner(reader)
			scanner.Buffer(make([]byte, 65536), 64<<20)
			if !scanner.Scan() {
				return fmt.Errorf("missing tournament header: %w", errors.Join(scanner.Err(), io.ErrUnexpectedEOF))
			}
			frame, err := haDecodeFrame(scanner.Bytes())
			if err != nil {
				return err
			}
			if frame.Kind != "header" || frame.Header == nil || frame.Assignment == nil || frame.Block != nil || frame.Footer != nil {
				return fmt.Errorf("invalid tournament header frame")
			}
			h, s := *frame.Header, *frame.Assignment
			if err := haCheckHeader(h); err != nil {
				return err
			}
			if err := s.validate(h); err != nil {
				return err
			}
			if header == nil {
				header = &h
				accumulator = haNewAccumulator(h)
				shardCount = s.Count
			} else if !reflect.DeepEqual(*header, h) || shardCount != s.Count {
				return fmt.Errorf("shard header/assignment mismatch")
			}
			if shards[s.Index] {
				return fmt.Errorf("duplicate shard %d", s.Index)
			}
			shards[s.Index] = true
			expected := map[string]bool{}
			for _, n := range h.Seats {
				for b := 0; b < h.Blocks; b++ {
					if s.has(h, n, b) {
						expected[haBlockKey(n, b)] = true
					}
				}
			}
			digest := sha256.New()
			count := 0
			footer := false
			for scanner.Scan() {
				data := scanner.Bytes()
				f, err := haDecodeFrame(data)
				if err != nil {
					return err
				}
				if footer {
					return fmt.Errorf("trailing data after tournament footer")
				}
				switch f.Kind {
				case "block":
					if f.Block == nil || f.Header != nil || f.Assignment != nil || f.Footer != nil {
						return fmt.Errorf("invalid block frame")
					}
					b := *f.Block
					key := haBlockKey(b.N, b.Index)
					if blocks[key] {
						return fmt.Errorf("duplicate block %s", key)
					}
					if !expected[key] {
						return fmt.Errorf("unexpected or duplicate assigned block %s", key)
					}
					if err := haValidateBlock(h, b); err != nil {
						return fmt.Errorf("block %s: %w", key, err)
					}
					delete(expected, key)
					blocks[key] = true
					count++
					_, _ = digest.Write(data)
					_, _ = digest.Write([]byte{'\n'})
					accumulator.add(b)
				case "footer":
					if f.Footer == nil || f.Block != nil || f.Header != nil || f.Assignment != nil {
						return fmt.Errorf("invalid footer")
					}
					if f.Footer.Blocks != count || f.Footer.SHA256 != fmt.Sprintf("%x", digest.Sum(nil)) {
						return fmt.Errorf("tournament footer count/digest mismatch")
					}
					if len(expected) != 0 {
						return fmt.Errorf("missing blocks in shard %d", s.Index)
					}
					accumulator.peakHeap = max(accumulator.peakHeap, f.Footer.PeakHeap)
					footer = true
				default:
					return fmt.Errorf("unexpected frame kind %q", f.Kind)
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			if !footer {
				return fmt.Errorf("missing tournament footer")
			}
			return nil
		}(); err != nil {
			return haSummary{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	if len(shards) != shardCount || len(blocks) != len(header.Seats)*header.Blocks {
		return haSummary{}, fmt.Errorf("missing shard/block: got %d/%d shards and %d/%d blocks", len(shards), shardCount, len(blocks), len(header.Seats)*header.Blocks)
	}
	return accumulator.finish(), nil
}
func haPrintSummary(w io.Writer, s haSummary) {
	fmt.Fprintf(w, "Haftgane policy/preset %d/%d, rules %d, partition %s, suite %s\n", s.Header.Policy, s.Header.Preset, s.Header.Rules, s.Header.Partition, s.Header.Suite)
	fmt.Fprintf(w, "%d complete deal blocks, %d unique games, %d accepted picks; complete=%t\n", s.Blocks, s.Games, s.Picks, s.Complete)
	if !s.Complete {
		fmt.Fprintln(w, "Shard only: no global strength intervals. Use tournament-tally with every shard.")
		return
	}
	for _, c := range s.Contrasts {
		fmt.Fprintf(w, "N=%d %s: blocks=%d rank %.4f [%.4f,%.4f], win %.4f [%.4f,%.4f], paired rank change %.4f [%.4f,%.4f]\n", c.N, c.Name, c.IndependentBlocks, c.Rank.Mean, c.Rank.Low, c.Rank.High, c.Win.Mean, c.Win.Low, c.Win.High, c.DeltaRank.Mean, c.DeltaRank.Low, c.DeltaRank.High)
	}
	fmt.Fprintf(w, "release eligible=%t; release gates passed=%t; self-play midrank sanity=%t\n", s.ReleaseEligible, s.GatesPassed, s.SelfplaySanity)
	for _, g := range s.Gates {
		fmt.Fprintf(w, "gate N=%d %s: pass=%t mean=%.4f target=%.4f 95%%=[%.4f,%.4f] interval-crosses-target=%t\n", g.N, g.Name, g.Pass, g.Metric.Mean, g.Target, g.Metric.Low, g.Metric.High, g.Uncertain)
	}
}
func haSaveSummary(path string, s haSummary) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
func runTournament(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("tournament", stderr)
	game := fs.String("game", "", "required: haftgane (Duel uses its existing commands)")
	partition := fs.String("partition", "development", "development, validation or release")
	blocks := fs.Int("blocks", 100, "independent deal blocks per player count")
	seats := fs.String("seats", "3,4,5,6,7", "distinct player counts, 3..7")
	suite := fs.String("suite", "all", "release, diagnostics or all")
	shardText := fs.String("shard", "0/1", "complete block assignment K/N")
	path := fs.String("out", "", "required JSONL report path (.gz supported); must not exist")
	summaryPath := fs.String("summary", "", "write full metric summary JSON (unsharded runs only)")
	check := fs.Bool("check-gates", false, "fail unless a complete eligible held-out run passes every release gate")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	ns := []int{}
	for _, part := range strings.Split(*seats, ",") {
		n, err := strconv.Atoi(part)
		if err != nil {
			fmt.Fprintln(stderr, "invalid player count")
			return errUsage
		}
		ns = append(ns, n)
	}
	h, err := haNewHeader(*partition, *blocks, ns, *suite)
	parsed, shardErr := parseShard(*shardText)
	assignment := haAssignment{parsed.k, parsed.n}
	if err != nil || shardErr != nil || *game != "haftgane" || *path == "" || fs.NArg() != 0 {
		fmt.Fprintf(stderr, "tournament requires -game haftgane, -out, valid partition/counts/suite/shard: %v %v\n", err, shardErr)
		return errUsage
	}
	if err := assignment.validate(h); err != nil {
		fmt.Fprintln(stderr, err)
		return errUsage
	}
	if assignment.Count > 1 && (*summaryPath != "" || *check) {
		fmt.Fprintln(stderr, "global summary/gates require tournament-tally after all shards")
		return errUsage
	}
	s, err := haWriteTournament(context.Background(), h, assignment, *path, stderr)
	if err != nil {
		return err
	}
	haPrintSummary(stdout, s)
	if err := haSaveSummary(*summaryPath, s); err != nil {
		return err
	}
	if *check && !s.GatesPassed {
		return errFailed
	}
	return nil
}
func runTournamentTally(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("tournament-tally", stderr)
	game := fs.String("game", "", "required: haftgane")
	summaryPath := fs.String("summary", "", "write complete metric summary JSON")
	check := fs.Bool("check-gates", false, "fail unless the complete held-out run passes every release gate")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *game != "haftgane" || fs.NArg() == 0 {
		fmt.Fprintln(stderr, "tournament-tally requires -game haftgane and every shard file")
		return errUsage
	}
	s, err := haMerge(fs.Args())
	if err != nil {
		return err
	}
	haPrintSummary(stdout, s)
	if err := haSaveSummary(*summaryPath, s); err != nil {
		return err
	}
	if *check && !s.GatesPassed {
		return errFailed
	}
	return nil
}
