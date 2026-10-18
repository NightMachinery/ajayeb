import { describe, expect, it } from 'vitest';
import { defaultHints, withFamily, withStrength } from './families';
import {
  HINT_DEFAULTS_KEY,
  HINT_HIDE_KEY,
  HintDefaultsStore,
  HintHideStore,
  parseLocalHide,
  parseSavedHints,
  type StorageLike
} from './local.svelte';

function memory(initial: Record<string, string> = {}): StorageLike & { data: Map<string, string> } {
  const data = new Map(Object.entries(initial));
  return {
    data,
    getItem: (k) => data.get(k) ?? null,
    setItem: (k, v) => void data.set(k, v)
  };
}

const blocked: StorageLike = {
  getItem: () => {
    throw new Error('SecurityError');
  },
  setItem: () => {
    throw new Error('SecurityError');
  }
};

describe('the local default (ajayeb:hints)', () => {
  it('starts each mode from its defaults when nothing is saved', () => {
    const store = new HintDefaultsStore(memory());
    expect(store.for('ai')).toEqual(defaultHints('ai'));
    expect(store.for('online')).toEqual(defaultHints('online'));
    expect(new HintDefaultsStore(blocked).for('hotseat')).toEqual(defaultHints('hotseat'));
    expect(new HintDefaultsStore(null).for('ai')).toEqual(defaultHints('ai'));
  });

  it('starts a new game from the last saved families, with the advisor of its mode', () => {
    const storage = memory();
    const store = new HintDefaultsStore(storage);
    store.save('ai', withFamily(defaultHints('ai'), 'prices', false));
    // Prices follow into every mode; the advisor turned on against the bot
    // stays off where its default is off.
    expect(store.for('online')).toEqual(withFamily(defaultHints('online'), 'prices', false));
    expect(store.for('ai')).toEqual(withFamily(defaultHints('ai'), 'prices', false));
    store.save('online', withFamily(defaultHints('online'), 'advisor', true));
    expect(store.for('online').advisor).toBe(true);
    expect(store.for('online').prices).toBe(true);
    expect(store.for('hotseat').advisor).toBe(false);

    // It survives a reload, under its key.
    expect(JSON.parse(storage.data.get(HINT_DEFAULTS_KEY)!)).toEqual({
      hints: withFamily(defaultHints('online'), 'advisor', true),
      advisor: { ai: true, online: true }
    });
    const again = new HintDefaultsStore(storage);
    expect(again.for('online')).toEqual(withFamily(defaultHints('online'), 'advisor', true));
    expect(again.for('ai').advisor).toBe(true);
  });

  it('remembers the advisor strength for every mode', () => {
    const store = new HintDefaultsStore(memory());
    expect(store.for('ai').advisorStrength).toBe('twoStage');
    store.save('ai', withStrength(defaultHints('ai'), 'deep'));
    expect(store.for('online').advisorStrength).toBe('deep');
    expect(store.for('ai').advisorStrength).toBe('deep');
  });

  it('reads a malformed or partial value leniently', () => {
    expect(parseSavedHints('nonsense')).toEqual({ hints: null, advisor: {} });
    const saved = parseSavedHints({
      hints: { prices: false },
      advisor: { ai: 'yes', online: true }
    });
    expect(saved.hints).toEqual(withFamily(defaultHints('ai'), 'prices', false));
    expect(saved.advisor).toEqual({ online: true });
    const store = new HintDefaultsStore(memory({ [HINT_DEFAULTS_KEY]: '{not json' }));
    expect(store.for('ai')).toEqual(defaultHints('ai'));
  });

  it('keeps working when storage is blocked', () => {
    const store = new HintDefaultsStore(blocked);
    store.save('hotseat', withFamily(defaultHints('hotseat'), 'recap', false));
    expect(store.for('hotseat').recap).toBe(false);
  });
});

describe('the local hide (ajayeb:hintsHidden)', () => {
  it('hides and shows families and the board marks, and persists them', () => {
    const storage = memory();
    const store = new HintHideStore(storage);
    expect(store.value).toEqual({ families: [], boardMarks: false });
    store.setFamily('prices', true);
    store.setFamily('science', true);
    store.setFamily('prices', true);
    expect(store.value.families).toEqual(['science', 'prices']);
    store.setFamily('science', false);
    store.setBoardMarks(true);
    expect(store.value).toEqual({ families: ['prices'], boardMarks: true });
    expect(new HintHideStore(storage).value).toEqual({ families: ['prices'], boardMarks: true });
    expect(JSON.parse(storage.data.get(HINT_HIDE_KEY)!)).toEqual({
      families: ['prices'],
      boardMarks: true
    });
  });

  it('drops what it does not know', () => {
    expect(parseLocalHide({ families: ['prices', 'law', 3, 'prices'], boardMarks: 'yes' })).toEqual(
      {
        families: ['prices'],
        boardMarks: false
      }
    );
    expect(parseLocalHide(null)).toEqual({ families: [], boardMarks: false });
    expect(new HintHideStore(blocked).value).toEqual({ families: [], boardMarks: false });
  });
});
