import { describe, expect, it } from 'vitest';
import type { StorageLike } from './identity';
import { parseSavedRooms, SavedRooms } from './rooms.svelte';

function memory(): StorageLike {
  const data = new Map<string, string>();
  return { getItem: (k) => data.get(k) ?? null, setItem: (k, v) => void data.set(k, v) };
}

describe('a saved tutorial room', () => {
  it('keeps its flag through a reload, a later remember and a claim reset', () => {
    const storage = memory();
    const rooms = new SavedRooms(storage);
    rooms.remember('TUTOR1', 'ai', 1000, 'q3Vb8Kx0Zr7TnW2mYc5JhA');
    rooms.updateDetails('TUTOR1', { bot: 'easy', age: 1, tutorial: true });
    rooms.remember('TUTOR1', 'ai', 2000);
    rooms.dropClaim('TUTOR1');
    expect(new SavedRooms(storage).list).toEqual([
      { code: 'TUTOR1', mode: 'ai', lastSeen: 2000, bot: 'easy', age: 1, tutorial: true }
    ]);
  });

  it('drops a flag that is not true, or on a room that is not an ai room', () => {
    expect(
      parseSavedRooms(
        JSON.stringify([
          { code: 'AAAAAA', mode: 'ai', lastSeen: 2, tutorial: 'yes' },
          { code: 'BBBBBB', mode: 'hotseat', lastSeen: 1, tutorial: true }
        ])
      )
    ).toEqual([
      { code: 'AAAAAA', mode: 'ai', lastSeen: 2 },
      { code: 'BBBBBB', mode: 'hotseat', lastSeen: 1 }
    ]);
  });
});
