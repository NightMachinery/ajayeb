import { describe, expect, it } from 'vitest';
import {
  fixtureNames,
  fixtures,
  protocolFixtureNames,
  protocolFixtures
} from '$lib/wire/fixtures.gen';
import type { State } from '$lib/wire';
import type { GameEvent, View } from '$lib/wire';
import { playAgainFor, playAgainFrom, turnState, type TurnContext } from './turn';

function state(name: (typeof fixtureNames)[number]): State {
  const frame = fixtures[fixtureNames.indexOf(name)];
  if (!frame) throw new Error(`no fixture named ${name}`);
  if (frame.t !== 'state') throw new Error(`fixture ${name} is not a state frame`);
  return frame;
}

function view(name: (typeof protocolFixtureNames)[number]): View {
  const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!v) throw new Error(`no protocol fixture named ${name}`);
  return v;
}

const baseConn = { conn: 'open', pending: false, slow: false } as const;

describe('turnState: connection takes priority over everything else', () => {
  it('reconnecting wins even with no view yet', () => {
    expect(
      turnState({
        view: null,
        mode: 'online',
        mySeats: [0],
        conn: 'reconnecting',
        pending: false,
        slow: false
      })
    ).toEqual({ kind: 'reconnecting' });
  });

  it('offline wins even with a view and a legal turn', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-seat0'),
      mode: 'ai',
      mySeats: [0],
      conn: 'offline',
      pending: false,
      slow: false
    };
    expect(turnState(ctx)).toEqual({ kind: 'offline' });
  });

  it('a pending request wins over a fresh turn, with the slow flag passed through', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-seat0'),
      mode: 'ai',
      mySeats: [0],
      ...baseConn,
      pending: true,
      slow: false
    };
    expect(turnState(ctx)).toEqual({ kind: 'sent', slow: false });
    expect(turnState({ ...ctx, slow: true })).toEqual({ kind: 'sent', slow: true });
  });

  it('connecting before the first view arrives', () => {
    expect(turnState({ view: null, mode: 'online', mySeats: [0], ...baseConn })).toEqual({
      kind: 'connecting'
    });
  });
});

describe('turnState: real recorded states', () => {
  it('state-game-over is gameOver regardless of mode or seats', () => {
    const f = state('state-game-over');
    const ctx: TurnContext = {
      view: f.view,
      mode: f.room.mode,
      mySeats: f.you.seats as (0 | 1)[],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'gameOver' });
  });

  it('state-ai-thinking reports the thinking seat even though toMove is the bot', () => {
    const f = state('state-ai-thinking');
    const ctx: TurnContext = {
      view: f.view,
      mode: f.room.mode,
      mySeats: f.you.seats as (0 | 1)[],
      thinking: f.thinking as 0 | 1,
      ...baseConn
    };
    expect(f.thinking).toBe(1);
    expect(turnState(ctx)).toEqual({ kind: 'aiThinking', seat: 1 });
  });

  it('state-draft is a hot-seat turn for the seat to move', () => {
    const f = state('state-draft');
    const ctx: TurnContext = {
      view: f.view,
      mode: f.room.mode,
      mySeats: f.you.seats as (0 | 1)[],
      ...baseConn
    };
    expect(f.room.mode).toBe('hotseat');
    expect(turnState(ctx)).toEqual({ kind: 'hotseatTurn', seat: 0 });
  });

  it('state-play is also a hot-seat turn', () => {
    const f = state('state-play');
    const ctx: TurnContext = {
      view: f.view,
      mode: f.room.mode,
      mySeats: f.you.seats as (0 | 1)[],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'hotseatTurn', seat: 0 });
  });

  it('state-pending-token is an online pending choice for the deciding seat', () => {
    const f = state('state-pending-token');
    const ctx: TurnContext = {
      view: f.view,
      mode: f.room.mode,
      mySeats: f.you.seats as (0 | 1)[],
      ...baseConn
    };
    expect(f.room.mode).toBe('online');
    expect(turnState(ctx)).toEqual({
      kind: 'pendingChoice',
      seat: 0,
      pendingKind: 'pickBoardToken'
    });
  });
});

