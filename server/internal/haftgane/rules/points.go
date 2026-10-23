package rules

import (
	"fmt"
	"strconv"
)

type PointKind uint8

const (
	PointNone PointKind = iota
	PointSides
	PointTurn
	PointDiscardBuild
	PointLastCard
	NumPointKinds
)

// Point stores fixed values. Its canonical id is computed, not a string inside
// the game value. Required is all seats or one decider; pending picks are absent.
type Point struct {
	Kind      PointKind
	Age, Turn uint8
	Required  SeatSet
}

func (p Point) ID() string {
	if p.Kind == PointSides && p.Age == 0 && p.Turn == 0 {
		return "sides"
	}
	if p.Age < 1 || p.Age > NumAges {
		return ""
	}
	prefix := "age-" + strconv.Itoa(int(p.Age))
	if p.Kind == PointLastCard && p.Turn == MaxTurns {
		return prefix + "-last-card"
	}
	if p.Turn < 1 || p.Turn > TurnsPerAge {
		return ""
	}
	if p.Kind == PointTurn {
		return prefix + "-turn-" + strconv.Itoa(int(p.Turn))
	}
	if p.Kind == PointDiscardBuild {
		return prefix + "-turn-" + strconv.Itoa(int(p.Turn)) + "-discard-build"
	}
	return ""
}
func (p Point) Validate(n uint8) error {
	all := AllSeats(n)
	if all == 0 || p.Required&^all != 0 {
		return fmt.Errorf("rules: invalid point seats")
	}
	if p.Kind == PointNone {
		if p != (Point{}) {
			return fmt.Errorf("rules: invalid closed point")
		}
		return nil
	}
	if p.ID() == "" {
		return fmt.Errorf("rules: invalid point reference")
	}
	if p.Kind == PointSides || p.Kind == PointTurn {
		if p.Required != all {
			return fmt.Errorf("rules: simultaneous point needs all seats")
		}
	} else if p.Required.Count() != 1 {
		return fmt.Errorf("rules: ability point needs one seat")
	}
	return nil
}
