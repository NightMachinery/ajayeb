// The frame leak check: given every WebSocket frame each viewer of one game
// received, in order, it proves that no frame named a card, wonder or token
// before that viewer was entitled to see it.
//
// It works from ground truth that the frames themselves end up revealing,
// never from the server's redaction code:
//
// - At game over every view carries a `revealed` section (docs/rules/
//   decisions.md, "Reveal at game end"): the cards set aside from each age,
//   the deals of ages never laid out, the cards still face down, and the
//   guilds, wonders and progress tokens left in the box. None of them may
//   appear in any frame before the game is over.
// - A face-down slot's card is learned when the slot turns face up (a
//   `reveal` event, or the slot shown face up in a later frame), or from the
//   final `revealed.faceDown`. No frame that shows the slot face down may
//   name that card.
// - An age's cards (every card its structure ever showed) may not appear
//   before that age is laid out, nor the second draft round's wonders
//   during the first round.
// - The Great Library's offer (the drawer's own `pending.tokens`) may reach
//   no other viewer: their views show only `pending.hiddenTokens`, and the
//   tokens put back in the box are named by nobody, the drawer included,
//   once the choice is made.
// - The game's seed (read from the server's room file by the caller) is never
//   part of any frame, even at the end.
//
// On top of those it runs one check that needs no ground truth: every slug a
// frame names must be public to that viewer by then, that is, shown in a
// public place of this frame or an earlier one (a face-up structure card, a
// city, a discard pile, a tucked card, a wonder, a token on the board or in
// a city, the draft offer, the open choice's options, or the final reveal).
//
// Frames are parsed and compared value by value: every whole string, and
// each colon-separated segment of a string (an action key such as
// `wonder:great-library:baths`), never raw substrings. The card `library`
// is a substring of the wonder `great-library`, and a substring test would
// see one in the other.

import { cardSlugs, tokenSlugs, wonderSlugs } from '../../web/src/lib/wire/catalog.gen';
import type { GameEvent, RevealView, View } from '../../web/src/lib/wire';

/** One frame as a viewer received it. */
export interface CapturedFrame {
  /** The raw text frame. */
  text: string;
  /** The room version of a state frame, else null. */
  version: number | null;
  /** Milliseconds since the capture started. */
  at: number;
  /** Which socket of this viewer received it (1 for the first; a reload opens another). */
  socket: number;
}

/** Every frame one viewer (a browser context or a protocol client) received, in order. */
export interface ViewerStream {
  name: string;
  frames: CapturedFrame[];
}

export type LeakRule =
  | 'revealed-secret'
  | 'box-token'
  | 'face-down'
  | 'later-age'
  | 'draft-round-2'
  | 'library-offer'
  | 'seed'
  | 'unexplained';

export interface Leak {
  viewer: string;
  /** The frame's place in its viewer's stream. */
  frame: number;
  version: number | null;
  ply: number | null;
  id: string;
  rule: LeakRule;
}

export interface LibraryCoverage {
  decider: number;
  offer: string[];
  /** Frames of the drawer that showed the offer. */
  deciderFrames: number;
  /** Frames of other viewers taken while the offer was open (each must hide it). */
  otherFrames: number;
}

export interface LeakReport {
  leaks: Leak[];
  /** Problems with the frames themselves (unparseable, the reveals disagree, and so on). */
  problems: string[];
  coverage: {
    frames: Record<string, number>;
    stateFrames: Record<string, number>;
    /** A game-over frame with a revealed section was seen. */
    reveal: boolean;
    /** Ids the final reveal lists as never shown during the game, by kind. */
    secrets: {
      removed: number;
      undealt: number;
      faceDown: number;
      boxGuilds: number;
      boxWonders: number;
      boxTokens: number;
    };
    /** Face-down slots whose card was learned (age:slot), across the game. */
    faceDownIdentities: number;
    /** Checks of one face-down slot in one frame against its known card. */
    faceDownChecks: number;
    /** Face-down slots seen in some frame whose card was never learned. */
    faceDownUnknown: number;
    /** Cards known per age structure (ground truth for "later age"). */
    ageCards: Record<number, number>;
    draftRound2: number;
    library: LibraryCoverage[];
    seedChecked: boolean;
  };
}

