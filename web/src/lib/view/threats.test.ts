import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { HintFamily, MilitaryView, PlayerHintsView, ScienceView, View } from '$lib/wire';
import { HINT_FAMILIES } from '$lib/hints/families';
import {
  glowHints,
  glowWarns,
  militaryGlow,
  scienceGlow,
  strongerGlow,
  type GlowHints,
  gameEnded,
  militaryDanger,
  militaryEndangered,
  militaryLeader,
  militaryMargin,
  nextLooting,
  scienceDanger
} from './threats';

/** A real recorded view, by the scenario/viewer name a protocol fixture
 *  carries (see server/internal/engine/testdata/protocol/). */
function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no protocol fixture named ${name}`);
  return view;
}

describe('militaryLeader / militaryEndangered / militaryMargin', () => {
  it('is null/null/0 when the pawn is centred (age-1-start)', () => {
    const { military } = fixture('age-1-start-seat0');
    expect(military.position).toBe(0);
    expect(militaryLeader(military)).toBeNull();
    expect(militaryEndangered(military)).toBeNull();
    expect(militaryMargin(military)).toBe(0);
  });

  it('leads toward seat 1 when the position is positive (starter-chosen, +3)', () => {
    const { military } = fixture('starter-chosen-seat0');
    expect(military.position).toBe(3);
    expect(militaryLeader(military)).toBe(0);
    expect(militaryEndangered(military)).toBe(1);
    expect(militaryMargin(military)).toBe(3);
  });
});

describe('militaryDanger', () => {
  it('is false below the threshold even for the endangered seat (position +3)', () => {
    const { military } = fixture('starter-chosen-seat0');
    expect(militaryDanger(military, 1)).toBe(false);
    expect(militaryDanger(military, 0)).toBe(false);
  });

  it('is true for the endangered seat at the threshold (looting fixture, position +6)', () => {
    const { military } = fixture('looting-seat0');
    expect(military.position).toBe(6);
    expect(militaryDanger(military, 1)).toBe(true);
    // The leader's own capital is never the one in danger.
    expect(militaryDanger(military, 0)).toBe(false);
  });
});

describe('militaryDanger after the end', () => {
  it('is false once the pawn has reached a capital (game-over-military-hidden, position 9)', () => {
    const view = fixture('game-over-military-hidden-seat0');
    expect(view.military.position).toBe(9);
    // The condition is met: no warning, even without the end flag.
    expect(militaryDanger(view.military, 1)).toBe(false);
    expect(militaryDanger(view.military, 1, gameEnded(view.phase))).toBe(false);
  });

  it('is false at the threshold once the game has ended by the end of Age III', () => {
    // The looting fixture's pawn (position 6) in a game that has ended by
    // civilian scoring: the warning is over with the game.
    const { military } = fixture('looting-seat0');
    const over = fixture('game-over-civilian-seat0');
    expect(over.result?.victory).toBe('civilian');
    expect(militaryDanger(military, 1, gameEnded(over.phase))).toBe(false);
    expect(militaryDanger(military, 1, gameEnded('play'))).toBe(true);
  });
});

describe('gameEnded', () => {
  it('is true only in the gameOver phase', () => {
    expect(gameEnded('gameOver')).toBe(true);
    expect(gameEnded('play')).toBe(false);
    expect(gameEnded('wonderDraft')).toBe(false);
    expect(gameEnded(undefined)).toBe(false);
  });
});

describe('nextLooting', () => {
  it('is null when the pawn is centred (no direction to look ahead in)', () => {
    const { military } = fixture('age-1-start-seat0');
    expect(nextLooting(military)).toBeNull();
  });

  it('finds the nearest still-present token ahead of the pawn (position +1)', () => {
    const { military } = fixture('card-destroyed-seat0');
    expect(military.position).toBe(1);
    const next = nextLooting(military);
    expect(next).not.toBeNull();
    expect(next?.at).toBe(3);
    expect(next?.value).toBe(2);
    expect(next?.present).toBe(true);
  });

  it('skips a token already claimed and finds the next one (position +3, the one at 3 is gone)', () => {
    const { military } = fixture('starter-chosen-seat0');
    const claimed = military.looting.find((l) => l.at === 3);
    expect(claimed?.present).toBe(false);
    const next = nextLooting(military);
    expect(next?.at).toBe(6);
    expect(next?.value).toBe(5);
  });

  it('is null once every token ahead of the pawn is already claimed (position +6)', () => {
    const { military } = fixture('looting-seat0');
    expect(nextLooting(military)).toBeNull();
  });
});

describe('scienceDanger', () => {
  it('is false with no symbols (age-1-start)', () => {
    expect(scienceDanger(fixture('age-1-start-seat0').players[0].science)).toBe(false);
  });

  it('is false with a single distinct symbol (library-token-taken)', () => {
    const { science } = fixture('library-token-taken-seat0').players[0];
    expect(science.distinct).toBe(1);
    expect(scienceDanger(science)).toBe(false);
  });

  it('is false once the scientific victory has happened (game-over-science, 6 distinct)', () => {
    const view = fixture('game-over-science-seat0');
    const { science } = view.players[0];
    expect(science.distinct).toBe(6);
    // The condition is met: no warning, whether or not the caller passes the end.
    expect(scienceDanger(science)).toBe(false);
    expect(scienceDanger(science, gameEnded(view.phase))).toBe(false);
  });

  it('is false at 5 distinct once the game has ended some other way', () => {
    expect(scienceDanger({ symbols: [], distinct: 5 }, true)).toBe(false);
  });

  it('is true exactly at the danger threshold (5 distinct, the "one pair away" case)', () => {
    // No recorded fixture stops at exactly 5 (a real game passes straight
    // through to 6, which ends it), so this boundary value is constructed
    // directly rather than drawn from a fixture.
    expect(scienceDanger({ symbols: [], distinct: 5 })).toBe(true);
    expect(scienceDanger({ symbols: [], distinct: 4 })).toBe(false);
  });
});

// --- Glow levels from the board hints (M10.HS4) ------------------------------

const EVERY: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);

function ph(over: Partial<PlayerHintsView> = {}): PlayerHintsView {
  return {
    coinsToNextVp: 3,
    symbols: [],
    need: 6,
    reachable: true,
    lawWins: false,
    chainTargets: [],
    guildPoints: [],
    toCapital: 9,
    capitalSafe: false,
    threats: [],
    ...over
  };
}

function track(position: number): MilitaryView {
  return { position, looting: [] } as unknown as MilitaryView;
}

function sci(distinct: number): ScienceView {
  return { distinct, symbols: [] } as unknown as ScienceView;
}

function gh(a: PlayerHintsView, b: PlayerHintsView, families = EVERY): GlowHints {
  return { families, hints: [a, b] };
}

describe('militaryGlow', () => {
  it('falls back to the margin rule without hints', () => {
    expect(militaryGlow(track(-6), 0)).toBe('soft');
    expect(militaryGlow(track(-5), 0)).toBe('none');
    expect(militaryGlow(track(-6), 0, true)).toBe('none');
  });

  it('is soft when the opponent is close, calm when it can no longer get there', () => {
    expect(militaryGlow(track(-6), 0, false, gh(ph({ toCapital: 3 }), ph()))).toBe('soft');
    expect(
      militaryGlow(track(-6), 0, false, gh(ph({ toCapital: 3, capitalSafe: true }), ph()))
    ).toBe('calm');
    expect(militaryGlow(track(-4), 0, false, gh(ph({ toCapital: 5 }), ph()))).toBe('none');
  });

  it('is strong when the opponent can take the capital with one action now', () => {
    const opp = ph({ threats: [{ seat: 1, card: 'arsenal', kind: 'military' }] });
    expect(militaryGlow(track(-7), 0, false, gh(ph({ toCapital: 2 }), opp))).toBe('strong');
    // A science win of the opponent is not a military glow.
    const sciOpp = ph({ threats: [{ seat: 1, card: 'library', kind: 'science' }] });
    expect(militaryGlow(track(-7), 0, false, gh(ph({ toCapital: 2 }), sciOpp))).toBe('soft');
  });

  it('follows the families this screen shows', () => {
    const opp = ph({ threats: [{ seat: 1, card: 'arsenal', kind: 'military' }] });
    const noThreats = new Set<HintFamily>(['military']);
    expect(militaryGlow(track(-7), 0, false, gh(ph({ toCapital: 2 }), opp, noThreats))).toBe(
      'soft'
    );
    const none = new Set<HintFamily>();
    // No military hints: the margin rule.
    expect(
      militaryGlow(track(-6), 0, false, gh(ph({ toCapital: 3, capitalSafe: true }), opp, none))
    ).toBe('soft');
  });
});

describe('scienceGlow', () => {
  it('warns at 5 different symbols without hints', () => {
    expect(scienceGlow(sci(5), 0)).toBe('soft');
    expect(scienceGlow(sci(4), 0)).toBe('none');
    expect(scienceGlow(sci(6), 0)).toBe('none');
  });

  it('is calm when the missing symbols can no longer come', () => {
    expect(scienceGlow(sci(5), 0, false, gh(ph({ need: 1, reachable: false }), ph()))).toBe('calm');
    expect(scienceGlow(sci(5), 0, false, gh(ph({ need: 1, reachable: true }), ph()))).toBe('soft');
  });

  it('is strong when the seat can win by science with one action now', () => {
    const own = ph({ need: 1, threats: [{ seat: 0, card: 'library', kind: 'science' }] });
    expect(scienceGlow(sci(5), 0, false, gh(own, ph()))).toBe('strong');
    expect(scienceGlow(sci(5), 0, true, gh(own, ph()))).toBe('none');
  });
});

describe('glow helpers', () => {
  it('orders and classifies the levels', () => {
    expect(strongerGlow('calm', 'soft')).toBe('soft');
    expect(strongerGlow('strong', 'none')).toBe('strong');
    expect(glowWarns('calm')).toBe(false);
    expect(glowWarns('soft')).toBe(true);
  });

  it('has no inputs at game over or with the board marks hidden', () => {
    const players = [{ hints: ph() }, { hints: ph() }] as never;
    expect(
      glowHints({ players, result: undefined }, { families: EVERY, boardMarks: true })
    ).toBeDefined();
    expect(
      glowHints({ players, result: undefined }, { families: EVERY, boardMarks: false })
    ).toBeUndefined();
    expect(
      glowHints({ players, result: {} as never }, { families: EVERY, boardMarks: true })
    ).toBeUndefined();
  });
});
