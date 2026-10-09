package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// fakeRooms records the specs it is asked for and answers with err, or a
// fixed code.
type fakeRooms struct {
	specs []room.Spec
	err   error
}

func (f *fakeRooms) CreateFromSpec(spec room.Spec) (string, error) {
	f.specs = append(f.specs, spec)
	if f.err != nil {
		return "", f.err
	}
	return "K7QX2M", nil
}

func newDevHandler(t *testing.T, rooms DevRooms) http.Handler {
	t.Helper()
	h, err := New(Config{Dev: true, DevEndpoints: true, DevRooms: rooms, Seed: func() uint64 { return 99 }}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// post sends a JSON body to POST /api/dev/rooms.
func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/dev/rooms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestDevEndpointsAreOffByDefault(t *testing.T) {
	h, err := New(Config{Dev: true}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	if rec := post(t, h, `{"mode":"hotseat"}`); rec.Code != http.StatusNotFound {
		t.Errorf("POST /api/dev/rooms without DevEndpoints = %d, want 404", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/dev/scenarios", nil); rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/dev/scenarios without DevEndpoints = %d, want 404", rec.Code)
	}
}

func TestDevRoomSpecs(t *testing.T) {
	open := store.Seat{Kind: store.SeatKindOpen}
	bot := func(level store.BotLevel) store.Seat { return store.Seat{Kind: store.SeatKindBot, Bot: level} }
	for _, tc := range []struct {
		name string
		body string
		want room.Spec
	}{
		{
			"hotseat scenario with a string seed",
			`{"mode":"hotseat","seed":"1","scenario":"mausoleum"}`,
			room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Seed: 1, Scenario: "mausoleum", Actions: []string{}},
		},
		{
			"online actions with a seed above 2^53",
			`{"mode":"online","seed":"18446744073709551615","actions":["draft:pyramids","draft:sphinx"]}`,
			room.Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{open, open}, Seed: 18446744073709551615, Actions: []string{"draft:pyramids", "draft:sphinx"}},
		},
		{
			"numeric seed",
			`{"mode":"hotseat","seed":12345}`,
			room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Seed: 12345, Actions: []string{}},
		},
		{
			"no seed takes the server's",
			`{"mode":"hotseat"}`,
			room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Seed: 99, Actions: []string{}},
		},
		{
			"ai room, human on seat 0 by default",
			`{"mode":"ai","seed":"5","bot":"hard","scenario":"great-library","actions":["wonder:great-library:baths"]}`,
			room.Spec{Mode: store.ModeAI, Seats: [2]store.Seat{open, bot(store.BotHard)}, Seed: 5, Scenario: "great-library", Actions: []string{"wonder:great-library:baths"}},
		},
		{
			"ai room, human on seat 1",
			`{"mode":"ai","seed":"5","bot":"easy","seat":1}`,
			room.Spec{Mode: store.ModeAI, Seats: [2]store.Seat{bot(store.BotEasy), open}, Seed: 5, Actions: []string{}},
		},
		{
			"options",
			`{"mode":"hotseat","seed":"7","options":{"firstPlayer":"seat1","centerPawn":"lastActiveStarts","hideAtEnd":true}}`,
			room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Seed: 7, Actions: []string{},
				Options: engine.Options{FirstPlayer: engine.FirstPlayerSeat1, CenterPawn: engine.LastActiveStarts, HideAtEnd: true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rooms := &fakeRooms{}
			rec := post(t, newDevHandler(t, rooms), tc.body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status %d %s, want 201", rec.Code, rec.Body.String())
			}
			if len(rooms.specs) != 1 || !reflect.DeepEqual(rooms.specs[0], tc.want) {
				t.Fatalf("spec = %+v\nwant %+v", rooms.specs, tc.want)
			}
			got := decodeBody[devRoomResponse](t, rec)
			want := devRoomResponse{
				OK: true, Code: "K7QX2M", Mode: tc.want.Mode, Seed: fmt.Sprint(tc.want.Seed),
				Scenario: tc.want.Scenario, Path: "/play/K7QX2M",
			}
			if got != want {
				t.Errorf("response = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDevRoomRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"not json", `mode=hotseat`, 400, "bad-request"},
		{"unknown field", `{"mode":"hotseat","players":2}`, 400, "bad-request"},
		{"trailing data", `{"mode":"hotseat"} {}`, 400, "bad-request"},
		{"unknown mode", `{"mode":"solo"}`, 400, "bad-request"},
		{"no mode", `{}`, 400, "bad-request"},
		{"ai without bot", `{"mode":"ai"}`, 400, "bad-request"},
		{"ai with an unknown bot", `{"mode":"ai","bot":"godlike"}`, 400, "bad-request"},
		{"bot outside ai", `{"mode":"hotseat","bot":"easy"}`, 400, "bad-request"},
		{"seat outside ai", `{"mode":"online","seat":1}`, 400, "bad-request"},
		{"seat 2", `{"mode":"ai","bot":"easy","seat":2}`, 400, "bad-request"},
		{"negative seed", `{"mode":"hotseat","seed":"-1"}`, 400, "bad-request"},
		{"fractional seed", `{"mode":"hotseat","seed":1.5}`, 400, "bad-request"},
		{"seed too big", `{"mode":"hotseat","seed":"18446744073709551616"}`, 400, "bad-request"},
		{"unknown option", `{"mode":"hotseat","options":{"firstPlayer":"seat9"}}`, 400, "bad-request"},
		{"unknown scenario", `{"mode":"hotseat","scenario":"no-such-thing"}`, 400, "unknown-scenario"},
		{"too large", `{"mode":"hotseat","actions":["` + strings.Repeat("a", maxDevBody) + `"]}`, 400, "bad-request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rooms := &fakeRooms{}
			rec := post(t, newDevHandler(t, rooms), tc.body)
			if rec.Code != tc.status {
				t.Fatalf("status %d %s, want %d", rec.Code, rec.Body.String(), tc.status)
			}
			if body := decodeBody[errorBody](t, rec); body.OK || body.Code != tc.code {
				t.Errorf("body = %+v, want code %q", body, tc.code)
			}
			if len(rooms.specs) != 0 {
				t.Errorf("a refused request reached the rooms: %+v", rooms.specs)
			}
		})
	}
}

func TestDevRoomUnknownScenarioListsTheNames(t *testing.T) {
	rec := post(t, newDevHandler(t, &fakeRooms{}), `{"mode":"hotseat","scenario":"nope"}`)
	body := decodeBody[errorBody](t, rec)
	if !strings.Contains(body.Msg, "great-library") {
		t.Errorf("msg = %q, want the list of named scenarios", body.Msg)
	}
}

func TestDevRoomMethodAndContentType(t *testing.T) {
	h := newDevHandler(t, &fakeRooms{})
	rec := do(t, h, http.MethodGet, "/api/dev/rooms", nil)
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodPost {
		t.Errorf("GET /api/dev/rooms = %d (Allow %q), want 405 Allow POST", rec.Code, rec.Header().Get("Allow"))
	}
	for _, ct := range []string{"", "text/plain", "application/x-www-form-urlencoded"} {
		req := httptest.NewRequest(http.MethodPost, "/api/dev/rooms", strings.NewReader(`{"mode":"hotseat"}`))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q: status %d, want 415", ct, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/dev/rooms", strings.NewReader(`{"mode":"hotseat"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("Content-Type with a charset: status %d, want 201", rec.Code)
	}
}

func TestDevRoomErrorsFromTheRooms(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("%w: action 3 is not legal", room.ErrInvalidSpec), 400, "invalid-room"},
		{room.ErrTooManyRooms, 503, "busy"},
		{room.ErrClosed, 503, "shutting-down"},
		{errors.New("disk on fire"), 500, "internal"},
	} {
		rec := post(t, newDevHandler(t, &fakeRooms{err: tc.err}), `{"mode":"hotseat"}`)
		if rec.Code != tc.status {
			t.Errorf("%v: status %d, want %d", tc.err, rec.Code, tc.status)
		}
		if body := decodeBody[errorBody](t, rec); body.Code != tc.code {
			t.Errorf("%v: code %q, want %q", tc.err, body.Code, tc.code)
		}
	}
}

func TestDevScenarios(t *testing.T) {
	h := newDevHandler(t, &fakeRooms{})
	rec := do(t, h, http.MethodGet, "/api/dev/scenarios", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := decodeBody[devScenariosResponse](t, rec)
	if !body.OK || !slices.Equal(body.Scenarios, engine.NamedScenarioNames()) {
		t.Errorf("scenarios = %+v, want engine.NamedScenarioNames()", body)
	}
	if rec := do(t, h, http.MethodPost, "/api/dev/scenarios", nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/dev/scenarios = %d, want 405", rec.Code)
	}
}

// TestDevRoomsWithTheRealManager creates rooms through a real Manager: a
// named scenario plus its trigger, a seed beyond a double's precision, and
// an illegal action log.
func TestDevRoomsWithTheRealManager(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := room.NewManager(room.Config{Store: st})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	h := newDevHandler(t, m)

	trigger := engine.NamedScenario("zeus-destroy").Trigger
	triggerJSON, err := json.Marshal(trigger)
	if err != nil {
		t.Fatal(err)
	}
	rec := post(t, h, `{"mode":"hotseat","seed":"12345678901234567891","scenario":"zeus-destroy","actions":`+string(triggerJSON)+`}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	created := decodeBody[devRoomResponse](t, rec)
	if created.Seed != "12345678901234567891" {
		t.Errorf("seed = %q, want it back exactly", created.Seed)
	}
	snap, err := m.Snapshot(context.Background(), created.Code)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Mode != store.ModeHotseat || !slices.Equal(snap.Actions, trigger) || snap.Seats[0].Kind != store.SeatKindOpen {
		t.Errorf("room = mode %s actions %q seats %+v", snap.Mode, snap.Actions, snap.Seats)
	}
	f, err := st.PeekHeader(created.Code)
	if err != nil {
		t.Fatal(err)
	}
	if f.Seed != 12345678901234567891 || f.Scenario != "zeus-destroy" {
		t.Errorf("stored seed %d scenario %q", f.Seed, f.Scenario)
	}

	rec = post(t, h, `{"mode":"ai","bot":"easy","actions":["construct:baths"]}`)
	if rec.Code != http.StatusBadRequest || decodeBody[errorBody](t, rec).Code != "invalid-room" {
		t.Errorf("an illegal action log: %d %s, want 400 invalid-room", rec.Code, rec.Body.String())
	}
}
