// link-log.ts: the game log line of a device link reset (docs/protocol.md,
// the room log's "link" entries): who reset whose link. Pure display
// mapping of the entry's seats; the names come in as data.

import { t } from '$lib/i18n';
import type { RoomLogEntry } from '$lib/wire';
import type { SeatNames } from './log-format';

/** A link reset as a line of text. */
export function formatLinkReset(entry: RoomLogEntry, names: SeatNames): string {
  const target = entry.target === 0 || entry.target === 1 ? names[entry.target] : '';
  if (entry.seat === 0 || entry.seat === 1) {
    const name = names[entry.seat];
    return entry.seat === entry.target
      ? t('online.log.linkResetOwn', { name })
      : t('online.log.linkResetOther', { name, target });
  }
  return t('online.log.linkReset', { target });
}
