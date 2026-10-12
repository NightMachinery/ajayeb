import { describe, expect, it } from 'vitest';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import type { State } from '$lib/wire';
import { isSelfSeat, numberedName, seatDisplayName } from './display-name';

function state(name: (typeof fixtureNames)[number]): State {
  const frame = fixtures[fixtureNames.indexOf(name)];
  if (!frame || frame.t !== 'state') throw new Error(`no state fixture named ${name}`);
  return frame;
}

describe('numberedName', () => {
  it('writes the number after the name', () => {
    expect(numberedName('Sara', 2)).toBe('Sara 2');
    expect(numberedName('Sara', 11)).toBe('Sara 11');
  });
  it('leaves an unnumbered name alone', () => {
    expect(numberedName('Sara', 0)).toBe('Sara');
    expect(numberedName('Sara', undefined)).toBe('Sara');
    expect(numberedName('Sara', 1)).toBe('Sara');
    expect(numberedName('Sara', 2.5)).toBe('Sara');
  });
});

describe('seatDisplayName', () => {
  it('numbers the later of two equal names, from a real server frame', () => {
    const st = state('state-names-numbered');
    const names = st.room.seats.map((s) => seatDisplayName(s, 'Sun'));
    expect(names).toEqual(['Sara', 'Sara 2']);
  });
  it('trims the name and falls back for a blank one', () => {
    expect(seatDisplayName({ name: '  Omid ', nameNo: 0 }, 'Moon')).toBe('Omid');
    expect(seatDisplayName({ name: '  ', nameNo: 0 }, 'Moon')).toBe('Moon');
    expect(seatDisplayName(undefined, 'Moon')).toBe('Moon');
    expect(seatDisplayName({ name: 'Omid', nameNo: 3 }, 'Moon')).toBe('Omid 3');
  });
});

describe('isSelfSeat', () => {
  it('marks the seat this connection holds, outside hot-seat rooms', () => {
    expect(isSelfSeat({ you: true }, 'online')).toBe(true);
    expect(isSelfSeat({ you: true }, 'ai')).toBe(true);
    expect(isSelfSeat({ you: false }, 'online')).toBe(false);
    expect(isSelfSeat({ you: true }, 'hotseat')).toBe(false);
    expect(isSelfSeat(undefined, 'online')).toBe(false);
  });
  it('reads the frame: seat 1 is the viewer in the numbered fixture', () => {
    const st = state('state-names-numbered');
    expect(st.room.seats.map((s) => isSelfSeat(s, st.room.mode))).toEqual([false, true]);
  });
});
