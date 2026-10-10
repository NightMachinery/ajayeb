import { describe, expect, it } from 'vitest';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import type { GameEvent, State } from '$lib/wire';
import { formatLogEvent, groupLogByAge } from './log-format';

const names = ['Sara', 'Omid'] as const;

function state(name: (typeof fixtureNames)[number]): State {
  const frame = fixtures[fixtureNames.indexOf(name)];
  if (!frame) throw new Error(`no fixture named ${name}`);
  if (frame.t !== 'state') throw new Error(`fixture ${name} is not a state frame`);
  return frame;
}

/** The i-th event of a real fixture's list, or a thrown error: every call
 *  site below already asserts the list's length first. */
function at(events: readonly GameEvent[], i: number): GameEvent {
  const event = events[i];
  if (!event) throw new Error(`no event at index ${i}`);
  return event;
}

describe('formatLogEvent: real recorded events', () => {
  it('draft names the drafting player and the catalog wonder name', () => {
    const event = at(state('state-ai-thinking').events, 0);
    expect(event.kind).toBe('draft');
    const line = formatLogEvent(event, names);
    expect(line?.seat).toBe(0);
    expect(line?.text).toBe('Sara drafted Circus Maximus.');
  });

  it('build, pay, pawn and gameOver from a real end-of-game sequence', () => {
    const events = state('state-game-over').events;
    expect(events.map((e) => e.kind)).toEqual(['build', 'pay', 'pawn', 'gameOver']);

    const build = at(events, 0);
    const pay = at(events, 1);
    const pawn = at(events, 2);
    const gameOver = at(events, 3);
    expect(formatLogEvent(build, names)).toEqual({
      kind: 'build',
      seat: 0,
      text: 'Sara built Walls.'
    });
    // Mechanical kinds that another line already covers are left out.
    expect(formatLogEvent(pay, names)).toBeNull();
    expect(formatLogEvent(gameOver, names)).toBeNull();

    expect(pawn.from).toBe(7);
    expect(pawn.to).toBe(9);
    expect(formatLogEvent(pawn, names)).toEqual({
      kind: 'pawn',
      text: 'The military pawn moves toward Moon.'
    });
  });

  it('build and a science pair from a real sequence', () => {
    const pendingTokenEvents = state('state-pending-token').events;
    const build = at(pendingTokenEvents, 0);
    const pair = at(pendingTokenEvents, 1);
    expect(formatLogEvent(build, names)).toEqual({
      kind: 'build',
      seat: 0,
      text: 'Sara built Library.'
    });
    expect(pair.symbol).toBe('quill');
    expect(formatLogEvent(pair, names)).toEqual({
      kind: 'pair',
      seat: 0,
      text: 'Sara completes a pair of Quill symbols.'
    });
  });

  it('build and a reveal from a real sequence; reveal is purely visual', () => {
    const playEvents = state('state-play').events;
    const build = at(playEvents, 0);
    const reveal = at(playEvents, 1);
    expect(formatLogEvent(build, names)).toEqual({
      kind: 'build',
      seat: 1,
      text: 'Omid built Clay Pool.'
    });
    expect(formatLogEvent(reveal, names)).toBeNull();
  });
});

describe('formatLogEvent: kinds with no recorded fixture', () => {
  // These GameEvent kinds never happen to appear in the five recorded state
  // fixtures (a draft-round reveal, a discard, a wonder, a loot, a token
  // pick, a destroy and a chosen starter), so each is built directly as a
  // typed GameEvent rather than drawn from one, the same way threats.test.ts
  // builds one ScienceView for a boundary case no fixture reaches.
  function event(partial: Partial<GameEvent> & { kind: GameEvent['kind'] }): GameEvent {
    return { private: false, ...partial };
  }

  it('discard', () => {
    const line = formatLogEvent(
      event({ kind: 'discard', seat: 1, card: 'baths', amount: 3 }),
      names
    );
    expect(line?.text).toBe('Omid discarded Baths for 3 coins.');
  });

  it('wonder, with the card tucked beneath it', () => {
    const line = formatLogEvent(
      event({ kind: 'wonder', seat: 0, wonder: 'pyramids', card: 'baths' }),
      names
    );
    expect(line?.text).toBe('Sara built the wonder Pyramids, using Baths.');
  });

  it('loot', () => {
    const line = formatLogEvent(event({ kind: 'loot', seat: 1, amount: 5, value: 5 }), names);
    expect(line?.text).toBe('Omid loses 5 coins to looting.');
  });

  it('token', () => {
    const line = formatLogEvent(event({ kind: 'token', seat: 0, token: 'agriculture' }), names);
    expect(line?.text).toBe('Sara takes the Agriculture token.');
  });

  it('destroy and rebuild', () => {
    expect(formatLogEvent(event({ kind: 'destroy', seat: 0, card: 'quarry' }), names)?.text).toBe(
      'Sara destroys Quarry.'
    );
    expect(formatLogEvent(event({ kind: 'rebuild', seat: 1, card: 'quarry' }), names)?.text).toBe(
      'Omid builds Quarry from the discard pile.'
    );
  });

  it('starterChosen names both the chooser and the seat chosen', () => {
    const line = formatLogEvent(event({ kind: 'starterChosen', seat: 1, starter: 0 }), names);
    expect(line?.text).toBe('Omid chooses Sara to start the next age.');
  });

  it('draftRound only logs the second round', () => {
    expect(formatLogEvent(event({ kind: 'draftRound', round: 1 }), names)).toBeNull();
    expect(formatLogEvent(event({ kind: 'draftRound', round: 2 }), names)?.text).toBe(
      'The second round of wonders is revealed.'
    );
  });

  it('coins is left out: mechanical, implied by the move that caused it', () => {
    expect(
      formatLogEvent(event({ kind: 'coins', seat: 0, amount: 6, balance: 9 }), names)
    ).toBeNull();
  });
});

