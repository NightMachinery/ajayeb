// Display mappers for the game screen: they sort the view's own choices by
// the card or wonder the player is looking at, and answer layout questions
// (which seat sits at the bottom, whether a move needs a second tap, whether
// the pass-device screen shows). They decide nothing about the game. Every
// action they return is a Choice or DisabledChoice the server sent, with its
// key, its cost breakdown and its reason, untouched.

import type { ConfirmMoves, Orientation } from '$lib/ui/settings.svelte';
import type {
  Choice,
  CostView,
  DisabledChoice,
  GameEvent,
  Mode,
  Phase,
  Reason,
  State,
  View
} from '$lib/wire';
import { formatLogEvent, type SeatNames } from '$lib/view/log-format';

export type Seat = 0 | 1;

/** One wonder the player could build with a given card, or could not, and why. */
export interface WonderOption {
  wonder: string;
  /** The action key (wonder:<wonder>:<card>), from the server. */
  key: string;
  /** The legal choice, when the wonder can be built with this card now. */
  choice?: Choice;
  /** Why it cannot, when it is one of the view's disabled choices. */
  reason?: Reason;
  /** What it costs (or would cost), from the server's breakdown. */
  cost?: CostView;
}

/** Everything the view offers for one card of the structure. */
export interface CardActions {
  card: string;
  /** Build the card (construct:<card>). */
  build?: Choice;
  /** Building it is not possible now, with the server's reason and cost. */
  buildBlocked?: DisabledChoice;
  /** Discard it for coins (discard:<card>). */
  discard?: Choice;
  /** Every wonder this card could go under, legal ones first. */
  wonders: WonderOption[];
}

/** The view's choices and disabled choices for one card, sorted by kind. */
export function cardActions(view: Pick<View, 'choices' | 'disabled'>, card: string): CardActions {
  const out: CardActions = { card, wonders: [] };
  for (const c of view.choices) {
    if (c.card !== card) continue;
    if (c.kind === 'construct') out.build = c;
    else if (c.kind === 'discard') out.discard = c;
    else if (c.kind === 'wonder' && c.wonder !== undefined) {
      out.wonders.push({ wonder: c.wonder, key: c.key, choice: c, cost: c.cost });
    }
  }
  for (const d of view.disabled) {
    if (d.card !== card) continue;
    if (d.kind === 'construct' && !out.build) out.buildBlocked = d;
    else if (d.kind === 'wonder' && d.wonder !== undefined) {
      if (out.wonders.some((w) => w.key === d.key)) continue;
      out.wonders.push({ wonder: d.wonder, key: d.key, reason: d.reason, cost: d.cost });
    }
  }
  return out;
}

/** The wonders this card can go under now: those of its wonder options
 *  that are one of the view's legal choices, in their order. */
export function readyWonders(actions: CardActions): string[] {
  return actions.wonders.filter((w) => w.choice !== undefined).map((w) => w.wonder);
}

/** Whether the card offers the player anything to do now. */
export function hasActions(actions: CardActions): boolean {
  return (
    actions.build !== undefined ||
    actions.discard !== undefined ||
    actions.wonders.some((w) => w.choice !== undefined)
  );
}

/** The legal wonder choices of one wonder: card slug to the choice. */
export function wonderChoicesByCard(
  view: Pick<View, 'choices'>,
  wonder: string
): ReadonlyMap<string, Choice> {
  const out = new Map<string, Choice>();
  for (const c of view.choices) {
    if (c.kind === 'wonder' && c.wonder === wonder && c.card !== undefined) out.set(c.card, c);
  }
  return out;
}

/** The structure slots whose card can go under this wonder now (the wonder-pick badges). */
export function pickableSlots(
  view: Pick<View, 'choices' | 'structure'>,
  wonder: string
): ReadonlySet<number> {
  const cards = wonderChoicesByCard(view, wonder);
  const out = new Set<number>();
  for (const slot of view.structure) {
    if (slot.card !== undefined && cards.has(slot.card)) out.add(slot.index);
  }
  return out;
}

