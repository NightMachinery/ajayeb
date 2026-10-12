import { describe, expect, it } from 'vitest';
import { UNDO_DEFAULT_KEY, UndoDefaultStore, parseSavedUndo } from './undo-default.svelte';

function memory(initial?: string) {
  const data = new Map<string, string>();
  if (initial !== undefined) data.set(UNDO_DEFAULT_KEY, initial);
  return {
    data,
    getItem: (k: string) => data.get(k) ?? null,
    setItem: (k: string, v: string) => void data.set(k, v)
  };
}

describe('the "Allow undo" local default', () => {
  it('is on for both modes until the player turns it off', () => {
    const store = new UndoDefaultStore(memory());
    expect(store.for('ai')).toBe(true);
    expect(store.for('hotseat')).toBe(true);
  });

  it('remembers each mode on its own, across a reload', () => {
    const storage = memory();
    new UndoDefaultStore(storage).save('hotseat', false);
    const again = new UndoDefaultStore(storage);
    expect(again.for('hotseat')).toBe(false);
    expect(again.for('ai')).toBe(true);
    expect(JSON.parse(storage.data.get(UNDO_DEFAULT_KEY) ?? '{}')).toEqual({ hotseat: false });
  });

  it('reads a damaged value leniently', () => {
    expect(parseSavedUndo('not json')).toEqual({});
    expect(parseSavedUndo('[1,2]')).toEqual({});
    expect(parseSavedUndo('{"ai":"no","hotseat":false,"online":false}')).toEqual({
      hotseat: false
    });
    expect(new UndoDefaultStore(null).for('ai')).toBe(true);
  });
});