describe('groupLogByAge', () => {
  it('puts pre-draft events in age 0 and starts a new group at each ageStart', () => {
    const events: GameEvent[] = [
      { kind: 'draft', private: false, seat: 0, wonder: 'pyramids' },
      { kind: 'ageStart', private: false, age: 1 },
      { kind: 'build', private: false, seat: 0, card: 'lumber-yard' },
      { kind: 'ageStart', private: false, age: 2 },
      { kind: 'build', private: false, seat: 1, card: 'brewery' }
    ];
    const groups = groupLogByAge(events, names);
    expect(groups.map((g) => g.age)).toEqual([0, 1, 2]);
    expect(groups[0]!.lines).toHaveLength(1);
    expect(groups[1]!.lines.map((l) => l.kind)).toEqual(['ageStart', 'build']);
    expect(groups[2]!.lines.map((l) => l.kind)).toEqual(['ageStart', 'build']);
  });

  it('drops a group left empty once its events all format to null', () => {
    const events: GameEvent[] = [
      { kind: 'draft', private: false, seat: 0, wonder: 'pyramids' },
      { kind: 'ageStart', private: false, age: 1 },
      { kind: 'reveal', private: false, card: 'baths', slot: 2 },
      { kind: 'coins', private: false, seat: 0, amount: 2, balance: 9 }
    ];
    // Age 1 here has only events that format to null (reveal, coins), so the
    // real recorded events above already prove ageStart itself always
    // survives; this isolates that an otherwise-silent age is left out.
    const withoutAgeStart = events.filter((e) => e.kind !== 'ageStart');
    const groups = groupLogByAge(withoutAgeStart, names);
    expect(groups).toHaveLength(1);
    expect(groups[0]!.age).toBe(0);
  });
});

describe('formatLogEvent: a skipped choice says why', () => {
  // The Statue of Zeus with no brown card to destroy, as the server recorded
  // it (server/internal/engine/testdata/protocol/effect-skipped.json).
  const zeus: GameEvent = {
    kind: 'effectSkipped',
    private: false,
    seat: 0,
    wonder: 'statue-of-zeus',
    skipped: 'destroyCard'
  };

  it('nothing to destroy names the wonder and the opponent', () => {
    expect(formatLogEvent(zeus, names)?.text).toBe(
      'Nothing to destroy for Statue of Zeus: Omid has no card of that colour.'
    );
  });

  it('an empty discard pile, an empty board and an empty box', () => {
    const at = (skipped: GameEvent['skipped'], wonder?: string) =>
      formatLogEvent({ kind: 'effectSkipped', private: false, seat: 1, wonder, skipped }, names)
        ?.text;
    expect(at('buildFromDiscard', 'mausoleum')).toBe(
      'Nothing to build for Mausoleum: the discard pile is empty.'
    );
    expect(at('pickBoardToken')).toBe('No progress token is left on the board to take.');
    expect(at('pickLibraryToken', 'great-library')).toBe(
      'Nothing to draw for Great Library: no progress token is left in the box.'
    );
  });

  it('falls back to the plain line when the event names no source', () => {
    expect(
      formatLogEvent(
        { kind: 'effectSkipped', private: false, seat: 0, skipped: 'destroyCard' },
        names
      )?.text
    ).toBe('Sara has no option for that choice.');
  });
});
