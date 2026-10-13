// turn.ts: the single source of "what now" for the TurnBar (the UX design's
// phrase for it). A pure mapper from the server view plus the session's own
// connection and request flags to one TurnState. TurnBar.svelte turns a
// TurnState into the message the player reads; nothing here calls t() or
// decides what is legal, it only reads what the view and the connection
// layer already know.

import type { GameEvent, Mode, PendingKind, View } from '$lib/wire';

export type Seat = 0 | 1;

/** A seat taking an extra turn, and the wonder that granted it when known. */
export interface PlayAgain {
  seat: Seat;
  wonder?: string;
}

/**
 * Who is taking an extra turn right now, read from the game's events,
 * oldest first. The view cannot say it: its extraTurn flag means
 * "plays again once this turn is over", and the engine clears it the moment
 * the extra turn starts. The events can: a wonder's extraTurn event grants
 * the extra turn, and the extra turn lasts until that seat's next play
 * action (build, discard or wonder), an age's end or the game's end.
 *
 * So, from the newest event backwards: an extraTurn event is the answer
 * (with the wonder event of the same seat just before it); a play action,
 * an extra turn lost, a new age, a starter choice or the end of the game
 * first means nobody plays again. Everything else (payments, coins, the
 * pawn, reveals, tokens, a destroy or a rebuild resolving the wonder's own
 * choice) happens inside a turn and is skipped.
 *
 * Display only: it re-reads the server's own events. The caller passes the
 * whole history's events (State.history, which a join, a reload or a
 * resync brings in full), so a page reloaded during the extra turn still
 * finds the grant. Only a log that lacks it (a fixture room seen from a
 * late join) gets undefined; the caller then falls back on the view's
 * playingAgain flag.
 */
export function playAgainFrom(log: readonly GameEvent[]): PlayAgain | undefined {
  for (let i = log.length - 1; i >= 0; i--) {
    const e = log[i];
    if (!e) continue;
    switch (e.kind) {
      case 'extraTurn': {
        if (e.seat !== 0 && e.seat !== 1) return undefined;
        const seat: Seat = e.seat;
        for (let j = i - 1; j >= 0; j--) {
          const w = log[j];
          if (!w) continue;
          if (w.kind === 'wonder') {
            return w.seat === seat && w.wonder ? { seat, wonder: w.wonder } : { seat };
          }
          if (w.kind === 'build' || w.kind === 'discard' || w.kind === 'ageStart') break;
        }
        return { seat };
      }
      case 'build':
      case 'discard':
      case 'wonder':
      case 'extraTurnLost':
      case 'ageStart':
      case 'starterChosen':
      case 'draft':
      case 'gameOver':
        return undefined;
      default:
        continue;
    }
  }
  return undefined;
}

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
  /** The seat taking an extra turn now, if any (playAgainFrom over the
   *  whole history's events, up to the frame on screen). */
  playAgain?: PlayAgain;
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
  | { kind: 'playAgain'; seat: Seat; wonder?: string; hotseat: boolean }
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
  // The whole extra turn reads "Play again" (UX design, TurnBar states). A
  // hot-seat device names the player, since both sit at it.
  if (ctx.playAgain?.seat === toMove && view.phase === 'play') {
    const hotseat = ctx.mode === 'hotseat';
    const wonder = ctx.playAgain.wonder;
    return wonder !== undefined
      ? { kind: 'playAgain', seat: toMove, wonder, hotseat }
      : { kind: 'playAgain', seat: toMove, hotseat };
  }
  if (ctx.mode === 'hotseat') return { kind: 'hotseatTurn', seat: toMove };
  return { kind: 'yourTurn', seat: toMove };
}
