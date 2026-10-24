package protocoltest

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// scrubbed, re-encoded or removed before hashing the received frames.
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
			// The reject is followed by the current state. Read it before
			// asking to resync: the outbox replaces a pending state frame
			// with a newer one, so a slow writer would otherwise send only
			// the resync's state and the transcript would differ.
			c.until("the state after the stale reject", func(f any) bool {
				_, ok := f.(wire.State)
				return ok
			})
			c.resync()
		}
	}
	c.resync()
	checkReplay(t, srv, f.Code)
	frames, _ := c.raw.since(0)
	transcript, actual := platformTranscriptDigests(frames)
	// Explicit capture-only mode supports byte diffs against master before
	// this test or its manifest has been merged there.
	if output := os.Getenv("PLATFORM_V1_STREAM_PATH"); output != "" {
		if err := os.WriteFile(output, transcript, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("raw v1 stream: %s (%d bytes, sha256=%s)", output, actual.Stream.Length, actual.Stream.SHA256)
		return
	}
	path := filepath.Join("testdata", "platform-v1-game.jsonl")
	if os.Getenv("UPDATE_PLATFORM_BASELINES") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, platformManifestBytes(actual), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifestBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := platformReadManifest(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if difference := platformTranscriptDifference(want, actual); difference != "" {
		file, err := os.CreateTemp("", "ajayeb-platform-v1-actual-*.jsonl")
		if err != nil {
			t.Fatalf("%s; could not create actual-stream artifact: %v", difference, err)
		}
		if _, err := file.Write(transcript); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		// Do not use t.TempDir: it deletes the diagnostic before a reader
		// can run the byte diff. This synthetic stream remains in OS temp.
		expectedPath := file.Name() + ".master.jsonl"
		t.Fatalf("%s\nactual stream: %s\ncapture expected stream from master (run from repository root):\n%s\nbyte diff: command cmp %s %s",
			difference, file.Name(), platformMasterStreamCommand(expectedPath), platformShellQuote(expectedPath), platformShellQuote(file.Name()))
	}

	snap := srv.snapshot(f.Code)
	t.Logf("v1 baseline: %d frames, %d bytes, %d accepted moves, final hash %s", len(frames), len(transcript), moves, engine.HashString(&snap.State))
}

// Frame lengths exclude the separator; the stream includes one newline
// after each untouched frame. Hashes pin whitespace, field order and values.
type platformFrameDigest struct {
	Index  int    `json:"index"`
	Kind   string `json:"kind"`
	Length int    `json:"length"`
	SHA256 string `json:"sha256"`
}

type platformStreamDigest struct {
	Frames int    `json:"frames"`
	Length int    `json:"length"`
	SHA256 string `json:"sha256"`
}

type platformManifest struct {
	Stream platformStreamDigest
	Frames []platformFrameDigest
}

func platformTranscriptDigests(frames [][]byte) ([]byte, platformManifest) {
	var stream []byte
	manifest := platformManifest{Frames: make([]platformFrameDigest, 0, len(frames))}
	for i, frame := range frames {
		var header struct {
			T string `json:"t"`
		}
		if err := json.Unmarshal(frame, &header); err != nil {
			header.T = "<invalid-json>"
		}
		manifest.Frames = append(manifest.Frames, platformFrameDigest{i, header.T, len(frame), fmt.Sprintf("%x", sha256.Sum256(frame))})
		stream = append(stream, frame...)
		stream = append(stream, '\n')
	}
	manifest.Stream = platformStreamDigest{len(frames), len(stream), fmt.Sprintf("%x", sha256.Sum256(stream))}
	return stream, manifest
}

func platformManifestBytes(manifest platformManifest) []byte {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	// These types contain only strings and integers, so encoding cannot fail.
	_ = encoder.Encode(manifest.Stream)
	for _, frame := range manifest.Frames {
		_ = encoder.Encode(frame)
	}
	return data.Bytes()
}

func platformReadManifest(data []byte) (platformManifest, error) {
	var manifest platformManifest
	lines := bufio.NewScanner(bytes.NewReader(data))
	if !lines.Scan() {
		return manifest, fmt.Errorf("missing stream digest")
	}
	if err := json.Unmarshal(lines.Bytes(), &manifest.Stream); err != nil {
		return manifest, err
	}
	for lines.Scan() {
		var frame platformFrameDigest
		if err := json.Unmarshal(lines.Bytes(), &frame); err != nil {
			return manifest, err
		}
		manifest.Frames = append(manifest.Frames, frame)
	}
	if err := lines.Err(); err != nil {
		return manifest, err
	}
	if manifest.Stream.Frames != len(manifest.Frames) {
		return manifest, fmt.Errorf("manifest declares %d frames but contains %d", manifest.Stream.Frames, len(manifest.Frames))
	}
	return manifest, nil
}

func platformFrameDescription(frames []platformFrameDigest, index int) string {
	if index >= len(frames) {
		return "<missing> length=0 sha256=<missing>"
	}
	f := frames[index]
	return fmt.Sprintf("index=%d kind=%s length=%d sha256=%s", f.Index, f.Kind, f.Length, f.SHA256)
}

func platformTranscriptDifference(want, actual platformManifest) string {
	for i := 0; i < max(len(want.Frames), len(actual.Frames)); i++ {
		if i >= len(want.Frames) || i >= len(actual.Frames) || want.Frames[i] != actual.Frames[i] {
			return fmt.Sprintf("live v1 transcript bytes changed: first differing frame %d\nexpected %s\nactual   %s", i,
				platformFrameDescription(want.Frames, i), platformFrameDescription(actual.Frames, i))
		}
	}
	if want.Stream != actual.Stream {
		return fmt.Sprintf("live v1 stream digest changed (frame records match): expected length=%d sha256=%s; actual length=%d sha256=%s",
			want.Stream.Length, want.Stream.SHA256, actual.Stream.Length, actual.Stream.SHA256)
	}
	return ""
}

func platformShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func platformMasterStreamCommand(output string) string {
	// Archive only server/, use master's real stack and existing test helpers,
	// and inject this capture test. TemporaryDirectory cleans the source copy;
	// the requested raw stream is outside it. No checkout or ref is changed.
	script := `import os,shutil,subprocess,sys,tempfile; reference=tempfile.TemporaryDirectory(prefix="ajayeb-platform-master-"); archive=tempfile.TemporaryFile(); subprocess.run(["git","archive","master","server"],stdout=archive,check=True); archive.seek(0); subprocess.run(["tar","-x","-C",reference.name],stdin=archive,check=True); shutil.copyfile("server/internal/protocoltest/platform_baseline_test.go",reference.name+"/server/internal/protocoltest/platform_baseline_test.go"); subprocess.run(["go","-C",reference.name+"/server","test","./internal/protocoltest","-run","^TestPlatformV1TranscriptBaseline$","-count=1"],env=dict(os.environ,PLATFORM_V1_STREAM_PATH=os.path.abspath(sys.argv[1])),check=True)`
	return "python3 tools/heavy.py run --kind test -- python3 -c " + platformShellQuote(script) + " " + platformShellQuote(output)
}

func TestPlatformV1DigestDiagnostics(t *testing.T) {
	original := []byte(`{"t":"welcome","v":1}`)
	_, want := platformTranscriptDigests([][]byte{original})
	changed := bytes.Replace(original, []byte(`"v":1`), []byte(`"v":2`), 1)
	_, actual := platformTranscriptDigests([][]byte{changed})
	difference := platformTranscriptDifference(want, actual)
	for _, fragment := range []string{"first differing frame 0", "expected index=0 kind=welcome length=21 sha256=" + want.Frames[0].SHA256, "actual   index=0 kind=welcome length=21 sha256=" + actual.Frames[0].SHA256} {
		if !strings.Contains(difference, fragment) {
			t.Fatalf("missing diagnostic %q: %s", fragment, difference)
		}
	}
	_, empty := platformTranscriptDigests(nil)
	if !strings.Contains(platformTranscriptDifference(want, empty), "<missing> length=0 sha256=<missing>") {
		t.Fatal("missing-frame diagnostic")
	}
	if !strings.Contains(platformTranscriptDifference(empty, actual), "first differing frame 0") {
		t.Fatal("extra-frame diagnostic")
	}
	if platformTranscriptDifference(want, want) != "" {
		t.Fatal("unchanged transcript differs")
	}
	actual = want
	actual.Stream.Length++
	if !strings.Contains(platformTranscriptDifference(want, actual), "frame records match") {
		t.Fatal("stream-only diagnostic")
	}
}
