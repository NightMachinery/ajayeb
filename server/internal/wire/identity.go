package wire

import "fmt"

// This file holds the frames of device links (package ident, "Claims"): a
// seat's link is /play/<code>#claim=<claim>, and a join that carries the
// claim acts in that room as the seat's user. The seat's own user and the
// game's admin may ask for a seat's link (link) and reset it (relink); the
// answer (linked) goes to the asking connection only, never in a state
// frame, and never to a spectator.

// Link asks for the device link of one seat: the server answers this
// connection alone with a Linked frame, or with reject not-admin when the
// connection is neither that seat's user nor the game's admin (a spectator
// never is), or reject bad-request when nobody holds the seat yet or a bot
// does.
type Link struct {
	T    string `json:"t" tstype:"'link'"`
	Code string `json:"code"`
	Seat int    `json:"seat" tstype:"0 | 1"`
}

// NewLink returns a Link frame with "t" filled in.
func NewLink(code string, seat int) Link { return Link{T: "link", Code: code, Seat: seat} }

func (m Link) validate() error {
	return validateSeatFrame("link", m.Code, m.Seat)
}

// Relink resets the device link of one seat, under the same rules as
// Link: every link made for the seat before stops working, the asking
// connection receives the new one in a Linked frame, and every connection
// receives a state frame whose room log records the reset.
type Relink struct {
	T    string `json:"t" tstype:"'relink'"`
	Code string `json:"code"`
	Seat int    `json:"seat" tstype:"0 | 1"`
}

// NewRelink returns a Relink frame with "t" filled in.
func NewRelink(code string, seat int) Relink { return Relink{T: "relink", Code: code, Seat: seat} }

func (m Relink) validate() error {
	return validateSeatFrame("relink", m.Code, m.Seat)
}

func validateSeatFrame(t, code string, seat int) error {
	if err := checkNonEmptyBounded(t+": code", code, maxCodeLen); err != nil {
		return err
	}
	if seat != 0 && seat != 1 {
		return fmt.Errorf("%s: seat %d is not 0 or 1", t, seat)
	}
	return nil
}

// Linked answers a link or relink frame: the current claim of one seat of
// the room. The client makes the link from it (/play/<code>#claim=<claim>).
// It goes only to the connection that asked.
type Linked struct {
	T     string `json:"t" tstype:"'linked'"`
	Code  string `json:"code"`
	Seat  int    `json:"seat" tstype:"0 | 1"`
	Claim string `json:"claim"`
}

// NewLinked returns a Linked frame with "t" filled in.
func NewLinked(code string, seat int, claim string) Linked {
	return Linked{T: "linked", Code: code, Seat: seat, Claim: claim}
}
