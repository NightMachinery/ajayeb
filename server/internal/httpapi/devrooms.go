package httpapi

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// DevRooms creates a room directly from a Spec; *room.Manager satisfies it.
type DevRooms interface {
	CreateFromSpec(spec room.Spec) (string, error)
}

// maxDevBody bounds the body of POST /api/dev/rooms. A whole game is under
// a hundred action keys, so this is generous.
const maxDevBody = 64 << 10

// devAPI serves the development endpoints:
//
//   - POST /api/dev/rooms creates a room at a chosen position (a named
//     scenario, an action log, or both, the actions applied after the
//     scenario) and answers its code. Its seats are open: the first user to
//     join takes them, as the mode allows (both seats of a hot-seat room,
//     the human seat of a bot room, seat 0 and then seat 1 of an online
//     room).
//   - GET /api/dev/scenarios lists the named scenarios.
//
// The body must be JSON (Content-Type application/json), so a page on
// another site cannot post to it with a simple form or fetch.
type devAPI struct {
	rooms DevRooms
	seed  func() uint64
}

func newDevAPI(rooms DevRooms, seed func() uint64) *devAPI {
	if seed == nil {
		seed = cryptoSeed
	}
	return &devAPI{rooms: rooms, seed: seed}
}

// cryptoSeed returns 64 random bits from crypto/rand.
func cryptoSeed() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:]) // never fails: it crashes the program if the system source does
	return binary.LittleEndian.Uint64(b[:])
}

// devRoomRequest is the body of POST /api/dev/rooms.
type devRoomRequest struct {
	// Mode is online, hotseat or ai.
	Mode store.Mode `json:"mode"`
	// Seed is the game's seed, a decimal string (or a JSON integer); absent
	// means a random one.
	Seed *devSeed `json:"seed"`
	// Scenario names a named scenario to start from.
	Scenario string `json:"scenario"`
	// Actions are action keys applied after the scenario (or the ordinary
	// start), each checked against the legal actions.
	Actions []string `json:"actions"`
	// Bot is the bot's level (easy, normal or hard); required in an ai
	// room, refused elsewhere.
	Bot store.BotLevel `json:"bot"`
	// Seat is the human's seat in an ai room, 0 (the default) or 1.
	Seat *int `json:"seat"`
	// Options are the game's options; absent means the defaults.
	Options *engine.Options `json:"options"`
}

// devRoomResponse answers a created room.
type devRoomResponse struct {
	OK       bool       `json:"ok"`
	Code     string     `json:"code"`
	Mode     store.Mode `json:"mode"`
	Seed     string     `json:"seed"` // decimal, so JavaScript keeps all 64 bits
	Scenario string     `json:"scenario"`
	Path     string     `json:"path"` // the page that plays the room
}

type devScenariosResponse struct {
	OK        bool     `json:"ok"`
	Scenarios []string `json:"scenarios"`
}

// devSeed is a seed given as a decimal string or a JSON integer.
type devSeed uint64

// UnmarshalJSON accepts "123" and 123, and nothing else (no sign, fraction
// or exponent).
func (s *devSeed) UnmarshalJSON(b []byte) error {
	text := string(b)
	if strings.HasPrefix(text, `"`) {
		if err := json.Unmarshal(b, &text); err != nil {
			return err
		}
	}
	n, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return fmt.Errorf("seed %s is not a decimal integer from 0 to 2^64-1", b)
	}
	*s = devSeed(n)
	return nil
}

// createRoom serves POST /api/dev/rooms.
func (d *devAPI) createRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method-not-allowed", "use POST")
		return
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported-media-type", "the body must be application/json")
		return
	}
	var req devRoomRequest
	if err := decodeStrict(http.MaxBytesReader(w, r.Body, maxDevBody), &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad-request", err.Error())
		return
	}
	spec, err := d.spec(req)
	if err != nil {
		code := "bad-request"
		if errors.Is(err, errUnknownScenario) {
			code = "unknown-scenario"
		}
		writeError(w, http.StatusBadRequest, code, err.Error())
		return
	}

	roomCode, err := d.rooms.CreateFromSpec(spec)
	switch {
	case err == nil:
	case errors.Is(err, room.ErrInvalidSpec):
		writeError(w, http.StatusBadRequest, "invalid-room", err.Error())
		return
	case errors.Is(err, room.ErrTooManyRooms):
		writeError(w, http.StatusServiceUnavailable, "busy", "the server holds as many rooms as it can")
		return
	case errors.Is(err, room.ErrClosed):
		writeError(w, http.StatusServiceUnavailable, "shutting-down", "the server is shutting down")
		return
	default:
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, devRoomResponse{
		OK:       true,
		Code:     roomCode,
		Mode:     spec.Mode,
		Seed:     strconv.FormatUint(spec.Seed, 10),
		Scenario: spec.Scenario,
		Path:     "/play/" + roomCode,
	})
}

var errUnknownScenario = errors.New("unknown scenario")

// spec turns a request into a room Spec, checking everything the room does
// not check itself.
func (d *devAPI) spec(req devRoomRequest) (room.Spec, error) {
	open := store.Seat{Kind: store.SeatKindOpen}
	spec := room.Spec{
		Mode:     req.Mode,
		Seats:    [2]store.Seat{open, open},
		Scenario: req.Scenario,
		Actions:  req.Actions,
	}
	if spec.Actions == nil {
		spec.Actions = []string{}
	}
	if req.Seed != nil {
		spec.Seed = uint64(*req.Seed)
	} else {
		spec.Seed = d.seed()
	}
	if req.Options != nil {
		spec.Options = *req.Options
	}
	if req.Scenario != "" && engine.NamedScenario(req.Scenario) == nil {
		return room.Spec{}, fmt.Errorf("%w %q; the named scenarios are %s",
			errUnknownScenario, req.Scenario, strings.Join(engine.NamedScenarioNames(), ", "))
	}

	switch req.Mode {
	case store.ModeOnline, store.ModeHotseat:
		if req.Bot != "" || req.Seat != nil {
			return room.Spec{}, errors.New("bot and seat are for ai rooms only")
		}
	case store.ModeAI:
		switch req.Bot {
		case store.BotEasy, store.BotNormal, store.BotHard:
		default:
			return room.Spec{}, fmt.Errorf("an ai room needs bot easy, normal or hard, not %q", req.Bot)
		}
		human := 0
		if req.Seat != nil {
			human = *req.Seat
		}
		if human != 0 && human != 1 {
			return room.Spec{}, fmt.Errorf("seat %d is not 0 or 1", human)
		}
		spec.Seats[1-human] = store.Seat{Kind: store.SeatKindBot, Bot: req.Bot}
	default:
		return room.Spec{}, fmt.Errorf("unknown mode %q; use online, hotseat or ai", req.Mode)
	}
	return spec, nil
}

// scenarios serves GET /api/dev/scenarios.
func (d *devAPI) scenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(w, http.StatusMethodNotAllowed, "method-not-allowed", "use GET")
		return
	}
	writeJSON(w, http.StatusOK, devScenariosResponse{OK: true, Scenarios: engine.NamedScenarioNames()})
}

// decodeStrict decodes one JSON value into v, refusing unknown fields and
// trailing data.
func decodeStrict(r io.Reader, v any) error {
	body, err := io.ReadAll(r)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return fmt.Errorf("the body is larger than %d bytes", tooBig.Limit)
		}
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("malformed body: %v", err)
	}
	// dec.More is not enough here: it reports false for a stray closing
	// bracket or brace, so a second Decode must find the end of the body.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("malformed body: trailing data after the JSON value")
	}
	return nil
}
