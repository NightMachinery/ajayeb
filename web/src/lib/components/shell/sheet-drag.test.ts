import { describe, expect, it } from 'vitest';
import { shouldDismiss } from './sheet-drag';

describe('shouldDismiss', () => {
  it('never closes without travel', () => {
    expect(shouldDismiss(0, 100, 600)).toBe(false);
    expect(shouldDismiss(-20, 100, 600)).toBe(false);
  });

  it('closes after a quarter of the panel, at most 120 px', () => {
    expect(shouldDismiss(99, 1000, 400)).toBe(false);
    expect(shouldDismiss(100, 1000, 400)).toBe(true);
    expect(shouldDismiss(119, 1000, 800)).toBe(false);
    expect(shouldDismiss(120, 1000, 800)).toBe(true);
  });

  it('closes on a short fast flick', () => {
    expect(shouldDismiss(40, 60, 800)).toBe(true);
    expect(shouldDismiss(40, 200, 800)).toBe(false);
  });

  it('ignores a flick too short to be deliberate', () => {
    expect(shouldDismiss(20, 5, 800)).toBe(false);
  });
});