export interface LeakCheckOptions {
  /** The game's seed as a decimal string (from the room file), to check it is never sent. */
  seed?: string;
}

const SLUGS: ReadonlySet<string> = new Set<string>([...cardSlugs, ...wonderSlugs, ...tokenSlugs]);

/**
 * Every value a parsed frame names: each string whole, each segment of a
 * string that has colons, each object key, and each number as its decimal
 * text. Never substrings.
 */
export function mentions(value: unknown): Set<string> {
  const out = new Set<string>();
  const walk = (v: unknown): void => {
    if (typeof v === 'string') {
      out.add(v);
      if (v.includes(':')) for (const part of v.split(':')) out.add(part);
    } else if (typeof v === 'number') {
      out.add(String(v));
    } else if (Array.isArray(v)) {
      for (const item of v) walk(item);
    } else if (v !== null && typeof v === 'object') {
      for (const [key, item] of Object.entries(v)) {
        out.add(key);
        walk(item);
      }
    }
  };
  walk(value);
  return out;
}

/** The slugs a view shows in public places: what this viewer may name from now on. */
export function publicSlugs(view: View): Set<string> {
  const out = new Set<string>();
  const add = (s: string | undefined) => {
    if (s !== undefined) out.add(s);
  };
  for (const offer of view.draft?.offer ?? []) add(offer.wonder);
  for (const slot of view.structure) if (slot.state === 'faceUp') add(slot.card);
  for (const p of view.players) {
    p.city.forEach(add);
    p.tokens.forEach(add);
    for (const w of p.wonders) {
      add(w.wonder);
      add(w.tucked);
    }
  }
  view.boardTokens.forEach(add);
  view.discard.forEach(add);
  if (view.pending) {
    view.pending.cards.forEach(add);
    // The library's offer is public only to the seat drawing from it.
    const library = view.pending.kind === 'pickLibraryToken';
    if (!library || view.viewer === view.pending.seat) view.pending.tokens.forEach(add);
  }
  const r = view.revealed;
  if (r) {
    for (const a of [...r.removed, ...r.undealt]) a.cards.forEach(add);
    for (const s of r.faceDown) add(s.card);
    [...r.boxGuilds, ...r.boxWonders, ...r.boxTokens].forEach(add);
  }
  return out;
}

interface Parsed {
  index: number;
  frame: CapturedFrame;
  msg: { t?: string; view?: View; events?: GameEvent[] } | null;
  view: View | null;
}

function parseStream(stream: ViewerStream, problems: string[]): Parsed[] {
  return stream.frames.map((frame, index) => {
    let msg: Parsed['msg'] = null;
    try {
      msg = JSON.parse(frame.text) as Parsed['msg'];
    } catch {
      problems.push(`${stream.name} frame ${index}: not JSON`);
    }
    const view = msg?.t === 'state' && msg.view ? msg.view : null;
    return { index, frame, msg, view };
  });
}

const slotKey = (age: number, slot: number) => `${age}:${slot}`;

