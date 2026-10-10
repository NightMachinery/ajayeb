import { describe, expect, it } from 'vitest';
import {
  fixtureNames,
  fixtures,
  protocolFixtureNames,
  protocolFixtures
} from '$lib/wire/fixtures.gen';
import type { State } from '$lib/wire';
import type { View } from '$lib/wire';
import { turnState, type TurnContext } from './turn';

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

  it('playAgain when the view says the active seat plays again', () => {
    // No recorded fixture happens to pause with extraTurn set (a real game
    // keeps playing through it), so this one field is overridden on a real
    // view rather than drawn from a fixture as-is.
    const v: View = { ...view('age-1-start-seat0'), extraTurn: true };
    const ctx: TurnContext = { view: v, mode: 'ai', mySeats: [0], ...baseConn };
    expect(turnState(ctx)).toEqual({ kind: 'playAgain', seat: 0 });
  });
});
