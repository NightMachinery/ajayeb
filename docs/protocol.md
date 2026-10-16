# Protocol

This page is the contract between the web client and the Go server. The
client opens one WebSocket to `/ws`, says `hello`, and then sends
**requests** (create a room, join one, act, undo, and so on) as JSON frames.
The server answers with frames of its own, chiefly `state`: a complete
snapshot of one room and its game, redacted for the connection that receives
it. The client never decides what is legal. It renders the `choices` of the
latest `state` and sends back their action keys.

The page covers, in order:

- the transport: the endpoint, origins, the life of a connection, close
  codes, pings and the outbox;
- every frame in each direction, with an example;
- the reject codes and the error codes;
- versions, idempotency and reconnecting;
- the limits;
- the HTTP endpoints beside the socket, including the development ones;
- three pieces of the contract that the engine owns and that other tools
  share: the view a viewer receives together with its events, the recorded
  game log, and the scenario format.

Everything is plain JSON. The Go types are the authority, and this page
explains them:

- the frames live in `server/internal/wire` (client frames in `client.go`,
  server frames in `server.go`) and the view in `server/internal/view`;
  `tygo` turns both into the TypeScript in `web/src/lib/wire/*.gen.ts`;
- a recorded example of every server frame is in
  `server/internal/wire/testdata/messages/`;
- the transport is `server/internal/ws`, the rooms that answer requests are
  `server/internal/room`, and the HTTP endpoints are
  `server/internal/httpapi`.

## Transport

### The endpoint

The socket lives at `/ws` on the same host and port as the web client, for
example `wss://example.org/ws`. Every WebSocket message is one **text** frame
holding one JSON object, whose `"t"` field names its type. A binary frame
closes the socket (status 1003). Nothing about the user travels in the URL:
the identity token goes only inside the `hello` frame, and the server never
logs it or sends it back.

### Origins

The server checks the `Origin` header of the upgrade request and accepts:

- each origin given with `-origin` (such as `https://example.org`; the flag
  repeats or takes a comma separated list), matched exactly on scheme, host
  and port, ignoring case;
- an origin whose host is the request's own `Host`, so the server's own page
  always works;
- a request with no `Origin` header, which a browser never sends but a tool
  such as `ajayeb-drive` does;
- with `-dev` only, `localhost` and `127.0.0.1` on any port, for the Vite
  dev server.

Any other origin is refused with HTTP 403 before the upgrade.

### Refusals before the upgrade

Some upgrade requests get an HTTP error instead of a socket:

- 403: the origin is not allowed;
- 426: the request is not a WebSocket upgrade (a malformed upgrade gets 400,
  and one that is not a GET gets 405);
- 429 `too-many-connections`: this client address already has 16 sockets
  open;
- 503 `busy`: the server already has 1024 sockets open;
- 503 `shutting-down`: the server is shutting down.

The 429 and 503 bodies have the shape of the HTTP API's errors and carry
`Retry-After: 5`:

```json
{"ok": false, "code": "too-many-connections", "msg": "Too Many Requests"}
```

The 400, 403, 405 and 426 bodies come from the WebSocket library and are
plain text.

### The client address

Six limits count per **client address**: on the socket, sockets, rooms
created, joins of codes that name no room, undos and bot searches; over HTTP,
game log downloads. The client address is the TCP peer's
address, unless that peer is one of the reverse proxies given with
`-trust-proxy` (IP addresses or CIDR prefixes, repeatable or comma
separated). For a trusted peer the server reads `X-Forwarded-For` from the
right: each trusted proxy appends the address it received the request from,
so the rightmost entry that is not itself a trusted proxy is the client.
Entries further left were written by the client and are never believed, and
a malformed entry stops the walk at the last believable address. A peer that
is not trusted cannot change its address, whatever headers it sends.

The limits count an IPv4 address on its own and an IPv6 address by its /64
prefix: one host normally holds a whole /64 and can send from any address in
it, so all of them count as one client address.

### The life of a connection

1. **Upgrade.** The client opens the socket.
2. **Hello.** The client's first frame must be `hello`, within 10 seconds.
   The server derives the user id from the token and answers `welcome`. A
   wrong first frame ends the connection:
   - no frame within 10 seconds: closed with 1008 `hello timeout`;
   - a `hello` whose `v` is not the server's protocol version: an `error`
     frame with code `protocol`, then 1008 `protocol`;
   - anything else (not a `hello`, malformed JSON, a token of the wrong
     shape): an `error` frame with code `bad-hello`, then 1008 `bad-hello`.
3. **Requests.** After `welcome`, the client sends `create`, `join`, `act`,
   `undo`, `hints`, `link`, `relink`, `resync`, `leave` and `rematch`. The
   server handles one connection's requests one at a time, in the order they
   arrive, and every outcome comes back as frames (`created`, `state`,
   `linked`, `reject`, `error`). A
   refused request leaves the connection open, whatever the frame says. A
   connection is in at most one room at a time: joining another room first
   leaves the current one. A second `hello` is answered with `reject
   bad-request` and changes nothing.
4. **The end.** Either side closes the socket. The server then detaches the
   connection from its room. The seat stays with its user, who gets it back
   on the next `join`, and the other connections receive a `state` with the
   new presence.

### Close codes

These are the close statuses the server sends, with the reason text where
there is one:

- 1000 (normal closure): the answer to the client's own close.
- 1003 (unsupported data): a binary frame.
- 1008 (policy violation), with one of these reasons:
  - `hello timeout`: no `hello` within 10 seconds;
  - `protocol`: an unsupported protocol version (after an `error` frame);
  - `bad-hello`: an invalid first frame (after an `error` frame);
  - `rate limit`: more than 20 frames in a row were refused by the frame
    rate limit;
  - `client too slow`: the connection let 32 frames other than `state`, or
    the `state` frames of 4 rooms, pile up unread.
- 1009 (message too big): a frame over 8192 bytes.
- 1012 (service restart): the server is shutting down or restarting.

When the server gives up on a dead connection (two pings in a row
unanswered) or a write fails or takes longer than 10 seconds, it drops the
socket without a closing handshake, so the client sees 1006 (abnormal
closure).

A client reconnects with backoff and jitter after any close it did not ask
for, except after `protocol` and `bad-hello`, which another try cannot fix:
`protocol` means the server runs a newer build (reload the page) and
`bad-hello` means the client itself is wrong.

### Pings

The server pings every connection every 25 seconds. Browsers answer pings on
their own, so client code does nothing, and a client never needs to ping the
server. Any pong clears the count of unanswered pings. At a tick that finds
the last two pings unanswered, the server drops the connection, so a peer
that vanished without closing is gone within about 75 seconds.

### The outbox

Each connection has an **outbox**: a queue the rooms write the
connection's frames into and a writer goroutine drains to the socket. A room
never waits for a client. Because a `state` frame is a complete snapshot,
the outbox keeps only the newest pending `state` of each room (latest wins):
a new one replaces the one still waiting, moves to the end of the queue,
inherits the old one's `ack` when it has none of its own, and puts the old
one's events in front of its own unless an undo came in between (the events
of an undone move describe a position that no longer exists, so the client
simply jumps to the new one). A `history` changes the merge: a new frame
with a history drops the old one's events (its history already holds them),
and a new live frame that replaces one with a history inherits that history
and appends its own events after the old one's, so the client still gets the
whole game. A slow client therefore skips intermediate
snapshots, but never loses an `ack` and still receives the events that lead
to the newest snapshot (up to 256 of them). Other
frames (`created`, `reject`, `error`) queue in order, at most 32 of them, and
the pending `state` frames may belong to at most 4 rooms (a connection is in
one room at a time; a few cover a switch from one room to the next). A
connection that lets more pile up is closed with 1008 `client too slow`.

