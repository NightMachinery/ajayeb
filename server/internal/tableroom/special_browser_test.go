package tableroom

// The raw browser recording is private test output, not a persisted fixture.
// Run this after the Home-flow spec, passing its two frames-*.json files in
// AJAYEB_SPECIAL_BROWSER_FRAMES (the operating system's path-list separator).
import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestSpecialBrowserFrames(t *testing.T) {
	paths := os.Getenv("AJAYEB_SPECIAL_BROWSER_FRAMES")
	if paths == "" {
		t.Skip("run the Home-flow special-decision spec first")
	}
	anyCanary := false
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
			checked := 0
			canary := false
			for _, log := range capture.Logs {
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
					seat := -1
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
					checked++
					if !canary && frame.View.Phase == "discard-build" && frame.View.You != nil && len(frame.View.You.Offer) == 0 {
						planted := frame
						copyPrivate := *frame.View.You
						planted.View.You = &copyPrivate
						planted.View.You.Offer = []wire.TableOffer{{Design: "baths", Count: 1, Legal: true}}
						raw, _ := json.Marshal(planted)
						if leaks := frameLeaks(t, checker, "foreign discard pile canary", frameViewer{Seat: seat, Client: client, Admin: admin}, []wire.TableState{planted}, [][]byte{raw}, nil); len(leaks) == 0 {
							t.Fatal("oracle accepted a discard pile in a waiting seat's frame")
						}
						canary = true
						anyCanary = true
					}
				}
				if leaks := viewerLeaks(t, checker, "browser special decisions", viewers, states, raws, nil); len(leaks) != 0 {
					t.Fatalf("%d leaks, first: %v", len(leaks), leaks[:min(5, len(leaks))])
				}
			}
			if checked == 0 {
				t.Fatalf("missing browser frames/canary: %d, %v", checked, canary)
			}
			t.Logf("checked %d raw browser frames; foreign-pile canary reached: %v", checked, canary)
		})
	}
	if !anyCanary {
		t.Fatal("no discard point reached for the foreign-pile canary")
	}
}
