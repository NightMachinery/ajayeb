package tableroom

import (
	"crypto/sha256"
	"encoding/base64"
)

// A table and a lobby remember each client (a tab) by its id, in their seq
// acknowledgements and pending floors. The wire allows an id of 128
// characters of almost any kind, which encoding/json may escape to six
// bytes each, so remembering ids as they are would let the record grow
// with their lengths past the store's read cap. An id of at most
// maxPlainClient characters from the web client's alphabet (base64url,
// which also covers its UUIDs) is remembered as it is; any other is
// remembered by its digest: "~" and the unpadded base64url SHA-256 of the
// id, maxClientKey characters that no plain id can equal. The record's
// bounds rest on this (TestLargestRecordFitsTheStore).
const (
	maxPlainClient = 40
	maxClientKey   = 1 + 43
)

// ClientKey is the form in which a table or a lobby remembers client id.
func ClientKey(id string) string {
	if len(id) <= maxPlainClient && plainClient(id) {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return "~" + base64.RawURLEncoding.EncodeToString(sum[:])
}

func plainClient(id string) bool {
	for i := range len(id) {
		switch c := id[i]; {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}
