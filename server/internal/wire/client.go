package wire

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file defines the client-to-server frames (doc.go's "client frames")
// and DecodeClient, which turns raw bytes from the websocket into one of
// them. Every frame type's zero value is never valid input: construct one
// with its NewXxx function, which fills in "t", or decode one with
// DecodeClient.

// ClientMessage is implemented by every client frame type. DecodeClient
// returns one; callers type-switch on the concrete type.
type ClientMessage interface {
	isClientMessage()
}

func (Hello) isClientMessage()   {}
func (Create) isClientMessage()  {}
func (Join) isClientMessage()    {}
func (Act) isClientMessage()     {}
func (Undo) isClientMessage()    {}
func (Resync) isClientMessage()  {}
func (Leave) isClientMessage()   {}
func (Rematch) isClientMessage() {}
func (Hints) isClientMessage()   {}
func (Link) isClientMessage()    {}
func (Relink) isClientMessage()  {}

// Hello is the first frame a client must send. The server ignores every
// other frame until it arrives, within the time internal/ws allows.
type Hello struct {
	T string `json:"t" tstype:"'hello'"`
	// V is the protocol version this client speaks (ProtocolVersion, from
	// the client's own build). The server decides what to do about a
	// mismatch; decoding only requires the field to be present.
	V int `json:"v"`
	// Token is the client's identity token (see package ident).
	Token string `json:"token"`
	// Name is the display name to use until the player changes it. Empty
	// means "the server picks a default".
	Name string `json:"name"`
	// ClientID identifies this tab/connection for act/undo idempotency
	// (seq) across a reconnect.
	ClientID string `json:"clientId"`
}

// NewHello returns a Hello frame with "t" filled in.
func NewHello(v int, token, name, clientID string) Hello {
	return Hello{T: "hello", V: v, Token: token, Name: name, ClientID: clientID}
}

func (m Hello) validate() error {
	if err := ident.ValidateToken(m.Token); err != nil {
		return fmt.Errorf("hello: token: %w", err)
	}
	if err := checkNonEmptyBounded("hello: clientId", m.ClientID, maxClientIDLen); err != nil {
		return err
	}
	if err := checkBounded("hello: name", m.Name, maxNameLen); err != nil {
		return err
	}
	return nil
}

// Mode is the kind of room Create starts.
type Mode string

// The modes.
const (
	ModeOnline  Mode = "online"
	ModeHotseat Mode = "hotseat"
	ModeAI      Mode = "ai"
)

// BotLevel is the strength of the bot in an "ai" room.
type BotLevel string

// The bot levels.
const (
	BotLevelEasy   BotLevel = "easy"
	BotLevelNormal BotLevel = "normal"
	BotLevelHard   BotLevel = "hard"
)

// SeatChoice is which seat Create's caller takes: a specific seat (0 or 1),
// or SeatRandom to let the server pick. On the wire it is the JSON number 0
// or 1, or the string "random"; TypeScript sees exactly that union
// (tstype:"'random' | 0 | 1" on the field that holds one). The zero
// SeatChoice is seat 0, so a Create frame with no "seat" key at all decodes
// to "take seat 0", not to "random" -- a defined, harmless default rather
// than an error.
type SeatChoice struct {
	random bool
	seat   int
}

// SeatRandom is "let the server pick".
var SeatRandom = SeatChoice{random: true}

// SeatNumber is a specific seat, 0 or 1. seat outside that range still
// builds a value (there is no error return), but it is never valid: every
// path that turns a SeatChoice into a seat goes through Seat, which reports
// the problem.
func SeatNumber(seat int) SeatChoice { return SeatChoice{seat: seat} }

// Random reports whether the choice is "random".
func (c SeatChoice) Random() bool { return c.random }

// Seat returns the chosen seat and true, or (0, false) for SeatRandom.
func (c SeatChoice) Seat() (int, bool) {
	if c.random {
		return 0, false
	}
	return c.seat, true
}

// MarshalJSON encodes the choice as the string "random" or the number 0/1.
func (c SeatChoice) MarshalJSON() ([]byte, error) {
	if c.random {
		return []byte(`"random"`), nil
	}
	return []byte(strconv.Itoa(c.seat)), nil
}

// UnmarshalJSON decodes the string "random" or the number 0/1, and rejects
// anything else (any other string, a float with a fractional part, a seat
// outside 0/1).
func (c *SeatChoice) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if s != "random" {
			return fmt.Errorf("seat %q is not \"random\" or 0 or 1", s)
		}
		*c = SeatChoice{random: true}
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("seat must be 0, 1 or \"random\": %w", err)
	}
	if n != 0 && n != 1 {
		return fmt.Errorf("seat %d is not 0 or 1", n)
	}
	*c = SeatChoice{seat: n}
	return nil
}