/** Whether a wonder can be built now with some card, or the server's reason it cannot. */
export function wonderStatus(
  view: Pick<View, 'choices' | 'disabled'>,
  wonder: string
): { buildable: boolean; reason?: Reason } {
  if (view.choices.some((c) => c.kind === 'wonder' && c.wonder === wonder)) {
    return { buildable: true };
  }
  const d = view.disabled.find((x) => x.kind === 'wonder' && x.wonder === wonder);
  return d ? { buildable: false, reason: d.reason } : { buildable: false };
}

/**
 * The structure slot of the card a pending act names, for the pending
 * outline: the key is looked up among the view's choices, never parsed.
 */
export function pendingSlot(
  view: Pick<View, 'choices' | 'structure'> | null,
  key: string | undefined
): number | undefined {
  if (!view || key === undefined) return undefined;
  const card = view.choices.find((c) => c.key === key)?.card;
  if (card === undefined) return undefined;
  return view.structure.find((s) => s.card === card)?.index;
}

/**
 * The seat drawn at the bottom (portrait) or on the left (landscape and
 * desktop): the player's own seat; seat 0 for a spectator; in a hot-seat
 * room, the seat to move when the board faces the active player, else seat 0.
 */
export function homeSeat(opts: {
  mode: Mode;
  mySeats: readonly number[];
  toMove?: number;
  active: number;
  orientation: Orientation;
}): Seat {
  if (opts.mySeats.length === 0) return 0;
  if (opts.mode === 'hotseat') {
    if (opts.orientation === 'fixed') return 0;
    const seat = opts.toMove ?? opts.active;
    return seat === 1 ? 1 : 0;
  }
  return opts.mySeats[0] === 1 ? 1 : 0;
}

/** Whether a move button asks for a second tap (UX design: online rooms by default). */
export function needsConfirm(mode: Mode, setting: ConfirmMoves): boolean {
  if (setting === 'always') return true;
  if (setting === 'never') return false;
  return mode === 'online';
}

/**
 * Whether the hot-seat pass-device screen shows: the seat to move is not
 * the one holding the device (the last seat that said it was ready).
 */
export function passPending(opts: {
  mode: Mode;
  enabled: boolean;
  phase: Phase;
  toMove?: number;
  holder: number | null;
}): boolean {
  return (
    opts.mode === 'hotseat' &&
    opts.enabled &&
    opts.phase !== 'gameOver' &&
    opts.toMove !== undefined &&
    opts.holder !== null &&
    opts.toMove !== opts.holder
  );
}

/** One line per event that the log shows, for summaries and the live region. */
export function eventLines(events: readonly GameEvent[], names: SeatNames): string[] {
  const out: string[] = [];
  for (const e of events) {
    const line = formatLogEvent(e, names);
    if (line) out.push(line.text);
  }
  return out;
}

/** What a bot's move is about to take, for the pre-roll highlight. */
export interface BotPick {
  /** The structure slot of the card it takes (build, discard or a wonder). */
  slot?: number;
  /** The wonder it takes in the draft. */
  wonder?: string;
}

/**
 * What the bot's answer `next` takes from the view on screen: the slot of
 * the card it builds, discards or puts under a wonder, or the wonder it
 * drafts. Read from the answer's own events (by the seat that was thinking)
 * and the slots on screen; undefined when it takes nothing visible.
 */
export function botPick(
  view: Pick<View, 'structure'> | null,
  next: Pick<State, 'events'> | null,
  seat: number | undefined
): BotPick | undefined {
  if (!view || !next || seat === undefined) return undefined;
  for (const e of next.events) {
    if (e.seat !== seat) continue;
    if (e.kind === 'draft' && e.wonder !== undefined) return { wonder: e.wonder };
    if ((e.kind === 'build' || e.kind === 'discard' || e.kind === 'wonder') && e.card) {
      const slot = view.structure.find((s) => s.card === e.card && s.state === 'faceUp');
      return slot ? { slot: slot.index } : undefined;
    }
  }
  return undefined;
}
