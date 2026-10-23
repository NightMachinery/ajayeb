package rules

import (
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

func designSlug(id catalog.DesignID) string {
	d, ok := catalog.DesignByID(id)
	if !ok {
		return ""
	}
	return d.Slug
}
func actionName(kind ActionKind) string {
	if kind < NumActionKinds {
		return actionVerbs[kind]
	}
	return ""
}

// FilterEvents never consults Private. Kind, action and source define each
// entitlement, so a wrong assertion or an unknown union fails closed.
func FilterEvents(evs []Event, viewer Seat) []view.Event {
	out := make([]view.Event, 0, len(evs))
	for _, e := range evs {
		if v, ok := filterEvent(e, viewer); ok {
			out = append(out, v)
		}
	}
	return out
}

func filterEvent(e Event, viewer Seat) (view.Event, bool) {
	v := view.Event{Kind: e.Kind.String()}
	switch e.Kind {
	case EvAgeStart:
		v.Age = int(e.Age)
	case EvHandsPassed:
		v.Age, v.Turn = int(e.Age), int(e.Turn)
	case EvGameOver:
	case EvSideChosen:
		if e.Seat >= MaxSeats || e.Side < catalog.Day || e.Side > catalog.Night {
			return view.Event{}, false
		}
		v.Seat, v.Side = int(e.Seat), e.Side.String()
	case EvMilitary:
		if e.Seat >= MaxSeats || e.Other >= MaxSeats || e.Outcome >= NumConflictOutcomes {
			return view.Event{}, false
		}
		v.Seat, v.Other, v.Age = int(e.Seat), int(e.Other), int(e.Age)
		v.Outcome = ([NumConflictOutcomes]string{"tie", "seat-wins", "other-wins"})[e.Outcome]
	case EvCoins:
		if e.Seat >= MaxSeats {
			return view.Event{}, false
		}
		v.Seat, v.Delta = int(e.Seat), int(e.Delta)
		switch e.Source {
		case CoinSale:
			v.Source = "sale"
		case CoinDesign:
			v.Source, v.Design = "design", designSlug(e.Design)
		case CoinStage:
			v.Source, v.Stage = "stage", int(e.Stage)
		default:
			return view.Event{}, false
		}
	case EvPlay:
		if e.Seat >= MaxSeats || e.Action < ActBuild || e.Action > ActSell {
			return view.Event{}, false
		}
		v.Seat, v.Play = int(e.Seat), actionName(e.Action)
		if e.Action == ActBuild || e.Action == ActStage {
			v.Left, v.Right, v.Printed = int(e.Left), int(e.Right), int(e.Printed)
		}
		if e.Action == ActStage {
			v.Stage = int(e.Stage)
		}
		if e.Action == ActBuild || e.Seat == viewer {
			v.Design = designSlug(e.Design)
		}
	case EvLeftoverDiscarded, EvCardKept:
		if e.Seat >= MaxSeats {
			return view.Event{}, false
		}
		v.Seat = int(e.Seat)
		if e.Seat == viewer {
			v.Design = designSlug(e.Design)
		}
	case EvDiscardBuild:
		if e.Seat >= MaxSeats {
			return view.Event{}, false
		}
		v.Seat, v.Design = int(e.Seat), designSlug(e.Design)
	case EvDiscardBuildNothing:
		if e.Seat >= MaxSeats {
			return view.Event{}, false
		}
		v.Seat = int(e.Seat)
	default:
		return view.Event{}, false
	}
	return v, true
}