describe('turnState: non-hot-seat turns, from real protocol fixtures', () => {
  it('yourTurn in an ai room when it is your own seat to move', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-seat0'),
      mode: 'ai',
      mySeats: [0],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'yourTurn', seat: 0 });
  });

  it('waitingOpponent in an online room when the other seat is to move', () => {
    const v = view('card-destroyed-seat0');
    expect(v.toMove).toBe(1);
    const ctx: TurnContext = { view: v, mode: 'online', mySeats: [0], ...baseConn };
    expect(turnState(ctx)).toEqual({ kind: 'waitingOpponent', seat: 1 });
  });

  it('spectating when this connection holds no seat', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-spectator'),
      mode: 'online',
      mySeats: [],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'spectating' });
  });

  it('pendingChoice for a chooseStarter and a destroyCard pending view', () => {
    const starter: TurnContext = {
      view: view('choose-starter-seat0'),
      mode: 'ai',
      mySeats: [0],
      ...baseConn
    };
    expect(turnState(starter)).toEqual({
      kind: 'pendingChoice',
      seat: 0,
      pendingKind: 'chooseStarter'
    });

    const destroy: TurnContext = {
      view: view('destroy-card-seat0'),
      mode: 'ai',
      mySeats: [0],
      ...baseConn
    };
    expect(turnState(destroy)).toEqual({
      kind: 'pendingChoice',
      seat: 0,
      pendingKind: 'destroyCard'
    });
  });

  it('playAgain for the seat taking an extra turn, with the wonder that granted it', () => {
    const v = view('age-1-start-seat0');
    const ctx: TurnContext = {
      view: v,
      mode: 'ai',
      mySeats: [0],
      ...baseConn,
      playAgain: { seat: 0, wonder: 'appian-way' }
    };
    expect(turnState(ctx)).toEqual({
      kind: 'playAgain',
      seat: 0,
      wonder: 'appian-way',
      hotseat: false
    });
    expect(turnState({ ...ctx, mode: 'hotseat', mySeats: [0, 1] })).toEqual({
      kind: 'playAgain',
      seat: 0,
      wonder: 'appian-way',
      hotseat: true
    });
  });

  it('an extra turn of the other seat, or the view flag alone, is no playAgain', () => {
    const v = view('age-1-start-seat0');
    const base: TurnContext = { view: v, mode: 'hotseat', mySeats: [0, 1], ...baseConn };
    expect(turnState({ ...base, playAgain: { seat: 1 } })).toEqual({
      kind: 'hotseatTurn',
      seat: 0
    });
    // view.extraTurn means "plays again once this turn is over"; on its own
    // it never says the current turn is the extra one.
    expect(turnState({ ...base, view: { ...v, extraTurn: true } })).toEqual({
      kind: 'hotseatTurn',
      seat: 0
    });
  });

  it('a pending choice wins over the extra turn', () => {
    const ctx: TurnContext = {
      view: view('destroy-card-seat0'),
      mode: 'ai',
      mySeats: [0],
      ...baseConn,
      playAgain: { seat: 0 }
    };
    expect(turnState(ctx)).toEqual({ kind: 'pendingChoice', seat: 0, pendingKind: 'destroyCard' });
  });
});

