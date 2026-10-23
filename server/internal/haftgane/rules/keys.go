package rules

import (
	"errors"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Frozen after M15.haftgane-core-types. Keys encode syntax, never legality.
type ActionKind uint8

const (
	ActNone ActionKind = iota
	ActSide
	ActBuild
	ActStage
	ActSell
	ActBuildDiscarded
	NumActionKinds
)

var actionVerbs = [NumActionKinds]string{"", "side", "build", "stage", "sell", "build-discarded"}
var ErrBadKey = errors.New("rules: invalid canonical action key")

// Action is a comparable union: side uses Side; build/stage use Design and
// Left/Right; sell/discard-build use Design only. All unused fields are zero.
type Action struct {
	Kind        ActionKind
	Design      catalog.DesignID
	Side        catalog.Side
	Left, Right uint8
}

// Valid checks a complete tagged action against the registered design ids.
// It does not check a position, affordability or membership in an offer.
func (a Action) Valid() bool {
	if a.Kind != ActSide {
		if _, known := catalog.DesignByID(a.Design); !known {
			return false
		}
	}
	switch a.Kind {
	case ActSide:
		return (a.Side == catalog.Day || a.Side == catalog.Night) && a.Design == 0 && a.Left == 0 && a.Right == 0
	case ActBuild, ActStage:
		return a.Design.Valid() && a.Side == catalog.NoSide
	case ActSell, ActBuildDiscarded:
		return a.Design.Valid() && a.Side == catalog.NoSide && a.Left == 0 && a.Right == 0
	}
	return false
}

// Key returns empty for an invalid union or an unregistered design. Amounts
// span uint8 here; Legal will later offer only affordable undominated payments.
func (a Action) Key() string {
	if !a.Valid() {
		return ""
	}
	if a.Kind == ActSide {
		return "side:" + a.Side.String()
	}
	design, ok := catalog.DesignByID(a.Design)
	if !ok {
		return ""
	}
	key := actionVerbs[a.Kind] + ":" + design.Slug
	if a.Kind == ActBuild || a.Kind == ActStage {
		key += ":" + strconv.Itoa(int(a.Left)) + ":" + strconv.Itoa(int(a.Right))
	}
	return key
}
func ParseKey(key string) (Action, error) {
	if len(key) == 0 || len(key) > MaxKeyBytes {
		return Action{}, ErrBadKey
	}
	parts := strings.Split(key, ":")
	var a Action
	switch parts[0] {
	case "side":
		if len(parts) != 2 {
			return Action{}, ErrBadKey
		}
		a.Kind = ActSide
		switch parts[1] {
		case "day":
			a.Side = catalog.Day
		case "night":
			a.Side = catalog.Night
		default:
			return Action{}, ErrBadKey
		}
	case "build", "stage", "sell", "build-discarded":
		need := 2
		if parts[0] == "build" || parts[0] == "stage" {
			need = 4
		}
		if len(parts) != need {
			return Action{}, ErrBadKey
		}
		for kind := ActBuild; kind < NumActionKinds; kind++ {
			if parts[0] == actionVerbs[kind] {
				a.Kind = kind
			}
		}
		id, ok := catalog.DesignBySlug(parts[1])
		if !ok {
			return Action{}, ErrBadKey
		}
		a.Design = id
		if need == 4 {
			left, err := strconv.ParseUint(parts[2], 10, 8)
			if err != nil {
				return Action{}, ErrBadKey
			}
			right, err := strconv.ParseUint(parts[3], 10, 8)
			if err != nil {
				return Action{}, ErrBadKey
			}
			a.Left, a.Right = uint8(left), uint8(right)
		}
	default:
		return Action{}, ErrBadKey
	}
	if !a.Valid() || a.Key() != key {
		return Action{}, ErrBadKey
	}
	return a, nil
}
