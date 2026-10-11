package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file defines actions and their canonical keys. It is frozen after
// slice S1: change it only through the coordinator.
//
// An action travels and is stored as its key, a short canonical string:
//
//	draft:<wonder>            take a wonder in the draft            draft:pyramids
//	construct:<card>          build an accessible card              construct:baths
//	discard:<card>            discard an accessible card for coins  discard:baths
//	wonder:<wonder>:<card>    build a wonder, tucking the card      wonder:pyramids:baths
//	destroy:<card>            destroy an opponent card (Zeus, Circus Maximus)  destroy:quarry
//	revive:<card>             build a discarded card (Mausoleum)    revive:garrison
//	token:<token>             take a progress token (board or Great Library)   token:law
//	starter:<seat>            choose who starts the next age (0 or 1)          starter:1
//	resign:<seat>             seat forfeits the game (0 or 1)                  resign:0
//
// Each action has exactly one key and each key exactly one action:
// ParseKey(a.Key()) == a for every valid action, and ParseKey accepts a string
// only if it is the key of its result. Slugs match exactly (no case folding,
// no whitespace, no "the-" prefix). Payment is never part of an action; the
// engine always applies the cheapest payment.
//
// resign is not a move of the game: LegalActions never lists it (so a bot,
// which only ever picks from that list, never resigns), yet IsLegal and
// Apply accept it for either seat at any time before the game is over. The
// server applies it when a seated player forfeits; it ends the game at once
// with the other seat's victory (VictoryForfeit). See legal.go.

// ActionType is the kind of an action.
type ActionType uint8

// The action types.
const (
	ActNone                ActionType = iota
	ActDraftWonder                    // Wonder
	ActConstructBuilding              // Card
	ActDiscardBuilding                // Card
	ActConstructWonder                // Wonder, and Card (the card tucked under it)
	ActDestroyCard                    // Card (an opponent card)
	ActBuildFromDiscard               // Card (a discarded card)
	ActChooseProgressToken            // Token (from the board or from the Great Library offer; the phase decides)
	ActChooseStarter                  // Starter
	ActResign                         // Starter holds the seat that resigns
	numActionTypes
)

// The verb of each action type: the first part of its key.
var actionVerbs = [numActionTypes]string{"", "draft", "construct", "discard", "wonder", "destroy", "revive", "token", "starter", "resign"}

// Which Action fields each type uses.
const (
	needCard uint8 = 1 << iota
	needWonder
	needToken
	needStarter
)

var actionFields = [numActionTypes]uint8{
	ActDraftWonder:         needWonder,
	ActConstructBuilding:   needCard,
	ActDiscardBuilding:     needCard,
	ActConstructWonder:     needWonder | needCard,
	ActDestroyCard:         needCard,
	ActBuildFromDiscard:    needCard,
	ActChooseProgressToken: needToken,
	ActChooseStarter:       needStarter,
	ActResign:              needStarter,
}

// Verb returns the first part of the type's keys, for example "construct",
// or "" for ActNone and invalid types.
func (t ActionType) Verb() string {
	if t < numActionTypes {
		return actionVerbs[t]
	}
	return ""
}

// String returns the verb, or "none" / "action#N".
func (t ActionType) String() string {
	switch {
	case t == ActNone:
		return "none"
	case t < numActionTypes:
		return actionVerbs[t]
	}
	return "action#" + strconv.Itoa(int(t))
}

// Action is one decision. It is a small comparable value (usable as a map
// key) and a tagged union on Type: only the fields that Type uses are set,
// and every other field is zero.
type Action struct {
	Type    ActionType
	Card    catalog.CardID   // construct, discard, wonder (the tucked card), destroy, revive
	Wonder  catalog.WonderID // draft, wonder
	Token   catalog.TokenID  // token
	Starter Seat             // starter; for resign, the seat that resigns
}

// DraftAction takes wonder w in the draft.
func DraftAction(w catalog.WonderID) Action { return Action{Type: ActDraftWonder, Wonder: w} }

