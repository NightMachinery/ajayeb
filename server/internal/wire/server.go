package wire

import "github.com/NightMachinery/ajayeb/server/internal/view"

// This file defines the server-to-client frames (doc.go's "server frames").
// The server only ever builds and marshals these; it never decodes its own
// messages, so there is no DecodeServer. ServerMessage and the NewXxx
// constructors exist so tests (the hygiene test and the testdata recorder)
// can enumerate every type and build a well-formed value of each without
// setting "t" by hand at every call site.

// ServerMessage is implemented by every server frame type.
type ServerMessage interface {
	isServerMessage()
}

func (Welcome) isServerMessage() {}
func (Created) isServerMessage() {}
func (State) isServerMessage()   {}
func (Reject) isServerMessage()  {}
func (Error) isServerMessage()   {}
func (Linked) isServerMessage()  {}

// Welcome answers hello.
type Welcome struct {
	T      string     `json:"t" tstype:"'welcome'"`
	V      int        `json:"v"` // ProtocolVersion this server speaks
	Server ServerInfo `json:"server"`
}

// NewWelcome returns a Welcome frame with "t" filled in.
func NewWelcome(v int, info ServerInfo) Welcome { return Welcome{T: "welcome", V: v, Server: info} }

// ServerInfo identifies the running server build.
type ServerInfo struct {
	Version string `json:"version"` // build version (git describe)
	Rules   int    `json:"rules"`   // engine.RulesVersion this server plays
}

// Created answers a successful create: the new room's code. It also tells
// every connection of a finished room the code of its rematch room; a
// connection that joined that room with a claim (a device link) then also
// receives the claim of its seat in the rematch room, so it can follow.
type Created struct {
	T     string  `json:"t" tstype:"'created'"`
	Code  string  `json:"code"`
	Claim *string `json:"claim,omitempty"`
}

// NewCreated returns a Created frame with "t" filled in.
func NewCreated(code string) Created { return Created{T: "created", Code: code} }

// RoomStatus is a room's lifecycle state, as a client sees it (there is no
// wire equivalent of store.StatusArchived: an archived room simply stops
// being reachable, as a "room-not-found" error).
type RoomStatus string

// The room statuses.
const (
	RoomStatusLobby    RoomStatus = "lobby"
	RoomStatusPlaying  RoomStatus = "playing"
	RoomStatusFinished RoomStatus = "finished"
)

// SeatKind says who holds a seat.
type SeatKind string

// The seat kinds.
const (
	SeatKindHuman SeatKind = "human"
	SeatKindBot   SeatKind = "bot"
	SeatKindOpen  SeatKind = "open" // an online room's second seat, not yet claimed
)

// SeatInfo is one seat, as the room frame reports it.
type SeatInfo struct {
	Seat      int       `json:"seat"`
	Name      string    `json:"name"`
	Kind      SeatKind  `json:"kind"`
	Bot       *BotLevel `json:"bot,omitempty"` // the bot's level; set only when Kind is SeatKindBot
	Connected bool      `json:"connected"`
	You       bool      `json:"you"` // this connection holds this seat
	// NameNo tells the seat's name apart from an equal name (compared
	// without case and outer spaces) that was in the room first: 0 for
	// none, else 2, 3 and so on, which the client shows after the name. It
	// never changes when people leave and come back; a rename recomputes
	// only the renamed seat's.
	NameNo int `json:"nameNo"`
}

// RoomInfo is the room-level information in a state frame: everything about
// the room itself, as opposed to the game running in it (which is View).
type RoomInfo struct {
	Mode       Mode       `json:"mode"`
	Status     RoomStatus `json:"status"`
	Seats      []SeatInfo `json:"seats"`
	Spectators int        `json:"spectators"`
	// Hints are the game's hint settings, the same for every connection
	// (players and spectators alike); the room sets them.
	Hints view.HintSettings `json:"hints"`
	// AllowUndo says whether the game allows undo at all, fixed when the
	// room was created: false in an online room, which has no undo, and in
	// a hot-seat or ai room created with it off. State.CanUndo says whether
	// an undo would be accepted right now.
	AllowUndo bool `json:"allowUndo"`
	// Tutorial says the room is a tutorial room (Create.Tutorial) whose
	// script this server knows: the player's frames carry State.Tutorial.
	Tutorial bool `json:"tutorial"`
}

// NewRoomInfo returns a RoomInfo with seats normalized to a non-nil,
// possibly empty slice.
func NewRoomInfo(mode Mode, status RoomStatus, seats []SeatInfo, spectators int) RoomInfo {
	if seats == nil {
		seats = []SeatInfo{}
	}
	return RoomInfo{Mode: mode, Status: status, Seats: seats, Spectators: spectators}
}

