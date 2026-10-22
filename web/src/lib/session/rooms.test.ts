import { describe, expect, it } from 'vitest';
import type { StorageLike } from './identity';
import { MAX_SAVED_ROOMS, parseSavedRooms, ROOMS_KEY, SavedRooms } from './rooms.svelte';

function memory(initial: Record<string, string> = {}): StorageLike & { data: Map<string, string> } {
  const data = new Map(Object.entries(initial));
  return {
    data,
    getItem: (k) => data.get(k) ?? null,
    setItem: (k, v) => void data.set(k, v)
  };
}

describe('SavedRooms', () => {
  it('stores validated summary details, keeps them with a claim reset, and refreshes from another tab', () => {
    const storage = memory();
    const first = new SavedRooms(storage);
    first.remember('AAAAAA', 'ai', 1000, 'q3Vb8Kx0Zr7TnW2mYc5JhA');
    first.updateDetails('AAAAAA', { bot: 'easy', age: 2 });
    first.dropClaim('AAAAAA');
    const second = new SavedRooms(storage);
    expect(second.list).toEqual([
      { code: 'AAAAAA', mode: 'ai', lastSeen: 1000, bot: 'easy', age: 2 }
    ]);
    second.forget('AAAAAA');
    first.refresh();
    expect(first.list).toEqual([]);
    expect(
      parseSavedRooms(
        JSON.stringify([{ code: 'AAAAAA', mode: 'ai', lastSeen: 1, bot: 'invalid', age: 99 }])
      )
    ).toEqual([{ code: 'AAAAAA', mode: 'ai', lastSeen: 1 }]);
  });
  it('starts empty and remembers rooms newest first', () => {
    const storage = memory();
    const rooms = new SavedRooms(storage);
    expect(rooms.list).toEqual([]);
    rooms.remember('AAAAAA', 'hotseat', 1000);
    rooms.remember('BBBBBB', 'ai', 2000);
    expect(rooms.list.map((r) => r.code)).toEqual(['BBBBBB', 'AAAAAA']);
    expect(JSON.parse(storage.data.get(ROOMS_KEY)!)).toEqual([
      { code: 'BBBBBB', mode: 'ai', lastSeen: 2000 },
      { code: 'AAAAAA', mode: 'hotseat', lastSeen: 1000 }
    ]);
  });

  it('moves a room seen again to the front without duplicating it', () => {
    const rooms = new SavedRooms(memory());
    rooms.remember('AAAAAA', 'hotseat', 1000);
    rooms.remember('BBBBBB', 'ai', 2000);
    rooms.remember('AAAAAA', 'hotseat', 3000);
    expect(rooms.list).toEqual([
      { code: 'AAAAAA', mode: 'hotseat', lastSeen: 3000 },
      { code: 'BBBBBB', mode: 'ai', lastSeen: 2000 }
    ]);
  });

  it('reads back what it stored', () => {
    const storage = memory();
    new SavedRooms(storage).remember('K7QX2M', 'online', 5);
    expect(new SavedRooms(storage).list).toEqual([{ code: 'K7QX2M', mode: 'online', lastSeen: 5 }]);
  });

  it('keeps at most MAX_SAVED_ROOMS, dropping the oldest', () => {
    const rooms = new SavedRooms(memory());
    for (let i = 0; i < MAX_SAVED_ROOMS + 5; i++) rooms.remember(`R${i}`, 'hotseat', i);
    expect(rooms.list).toHaveLength(MAX_SAVED_ROOMS);
    expect(rooms.list[0]?.code).toBe(`R${MAX_SAVED_ROOMS + 4}`);
    expect(rooms.list.some((r) => r.code === 'R0')).toBe(false);
  });

  it('forgets a room', () => {
    const storage = memory();
    const rooms = new SavedRooms(storage);
    rooms.remember('AAAAAA', 'hotseat', 1);
    rooms.remember('BBBBBB', 'ai', 2);
    rooms.forget('AAAAAA');
    rooms.forget('NOPE00');
    expect(rooms.list.map((r) => r.code)).toEqual(['BBBBBB']);
    expect(new SavedRooms(storage).list.map((r) => r.code)).toEqual(['BBBBBB']);
  });

  it('ignores a malformed code or mode', () => {
    const rooms = new SavedRooms(memory());
    rooms.remember('bad code', 'hotseat', 1);
    rooms.remember('AAAAAA', 'chess' as 'hotseat', 1);
    expect(rooms.list).toEqual([]);
  });

  it('survives blocked storage', () => {
    const blocked: StorageLike = {
      getItem: () => {
        throw new Error('SecurityError');
      },
      setItem: () => {
        throw new Error('SecurityError');
      }
    };
    const rooms = new SavedRooms(blocked);
    rooms.remember('AAAAAA', 'hotseat', 1);
    expect(rooms.list).toHaveLength(1);
    expect(new SavedRooms(null).list).toEqual([]);
  });
});