// ConstructAction builds card c.
func ConstructAction(c catalog.CardID) Action { return Action{Type: ActConstructBuilding, Card: c} }

// DiscardAction discards card c for coins.
func DiscardAction(c catalog.CardID) Action { return Action{Type: ActDiscardBuilding, Card: c} }

// WonderAction builds wonder w, tucking card c under it.
func WonderAction(w catalog.WonderID, c catalog.CardID) Action {
	return Action{Type: ActConstructWonder, Wonder: w, Card: c}
}

// DestroyAction destroys the opponent's card c.
func DestroyAction(c catalog.CardID) Action { return Action{Type: ActDestroyCard, Card: c} }

// ReviveAction builds the discarded card c for free.
func ReviveAction(c catalog.CardID) Action { return Action{Type: ActBuildFromDiscard, Card: c} }

// TokenAction takes progress token t.
func TokenAction(t catalog.TokenID) Action { return Action{Type: ActChooseProgressToken, Token: t} }

// StarterAction makes seat s start the next age.
func StarterAction(s Seat) Action { return Action{Type: ActChooseStarter, Starter: s} }

// ResignAction makes seat s forfeit the game. It is never in LegalActions;
// see the top of this file.
func ResignAction(s Seat) Action { return Action{Type: ActResign, Starter: s} }

// Validate checks the action's shape, independently of any game state: a
// known type, every field the type uses set to a registered id (or, for
// Starter, to 0 or 1), and every other field zero. It does not say whether
// the action is legal now; IsLegal does. Errors wrap ErrMalformedAction.
func (a Action) Validate() error {
	if a.Type == ActNone || a.Type >= numActionTypes {
		return fmt.Errorf("%w: unknown type %d", ErrMalformedAction, a.Type)
	}
	need := actionFields[a.Type]
	verb := actionVerbs[a.Type]
	if need&needCard != 0 {
		if catalog.Card(a.Card) == nil {
			return fmt.Errorf("%w: %s: card id %d is not registered", ErrMalformedAction, verb, a.Card)
		}
	} else if a.Card != 0 {
		return fmt.Errorf("%w: %s takes no card", ErrMalformedAction, verb)
	}
	if need&needWonder != 0 {
		if catalog.Wonder(a.Wonder) == nil {
			return fmt.Errorf("%w: %s: wonder id %d is not registered", ErrMalformedAction, verb, a.Wonder)
		}
	} else if a.Wonder != 0 {
		return fmt.Errorf("%w: %s takes no wonder", ErrMalformedAction, verb)
	}
	if need&needToken != 0 {
		if catalog.Token(a.Token) == nil {
			return fmt.Errorf("%w: %s: token id %d is not registered", ErrMalformedAction, verb, a.Token)
		}
	} else if a.Token != 0 {
		return fmt.Errorf("%w: %s takes no token", ErrMalformedAction, verb)
	}
	if need&needStarter != 0 {
		if a.Starter > 1 {
			return fmt.Errorf("%w: %s: seat %d is not 0 or 1", ErrMalformedAction, verb, a.Starter)
		}
	} else if a.Starter != 0 {
		return fmt.Errorf("%w: %s takes no seat", ErrMalformedAction, verb)
	}
	return nil
}

// Key returns the action's canonical key, or "" if the action is malformed
// (see Validate). Use MarshalText to get the reason as an error instead.
func (a Action) Key() string {
	k, err := a.key()
	if err != nil {
		return ""
	}
	return k
}

func (a Action) key() (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	verb := actionVerbs[a.Type]
	switch a.Type {
	case ActDraftWonder:
		return verb + ":" + catalog.Wonder(a.Wonder).Slug, nil
	case ActConstructWonder:
		return verb + ":" + catalog.Wonder(a.Wonder).Slug + ":" + catalog.Card(a.Card).Slug, nil
	case ActChooseProgressToken:
		return verb + ":" + catalog.Token(a.Token).Slug, nil
	case ActChooseStarter, ActResign:
		if a.Starter == 0 {
			return verb + ":0", nil
		}
		return verb + ":1", nil
	default: // the card actions
		return verb + ":" + catalog.Card(a.Card).Slug, nil
	}
}

