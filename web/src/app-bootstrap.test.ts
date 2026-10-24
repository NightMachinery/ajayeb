// The inline bootstrap in app.html decides before the first paint whether
// Home keeps its build-time tutorial offer (M11.home-cold-cls). It cannot
// import the app's code, so it repeats offersTutorial over the raw storage;
// this test runs the real script against the stores' own parsing.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { SavedRooms } from '$lib/session/rooms.svelte';
import { offersTutorial, TipsStore } from '$lib/ui/tips.svelte';

const html = readFileSync(new URL('./app.html', import.meta.url), 'utf8');
const script = /<script>([\s\S]*?)<\/script>/.exec(html)?.[1] ?? '';

type Stored = Record<string, string>;

function storage(stored: Stored | 'blocked') {
  return {
    getItem(key: string): string | null {
      if (stored === 'blocked') throw new Error('storage blocked');
      return stored[key] ?? null;
    },
    setItem(): void {}
  };
}

/** The root's attributes after the bootstrap runs on this storage, with
 *  the system's colour scheme dark or not. */
function boot(stored: Stored | 'blocked', systemDark = false): Map<string, string> {
  const attributes = new Map<string, string>();
  const root = {
    setAttribute: (name: string, value: string) => attributes.set(name, value),
    removeAttribute: (name: string) => attributes.delete(name),
    classList: { toggle() {} },
    style: {}
  };
  const run = new Function('window', 'document', 'localStorage', 'setTimeout', script);
  run(
    { matchMedia: (q: string) => ({ matches: systemDark && q.includes('dark') }) },
    { documentElement: root, querySelectorAll: () => [] },
    storage(stored),
    () => 0
  );
  return attributes;
}

/** Whether Home offers the tutorial once the app has read the same storage. */
function app(stored: Stored | 'blocked'): boolean {
  const s = storage(stored);
  return offersTutorial(new TipsStore(s).value.tutorial, new SavedRooms(s).list.length);
}

const room = { code: 'ABCDEF', mode: 'ai', lastSeen: 1 };
const tips = (tutorial: object) => JSON.stringify({ off: false, seen: [], tutorial });

const agreed: [string, Stored | 'blocked'][] = [
  ['a new device', {}],
  ['blocked storage', 'blocked'],
  ['unreadable JSON', { 'ajayeb:tips': '{', 'ajayeb:rooms': '[' }],
  ['saved rooms', { 'ajayeb:rooms': JSON.stringify([room]) }],
  ['a declined offer', { 'ajayeb:tips': tips({ offer: 'declined' }) }],
  ['a finished tutorial', { 'ajayeb:tips': tips({ offer: 'new', done: true }) }],
  ['an offer shown again', { 'ajayeb:tips': tips({ offer: 'new', done: false }) }],
  ['an empty rooms list', { 'ajayeb:rooms': '[]' }],
  ['rooms that are not a list', { 'ajayeb:rooms': JSON.stringify(room) }]
];

describe('the bootstrap tutorial offer', () => {
  it('found the inline script', () => {
    expect(script).toContain("'ajayeb:tips'");
  });

  for (const [name, stored] of agreed) {
    it(`agrees with the app for ${name}`, () => {
      expect(boot(stored).has('data-boot-offer')).toBe(app(stored));
    });
  }

  // The one known divergence: the bootstrap counts stored entries without
  // validating them, so a list of only malformed rooms hides the offer
  // before mount and the app inserts it after (the old shift, never a lost
  // offer, since the hiding rule ends with Home's loading state).
  it('differs only for a rooms list whose entries are all malformed', () => {
    const stored = { 'ajayeb:rooms': JSON.stringify([{ code: 'bad code', mode: 'ai' }]) };
    expect(app(stored)).toBe(true);
    expect(boot(stored).has('data-boot-offer')).toBe(false);
  });

  it('marks a stored Persian locale as pending, and English as not', () => {
    expect(
      boot({ 'ajayeb:settings': JSON.stringify({ locale: 'fa' }) }).get('data-locale-pending')
    ).toBe('fa');
    expect(
      boot({ 'ajayeb:settings': JSON.stringify({ locale: 'en' }) }).has('data-locale-pending')
    ).toBe(false);
  });
});

describe('the bootstrap theme', () => {
  const themed = (stored: Stored | 'blocked', systemDark: boolean) => {
    const a = boot(stored, systemDark);
    return [a.get('data-theme'), a.get('data-theme-pref')];
  };

  it('gives a new visitor light, whatever the system scheme', () => {
    expect(themed({}, true)).toEqual(['light', 'light']);
    expect(themed({}, false)).toEqual(['light', 'light']);
    expect(themed('blocked', true)).toEqual(['light', 'light']);
    expect(themed({ 'ajayeb:theme': 'sepia' }, true)).toEqual(['light', 'light']);
  });

  it('keeps a stored choice', () => {
    expect(themed({ 'ajayeb:theme': 'dark' }, false)).toEqual(['dark', 'dark']);
    expect(themed({ 'ajayeb:theme': 'light' }, true)).toEqual(['light', 'light']);
    expect(themed({ 'ajayeb:theme': 'system' }, true)).toEqual(['dark', 'system']);
    expect(themed({ 'ajayeb:theme': 'system' }, false)).toEqual(['light', 'system']);
  });
});