// YouInfo is what this connection may do.
type YouInfo struct {
	// Seats are the seats this connection may act for: both in a hot-seat
	// room, one in an online or ai room, none for a spectator.
	Seats  []int `json:"seats"`
	CanAct bool  `json:"canAct"` // a legal action exists for one of Seats right now
	// Admin says this connection's user is the game's admin, who may change
	// the hint settings (the hints frame) and ask for and reset any seat's
	// device link (link, relink): the player who created the game.
	Admin bool `json:"admin"`
}

// NewYouInfo returns a YouInfo with seats normalized to a non-nil, possibly
// empty slice.
func NewYouInfo(seats []int, canAct bool) YouInfo {
	if seats == nil {
		seats = []int{}
	}
	return YouInfo{Seats: seats, CanAct: canAct}
}

// Ack acknowledges the act or undo this state frame is a direct response
// to, for the sender's own idempotency bookkeeping.
type Ack struct {
	ClientID string `json:"clientId"`
	Seq      int    `json:"seq"`
}

// MaxHistory bounds State.History: the newest entries are kept. A game has
// at most engine.MaxGameActions actions, well below it, so in practice the
// history is always the whole game (a room test checks the bound).
const MaxHistory = 512

// HistoryEntry is one applied action of the game, as one viewer may see it:
// the ply it was applied at, the seat that took it (for a resign, the seat
// that resigned) and its events, filtered for the viewer exactly as the
// events of a live state frame are.
type HistoryEntry struct {
	Ply    int              `json:"ply"`
	Seat   int              `json:"seat" tstype:"0 | 1"`
	Events []view.GameEvent `json:"events"`
}

// UndoUnavailableReason says why this connection cannot undo. The first
// matching reason wins: online, disabled, spectator, game over, no human
// decision. An available undo has no reason.
type UndoUnavailableReason string

const (
	UndoUnavailableOnline          UndoUnavailableReason = "online"
	UndoUnavailableDisabled        UndoUnavailableReason = "disabled"
	UndoUnavailableSpectator       UndoUnavailableReason = "spectator"
	UndoUnavailableGameOver        UndoUnavailableReason = "game-over"
	UndoUnavailableNoHumanDecision UndoUnavailableReason = "no-human-decision"
)

// State is a full snapshot: the room, what this connection may do, an
// optional ack, the events the triggering action produced (redacted for this
// viewer) and the full game view. Every state frame is a complete snapshot,
// not a diff, so a slow or reconnecting client only ever needs the latest
// one (see internal/ws's latest-wins outbox).
type State struct {
	T       string   `json:"t" tstype:"'state'"`
	Code    string   `json:"code"`
	Version int      `json:"version"`
	Room    RoomInfo `json:"room"`
	You     YouInfo  `json:"you"`
	// CanUndo says whether an undo from this connection, sent with this
	// frame's version as its base, would be accepted now: a hot-seat or ai
	// room that allows undo (RoomInfo.AllowUndo), a game that is not over,
	// and a human decision to take back.
	CanUndo bool `json:"canUndo"`
	// UndoUnavailable is present exactly when CanUndo is false.
	UndoUnavailable *UndoUnavailableReason `json:"undoUnavailable,omitempty"`
	// Thinking is the seat whose bot is searching for its move right now;
	// absent when no bot is searching.
	Thinking *int `json:"thinking,omitempty" tstype:"0 | 1"`
	// Ack is set only when this state is the direct reply to an act or undo
	// from the connection receiving it.
	Ack    *Ack             `json:"ack,omitempty"`
	Events []view.GameEvent `json:"events"`
	// History is every applied action of the game so far, oldest first (at
	// most MaxHistory, the newest), each filtered for this viewer. Only the
	// first state frame of a join, the answer to a resync and the frames
	// that follow an undo carry it; a live frame does not. Events, when a
	// frame carries both, happened after History's last entry.
	History *[]HistoryEntry `json:"history,omitempty"`
	// RoomLog is every room-level entry of the game log so far (at most
	// MaxRoomLog, the newest), oldest first: things that happened to the
	// room rather than in the game, such as a change of the hint settings.
	// Every frame carries the whole list, the same for every viewer.
	RoomLog []RoomLogEntry `json:"roomLog"`
	// Tutorial is where the tutorial stands, in a tutorial room's frames
	// for the connection that holds the player's seat; absent in every
	// other frame (other rooms, spectators).
	Tutorial *TutorialState `json:"tutorial,omitempty"`
	View     view.View      `json:"view"`
}

