import { afterEach, describe, expect, it } from 'vitest';
import { i18n, type Locale } from '$lib/i18n';
import {
  DEFAULT_SETTINGS,
  parseSettings,
  SETTINGS_KEY,
  SettingsStore,
  type Settings,
  type StorageLike
} from './settings.svelte';

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

describe('parseSettings', () => {
  it('returns the defaults for nothing stored', () => {
    expect(parseSettings(null)).toEqual(DEFAULT_SETTINGS);
    expect(parseSettings('')).toEqual(DEFAULT_SETTINGS);
  });

  it('returns the defaults for unparsable or non-object JSON', () => {
    expect(parseSettings('not json')).toEqual(DEFAULT_SETTINGS);
    expect(parseSettings('42')).toEqual(DEFAULT_SETTINGS);
    expect(parseSettings('"a string"')).toEqual(DEFAULT_SETTINGS);
    expect(parseSettings('null')).toEqual(DEFAULT_SETTINGS);
  });

  it('keeps every valid field', () => {
    const stored: Settings = {
      animationSpeed: 'relaxed',
      confirmMoves: 'always',
      colorPatterns: false,
      cardText: 'large',
      hotseatPassDevice: false,
      hotseatOrientation: 'fixed',
      sound: true,
      soundVolume: 'loud',
      haptics: false,
      showOpponentMove: false,
      locale: 'fa'
    };
    expect(parseSettings(JSON.stringify(stored))).toEqual(stored);
  });

  it('drops a field with the wrong shape back to its default, keeping the rest', () => {
    const raw = JSON.stringify({
      animationSpeed: 'warp', // not a known speed
      confirmMoves: 'always',
      colorPatterns: 'yes', // not a boolean
      cardText: 'large',
      hotseatPassDevice: 'on', // not a boolean
      hotseatOrientation: 'sideways', // not an orientation
      sound: 1, // not a boolean
      soundVolume: 11, // not a volume step
      haptics: 'off', // not a boolean
      showOpponentMove: 'yes', // not a boolean
      locale: 'xx' // not a locale with a table
    });
    expect(parseSettings(raw)).toEqual({
      ...DEFAULT_SETTINGS,
      confirmMoves: 'always',
      cardText: 'large'
    });
  });

  it('ignores unknown extra fields', () => {
    const raw = JSON.stringify({ ...DEFAULT_SETTINGS, future: 'field' });
    expect(parseSettings(raw)).toEqual(DEFAULT_SETTINGS);
  });
});

describe('the opponent move summary setting', () => {
  it('is on by default, also for a value stored before it existed', () => {
    expect(DEFAULT_SETTINGS.showOpponentMove).toBe(true);
    expect(parseSettings(JSON.stringify({ sound: true })).showOpponentMove).toBe(true);
  });

  it('keeps a stored choice to turn it off', () => {
    const store = new SettingsStore(memory());
    store.set('showOpponentMove', false);
    expect(store.value.showOpponentMove).toBe(false);
    expect(parseSettings(JSON.stringify(store.value)).showOpponentMove).toBe(false);
  });
});

describe('sound and haptics settings', () => {
  it('default to sound off at the medium step, and haptics on', () => {
    expect(DEFAULT_SETTINGS.sound).toBe(false);
    expect(DEFAULT_SETTINGS.soundVolume).toBe('medium');
    expect(DEFAULT_SETTINGS.haptics).toBe(true);
  });

  it('fill in for a value stored before they existed', () => {
    const older = JSON.stringify({ animationSpeed: 'fast', colorPatterns: false });
    expect(parseSettings(older)).toEqual({
      ...DEFAULT_SETTINGS,
      animationSpeed: 'fast',
      colorPatterns: false
    });
  });

  it('are independent of the animation speed', () => {
    const store = new SettingsStore(memory());
    store.set('animationSpeed', 'off');
    store.set('sound', true);
    expect(store.value.sound).toBe(true);
    store.set('animationSpeed', 'normal');
    store.set('sound', false);
    expect(store.value.animationSpeed).toBe('normal');
  });
});

describe('SettingsStore', () => {
  it('starts at the defaults when nothing is stored', () => {
    const store = new SettingsStore(memory());
    expect(store.value).toEqual(DEFAULT_SETTINGS);
  });

  it('reads back a previously stored value', () => {
    const storage = memory();
    new SettingsStore(storage).set('animationSpeed', 'fast');
    expect(new SettingsStore(storage).value.animationSpeed).toBe('fast');
  });

  it('persists one field at a time, keeping the others', () => {
    const storage = memory();
    const store = new SettingsStore(storage);
    store.set('cardText', 'large');
    store.set('colorPatterns', false);
    expect(store.value).toEqual({ ...DEFAULT_SETTINGS, cardText: 'large', colorPatterns: false });
    expect(JSON.parse(storage.data.get(SETTINGS_KEY)!)).toEqual(store.value);
  });

  it('updates several fields with one write, keeping the others', () => {
    const storage = memory();
    const store = new SettingsStore(storage);
    store.update({ hotseatPassDevice: true, hotseatOrientation: 'face' });
    expect(store.value).toEqual({
      ...DEFAULT_SETTINGS,
      hotseatPassDevice: true,
      hotseatOrientation: 'face'
    });
    expect(new SettingsStore(storage).value.hotseatOrientation).toBe('face');
  });

  it('resets to the defaults', () => {
    const store = new SettingsStore(memory());
    store.set('confirmMoves', 'never');
    store.reset();
    expect(store.value).toEqual(DEFAULT_SETTINGS);
  });

  it('survives blocked or missing storage', () => {
    const store = new SettingsStore(blocked);
    store.set('animationSpeed', 'off');
    expect(store.value.animationSpeed).toBe('off');
    expect(new SettingsStore(null).value).toEqual(DEFAULT_SETTINGS);
  });
});

describe('the language setting', () => {
  afterEach(() => i18n.set('en'));

  it('is English by default, also for a value stored before it existed', () => {
    expect(DEFAULT_SETTINGS.locale).toBe('en');
    expect(parseSettings(JSON.stringify({ sound: true })).locale).toBe('en');
    expect(parseSettings(JSON.stringify({ locale: 'fa' })).locale).toBe('fa');
  });

  it('applies the stored locale when the store loads, before anything renders', () => {
    const applied: Locale[] = [];
    new SettingsStore(memory({ [SETTINGS_KEY]: '{"locale":"fa"}' }), (l) => applied.push(l));
    expect(applied).toEqual(['fa']);
  });

  it('switches the language and its direction at once', () => {
    const storage = memory();
    const store = new SettingsStore(storage, (l) => i18n.set(l));
    expect(i18n.dir).toBe('ltr');
    store.set('locale', 'fa');
    expect(i18n.locale).toBe('fa');
    expect(i18n.dir).toBe('rtl');
    expect(JSON.parse(storage.data.get(SETTINGS_KEY)!).locale).toBe('fa');
    store.set('locale', 'en');
    expect(i18n.dir).toBe('ltr');
  });

  it('keeps the language when another field changes', () => {
    const applied: Locale[] = [];
    const store = new SettingsStore(memory(), (l) => applied.push(l));
    store.set('locale', 'fa');
    store.set('sound', true);
    expect(store.value.locale).toBe('fa');
    expect(applied).toEqual(['en', 'fa', 'fa']);
  });
});
