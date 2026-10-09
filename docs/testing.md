# Testing

Ajayeb is tested in layers, from the rules themselves up to a full game
played over a real WebSocket against a real server binary. Each layer trusts
the one below it and adds what that layer cannot catch on its own.

## Engine and AI unit tests

`server/internal/engine` and `server/internal/engine/catalog` carry the
rules tests, written from `docs/rules/` before the code that implements
each rule. `server/internal/ai` and `server/cmd/ajayeb-sim` cover bot
strength and the playout sweep. These run as part of `make test-go`, in
process, with no server, no socket and no browser; they are the cheapest
layer and catch almost everything.

## The view and protocol fixtures

`server/internal/engine/testdata/protocol/` holds one golden JSON file per
named position: what every viewer (`seat0`, `seat1`, `spectator`) sees
there, full view and filtered events side by side. `TestProtocolFixtures`
in `server/internal/engine/view_test.go` writes and checks them
(`-update` after a deliberate change). [protocol.md](protocol.md) describes
the format in full; this page only says where it sits among the other
layers and how `ajayeb-gen fixtures` turns it into something the web
client's own type checker reads (below).

## `internal/protocoltest`: full games over the real stack

`server/internal/protocoltest` plays whole games over real WebSocket
connections against an `httptest` server built from the actual `ws`,
`room` and `store` packages, the same wiring `cmd/ajayeb` uses. Nothing
here is mocked: a room is a real room actor, a bot is the real AI package
(with a one-iteration budget, so it is fast and deterministic without being
a different code path), and the socket is the real protocol.

It covers, per game mode (online with a spectator, hot-seat, versus a bot):

- a full playthrough to a result, with the replayed action log hashing to
  the same state as the room's own final state (`engine.Hash`);
- that no hidden information (a face-down card's identity, a later age's
  undealt deck, box guilds and wonders, the Great Library's undrawn tokens,
  the seed) appears in a connection's raw frames before it is entitled to
  see it. The check reconstructs what should still be hidden from the
  room's own ground-truth state (`room.Manager.Snapshot`) and scans every
  frame byte for it; there is no exported redaction helper to call
  directly, so this is a best-effort mirror of `internal/engine/view.go`'s
  documented redaction rules, not a byte-for-byte copy of its logic;
- reconnecting mid-game (the socket drops, `hello` and `join` again,
  resending the last unacknowledged `act` does not apply it twice);
- restarting the server on the same data directory mid-game and finishing
  the same game, with the same final hash;
- a stale `base`, a duplicate `seq`, an illegal key and an out-of-turn
  action, each rejected with the right code;
- undo in hot-seat and versus-bot rooms, checked by the resulting turn and
  choices rather than by the room's version number (undo does not rewind
  the version; it is a monotonic counter that goes up on every accepted
  request, undo included).

Run it with the race detector, through the guard, from `server/`:

    python3 ../tools/heavy.py run --kind test -- go test -race ./internal/protocoltest/...

## `ajayeb-drive`: a thin client against a running binary

`server/cmd/ajayeb-drive` is a small WebSocket client that plays full games
against a server that is already running, the way a real client would: it
never computes legality or cost, it only picks a random choice from the
view it is sent. It takes `--origin`, `--mode online|hotseat|ai`, `--games`
and `--seed`, exits non-zero on any protocol error or a game that ends
without a result, and prints one summary line per game.

`--concurrency` plays several games at once against the one server, which
is how we measure the server's resident memory under load: start a test
binary, run `ajayeb-drive --games 20 --concurrency 20`, and sample the
server process's RSS while it is in flight. Each connection sends a
synthetic `X-Forwarded-For` address, so a server started with `-trust-proxy`
naming the driver's own address can be driven past its default
per-address socket cap (16) for a true 20-connections-at-once run; without
`-trust-proxy` the header is simply ignored. A measured run of 20
concurrent bot games against a local test binary peaked at about 19 to 20
MB resident, comfortably under the working target of 40 MB for this
scenario (see [dev/ram-policy.md](dev/ram-policy.md) for the server's other
budgets).

## The TypeScript fixture check

The client's generated types (`web/src/lib/wire/*.gen.ts`, from `tygo`)
describe the wire shapes, but nothing used to prove that a real server
frame actually fits them. `ajayeb-gen fixtures` closes that gap:

- it reads every recorded frame in
  `server/internal/wire/testdata/messages/` and every viewer's view from
  `server/internal/engine/testdata/protocol/`, and splices them, almost
  verbatim, into `web/src/lib/wire/fixtures.gen.ts` as two `satisfies`
  assertions: `fixtures satisfies ServerMessage[]` and `protocolFixtures
  satisfies View[]`;
- `pnpm -C web check` (`svelte-check`, part of `make check-web`) then fails
  the moment a real recorded shape no longer fits the generated types,
  rather than at runtime in a browser;
- `web/src/lib/wire/fixtures.test.ts` adds a few runtime checks on top: the
  arrays are non-empty, every recorded frame type is represented, and
  nothing in a real frame or view is a JSON `null` (the engine's own
  convention is an empty array or an absent key, never `null`; a nil Go
  slice marshaled without this convention is exactly the kind of mistake
  this check exists to catch).

`ajayeb-gen` with no subcommand runs `fixtures` with its defaults, so
`make gen` and `make gen-check` regenerate and check this file along with
everything `tygo` produces.

Running `ajayeb-gen fixtures` for the first time (while
`server/internal/wire` and `server/internal/engine` still used `tygo`'s
fallback `never` type for a cross-package field, `view.GameEvent`) is what
actually found that gap: `tygo.yaml` needed a `type_mappings` entry for it,
exactly as the file's own comment already said it would. That is the
fixture check doing its job: a real recorded `state` frame's `events` field
stopped type-checking the moment something tried to use it as more than an
opaque blob.
