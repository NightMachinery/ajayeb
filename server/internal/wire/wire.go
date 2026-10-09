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
// or one containing a control character (which would make a poor display
// name and could be used to smuggle terminal escapes into a log).
func checkNonEmptyBounded(field, s string, max int) error {
	if s == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	n := 0
	for _, r := range s {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains a control character", field)
		}
		n++
		if n > max {
			return fmt.Errorf("%s is longer than %d characters", field, max)
		}
	}
	return nil
}

// checkBounded is checkNonEmptyBounded without the non-empty requirement,
// for fields that may legitimately be absent (an empty display name means
// "use a default").
func checkBounded(field, s string, max int) error {
	if s == "" {
		return nil
	}
	n := 0
	for _, r := range s {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains a control character", field)
		}
		n++
		if n > max {
			return fmt.Errorf("%s is longer than %d characters", field, max)
		}
	}
	return nil
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
