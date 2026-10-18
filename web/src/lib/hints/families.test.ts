import { describe, expect, it } from 'vitest';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import type { HintSettings, State } from '$lib/wire';
import {
  HINT_FAMILIES,
  defaultHints,
  enabledFamilies,
  hintRows,
  isHintFamily,
  parseHints,
  sameHints,
  summarizeHints,
  visibleFamilies,
  withFamily,
  withStrength
} from './families';
import { summaryText } from './text';

function recorded(name: string): State {
  const frame = fixtures[(fixtureNames as readonly string[]).indexOf(name)];
  if (!frame || frame.t !== 'state') throw new Error(`no recorded state ${name}`);
  return frame as State;
}

describe('the families', () => {
  it('name every switch of the recorded settings, in the server order', () => {
    // The switches, then the advisor's strength, the one setting that is not a switch.
    const hints = recorded('state-draft').room.hints;
    expect(Object.keys(hints)).toEqual([...HINT_FAMILIES, 'advisorStrength']);
    expect(hints.advisorStrength).toBe('twoStage');
    expect(isHintFamily('economy')).toBe(true);
    expect(isHintFamily('law')).toBe(false);
  });

  it('default to every family on, the advisor only against the bot', () => {
    expect(enabledFamilies(defaultHints('ai'))).toEqual([...HINT_FAMILIES]);
    for (const mode of ['hotseat', 'online'] as const) {
      expect(defaultHints(mode)).toEqual(withFamily(defaultHints('ai'), 'advisor', false));
    }
    // The server's own defaults, as the recorded frames carry them.
    expect(recorded('state-draft').room.hints).toEqual(defaultHints('hotseat'));
    expect(recorded('state-ai-thinking').room.hints).toEqual(defaultHints('ai'));
  });

  it('compare and switch without touching the input', () => {
    const h = defaultHints('ai');
    const off = withFamily(h, 'prices', false);
    expect(h.prices).toBe(true);
    expect(off.prices).toBe(false);
    expect(sameHints(h, off)).toBe(false);
    expect(sameHints(off, withFamily(h, 'prices', false))).toBe(true);
    expect(sameHints(h, withStrength(h, 'quick'))).toBe(false);
    expect(withStrength(h, 'deep').advisorStrength).toBe('deep');
    expect(h.advisorStrength).toBe('twoStage');
  });
});

describe('visibleFamilies', () => {
  const room = withFamily(withFamily(defaultHints('online'), 'prices', false), 'counting', false);

  it('is the families the game has on, with no local hide', () => {
    const shown = visibleFamilies(room);
    expect(shown.has('prices')).toBe(false);
    expect(shown.has('advisor')).toBe(false);
    expect(shown.has('economy')).toBe(true);
    expect(shown.size).toBe(HINT_FAMILIES.length - 3);
  });

  it('takes the locally hidden families away, and never adds one', () => {
    const shown = visibleFamilies(room, { families: ['economy', 'prices'], boardMarks: true });
    expect(shown.has('economy')).toBe(false);
    expect(shown.has('prices')).toBe(false);
    expect(shown.size).toBe(HINT_FAMILIES.length - 4);
  });

  it('is empty without settings', () => {
    expect(visibleFamilies(null).size).toBe(0);
    expect(visibleFamilies(undefined, { families: [], boardMarks: false }).size).toBe(0);
  });
});

describe('the editor mapper', () => {
  it('gives one row per family with its keys, switch and hide', () => {
    const rows = hintRows(withFamily(defaultHints('ai'), 'tempo', false), {
      families: ['science'],
      boardMarks: false
    });
    expect(rows.map((r) => r.family)).toEqual([...HINT_FAMILIES]);
    const tempo = rows.find((r) => r.family === 'tempo')!;
    expect(tempo).toEqual({
      family: 'tempo',
      on: false,
      hidden: false,
      name: 'hints.family.tempo',
      hint: 'hints.family.tempo.hint'
    });
    expect(rows.find((r) => r.family === 'science')!.hidden).toBe(true);
  });
});

describe('the summary', () => {
  it('says all, none, all but the few off, or only the few on', () => {
    const all = defaultHints('ai');
    const none = Object.fromEntries(
      HINT_FAMILIES.map((f) => [f, false])
    ) as unknown as HintSettings;
    expect(summarizeHints(all)).toEqual({ kind: 'all' });
    expect(summarizeHints(none)).toEqual({ kind: 'none' });
    expect(summarizeHints(defaultHints('online'))).toEqual({ kind: 'allBut', off: ['advisor'] });
    expect(summarizeHints(withFamily(none, 'recap', true))).toEqual({
      kind: 'only',
      on: ['recap']
    });
    expect(summaryText(defaultHints('online'))).toBe('All hints on except Advisor');
    expect(summaryText(withFamily(defaultHints('online'), 'prices', false))).toBe(
      'All hints on except Prices and Advisor'
    );
    expect(summaryText(withFamily(none, 'recap', true))).toBe('Only Recap');
    expect(summaryText(all)).toBe('All hints on');
    expect(summaryText(none)).toBe('No hints');
  });
});

describe('parseHints', () => {
  it('keeps the booleans it knows and takes the fallback for the rest', () => {
    const fallback = defaultHints('online');
    expect(parseHints({ prices: false, advisor: true, law: false, recap: 'no' }, fallback)).toEqual(
      {
        ...fallback,
        prices: false,
        advisor: true
      }
    );
    expect(parseHints(null, fallback)).toEqual(fallback);
    expect(parseHints('x', fallback)).toEqual(fallback);
  });

  it('keeps a known advisor strength, and takes the fallback for any other', () => {
    const fallback = defaultHints('ai');
    expect(fallback.advisorStrength).toBe('twoStage');
    expect(parseHints({ advisorStrength: 'deep' }, fallback).advisorStrength).toBe('deep');
    expect(parseHints({ advisorStrength: 'slow' }, fallback).advisorStrength).toBe('twoStage');
    expect(parseHints({ prices: false }, fallback).advisorStrength).toBe('twoStage');
  });
});
