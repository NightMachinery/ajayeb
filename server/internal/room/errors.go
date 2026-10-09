package room

import (
	"errors"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The errors of the Manager's methods. The Manager has already pushed the
// matching wire frame into the client's outbox when it returns one of
// them; the transport only decides whether to keep the connection.
var (
	// ErrRoomNotFound: no room has that code, its code is malformed, or its
	// file is archived. Frame: error room-not-found.
	ErrRoomNotFound = errors.New("room: no such room")
	// ErrRoomFull: the room takes no more connections of this kind: a new
	// spectator once the room has MaxClients connections, or one more
	// connection of a user who already has MaxClients there. Frame: error
	// room-full.
	ErrRoomFull = errors.New("room: the room is full")
	// ErrForbidden: this user may not do that in this room (join someone
	// else's hot-seat or bot room, rematch as a spectator). Frame: error
	// forbidden.
	ErrForbidden = errors.New("room: forbidden")
	// ErrTooManyRooms: the server holds as many rooms in memory as it
	// allows (a create, a rematch, or a join that must load a room). Frame:
	// error busy.
	ErrTooManyRooms = errors.New("room: too many rooms in memory")
	// ErrNotJoined: a request names a room this client has not joined.
	// Frame: reject bad-request.
	ErrNotJoined = errors.New("room: the client has not joined that room")
	// ErrNotOver: rematch before the game is over. Frame: reject
	// bad-request.
	ErrNotOver = errors.New("room: the game is not over")
	// ErrClosed: the Manager is shutting down. No frame: the transport
	// closes the connection with "service restart".
	ErrClosed = errors.New("room: the server is shutting down")
	// ErrInvalidSpec: the room asked for cannot exist: an unknown mode or
	// bot level, seats that do not fit the mode, or an action log or
	// scenario that does not replay. Frame: error forbidden.
	ErrInvalidSpec = errors.New("room: invalid room settings")
	// ErrStorage: the room could not be read or written. Frame: error
	// room-not-found for a load, nothing else (the room keeps running in
	// memory and retries).
	ErrStorage = errors.New("room: storage failure")
)

// errorCode maps an error of this package to the wire error code the
// client receives.
func errorCode(err error) wire.ErrorCode {
	switch {
	case errors.Is(err, ErrRoomFull):
		return wire.ErrorCodeRoomFull
	case errors.Is(err, ErrTooManyRooms):
		return wire.ErrorCodeBusy
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrInvalidSpec):
		return wire.ErrorCodeForbidden
	default:
		return wire.ErrorCodeRoomNotFound
	}
}

// errorMsg is the developer hint sent with an error frame (never shown to a
// player).
func errorMsg(err error) string {
	switch {
	case errors.Is(err, ErrRoomFull):
		return "the room takes no more connections"
	case errors.Is(err, ErrTooManyRooms):
		return "the server has too many rooms open; try again later"
	case errors.Is(err, ErrForbidden):
		return "this user may not do that in this room"
	case errors.Is(err, ErrInvalidSpec):
		return "invalid room settings"
	default:
		return "no room with that code"
	}
}
