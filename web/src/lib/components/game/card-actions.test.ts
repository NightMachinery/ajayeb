import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { GameEvent, View } from '$lib/wire';
import {
  botPick,
  cardActions,
  eventLines,
  hasActions,
  homeSeat,
  needsConfirm,
  passPending,
  pendingSlot,
  pickableSlots,
  toastLines,
  wonderChoicesByCard,
  wonderStatus
} from './card-actions';

function view(name: (typeof protocolFixtureNames)[number]): View {
  const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!v) throw new Error(`no protocol fixture named ${name}`);
  return v;
}

describe('cardActions', () => {
  const costs = view('costs-seat0');

  it('sorts one card’s choices by kind, legal wonders first', () => {
    const a = cardActions(costs, 'courthouse');
    expect(a.build?.key).toBe('construct:courthouse');
    expect(a.build?.cost).toBeDefined();
    expect(a.buildBlocked).toBeUndefined();
    expect(a.discard?.key).toBe('discard:courthouse');
    expect(a.discard?.gain).toBeGreaterThan(0);
    expect(a.wonders.map((w) => [w.key, w.choice !== undefined, w.reason])).toEqual([
      ['wonder:pyramids:courthouse', true, undefined],
      ['wonder:temple-of-artemis:courthouse', true, undefined],
      ['wonder:colossus:courthouse', false, 'unaffordable'],
      ['wonder:mausoleum:courthouse', false, 'unaffordable']
    ]);
    expect(hasActions(a)).toBe(true);
  });

  it('reports a build the server refused, with its reason and cost', () => {
    const a = cardActions(costs, 'parade-ground');
    expect(a.build).toBeUndefined();
    expect(a.buildBlocked?.reason).toBe('unaffordable');
    expect(a.buildBlocked?.cost?.affordable).toBe(false);
    expect(a.discard?.key).toBe('discard:parade-ground');
  });

  it('has nothing for a card the view offers nothing for', () => {
    const a = cardActions(costs, 'lumber-yard');
    expect(a).toEqual({ card: 'lumber-yard', wonders: [] });
    expect(hasActions(a)).toBe(false);
  });

  it('has nothing on the waiting seat’s view', () => {
    const waiting = view('costs-seat1');
    expect(waiting.choices).toHaveLength(0);
    expect(hasActions(cardActions(waiting, 'courthouse'))).toBe(false);
  });

  it('offers no wonder once seven are built', () => {
    const a = cardActions(view('seventh-wonder-seat0'), 'walls');
    expect(a.build?.key).toBe('construct:walls');
    expect(a.wonders).toEqual([]);
  });
});