// String returns the key, or a debugging form for a malformed action.
func (a Action) String() string {
	if k, err := a.key(); err == nil {
		return k
	}
	return fmt.Sprintf("Action{Type:%d Card:%d Wonder:%d Token:%d Starter:%d}",
		a.Type, a.Card, a.Wonder, a.Token, a.Starter)
}

// MarshalText encodes the action as its key; a malformed action is an error.
// So in JSON an action is a string, for example "wonder:pyramids:baths".
func (a Action) MarshalText() ([]byte, error) {
	k, err := a.key()
	if err != nil {
		return nil, err
	}
	return []byte(k), nil
}

// UnmarshalText decodes a key strictly, as ParseKey does. On error the action
// is left unchanged.
func (a *Action) UnmarshalText(b []byte) error {
	v, err := ParseKey(string(b))
	if err != nil {
		return err
	}
	*a = v
	return nil
}

// maxKeyLen bounds keys. The longest real key is about 50 bytes.
const maxKeyLen = 96

// ParseKey parses a canonical action key. It is strict: it accepts exactly
// the strings that Key produces for valid actions and rejects everything
// else (unknown verbs, missing or extra parts, slugs of the wrong kind or not
// registered, any change of case or spacing, a seat other than "0" or "1").
// Errors wrap ErrBadKey.
func ParseKey(key string) (Action, error) {
	if key == "" || len(key) > maxKeyLen {
		return Action{}, badKey(key, "a key is 1 to "+strconv.Itoa(maxKeyLen)+" bytes")
	}
	verb, rest, ok := strings.Cut(key, ":")
	if !ok {
		return Action{}, badKey(key, "missing ':'")
	}
	switch verb {
	case "draft":
		w, ok := catalog.WonderBySlug(rest)
		if !ok {
			return Action{}, badKey(key, "unknown wonder")
		}
		return DraftAction(w), nil
	case "construct", "discard", "destroy", "revive":
		c, ok := catalog.CardBySlug(rest)
		if !ok {
			return Action{}, badKey(key, "unknown card")
		}
		a := Action{Card: c}
		switch verb {
		case "construct":
			a.Type = ActConstructBuilding
		case "discard":
			a.Type = ActDiscardBuilding
		case "destroy":
			a.Type = ActDestroyCard
		default:
			a.Type = ActBuildFromDiscard
		}
		return a, nil
	case "wonder":
		ws, cs, ok := strings.Cut(rest, ":")
		if !ok {
			return Action{}, badKey(key, "a wonder key is wonder:<wonder>:<card>")
		}
		w, ok := catalog.WonderBySlug(ws)
		if !ok {
			return Action{}, badKey(key, "unknown wonder")
		}
		c, ok := catalog.CardBySlug(cs)
		if !ok {
			return Action{}, badKey(key, "unknown card")
		}
		return WonderAction(w, c), nil
	case "token":
		t, ok := catalog.TokenBySlug(rest)
		if !ok {
			return Action{}, badKey(key, "unknown token")
		}
		return TokenAction(t), nil
	case "starter", "resign":
		var seat Seat
		switch rest {
		case "0":
		case "1":
			seat = 1
		default:
			return Action{}, badKey(key, "the seat is 0 or 1")
		}
		if verb == "resign" {
			return ResignAction(seat), nil
		}
		return StarterAction(seat), nil
	}
	return Action{}, badKey(key, "unknown verb")
}

func badKey(key, why string) error {
	if len(key) > maxKeyLen {
		key = key[:maxKeyLen] + "..."
	}
	return fmt.Errorf("%w %q: %s", ErrBadKey, key, why)
}
