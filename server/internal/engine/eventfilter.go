package engine

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file filters the events of an action for one viewer, with the same
// redaction as the view (view.go). Like the view it is a whitelist: each
// event kind copies into the DTO only the fields that kind uses (listed on
// view.GameEvent), converted to slugs and named numbers, so a field set by
// mistake on an engine event never reaches a client.
//
// A private event (Event.Private) reaches only its Seat. Every EvLibraryDraw
// is private, whatever its Private flag says: the Great Library's offer is
// the drawer's secret. The other viewers receive a public summary of each
// draw instead (the drawer and the place in the offer, never the token), so
// they can still show that tokens were drawn. Any other private event is
// dropped for them.

// FilterEvents returns the events evs as viewer v may see them, in order.
// The result is never nil.
func FilterEvents(evs []Event, v Viewer) []view.GameEvent {
	out := make([]view.GameEvent, 0, len(evs))
	seat, seated := v.Seat()
	for i := range evs {
		e := &evs[i]
		if e.Kind == EvNone || e.Kind >= numEventKinds {
			continue
		}
		if e.Private || e.Kind == EvLibraryDraw {
			if !seated || seat != e.Seat {
				if e.Kind == EvLibraryDraw {
					out = append(out, view.GameEvent{
						Kind: view.EventKindLibraryDraw, Seat: dtoInt(int(e.Seat)), Place: dtoInt(int(e.A)),
					})
				}
				continue
			}
			ge := eventView(e)
			ge.Private = true
			out = append(out, ge)
			continue
		}
		out = append(out, eventView(e))
	}
	return out
}

// eventView converts one event that the viewer may see in full, copying
// only the fields its kind uses.
func eventView(e *Event) view.GameEvent {
	ge := view.GameEvent{Kind: view.EventKind(e.Kind.String())}
	a, b := int(e.A), int(e.B)
	seat := func() { ge.Seat = dtoInt(int(e.Seat)) }
	switch e.Kind {
	case EvDraft:
		seat()
		ge.Wonder = dtoWonderPtr(e.Wonder)
		auto := a == 1
		ge.Auto = &auto
	case EvDraftRound:
		ge.Round = dtoInt(a)
	case EvAgeStart:
		ge.Age = dtoInt(a)
	case EvBuild, EvDestroy, EvRebuild:
		seat()
		ge.Card = dtoCardPtr(e.Card)
	case EvDiscard:
		seat()
		ge.Card = dtoCardPtr(e.Card)
		ge.Amount = dtoInt(a)
	case EvWonder:
		seat()
		ge.Wonder = dtoWonderPtr(e.Wonder)
		ge.Card = dtoCardPtr(e.Card)
	case EvPay:
		seat()
		ge.Amount = dtoInt(a)
		ge.ToOpponent = dtoInt(b)
	case EvReveal:
		ge.Slot = dtoInt(a)
		ge.Card = dtoCardPtr(e.Card)
	case EvCoins:
		seat()
		ge.Amount = dtoInt(a)
		ge.Balance = dtoInt(b)
		ge.Card = dtoCardPtr(e.Card)
		ge.Wonder = dtoWonderPtr(e.Wonder)
		ge.Token = dtoTokenPtr(e.Token)
	case EvPawn:
		seat()
		ge.From = dtoInt(a)
		ge.To = dtoInt(b)
	case EvLoot:
		seat()
		ge.Amount = dtoInt(a)
		ge.Value = dtoInt(b)
	case EvPair:
		seat()
		if sym := eventSymbol(a); sym != "" {
			ge.Symbol = &sym
		}
	case EvToken:
		seat()
		ge.Token = dtoTokenPtr(e.Token)
		lib := a == 1
		ge.FromLibrary = &lib
	case EvLibraryDraw:
		seat()
		ge.Token = dtoTokenPtr(e.Token)
		ge.Place = dtoInt(a)
	case EvWonderRemoved:
		seat()
		ge.Wonder = dtoWonderPtr(e.Wonder)
	case EvEffectSkipped:
		seat()
		ge.Wonder = dtoWonderPtr(e.Wonder)
		ge.Card = dtoCardPtr(e.Card)
		if k, ok := pendingKind(Phase(a)); ok {
			ge.Skipped = &k
		}
	case EvExtraTurn, EvExtraTurnLost:
		seat()
	case EvStarterChosen:
		seat()
		ge.Starter = dtoInt(a)
	case EvGameOver:
		if k := VictoryKind(a); k > VictoryNone && k < numVictoryKinds {
			vk := view.VictoryKind(k.String())
			ge.Victory = &vk
		}
		if int8(b) != SharedVictory {
			ge.Winner = dtoInt(b)
		}
	}
	return ge
}

// eventSymbol names the science symbol a, or returns "" for no symbol.
func eventSymbol(a int) view.ScienceSymbol {
	if a > int(catalog.NoSymbol) && a < int(catalog.NumSymbols) {
		return view.ScienceSymbol(catalog.Symbol(a).String())
	}
	return ""
}

// pendingKind returns the pending kind of a phase that opens a choice.
func pendingKind(p Phase) (view.PendingKind, bool) {
	switch p {
	case PhaseChooseStarter:
		return view.PendingKindChooseStarter, true
	case PhaseDestroyCard:
		return view.PendingKindDestroyCard, true
	case PhaseBuildFromDiscard:
		return view.PendingKindBuildFromDiscard, true
	case PhasePickBoardToken:
		return view.PendingKindPickBoardToken, true
	case PhasePickLibraryToken:
		return view.PendingKindPickLibraryToken, true
	}
	return "", false
}
