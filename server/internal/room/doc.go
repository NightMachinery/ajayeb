// Package room runs game rooms. A room owns one game and its mode (online,
// hot-seat or against a bot), the seats and who holds them, the action log,
// per-client idempotency state, bot scheduling, undo and persistence.
//
// # Actor model
//
// Each room is an actor: a single goroutine owns all room state and processes
// messages from its inbox channel one at a time, so room logic needs no locks.
// Connections, bots and the idle timer talk to a room only by sending to its
// inbox, which is bounded (Config.InboxSize). Outbound frames go to each
// connection's Outbox, which never blocks: state frames are full snapshots,
// so a pending one is replaced by the next (latest wins, keeping its ack,
// and its events unless an undo came in between), and a client that lets
// other frames pile up is reported as overflowed for the transport to drop.
// A slow client never blocks a room.
//
// The Manager is the registry in front of the actors. It creates rooms
// (Create for a create frame, CreateFromSpec for development endpoints and
// tests), loads a room from the store when a join names one that is not in
// memory, routes each request of a Client to the room it joined, and stops
// everything on Close. Every outcome reaches the client as frames pushed into
// its outbox (created, state, reject, error); the errors the Manager returns
// only inform the transport.
//
// # Seats and modes
//
//   - online: the creator takes the seat it chose; the first other user to
//     join takes the other seat, and the room goes from lobby to playing.
//     Later users watch as spectators (a public view, no choices). A user
//     keeps a seat across connections.
//   - hotseat: the creator holds both seats, and the room sends the view of
//     the seat to move. Nobody else may join.
//   - ai: the creator takes the seat it chose (seat 0 by default) and a bot
//     the other. Nobody else may join.
//
// A join that a hot-seat or bot room would refuse is refused from the room
// file's header when the room is not in memory, without loading it, so such
// joins never take a slot among the rooms in memory.
//
// A room made from a Spec may leave seats open; the first joiner claims
// them (both seats of a hot-seat room). Presence is per seat: a bot is
// always connected, a human when one of the user's connections is attached.
// Leave and Disconnect only detach a connection; seats stay with their users.
//
// Each state frame also names the level of each bot seat, says whether this
// connection may undo now (canUndo), and which seat's bot is searching
// (thinking), so the client never works these out itself.
//
// # Requests
//
// The room version starts at the length of the action log of a new room
// and goes up by one with every accepted act and undo. It is stored in the
// room file, so it never goes down, not even across an eviction and reload
// after an undo, and a client's base and a bot's answer can always be
// checked against it: one version always names one position. An act is
// handled in this order:
//
//  1. the sender must be attached, and not as a spectator;
//  2. a seq not newer than the last one applied for the sender's user and
//     client id only re-sends the current state, with an ack for that seq,
//     and applies nothing (this check comes before the turn check so that
//     an act re-sent after the turn passed resolves instead of being
//     refused);
//  3. the game must be running, and the sender must hold the seat to move
//     (in a hot-seat room the frame's optional seat must be that seat);
//  4. base must equal the version, or the reply is reject stale followed by
//     a fresh state;
//  5. the key must belong to the engine's legal actions; the room applies
//     the engine's own action for it, never one built from client data.
//
// Only an applied act or undo records its seq. A refused one (stale, illegal,
// not your turn) does not use it up: the client may send it again with the
// same seq, and it is judged afresh. The last applied seq of each client is
// kept in a bounded LRU (Config.SeqClients) and stored in the room file, keyed
// by user id and client id together ("<user>:<client>"): clients pick their
// own ids, so the user id keeps one user's seqs from ever shadowing another's.
// Undo (hot-seat and bot rooms, never once the game is over) cuts the log back
// to just before the previous human decision, which in a bot room also removes
// the bot's answers after it, and replays from the seed, so hidden cards
// cannot change. Rematch, once the game is over, creates one new room with the
// same mode and seats and a fresh deal, stores its code, and sends every
// connection of the old room a created frame with that one code, however many
// players ask and also after a reload. Creating that room first asks the
// requester's Client.AllowCreate (the transport's creation limits); a refusal
// is ErrRateLimited, and creates nothing.
//
// # Bots
//
// When a bot is to move in a running game and a connection is attached, the
// room starts one search in its own goroutine, under a context that is
// cancelled by an undo, when the last connection leaves (the next attach
// starts it again) and when the room stops, through Config.BotPool when one is
// set. A room nobody is attached to never searches, so rooms left behind
// cannot keep the pool busy. The bot receives only an engine Observation of
// its seat. Its seed is derived from the room's seed and the game's ply (not
// the room version), so a game replays the same way and an undo cannot reroll
// the bot's answer. The search waits out Config.BotDelay on the room clock,
// then posts its answer to the inbox tagged with its generation, a counter
// the room bumps for every search it starts; the room drops the answer
// unless it comes from the search running for the current version. The
// version alone cannot tell searches apart: a detach cancels a search and
// the next attach starts another for the same version, and the cancelled
// one may still answer while the new one runs. A bot that fails,
// or answers with an illegal action, is replaced by the first legal action,
// and logged, so a game never stalls.
//
// # Persistence and memory
//
// The room writes its file through the store after every change (an accepted
// action, an undo, a seat claimed, a rematch created). A failed write is
// logged and leaves the room dirty: it keeps playing from memory, retries on
// the next change, and is not evicted until a write succeeds. A room with no
// connection for Config.IdleTimeout (30 minutes by default) leaves memory and
// is reloaded from its file, by replay, on the next join, with its stored
// version, seqs and rematch code. The Manager holds at most Config.MaxRooms
// rooms. When a create or a load finds it full, the room with no connection
// (and nothing left to save) that has been idle longest is evicted at once to
// make room; only when every room has a connection is the answer error busy.
// Rooms nobody uses therefore never lock anyone out. A room admits a new
// spectator only while it has fewer than Config.MaxClients connections, and a
// user who holds a seat while that user has fewer than Config.MaxClients
// connections there, so spectators can never lock a player out.
//
// # Rules
//
//   - The room applies an action only when its key appears in the engine's
//     legal actions for the seat to move. It never builds actions from client
//     data.
//   - Every state frame is built per connection with engine.ViewFor and
//     engine.FilterEvents for that connection's viewer; nothing else decides
//     what a client may see.
//   - A bot runs off the actor goroutine on an engine Observation and reports
//     its key back through the inbox together with the version it saw; a
//     result that went stale (for example after an undo) is dropped.
//   - Hot-seat and bot rooms accept only the creator's user id, and only they
//     support undo. Online rooms seat the first two distinct users; later
//     joiners are spectators.
//   - Time comes from an injected Clock, never from time.Now in room logic, so
//     bot delays and idle expiry are testable with a fake clock (package
//     roomtest).
//   - Logging goes only through an injected *slog.Logger. Rooms only ever see
//     user ids, never tokens.
package room
