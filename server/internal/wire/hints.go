package wire

import (
	"encoding/json"
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file holds what the wire knows about a game's hint settings
// (docs/dev/decisions.md D23): the names of the hint families, the client
// frame with which the game's admin changes the settings, and the room log
// entry that records a change for both players. The settings themselves are
// view.HintSettings.

// HintFamily names one hint family: one switch of view.HintSettings, under
// that field's json name.
type HintFamily string

// The hint families, in the order the settings list them.
const (
	HintFamilyPrices   HintFamily = "prices"
	HintFamilyEconomy  HintFamily = "economy"
	HintFamilyScience  HintFamily = "science"
	HintFamilyMilitary HintFamily = "military"
	HintFamilyThreats  HintFamily = "threats"
	HintFamilyTempo    HintFamily = "tempo"
	HintFamilyWonders  HintFamily = "wonders"
	HintFamilyTokens   HintFamily = "tokens"
	HintFamilyScoring  HintFamily = "scoring"
	HintFamilyCounting HintFamily = "counting"
	HintFamilyRecap    HintFamily = "recap"
	HintFamilyAdvisor  HintFamily = "advisor"
)

// hintFamilies is every family in order. A test checks it against the
// fields of view.HintSettings, in both directions.
var hintFamilies = [...]HintFamily{
	HintFamilyPrices, HintFamilyEconomy, HintFamilyScience, HintFamilyMilitary,
	HintFamilyThreats, HintFamilyTempo, HintFamilyWonders, HintFamilyTokens,
	HintFamilyScoring, HintFamilyCounting, HintFamilyRecap, HintFamilyAdvisor,
}

// HintFamilies returns every hint family, in the order the settings list
// them.
func HintFamilies() []HintFamily {
	return append([]HintFamily{}, hintFamilies[:]...)
}

// hintField returns a pointer to the switch of family f in h, or nil for a
// name that is not a family.
func hintField(h *view.HintSettings, f HintFamily) *bool {
	switch f {
	case HintFamilyPrices:
		return &h.Prices
	case HintFamilyEconomy:
		return &h.Economy
	case HintFamilyScience:
		return &h.Science
	case HintFamilyMilitary:
		return &h.Military
	case HintFamilyThreats:
		return &h.Threats
	case HintFamilyTempo:
		return &h.Tempo
	case HintFamilyWonders:
		return &h.Wonders
	case HintFamilyTokens:
		return &h.Tokens
	case HintFamilyScoring:
		return &h.Scoring
	case HintFamilyCounting:
		return &h.Counting
	case HintFamilyRecap:
		return &h.Recap
	case HintFamilyAdvisor:
		return &h.Advisor
	}
	return nil
}

// HintOn reports whether family f is on in h (false for an unknown name).
func HintOn(h view.HintSettings, f HintFamily) bool {
	if p := hintField(&h, f); p != nil {
		return *p
	}
	return false
}

// SetHint switches family f in h on or off; an unknown name changes nothing.
func SetHint(h *view.HintSettings, f HintFamily, on bool) {
	if p := hintField(h, f); p != nil {
		*p = on
	}
}

// DiffHints lists the families that are on in next but not in prev (on) and
// those that are off in next but on in prev (off), each in family order and
// never nil.
func DiffHints(prev, next view.HintSettings) (on, off []HintFamily) {
	on, off = []HintFamily{}, []HintFamily{}
	for _, f := range hintFamilies {
		switch a, b := HintOn(prev, f), HintOn(next, f); {
		case !a && b:
			on = append(on, f)
		case a && !b:
			off = append(off, f)
		}
	}
	return on, off
}

// checkHintsComplete refuses a settings object that leaves out a family, so
// that a client built before a family existed cannot switch it off by
// silence. raw is the JSON object of the settings (DisallowUnknownFields has
// already refused names that are not families).
func checkHintsComplete(field string, raw json.RawMessage) error {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return fmt.Errorf("%s: %v", field, err)
	}
	for _, f := range hintFamilies {
		v, ok := keys[string(f)]
		if !ok {
			return fmt.Errorf("%s: family %q is missing", field, f)
		}
		if string(v) == "null" {
			return fmt.Errorf("%s: family %q is null", field, f)
		}
	}
	return nil
}

// Hints replaces the room's hint settings. Only the game's admin may send
// it, while the game has not ended; every connection then receives a state
// frame with the new settings and a room log entry naming the families
// that changed.
type Hints struct {
	T    string `json:"t" tstype:"'hints'"`
	Code string `json:"code"`
	// Seq is the client's request counter, shared with act and undo: an
	// applied hints frame uses up its seq, and a repeated one is only
	// acknowledged.
	Seq int `json:"seq"`
	// Hints are the complete new settings: every family must be present.
	Hints view.HintSettings `json:"hints"`
}

// NewHints returns a Hints frame with "t" filled in.
func NewHints(code string, seq int, h view.HintSettings) Hints {
	return Hints{T: "hints", Code: code, Seq: seq, Hints: h}
}

func (m Hints) validate() error {
	if err := checkNonEmptyBounded("hints: code", m.Code, maxCodeLen); err != nil {
		return err
	}
	if m.Seq < 0 {
		return fmt.Errorf("hints: seq %d is negative", m.Seq)
	}
	return nil
}

// MaxRoomLog bounds State.RoomLog: the newest entries are kept.
const MaxRoomLog = 64

// RoomLogKind says what a room log entry records.
type RoomLogKind string

// The room log kinds.
const (
	// RoomLogHints: the admin changed the hint settings.
	RoomLogHints RoomLogKind = "hints"
)

// RoomLogEntry is one room-level event of the game log: something that
// happened to the room rather than in the game, the same for every viewer.
// The client shows it among the game's own events, before the action at
// ply At.
type RoomLogEntry struct {
	Kind RoomLogKind `json:"kind" tstype:"'hints'"`
	// At is the game's ply when it happened: the entry comes after every
	// history entry of a smaller ply.
	At int `json:"at"`
	// Seat is the seat of the player who made the change; absent in a
	// hot-seat room, where one person holds both seats.
	Seat *int `json:"seat,omitempty" tstype:"0 | 1"`
	// On and Off are the families switched on and off (kind hints), in
	// family order.
	On  []HintFamily `json:"on"`
	Off []HintFamily `json:"off"`
}
