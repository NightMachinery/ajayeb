import { describe, expect, it } from 'vitest';
import type { RoomInfo } from '$lib/wire';
import { botSeat, personaName, seatNames } from './names';

function room(seats: RoomInfo['seats']): Pick<RoomInfo, 'seats'> {
  return { seats };
}

describe('seat names', () => {
  it('names a bot seat by its persona, from its level', () => {
    expect(personaName('easy')).toBe('Apprentice');
    expect(personaName('normal')).toBe('Scholar');
    expect(personaName('hard')).toBe('Master');
    const r = room([
      { seat: 0, name: 'Sara', kind: 'human', connected: true, you: true },
      { seat: 1, name: '', kind: 'bot', bot: 'normal', connected: true, you: false }
    ]);
    expect(seatNames(r)).toEqual(['Sara', 'Scholar']);
    expect(botSeat(r)).toBe(1);
  });

  it('works with the bot at the Sun', () => {
    const r = room([
      { seat: 0, name: '', kind: 'bot', bot: 'hard', connected: true, you: false },
      { seat: 1, name: '', kind: 'human', connected: true, you: true }
    ]);
    expect(seatNames(r)).toEqual(['Master', 'Moon']);
    expect(botSeat(r)).toBe(0);
  });

  it('falls back to the crest names, and has no bot seat without a bot', () => {
    expect(seatNames(null)).toEqual(['Sun', 'Moon']);
    const r = room([
      { seat: 0, name: '  ', kind: 'human', connected: true, you: true },
      { seat: 1, name: 'Omid', kind: 'human', connected: true, you: true }
    ]);
    expect(seatNames(r)).toEqual(['Sun', 'Omid']);
    expect(botSeat(r)).toBeUndefined();
  });
});
