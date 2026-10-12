// openRules() and the overlay's state (rules-overlay.svelte.ts). The overlay
// component itself is replaced by a stub mounter: these tests cover the
// helper later slices call ("More in Rules"), not the rendering.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { closeRules, openRules, rulesOverlay, setRulesMounter } from './rules-overlay.svelte';

afterEach(() => {
  closeRules();
  setRulesMounter(undefined);
});

describe('openRules', () => {
  it('opens at a section, and again at the same one with a new request', async () => {
    setRulesMounter(async () => {});
    await openRules('military');
    expect(rulesOverlay.open).toBe(true);
    expect(rulesOverlay.section).toBe('military');
    const first = rulesOverlay.request;
    await openRules('military');
    expect(rulesOverlay.request).toBe(first + 1);
  });

  it('opens at the top with no section or an unknown one', async () => {
    setRulesMounter(async () => {});
    await openRules();
    expect(rulesOverlay.section).toBeUndefined();
    await openRules('no-such-section');
    expect(rulesOverlay.open).toBe(true);
    expect(rulesOverlay.section).toBeUndefined();
  });

  it('mounts the overlay once', async () => {
    const mount = vi.fn(async () => {});
    setRulesMounter(mount);
    await openRules('science');
    closeRules();
    await openRules('tokens');
    expect(mount).toHaveBeenCalledTimes(1);
  });

  it('tries the mount again after a failed load', async () => {
    const mount = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValue(undefined);
    setRulesMounter(mount);
    await expect(openRules()).rejects.toThrow('offline');
    await openRules();
    expect(mount).toHaveBeenCalledTimes(2);
  });

  it('closes', async () => {
    setRulesMounter(async () => {});
    await openRules('goal');
    closeRules();
    expect(rulesOverlay.open).toBe(false);
  });
});
