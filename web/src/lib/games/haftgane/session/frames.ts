// The shapes of the protocol-2 frames Haftgane's Session reads
// (docs/protocol.md, "Platform snapshots and rejects"). The bootstrap keeps
// a snapshot as `unknown` until these guards accept it; the Session runs
// every later frame through them too, so nothing typed as a TableState is
// anything else. They check types and required fields, and tolerate
// additive fields, as the Duel client's frame parsing does.

import type { Created } from '$lib/wire';
import type {
  LobbyReject,
  LobbyState,
  PlatformLinked,
  TableReject,
  TableState
} from '$lib/wire/platform.gen';

type Guard = (value: unknown) => boolean;

function object(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}
const str: Guard = (v) => typeof v === 'string';
const bool: Guard = (v) => typeof v === 'boolean';
const int: Guard = (v) => Number.isSafeInteger(v);
const count: Guard = (v) => Number.isSafeInteger(v) && (v as number) >= 0;
const seat: Guard = (v) => Number.isSafeInteger(v) && (v as number) >= 0 && (v as number) <= 6;
const list =
  (item: Guard): Guard =>
  (v) =>
    Array.isArray(v) && v.every(item);
const oneOf =
  (...values: readonly unknown[]): Guard =>
  (v) =>
    values.includes(v);

/** Checks an object's required and optional fields; others may be present. */
function shape(required: Record<string, Guard>, optional: Record<string, Guard> = {}): Guard {
  return (value) => {
    const o = object(value);
    if (!o) return false;
    for (const [k, g] of Object.entries(required)) if (!g(o[k])) return false;
    for (const [k, g] of Object.entries(optional)) if (o[k] !== undefined && !g(o[k])) return false;
    return true;
  };
}

const platformSeat = shape(
  { seat: seat, kind: oneOf('open', 'reserved', 'human', 'bot') },
  { userId: str, bot: str }
);
const human = shape(
  {
    userId: str,
    name: str,
    nameNo: count,
    connected: bool,
    creator: bool,
    admin: bool,
    temporary: bool
  },
  { promotedBy: str }
);
const you = shape(
  { userId: str },
  { seat: seat, previousSeats: list(shape({ userId: str, seat: seat })) }
);
const logEntry = shape(
  { kind: oneOf('seat', 'role', 'resign', 'link'), at: count, by: str },
  {
    userId: str,
    seat: seat,
    control: oneOf('human', 'bot'),
    role: oneOf('admin', 'temporary', 'human')
  }
);
const ack = shape({ clientId: str, seq: count });
const pointRef = shape({ id: str, revision: count });

const score = shape({
  wonder: int,
  treasure: int,
  military: int,
  blue: int,
  yellow: int,
  green: int,
  purple: int,
  total: int
});
const tablePlay = shape({
  kind: str,
  design: str,
  stage: int,
  left: int,
  right: int,
  printed: int
});
const city = shape(
  {
    seat,
    board: str,
    side: str,
    coins: int,
    built: list(str),
    stages: int,
    handSize: count,
    hasKept: bool,
    victories: list(int),
    defeats: count,
    shields: count,
    relative: int,
    relation: str,
    score,
    science: shape({ counts: list(int), choices: list(int), points: int })
  },
  { last: tablePlay }
);
const choice = shape({
  key: str,
  kind: str,
  design: str,
  side: str,
  stage: int,
  left: int,
  right: int,
  printed: int,
  total: int,
  gain: int,
  free: str,
  witness: list(shape({ resource: str, source: str, producer: int, price: int }))
});
const seen = shape({ point: str, hand: list(str), played: str, action: str });
const bounds = shape({ certain: count, atMost: count });
const privateView = shape({
  seat,
  hand: list(str),
  kept: str,
  tucked: list(str),
  choices: list(choice),
  disabled: list(
    shape({ kind: str, design: str, stage: int, reason: str, cheapest: int, shortBy: int })
  ),
  offer: list(shape({ design: str, count, legal: bool })),
  notebook: shape({
    seen: list(seen),
    pileLooks: list(shape({ point: str, cards: list(shape({ design: str, count })) })),
    statistics: shape({
      circulating: list(shape({ colour: str, bounds })),
      returning: list(shape({ design: str, bounds })),
      neighbourPicks: list(shape({ point: str, seat, design: str }))
    })
  }),
  neighbours: list(
    shape({
      seat,
      direction: str,
      outlook: int,
      units: list(
        shape({
          choice: bool,
          producer: int,
          count,
          options: list(
            shape({ resource: str, price: int }, { discount: shape({ design: str, stage: int }) })
          )
        })
      )
    })
  )
});
const result = shape({ scores: list(score), rank: list(int), winners: list(seat) });
const reveal = shape({
  tucked: list(list(str)),
  pile: list(str),
  box: list(str),
  leftovers: list(shape({ age: count, seat, design: str, kept: bool }))
});
const tableView = shape(
  { phase: str, age: int, turn: int, pass: str, pile: count, seats: list(city), canDownload: bool },
  { you: privateView, result, reveal }
);
const tableEvent = shape({
  kind: str,
  seat: int,
  other: int,
  age: int,
  turn: int,
  play: str,
  design: str,
  stage: int,
  side: str,
  left: int,
  right: int,
  printed: int,
  delta: int,
  source: str,
  outcome: str,
  pass: str
});
const historyEntry = shape({ batch: count, events: list(tableEvent) });
const tablePoint = shape({ ref: pointRef, kind: str, required: list(seat), confirmed: list(seat) });

