// move-summary.ts: what the other side did since the viewer last acted,
// consolidated from the game's history (docs/protocol.md, the state
// frame's `history`). A later popup ("while you were away", or the bot's
// last turn) reads it. Pure display mapping: it adds up only numbers the
// server sent in the events and decides nothing about the rules.
//
// The other side's entries are the ones after the viewer's own last entry.
// An extra turn makes them several. Online and against the bot, the viewer
// is the seat this client plays. In hot-seat it is the player to move, so
// the other side is the player who moved before them (summaryViewer).

import type {
  GameEvent,
  HistoryEntry,
  Mode,
  PendingKind,
  ScienceSymbol,
  VictoryKind
} from '$lib/wire';

export type Seat = 0 | 1;

/** The kinds of event that name a move a player made. */
export type SummaryActionKind =
  | 'draft'
  | 'build'
  | 'discard'
  | 'wonder'
  | 'starterChosen'
  | 'token'
  | 'destroy'
  | 'rebuild'
  | 'resign';

const ACTION_KINDS: ReadonlySet<string> = new Set<SummaryActionKind>([
  'draft',
  'build',
  'discard',
  'wonder',
  'starterChosen',
  'token',
  'destroy',
  'rebuild',
  'resign'
]);

/** One move, as its event names it. */
export interface SummaryAction {
  /** The ply of the history entry it belongs to. */
  ply: number;
  kind: SummaryActionKind;
  /** The seat the event names (a draft's last wonder may go to the viewer). */
  seat: Seat | null;
  card?: string;
  wonder?: string;
  token?: string;
  /** starterChosen: the seat chosen to start the age. */
  starter?: Seat;
  /** draft: the wonder was the round's last, given without a choice. */
  auto?: boolean;
}

export interface MoveSummary {
  /** The seat whose moves these are. */
  seat: Seat;
  /** The plies of the summarized entries, oldest first. */
  plies: number[];
  /** Every move in them, in order. */
  actions: SummaryAction[];
  /** The net change of each seat's coins over these moves. */
  coins: [number, number];
  /** Coins each seat paid for its moves, including trade. */
  paid: [number, number];
  /** Of those, the part that went to the other seat (Commerce/trading). */
  paidToOpponent: [number, number];
  /** The conflict pawn's first and last position, if it moved. */
  pawn: { from: number; to: number } | null;
  /** Coins lost to the military track's looting, by the seat that lost them. */
  loot: { seat: Seat; amount: number }[];
  /** Science pairs made. */
  pairs: { seat: Seat; symbol: ScienceSymbol }[];
  /** Progress tokens taken. */
  tokens: { seat: Seat; token: string; fromLibrary: boolean }[];
  /** Cards destroyed, by the seat that lost them. */
  destroyed: { seat: Seat; card: string }[];
  /** Wonders removed from the game (the eighth one), by owner. */
  wondersRemoved: { seat: Seat; wonder: string }[];
  /** Cards of the structure turned face up. */
  revealed: string[];
  /** Ages that began. */
  agesStarted: number[];
  /** Choices skipped because they had no option. */
  skipped: { seat: Seat; kind: PendingKind }[];
  /** The other side earned an extra turn (and took it, if more entries follow). */
  extraTurns: number;
  /** The game ended in these moves. */
  gameOver: { victory: VictoryKind; winner: Seat | null } | null;
}

function asSeat(n: number | undefined): Seat | null {
  return n === 0 || n === 1 ? n : null;
}

/**
 * The viewer of a summary: the seat this client plays online or against the
 * bot, the seat to move in hot-seat, and null for a spectator (who has no
 * "since my last move").
 */
export function summaryViewer(
  mode: Mode,
  yourSeats: readonly number[],
  toMove: number | undefined
): Seat | null {
  if (mode === 'hotseat') return asSeat(toMove);
  return yourSeats.length === 1 ? asSeat(yourSeats[0]) : null;
}

/**
 * The entries of the other side since viewer's last entry, oldest first.
 * Empty when the viewer acted last, or when the other side has not acted.
 */
export function entriesSinceViewer(history: readonly HistoryEntry[], viewer: Seat): HistoryEntry[] {
  let i = history.length;
  while (i > 0 && history[i - 1]!.seat !== viewer) i--;
  return history.slice(i);
}

