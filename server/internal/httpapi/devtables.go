package httpapi

import (
	"errors"
	"mime"
	"net/http"
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
)

// Tables is the Haftgane table kind's HTTP surface (a *tableroom.Tables):
// the development creation path, POST /api/dev/tables, and the
// ajayeb-log/2 download on the same path as a Duel room's log.
type Tables interface {
	CreateTable(spec tableroom.Spec) (string, error)
	ReadTableLog(code string) (*tableroom.Log, error)
}

// devTableRequest is the body of POST /api/dev/tables: a filled table
// from a seat list, the direct path beside the lobby's Start. Every seat
// is filled: a human by their user id (the welcome frame's) and name, or a
// bot. Creator, if set, is the table's one admin; without one there is no
// real admin, so the temporary-admin rule applies at once.
type devTableRequest struct {
	// Seats are the lobby slots, 3 to 7: {"userId":"...","name":"..."} for
	// a human, {"bot":"easy"} (or normal, hard) for a bot.
	Seats   []devTableSeat `json:"seats"`
	Creator string         `json:"creator"`
	// Seed is the game's seed, a decimal string or a JSON integer; absent
	// means a random one.
	Seed          *devSeed `json:"seed"`
	RandomSeating bool     `json:"randomSeating"`
	HideAtEnd     bool     `json:"hideAtEnd"`
	AllowUndo     bool     `json:"allowUndo"`
	// Prefix are complete batches resolved before the table opens, each
	// checked by a replay.
	Prefix []tableroom.BatchRecord `json:"prefix"`
}

type devTableSeat struct {
	Bot    string `json:"bot"`
	UserID string `json:"userId"`
	Name   string `json:"name"`
}

type devTableResponse struct {
	OK   bool   `json:"ok"`
	Code string `json:"code"`
	Seed string `json:"seed"` // decimal, so JavaScript keeps all 64 bits
}

// createTable serves POST /api/dev/tables.
func (d *devAPI) createTable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method-not-allowed", "use POST")
		return
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported-media-type", "the body must be application/json")
		return
	}
	var req devTableRequest
	if err := decodeStrict(http.MaxBytesReader(w, r.Body, maxDevBody), &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad-request", err.Error())
		return
	}
	seed := d.seed()
	if req.Seed != nil {
		seed = uint64(*req.Seed)
	}
	spec := tableroom.Spec{Seed: &seed, RandomSeating: req.RandomSeating, HideAtEnd: req.HideAtEnd,
		AllowUndo: req.AllowUndo, Prefix: req.Prefix, Creator: req.Creator}
	for _, s := range req.Seats {
		spec.Participants = append(spec.Participants, tableroom.Participant{Bot: s.Bot, UserID: s.UserID, Name: s.Name})
	}
	code, err := d.tables.CreateTable(spec)
	switch {
	case err == nil:
	case errors.Is(err, tableroom.ErrCreationOff):
		writeError(w, http.StatusForbidden, "creation-off", "table creation is off; start the server with -haftgane-tables")
		return
	case errors.Is(err, tableroom.ErrSpec):
		writeError(w, http.StatusBadRequest, "invalid-table", err.Error())
		return
	case errors.Is(err, room.ErrTooManyRooms):
		writeError(w, http.StatusServiceUnavailable, "busy", "the server holds as many rooms as it can")
		return
	case errors.Is(err, room.ErrClosed):
		writeError(w, http.StatusServiceUnavailable, "shutting-down", "the server is shutting down")
		return
	default:
		writeError(w, http.StatusInternalServerError, "internal", "the table could not be created")
		return
	}
	writeJSON(w, http.StatusCreated, devTableResponse{OK: true, Code: code, Seed: strconv.FormatUint(seed, 10)})
}

// serveTableLog answers the log download for a code that names a table.
func (a *logAPI) serveTableLog(w http.ResponseWriter, code string, l *tableroom.Log, err error) {
	switch {
	case err == nil:
	case errors.Is(err, tableroom.ErrLogRunning):
		writeError(w, http.StatusConflict, "game-running", "the game has not ended")
		return
	case errors.Is(err, tableroom.ErrLogHidden):
		writeError(w, http.StatusForbidden, "log-hidden", "this game keeps its hidden cards hidden after the end")
		return
	case errors.Is(err, tableroom.ErrLogArchived):
		writeError(w, http.StatusGone, "archived", "the room was archived and its log is not trusted")
		return
	default:
		writeError(w, http.StatusInternalServerError, "internal", "the room's log does not replay")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="ajayeb-`+code+`-`+a.now().UTC().Format("2006-01-02")+`.json"`)
	writeJSON(w, http.StatusOK, l)
}
