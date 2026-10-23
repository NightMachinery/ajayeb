// Package fakeengine is a small, pure and deterministic stand-in for
// Haftgane's rules engine, written so that the Haftgane room actor
// (M16.haftgane-room-actor) can be built and tested before the real engine
// (M15, server/internal/haftgane/rules) exists.
//
// It implements exactly the concrete contract of
// docs/haftgane/engine-architecture.md section 16: NewGame, SeatOrder,
// Replay, and the Game methods Point, Legal, Resolve, ViewFor, History,
// Observe, Fingerprint and CheckInvariants, with the Batch and Pick types.
// Its view DTOs live in the sibling package fakeview, as the real ones live
// in haftgane/view, so moving the actor to the real engine
// (M16.haftgane-engine-contract) is a change of two import paths plus
// whatever the real types add.
//
// # The fake game
//
// Three to seven seats play two Ages of two simultaneous turns. Each seat is
// dealt a sorted hand of three cards per Age; on each turn every seat plays
// one card from its hand in one of three ways, chosen as an action key:
//
//   - build:<design>:<left>:<right> puts the design in the seat's city. A
//     design costs 0 to 2 coins, all paid to the neighbours in any split;
//     each split is its own key, as in Haftgane. A city holds each design
//     once.
//   - tuck:<design> puts the card face down under the seat's board, at most
//     two per city. The card stays private.
//   - sell:<design> puts the card face down on the pile for 2 coins. The card
//     stays private.
//
// After turn one the hands pass (left in Age 1, right in Age 2). After the
// last turn of an Age the card left in each hand goes face down to the pile,
// except for the seat whose board is the orchard, which keeps it.
//
// Two boards give one-seat ability points, so a game exercises the
// platform's one-seat points, including two of them in a row:
//
//   - The archive's holder, after a turn in which it tucked, takes any
//     design from the pile it has not built: a salvage point, key
//     salvage:<design>, whose offer is the private pile contents.
//   - The orchard's holder, after each Age's last turn (and its salvage),
//     plays its kept card: a last-card point with the turn's keys.
//
// Age 2's deck adds four single-copy relics, of which the four left undealt
// form the box. The score is the built designs' points, three per tucked
// card and a point per three coins; ties share a rank. At the end every
// hidden card is revealed unless Options.HideAtEnd is set.
//
// Designs, not copies, are the identity: hands, cities, the pile and keys
// hold design ids, so two copies of one design in a hand are one design
// listed twice and no id encodes a deal position.
//
// # Purity, and who may import it
//
// The package follows AGENTS.md "Go engine purity": the standard library
// only, no clocks, I/O, synchronisation or maps, integers only, and no
// randomness after NewGame (its own seeded splitmix64 streams in rng.go,
// since a Haftgane engine file may not import math/rand/v2). purity_test.go
// enforces this.
//
// It is test infrastructure. Test files anywhere may import it; non-test
// code only in the harness packages (internal/tabletest and below) and
// cmd/ajayeb-drive. guard_test.go fails if the server binary
// (cmd/ajayeb) could reach it, or any other package imports it.
package fakeengine
