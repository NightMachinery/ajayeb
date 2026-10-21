package protocoltest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// A complete hot-seat game through the real ws/room/store stack. Other
// modes, viewers and uncommon phases remain covered by the existing
// protocol/leak suites and individual engine/wire goldens. No fields are
// scrubbed, re-encoded or removed from the received frames.
func TestPlatformV1TranscriptBaseline(t *testing.T) {
	data := t.TempDir()
	// Public synthetic key/token used only by this disposable test server.
	if err := os.WriteFile(filepath.Join(data, "secret.key"), bytes.Repeat([]byte{0x42}, 32), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := ident.LoadSecret(data)
	if err != nil {
		t.Fatal(err)
	}
	const token = "AAAAAAAAAAAAAAAAAAAAAA"
	user, err := secret.UserID(token)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	f := store.New("BASENX", store.ModeHotseat, engine.Options{}, 11, time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC))
	f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: user, Name: "Fixture"}, {Kind: store.SeatKindHuman, UserID: user, Name: "Fixture"}}
	f.Admin, f.Status = user, store.StatusPlaying
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	srv := newServer(t, data, 100)
	c := dial(t, srv, "Fixture")
	c.token, c.clientID = token, "fixture-tab"
	c.hello()
	c.join(f.Code)
	// Pin actual server rejection bytes, not just constructor examples.
	badSeq := c.nextSeq()
	c.actAt(badSeq, c.latest.Version, "construct:fixture-missing", seatInt(*c.latest.View.ToMove))
	if got := c.untilReject(badSeq); got.Code != wire.RejectCodeIllegal {
		t.Fatalf("illegal reject: %+v", got)
	}
	moves := 0
	for !over(c) {
		if moves > 200 || len(c.latest.View.Choices) == 0 {
			t.Fatal("fixture game stopped making progress")
		}
		seq := c.act(c.latest.View.Choices[0].Key, seatInt(*c.latest.View.ToMove))
		c.untilAck(seq)
		moves++
		if moves == 1 {
			staleSeq := c.nextSeq()
			c.actAt(staleSeq, 0, "construct:fixture-missing", nil)
			if got := c.untilReject(staleSeq); got.Code != wire.RejectCodeStale {
				t.Fatalf("stale reject: %+v", got)
			}
			c.resync()
		}
	}
	c.resync()
	checkReplay(t, srv, f.Code)
	frames, _ := c.raw.since(0)
	var transcript []byte
	for _, frame := range frames {
		transcript = append(transcript, frame...)
		transcript = append(transcript, '\n')
	}
	path := filepath.Join("testdata", "platform-v1-game.jsonl")
	if os.Getenv("UPDATE_PLATFORM_BASELINES") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, transcript, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(transcript, want) {
		t.Fatal("live v1 transcript bytes changed; regenerate deliberately (docs/dev/protocol-fixtures.md)")
	}
	snap := srv.snapshot(f.Code)
	t.Logf("v1 baseline: %d frames, %d bytes, %d accepted moves, final hash %s", len(frames), len(transcript), moves, engine.HashString(&snap.State))
}
