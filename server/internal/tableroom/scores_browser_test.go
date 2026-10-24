package tableroom

// The raw browser recording is private test output, not a persisted fixture.
// Run this after the end-of-game spec (e2e/specs/haftgane-scores.full.spec.ts),
// passing its frames-*.json files, one per end policy, in
// AJAYEB_SCORES_BROWSER_FRAMES (the operating system's path-list separator).
import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestScoresBrowserFrames runs the leak oracle over every table frame the
// seated player and the spectator received, from the first turn to the
// final scores, and then proves it reads the end: a changed total, a
// download flag the end policy does not grant, and, under the hidden end,
// a planted reveal of the box's guilds each fail it.
func TestScoresBrowserFrames(t *testing.T) {
	paths := os.Getenv("AJAYEB_SCORES_BROWSER_FRAMES")
	if paths == "" {
		t.Skip("run the end-of-game spec first")
	}
	for _, path := range filepath.SplitList(paths) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var capture struct {
				Record Record `json:"record"`
				Logs   [][]struct {
					Dir string          `json:"dir"`
					Msg json.RawMessage `json:"msg"`
				} `json:"logs"`
			}
			if err = json.Unmarshal(data, &capture); err != nil {
				t.Fatal(err)
			}
			setup, err := recordSetup(capture.Record.Setup)
			if err != nil {
				t.Fatal(err)
			}
			batches := []rules.Batch{}
			for _, b := range capture.Record.Batches {
				next := rules.Batch{Point: b.Point}
				for _, p := range b.Picks {
					next.Picks = append(next.Picks, rules.Pick{Seat: rules.Seat(p.Seat), Key: p.Key})
				}
				batches = append(batches, next)
			}
			truth, err := leakcheck.FromReal(setup, batches)
			if err != nil {
				t.Fatal(err)
			}
			checker := leakcheck.New(truth, frameConfig)
			hidden := capture.Record.Setup.HideAtEnd
			seated, watched := 0, 0
			for i, log := range capture.Logs {
				var states []wire.TableState
				var raws [][]byte
				var viewers []frameViewer
				client := ""
				for _, entry := range log {
					var tag struct {
						T      string `json:"t"`
						Client string `json:"clientId"`
					}
					if err = json.Unmarshal(entry.Msg, &tag); err != nil {
						t.Fatal(err)
					}
					if entry.Dir == "sent" && tag.T == "hello" {
						client = tag.Client
					}
					if entry.Dir != "recv" || tag.T != "table-state" {
						continue
					}
					var frame wire.TableState
					if err = json.Unmarshal(entry.Msg, &frame); err != nil {
						t.Fatal(err)
					}
					seat := leakcheck.Spectator
					if frame.You.Seat != nil {
						seat = *frame.You.Seat
					}
					admin := false
					for _, human := range frame.Humans {
						if human.UserID == frame.You.UserID {
							admin = human.Admin
						}
					}
					states = append(states, frame)
					raws = append(raws, entry.Msg)
					viewers = append(viewers, frameViewer{Seat: seat, Client: client, Admin: admin})
				}
				if len(states) == 0 {
					t.Fatalf("log %d holds no table frame", i)
				}
				label := "browser end of game"
				if leaks := viewerLeaks(t, checker, label, viewers, states, raws, nil); len(leaks) != 0 {
					t.Fatalf("log %d: %d leaks, first: %v", i, len(leaks), leaks[:min(5, len(leaks))])
				}
				last := states[len(states)-1]
				if last.Status != "finished" || last.View.Result == nil {
					t.Fatalf("log %d ends before the scores: status %q", i, last.Status)
				}
				if last.View.CanDownload == hidden || (last.View.Reveal != nil) == hidden {
					t.Fatalf("log %d: canDownload %v and reveal %v under hideAtEnd %v", i, last.View.CanDownload, last.View.Reveal != nil, hidden)
				}
				if viewers[len(viewers)-1].Seat == leakcheck.Spectator {
					watched += len(states)
				} else {
					seated += len(states)
				}

				// The canaries replace the last frame of the same stream, so
				// each differs from the clean run by its plant alone.
				plant := func(name string, change func(fr *wire.TableState)) {
					planted := last
					v := last.View
					result := *v.Result
					result.Scores = slices.Clone(result.Scores)
					v.Result = &result
					planted.View = v
					change(&planted)
					raw, err := json.Marshal(planted)
					if err != nil {
						t.Fatal(err)
					}
					ps := append(slices.Clone(states[:len(states)-1]), planted)
					rs := append(slices.Clone(raws[:len(raws)-1]), raw)
					if leaks := viewerLeaks(t, checker, name, viewers, ps, rs, nil); len(leaks) == 0 {
						t.Errorf("log %d: the oracle accepted %s", i, name)
					}
				}
				plant("a changed total", func(fr *wire.TableState) { fr.View.Result.Scores[0].Total++ })
				plant("a download flag the policy does not grant", func(fr *wire.TableState) { fr.View.CanDownload = !fr.View.CanDownload })
				if hidden {
					box := truth.Moments[len(truth.Moments)-1].Box
					if len(box) == 0 {
						t.Fatal("no guild left in the box to plant")
					}
					plant("a revealed box under the hidden end", func(fr *wire.TableState) {
						var reveal wire.TableReveal
						if err := json.Unmarshal([]byte(`{"tucked":[],"leftovers":[],"pile":[],"box":[]}`), &reveal); err != nil {
							t.Fatal(err)
						}
						reveal.Box = slices.Clone(box)
						fr.View.Reveal = &reveal
					})
				}
			}
			if seated == 0 || watched == 0 {
				t.Fatalf("frames: %d seated, %d spectator; the spec records both", seated, watched)
			}
			t.Logf("checked %d seated and %d spectator frames, hideAtEnd %v", seated, watched, hidden)
		})
	}
}