// Create starts a new room. The server replies with Created.
type Create struct {
	T    string     `json:"t" tstype:"'create'"`
	Mode Mode       `json:"mode"`
	Bot  *BotLevel  `json:"bot,omitempty"` // required when Mode is ModeAI; must be absent otherwise
	Seat SeatChoice `json:"seat" tstype:"'random' | 0 | 1"`
	// Names are display names by seat. A hot-seat or ai room uses only
	// Names[0] (the creator's own name); Names is never nil (an absent or
	// empty entry means "use a default").
	Names []string `json:"names"`
	// Hints are the game's hint settings (every family present); absent
	// means the defaults of the mode. The creator becomes the game's admin,
	// who may change them later (the hints frame).
	Hints *view.HintSettings `json:"hints,omitempty"`
	// AllowUndo says whether the game allows undo, fixed for its whole
	// life; absent means true. Hot-seat and ai rooms only: an online room
	// has no undo, so the field must be absent there.
	AllowUndo *bool `json:"allowUndo,omitempty"`
	// Tutorial asks for a tutorial room: the guided first game against the
	// Easy bot, on the tutorial script's fixed deal (package tutorial).
	// Absent means false. With true the mode must be ai, the bot easy and
	// the seat 0 or absent; the server then fixes the seed, the options and
	// the seats.
	Tutorial *bool `json:"tutorial,omitempty"`
}

// NewCreate returns a Create frame with "t" filled in. names is copied as
// given (pass nil or []string{} for "use defaults"); bot is nil unless mode
// is ModeAI.
func NewCreate(mode Mode, bot *BotLevel, seat SeatChoice, names []string) Create {
	if names == nil {
		names = []string{}
	}
	return Create{T: "create", Mode: mode, Bot: bot, Seat: seat, Names: names}
}

func (m Create) validate() error {
	switch m.Mode {
	case ModeOnline, ModeHotseat, ModeAI:
	default:
		return fmt.Errorf("create: unknown mode %q", m.Mode)
	}
	if m.Mode == ModeAI {
		if m.Bot == nil {
			return fmt.Errorf("create: mode %q requires \"bot\"", m.Mode)
		}
		switch *m.Bot {
		case BotLevelEasy, BotLevelNormal, BotLevelHard:
		default:
			return fmt.Errorf("create: unknown bot level %q", *m.Bot)
		}
	} else if m.Bot != nil {
		return fmt.Errorf("create: mode %q must not set \"bot\"", m.Mode)
	}
	if m.Mode == ModeOnline && m.AllowUndo != nil {
		return fmt.Errorf("create: mode %q has no undo and must not set \"allowUndo\"", m.Mode)
	}
	if seat, ok := m.Seat.Seat(); ok && seat != 0 && seat != 1 {
		return fmt.Errorf("create: seat %d is not 0 or 1", seat)
	}
	if m.Tutorial != nil && *m.Tutorial {
		if m.Mode != ModeAI || m.Bot == nil || *m.Bot != BotLevelEasy {
			return fmt.Errorf("create: a tutorial is mode %q with bot %q", ModeAI, BotLevelEasy)
		}
		if seat, ok := m.Seat.Seat(); !ok || seat != 0 {
			return errors.New("create: a tutorial player takes seat 0")
		}
	}
	if m.Hints != nil {
		if err := checkAdvisorStrength("create: hints", m.Hints.AdvisorStrength); err != nil {
			return err
		}
	}
	if len(m.Names) > 2 {
		return fmt.Errorf("create: %d names, at most 2", len(m.Names))
	}
	for i, name := range m.Names {
		if err := checkBounded(fmt.Sprintf("create: names[%d]", i), name, maxNameLen); err != nil {
			return err
		}
	}
	return nil
}

