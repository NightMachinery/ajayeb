package engine

import "errors"

// ErrBadKey is wrapped by every ParseKey error: the string is not the key of
// any action.
var ErrBadKey = errors.New("engine: bad action key")

// ErrMalformedAction is wrapped by every Action.Validate error: the action
// value is not well formed, whatever the game state.
var ErrMalformedAction = errors.New("engine: malformed action")

// Reason says why an action is not legal in the current state. The server
// maps reasons to error codes for clients, and the view uses them to explain
// disabled options, so each reason has a stable wire name (String).
type Reason uint8

// The reasons.
const (
	ReasonNone              Reason = iota
	ReasonGameOver                 // the game has ended
	ReasonWrongPhase               // this type of action is not what the game is waiting for
	ReasonMalformed                // the action is not well formed (Action.Validate failed)
	ReasonNotInDraft               // the wonder is not on offer in this draft round
	ReasonNotAccessible            // the card is not accessible in the structure
	ReasonWonderUnavailable        // the wonder is not one of the mover's unbuilt wonders, or it was removed
	ReasonUnaffordable             // the mover cannot pay the cost
	ReasonNotTarget                // the card is not an opponent card of the colour to destroy
	ReasonNotInDiscard             // the card is not in the discard pile
	ReasonTokenUnavailable         // the token is not on offer (not on the board, or not in the Great Library draw)
	numReasons
)

var reasonNames = [numReasons]string{
	"none", "gameOver", "wrongPhase", "malformed", "notInDraft", "notAccessible",
	"wonderUnavailable", "unaffordable", "notTarget", "notInDiscard", "tokenUnavailable",
}

// String returns the wire name, for example "unaffordable".
func (r Reason) String() string { return enumString(reasonNames[:], uint8(r), "reason") }

// MarshalText encodes the reason as its wire name.
func (r Reason) MarshalText() ([]byte, error) { return enumText(reasonNames[:], uint8(r), "reason") }

// UnmarshalText decodes a wire name exactly.
func (r *Reason) UnmarshalText(b []byte) error {
	v, err := parseEnum(reasonNames[:], b, "reason")
	if err == nil {
		*r = Reason(v)
	}
	return err
}

// IllegalError reports an action that is not legal in the state it was
// applied to. IsLegal and Apply return it as a *IllegalError; the state is
// left unchanged.
type IllegalError struct {
	Reason Reason
	Action Action
	Phase  Phase // the phase of the state the action was checked against
}

func (e *IllegalError) Error() string {
	return "engine: illegal action " + e.Action.String() + " in phase " + e.Phase.String() + ": " + e.Reason.String()
}

// illegal builds an *IllegalError for action a in state s.
func illegal(s *State, a Action, r Reason) *IllegalError {
	return &IllegalError{Reason: r, Action: a, Phase: s.Phase}
}
