// The room log in the game log (docs/protocol.md, the state frame's
// roomLog): each change of the hint settings, as a line among the game's
// own events. Pure display mapping. A room log entry names the ply it
// happened at, so it goes before the history entry of that ply (after
// every action already applied), in whatever age that falls.

import { t } from '$lib/i18n';
import {
  formatLogEvent,
  initialLogAge,
  type LogAgeGroup,
  type LogLine,
  type SeatNames
} from '$lib/view/log-format';
import { formatLinkReset } from '$lib/view/link-log';
import type { HistoryEntry, RoomLogEntry } from '$lib/wire';
import { familyList } from './text';

/** One room log entry as a line of text. */
export function formatRoomLogEntry(entry: RoomLogEntry, names: SeatNames): string {
  // A device link reset (lib/view/link-log.ts) is in the same room log.
  if (entry.kind === 'link') return formatLinkReset(entry, names);
  const parts: string[] = [];
  if (entry.on.length > 0) {
    parts.push(t('hints.log.on', { families: familyList(entry.on) }));
  }
  if (entry.off.length > 0) {
    parts.push(t('hints.log.off', { families: familyList(entry.off) }));
  }
  const changes = parts.join(t('hints.log.separator'));
  if (entry.seat === 0 || entry.seat === 1) {
    return t('hints.log.changedBy', { name: names[entry.seat], changes });
  }
  return t('hints.log.changed', { changes });
}

/** A line of the game log: a game event's, or a room log entry's. */
export type GameLogLine =
  LogLine | { readonly kind: 'roomHints'; readonly text: string; readonly seat?: 0 | 1 };

export interface GameLogGroup {
  readonly age: LogAgeGroup['age'];
  readonly lines: GameLogLine[];
}

/**
 * The game log grouped by age, as lib/view/log-format's groupLogByAge
 * builds it from the history, with the room log entries placed among the
 * events: each before the history entry of its ply. Only the history before
 * `shownPly` counts (the frame on screen; a bot's answer still being paced
 * is not in the log yet), and so only the room log entries up to it.
 */
export function groupGameLog(
  history: readonly HistoryEntry[],
  roomLog: readonly RoomLogEntry[],
  names: SeatNames,
  shownPly?: number,
  currentAge = 0
): GameLogGroup[] {
  const shown = history.filter((entry) => shownPly === undefined || entry.ply < shownPly);
  const groups: { age: number; lines: GameLogLine[] }[] = [
    {
      age: initialLogAge(
        shown.flatMap((entry) => entry.events),
        currentAge
      ),
      lines: []
    }
  ];
  const pending = roomLog
    .filter((e) => shownPly === undefined || e.at <= shownPly)
    .map((e, i) => ({ e, i }))
    .sort((a, b) => a.e.at - b.e.at || a.i - b.i)
    .map(({ e }) => e);
  let next = 0;
  const emitRoom = (upTo: number): void => {
    while (next < pending.length && pending[next]!.at <= upTo) {
      const e = pending[next++]!;
      const line: GameLogLine =
        e.seat === 0 || e.seat === 1
          ? { kind: 'roomHints', text: formatRoomLogEntry(e, names), seat: e.seat }
          : { kind: 'roomHints', text: formatRoomLogEntry(e, names) };
      groups[groups.length - 1]!.lines.push(line);
    }
  };
  for (const entry of history) {
    if (shownPly !== undefined && entry.ply >= shownPly) break;
    emitRoom(entry.ply);
    for (const event of entry.events) {
      if (event.kind === 'ageStart' && event.age !== undefined) {
        groups.push({ age: event.age, lines: [] });
      }
      const line = formatLogEvent(event, names);
      if (line) groups[groups.length - 1]!.lines.push(line);
    }
  }
  emitRoom(Number.POSITIVE_INFINITY);
  return groups.filter((g) => g.lines.length > 0);
}
