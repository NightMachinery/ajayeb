import { describe, expect, it } from 'vitest';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import type { RoomLogEntry } from '$lib/wire';
import { formatLinkReset } from './link-log';

function reset(seat: 0 | 1 | undefined, target: 0 | 1): RoomLogEntry {
  const e: RoomLogEntry = { kind: 'link', at: 0, on: [], off: [], target };
  if (seat !== undefined) e.seat = seat;
  return e;
}

describe('formatLinkReset', () => {
  const names = ['Sara', 'Sara 2'] as const;
  it('says who reset whose link', () => {
    expect(formatLinkReset(reset(1, 1), names)).toBe('Sara 2 reset their device link.');
    expect(formatLinkReset(reset(0, 1), names)).toBe('Sara reset the device link of Sara 2.');
    expect(formatLinkReset(reset(undefined, 0), names)).toBe('The device link of Sara was reset.');
  });
  it('reads a real server frame', () => {
    const frame = fixtures[fixtureNames.indexOf('state-names-numbered')];
    if (frame?.t !== 'state') throw new Error('no fixture');
    const entry = frame.roomLog[0]!;
    expect(entry.kind).toBe('link');
    expect(formatLinkReset(entry, names)).toBe('Sara 2 reset their device link.');
  });
});