## Client frames

Every client frame is checked strictly. After `hello`, a frame with an
unknown `"t"`, a field the frame does not declare, data after the JSON
value, or a field out of bounds (see the limits below) is answered with
`reject bad-request` (carrying the frame's `seq` when it has one), and
nothing else happens. Room codes are the six-character codes the server hands out, such
as `K7QX2M`.

### hello

The first frame of every connection.

```json
{"t": "hello", "v": 1, "token": "Vf3kQ9xZr2LmT8wB1nYc0A", "name": "Sara", "clientId": "4f1c9a2e7b5d"}
```

- `v` is the protocol version the client was built for, currently 1. A
  `hello` without `v` counts as version 0.
- `token` is the user's identity token: 16 random bytes as 22 characters of
  unpadded base64url, made once by the browser and kept in `localStorage`.
  The server derives the user id from it as an HMAC with the server's
  secret, so the token itself is never stored, logged or sent back. The same
  token on two devices is the same user.
- `name` is the display name to use for a seat the user takes, up to 64
  characters with no control characters and no invisible formatting
  characters (see the limits below). Empty means a default.
- `clientId` names this tab, for idempotency (see below): 1 to 128
  characters, under the same character rules as `name`. Make it random
  (`crypto.randomUUID()`), keep it for the life of the tab together with the
  tab's `seq` counter, and give every tab its own.

The answer is `welcome`, or an `error` and the end of the connection.

### create

Creates a room and joins the sender to it.

```json
{"t": "create", "mode": "ai", "bot": "normal", "seat": "random", "names": ["Sara"]}
```

- `mode` is `online` (two devices), `hotseat` (one device, both seats) or
  `ai` (against a bot).
- `bot` is the bot's level, `easy`, `normal` or `hard`. It is required in an
  `ai` room and refused in any other.
- `seat` is the creator's seat in an `online` or `ai` room: `0`, `1`, or
  `"random"` to let the server pick. Absent means 0. A hot-seat room ignores
  it.
- `names` holds at most two display names, by seat. An online room uses
  the entry of the creator's seat, an `ai` room the first entry, and a
  hot-seat room both. An absent or empty entry falls back to the `hello`
  name (or a default for a hot-seat room's second seat).
- `hints` is optional: the game's hint settings (see `hints` below), an
  object with all twelve families as booleans. Absent means the defaults
  of the mode: every family on, except `advisor`, which is on only in an
  `ai` room. A `hints` object that lacks a family, or holds `null` for one,
  is refused as `bad-request`.
- `allowUndo` is optional, for a hot-seat or `ai` room: `false` makes a
  game without undo. Absent means `true`. It is fixed when the room is
  created: nothing changes it afterwards, a rematch keeps it, and a room
  stored before the field existed allows undo. An online room has no undo,
  so a `create` for one that names `allowUndo` at all, `true` or `false`,
  is refused as `bad-request`.

The creator becomes the game's **admin**, the one user who may change its
hint settings. The admin and the settings are stored with the room, so they
survive a reload and a server restart, and a rematch keeps both.

The answer is `created` with the room's code, then the room's first
`state`. Over the room creation limit the answer is `reject rate` with seq
0; when the server holds as many rooms as it can, `error busy`; settings the
room refuses get `error forbidden`.

### join

Attaches the connection to a room, as a player or as a spectator.

```json
{"t": "join", "code": "K7QX2M"}
```

A user who holds a seat gets it back. In an online room a new user takes the
open seat (the room goes from `lobby` to `playing`), and once both seats are
taken later users watch as spectators, with a public view and no choices.
Hot-seat and `ai` rooms admit only their creator. A room that is not in
memory is loaded from disk first. Joining a room the connection is already
in just sends a fresh `state`, unless the join's `claim` differs from the
one it joined with (a device link opened over the room's page): the
connection is then detached and admitted afresh under the new claim, and
stays detached if that fails. In an online room, a user who comes back to
their seat under another display name (the `name` of their `hello`) renames
the seat, and only that seat's name number is recomputed (see `nameNo`
under `state`).

A join may carry a seat's **claim**, from that seat's device link (see
[Device links](#device-links)):

```json
{"t": "join", "code": "K7QX2M", "claim": "q3Vb8Kx0Zr7TnW2mYc5JhA"}
```

With a valid claim the connection acts in this room as the user of the seat
the claim was made for, not as its own user: it gets that seat back, both
seats of a hot-seat room, the human seat of an `ai` room, and the admin's
rights when that user is the admin. Its own identity is not changed and is
not used in this room, and a claim never renames the seat. A claim that is
malformed, made for another room, or revoked by a `relink` is refused with
`error bad-claim`, and counts against the address's joins of unknown rooms
like a code that names no room. A join without `claim` behaves as before.

The answer is a `state`, and every other connection of the room receives a
`state` with the new presence. The refusals are `error room-not-found`,
`error room-full`, `error forbidden` (another user's hot-seat or `ai` room;
the server tells this from the room file's header, without loading the
room) and `error busy` (the room must be loaded and every room in memory has
a connection). A client address whose joins have named too many rooms that
do not exist (30, earning one back every 2 minutes) gets `reject rate` with
seq 0 for every `join`. That bounds how fast one address can guess codes,
not how fast anyone can: see the joins of unknown rooms under
[Limits](#limits) for what is left.

### act

Submits one game action.

```json
{"t": "act", "code": "K7QX2M", "seq": 17, "base": 42, "key": "construct:baths"}
```

- `seq` is the client's request counter (see idempotency below).
- `base` is the `version` of the `state` the action was chosen in.
- `key` is the action's key, copied from a `choices` entry of that state,
  such as `construct:baths`, `wonder:pyramids:baths` or `token:law`.
- `seat` is optional, 0 or 1. When present it must be the seat to move, or
  the act is refused with `not-your-turn`. A hot-seat client sends it, so a
  tap left over on the previous player's screen cannot act for the next
  one.

**Forfeit.** The key `resign:0` or `resign:1` forfeits that seat's game. It
is never among `choices` (so no bot ever sends it) and travels in an
ordinary `act`. The room accepts it only from a player for their own seat,
whoever is to move: in an online room either player, in an `ai` room the
human, and in a hot-seat room the seat to move (with `seat`, if present,
equal to it). A spectator, a player naming the other seat, or a hot-seat
forfeit of the waiting seat gets `not-your-turn`; a forfeit in the lobby
gets `not-your-turn` too, and one after the end `illegal`. `base` is checked
as for any act, so a forfeit sent while a bot answered is refused `stale`
and may be sent again. An accepted forfeit cancels a bot search, ends the
game with the result `{"victory": "forfeit", "winner": <other seat>}`, emits
a `resign` event (its `seat` is the seat that forfeited) before `gameOver`,
reveals the hidden cards like any other end, and cannot be undone. An open
Great Library offer goes back to the box unseen. The ruling is in
[docs/rules/decisions.md](rules/decisions.md).

The room checks, in order: the connection has joined and is not a spectator;
the `seq` is new (a repeated one only gets the current `state` with an
`ack`); the game is running; the sender holds the seat to move; `base` is the
current version; the key is among the legal actions. Then it applies the
engine's own action for that key. The sender receives a `state` with an
`ack` for the `seq`, and every other connection a `state` without one. A
refusal is a `reject` with the request's `seq`: `bad-request` (not joined),
`not-your-turn`, `illegal`, or `stale` followed at once by a fresh `state`.

### undo

Takes back the last human decision, in a hot-seat or `ai` room.

```json
{"t": "undo", "code": "H8TQ4R", "seq": 18, "base": 43}
```

`seq` and `base` work as in `act`. In an `ai` room the undo also removes the
bot's answers after that decision and cancels a bot search under way.
Because the room replays the shortened log from the seed, hidden cards
cannot change. The `canUndo` field of the latest `state` says whether an
undo would be accepted now. The answer is a `state` with an `ack`, or a
`reject`: `bad-request` (not joined, an online room, a spectator),
`undo-off` (the game was created with `allowUndo` false), `illegal` (the
game is over, or there is nothing to undo), `stale`, or
`rate` (the client address sent more than its undo limit, 60 at once and
then one a second; the `seq` is not used up).

### resync

Asks for a fresh `state` of the joined room, without an `ack`.

```json
{"t": "resync", "code": "K7QX2M"}
```

A client sends it when it suspects it missed something. A room it has not
joined answers `reject bad-request` with seq 0.

### leave

Detaches the connection from the room.

```json
{"t": "leave", "code": "K7QX2M"}
```

There is no answer. The seat stays with its user, and the other connections
receive a `state` with the new presence. A `leave` for a room the connection
is not in does nothing. Closing the socket has the same effect.

### rematch

Asks for a new game with the same mode and seats, once the game is over.

```json
{"t": "rematch", "code": "K7QX2M"}
```

The room creates one new room with a fresh deal, stores its code, and sends
every connection of the old room a `created` frame with that code. Each
client then sends `join` with the new code. However many players ask, and
even after the old room was reloaded from disk, every rematch request of a
room answers the same code. The refusals are `reject bad-request` (not
joined, or the game is not over), `error forbidden` (a spectator), `error
busy`, and `reject rate` with seq 0: the request that creates the rematch
room counts against the requester's room creation limits, like a `create`.
A request answered with a rematch code that already exists counts nothing.
A connection that joined the old room with a claim receives, in its
`created` frame, the claim of its seat in the rematch room, since its own
identity holds no seat there.

### hints

Replaces the game's hint settings. Only the game's admin may send it.

```json
{"t": "hints", "code": "K7QX2M", "seq": 19, "hints": {"prices": true, "economy": true, "science": true, "military": true, "threats": false, "tempo": true, "wonders": true, "tokens": true, "scoring": true, "counting": true, "recap": true, "advisor": false}}
```

- `seq` is the client's request counter, shared with `act` and `undo` (see
  idempotency below). There is no `base`: the settings are not part of the
  game, so a change never goes stale and does not change the room version.
- `hints` is the whole new settings object, every family present. The
  families are `prices`, `economy`, `science`, `military`, `threats`,
  `tempo`, `wonders`, `tokens`, `scoring`, `counting`, `recap` and
  `advisor`. They are the same for both players: they say what the game's
  hints may show, and each client draws its hints from them. A player may
  hide more on their own screen; that choice stays on the device and never
  reaches the server.

The room checks, in order: the connection has joined (else `bad-request`)
and is not a spectator (else `not-admin`); the `seq` is new (a repeated one
only gets the current `state` with an `ack`); the sender is the admin (else
`not-admin`); the game is not over (else `game-over`). Then it stores the
settings, adds an entry to the room log (see `roomLog` under `state`), and
sends every connection a `state`, the sender's with an `ack`. The admin may
change the settings in the lobby too, before the second player arrives.

### link

Asks for one seat's device link (see [Device links](#device-links)).

```json
{"t": "link", "code": "K7QX2M", "seat": 1}
```

The room checks, in order: the connection has joined (else `reject
bad-request`) and is not a spectator (else `reject not-admin`); device
links are on for this server (else `reject bad-request`); a human holds the
seat (else `reject bad-request`: the seat is open or a bot's); the
connection acts as that seat's user or as the game's admin (else `reject
not-admin`). All these rejects carry seq 0. The answer is a `linked` frame,
to this connection alone. Nothing else changes: no other connection hears of
it, and the room version stays.

### relink

Resets one seat's device link, under the same checks as `link`.

```json
{"t": "relink", "code": "K7QX2M", "seat": 1}
```

The room raises the seat's **epoch** (in a hot-seat room, the epoch of both
seats, which belong to one user), so every claim made for it before stops
working, and adds an entry to the room log. The sender receives the new
claim in a `linked` frame; every connection receives a `state` with the
log entry. A connection that joined with a revoked claim receives `error
bad-claim` and is detached from the room, like one that left: its next
request is refused as not joined. The sender keeps its place even when it
joined with the claim it revoked. The room version does not change.

## Server frames

### welcome

Answers `hello`.

```json
{"t": "welcome", "v": 1, "server": {"version": "dev", "rules": 1}}
```

- `v` is the protocol version the server speaks.
- `server.version` is the build version (from `git describe`; `dev` for a
  development build).
- `server.rules` is the engine's rules version, the same number the game
  log records.

### created

Gives the code of a new room: the answer to `create` (a `state` follows), or
the rematch room of a finished game (the client sends `join` with it).

```json
{"t": "created", "code": "K7QX2M"}
```

`claim`, present only in the rematch frame of a connection that joined the
old room with a claim, is the claim of that connection's seat in the
rematch room; the client joins the rematch room with it.

### linked

Answers `link` and `relink`: one seat's current claim. It goes only to the
connection that asked, and never appears in a `state`.

```json
{"t": "linked", "code": "K7QX2M", "seat": 1, "claim": "q3Vb8Kx0Zr7TnW2mYc5JhA"}
```

The client builds the device link from it: `/play/<code>#claim=<claim>`.

### state

A complete snapshot of one room, built for the connection that receives it.
The room sends one after every change: an applied action (a player's or a
bot's), an undo, a join, a leave, a seat taken, and in answer to a `resync`,
a repeated `seq` or a `stale` refusal. This example from a hot-seat room is
shortened: the real `view` has many more fields and choices, and is the
full game view described under "The view and its events" below.

```json
{
  "t": "state",
  "code": "H8TQ4R",
  "version": 10,
  "room": {
    "mode": "hotseat",
    "status": "playing",
    "seats": [
      {"seat": 0, "name": "Sara", "nameNo": 0, "kind": "human", "connected": true, "you": true},
      {"seat": 1, "name": "Omid", "nameNo": 0, "kind": "human", "connected": true, "you": true}
    ],
    "spectators": 0,
    "hints": {"prices": true, "economy": true, "science": true, "military": true, "threats": true, "tempo": true, "wonders": true, "tokens": true, "scoring": true, "counting": true, "recap": true, "advisor": false},
    "allowUndo": true
  },
  "you": {"seats": [0, 1], "canAct": true, "admin": true},
  "canUndo": true,
  "ack": {"clientId": "4f1c9a2e7b5d", "seq": 10},
  "events": [
    {"kind": "build", "private": false, "seat": 1, "card": "clay-pool"},
    {"kind": "reveal", "private": false, "card": "stone-reserve", "slot": 10}
  ],
  "roomLog": [],
  "view": {
    "schema": 1,
    "ply": 10,
    "phase": "play",
    "toMove": 0,
    "choices": [
      {"key": "discard:stone-reserve", "kind": "discard", "card": "stone-reserve", "gain": 2}
    ]
  }
}
```

- `code` is the room, and `version` the room's version (see below); an
  `act` or `undo` chosen in this state sends it as `base`.
- `room` describes the room rather than the game: `mode`; `status`, which is
  `lobby` (an online room waiting for its second player), `playing` or
  `finished`; one entry per seat with its display name, its name number
  (`nameNo`, below), its `kind` (`human`, `bot`, or `open` for a seat
  nobody has taken yet), the bot's level as `bot` for a bot seat, whether
  someone is `connected` there (a bot always is), and whether this
  connection holds it (`you`); the number of `spectators`; the game's hint
  settings, `hints` (see the `hints` frame); and `allowUndo`, whether the
  game allows undo at all (fixed at creation; always `false` in an online
  room). Every viewer sees the same value.
- `nameNo` tells equal display names apart. It is 0 for a name that needs
  no number. When a human takes a seat, or renames it, under a name equal
  to the other human seat's (compared without case and outer spaces), the
  other seat came first and keeps its number, and this seat takes the next
  one: 2 after an unnumbered name. Only the seat that took or changed its
  name is renumbered, so a number stays when the other player leaves or
  renames. A hot-seat room created with two equal names numbers seat 1.
  The client shows the name as "Sara 2" (through its message tables), and
  shows the name alone for 0.
- `you.seats` are the seats this connection may act for (both in a hot-seat
  room, one for a player, none for a spectator), and `you.canAct` says
  whether one of them has a legal action now. `you.admin` says whether this
  connection's user is the game's admin, who may send `hints`; it is never
  true for a spectator.
- `canUndo` says whether an `undo` with this frame's version as its base
  would be accepted now. It is never true when `room.allowUndo` is false.
  `undoUnavailable` is present exactly when `canUndo` is false, with the
  first applicable code in this precedence: `online` (online room),
  `disabled` (undo explicitly off), `spectator` (no held seat), `game-over`,
  `no-human-decision` (no human decision remains to take back). The last
  includes an untouched game and a bot-first game before the human acts.
  Every frame path carries this pair, including presence, join, resync and
  undo. An available undo omits the reason. The client translates the code;
  local offline and pending-request explanations take precedence while the
  transport prevents sending.
- Every affordable cost breakdown carries `coinsAfter`, the payer's purse
  immediately after subtracting `total`, before Urbanism or any later
  building/wonder effect. This includes free, chain-free, traded and wonder
  payments, both seats' face-up previews and available wonder costs. Economy
  changes the recipient, not this purse. Unaffordable previews omit the
  field and retain `shortfall`; no impossible purse or clamped deficit is
  reported. Every legal choice with a cost carries it.
- `thinking`, present only while a bot is searching, is the bot's seat.
- `ack`, present only in the sender's copy of the direct answer to an `act`
  or `undo`, names that request's client id and `seq`.
- `events` are the events of what changed since the client's previous
  `state`, already filtered for this viewer; often empty.
- `history`, present only in the first frame a connection gets after a
  `create` or `join` (a repeated join included), in every answer to a
  `resync`, and in every connection's frame after an undo, is the whole game
  so far: one entry per applied action, oldest first, each with the `ply`
  the action was taken at, the `seat` that took it (the seat to move, or the
  seat that forfeited) and its `events`, filtered for this viewer exactly
  like live events. A room loaded from disk rebuilds it by replay. Live
  frames (moves, presence, a `stale` or repeated-`seq` answer) do not carry
  it. In a frame with a history, `events` are empty, or, after an outbox
  merge, the events that follow the history's last entry. The client
  replaces its log with it, and must not animate or toast it. It holds at
  most 512 entries (`wire.MaxHistory`, the newest kept); a game takes at most
  `engine.MaxGameActions` (189) actions, so a real game never reaches the cap.
- `roomLog` is every room-level entry of the game log so far, oldest first:
  things that happened to the room rather than in the game: a change of the
  hint settings (`kind` `hints`) or a reset device link (`kind` `link`). Every frame carries the whole list,
  the same for every viewer, and the room keeps the newest 64
  (`wire.MaxRoomLog`). An entry is
  `{"kind": "hints", "at": 4, "seat": 0, "on": ["advisor"], "off": ["prices"]}`:
  `at` is the ply the change was made at, so the client shows it before the
  history entry of that ply; `seat` is the admin's seat, absent in a
  hot-seat room, where the admin holds both; `on` and `off` list the
  families switched, in family order. Changes made by the same seat at the
  same ply fold into one entry, and an entry whose changes cancel out is
  dropped. After an undo, entries past the new ply move back to it. The
  room log is stored with the room, but it is not part of the game log
  (`ajayeb-log/1`), which records only the game. A reset device link is
  `{"kind": "link", "at": 4, "seat": 0, "on": [], "off": [], "target": 1}`:
  `seat` is the seat of whoever reset it (absent in a hot-seat room),
  `target` the seat whose link was reset, and `on` and `off` are empty.
  Link entries never fold together.
- `view` is the game as this viewer may see it, including its `choices`.
- `view.review`, present only while the game's `advisor` family is on and
  only in a seat's own view, is the move review of that seat's last play
  action (a build, a discard or a wonder): the action's `ply` and key
  (`played`), and, when a clearly better action existed, its key
  (`better`), a `reason` code (`wins`, `stopsWin`, `pair`, `deniesPair`,
  `loot` or `points`; `none` without `better`), the rounded `gap` in
  points, and the reason's detail (`threat`, `symbol` or `coins`). The
  server ranks the actions on that seat's own observation of the position
  before the move, under a fixed evaluation budget, so the review is the
  same for every hidden deal the seat could not tell apart and never names
  a card the move turned over. It stays until the seat's next play action,
  goes with an undo, and is absent from the opponent's and a spectator's
  view. The client shows it inside the summary of the opponent's next move.

### reject

Refuses one request. The connection stays open.

```json
{"t": "reject", "seq": 17, "code": "stale", "msg": "base is not the current version"}
```

`seq` is the refused request's `seq`, or 0 when the request has none (a
`create`, a `resync`, a malformed frame without one). `code` is one of the
reject codes below. `msg` is a hint for developers, never text for a player:
the client maps `code` to a translated message.

### error

Reports a problem with a room request or with the connection itself.

```json
{"t": "error", "code": "room-not-found", "msg": "no room with that code"}
```

Only `bad-hello` and `protocol` end the connection; after any other error
the client may go on (join another room, try again later). `msg` is a
developer hint, as in `reject`.

## Reject codes

A reject refuses one request, and the client can usually recover.

- `stale`: the request's `base` is not the room's current version, so it was
  chosen in a position that has since changed. A fresh `state` follows at
  once. The player chooses again in the new position.
- `illegal`: the key is not among the legal actions now; also an `act` or
  `undo` after the game is over, and an `undo` with nothing to undo.
- `not-your-turn`: the sender does not hold the seat to move (or the act's
  `seat` is not the seat to move); also an act from a spectator, and an act
  in an online room still waiting for its second player.
- `rate`: the frame came faster than the frame rate limit allows and was
  dropped unhandled (its `seq` is echoed when it has one); a `create`, or a
  `rematch` that would create a room, went over the room creation limit
  (seq 0); a `join` came from a client address over its limit of joins that
  named no room (seq 0); or an `undo` went over the undo limit (its `seq`).
- `bad-request`: the frame is malformed (a `create` for an online room
  with `allowUndo` counts); a second `hello`; a request for a
  room this connection has not joined; an `undo` in an online room or from a
  spectator; a `rematch` before the game is over; a `link` or `relink` for a
  seat that is open or a bot's, or on a server whose device links are off.
- `not-admin`: a `hints` frame from a player who is not the game's admin, or
  from a spectator; a `link` or `relink` from a spectator, or from a player
  who is neither that seat's user nor the game's admin.
- `game-over`: a `hints` frame after the game has ended; the settings of a
  finished game no longer change.
- `undo-off`: an `undo` in a hot-seat or `ai` room created with `allowUndo`
  false.

A refused `act`, `undo` or `hints` does not use up its `seq`, so the client may send
it again with the same `seq`.

## Error codes

- `room-not-found`: no room has that code, the room is archived, or it could
  not be loaded.
- `room-full`: the room takes no more connections of this kind: a new
  spectator once the room has 32 connections, or one more connection of a
  user who already has 32 there. Spectators can never lock a player out.
- `forbidden`: this user may not do that in this room: join another user's
  hot-seat or `ai` room, ask for a rematch as a spectator, or create a room
  with settings the server refuses.
- `bad-hello`: the first frame was not a valid `hello`. The connection
  closes (1008).
- `protocol`: the `hello` asked for a protocol version the server does not
  speak. The connection closes (1008).
- `busy`: the server holds as many rooms in memory as it can, and every one
  of them has a connection (a `create`, a `rematch`, or a `join` that must
  load a room). Try again later.
- `bad-claim`: a `join` carried a claim that is not one of this room's
  (malformed, made for another room, or revoked by a `relink`); or, sent on
  its own, the device link this connection joined with was just reset, and
  the connection has left the room. A join without the claim, or with a
  fresh link, may follow.

## Versions, idempotency and reconnecting

### The room version

Every room has a **version**, an integer that goes up by one with every
applied action (a player's or a bot's) and every undo, and never goes down:
it is stored in the room file, so it survives an undo, an eviction from
memory and a server restart. One version always names one position. Every
`state` carries it, and an `act` or `undo` names the version it was chosen
in as `base`. A request whose `base` is not the current version is refused
as `stale`, which is what keeps a player from acting on a position that has
changed under them.

### Idempotency

Each `act`, `undo` and `hints` carries a `seq`. The rules:

- `seq` is a counter per client id: start at 1 and add one for every `act`,
  `undo` and `hints` the tab sends, in any room.
- The room remembers, for each user and client id, the last `seq` it
  applied. A request whose `seq` is not newer than that is not applied
  again: the sender only receives the current `state` with an `ack` for that
  `seq`. A double tap, a frame sent twice, or a request re-sent after a
  reconnect therefore takes effect at most once. Keying by the user as well
  means another user who picks the same client id cannot make the room take
  this tab's new requests for old ones.
- Only an applied request uses up its `seq`. After a refusal the client may
  send the same `seq` again, and the room judges it afresh.
- The `seq` check comes before the turn check, so a request re-sent after
  the turn has passed resolves with an `ack` instead of a refusal.
- A room remembers the last `seq` of 32 clients, in memory and in its
  file. For a client id it has forgotten, `base` is the second guard: the
  first copy of a request moved the version past the copy's `base`, so a
  late duplicate is refused as `stale`.
- Never reuse a client id with a counter that started over, or the room
  takes the new requests for old ones and only acknowledges them. Keep the
  client id and the counter together (for example both in the tab's
  `sessionStorage`), and make a new client id whenever the counter is lost.

A request is **pending** from the moment it is sent until a frame resolves
it: a `state` whose `ack` names its `seq`, or a `reject` with its `seq`.

### Reconnecting

After the socket drops, the client:

1. opens a new socket, with exponential backoff and jitter;
2. sends `hello` with the same token and the same client id;
3. sends `join` with the room's code, and receives a fresh `state`; the user
   is back in their seat;
4. settles a pending request. Re-sending it unchanged (same `seq`, same
   `base`) is always safe: if the room applied it before the drop, the
   answer is an `ack`; if not, it is judged afresh, and a position that has
   moved on answers `stale` with a fresh `state`. A client may instead skip
   the re-send when the fresh state's version is already past the pending
   request's `base`, since the request was then either applied or overtaken.

A `rematch` asked again after a reconnect answers the same code as before,
so a client that missed the `created` frame just asks again. After a 1012
close the server is restarting; the rooms are on disk, so no game is lost.
A client that joined with a claim sends the same claim again on every
reconnect.

## Device links

A **device link** moves one player to another device without moving their
identity token. It is the room's invite link with the seat's **claim** in
the fragment: `/play/<code>#claim=<claim>`. The claim is the first 22
characters (132 bits) of the unpadded base64url form of
HMAC-SHA256(server secret, `claim:<code>:<user id>:<epoch>`), where the user
id is the seat's user and the epoch a per-seat counter stored with the room
(0 at first). Package `ident` derives and checks it (`ident/doc.go`); the
check is constant-time.

- **Room-scoped.** The code is in the MAC, so a claim works in its own room
  only, and a rematch room gets fresh claims.
- **Not the token.** The claim reveals neither the token nor the user id,
  and the server derives it again on demand instead of storing it.
- **Kept in the address.** The fragment never reaches the server in an
  HTTP request or a log, and it stays in the address bar, so a reload keeps
  the seat. The client also remembers the claim with the room in
  `ajayeb:rooms`, and opening a device link never asks for a display name.
- **Revocable.** `relink` raises the epoch, which revokes every claim made
  before it and detaches the connections using one.

**Security.** A device link is a password for that seat in that room:
whoever holds it plays as that player there (as the admin too, for the
admin's seat), until the seat's link is reset. It gives nothing outside the
room. Share it like a password, and reset it when it may have leaked. Only
the seat's own user and the game's admin may see or reset a seat's link.

## Limits

These are the production values. Most are fields of `ws.Limits` or
`room.Config`, whose defaults these are, and tests set them through a fake
clock instead of waiting; the bot pool's are server flags, and the room
file cap is a constant of the store.

- **Frame size:** 8192 bytes per client frame; a bigger one closes the
  socket with 1009. Inside a frame: `name` up to 64 characters, `clientId`
  up to 128, a room code up to 32, an action key up to 96, at most two
  `names`, a token of exactly 22 characters, and `seq` and `base` not
  negative.
- **Characters:** no string in a client frame may hold a control character
  or an invisible formatting character (Unicode category Cf): no bidi
  override, embedding, isolate or mark, which would reorder the text around
  a name wherever it is shown, and no zero width space or byte order mark,
  which would let a name pass for another. The zero width non-joiner and
  joiner (Persian and other scripts, emoji sequences) and the tag characters
  of emoji flags are allowed.
- **Frame rate:** per connection, a token bucket that holds 20 frames and
  refills at 10 per second. Every frame counts, `hello` included. A frame
  over it gets `reject rate` and is not handled; more than 20 such frames in
  a row close the socket with 1008 `rate limit`.
- **Hello:** within 10 seconds of the upgrade.
- **Sockets:** 16 per client address, an IPv4 address or an IPv6 /64 (429
  beyond), and 1024 per server (503 beyond).
- **Room creation:** 20 per hour per user id and 60 per hour per client
  address. Each is a token bucket whose burst is its hourly count, so a user
  may create 20 rooms at once and then one every 3 minutes. A refused
  `create` uses up neither bucket. The limiters remember at most 4096 users
  and 4096 addresses: buckets that have refilled are forgotten first, and
  when every remembered bucket is still refilling, a new user or address is
  refused too, rather than letting a flood push out the limits of others.
  A `rematch` that creates the rematch room counts as a create.
- **Joins of unknown rooms:** 30 per hour per client address (an IPv4
  address or an IPv6 /64), as a token bucket of 30. A `join` that finds no
  room (no such code, a malformed one, an archived room) uses one up; past
  the limit every `join` from the address gets `reject rate` (seq 0). A
  `join` refused with `bad-claim` uses one up too, so claims cannot be
  guessed faster than codes. An address the limiter does not remember is
  never refused.
  This stops one address from guessing codes at the frame rate, but it
  does not make codes unguessable. Someone who holds many /64s (a free
  tunnel-broker IPv6 /48 holds 65,536 of them) gets 30 misses an hour from
  each, and once the limiter remembers 4096 addresses whose buckets are all
  still refilling, the misses of any address it does not remember go
  uncounted, so they are not limited at all. A room code is 6 characters
  from a 31-character alphabet, about 8.9e8 codes, so such an attacker can
  find stored rooms, watch games in progress, and take the open seat of an
  online lobby before the invited friend does. A device link cannot be
  guessed this way: its claim is an HMAC under the server's secret, and a
  wrong one counts as a miss. Room codes stay 6 characters for now; a
  longer code for online rooms, or an invite secret kept apart from the
  code and required to claim an open seat, is left to a later decision.
- **Undo:** 60 per minute per client address, as a token bucket of 60; an
  `undo` over it gets `reject rate` with its `seq`. Every applied undo
  replays and rewrites the room file.
- **Liveness:** a ping every 25 seconds; dropped at the tick after two pings
  in a row went unanswered.
- **Writes:** each frame must be written within 10 seconds, or the socket is
  dropped.
- **Outbox:** at most 32 pending frames other than `state` and the pending
  `state` frames of at most 4 rooms (1008 `client too slow` beyond); a
  merged `state` keeps at most the 256 newest events.
- **History:** a `state` frame's `history` holds at most 512 entries
  (`wire.MaxHistory`), more than the 189 actions a game can take
  (`engine.MaxGameActions`, checked by a test).
- **Rooms:** at most 1024 rooms in memory. When a create or a load finds
  them all there, the room with no connection that has been idle longest
  leaves memory to make room (it is on disk, and comes back on its next
  `join`); only when every room in memory has a connection is the answer
  `error busy`. A room admits a new spectator only while it has fewer than
  32 connections, and a seated user only while that user has fewer than 32
  connections there (`error room-full`); each room's inbox holds 64
  messages; a room with no connection for 30 minutes leaves memory and is
  reloaded from disk on the next `join`. A bot searches only while its
  room has a connection: the last connection to leave cancels the search,
  and the next one to join starts it again.
- **Bot searches:** at most 2 run at once across the server
  (`-bot-searches`), and one client address (an IPv4 address or an IPv6
  /64, the key of the limits above) holds at most 2 of them at once
  (`-bot-searches-per-address`). A bot room's search counts under the
  address of the person playing it. Further searches wait without holding
  a slot, and a search whose room is left, evicted or undone leaves the
  queue at once. When a slot frees up it goes to a waiting search whose
  address holds the fewest slots, and among those to the address that got
  one longest ago, so however many bot rooms one address runs, another
  address's bot waits for at most one search to end (`ai.Pool`). The room
  never waits for the pool itself: the search runs in its own goroutine.
- **Room files:** the store reads a room file of at most 704 KiB
  (`store.MaxFileBytes`), four times the most that a room file can hold
  (189 actions, the 64 entries of the room log, and the 32 remembered
  clients). A larger file, which this server never writes, is refused
  unread (`store.ErrTooLarge`): the room is not found, and its log
  download answers 500. The file stays on disk for a look.
- **The identity secret:** `<data>/secret.key` must be a regular file (a
  symbolic link is refused, not followed) of exactly 32 bytes. If its
  group or other users may read or write it, the server sets its mode to
  0600 and logs a warning, and refuses to start if it cannot.
- **Development endpoint bodies:** 64 KiB.

Known gaps, each left for a larger design change:

- **Room writes:** every applied act or undo replays the room and rewrites
  its file at once; writes are not coalesced. Rooms banked in advance,
  within the create limits, can therefore drive disk writes at the frame
  rate.
- **Loads of known codes:** a `join` of an existing room that is not in
  memory loads it from disk, at up to the frame rate. This costs CPU (a
  read of at most the room file cap and a replay of at most 189 actions)
  but no longer locks anyone out, because idle rooms are evicted to make
  space.
- **Room codes:** guessable from many addresses, as described under joins
  above.

## HTTP endpoints

Beside the socket, the server answers plain HTTP:

- `GET /health` answers `{"ok": true, "version": "dev"}`, with the build
  version.
- `/ws` is the WebSocket endpoint above.
- `GET /api/rooms/{code}/log` downloads the game log of a finished game
  (see "The game log"). The log holds the seed, which determines every
  hidden card, so it answers only once the game has ended, and never for
  a game whose `hideAtEnd` option keeps the cards hidden after the end.
  It is limited per client address (a bucket of 10 requests that refills
  one request every 2 minutes; a miss counts too) and answers 404 for an
  unknown room, 409 while the game runs, 403 when `hideAtEnd` applies,
  405 for another method and 429 over the limit.
- Any other path under `/api/` answers 404
  `{"ok": false, "code": "not-found", "msg": "no such API endpoint"}`.
- Every other path serves the web client: its built files, and
  `index.html` for client routes such as `/play/K7QX2M`.

Every response carries these security headers (`httpapi/security.go`):

- `X-Content-Type-Options: nosniff` and `Referrer-Policy: same-origin`.
- `X-Frame-Options: DENY`, and the CSP's `frame-ancestors 'none'`: no
  other page may frame the app.
- `Cross-Origin-Opener-Policy: same-origin` and
  `Cross-Origin-Resource-Policy: same-origin`.
- A `Permissions-Policy` that turns off the powerful features the app
  never uses: accelerometer, bluetooth, camera, display-capture,
  geolocation, gyroscope, hid, idle-detection, magnetometer, microphone,
  midi, payment, serial, usb and xr-spatial-tracking. It leaves the
  clipboard (device links are copied), full screen and autoplay alone.
  Vibration and Web Audio, which the app uses, have no feature in it.
- A `Content-Security-Policy`:
  `default-src 'self'; script-src 'self' <hashes>; style-src 'self'
  'unsafe-inline'; img-src 'self' data: blob:; font-src 'self';
  connect-src 'self'; worker-src 'self'; manifest-src 'self';
  frame-ancestors 'none'; base-uri 'self'; form-action 'self';
  object-src 'none'`. `style-src` allows inline styles because Svelte's
  `style:` directives and the tooltip placement write style attributes.
  `connect-src 'self'` covers the same-origin `ws:` and `wss:` socket in
  current Chrome (checked by a production-mode browser probe).

The CSP is computed, never kept by hand. A response that is not an HTML
file has no `<hashes>`. An HTML file the server serves (the app shell
`index.html`, for every client route too) gets a `'sha256-...'` source for
each of its inline scripts: today the theme bootstrap of `web/src/app.html`
and SvelteKit's start script. The server finds them in the bytes it
serves, with a small scanner that skips scripts with `src` and those inside
comments, and hashes their text with line breaks normalized as browsers
do. HTML is hashed on each request and its policy cached by that digest,
so a `-static` rebuild gets a new policy even if it preserves file size
and timestamp. A new inline script needs no server change.

A `-dev` server sends the CSP in `Content-Security-Policy-Report-Only`
instead, so a Vite setup on another port is not blocked. Its report-only
policy permits same-origin framing, and `X-Frame-Options: SAMEORIGIN`
lets the test gallery frame fixtures. Production enforces the policy and
sends `X-Frame-Options: DENY`. End-to-end tests use development mode;
a separate production-mode Chrome probe must check enforced CSP, including
the same-origin WebSocket, after security changes.

The server sends no `Strict-Transport-Security`: one deployment serves
plain HTTP on purpose, so HSTS belongs to each deployment's TLS proxy
(`docs/self-hosting.md`).

Every JSON response also carries `Cache-Control: no-store`. An API error
always has the shape
`{"ok": false, "code": "...", "msg": "..."}`, where `code` is stable and
`msg` is a developer hint.

### Development endpoints

With `-dev-endpoints`, which the server refuses to start without `-dev`, two
more endpoints exist. A production server never enables them. They let
end-to-end tests and manual sessions open a room at a chosen position
instead of playing toward it. They answer only a client on the same machine:
a request from a peer that is not a loopback address, or one that a proxy
forwarded (it carries `X-Forwarded-For`, `Forwarded` or `X-Real-IP`), gets
403 `forbidden`. A room made from a chosen seed hides nothing from whoever
chose it, so a remote player must never be able to make one and invite
someone to it.

`POST /api/dev/rooms` creates a room. The body must be JSON, sent with
`Content-Type: application/json` (otherwise 415 `unsupported-media-type`),
so a page on another site cannot post it with a plain form.

```json
{"mode": "hotseat", "seed": "1", "scenario": "mausoleum"}
```

```json
{"mode": "ai", "bot": "easy", "seat": 1, "seed": "910002", "actions": ["draft:circus-maximus", "draft:pyramids"]}
```

- `mode` (required) is `online`, `hotseat` or `ai`.
- `seed` is the game's 64-bit seed as a decimal string, so a JavaScript
  caller keeps every bit (a JSON integer is accepted too). Absent means a
  random seed or, when the server was started with `-seed-base` (which it
  refuses without `-dev`, since such seeds make every deal predictable), the
  next seed of the sequence base, base+1, and so on.
- `scenario` names a named scenario (see `GET /api/dev/scenarios` and
  `docs/dev/scenarios.md`). The room opens at the scenario's position,
  before its trigger, so the test plays the trigger itself through the UI.
- `actions` are action keys applied in order, after the scenario when both
  are given, or else from the ordinary start; each must be legal when its
  turn comes.
- `bot` is the bot's level (`easy`, `normal` or `hard`): required in an `ai`
  room and refused elsewhere.
- `seat` is the human's seat in an `ai` room, 0 (the default) or 1; refused
  elsewhere.
- `options` are the game's options, as in the game log below
  (`firstPlayer`, `centerPawn`, `hideAtEnd`); absent means the defaults.

Unknown fields are refused. The room's seats are left open, except the bot's
seat in an `ai` room: the first user to `join` claims them as the mode
allows (both seats of a hot-seat room, the human seat of an `ai` room, seat
0 and then seat 1 of an online room). A test therefore opens the returned
`path` in a browser, whose client joins the room. The answer is 201:

```json
{"ok": true, "code": "K7QX2M", "mode": "hotseat", "seed": "1", "scenario": "mausoleum", "path": "/play/K7QX2M"}
```

The errors:

- 400 `bad-request`: a malformed body, an unknown mode, a bad seed, `bot`
  or `seat` where they do not belong, or a body over 64 KiB;
- 400 `unknown-scenario`: no scenario has that name (`msg` lists the names);
- 400 `invalid-room`: the position cannot be built, such as an action that
  is not legal when its turn comes;
- 403 `forbidden`: the client is not on the same machine (see above);
- 405 `method-not-allowed`: not a POST;
- 415 `unsupported-media-type`: the body is not declared as JSON;
- 503 `busy`: the server holds as many rooms as it can;
- 503 `shutting-down`: the server is shutting down.

`GET /api/dev/scenarios` lists the named scenarios, sorted (the list here
is shortened):

```json
{"ok": true, "scenarios": ["age-transition-chooser", "appian-economy", "circus-maximus"]}
```

## The view and its events

A game state is never sent as-is. `internal/engine.ViewFor(state, viewer)`
builds a `view.View`, a redacted snapshot for one viewer, and
`internal/engine.FilterEvents(events, viewer)` redacts the events an action
produced the same way. The viewer is seat 0, seat 1, or a spectator
(`engine.Spectator`). The DTO types themselves live in `internal/view`, whose
package comment (`server/internal/view/doc.go`) is the authoritative list of
what is public and what is hidden; this section summarises it and points at
the code that enforces it, `server/internal/engine/view.go` and
`server/internal/engine/eventfilter.go`.

Both functions build a whitelist: every field of the DTO is copied in
individually from the part of the state that viewer may see, so a field
nobody thought to redact cannot leak by accident. Hidden from every viewer
until the game is over: the cards in face-down slots (only their backs
show), the cards set aside from each age's deck (only their count), the
deals of later ages, the guilds and wonders left in the box, the second
draft round until the first one ends, and the progress tokens in the box
(only their count). At game over these appear under the view's `revealed`
section, unless the game's `hideAtEnd` option (see the game log below) keeps
them hidden even then. `choices` and `disabled`, the legal actions, are
filled in only for the seat to move; a spectator and the waiting seat always
get empty lists.

One secret is asymmetric: the three progress tokens the Great Library draws
from the box. While that choice is open, only the deciding seat's view lists
the drawn tokens, in `pending.tokens` and in matching `choices` entries;
every other viewer's view shows only how many were drawn, in
`pending.hiddenTokens`. The events agree: a `libraryDraw` event normally
reports the token drawn, but it is always treated as private regardless of
its own private flag, so every viewer but the drawer instead receives a
public summary (the drawer's seat and the token's place in the offer,
never the token itself). More generally, a private event reaches only the
seat it names; every other viewer simply does not receive it. Because the
state frame's `history` re-sends every event through the same filter, the
drawer's own history keeps naming the three tokens it drew, including the
two that went back to the box. That is not a leak: the drawer saw them. The
no-leak checks (`server/internal/protocoltest`, `hiddenFor`, and the e2e
frame checker in `e2e/fixtures/leak-check.ts`) allow a viewer exactly the
tokens it drew itself, and still treat every other box token as hidden.

**Costs on face-up cards.** Every face-up card in the structure, accessible
or still covered, carries both seats' cost breakdowns (`costs`), computed
as if it were accessible now, so the card sheet can say what a covered card
would cost. A face-down card carries none, since its identity is hidden.
This made a mid-age `view` about 2.6 to 3.5 KB (15 to 20 percent) larger:
measured on seeds 1 to 3 at a mid-age position of Age I and of Age II, the view went
from 15.6 to 19.3 KB before to 18.2 to 21.9 KB after (seed 1, Age II: 18150
bytes before, 21255 after). A join or resync frame of a finished 71-action
game, history included, is about 19.7 KB, against 7.9 KB for the same frame
without its history.

The protocol fixtures in `server/internal/engine/testdata/protocol/` are
golden JSON files, one per named position, each recording what all three
viewers (`seat0`, `seat1`, `spectator`) receive there: the full view and the
filtered events, side by side. `pick-library-token.json` and
`library-token-taken.json` show the Great Library asymmetry directly: the
drawer's `pending.tokens` lists three slugs while the other two viewers see
`pending.tokens` empty and `pending.hiddenTokens: 3`.
`game-over-civilian.json`, `game-over-science.json` and
`game-over-shared.json` show an ordinary end of game, where every viewer's
`revealed` section lists the hidden cards, guilds, wonders and tokens;
`game-over-military-hidden.json` sets `hideAtEnd`, and its views carry no
`revealed` section at all. These fixtures are written and checked by
`TestProtocolFixtures` in `server/internal/engine/view_test.go`
(`go test ./internal/engine/ -run TestProtocolFixtures -update` after a
deliberate change), and together they cover every phase, every kind of
choice and every kind of event at least once, so the client's own tests and
dev pages have a real example of each to read against.

## The game log

A game is fully described by its seed, its options, an optional starting
scenario and the keys of every action applied to it. The log format
`ajayeb-log/1` stores exactly that, nothing more, so a log is small and a
replay reproduces the game exactly:

    {
      "format": "ajayeb-log/1",
      "rules": 1,
      "seed": "1234567890123",
      "options": {
        "firstPlayer": "random",
        "centerPawn": "lastActiveChooses",
        "hideAtEnd": false
      },
      "scenario": null,
      "actions": [
        {"ply": 0, "seat": 0, "action": "draft:pyramids"},
        {"ply": 1, "seat": 1, "action": "draft:gardens"}
      ],
      "finalHash": "9f2c0e5a17b3d4c8"
    }

Field by field:

- `format` is always the string `ajayeb-log/1`. A reader checks it and
  refuses a log in a format it does not know, rather than guessing.
- `rules` is the engine's `RulesVersion`, an integer that goes up whenever a
  change to the rules or to setup could make an old log replay differently.
  A reader refuses a log whose `rules` does not match the engine it is
  running, instead of silently replaying it under different rules.
- `seed` is the 64-bit setup seed, written as a JSON string (not a number) so
  that a JavaScript reader keeps every bit of it; a double cannot hold a full
  64-bit integer exactly.
- `options` are the engine's `Options`: `firstPlayer` (how the first player
  of the game is chosen), `centerPawn` (the ruling used when the pawn starts
  an age centred) and `hideAtEnd` (whether removed cards, box guilds, box
  wonders and box tokens stay hidden once the game is over, instead of being
  revealed for review).
- `scenario` is `null` for an ordinary game. For a game that started from a
  named or ad hoc starting position, it is the Scenario object described
  below, and the log is replayed by building that scenario first instead of
  calling plain setup.
- `actions` is the list of applied actions in order. Each entry names the
  `ply` the game was at and the `seat` that was to move when the action was
  applied (a reader checks both before applying it, which catches a log that
  was edited or reordered by hand), and `action` is the action's canonical
  key, such as `construct:baths` or `wonder:pyramids:baths`. A forfeit is
  stored like any action, as `resign:0` or `resign:1`; its entry's `seat`
  is still the seat that was to move, which need not be the seat that
  forfeited.
- `finalHash` is optional: 16 lowercase hex digits of the FNV-1a hash the
  engine computes over the final state's fixed-size binary encoding. When
  present, a replay that reaches a different hash is reported as a mismatch
  rather than silently accepted, which catches a log doctored by hand or a
  rules change that was not meant to affect old games.

A store that persists a room, a tool that replays or verifies a game, and a
script that generates fixtures from a scripted game all read and write this
same shape, so none of them needs to agree with the others out of band.

### Downloading a game's log

`GET /api/rooms/{code}/log` answers the log of a room's game once that game
has ended, for the game menu's "Download replay" item. The answer is 200
with the log above as its body, `Content-Type: application/json` and
`Content-Disposition: attachment; filename="ajayeb-K7QX2M.json"`. The code
is matched without regard to case. The log of a room that started from a
named scenario carries that scenario as an object, so it replays anywhere.

The seed determines every card, wonder and token the game kept hidden, so
the log is served only when the end of the game has already shown all of
them to anyone watching the room: whoever knows the code may watch it, and
the code is all the request needs. The refusals:

- 404 `not-found`: no room has that code (or the code is malformed);
- 409 `game-running`: the game has not ended (a lobby or a running game);
- 403 `log-hidden`: the game's `hideAtEnd` option is on, so its secrets stay
  hidden after the end too, and a log would reveal them;
- 410 `archived`: the room file could not be trusted (see the store's
  archiving rule in `docs/architecture.md`);
- 405 `method-not-allowed`: not a GET or HEAD;
- 429 `rate`, with `Retry-After`: the client address (as for the socket's
  limits) has used up its downloads. Each address has a bucket of 10
  requests that refills one request every 2 minutes, and every request
  counts, a miss included, so the endpoint is no faster a way to guess room
  codes than a `join`. The limiter remembers at most 4096 addresses and,
  like the room creation limit, refuses a newcomer while all of them are
  still refilling.

To replay a downloaded log and check its final hash, run
`go -C server run ./cmd/ajayeb-sim replay ajayeb-K7QX2M.json` (or a built
`ajayeb-sim replay FILE`). It replays every action through the engine's own
legality checks and reports a mismatch with the log's `finalHash`.

## The scenario format

A Scenario describes a starting position that is not the plain start of the
game: a particular age, particular cities, particular tokens, and so on. It
is used to build fixed positions for tests, for the developer-only endpoint
that creates a room at a chosen position, and for the named scenarios the
engine ships for its own tests. Every field is optional, and the zero
Scenario is the ordinary start of Age I after a draft in which each player
took the wonders offered to it in order. Names of cards, wonders and tokens
are the kebab-case wire slugs, not numeric ids.

The fields:

- `name`: a label for a named scenario; not used by `NewGameFromScenario`
  itself.
- `firstPlayer`: overrides which seat started the draft and Age I, and which
  seat is to move unless `toMove` says otherwise.
- `wonders`: each seat's four wonders, in slot order. When it is left out,
  the draft is completed automatically, each pick taking the first wonder
  still on offer.
- `age`: which age's structure is laid out, 1 to 3 (0 means 1). Earlier
  ages' cards that the scenario does not place elsewhere go out of play.
- `players`: each seat's `coins` (default the starting amount), `city` (the
  cards it has built, as slugs), `tokens` (progress tokens it holds) and
  `built` (the wonders it has built, each with the wonder's slug and,
  optionally, the slug of the card tucked under it; when a built wonder's
  tucked card is left out, an unnamed card is used).
- `boardTokens`: when given, exactly the set of tokens on the board;
  otherwise the board keeps its dealt tokens minus any given to a player or
  placed in `boxTokenOrder`.
- `boxTokenOrder`: puts these tokens at the front of the box, in this order,
  so that a scripted Great Library draw is reproducible (it draws the first
  three). The rest of the box follows in its dealt order.
- `discard`: cards in the discard pile.
- `accessible`: cards of the current age moved into accessible, face-up
  slots of the structure.
- `onlyLeft`: empties the structure down to just these cards of the current
  age, placed in the first slots in layout order, for setting up the end of
  an age.
- `pawn`: the conflict pawn's position, -8 to 8 (positive toward seat 1's
  capital); any looting token the pawn has already reached is gone.
- `toMove`: the seat that is to move.
- `trigger` and `expect`: for a named scenario's own tests, a script of
  action keys to apply once the scenario is built, and a short description
  of what that script should lead to. `NewGameFromScenario` ignores both;
  they exist only for the engine's own test suite.

`NewGameFromScenario(seed, options, scenario)` builds the position by running
an ordinary `NewGame(seed, options)` first, so that everything the scenario
leaves unspecified is still random but reproducible from the seed, and then
rearranging exactly what the scenario names. The result always satisfies
`CheckInvariants`; a scenario that asks for an impossible position (a card
named twice, a wonder a seat was not dealt, and so on) is rejected with an
error instead of producing a broken state.
