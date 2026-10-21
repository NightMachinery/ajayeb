package wire

// LobbyOptions replaces the lobby's complete platform options. Absent values
// mean random seating, random Duel starter, undo enabled and final reveal on.
// Game-specific options are deferred until their rules and view contracts land.
type LobbyOptions struct {
	Seating     *string `json:"seating,omitempty" tstype:"'random' | 'lobby'"`
	FirstPlayer *int    `json:"firstPlayer,omitempty" tstype:"0 | 1"`
	AllowUndo   *bool   `json:"allowUndo,omitempty"`
	FinalReveal *bool   `json:"finalReveal,omitempty"`
}

// LobbyCreate opens the shared lobby, without choosing a game package. Start
// selects Duel for two seats or Haftgane for three through seven.
type LobbyCreate struct {
	T         string       `json:"t" tstype:"'lobby-create'"`
	SeatCount int          `json:"seatCount"`
	Options   LobbyOptions `json:"options"`
}

// LobbyEdit is an admin operation except release, which releases the sender's
// own slot. Its operation selects exactly one payload: seats/seatCount,
// options/options, add-bot/seat+bot, remove-bot/seat, seat/seat+userId,
// remove-human/userId, release/no payload, open-reservation/seat.
type LobbyEdit struct {
	T         string        `json:"t" tstype:"'lobby-edit'"`
	Code      string        `json:"code"`
	Seq       int           `json:"seq"`
	Op        string        `json:"op" tstype:"'seats' | 'options' | 'add-bot' | 'remove-bot' | 'seat' | 'remove-human' | 'release' | 'open-reservation'"`
	SeatCount *int          `json:"seatCount,omitempty"`
	Options   *LobbyOptions `json:"options,omitempty"`
	Seat      *int          `json:"seat,omitempty"`
	UserID    *string       `json:"userId,omitempty"`
	Bot       *string       `json:"bot,omitempty"`
}

// LobbyRole uses TableRole's role policy in the lobby.
type LobbyRole struct {
	T      string  `json:"t" tstype:"'lobby-role'"`
	Code   string  `json:"code"`
	Seq    int     `json:"seq"`
	Op     string  `json:"op" tstype:"'promote' | 'demote' | 'step-down'"`
	UserID *string `json:"userId,omitempty"`
}

// LobbyStart requires a seated human admin. FillBots explicitly fills open
// seats only; reserved seats must first be opened. There is no ready counter.
type LobbyStart struct {
	T        string `json:"t" tstype:"'lobby-start'"`
	Code     string `json:"code"`
	Seq      int    `json:"seq"`
	FillBots bool   `json:"fillBots"`
}

// LobbyState is a full shared-lobby snapshot. The next snapshot after Start
// selects the game entry by its frame family under the same room code.
type LobbyState struct {
	T         string             `json:"t" tstype:"'lobby-state'"`
	Game      string             `json:"game" tstype:"'lobby'"`
	Code      string             `json:"code"`
	Revision  int                `json:"revision"`
	SeatCount int                `json:"seatCount"`
	Options   LobbyOptions       `json:"options"`
	Seats     []PlatformSeat     `json:"seats"`
	Humans    []PlatformHuman    `json:"humans"`
	You       PlatformYou        `json:"you"`
	CanStart  bool               `json:"canStart"`
	Ack       *PlatformAck       `json:"ack,omitempty"`
	RoomLog   []PlatformLogEntry `json:"roomLog"`
}

// LobbyReject is private; it never discloses reservation secrets or claims.
type LobbyReject struct {
	T    string `json:"t" tstype:"'lobby-reject'"`
	Seq  int    `json:"seq"`
	Code string `json:"code" tstype:"'bad-request' | 'stale' | 'not-admin' | 'forbidden' | 'rate' | 'storage'"`
}