describe('parseSavedRooms', () => {
  it('drops anything malformed or repeated', () => {
    const raw = JSON.stringify([
      { code: 'AAAAAA', mode: 'hotseat', lastSeen: 3 },
      { code: 'AAAAAA', mode: 'ai', lastSeen: 1 },
      { code: 'lower', mode: 'ai', lastSeen: 2 },
      { code: 'CCCCCC', mode: 'ai' },
      { code: 'DDDDDD', mode: 'ai', lastSeen: 'x' },
      'nonsense',
      null,
      { code: 'EEEEEE', mode: 'online', lastSeen: 9, extra: true }
    ]);
    expect(parseSavedRooms(raw)).toEqual([
      { code: 'EEEEEE', mode: 'online', lastSeen: 9 },
      { code: 'AAAAAA', mode: 'hotseat', lastSeen: 3 }
    ]);
    expect(parseSavedRooms('not json')).toEqual([]);
    expect(parseSavedRooms('{"code":"AAAAAA"}')).toEqual([]);
    expect(parseSavedRooms(null)).toEqual([]);
  });
});

describe('SavedRooms with device links', () => {
  const CLAIM = 'q3Vb8Kx0Zr7TnW2mYc5JhA';
  const OTHER = 'Hw4pZ9cN1sKe6uTb0LgQvY';

  it('keeps a room’s claim, and reads it back', () => {
    const storage = memory();
    const rooms = new SavedRooms(storage);
    rooms.remember('AAAAAA', 'online', 1000, CLAIM);
    expect(rooms.claimFor('AAAAAA')).toBe(CLAIM);
    expect(rooms.claimFor('BBBBBB')).toBeNull();
    expect(new SavedRooms(storage).list).toEqual([
      { code: 'AAAAAA', mode: 'online', lastSeen: 1000, claim: CLAIM }
    ]);
  });

  it('keeps the claim when the room is seen again without one, and replaces or drops it on request', () => {
    const rooms = new SavedRooms(memory());
    rooms.remember('AAAAAA', 'online', 1000, CLAIM);
    rooms.remember('AAAAAA', 'online', 2000);
    expect(rooms.claimFor('AAAAAA')).toBe(CLAIM);
    rooms.remember('AAAAAA', 'online', 3000, OTHER);
    expect(rooms.claimFor('AAAAAA')).toBe(OTHER);
    rooms.remember('AAAAAA', 'online', 4000, null);
    expect(rooms.list).toEqual([{ code: 'AAAAAA', mode: 'online', lastSeen: 4000 }]);
  });

  it('drops a claim that stopped working but keeps the room', () => {
    const rooms = new SavedRooms(memory());
    rooms.remember('AAAAAA', 'hotseat', 1000, CLAIM);
    rooms.dropClaim('AAAAAA');
    expect(rooms.list).toEqual([{ code: 'AAAAAA', mode: 'hotseat', lastSeen: 1000 }]);
    rooms.dropClaim('BBBBBB');
    expect(rooms.list).toHaveLength(1);
  });

  it('ignores a malformed claim in storage, dropping that entry', () => {
    const raw = JSON.stringify([
      { code: 'AAAAAA', mode: 'online', lastSeen: 2, claim: 'nope' },
      { code: 'BBBBBB', mode: 'online', lastSeen: 1, claim: CLAIM }
    ]);
    expect(parseSavedRooms(raw)).toEqual([
      { code: 'BBBBBB', mode: 'online', lastSeen: 1, claim: CLAIM }
    ]);
  });
});

it('preserves known game hints and survives an old tab rewriting only legacy fields', () => {
  const storage = memory();
  const rooms = new SavedRooms(storage);
  rooms.remember('AAAAAA', 'online', 1000, undefined, 'haftgane');
  rooms.remember('BBBBBB', 'ai', 2000);
  rooms.updateDetails('BBBBBB', { bot: 'easy', age: 1 });
  expect(parseSavedRooms(storage.data.get(ROOMS_KEY)!)[1]?.game).toBe('haftgane');
  // Exact legacy reconstruction: unknown game is erased, but mode online keeps
  // the Haftgane room. A new tab must discover it rather than assume Duel.
  const old = JSON.parse(storage.data.get(ROOMS_KEY)!) as {
    code: string;
    mode: 'online' | 'ai' | 'hotseat';
    lastSeen: number;
  }[];
  storage.setItem(
    ROOMS_KEY,
    JSON.stringify(old.map(({ code, mode, lastSeen }) => ({ code, mode, lastSeen })))
  );
  rooms.refresh();
  expect(rooms.list[1]).toEqual({ code: 'AAAAAA', mode: 'online', lastSeen: 1000 });
  rooms.setGame('AAAAAA', 'haftgane');
  rooms.remember('AAAAAA', 'online', 3000);
  expect(rooms.list[0]?.game).toBe('haftgane');
  rooms.updateDetails('AAAAAA', { age: 2 });
  expect(rooms.list[0]?.game).toBe('haftgane');
  rooms.setGame('AAAAAA', 'duel'); // authoritative snapshot corrects a stale hint
  expect(rooms.list[0]?.game).toBe('duel');
  expect(
    parseSavedRooms(
      JSON.stringify([{ code: 'CCCCCC', mode: 'online', lastSeen: 1, game: 'unknown' }])
    )
  ).toEqual([{ code: 'CCCCCC', mode: 'online', lastSeen: 1 }]);
});
