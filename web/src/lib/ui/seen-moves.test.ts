import { describe, expect, it } from 'vitest';
import { MAX_SEEN_ROOMS, parseSeenMoves, SeenMoves } from './seen-moves';

describe('seen opponent moves', () => {
  it('reads leniently, keeps room/seat identity and bounds storage', () => {
    expect(
      parseSeenMoves([null, ['BAD', 1], ['ROOM:online:0', -2], ['ROOM:online:0', 2.1]])
    ).toEqual([]);
    const rows = Array.from({ length: MAX_SEEN_ROOMS + 2 }, (_, i) => [`R${i}:online:0`, i]);
    expect(parseSeenMoves(rows)).toHaveLength(MAX_SEEN_ROOMS);
    const store = new SeenMoves(null);
    store.remember('ROOM:online:0', 9);
    expect(store.get('ROOM:online:1')).toBe(-1);
    expect(store.get('OTHER:online:0')).toBe(-1);
  });
  it('survives reload and lets undo lower the marker', () => {
    let raw = '';
    const storage = {
      getItem: () => raw,
      setItem: (_: string, value: string) => {
        raw = value;
      }
    };
    const store = new SeenMoves(storage);
    store.remember('ROOM:ai:0', 12);
    expect(new SeenMoves(storage).get('ROOM:ai:0')).toBe(12);
    store.remember('ROOM:hotseat:1', 13);
    store.rewind('ROOM', 9);
    expect(store.get('ROOM:hotseat:1')).toBe(9);
    expect(new SeenMoves(storage).get('ROOM:ai:0')).toBe(9);
  });
  it('keeps working with invalid JSON or blocked storage', () => {
    const store = new SeenMoves({
      getItem: () => '{',
      setItem: () => {
        throw Error('blocked');
      }
    });
    expect(store.get('ROOM:ai:0')).toBe(-1);
    store.remember('ROOM:ai:0', 4);
    expect(store.get('ROOM:ai:0')).toBe(4);
  });
});
