package ident

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

// ClaimLen is the length of a claim: the first 22 characters of the
// unpadded base64url form of an HMAC-SHA256, 132 bits of the MAC. That is
// as hard to guess as a token, and short enough for a link.
const ClaimLen = 22

// ErrBadClaim is wrapped by every ValidateClaim error.
var ErrBadClaim = errors.New("ident: malformed claim")

// Claim derives the claim of one seat of one room: the capability that lets
// any connection act in room code as userID (see doc.go, "Claims"). It is
//
//	base64url(HMAC-SHA256(secret, "claim:" + code + ":" + userID + ":" + epoch))[:22]
//
// where epoch is the seat's link epoch, which the room stores and raises to
// revoke every claim derived under the old one. The same inputs always
// derive the same claim under the same secret, so nothing is stored.
func (s Secret) Claim(code, userID string, epoch int) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte("claim:"))
	mac.Write([]byte(code))
	mac.Write([]byte(":"))
	mac.Write([]byte(userID))
	mac.Write([]byte(":"))
	mac.Write([]byte(strconv.Itoa(epoch)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:ClaimLen]
}

// VerifyClaim reports whether claim is the claim of userID in room code at
// epoch. It recomputes the claim and compares the two in constant time, so
// the time it takes says nothing about how much of a guess was right. A
// malformed claim is never valid.
func (s Secret) VerifyClaim(claim, code, userID string, epoch int) bool {
	if ValidateClaim(claim) != nil || userID == "" {
		return false
	}
	return hmac.Equal([]byte(claim), []byte(s.Claim(code, userID, epoch)))
}

// ValidateClaim checks that claim has the shape of a claim: exactly
// ClaimLen characters of the base64url alphabet.
func ValidateClaim(claim string) error {
	if len(claim) != ClaimLen {
		return fmt.Errorf("%w: length %d, want %d", ErrBadClaim, len(claim), ClaimLen)
	}
	for i := 0; i < len(claim); i++ {
		c := claim[i]
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_'
		if !ok {
			return fmt.Errorf("%w: character %d is not base64url", ErrBadClaim, i)
		}
	}
	return nil
}