describe('turnState: the bot thinking', () => {
  const thinkingFrame = () => state('state-ai-thinking');

  it('reads the thinking seat from the frame, whatever the view says', () => {
    const f = thinkingFrame();
    expect(f.room.mode).toBe('ai');
    const ctx: TurnContext = {
      view: f.view,
      mode: 'ai',
      mySeats: [0],
      thinking: 1,
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'aiThinking', seat: 1 });
  });

  it('a bot seat to move reads as thinking even between thinking frames', () => {
    const v = view('card-destroyed-seat0');
    expect(v.toMove).toBe(1);
    const ctx: TurnContext = { view: v, mode: 'ai', mySeats: [0], ...baseConn };
    expect(turnState(ctx)).toEqual({ kind: 'aiThinking', seat: 1 });
  });

  it('the human seat to move in a bot room is the human turn, not thinking', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-seat0'),
      mode: 'ai',
      mySeats: [0],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'yourTurn', seat: 0 });
  });

  it('the human sitting at the Moon sees the Sun bot thinking', () => {
    const v = view('age-1-start-seat1');
    expect(v.toMove).toBe(0);
    expect(turnState({ view: v, mode: 'ai', mySeats: [1], ...baseConn })).toEqual({
      kind: 'aiThinking',
      seat: 0
    });
    expect(turnState({ view: v, mode: 'ai', mySeats: [1], thinking: 0, ...baseConn })).toEqual({
      kind: 'aiThinking',
      seat: 0
    });
  });

  it('a request in flight and a dropped connection still win over thinking', () => {
    const f = thinkingFrame();
    const ctx: TurnContext = { view: f.view, mode: 'ai', mySeats: [0], thinking: 1, ...baseConn };
    expect(turnState({ ...ctx, pending: true })).toEqual({ kind: 'sent', slow: false });
    expect(turnState({ ...ctx, conn: 'reconnecting' })).toEqual({ kind: 'reconnecting' });
  });

  it('an online opponent to move is still waited for', () => {
    const v = view('card-destroyed-seat0');
    expect(turnState({ view: v, mode: 'online', mySeats: [0], ...baseConn })).toEqual({
      kind: 'waitingOpponent',
      seat: 1
    });
  });

  it('an online opponent to move who dropped is waited for as away', () => {
    const v = view('card-destroyed-seat0');
    const ctx: TurnContext = {
      view: v,
      mode: 'online',
      mySeats: [0],
      connected: [true, false],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'waitingOpponent', seat: 1, away: true });
    expect(turnState({ ...ctx, connected: [true, true] })).toEqual({
      kind: 'waitingOpponent',
      seat: 1
    });
  });

  it('an away opponent does not matter on your own turn', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-seat0'),
      mode: 'online',
      mySeats: [0],
      connected: [true, false],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'yourTurn', seat: 0 });
  });

  it('reconnecting and offline win over an away opponent', () => {
    const ctx: TurnContext = {
      view: view('card-destroyed-seat0'),
      mode: 'online',
      mySeats: [0],
      connected: [true, false],
      ...baseConn
    };
    expect(turnState({ ...ctx, conn: 'reconnecting' })).toEqual({ kind: 'reconnecting' });
    expect(turnState({ ...ctx, conn: 'offline' })).toEqual({ kind: 'offline' });
  });

  it('a spectator watches whoever is away', () => {
    const ctx: TurnContext = {
      view: view('age-1-start-spectator'),
      mode: 'online',
      mySeats: [],
      connected: [false, false],
      ...baseConn
    };
    expect(turnState(ctx)).toEqual({ kind: 'spectating' });
  });
});

