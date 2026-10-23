// Synthetic protocol-2 frames for the Session and Presenter tests: a
// three-seat table where seat 0 is this tab's user ("u0"), seat 1 another
// human ("u1"), seat 2 a bot, and "w" a spectator. Each helper returns a
// complete frame that frames.ts accepts; overrides replace fields whole.

import type { LobbyState, TableChoice, TableState } from '$lib/wire/platform.gen';

export const CODE = 'TBLAAA';

export function choice(design: string, left = 0, right = 0): TableChoice {
  return { key: `build:${design}:${left}:${right}`, kind: 'build', design, left, right };
}

export const CHOICES = [choice('amber'), choice('basalt'), choice('cedar')];

/** A table frame as seat `seat` (or a spectator, for null) receives it. */
export function tableFrame(
  overrides: Partial<TableState> = {},
  seat: number | null = 0
): TableState {
  const you = seat === null ? { userId: 'w' } : { userId: `u${seat}`, seat };
  return {
    t: 'table-state',
    game: 'haftgane',
    code: CODE,
    revision: 1 << 20,
    batch: 0,
    status: 'playing',
    seats: [
      { seat: 0, kind: 'human', userId: 'u0' },
      { seat: 1, kind: 'human', userId: 'u1' },
      { seat: 2, kind: 'bot', bot: 'easy' }
    ],
    humans: [
      {
        userId: 'u0',
        name: 'Ana',
        nameNo: 1,
        connected: true,
        creator: true,
        admin: true,
        temporary: false
      },
      {
        userId: 'u1',
        name: 'Bo',
        nameNo: 1,
        connected: true,
        creator: false,
        admin: false,
        temporary: false
      },
      {
        userId: 'w',
        name: 'Wen',
        nameNo: 1,
        connected: true,
        creator: false,
        admin: false,
        temporary: false
      }
    ],
    you,
    startingHumanCount: 2,
    allowUndo: false,
    undoOff: false,
    finalReveal: true,
    point: {
      ref: { id: 'age-1-turn-1', revision: 1 },
      kind: 'play',
      required: [0, 1, 2],
      confirmed: []
    },
    events: [],
    roomLog: [],
    view: {
      phase: 'play',
      age: 1,
      turn: 1,
      pass: 'left',
      pile: 0,
      seats: [0, 1, 2].map((s) => ({
        seat: s,
        board: 'rhodos',
        coins: 3,
        built: [],
        stages: 0,
        handSize: 3,
        hasKept: false
      })),
      ...(seat === null
        ? {}
        : {
            you: {
              seat,
              hand: CHOICES.map((c) => c.design),
              kept: '',
              tucked: [],
              choices: CHOICES,
              offer: [],
              seen: []
            }
          })
    },
    ...overrides
  };
}

export function lobbyFrame(overrides: Partial<LobbyState> = {}): LobbyState {
  return {
    t: 'lobby-state',
    game: 'lobby',
    code: CODE,
    revision: 1,
    seatCount: 3,
    options: {},
    seats: [
      { seat: 0, kind: 'human', userId: 'u0' },
      { seat: 1, kind: 'human', userId: 'u1' },
      { seat: 2, kind: 'open' }
    ],
    humans: [
      {
        userId: 'u0',
        name: 'Ana',
        nameNo: 1,
        connected: true,
        creator: true,
        admin: true,
        temporary: false
      },
      {
        userId: 'u1',
        name: 'Bo',
        nameNo: 1,
        connected: true,
        creator: false,
        admin: false,
        temporary: false
      }
    ],
    you: { userId: 'u0', seat: 0 },
    canStart: true,
    roomLog: [],
    ...overrides
  };
}
