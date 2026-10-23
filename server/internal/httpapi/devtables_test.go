package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
)

// fakeTables records the table specs it is asked for and answers each
// code's log from logs, or tableroom.ErrNotTable.
type fakeTables struct {
	specs []tableroom.Spec
	err   error
	logs  map[string]tableLogAnswer
}

type tableLogAnswer struct {
	log *tableroom.Log
	err error
}

func (f *fakeTables) CreateTable(spec tableroom.Spec) (string, error) {
	f.specs = append(f.specs, spec)
	if f.err != nil {
		return "", f.err
	}
	return "T7QX2M", nil
}

func (f *fakeTables) ReadTableLog(code string) (*tableroom.Log, error) {
	if a, ok := f.logs[code]; ok {
		return a.log, a.err
	}
	return nil, tableroom.ErrNotTable
}

func postTable(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := devRequest(http.MethodPost, "/api/dev/tables", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDevTables(t *testing.T) {
	// Without Tables, or without the dev endpoints, the path is unknown.
	for _, cfg := range []Config{
		{Dev: true, DevEndpoints: true, DevRooms: &fakeRooms{}},
		{Dev: true, Tables: &fakeTables{}},
	} {
		h, err := New(cfg, testFS())
		if err != nil {
			t.Fatal(err)
		}
		if rec := postTable(t, h, `{"seats":[{},{},{}]}`); rec.Code != http.StatusNotFound {
			t.Errorf("POST /api/dev/tables = %d, want 404", rec.Code)
		}
	}

	tables := &fakeTables{}
	h, err := New(Config{Dev: true, DevEndpoints: true, DevRooms: &fakeRooms{}, Tables: tables, Seed: func() uint64 { return 99 }}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	rec := postTable(t, h, `{"seats":[{"userId":"u0","name":"Ann"},{"bot":"easy"},{}],"creator":"u0","seed":"18446744073709551615","hideAtEnd":true,"allowUndo":true,
		"prefix":[{"point":"age-1-turn-1","picks":[{"seat":0,"key":"tuck:amber"}]}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	got := decodeBody[devTableResponse](t, rec)
	if !got.OK || got.Code != "T7QX2M" || got.Seed != "18446744073709551615" {
		t.Fatalf("response %+v", got)
	}
	sp := tables.specs[0]
	if len(sp.Participants) != 3 || sp.Participants[1].Bot != "easy" || sp.Participants[0] != (tableroom.Participant{UserID: "u0", Name: "Ann"}) ||
		sp.Participants[2] != (tableroom.Participant{}) ||
		*sp.Seed != 1<<64-1 || !sp.HideAtEnd || !sp.AllowUndo || sp.RandomSeating || sp.Creator != "u0" ||
		len(sp.Prefix) != 1 || sp.Prefix[0].Picks[0].Key != "tuck:amber" {
		t.Fatalf("spec %+v", sp)
	}
	if rec := postTable(t, h, `{"seats":[{},{},{}]}`); rec.Code != http.StatusCreated || *tables.specs[1].Seed != 99 {
		t.Fatalf("the default seed: %d %+v", rec.Code, tables.specs[1])
	}

	for body, code := range map[string]string{
		`{"seats":[{}],"extra":1}`:      "bad-request",
		`{"seats":[{"user":"x"}]}`:      "bad-request",
		`{"seats":[{}],"seed":"-1"}`:    "bad-request",
		`not json`:                      "bad-request",
		`{"seats":[{},{},{},{}],"x":0}`: "bad-request",
	} {
		if rec := postTable(t, h, body); rec.Code != http.StatusBadRequest || errCode(t, rec) != code {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	for err, want := range map[error]int{
		tableroom.ErrCreationOff: http.StatusForbidden,
		tableroom.ErrSpec:        http.StatusBadRequest,
		room.ErrTooManyRooms:     http.StatusServiceUnavailable,
		room.ErrClosed:           http.StatusServiceUnavailable,
		errors.New("disk"):       http.StatusInternalServerError,
	} {
		tables.err = err
		if rec := postTable(t, h, `{"seats":[{},{},{}]}`); rec.Code != want {
			t.Errorf("%v: %d, want %d", err, rec.Code, want)
		}
	}
	// Only a loopback client that no proxy forwarded.
	req := devRequest(http.MethodPost, "/api/dev/tables", `{"seats":[{},{},{}]}`)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.7:40000"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a remote client: %d", rec.Code)
	}
}

// TestTableLogs: a table's code answers its ajayeb-log/2 under the end
// policy, with Duel's statuses and errors; any other code reaches Duel's
// path unchanged.
func TestTableLogs(t *testing.T) {
	fx := newLogFixture(t)
	finished := &tableroom.Log{Format: tableroom.LogFormat, Game: tableroom.Game, Rules: 7,
		Batches: []tableroom.BatchRecord{}, Hash: "h"}
	tables := &fakeTables{logs: map[string]tableLogAnswer{
		"TFNSHD": {log: finished},
		"TRUNNG": {err: tableroom.ErrLogRunning},
		"THIDDN": {err: tableroom.ErrLogHidden},
		"TARCHV": {err: tableroom.ErrLogArchived},
		"TBROKN": {err: tableroom.ErrLogReplay},
	}}
	h, err := New(Config{Version: "test", Logs: fx.st, Tables: tables, Clock: func() time.Time { return fx.now }}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	rec := do(t, h, http.MethodGet, "/api/rooms/tfnshd/log", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Disposition") != `attachment; filename="ajayeb-TFNSHD-2026-10-01.json"` {
		t.Fatalf("finished: %d %s", rec.Code, rec.Body.String())
	}
	var l tableroom.Log
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil || l.Format != "ajayeb-log/2" || l.Hash != "h" {
		t.Fatalf("the log: %+v %v", l, err)
	}
	for code, want := range map[string]struct {
		status int
		code   string
	}{
		"TRUNNG": {http.StatusConflict, "game-running"},
		"THIDDN": {http.StatusForbidden, "log-hidden"},
		"TARCHV": {http.StatusGone, "archived"},
		"TBROKN": {http.StatusInternalServerError, "internal"},
		"NOROOM": {http.StatusNotFound, "not-found"},
	} {
		rec := do(t, h, http.MethodGet, "/api/rooms/"+code+"/log", nil)
		if rec.Code != want.status || errCode(t, rec) != want.code {
			t.Errorf("%s: %d %s", code, rec.Code, rec.Body.String())
		}
	}
	// A Duel room's log is Duel's.
	if rec := do(t, h, http.MethodGet, "/api/rooms/FNSHED/log", nil); rec.Code != http.StatusOK || !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("a Duel log: %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/rooms/PRAYNG/log", nil); rec.Code != http.StatusConflict {
		t.Fatalf("a running Duel room: %d", rec.Code)
	}
}
