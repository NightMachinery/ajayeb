package ident

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// TokenBytes is the length of a client token: 128 bits, enough that guessing
// one is infeasible, short enough to keep in localStorage comfortably.
const TokenBytes = 16

// TokenLen is the length of a token's wire form: TokenBytes encoded as
// unpadded base64url (RFC 4648 section 5), 22 characters.
const TokenLen = 22

// ErrBadToken is wrapped by every ValidateToken error.
var ErrBadToken = errors.New("ident: malformed token")

// NewToken generates a fresh random client token. In production the browser
// mints its own token and the server never calls this; it exists for tests
// and for any server-side tooling that needs a realistic token.
func NewToken() (string, error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("ident: generating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateToken checks that token has the shape of a real client token:
// exactly TokenLen characters of unpadded base64url, decoding to exactly
// TokenBytes bytes. A token is never looked up or compared against anything
// server side (the server has nothing to compare it to); this only rejects
// garbage before it is used to derive a user id.
func ValidateToken(token string) error {
	if len(token) != TokenLen {
		return fmt.Errorf("%w: length %d, want %d", ErrBadToken, len(token), TokenLen)
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadToken, err)
	}
	if len(b) != TokenBytes {
		return fmt.Errorf("%w: decodes to %d bytes, want %d", ErrBadToken, len(b), TokenBytes)
	}
	return nil
}
