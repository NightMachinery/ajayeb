package main

import "github.com/NightMachinery/ajayeb/server/internal/room"

// roomKinds is the single production registration list for the Manager, store,
// reaper and read-only preflight. No schema-2 actor or codec is enabled yet.
func roomKinds() (*room.Kinds, error) { return room.NewKinds(room.DuelKind()) }
