package wire

// PlatformProtocolVersion adds table and lobby families. ProtocolVersion remains
// the Duel client contract; Duel frames have identical bytes on either version.
const PlatformProtocolVersion = 2

// PlatformAck acknowledges a request from one tab, as Duel's Ack does.
type PlatformAck struct {
	ClientID string `json:"clientId"`
	Seq      int    `json:"seq"`
}

// PlatformHuman contains public identity, presence and role metadata. Temporary
// admins have seat powers but cannot promote or demote. PromotedBy records the
// real admin who granted the role; absence means no such promotion.
type PlatformHuman struct {
	UserID     string  `json:"userId"`
	Name       string  `json:"name"`
	NameNo     int     `json:"nameNo"`
	Connected  bool    `json:"connected"`
	Creator    bool    `json:"creator"`
	Admin      bool    `json:"admin"`
	Temporary  bool    `json:"temporary"`
	PromotedBy *string `json:"promotedBy,omitempty"`
}

// PlatformSeat has no hidden game information. A reserved seat stays reserved
// until an admin opens it; fillBots never silently fills a reservation.
type PlatformSeat struct {
	Seat   int     `json:"seat"`
	Kind   string  `json:"kind" tstype:"'open' | 'reserved' | 'human' | 'bot'"`
	UserID *string `json:"userId,omitempty"`
	Bot    *string `json:"bot,omitempty"`
}

// PreviousSeat is sent only in an admin's own frame, never public metadata.
type PreviousSeat struct {
	UserID string `json:"userId"`
	Seat   int    `json:"seat"`
}

// PlatformYou holds connection-specific metadata, including admin-only hints.
// No previous-seat hint is sent to a non-admin. A former player gets only its
// own FormerSeat, the seat it held last, while it watches a playing table.
// CanPromote goes to a real admin of a table: whether the admin cap leaves
// room for another admin.
type PlatformYou struct {
	UserID        string          `json:"userId"`
	Seat          *int            `json:"seat,omitempty"`
	PreviousSeats *[]PreviousSeat `json:"previousSeats,omitempty"`
	FormerSeat    *int            `json:"formerSeat,omitempty"`
	CanPromote    *bool           `json:"canPromote,omitempty"`
}

// PlatformLogEntry describes public control changes, never submitted action
// keys, observations or seeds. At is a completed batch count (zero in lobby).
type PlatformLogEntry struct {
	Kind    string  `json:"kind" tstype:"'seat' | 'role' | 'resign' | 'link'"`
	At      int     `json:"at"`
	By      string  `json:"by"`
	UserID  *string `json:"userId,omitempty"`
	Seat    *int    `json:"seat,omitempty"`
	Control *string `json:"control,omitempty" tstype:"'human' | 'bot'"`
	Role    *string `json:"role,omitempty" tstype:"'admin' | 'temporary' | 'human'"`
}

// PlatformLink and PlatformRelink extend the common device-link operation to
// seven seats. Actual room dispatch still restricts Duel to seats zero and one.
type PlatformLink struct {
	T    string `json:"t" tstype:"'link'"`
	Code string `json:"code"`
	Seat int    `json:"seat"`
}
type PlatformRelink struct {
	T    string `json:"t" tstype:"'relink'"`
	Code string `json:"code"`
	Seat int    `json:"seat"`
}

// PlatformLinked is a private response to a non-Duel device-link operation.
// Duel continues to emit Linked on both protocol versions.
type PlatformLinked struct {
	T     string `json:"t" tstype:"'linked'"`
	Code  string `json:"code"`
	Seat  int    `json:"seat"`
	Claim string `json:"claim"`
}