function sameReveal(a: RevealView, b: RevealView): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/** Runs every check over the streams of one game. */
export function checkLeaks(streams: ViewerStream[], opts: LeakCheckOptions = {}): LeakReport {
  const problems: string[] = [];
  const parsed = streams.map((s) => ({ name: s.name, frames: parseStream(s, problems) }));

  // Ground truth, gathered from every viewer's frames.
  const identity = new Map<string, string>();
  const ageCards = new Map<number, Set<string>>();
  const draftRound2 = new Set<string>();
  const offers = new Map<string, { decider: number; offer: string[] }>();
  let reveal: RevealView | null = null;
  let finalAge = 0;
  const cardsOfAge = (age: number) => {
    let set = ageCards.get(age);
    if (!set) ageCards.set(age, (set = new Set()));
    return set;
  };

  for (const stream of parsed) {
    for (const { view, msg } of stream.frames) {
      if (!view) continue;
      for (const slot of view.structure) {
        if (slot.state === 'faceUp' && slot.card !== undefined) {
          identity.set(slotKey(view.age, slot.index), slot.card);
          cardsOfAge(view.age).add(slot.card);
        }
      }
      for (const ev of msg?.events ?? []) {
        if (ev.kind === 'reveal' && ev.card !== undefined && ev.slot !== undefined) {
          identity.set(slotKey(view.age, ev.slot), ev.card);
          cardsOfAge(view.age).add(ev.card);
        }
      }
      if (view.draft?.round === 2) for (const o of view.draft.offer) draftRound2.add(o.wonder);
      const p = view.pending;
      if (p?.kind === 'pickLibraryToken' && p.tokens.length > 0) {
        offers.set(p.tokens.join(','), { decider: p.seat, offer: [...p.tokens] });
      }
      if (view.phase === 'gameOver' && view.revealed) {
        if (reveal && !sameReveal(reveal, view.revealed)) {
          problems.push(`${stream.name}: its revealed section differs from another viewer's`);
        }
        reveal ??= view.revealed;
        finalAge = view.age;
      }
    }
  }
  if (reveal) {
    for (const s of reveal.faceDown) {
      identity.set(slotKey(finalAge, s.slot), s.card);
      cardsOfAge(finalAge).add(s.card);
    }
    for (const a of reveal.undealt) for (const c of a.cards) cardsOfAge(a.age).add(c);
  }

  // Never shown before the end: the final reveal, minus what was public at
  // some point anyway (nothing, if the server is right; the unexplained
  // check below catches a public place that should not have shown it).
  const revealedSecrets = new Set<string>();
  const boxTokens = new Set<string>(reveal?.boxTokens ?? []);
  if (reveal) {
    for (const a of [...reveal.removed, ...reveal.undealt])
      a.cards.forEach((c) => revealedSecrets.add(c));
    for (const s of reveal.faceDown) revealedSecrets.add(s.card);
    [...reveal.boxGuilds, ...reveal.boxWonders].forEach((c) => revealedSecrets.add(c));
  }

  const leaks: Leak[] = [];
  const frames: Record<string, number> = {};
  const stateFrames: Record<string, number> = {};
  let faceDownChecks = 0;
  const unknownSlots = new Set<string>();
  const library: LibraryCoverage[] = [...offers.values()].map((o) => ({
    decider: o.decider,
    offer: o.offer,
    deciderFrames: 0,
    otherFrames: 0
  }));
  // A seed of fewer than four digits is a number frames hold anyway (a
  // version, a ply, a coin count), so it cannot be told apart; the e2e
  // server deals from --seed-base 1000.
  const seedPattern =
    opts.seed && /^[0-9]{4,}$/.test(opts.seed)
      ? new RegExp(`(?<![0-9])${opts.seed}(?![0-9])`)
      : undefined;

  for (const stream of parsed) {
    frames[stream.name] = stream.frames.length;
    stateFrames[stream.name] = 0;
    const seen = new Set<string>();
    let last: View | null = null;

    for (const { index, frame, msg, view } of stream.frames) {
      if (view) {
        stateFrames[stream.name] = (stateFrames[stream.name] ?? 0) + 1;
        last = view;
      }
      const named = mentions(msg);
      const report = (id: string, rule: LeakRule) =>
        leaks.push({
          viewer: stream.name,
          frame: index,
          version: frame.version,
          ply: last?.ply ?? null,
          id,
          rule
        });

      // The seed as a whole value, or as a run of digits anywhere in the text
      // (a JSON number past 2^53 would not survive parsing).
      if (seedPattern && (named.has(opts.seed ?? '') || seedPattern.test(frame.text))) {
        report(opts.seed ?? '', 'seed');
      }

      const slugs = [...named].filter((s) => SLUGS.has(s));
      const over = last?.phase === 'gameOver';
      const here = view ? publicSlugs(view) : new Set<string>();
      for (const s of here) seen.add(s);

      // The face-down slots of this frame, each against its known card.
      const faceDown = new Map<string, number>();
      if (view) {
        for (const slot of view.structure) {
          if (slot.state !== 'faceDown') continue;
          const card = identity.get(slotKey(view.age, slot.index));
          if (card === undefined) {
            unknownSlots.add(slotKey(view.age, slot.index));
            continue;
          }
          faceDownChecks += 1;
          faceDown.set(card, slot.index);
        }
      }

      // The Great Library: the offer reaches only the drawer.
      const pending = view?.pending;
      for (const lib of library) {
        if (pending?.kind !== 'pickLibraryToken' || pending.seat !== lib.decider) continue;
        if (view?.viewer === lib.decider) lib.deciderFrames += 1;
        else {
          lib.otherFrames += 1;
          if (pending.tokens.length > 0 || pending.hiddenTokens !== lib.offer.length) {
            problems.push(
              `${stream.name} frame ${index}: a non-drawer's library choice shows ` +
                `tokens ${JSON.stringify(pending.tokens)} and hiddenTokens ${pending.hiddenTokens}`
            );
          }
        }
      }
      const drawerNow =
        pending?.kind === 'pickLibraryToken' && view?.viewer === pending.seat
          ? new Set(pending.tokens)
          : new Set<string>();

      const age = view?.age ?? last?.age ?? 0;
      for (const s of slugs) {
        if (!over) {
          if (revealedSecrets.has(s)) report(s, 'revealed-secret');
          if (boxTokens.has(s) && !drawerNow.has(s)) report(s, 'box-token');
          if (faceDown.has(s)) report(s, 'face-down');
          for (const [a, cards] of ageCards) {
            if (a > age && cards.has(s)) report(s, 'later-age');
          }
          if (last?.phase === 'wonderDraft' && last.draft?.round === 1 && draftRound2.has(s)) {
            report(s, 'draft-round-2');
          }
          for (const lib of library) {
            if (lib.offer.includes(s) && view?.viewer !== lib.decider && !seen.has(s)) {
              report(s, 'library-offer');
            }
          }
        }
        if (!seen.has(s)) report(s, 'unexplained');
      }
    }
  }

  const count = (xs: { cards: string[] }[]) => xs.reduce((n, a) => n + a.cards.length, 0);
  return {
    leaks,
    problems,
    coverage: {
      frames,
      stateFrames,
      reveal: reveal !== null,
      secrets: {
        removed: reveal ? count(reveal.removed) : 0,
        undealt: reveal ? count(reveal.undealt) : 0,
        faceDown: reveal?.faceDown.length ?? 0,
        boxGuilds: reveal?.boxGuilds.length ?? 0,
        boxWonders: reveal?.boxWonders.length ?? 0,
        boxTokens: reveal?.boxTokens.length ?? 0
      },
      faceDownIdentities: identity.size,
      faceDownChecks,
      faceDownUnknown: [...unknownSlots].filter((k) => !identity.has(k)).length,
      ageCards: Object.fromEntries([...ageCards].map(([a, s]) => [a, s.size])),
      draftRound2: draftRound2.size,
      library,
      seedChecked: seedPattern !== undefined
    }
  };
}

/** A short text of a report's leaks and problems, for an assertion message. */
export function describeLeaks(report: LeakReport, max = 20): string {
  const lines = report.leaks
    .slice(0, max)
    .map(
      (l) =>
        `${l.viewer} frame ${l.frame} (version ${l.version ?? '-'}, ply ${l.ply ?? '-'}): ${l.id} [${l.rule}]`
    );
  if (report.leaks.length > max) lines.push(`... and ${report.leaks.length - max} more`);
  return [...lines, ...report.problems].join('\n');
}