const tableState = shape(
  {
    t: oneOf('table-state'),
    game: oneOf('haftgane'),
    code: str,
    revision: count,
    batch: count,
    status: oneOf('playing', 'finished'),
    seats: list(platformSeat),
    humans: list(human),
    you,
    startingHumanCount: count,
    allowUndo: bool,
    undoOff: bool,
    finalReveal: bool,
    events: list(tableEvent),
    roomLog: list(logEntry),
    view: tableView
  },
  {
    point: tablePoint,
    pending: shape({ key: str, confirm: bool }),
    ack,
    history: list(historyEntry)
  }
);

const lobbyOptions = shape(
  {},
  {
    seating: oneOf('random', 'lobby'),
    sides: oneOf('choose', 'day', 'random'),
    firstPlayer: oneOf(0, 1),
    allowUndo: bool,
    finalReveal: bool
  }
);
const lobbyState = shape(
  {
    t: oneOf('lobby-state'),
    game: oneOf('lobby'),
    code: str,
    revision: count,
    seatCount: count,
    options: lobbyOptions,
    seats: list(platformSeat),
    humans: list(human),
    you,
    canStart: bool,
    roomLog: list(logEntry)
  },
  { ack }
);

const TABLE_REJECTS = [
  'decision-closed',
  'illegal',
  'stale',
  'bad-request',
  'not-admin',
  'forbidden',
  'rate',
  'storage'
] as const;
const LOBBY_REJECTS = [
  'bad-request',
  'stale',
  'not-admin',
  'forbidden',
  'rate',
  'storage'
] as const;

const tableReject = shape({ t: oneOf('table-reject'), seq: count, code: oneOf(...TABLE_REJECTS) });
const lobbyReject = shape({ t: oneOf('lobby-reject'), seq: count, code: oneOf(...LOBBY_REJECTS) });
const linked = shape({ t: oneOf('linked'), code: str, seat: seat, claim: str });
const created = shape({ t: oneOf('created'), code: str }, { claim: str });
// Common requests (resync, link, relink, rematch) dropped by the frame rate
// limit are refused in Duel's family (docs/protocol.md, "Reject codes"). Only
// its seq and code are read.
export interface CommonReject {
  t: 'reject';
  seq: number;
  code: string;
}
const duelReject = shape({ t: oneOf('reject'), seq: count, code: str });

export function isTableState(value: unknown): value is TableState {
  return tableState(value);
}
export function isLobbyState(value: unknown): value is LobbyState {
  return lobbyState(value);
}

/** A frame the Haftgane Session takes, by family. */
export type SessionFrame =
  TableState | LobbyState | TableReject | LobbyReject | PlatformLinked | Created | CommonReject;

/**
 * The frame as one the Session reads, or null when it is none of them.
 * `invalid` is true when the frame claims a family the Session reads but
 * does not have its shape: a client and server out of step, which no
 * retry fixes.
 */
export function parseFrame(value: unknown): { frame: SessionFrame | null; invalid: boolean } {
  const t = object(value)?.['t'];
  const checks: Record<string, Guard> = {
    'table-state': tableState,
    'lobby-state': lobbyState,
    'table-reject': tableReject,
    'lobby-reject': lobbyReject,
    linked,
    created,
    reject: duelReject
  };
  const check = typeof t === 'string' ? checks[t] : undefined;
  if (!check) return { frame: null, invalid: false };
  return check(value)
    ? { frame: value as SessionFrame, invalid: false }
    : { frame: null, invalid: true };
}