/**
 * What the other side did since the viewer last acted, or null when there
 * is nothing to tell.
 */
export function summarizeSince(history: readonly HistoryEntry[], viewer: Seat): MoveSummary | null {
  const entries = entriesSinceViewer(history, viewer);
  if (entries.length === 0) return null;
  return summarize(entries, viewer === 0 ? 1 : 0);
}

/** Consolidates entries, all taken by seat, into one summary. */
export function summarize(entries: readonly HistoryEntry[], seat: Seat): MoveSummary {
  const s: MoveSummary = {
    seat,
    plies: [],
    actions: [],
    coins: [0, 0],
    paid: [0, 0],
    paidToOpponent: [0, 0],
    pawn: null,
    loot: [],
    pairs: [],
    tokens: [],
    destroyed: [],
    wondersRemoved: [],
    revealed: [],
    agesStarted: [],
    skipped: [],
    extraTurns: 0,
    gameOver: null
  };
  for (const entry of entries) {
    s.plies.push(entry.ply);
    for (const e of entry.events) take(s, entry.ply, e);
  }
  return s;
}

function take(s: MoveSummary, ply: number, e: GameEvent): void {
  const seat = asSeat(e.seat);
  if (ACTION_KINDS.has(e.kind)) s.actions.push(action(ply, e, seat));
  switch (e.kind) {
    case 'pay':
      // The payer's coins went down by the total; the part for the
      // opponent went to them (no separate coins event says so).
      if (seat !== null) {
        const total = e.amount ?? 0;
        const toOpp = e.toOpponent ?? 0;
        s.coins[seat] -= total;
        s.coins[other(seat)] += toOpp;
        s.paid[seat] += total;
        s.paidToOpponent[seat] += toOpp;
      }
      break;
    case 'discard':
      if (seat !== null) s.coins[seat] += e.amount ?? 0;
      break;
    case 'coins':
      if (seat !== null) s.coins[seat] += e.amount ?? 0;
      break;
    case 'loot':
      if (seat !== null) {
        s.coins[seat] -= e.amount ?? 0;
        s.loot.push({ seat, amount: e.amount ?? 0 });
      }
      break;
    case 'pawn':
      if (e.from !== undefined && e.to !== undefined) {
        s.pawn = s.pawn ? { from: s.pawn.from, to: e.to } : { from: e.from, to: e.to };
      }
      break;
    case 'pair':
      if (seat !== null && e.symbol) s.pairs.push({ seat, symbol: e.symbol });
      break;
    case 'token':
      if (seat !== null && e.token)
        s.tokens.push({ seat, token: e.token, fromLibrary: !!e.fromLibrary });
      break;
    case 'destroy':
      // The event names the destroyer; the card was the other seat's.
      if (seat !== null && e.card) s.destroyed.push({ seat: other(seat), card: e.card });
      break;
    case 'wonderRemoved':
      if (seat !== null && e.wonder) s.wondersRemoved.push({ seat, wonder: e.wonder });
      break;
    case 'reveal':
      if (e.card) s.revealed.push(e.card);
      break;
    case 'ageStart':
      if (e.age !== undefined) s.agesStarted.push(e.age);
      break;
    case 'effectSkipped':
      if (seat !== null && e.skipped) s.skipped.push({ seat, kind: e.skipped });
      break;
    case 'extraTurn':
      if (seat === s.seat) s.extraTurns++;
      break;
    case 'gameOver':
      if (e.victory) s.gameOver = { victory: e.victory, winner: asSeat(e.winner) };
      break;
    default:
      break;
  }
}

function action(ply: number, e: GameEvent, seat: Seat | null): SummaryAction {
  const a: SummaryAction = { ply, kind: e.kind as SummaryActionKind, seat };
  if (e.card !== undefined) a.card = e.card;
  if (e.wonder !== undefined) a.wonder = e.wonder;
  if (e.token !== undefined) a.token = e.token;
  const starter = asSeat(e.starter);
  if (starter !== null) a.starter = starter;
  if (e.auto) a.auto = true;
  return a;
}

function other(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
}
