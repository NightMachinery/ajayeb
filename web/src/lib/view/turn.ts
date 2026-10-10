// turn.ts: the single source of "what now" for the TurnBar (the UX design's
// phrase for it). A pure mapper from the server view plus the session's own
// connection and request flags to one TurnState. TurnBar.svelte turns a
// TurnState into the message the player reads; nothing here calls t() or
// decides what is legal, it only reads what the view and the connection
// layer already know.

import type { Mode, PendingKind, View } from '$lib/wire';

export type Seat = 0 | 1;

/** What the session layer knows beyond the view itself. A later slice wires
 *  this from Session/Presenter; it is a plain, narrow interface here so this
 *  file (and its tests) depend on nothing but data. */
export interface TurnContext {
  /** The latest view, or null before the first one arrives. */
  view: View | null;
  /** The room's mode: a hot-seat room controls both seats from one device. */
  mode: Mode;
  /** The seats this connection may act for (both in hot-seat, one online,
   *  none for a spectator). */
  mySeats: readonly Seat[];
  conn: 'connecting' | 'open' | 'reconnecting' | 'offline';
  /** An act or undo is in flight, awaiting its ack. */
  pending: boolean;
  /** The pending request has been in flight 300ms or more. */
  slow: boolean;
  /** The seat whose bot is searching right now (the state frame's
   *  `thinking`), if any. */
  thinking?: Seat;
  /** Whether each seat's player is connected (the room's seats, online
   *  rooms only): false for a player who dropped. */
  connected?: readonly [boolean | undefined, boolean | undefined];
}

export type TurnState =
  | { kind: 'connecting' }
  | { kind: 'reconnecting' }
  | { kind: 'offline' }
  | { kind: 'sent'; slow: boolean }
  | { kind: 'gameOver' }
  | { kind: 'spectating' }
  | { kind: 'aiThinking'; seat: Seat }
  | { kind: 'waitingOpponent'; seat: Seat; away?: true }
  | { kind: 'pendingChoice'; seat: Seat; pendingKind: PendingKind }
  | { kind: 'playAgain'; seat: Seat }
  | { kind: 'hotseatTurn'; seat: Seat }
  | { kind: 'yourTurn'; seat: Seat };

/**
 * The TurnBar's state, in priority order: a broken or recovering connection
 * always wins (nothing else is actionable then), then a request already in
 * flight, then the game being over, then whichever kind of turn is open.
 */
export function turnState(ctx: TurnContext): TurnState {
  if (ctx.conn === 'reconnecting') return { kind: 'reconnecting' };
  if (ctx.conn === 'offline') return { kind: 'offline' };
  if (ctx.pending) return { kind: 'sent', slow: ctx.slow };
  if (!ctx.view) return { kind: 'connecting' };

  const view = ctx.view;
  if (view.phase === 'gameOver') return { kind: 'gameOver' };
  if (ctx.mySeats.length === 0) return { kind: 'spectating' };
  if (ctx.thinking !== undefined) return { kind: 'aiThinking', seat: ctx.thinking };

  // The view's toMove is a plain number on the wire (0 or 1 in practice,
  // enforced by the engine, not by this type); Seat narrows it for callers.
  const toMove = view.toMove as Seat | undefined;
  if (toMove === undefined) return { kind: 'connecting' };

  // In a bot room the other seat is always the bot: between its thinking
  // frames (a resync, a frame the outbox merged) it is still the one to
  // answer, so it reads as thinking rather than as an opponent to wait for.
  if (!ctx.mySeats.includes(toMove)) {
    if (ctx.mode === 'ai') return { kind: 'aiThinking', seat: toMove };
    // Online, the player to move may have dropped: the bar says so.
    return ctx.connected?.[toMove] === false
      ? { kind: 'waitingOpponent', seat: toMove, away: true }
      : { kind: 'waitingOpponent', seat: toMove };
  }

  if (view.pending && view.pending.seat === toMove) {
    return { kind: 'pendingChoice', seat: toMove, pendingKind: view.pending.kind };
  }
  if (view.extraTurn) return { kind: 'playAgain', seat: toMove };
  if (ctx.mode === 'hotseat') return { kind: 'hotseatTurn', seat: toMove };
  return { kind: 'yourTurn', seat: toMove };
}
