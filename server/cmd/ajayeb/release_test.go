package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/lobbyroom"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestReleaseFlagNeedsNoDevelopmentMode(t *testing.T) {
	o, err := parseFlags([]string{"--haftgane-tables"}, io.Discard)
	if err != nil || !o.tables || o.dev || o.devEndpoints {
		t.Fatalf("release flags: %+v %v", o, err)
	}
}

// Real current records exercise the production registration, old schema-only
// readers, future record formats, read-only scans and the shared reaper.
func TestReleaseRecordsScanRollbackAndRetention(t *testing.T) {
	dir := t.TempDir()
	tables := tableroom.New(tableroom.Config{Creation: true})
	lobbies := lobbyroom.New(lobbyroom.Config{Creation: true, Tables: tables})
	kinds, err := roomKinds(tables, lobbies)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(dir, nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	m, err := room.NewManager(room.Config{Store: st, Kinds: kinds, KindOf: st.Game})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	tables.Bind(m, st)
	lobbies.Bind(m, st)
	c := room.NewClient("release-human", "release-tab", "Player")
	c.ProtocolVersion = 2
	lobby, err := m.CreatePlatform(context.Background(), c, wire.LobbyCreate{T: "lobby-create", SeatCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	seed := uint64(42)
	table, err := tables.CreateTable(tableroom.Spec{Seed: &seed, Creator: c.UserID, Participants: []tableroom.Participant{{UserID: c.UserID, Name: "Player"}, {Bot: tableroom.BotNormal}, {Bot: tableroom.BotNormal}}})
	if err != nil {
		t.Fatal(err)
	}
	m.Close()
	before := map[string][]byte{}
	for _, code := range []string{lobby, table} {
		p := filepath.Join(st.Dir(), code+".json")
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var head struct{ Schema, Format int }
		if err = json.Unmarshal(raw, &head); err != nil || head.Schema != 2 || head.Format != 6 {
			t.Fatalf("not current record: %+v %v", head, err)
		}
		before[p] = raw
	}
	if report := store.CheckDataDirWithCodecs(dir, kinds.Codecs()); report.Errors != 0 || report.Playable != 2 {
		t.Fatalf("current scan: %+v", report)
	}
	if report := store.CheckDataDir(dir); report.Errors != 2 || report.Playable != 0 {
		t.Fatalf("old schema-only scan must refuse: %+v", report)
	}
	for p, raw := range before {
		after, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(raw, after) {
			t.Fatal("scan changed bytes")
		}
		var obj map[string]any
		if err = json.Unmarshal(raw, &obj); err != nil {
			t.Fatal(err)
		}
		obj["format"] = 7
		future, _ := json.Marshal(obj)
		if err = os.WriteFile(p, future, 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if rc := checkRooms(dir, &out, io.Discard); rc != 1 {
			t.Fatalf("future format scan exit=%d %s", rc, &out)
		}
		if err = os.WriteFile(p, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Neither lobby nor table loading extends retention. Both expire under
	// the same 14-day idle rule as an unfinished Duel room.
	reaper := store.NewReaper(st, func() time.Time { return time.Now().Add(15 * 24 * time.Hour) })
	removed, err := reaper.Sweep()
	if err != nil || len(removed) != 2 {
		t.Fatalf("retention removed=%v err=%v", removed, err)
	}
}
