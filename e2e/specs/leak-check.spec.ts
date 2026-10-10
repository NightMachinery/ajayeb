// The frame leak checker's own test (fixtures/leak-check.ts), with no
// browser: a whole online game is played over the real socket by two
// protocol clients and a spectator (fixtures/ws-game.ts), from the
// great-library scenario so that the one asymmetric secret, the Great
// Library's offer, is part of it. The checker must pass the frames as the
// server sent them, and must catch a leak planted in a copy of one of them:
// a face-down card's identity, a removed card, the library offer in the
// other player's view, the seed, and a later age's card.

import { expect, test } from '@playwright/test';
import type { State } from '../../web/src/lib/wire';
import {
  checkLeaks,
  describeLeaks,
  mentions,
  type CapturedFrame,
  type LeakRule,
  type ViewerStream
} from '../fixtures/leak-check';
import { recordOnlineGame, recordPolicy, type RecordedOnlineGame } from '../fixtures/ws-game';

const ORIGIN = 'http://127.0.0.1:38954';

/** A copy of the streams with frame `index` of `viewer` rewritten by `edit`. */
function plant(
  streams: ViewerStream[],
  viewer: string,
  index: number,
  edit: (msg: State) => void
): ViewerStream[] {
  return streams.map((s) => {
    if (s.name !== viewer) return s;
    const frames: CapturedFrame[] = s.frames.map((f, i) => {
      if (i !== index) return f;
      const msg = JSON.parse(f.text) as State;
      edit(msg);
      return { ...f, text: JSON.stringify(msg) };
    });
    return { ...s, frames };
  });
}

/** The state frames of one viewer with their index in its stream. */
function states(game: RecordedOnlineGame, viewer: string): { index: number; msg: State }[] {
  const stream = game.streams.find((s) => s.name === viewer);
  if (!stream) throw new Error(`no stream ${viewer}`);
  return stream.frames.flatMap((f, index) => {
    const msg = JSON.parse(f.text) as State;
    return msg.t === 'state' ? [{ index, msg }] : [];
  });
}

function rulesAt(
  report: ReturnType<typeof checkLeaks>,
  viewer: string,
  frame: number,
  id: string
): LeakRule[] {
  return report.leaks
    .filter((l) => l.viewer === viewer && l.frame === frame && l.id === id)
    .map((l) => l.rule);
}

