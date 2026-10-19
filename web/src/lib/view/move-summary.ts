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
//
// The opponent move summary popup (components/game/MoveSummary.svelte,
// M10.S03b) shows `lines`: one line per move in the order it happened,
// each with the coins, the pawn, the looting and the science pairs it
// brought, and the age changes, extra turns and skipped choices between
// them. latestSummary() is what its top bar button reopens: the other
// side's last run of moves, even after the viewer has moved since.
// finalTurn() is the turn that ended the game, which the result screen
// shows in place of a popup (M11.gameover-last-move).

import type {
  GameEvent,
  HistoryEntry,
  Mode,
  PendingKind,
  ScienceSymbol,
  State,
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

/** The kinds of line the summary popup shows. */
export type SummaryLineKind =
  | SummaryActionKind
  | 'ageStart'
  | 'extraTurn'
  | 'extraTurnLost'
  | 'wonderRemoved'
  | 'skipped'
  | 'gameOver';

/**
 * One line of the summary: a move (a card taken, a wonder, a token, a
 * choice) with what it brought, or something that happened between moves.
 */
export interface SummaryLine {
  /** The ply of the history entry it belongs to. */
  ply: number;
  kind: SummaryLineKind;
  /** The seat the event names (for wonderRemoved, the wonder's owner). */
  seat: Seat | null;
  card?: string;
  wonder?: string;
  token?: string;
  /** starterChosen: the seat chosen to start the age. */
  starter?: Seat;
  /** ageStart: the age that began. */
  age?: number;
  /** draft: the round's last wonder, given without a choice. */
  auto?: boolean;
  /** token: taken from the Great Library's offer. */
  fromLibrary?: boolean;
  /** skipped: the choice that had no option. */
  skipped?: PendingKind;
  /** discard: the coins it brought. */
  amount?: number;
  /** gameOver: how, and who won (null for a shared victory). */
  victory?: VictoryKind;
  winner?: Seat | null;
  /** Coins paid for the move, trade included, and the part the other seat got. */
  paid: number;
  toOpponent: number;
  /** Other changes of each seat's coins (a card's or a wonder's coins). */
  coins: [number, number];
  /** The conflict pawn's first and last position in this move, if it moved. */
  pawn: { from: number; to: number } | null;
  /** Coins lost to looting, by the seat that lost them. */
  loot: { seat: Seat; amount: number; value: number }[];
  /** Science pairs made by this move. */
  pairs: ScienceSymbol[];
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
  /** The popup's lines, in the order things happened. */
  lines: SummaryLine[];
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

/**
 * The other side's last run of moves, as the viewer sees it: the moves
 * since the viewer's last action, or, when the viewer has moved since, the
 * run before that. Null before the other side has moved. The summary popup's
 * button reopens this.
 */
export function latestSummary(history: readonly HistoryEntry[], viewer: Seat): MoveSummary | null {
  let end = history.length;
  while (end > 0 && history[end - 1]!.seat === viewer) end--;
  return summarizeSince(history.slice(0, end), viewer);
}

/** The parts of a state frame summaryTrigger reads. */
export interface TriggerFrame {
  code: State['code'];
  version: State['version'];
  events: State['events'];
  history?: State['history'];
  room: Pick<State['room'], 'mode'>;
  you: Pick<State['you'], 'seats'>;
  view: Pick<State['view'], 'toMove' | 'phase'>;
}

/**
 * Whether a frame opens the summary popup by itself, and for which entry:
 * the ply of the other side's last entry, or null for no popup. Only a
 * live frame does (the same room as the frame before it and a newer
 * version, without a history, which only a join, a resync and the frame
 * after an undo carry: those replay the game, nothing happened just now).
 * It must bring events, the other side must have moved last, and the turn
 * must be the viewer's again (so a bot's reply, both moves of an extra turn
 * and the choices inside a turn make one popup). Never during the wonder
 * draft, whose screen shows every pick, and never once the game is over:
 * the result screen shows the final turn itself (finalTurn), because two
 * modal layers at once would fight for taps and focus on a phone. `history`
 * is the Session's history with this frame taken in. With `seenPly`, unseen
 * history and a visibility wake may also trigger it; undo must be handled
 * by the caller before this check.
 */
export function summaryTrigger(
  prev: TriggerFrame | null,
  next: TriggerFrame,
  history: readonly HistoryEntry[],
  seenPly?: number
): number | null {
  // A seen marker also permits history frames and waking without a new frame.
  // The caller handles undo separately, clamping the marker to its new history.
  if (seenPly === undefined) {
    if (!prev || prev.code !== next.code || next.version <= prev.version) return null;
    if (next.history || next.events.length === 0) return null;
  }
  if (next.view.phase === 'wonderDraft' || next.view.phase === 'gameOver') return null;
  const viewer = summaryViewer(next.room.mode, next.you.seats, next.view.toMove);
  if (viewer === null || next.view.toMove !== viewer) return null;
  const last = history[history.length - 1];
  if (!last || last.seat === viewer || (seenPly !== undefined && last.ply <= seenPly)) return null;
  return last.ply;
}

/**
 * The turn that ended the game, for the result screen: the last run of
 * entries by the seat that made the final one, so a wonder's extra turn and
 * the choices inside the turn (a token, a card to destroy) come with it.
 * Null for an empty history.
 */
export function finalTurn(history: readonly HistoryEntry[]): MoveSummary | null {
  const seat = asSeat(history[history.length - 1]?.seat);
  if (seat === null) return null;
  let start = history.length - 1;
  while (start > 0 && history[start - 1]!.seat === seat) start--;
  return summarize(history.slice(start), seat);
}

/** The ply of a summary's last entry. */
export function lastPly(s: MoveSummary): number | undefined {
  return s.plies[s.plies.length - 1];
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
    gameOver: null,
    lines: []
  };
  for (const entry of entries) {
    s.plies.push(entry.ply);
    for (const e of entry.events) take(s, entry.ply, e);
    s.lines.push(...entryLines(entry));
  }
  return s;
}

function newLine(ply: number, kind: SummaryLineKind, seat: Seat | null): SummaryLine {
  return {
    ply,
    kind,
    seat,
    paid: 0,
    toOpponent: 0,
    coins: [0, 0],
    pawn: null,
    loot: [],
    pairs: []
  };
}

/**
 * One history entry's lines. A move's own events (what it paid, the coins
 * it brought, the pawn, looting, a pair) go on the move's line; an age
 * change, an extra turn, a removed wonder, a skipped choice and the end of
 * the game get lines of their own. A reveal, the draft's second round and
 * the Great Library's draw are left out: the board shows them, and the
 * token taken says the rest.
 */
export function entryLines(entry: HistoryEntry): SummaryLine[] {
  const lines: SummaryLine[] = [];
  let head: SummaryLine | null = null;
  for (const e of entry.events) {
    const seat = asSeat(e.seat);
    if (ACTION_KINDS.has(e.kind)) {
      const line = newLine(entry.ply, e.kind as SummaryActionKind, seat);
      if (e.card !== undefined) line.card = e.card;
      if (e.wonder !== undefined) line.wonder = e.wonder;
      if (e.token !== undefined) line.token = e.token;
      const starter = asSeat(e.starter);
      if (starter !== null) line.starter = starter;
      if (e.auto) line.auto = true;
      if (e.kind === 'token') line.fromLibrary = !!e.fromLibrary;
      if (e.kind === 'discard') line.amount = e.amount ?? 0;
      lines.push(line);
      head = line;
      continue;
    }
    switch (e.kind) {
      case 'pay':
        if (head) {
          head.paid += e.amount ?? 0;
          head.toOpponent += e.toOpponent ?? 0;
        }
        break;
      case 'coins':
        if (head && seat !== null) head.coins[seat] += e.amount ?? 0;
        break;
      case 'pawn':
        if (head && e.from !== undefined && e.to !== undefined) {
          head.pawn = head.pawn ? { from: head.pawn.from, to: e.to } : { from: e.from, to: e.to };
        }
        break;
      case 'loot':
        if (head && seat !== null) {
          head.loot.push({ seat, amount: e.amount ?? 0, value: e.value ?? 0 });
        }
        break;
      case 'pair':
        if (head && e.symbol) head.pairs.push(e.symbol);
        break;
      case 'ageStart': {
        const line = newLine(entry.ply, 'ageStart', null);
        if (e.age !== undefined) line.age = e.age;
        lines.push(line);
        break;
      }
      case 'extraTurn':
      case 'extraTurnLost':
        lines.push(newLine(entry.ply, e.kind, seat));
        break;
      case 'wonderRemoved':
        if (e.wonder)
          lines.push({ ...newLine(entry.ply, 'wonderRemoved', seat), wonder: e.wonder });
        break;
      case 'effectSkipped': {
        const line = newLine(entry.ply, 'skipped', seat);
        if (e.skipped) line.skipped = e.skipped;
        if (e.wonder !== undefined) line.wonder = e.wonder;
        if (e.card !== undefined) line.card = e.card;
        lines.push(line);
        break;
      }
      case 'gameOver':
        if (e.victory) {
          lines.push({
            ...newLine(entry.ply, 'gameOver', null),
            victory: e.victory,
            winner: asSeat(e.winner)
          });
        }
        break;
      default:
        break;
    }
  }
  return lines;
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
