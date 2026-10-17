import { describe, expect, it } from 'vitest';
import { CoachAcks } from './coach.svelte';
import { ackKey } from './steps';

describe('the coach acknowledgements', () => {
  it('remember Next per position, and an undo forgets from its ply on', () => {
    const acks = new CoachAcks([]);
    acks.add(ackKey('ROOM', 6, 0, 'ageOne'));
    acks.add(ackKey('ROOM', 8, 0, 'reveal'));
    acks.add(ackKey('OTHER', 8, 0, 'reveal'));
    expect(acks.has(ackKey('ROOM', 6, 0, 'ageOne'))).toBe(true);
    acks.forgetFrom('ROOM', 7);
    expect(acks.has(ackKey('ROOM', 6, 0, 'ageOne'))).toBe(true);
    expect(acks.has(ackKey('ROOM', 8, 0, 'reveal'))).toBe(false);
    expect(acks.has(ackKey('OTHER', 8, 0, 'reveal'))).toBe(true);
  });
});
