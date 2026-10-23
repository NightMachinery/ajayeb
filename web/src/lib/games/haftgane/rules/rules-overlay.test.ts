// openHaftganeRules() and the overlay's state (rules-overlay.svelte.ts). The
// overlay component itself is replaced by a stub mounter: these tests cover
// the helper Haftgane's screens will call, not the rendering.
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  closeHaftganeRules,
  hgRulesOverlay,
  openHaftganeRules,
  setHgRulesMounter
} from './rules-overlay.svelte';

afterEach(() => {
  closeHaftganeRules();
  setHgRulesMounter(undefined);
});

describe('openHaftganeRules', () => {
  it('opens at a section, and again at the same one with a new request', async () => {
    setHgRulesMounter(async () => {});
    await openHaftganeRules('military');
    expect(hgRulesOverlay.open).toBe(true);
    expect(hgRulesOverlay.section).toBe('military');
    const first = hgRulesOverlay.request;
    await openHaftganeRules('military');
    expect(hgRulesOverlay.request).toBe(first + 1);
  });

  it('opens at the top with no section or an unknown one', async () => {
    setHgRulesMounter(async () => {});
    await openHaftganeRules();
    expect(hgRulesOverlay.section).toBeUndefined();
    await openHaftganeRules('tokens');
    expect(hgRulesOverlay.open).toBe(true);
    expect(hgRulesOverlay.section).toBeUndefined();
  });

  it('mounts the overlay once', async () => {
    const mount = vi.fn(async () => {});
    setHgRulesMounter(mount);
    await openHaftganeRules('science');
    closeHaftganeRules();
    await openHaftganeRules('trade');
    expect(mount).toHaveBeenCalledTimes(1);
  });

  it('tries the mount again after a failed load', async () => {
    const mount = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValue(undefined);
    setHgRulesMounter(mount);
    await expect(openHaftganeRules()).rejects.toThrow('offline');
    await openHaftganeRules();
    expect(mount).toHaveBeenCalledTimes(2);
  });

  it('closes', async () => {
    setHgRulesMounter(async () => {});
    await openHaftganeRules('goal');
    closeHaftganeRules();
    expect(hgRulesOverlay.open).toBe(false);
  });
});
