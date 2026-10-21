package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ErrGame identifies an unsupported game/codec. It also matches ErrSchema so
// existing callers and the preflight counters treat it as unsupported storage.
var ErrGame = fmt.Errorf("%w: unknown room game or codec", ErrSchema)

const maxGameIDBytes = 32

// RetentionHeader is validated, transient metadata for the shared retention
// policy. It is not a saved-room format; each codec owns its complete file type.
type RetentionHeader struct {
	Code       string
	Status     Status
	Updated    time.Time
	ArchivedAt *time.Time
}

// Inspection keeps game decoding and read-only compatibility checks inside its
// codec. Check reports a hash-only mismatch after a compatible legal replay;
// an error means that this known game's record would archive. It must not write.
// The store calls Check only for live records, never archived ones.
type Inspection interface {
	Retention() RetentionHeader
	Check() (hashMismatch bool, err error)
}

// Codec registers one schema-2 game at composition time. Decode must validate
// its own file and return retention metadata, without replaying or writing.
// Schema 1 is reserved for the existing Duel codec and cannot be replaced.
type Codec struct {
	Game   string
	Decode func([]byte) (Inspection, error)
}

// Codecs is an immutable registry. Build it before opening a store or running
// preflight; no process-global registration or runtime mutation is supported.
type Codecs struct {
	schema2 map[string]func([]byte) (Inspection, error)
}

// NewCodecs includes schema-1 Duel and the supplied schema-2 codecs. Duplicate
// ids and malformed registrations fail composition, before any file is read.
func NewCodecs(codecs ...Codec) (*Codecs, error) {
	r := &Codecs{schema2: make(map[string]func([]byte) (Inspection, error))}
	for _, c := range codecs {
		if !validGameID(c.Game) || c.Game == "duel" || c.Decode == nil {
			return nil, fmt.Errorf("store: invalid codec registration %q", c.Game)
		}
		if _, ok := r.schema2[c.Game]; ok {
			return nil, fmt.Errorf("store: duplicate codec registration %q", c.Game)
		}
		r.schema2[c.Game] = c.Decode
	}
	return r, nil
}

// No schema-2 codec is enabled in production yet.
var productionCodecs = &Codecs{}

func validGameID(s string) bool {
	if len(s) == 0 || len(s) > maxGameIDBytes || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

// fileKind reads only bounded discriminators, independent of game payload
// types. The complete input is already capped by MaxFileBytes. No field order
// is required. Reject duplicate discriminators (including case aliases) rather
// than letting two readers select different codecs for the same bytes.
func fileKind(data []byte) (int, string, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil {
		return 0, "", err
	}
	if token != json.Delim('{') {
		return 0, "", fmt.Errorf("%w: expected room object", ErrSchema)
	}
	var schema int
	var game string
	var schemaSeen, gameSeen bool
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return 0, "", err
		}
		key, ok := token.(string)
		if !ok {
			return 0, "", fmt.Errorf("%w: expected field name", ErrSchema)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return 0, "", err
		}
		switch {
		case strings.EqualFold(key, "schema"):
			if schemaSeen || len(raw) > 10 {
				return 0, "", fmt.Errorf("%w: invalid or duplicate schema", ErrSchema)
			}
			schemaSeen = true
			if err := json.Unmarshal(raw, &schema); err != nil || schema <= 0 {
				return 0, "", fmt.Errorf("%w: invalid schema", ErrSchema)
			}
		case strings.EqualFold(key, "game"):
			if gameSeen || len(raw) > maxGameIDBytes+2 {
				return 0, "", fmt.Errorf("%w: invalid or duplicate game", ErrGame)
			}
			gameSeen = true
			if err := json.Unmarshal(raw, &game); err != nil || !validGameID(game) {
				return 0, "", fmt.Errorf("%w: invalid game id", ErrGame)
			}
		}
	}
	if _, err := d.Token(); err != nil {
		return 0, "", err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return 0, "", fmt.Errorf("store: trailing room data: %v", err)
	}
	if !schemaSeen || schema != 1 && schema != 2 {
		return 0, "", fmt.Errorf("%w: schema %d", ErrSchema, schema)
	}
	if schema == 1 && !gameSeen {
		game = "duel"
	}
	if !validGameID(game) {
		return 0, "", fmt.Errorf("%w: schema %d requires game", ErrGame, schema)
	}
	return schema, game, nil
}

func (r *Codecs) decode(data []byte) (Inspection, error) {
	schema, game, err := fileKind(data)
	if err != nil {
		return nil, err
	}
	if schema == Schema && game == "duel" {
		return decodeDuel(data)
	}
	if schema != 2 {
		return nil, fmt.Errorf("%w: game %q in schema %d", ErrGame, game, schema)
	}
	decode := r.schema2[game]
	if decode == nil {
		return nil, fmt.Errorf("%w: game %q in schema %d", ErrGame, game, schema)
	}
	record, err := decode(data)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("store: codec %q returned no record", game)
	}
	h := record.Retention()
	if ValidateCode(h.Code) != nil || h.Updated.IsZero() || h.ArchivedAt != nil && h.ArchivedAt.IsZero() {
		return nil, fmt.Errorf("store: invalid retention header for %q", game)
	}
	switch h.Status {
	case StatusLobby, StatusPlaying, StatusFinished, StatusArchived:
	default:
		return nil, fmt.Errorf("store: invalid status for %q", game)
	}
	return record, nil
}

func (r *Codecs) read(path string) (Inspection, []byte, error) {
	data, err := readCapped(path)
	if err != nil {
		return nil, nil, err
	}
	record, err := r.decode(data)
	if err != nil {
		return nil, nil, fmt.Errorf("store: decoding %s: %w", path, err)
	}
	return record, data, nil
}

// duelInspection delegates to the existing File and replay without normalizing
// or adding fields to schema 1. The typed Duel Load/Save APIs stay separate.
type duelInspection struct{ file *File }

func decodeDuel(data []byte) (Inspection, error) {
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	f.Version = max(f.Version, len(f.Actions))
	return duelInspection{&f}, nil
}
func (d duelInspection) Retention() RetentionHeader {
	f := d.file
	return RetentionHeader{Code: f.Code, Status: f.Status, Updated: f.Updated, ArchivedAt: f.ArchivedAt}
}
func (d duelInspection) Check() (bool, error) {
	s, _, err := assess(d.file)
	if err != nil {
		return false, err
	}
	return engine.HashString(&s) != d.file.Hash, nil
}
