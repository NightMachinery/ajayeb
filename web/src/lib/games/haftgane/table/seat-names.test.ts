import { describe, expect, it } from 'vitest';
import { tableFrame } from '../session/test-frames';
import { botName, persona, seatName } from './seat-names';

describe('generated bot display parts', () => {
  it('numbers each level in seat order without parsing names', () => {
    const frame = tableFrame({
      seats: [
        { seat: 0, kind: 'bot', bot: 'easy' },
        { seat: 1, kind: 'bot', bot: 'hard' },
        { seat: 2, kind: 'bot', bot: 'easy' }
      ]
    });
    expect(botName(frame, 0)).toEqual({ label: persona('easy'), suffix: '' });
    expect(botName(frame, 1)).toEqual({ label: persona('hard'), suffix: '' });
    expect(botName(frame, 2)).toEqual({ label: persona('easy'), suffix: '2' });
    expect(seatName(frame, 2)).toBe(`${persona('easy')} 2`);
  });
  it('leaves a numbered human name as user text', () => {
    const frame = tableFrame();
    frame.humans[0]!.name = 'Apprentice 42';
    expect(botName(frame, 0)).toBeNull();
    expect(seatName(frame, 0)).toBe('Apprentice 42');
  });
});
