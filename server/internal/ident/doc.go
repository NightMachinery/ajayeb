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
// # Claims
//
// A claim moves one seat of one room to another device (a "device link",
// /play/<code>#claim=<claim>). It is derived, never stored:
//
//	claim = base64url(HMAC-SHA256(secret, "claim:" + code + ":" + userId + ":" + epoch))[:22]
//
// where epoch is a small counter the room keeps for each seat (0 until the
// seat's link is first reset). A join that carries a valid claim acts in
// that room, and only there, as the seat's user; the connection keeps its
// own user id everywhere else. The server checks a claim by recomputing it
// for the room's seats and comparing in constant time (Secret.VerifyClaim).
//
// A claim is a room-scoped capability, not the token:
//
//   - It reveals neither the token nor the user id: both go into the MAC,
//     and a MAC cannot be run backwards.
//   - It is worth nothing in any other room, because the room code is part
//     of what is MACed.
//   - Raising the seat's epoch (a link reset) revokes every claim made
//     under the old one, at once.
//
// That is why a claim may sit in a URL while a token never may. The link
// puts it in the fragment (after #), which a browser never sends in an
// HTTP request, so it reaches no server log; it stays in the address bar,
// so a reload keeps the same identity. A claim link is still a password for
// that seat in that room: whoever holds it plays the seat until it is
// reset.
//
// # Rules
//
//   - Tokens never appear in URLs, logs, room files or any server frame.
//     An invite link is just /play/<code>.
//   - Claims never appear in logs or room files, and the server sends one
//     only to the connection that asked for it, which must be the seat's
//     user or the game's admin.
//   - Token and claim formats are validated before use, and secrets and
//     MACs are compared in constant time.
//   - Derivations are domain separated by a prefix: "user:" for user ids,
//     "claim:" for claims.
package ident