describe('playAgainFrom: the extra turn, from the events seen', () => {
  // The events of the Appian Way bought by trade, as the server recorded
  // them (server/internal/engine/testdata/protocol/appian-economy.json).
  const appian: GameEvent[] = [
    { kind: 'wonder', private: false, seat: 0, card: 'rostrum', wonder: 'appian-way' },
    { kind: 'pay', private: false, seat: 0, amount: 10, toOpponent: 10 },
    { kind: 'coins', private: false, seat: 0, wonder: 'appian-way', amount: 3, balance: 8 },
    { kind: 'extraTurn', private: false, seat: 0 },
    { kind: 'coins', private: false, seat: 1, wonder: 'appian-way', amount: -3, balance: 7 },
    { kind: 'reveal', private: false, card: 'shelf-quarry', slot: 15 }
  ];
  const earlier: GameEvent[] = [
    { kind: 'ageStart', private: false, age: 1 },
    { kind: 'build', private: false, seat: 1, card: 'baths' }
  ];

  it('nothing before any extra turn', () => {
    expect(playAgainFrom([])).toBeUndefined();
    expect(playAgainFrom(earlier)).toBeUndefined();
  });

  it('the wonder that grants it starts the extra turn, named', () => {
    expect(playAgainFrom([...earlier, ...appian])).toEqual({ seat: 0, wonder: 'appian-way' });
  });

  it("the seat's next play action ends it", () => {
    for (const kind of ['build', 'discard', 'wonder'] as const) {
      const next: GameEvent = { kind, private: false, seat: 0, card: 'stable' };
      expect(playAgainFrom([...appian, next]), kind).toBeUndefined();
    }
  });

  it('a choice the wonder opened (a destroy, a token) keeps it', () => {
    const zeus: GameEvent[] = [
      { kind: 'wonder', private: false, seat: 1, card: 'rostrum', wonder: 'statue-of-zeus' },
      { kind: 'extraTurn', private: false, seat: 1 }
    ];
    const destroy: GameEvent = { kind: 'destroy', private: false, seat: 1, card: 'quarry' };
    expect(playAgainFrom([...zeus, destroy])).toEqual({ seat: 1, wonder: 'statue-of-zeus' });
  });

  it('an extra turn lost with the age, a new age or the end of the game ends it', () => {
    const lost: GameEvent = { kind: 'extraTurnLost', private: false, seat: 0 };
    const age: GameEvent = { kind: 'ageStart', private: false, age: 2 };
    expect(playAgainFrom([...appian, lost, age])).toBeUndefined();
    expect(playAgainFrom([...appian, age])).toBeUndefined();
    expect(playAgainFrom([...appian, { kind: 'gameOver', private: false }])).toBeUndefined();
  });

  it('a second extra turn in a row is its own, with its own wonder', () => {
    const sphinx: GameEvent[] = [
      { kind: 'wonder', private: false, seat: 0, card: 'stable', wonder: 'sphinx' },
      { kind: 'extraTurn', private: false, seat: 0 }
    ];
    expect(playAgainFrom([...appian, ...sphinx])).toEqual({ seat: 0, wonder: 'sphinx' });
  });

  it('an extra turn whose wonder event was not seen is still an extra turn', () => {
    expect(playAgainFrom([{ kind: 'extraTurn', private: false, seat: 1 }])).toEqual({ seat: 1 });
  });
});

describe('playAgainFor: history and server-view fallback', () => {
  const appian = view('appian-economy-seat0');
  const grant: GameEvent[] = [
    { kind: 'wonder', private: false, seat: 0, card: 'rostrum', wonder: 'appian-way' },
    { kind: 'extraTurn', private: false, seat: 0 }
  ];

  it('a reload without history still displays the extra turn from the server view', () => {
    expect(appian.playingAgain).toBe(true);
    expect(appian.extraTurn).toBe(false);
    expect(playAgainFor(appian, [])).toEqual({ seat: 0 });
    expect(
      turnState({
        view: appian,
        mode: 'hotseat',
        mySeats: [0, 1],
        ...baseConn,
        playAgain: playAgainFor(appian, [])
      })
    ).toEqual({ kind: 'playAgain', seat: 0, hotseat: true });
  });

  it('a normal reload prefers the granting wonder from history', () => {
    expect(playAgainFor(appian, grant)).toEqual({ seat: 0, wonder: 'appian-way' });
    expect(
      turnState({
        view: appian,
        mode: 'hotseat',
        mySeats: [0, 1],
        ...baseConn,
        playAgain: playAgainFor(appian, grant)
      })
    ).toEqual({ kind: 'playAgain', seat: 0, wonder: 'appian-way', hotseat: true });
  });

  it('the following ordinary move clears the extra turn', () => {
    const after = { ...appian, playingAgain: false, toMove: 1 };
    expect(
      playAgainFor(after, [...grant, { kind: 'discard', private: false, seat: 0, card: 'stable' }])
    ).toBeUndefined();
    expect(playAgainFor(after, [])).toBeUndefined();
  });

  it('requires the current-turn flag and a valid seat for the fallback', () => {
    expect(playAgainFor({ ...appian, toMove: 1 }, [])).toEqual({ seat: 1 });
    const futureExtraTurn = { ...appian, extraTurn: true, playingAgain: false };
    expect(playAgainFor(futureExtraTurn, [])).toBeUndefined();
    expect(playAgainFor({ playingAgain: true }, [])).toBeUndefined();
    expect(playAgainFor({ playingAgain: true, toMove: 2 }, [])).toBeUndefined();
    expect(playAgainFor(null, [])).toBeUndefined();
  });
});