// Join asks to join an existing room as a player or spectator.
type Join struct {
	T    string `json:"t" tstype:"'join'"`
	Code string `json:"code"`
	// Claim, when present, is a device link's claim (package ident): with
	// a valid one the connection acts in this room as the user of the seat
	// it was made for, and as itself everywhere else. An invalid one is
	// refused with error bad-claim and counts against the join limit.
	Claim *string `json:"claim,omitempty"`
}

// NewJoin returns a Join frame with "t" filled in.
func NewJoin(code string) Join { return Join{T: "join", Code: code} }

// maxClaimLen bounds Join.Claim. A real claim is ident.ClaimLen
// characters; anything else of a sane length is left to the room, which
// refuses it as a bad claim (counted against the join limit) rather than as
// a malformed frame.
const maxClaimLen = 64

func (m Join) validate() error {
	if err := checkNonEmptyBounded("join: code", m.Code, maxCodeLen); err != nil {
		return err
	}
	if m.Claim != nil {
		return checkBounded("join: claim", *m.Claim, maxClaimLen)
	}
	return nil
}

// Act submits one game action.
type Act struct {
	T    string `json:"t" tstype:"'act'"`
	Code string `json:"code"`
	// Seq is this client's monotonic request counter, for idempotency: a
	// duplicate seq from the same client (identified by Hello.ClientID)
	// re-sends the current state without applying anything twice.
	Seq int `json:"seq"`
	// Base is the view version this action was chosen against. An act whose
	// Base is not the room's current version is rejected as stale.
	Base int `json:"base"`
	// Key is the action's canonical key (see internal/engine's Action.Key),
	// for example "construct:baths" or "wonder:pyramids:baths".
	Key string `json:"key"`
	// Seat is which seat is acting; set only in a hot-seat room, where one
	// connection controls both. Absent elsewhere, where the acting seat is
	// implied by the connection's identity.
	Seat *int `json:"seat,omitempty" tstype:"0 | 1"`
}

// NewAct returns an Act frame with "t" filled in. seat is nil outside
// hot-seat rooms.
func NewAct(code string, seq, base int, key string, seat *int) Act {
	return Act{T: "act", Code: code, Seq: seq, Base: base, Key: key, Seat: seat}
}

func (m Act) validate() error {
	if err := checkNonEmptyBounded("act: code", m.Code, maxCodeLen); err != nil {
		return err
	}
	if err := checkNonEmptyBounded("act: key", m.Key, maxActionKeyLen); err != nil {
		return err
	}
	if m.Seq < 0 {
		return fmt.Errorf("act: seq %d is negative", m.Seq)
	}
	if m.Base < 0 {
		return fmt.Errorf("act: base %d is negative", m.Base)
	}
	if m.Seat != nil && *m.Seat != 0 && *m.Seat != 1 {
		return fmt.Errorf("act: seat %d is not 0 or 1", *m.Seat)
	}
	return nil
}

// Undo asks to take back the previous decision. Hot-seat and ai rooms only.
type Undo struct {
	T    string `json:"t" tstype:"'undo'"`
	Code string `json:"code"`
	Seq  int    `json:"seq"`
	Base int    `json:"base"`
}

// NewUndo returns an Undo frame with "t" filled in.
func NewUndo(code string, seq, base int) Undo {
	return Undo{T: "undo", Code: code, Seq: seq, Base: base}
}

func (m Undo) validate() error {
	if err := checkNonEmptyBounded("undo: code", m.Code, maxCodeLen); err != nil {
		return err
	}
	if m.Seq < 0 {
		return fmt.Errorf("undo: seq %d is negative", m.Seq)
	}
	if m.Base < 0 {
		return fmt.Errorf("undo: base %d is negative", m.Base)
	}
	return nil
}

