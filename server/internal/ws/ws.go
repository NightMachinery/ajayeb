package ws

import "github.com/coder/websocket"

// CloseServiceRestart is the close status sent to every open socket when the
// server shuts down. Status 1012 (service restart) tells clients to reconnect
// with backoff, so a redeploy loses no game.
const CloseServiceRestart = websocket.StatusServiceRestart

// ReadLimit is the largest client frame the server accepts, in bytes. A
// larger frame closes the connection with status 1009 (message too big).
const ReadLimit int64 = 8 << 10
