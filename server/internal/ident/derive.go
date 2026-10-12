package ident

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// userIDHexLen is how much of the HMAC's hex digest becomes a user id: 32
// hex characters, the first 16 bytes of the 32-byte MAC. Half the MAC is
// still 128 bits of preimage resistance, plenty for an id that is not itself
// a secret.
const userIDHexLen = 32

// UserID derives the stable server-side identity for a client token: the
// hex-encoded HMAC-SHA256 of the token under s's key, truncated to
// userIDHexLen characters. The same token always derives the same id under
// the same secret, and a token is rejected (ValidateToken) before it is
// MACed.
func (s Secret) UserID(token string) (string, error) {
	if err := ValidateToken(token); err != nil {
		return "", err
	}
	return s.derive("user", token), nil
}

// derive computes hex(HMAC-SHA256(secret, prefix+":"+id))[:userIDHexLen],
// the building block behind user ids. The prefix is domain separation: it
// is part of what is MACed, not appended to the result, so an id valid
// under one prefix (for example "user") can never be mistaken for a
// different kind of value derived under another prefix (a room-scoped seat
// claim, "claim", see Claim). Any comparison of a MAC against an expected
// value must use hmac.Equal (or crypto/subtle.ConstantTimeCompare), never
// ==, so that a timing difference cannot leak information about the
// secret; VerifyClaim is the one place this package compares.
func (s Secret) derive(prefix, id string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(prefix))
	mac.Write([]byte(":"))
	mac.Write([]byte(id))
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)[:userIDHexLen]
}