// Resync asks for a fresh state frame, with no side effect on the game.
type Resync struct {
	T    string `json:"t" tstype:"'resync'"`
	Code string `json:"code"`
}

// NewResync returns a Resync frame with "t" filled in.
func NewResync(code string) Resync { return Resync{T: "resync", Code: code} }

func (m Resync) validate() error {
	return checkNonEmptyBounded("resync: code", m.Code, maxCodeLen)
}

// Leave asks to leave a room (as a player or a spectator).
type Leave struct {
	T    string `json:"t" tstype:"'leave'"`
	Code string `json:"code"`
}

// NewLeave returns a Leave frame with "t" filled in.
func NewLeave(code string) Leave { return Leave{T: "leave", Code: code} }

func (m Leave) validate() error {
	return checkNonEmptyBounded("leave: code", m.Code, maxCodeLen)
}

// Rematch asks to start a fresh game with the same seats, once the current
// one is over.
type Rematch struct {
	T    string `json:"t" tstype:"'rematch'"`
	Code string `json:"code"`
}

// NewRematch returns a Rematch frame with "t" filled in.
func NewRematch(code string) Rematch { return Rematch{T: "rematch", Code: code} }

func (m Rematch) validate() error {
	return checkNonEmptyBounded("rematch: code", m.Code, maxCodeLen)
}

// DecodeClient strictly decodes one client frame: it rejects a frame over
// MaxClientFrameBytes, an unrecognized "t", any JSON field the matching type
// does not declare, trailing data after the value, and whatever that type's
// own validate finds wrong (an empty code, an unknown mode, and so on). The
// returned value's concrete type is one of this file's frame types; a
// caller normally type-switches on it.
func DecodeClient(raw []byte) (ClientMessage, error) {
	if len(raw) > MaxClientFrameBytes {
		return nil, fmt.Errorf("%w: frame is %d bytes, max is %d", ErrBadFrame, len(raw), MaxClientFrameBytes)
	}
	t, err := peekType(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
	}

	switch t {
	case "hello":
		var m Hello
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "create":
		var m Create
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		if m.Hints != nil {
			if err := checkHintsField("create: hints", raw); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
			}
		}
		return m, nil
	case "join":
		var m Join
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "act":
		var m Act
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "undo":
		var m Undo
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "resync":
		var m Resync
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "leave":
		var m Leave
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "rematch":
		var m Rematch
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "link":
		var m Link
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "relink":
		var m Relink
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	case "hints":
		var m Hints
		if err := strictUnmarshal(raw, &m); err != nil {
			return nil, frameErr(t, err)
		}
		if err := m.validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		if err := checkHintsField("hints: hints", raw); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadFrame, err)
		}
		return m, nil
	default:
		return nil, fmt.Errorf("%w: unknown frame type %q", ErrBadFrame, t)
	}
}

// clientMessageTypes lists one zero-ish value of every client frame type,
// built through its NewXxx constructor so each carries the right "t". The
// hygiene test and a handful of others walk this list instead of repeating
// it by hand.
var clientMessageTypes = []ClientMessage{
	NewHello(ProtocolVersion, strings.Repeat("A", ident.TokenLen), "", "c1"),
	NewCreate(ModeOnline, nil, SeatRandom, nil),
	NewJoin("K7QX2M"),
	NewAct("K7QX2M", 1, 0, "construct:baths", nil),
	NewUndo("K7QX2M", 1, 0),
	NewResync("K7QX2M"),
	NewLeave("K7QX2M"),
	NewRematch("K7QX2M"),
	NewHints("K7QX2M", 1, view.HintSettings{}),
	NewLink("K7QX2M", 0),
	NewRelink("K7QX2M", 1),
}

// checkHintsField checks the "hints" object of a frame that has one (see
// checkHintsComplete). The frame has already decoded strictly, so it is
// valid JSON with a "hints" object.
func checkHintsField(field string, raw []byte) error {
	var head struct {
		Hints json.RawMessage `json:"hints"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return fmt.Errorf("%s: %v", field, err)
	}
	if len(head.Hints) == 0 {
		return fmt.Errorf("%s is missing", field)
	}
	return checkHintsComplete(field, head.Hints)
}
