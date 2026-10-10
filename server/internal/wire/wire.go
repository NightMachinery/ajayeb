package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ProtocolVersion is the wire protocol version this build of the server (and
// the client it ships with) speaks. hello and welcome both carry it as "v";
// an incompatible change to the frames in this package bumps it.
const ProtocolVersion = 1

// MaxClientFrameBytes bounds one decoded client frame. internal/ws enforces
// the same figure at the transport level (its own ReadLimit) before a frame
// ever reaches DecodeClient; this is a second, independent check so that
// package wire rejects an oversize frame correctly even when called
// directly, such as from a test.
const MaxClientFrameBytes = 8 << 10

// ErrBadFrame is wrapped by every error DecodeClient returns: the frame is
// too large, names an unrecognized "t", has a field DisallowUnknownFields
// would catch, or fails one of a frame's own semantic checks (an empty
// code, an unknown mode, and so on).
var ErrBadFrame = errors.New("wire: malformed client frame")

// strictUnmarshal decodes raw into v, rejecting any JSON field v does not
// declare and any data left over after the one value.
func strictUnmarshal(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("trailing data after the JSON value")
	}
	return nil
}

// peekType reads just the "t" field of a frame, permissively (every other
// field is still unknown at this point), so DecodeClient can pick which
// concrete type to strictly decode into next.
func peekType(raw []byte) (string, error) {
	var head struct {
		T string `json:"t"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return "", err
	}
	if head.T == "" {
		return "", errors.New(`missing "t"`)
	}
	return head.T, nil
}

// frameErr wraps a decode error with ErrBadFrame and the frame type it
// happened in.
func frameErr(t string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrBadFrame, t, err)
}

// Field-length limits shared by several frame types' validate methods.
// They are generous bounds against abuse (MaxClientFrameBytes already caps
// the whole frame), not an attempt to pick a "correct" display-name length.
const (
	maxNameLen      = 64  // Hello.Name, Create.Names entries
	maxClientIDLen  = 128 // Hello.ClientID
	maxCodeLen      = 32  // room codes; package store's own alphabet is the authority on the exact shape
	maxActionKeyLen = 96  // Act.Key; matches the longest key engine.ParseKey accepts, with slack
)

// checkNonEmptyBounded rejects an empty string, one longer than max runes,
// or one containing a character refused by unsafeRune.
func checkNonEmptyBounded(field, s string, max int) error {
	if s == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	return checkBounded(field, s, max)
}

// checkBounded is checkNonEmptyBounded without the non-empty requirement,
// for fields that may legitimately be absent (an empty display name means
// "use a default").
func checkBounded(field, s string, max int) error {
	n := 0
	for _, r := range s {
		if unsafeRune(r) {
			return fmt.Errorf("%s contains a control or invisible formatting character", field)
		}
		n++
		if n > max {
			return fmt.Errorf("%s is longer than %d characters", field, max)
		}
	}
	return nil
}

// unsafeRune reports whether r may not appear in a client string. Control
// characters would make a poor display name and could smuggle terminal
// escapes into a log. Format characters (Unicode category Cf) are invisible:
// the bidi overrides, embeddings, isolates and marks reorder the text
// around a name wherever it is shown (dir="auto" does not undo an explicit
// override), and the zero width ones let a name look like another. Three
// kinds of Cf stay allowed because real text needs them: the zero width
// non-joiner and joiner (Persian and other scripts, emoji sequences) and the
// tag characters of emoji flag sequences.
func unsafeRune(r rune) bool {
	switch {
	case unicode.IsControl(r):
		return true
	case r == 0x200C, r == 0x200D, r >= 0xE0020 && r <= 0xE007F: // ZWNJ, ZWJ, tags
		return false
	}
	return unicode.Is(unicode.Cf, r)
}

// discriminator extracts the literal T.-field value tygo emits for a message
// type from its tstype tag (for example `'hello'` -> "hello"). It is used
// only by the hygiene test, to check every type's "t" field against its own
// type name without hand-maintaining a parallel list.
func discriminator(tag string) (string, bool) {
	s := strings.TrimSpace(tag)
	if len(s) < 2 || s[0] != '\'' || s[len(s)-1] != '\'' {
		return "", false
	}
	return s[1 : len(s)-1], true
}