test.describe('the frame leak checker', () => {
  let game: RecordedOnlineGame;

  test.beforeAll(async () => {
    // The scenario's trigger first (seat 0 builds the Great Library and
    // keeps Law), then the recording policy to the end of the game.
    const trigger = ['wonder:great-library:rostrum', 'token:law'];
    game = await recordOnlineGame(ORIGIN, {
      seed: '7',
      scenario: 'great-library',
      policy: (view) => view.choices.find((c) => trigger.includes(c.key)) ?? recordPolicy(view)
    });
    expect(game.keys.slice(0, 2)).toEqual(trigger);
  });

  test('passes an online game as the server sent it, the Great Library included', () => {
    const report = checkLeaks(game.streams);
    expect(report.leaks, describeLeaks(report)).toEqual([]);
    expect(report.problems).toEqual([]);
    const c = report.coverage;
    expect(c.reveal).toBe(true);
    expect(c.faceDownUnknown).toBe(0);
    expect(c.faceDownChecks).toBeGreaterThan(0);
    expect(c.secrets.removed).toBeGreaterThan(0);
    expect(c.secrets.boxTokens).toBeGreaterThan(0);
    expect(c.library).toEqual([
      expect.objectContaining({ decider: 0, offer: ['law', 'philosophy', 'strategy'] })
    ]);
    expect(c.library[0]?.deciderFrames).toBeGreaterThan(0);
    expect(c.library[0]?.otherFrames).toBeGreaterThanOrEqual(2);
    console.log(
      `leak checker: ${game.keys.length} actions; ${JSON.stringify(c.frames)} frames; ` +
        `${c.faceDownChecks} face-down checks`
    );
  });

  test('compares whole values and key segments, never substrings', () => {
    const key = { key: 'wonder:great-library:baths', card: 'stone-reserve' };
    const named = mentions(key);
    expect(named.has('great-library')).toBe(true);
    expect(named.has('baths')).toBe(true);
    expect(named.has('wonder')).toBe(true);
    // A substring test would see the card "library" in the wonder's slug.
    expect(JSON.stringify(key)).toContain('library');
    expect(named.has('library')).toBe(false);
    expect(named.has('reserve')).toBe(false);
  });

  test("catches a face-down card's identity planted in its slot", () => {
    const frames = states(game, 'seat1');
    const report = checkLeaks(game.streams);
    expect(report.leaks).toEqual([]);
    // The first frame of seat 1 with a face-down slot whose card the game
    // later revealed: write that card into the slot, still face down.
    const target = frames.find(({ msg }) => msg.view.structure.some((s) => s.state === 'faceDown'));
    if (!target) throw new Error('no frame with a face-down slot');
    const slot = target.msg.view.structure.find((s) => s.state === 'faceDown')!;
    const card = revealedCard(game, target.msg.view.age, slot.index);
    const planted = plant(game.streams, 'seat1', target.index, (msg) => {
      const s = msg.view.structure[slot.index]!;
      s.card = card;
    });
    const leaks = checkLeaks(planted);
    expect(rulesAt(leaks, 'seat1', target.index, card)).toEqual(
      expect.arrayContaining(['face-down', 'unexplained'])
    );
  });

  test('catches a removed card planted in an event', () => {
    const frames = states(game, 'spectator');
    const end = frames.at(-1)?.msg.view.revealed;
    const removed = end?.removed.flatMap((a) => a.cards)[0];
    if (!removed) throw new Error('the game removed no card');
    const target = frames[1]!;
    const planted = plant(game.streams, 'spectator', target.index, (msg) => {
      msg.events.push({ kind: 'build', private: false, seat: 0, card: removed });
    });
    const leaks = checkLeaks(planted);
    expect(rulesAt(leaks, 'spectator', target.index, removed)).toEqual(
      expect.arrayContaining(['revealed-secret', 'unexplained'])
    );
  });

  test("catches the Great Library offer planted in the other player's view", () => {
    const target = states(game, 'seat1').find(
      ({ msg }) => msg.view.pending?.kind === 'pickLibraryToken'
    );
    if (!target) throw new Error('seat 1 never saw the library choice');
    const planted = plant(game.streams, 'seat1', target.index, (msg) => {
      msg.view.pending!.tokens = ['law', 'philosophy', 'strategy'];
    });
    const report = checkLeaks(planted);
    expect(rulesAt(report, 'seat1', target.index, 'philosophy')).toEqual(
      expect.arrayContaining(['box-token', 'library-offer'])
    );
    expect(rulesAt(report, 'seat1', target.index, 'law')).toContain('library-offer');
    expect(report.problems.join('\n')).toContain("a non-drawer's library choice");
  });

  test('catches the seed and a later age card', () => {
    const frames = states(game, 'seat0');
    const target = frames[0]!;
    const seed = '9081726354';
    const lateCard = [...ageCards(game, target.msg.view.age + 1)][0];
    if (!lateCard) throw new Error('no card of a later age is known');
    const planted = plant(game.streams, 'seat0', target.index, (msg) => {
      (msg as unknown as { seed: string }).seed = seed;
      msg.events.push({ kind: 'discard', private: false, seat: 1, card: lateCard });
    });
    const report = checkLeaks(planted, { seed });
    expect(report.coverage.seedChecked).toBe(true);
    expect(rulesAt(report, 'seat0', target.index, seed)).toEqual(['seed']);
    expect(rulesAt(report, 'seat0', target.index, lateCard)).toEqual(
      expect.arrayContaining(['later-age', 'unexplained'])
    );
    // The unplanted frames still pass with the same seed.
    expect(checkLeaks(game.streams, { seed }).leaks).toEqual([]);
  });
});

/** The card a slot of an age held, as the game's frames later showed it. */
function revealedCard(game: RecordedOnlineGame, age: number, slot: number): string {
  for (const stream of game.streams) {
    for (const f of stream.frames) {
      const msg = JSON.parse(f.text) as State;
      if (msg.t !== 'state') continue;
      const view = msg.view;
      if (view.age === age && view.structure[slot]?.state === 'faceUp') {
        return view.structure[slot]!.card!;
      }
      const end = view.revealed?.faceDown.find((s) => s.slot === slot);
      if (view.phase === 'gameOver' && view.age === age && end) return end.card;
    }
  }
  throw new Error(`slot ${slot} of age ${age} was never revealed`);
}

/** Every card the structure of an age showed face up, in any frame. */
function ageCards(game: RecordedOnlineGame, age: number): Set<string> {
  const out = new Set<string>();
  for (const stream of game.streams) {
    for (const f of stream.frames) {
      const msg = JSON.parse(f.text) as State;
      if (msg.t !== 'state' || msg.view.age !== age) continue;
      for (const s of msg.view.structure) if (s.state === 'faceUp' && s.card) out.add(s.card);
    }
  }
  return out;
}
