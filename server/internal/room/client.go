package room

// Client is one connection, as the rooms see it: who is connected (the
// user id derived from the token by package ident, never the token itself),
// which tab (the client id that scopes act and undo seqs), the display name
// from hello, and the outbox the rooms push frames into.
//
// UserID, ClientID, Name and Out are fixed once the client is made. The
// transport owns the client: it must not call two Manager methods for the
// same client at the same time (a connection's read loop calls them one
// after another), because the Manager records on the client which room it
// has joined.
type Client struct {
	// ProtocolVersion is fixed by hello. Zero retains v1 for legacy callers.
	ProtocolVersion int
	UserID          string
	ClientID        string
	Name            string
	Out             *Outbox

	// AllowCreate, when set, is asked (on a room's goroutine) before the
	// rooms create a room on this client's behalf other than through
	// Manager.Create: today, a rematch that must create its rematch room. It
	// reports whether the creation may go ahead, and counts it if so; a
	// refusal reaches the client as reject rate. The transport points it at
	// its room creation limits. Nil allows every creation.
	AllowCreate func() bool

	// Addr is the key the transport counts the client's network address
	// under for its per-address limits (an IPv4 address, or an IPv6 /64),
	// or "" when it is not known. A bot room's searches wait for the bot
	// pool under the address of the person playing it (ai.Pool), so that
	// one address with many bot rooms cannot hold up the bots of others.
	Addr string

	joined string // the code of the room this client is attached to, or ""
}

// NewClient returns a client with a fresh outbox.
func NewClient(userID, clientID, name string) *Client {
	return &Client{UserID: userID, ClientID: clientID, Name: name, Out: NewOutbox()}
}

// Joined returns the code of the room the client is attached to, or "".
func (c *Client) Joined() string { return c.joined }
