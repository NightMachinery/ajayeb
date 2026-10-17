import { describe, expect, it } from 'vitest';
import { replay } from './replay.svelte';

describe('the replay offer', () => {
  it('replays the move on offer until it is withdrawn', () => {
    let played = 0;
    replay.withdraw();
    expect(replay.available).toBe(false);
    expect(replay.play()).toBe(false);
    replay.offer(12, () => {
      played += 1;
      return true;
    });
    expect(replay.available).toBe(true);
    expect(replay.version).toBe(12);
    expect(replay.play()).toBe(true);
    expect(played).toBe(1);
    replay.withdraw();
    expect(replay.play()).toBe(false);
    expect(played).toBe(1);
  });
});