// NewState returns a State frame with "t" filled in. events is normalized to
// a non-nil, possibly empty slice, and RoomLog starts empty. CanUndo and
// Thinking start out false and absent; the room sets them. UndoUnavailable
// starts with the first public restriction, or no-human-decision until the
// room has checked its history.
func NewState(code string, version int, room RoomInfo, you YouInfo, ack *Ack, events []view.GameEvent, v view.View) State {
	if events == nil {
		events = []view.GameEvent{}
	}
	reason := UndoUnavailableNoHumanDecision
	switch {
	case room.Mode == ModeOnline:
		reason = UndoUnavailableOnline
	case !room.AllowUndo:
		reason = UndoUnavailableDisabled
	case len(you.Seats) == 0:
		reason = UndoUnavailableSpectator
	case v.Phase == view.PhaseGameOver:
		reason = UndoUnavailableGameOver
	}
	return State{UndoUnavailable: &reason, T: "state", Code: code, Version: version, Room: room, You: you, Ack: ack, Events: events, RoomLog: []RoomLogEntry{}, View: v}
}

// RejectCode says why an act, undo or similar request was refused without
// ending the connection: the client can usually recover (resync, or just
// try again).
type RejectCode string

// The reject codes.
const (
	RejectCodeStale       RejectCode = "stale"         // Base did not match the room's current version; a fresh state follows
	RejectCodeIllegal     RejectCode = "illegal"       // the key is not in the current legal set
	RejectCodeNotYourTurn RejectCode = "not-your-turn" // this connection does not hold the seat to move
	RejectCodeRate        RejectCode = "rate"          // the client is sending faster than the rate limit allows
	RejectCodeBadRequest  RejectCode = "bad-request"   // the frame itself was malformed in some other way
	RejectCodeNotAdmin    RejectCode = "not-admin"     // only the game's admin may do that (change the hint settings; another seat's device link)
	RejectCodeGameOver    RejectCode = "game-over"     // the game has ended, so its settings no longer change
	RejectCodeUndoOff     RejectCode = "undo-off"      // the game was created with undo off
)

// Reject answers one request (identified by its Seq) that could not be
// applied. code is a stable, machine-readable reason; msg is a developer
// hint, never a string to show a player (see package wire's doc comment on
// display strings).
type Reject struct {
	T    string     `json:"t" tstype:"'reject'"`
	Seq  int        `json:"seq"`
	Code RejectCode `json:"code"`
	Msg  string     `json:"msg"`
}

// NewReject returns a Reject frame with "t" filled in.
func NewReject(seq int, code RejectCode, msg string) Reject {
	return Reject{T: "reject", Seq: seq, Code: code, Msg: msg}
}

// ErrorCode says why the connection itself is being refused or closed, as
// opposed to one request (RejectCode).
type ErrorCode string

// The error codes.
const (
	ErrorCodeRoomNotFound ErrorCode = "room-not-found" // no such code (or it is archived)
	ErrorCodeRoomFull     ErrorCode = "room-full"      // the room takes no more connections (spectators, or one user's tabs)
	ErrorCodeForbidden    ErrorCode = "forbidden"      // this user may not do that in this room
	ErrorCodeBadHello     ErrorCode = "bad-hello"      // the hello frame itself was invalid
	ErrorCodeProtocol     ErrorCode = "protocol"       // unsupported protocol version
	ErrorCodeBusy         ErrorCode = "busy"           // the server holds as many rooms as it can; try again later
	ErrorCodeBadClaim     ErrorCode = "bad-claim"      // the join's claim is not a current device link of this room
)

// Error reports a connection-level problem.
type Error struct {
	T    string    `json:"t" tstype:"'error'"`
	Code ErrorCode `json:"code"`
	Msg  string    `json:"msg"`
}

// NewError returns an Error frame with "t" filled in.
func NewError(code ErrorCode, msg string) Error { return Error{T: "error", Code: code, Msg: msg} }

// serverMessageTypes lists one representative value of every server frame
// type, for the hygiene test and the testdata recorder to walk.
var serverMessageTypes = []ServerMessage{
	NewWelcome(ProtocolVersion, ServerInfo{Version: "dev", Rules: 1}),
	NewCreated("K7QX2M"),
	NewState("K7QX2M", 1, NewRoomInfo(ModeHotseat, RoomStatusPlaying, nil, 0), NewYouInfo(nil, false), nil, nil, view.View{}),
	NewReject(1, RejectCodeStale, "base 3 is behind the current version 5"),
	NewError(ErrorCodeRoomNotFound, "no room with that code"),
	NewLinked("K7QX2M", 0, "AAAAAAAAAAAAAAAAAAAAAA"),
}
