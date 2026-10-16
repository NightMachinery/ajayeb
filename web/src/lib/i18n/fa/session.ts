// Persian messages of the connection layer: the connection states the turn
// bar shows. Partial; the reject and error codes stay English for now.

import type { session as en } from '../en/session';
import type { Translation } from '../types';

export const session = {
  'session.conn.connecting': 'در حال اتصال…',
  'session.conn.open': 'متصل',
  'session.conn.reconnecting': 'در حال اتصال دوباره…',
  'session.conn.offline': 'آفلاین. در حال تلاش برای اتصال دوباره…',
  'session.pendingSlow': 'در انتظار سرور…'
} as const satisfies Translation<typeof en>;
