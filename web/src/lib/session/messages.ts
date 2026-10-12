// The server's reject and error codes, mapped to message keys. The server's
// msg field is a developer hint and is never shown to a player.

import type { MessageKey } from '$lib/i18n';
import type { ConnState } from './session.svelte';

const REJECTS = {
  stale: 'session.reject.stale',
  illegal: 'session.reject.illegal',
  'not-your-turn': 'session.reject.notYourTurn',
  rate: 'session.reject.rate',
  'bad-request': 'session.reject.badRequest',
  'not-admin': 'session.reject.notAdmin',
  'game-over': 'session.reject.gameOver',
  'undo-off': 'session.reject.undoOff'
} as const satisfies Readonly<Record<string, MessageKey>>;

const ERRORS = {
  'room-not-found': 'session.error.roomNotFound',
  'room-full': 'session.error.roomFull',
  forbidden: 'session.error.forbidden',
  'bad-hello': 'session.error.badHello',
  protocol: 'session.error.protocol',
  busy: 'session.error.busy',
  'bad-claim': 'session.error.badClaim'
} as const satisfies Readonly<Record<string, MessageKey>>;

const CONN = {
  connecting: 'session.conn.connecting',
  open: 'session.conn.open',
  reconnecting: 'session.conn.reconnecting',
  offline: 'session.conn.offline'
} as const satisfies Readonly<Record<ConnState, MessageKey>>;

/** A message about a refusal; none takes params, so t(key) needs none. */
export type NoticeKey =
  | (typeof REJECTS)[keyof typeof REJECTS]
  | (typeof ERRORS)[keyof typeof ERRORS]
  | 'session.error.unknown';

export type ConnKey = (typeof CONN)[ConnState];

function lookup<T extends Readonly<Record<string, NoticeKey>>>(table: T, code: string): NoticeKey {
  return Object.hasOwn(table, code) ? table[code as keyof T]! : 'session.error.unknown';
}

/** The message for a reject code; a code this client does not know gets a generic one. */
export function rejectMessage(code: string): NoticeKey {
  return lookup(REJECTS, code);
}

/** The message for an error code; a code this client does not know gets a generic one. */
export function errorMessage(code: string): NoticeKey {
  return lookup(ERRORS, code);
}

export function connMessage(conn: ConnState): ConnKey {
  return CONN[conn];
}
