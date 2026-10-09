package store

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// CodeLength is the length of a room code.
const CodeLength = 6

// codeAlphabet is deliberately missing the characters people confuse at a
// glance: 0/O, 1/I/L. 31 symbols, all of them unambiguous read aloud or
// copied from a screenshot.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// ErrBadCode is wrapped by every ValidateCode error.
var ErrBadCode = errors.New("store: malformed room code")

// ValidateCode checks that code has the exact shape GenerateCode produces:
// CodeLength characters, every one from codeAlphabet. Every path that turns
// an untrusted code (one taken from a join request, for instance) into a
// file path goes through this first, so a code can never smuggle a path
// separator or a "..".
func ValidateCode(code string) error {
	if len(code) != CodeLength {
		return fmt.Errorf("%w: length %d, want %d", ErrBadCode, len(code), CodeLength)
	}
	for _, r := range code {
		if !strings.ContainsRune(codeAlphabet, r) {
			return fmt.Errorf("%w: character %q is not in the room-code alphabet", ErrBadCode, r)
		}
	}
	return nil
}

// GenerateCode returns a fresh random room code: CodeLength characters from
// codeAlphabet, drawn with crypto/rand (via math/big, so each character is
// unbiased: no character is more likely than another from a naive modulo).
func GenerateCode() (string, error) {
	n := big.NewInt(int64(len(codeAlphabet)))
	b := make([]byte, CodeLength)
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", fmt.Errorf("store: generating room code: %w", err)
		}
		b[i] = codeAlphabet[idx.Int64()]
	}
	return string(b), nil
}
