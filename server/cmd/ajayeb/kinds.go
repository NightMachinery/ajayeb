package main

import (
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
)

// roomKinds is the single production registration list for the Manager, store,
// reaper and read-only preflight. Haftgane's table kind is always registered,
// so saved tables load and archive like Duel's rooms; creating one is the
// separate -haftgane-tables switch.
func roomKinds(tables *tableroom.Tables) (*room.Kinds, error) {
	return room.NewKinds(room.DuelKind(), tables.Kind())
}
