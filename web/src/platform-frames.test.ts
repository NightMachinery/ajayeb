// Compile the new generated discriminated unions independently of Duel's entry.
import { expect, it } from 'vitest';
import type { PlatformClientMessage, PlatformServerMessage } from './lib/wire/platform.gen';

const requests = [
  {
    t: 'table-pick',
    code: 'REGAAA',
    seq: 1,
    point: { id: 'age-1-turn-1', revision: 0 },
    key: 'sell:clay-pool',
    confirm: false
  },
  { t: 'table-undo', code: 'REGAAA', seq: 2, base: 0 },
  { t: 'table-seat', code: 'REGAAA', seq: 3, seat: 6, control: 'bot', bot: 'easy' },
  { t: 'table-role', code: 'REGAAA', seq: 4, op: 'step-down' },
  { t: 'table-resign', code: 'REGAAA', seq: 5, confirm: true },
  { t: 'lobby-create', seatCount: 3, options: {} },
  { t: 'lobby-edit', code: 'REGAAA', seq: 6, op: 'open-reservation', seat: 2 },
  { t: 'lobby-role', code: 'REGAAA', seq: 7, op: 'promote', userId: 'spectator' },
  { t: 'lobby-start', code: 'REGAAA', seq: 8, fillBots: false },
  { t: 'link', code: 'REGAAA', seat: 6 },
  { t: 'relink', code: 'REGAAA', seat: 6 }
] satisfies PlatformClientMessage[];

const snapshots = [
  {
    t: 'table-state',
    game: 'haftgane',
    code: 'REGAAA',
    revision: 0,
    batch: 0,
    status: 'playing',
    seats: [],
    humans: [],
    you: { userId: 'spectator' },
    startingHumanCount: 1,
    allowUndo: true,
    undoOff: false,
    finalReveal: true,
    events: [],
    roomLog: [],
    view: { phase: 'turn', age: 1, turn: 1, pass: 'left', pile: 0, seats: [] }
  },
  {
    t: 'lobby-state',
    game: 'lobby',
    code: 'REGAAA',
    revision: 0,
    seatCount: 3,
    options: {},
    seats: [],
    humans: [],
    you: { userId: 'spectator' },
    canStart: false,
    roomLog: []
  },
  { t: 'table-reject', seq: 1, code: 'decision-closed' },
  { t: 'lobby-reject', seq: 2, code: 'not-admin' },
  { t: 'linked', code: 'REGAAA', seat: 6, claim: 'private' }
] satisfies PlatformServerMessage[];

it('keeps the generated platform families distinct from Duel state and action', () => {
  expect(requests.map((frame) => frame.t)).toEqual([
    'table-pick',
    'table-undo',
    'table-seat',
    'table-role',
    'table-resign',
    'lobby-create',
    'lobby-edit',
    'lobby-role',
    'lobby-start',
    'link',
    'relink'
  ]);
  expect(snapshots.map((frame) => frame.t)).toEqual([
    'table-state',
    'lobby-state',
    'table-reject',
    'lobby-reject',
    'linked'
  ]);
});