describe('wonder-first helpers', () => {
  const costs = view('costs-seat0');

  it('maps a wonder’s legal choices by card, and marks their slots', () => {
    const byCard = wonderChoicesByCard(costs, 'pyramids');
    expect([...byCard.keys()]).toEqual([
      'courthouse',
      'statue',
      'aqueduct',
      'caravansery',
      'walls',
      'parade-ground'
    ]);
    expect(byCard.get('walls')?.key).toBe('wonder:pyramids:walls');
    expect([...pickableSlots(costs, 'pyramids')].sort((x, y) => x - y)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(pickableSlots(costs, 'colossus').size).toBe(0);
  });

  it('says whether a wonder can be built, or the server’s reason', () => {
    expect(wonderStatus(costs, 'pyramids')).toEqual({ buildable: true });
    expect(wonderStatus(costs, 'colossus')).toEqual({ buildable: false, reason: 'unaffordable' });
    expect(wonderStatus(view('seventh-wonder-seat0'), 'pyramids')).toEqual({ buildable: false });
  });
});

describe('pendingSlot', () => {
  const costs = view('costs-seat0');

  it('finds the slot of the card a sent key names', () => {
    expect(pendingSlot(costs, 'construct:walls')).toBe(4);
    expect(pendingSlot(costs, 'wonder:pyramids:statue')).toBe(1);
  });

  it('is undefined for a key that names no card or is not a choice', () => {
    expect(pendingSlot(costs, undefined)).toBeUndefined();
    expect(pendingSlot(costs, 'construct:lumber-yard')).toBeUndefined();
    expect(pendingSlot(view('pick-board-token-seat0'), 'token:law')).toBeUndefined();
    expect(pendingSlot(null, 'construct:walls')).toBeUndefined();
  });
});

describe('homeSeat', () => {
  const base = { active: 0, orientation: 'face' as const };

  it('puts a player’s own seat at home, and seat 0 for a spectator', () => {
    expect(homeSeat({ ...base, mode: 'online', mySeats: [1], toMove: 0 })).toBe(1);
    expect(homeSeat({ ...base, mode: 'ai', mySeats: [0], toMove: 1 })).toBe(0);
    expect(homeSeat({ ...base, mode: 'online', mySeats: [], toMove: 1 })).toBe(0);
  });

  it('faces the seat to move in hot-seat, or stays fixed', () => {
    expect(homeSeat({ ...base, mode: 'hotseat', mySeats: [0, 1], toMove: 1 })).toBe(1);
    expect(homeSeat({ ...base, mode: 'hotseat', mySeats: [0, 1], active: 1 })).toBe(1);
    expect(
      homeSeat({ ...base, mode: 'hotseat', mySeats: [0, 1], toMove: 1, orientation: 'fixed' })
    ).toBe(0);
  });
});

describe('needsConfirm', () => {
  it('asks online by default, always or never when set', () => {
    expect(needsConfirm('online', 'onlineOnly')).toBe(true);
    expect(needsConfirm('hotseat', 'onlineOnly')).toBe(false);
    expect(needsConfirm('ai', 'onlineOnly')).toBe(false);
    expect(needsConfirm('ai', 'always')).toBe(true);
    expect(needsConfirm('online', 'never')).toBe(false);
  });
});

describe('passPending', () => {
  const base = { mode: 'hotseat' as const, enabled: true, phase: 'play' as const };

  it('shows when the seat to move does not hold the device', () => {
    expect(passPending({ ...base, toMove: 1, holder: 0 })).toBe(true);
    expect(passPending({ ...base, toMove: 0, holder: 0 })).toBe(false);
  });

  it('never shows off, outside hot-seat, at the end, or before the first frame', () => {
    expect(passPending({ ...base, enabled: false, toMove: 1, holder: 0 })).toBe(false);
    expect(passPending({ ...base, mode: 'ai', toMove: 1, holder: 0 })).toBe(false);
    expect(passPending({ ...base, phase: 'gameOver', holder: 0 })).toBe(false);
    expect(passPending({ ...base, toMove: 1, holder: null })).toBe(false);
  });
});

describe('eventLines', () => {
  it('keeps the events the log shows, in order', () => {
    const events: GameEvent[] = [
      { kind: 'pay', private: false, seat: 0, amount: 2 },
      { kind: 'build', private: false, seat: 0, card: 'lumber-yard' },
      { kind: 'reveal', private: false, slot: 3, card: 'baths' },
      { kind: 'discard', private: false, seat: 1, card: 'baths', amount: 2 }
    ];
    expect(eventLines(events, ['Sara', 'Omid'])).toEqual([
      'Sara built Lumber Yard.',
      'Omid discarded Baths for 2 coins.'
    ]);
  });
});

describe('toastLines', () => {
  const names = ['Sara', 'Omid'] as const;

  it("an ordinary move: only the move's own line", () => {
    const events: GameEvent[] = [
      { kind: 'build', private: false, seat: 0, card: 'lumber-yard' },
      { kind: 'pay', private: false, seat: 0, amount: 2 },
      { kind: 'pawn', private: false, seat: 0, from: 0, to: 1 }
    ];
    expect(toastLines(events, names)).toEqual(['Sara built Lumber Yard.']);
  });

  it('a skipped choice adds its line after the wonder', () => {
    // The Statue of Zeus with nothing to destroy, as the server recorded it
    // (server/internal/engine/testdata/protocol/effect-skipped.json).
    const events: GameEvent[] = [
      { kind: 'wonder', private: false, seat: 0, card: 'rostrum', wonder: 'statue-of-zeus' },
      { kind: 'pay', private: false, seat: 0, amount: 10, toOpponent: 0 },
      { kind: 'pawn', private: false, seat: 0, from: 0, to: 1 },
      {
        kind: 'effectSkipped',
        private: false,
        seat: 0,
        wonder: 'statue-of-zeus',
        skipped: 'destroyCard'
      },
      { kind: 'reveal', private: false, card: 'brewery', slot: 15 }
    ];
    expect(toastLines(events, names)).toEqual([
      'Sara built the wonder Statue of Zeus, using Rostrum.',
      'Nothing to destroy for Statue of Zeus: Omid has no card of that colour.'
    ]);
  });

  it('nothing to show: no lines', () => {
    expect(toastLines([{ kind: 'reveal', private: false, slot: 1, card: 'baths' }], names)).toEqual(
      []
    );
  });
});

describe('botPick', () => {
  const v = view('age-1-start-seat0');
  const faceUp = v.structure.find((s) => s.state === 'faceUp' && s.accessible && s.card);
  const ev = (e: Partial<GameEvent> & Pick<GameEvent, 'kind'>): GameEvent => ({
    private: false,
    ...e
  });

  it('finds the slot of the card the bot builds, discards or puts under a wonder', () => {
    if (!faceUp?.card) throw new Error('no face-up card in the fixture');
    for (const kind of ['build', 'discard', 'wonder'] as const) {
      const next = { events: [ev({ kind, seat: 1, card: faceUp.card, wonder: 'pyramids' })] };
      expect(botPick(v, next, 1)).toEqual({ slot: faceUp.index });
    }
  });

  it('names the wonder the bot drafts', () => {
    const next = { events: [ev({ kind: 'draft', seat: 1, wonder: 'colossus' })] };
    expect(botPick(v, next, 1)).toEqual({ wonder: 'colossus' });
  });

  it('ignores the other seat, other events and cards not on screen', () => {
    if (!faceUp?.card) throw new Error('no face-up card in the fixture');
    expect(botPick(v, { events: [ev({ kind: 'build', seat: 0, card: faceUp.card })] }, 1)).toBe(
      undefined
    );
    expect(botPick(v, { events: [ev({ kind: 'token', seat: 1, token: 'law' })] }, 1)).toBe(
      undefined
    );
    expect(botPick(v, { events: [ev({ kind: 'build', seat: 1, card: 'no-such-card' })] }, 1)).toBe(
      undefined
    );
    expect(botPick(null, { events: [] }, 1)).toBe(undefined);
    expect(botPick(v, null, 1)).toBe(undefined);
    expect(botPick(v, { events: [] }, undefined)).toBe(undefined);
  });
});
