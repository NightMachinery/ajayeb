package fakeengine

import (
	"strconv"
	"strings"
)

var actionNames = [...]string{ActBuild: "build", ActTuck: "tuck", ActSell: "sell", ActSalvage: "salvage"}

// String returns the action kind's wire name.
func (k ActionKind) String() string {
	if k == 0 || int(k) >= len(actionNames) {
		return ""
	}
	return actionNames[k]
}

var pointNames = [...]string{PointNone: "none", PointTurn: "turn", PointSalvage: "salvage", PointLastCard: "last-card"}

// String returns the point kind's wire name.
func (k PointKind) String() string {
	if int(k) >= len(pointNames) {
		return ""
	}
	return pointNames[k]
}

var phaseNames = [...]string{PhaseTurn: "turn", PhaseSalvage: "salvage", PhaseLastCard: "last-card", PhaseOver: "over"}

// String returns the phase's wire name.
func (p Phase) String() string {
	if p == 0 || int(p) >= len(phaseNames) {
		return ""
	}
	return phaseNames[p]
}

// Key returns the action's canonical key: build:<design>:<left>:<right>,
// tuck:<design>, sell:<design> or salvage:<design>. A key never names a
// seat.
func (a Action) Key() string {
	slug := a.Design.Slug()
	switch a.Kind {
	case ActBuild:
		return "build:" + slug + ":" + strconv.Itoa(int(a.Left)) + ":" + strconv.Itoa(int(a.Right))
	case ActTuck, ActSell, ActSalvage:
		return a.Kind.String() + ":" + slug
	}
	return ""
}

// ParseKey parses a canonical key. It accepts only the exact strings Key
// produces; membership in Legal, not parsing, makes a key acceptable.
func ParseKey(key string) (Action, bool) {
	parts := strings.Split(key, ":")
	var a Action
	switch parts[0] {
	case "build":
		if len(parts) != 4 {
			return Action{}, false
		}
		l, errL := strconv.ParseUint(parts[2], 10, 8)
		r, errR := strconv.ParseUint(parts[3], 10, 8)
		if errL != nil || errR != nil {
			return Action{}, false
		}
		a = Action{Kind: ActBuild, Left: uint8(l), Right: uint8(r)}
	case "tuck":
		a.Kind = ActTuck
	case "sell":
		a.Kind = ActSell
	case "salvage":
		a.Kind = ActSalvage
	default:
		return Action{}, false
	}
	if a.Kind != ActBuild && len(parts) != 2 {
		return Action{}, false
	}
	d, ok := DesignBySlug(parts[1])
	if !ok {
		return Action{}, false
	}
	a.Design = d
	if a.Key() != key {
		return Action{}, false
	}
	return a, true
}

// pointID returns the logical id of a point, which follows from the
// position alone: age-A-turn-T, age-A-turn-T-salvage or age-A-last-card.
func pointID(kind PointKind, age, turn uint8) string {
	a := "age-" + strconv.Itoa(int(age))
	switch kind {
	case PointTurn:
		return a + "-turn-" + strconv.Itoa(int(turn))
	case PointSalvage:
		return a + "-turn-" + strconv.Itoa(int(turn)) + "-salvage"
	case PointLastCard:
		return a + "-last-card"
	}
	return ""
}
