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
	UserID   string
	ClientID string
	Name     string
	Out      *Outbox

	joined string // the code of the room this client is attached to, or ""
}

// NewClient returns a client with a fresh outbox.
func NewClient(userID, clientID, name string) *Client {
	return &Client{UserID: userID, ClientID: clientID, Name: name, Out: NewOutbox()}
}

// Joined returns the code of the room the client is attached to, or "".
func (c *Client) Joined() string { return c.joined }
