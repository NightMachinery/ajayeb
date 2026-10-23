package main

import (
	"github.com/NightMachinery/ajayeb/server/internal/lobbyroom"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
)

// roomKinds is the single production registration list for the Manager, store,
// reaper and read-only preflight. Haftgane's table kind and the shared lobby
// are always registered, so saved tables and lobbies load and archive like
// Duel's rooms; creating either is the separate -haftgane-tables switch.
func roomKinds(tables *tableroom.Tables, lobbies *lobbyroom.Lobbies) (*room.Kinds, error) {
	return room.NewKinds(room.DuelKind(), tables.Kind(), lobbies.Kind())
}
