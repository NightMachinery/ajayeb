// Package ident handles player identity without accounts.
//
// The browser keeps a random 128-bit token (22 characters of unpadded
// base64url) in localStorage and sends it only in the hello frame. The server
// never stores the token. It derives a stable user id as
//
//	userId = hex(HMAC-SHA256(secret, "user:" + token))[:32]
//
// where secret is read from <data>/secret.key, a file with mode 0600 that is
// created on first start.
//
// # Rules
//
//   - Tokens never appear in URLs, logs, room files or any server frame.
//     An invite link is just /play/<code>.
//   - Token formats are validated before use, and secrets and MACs are
//     compared in constant time.
//   - Derivations are domain separated by a prefix ("user:" now; a later
//     room-scoped seat claim would use "claim:<code>:<userId>").
package ident
