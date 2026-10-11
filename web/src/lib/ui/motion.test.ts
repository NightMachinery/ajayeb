import { describe, expect, it } from 'vitest';
import { botDwellMs, botPickMs, effectiveSpeed, Motion, scaleMs } from './motion.svelte';
import { SettingsStore } from './settings.svelte';

describe('effectiveSpeed', () => {
  it('passes the preference straight through (D21: the OS preference is not read)', () => {
    expect(effectiveSpeed('normal')).toBe('normal');
    expect(effectiveSpeed('fast')).toBe('fast');
    expect(effectiveSpeed('relaxed')).toBe('relaxed');
    expect(effectiveSpeed('off')).toBe('off');
  });
});

describe('scaleMs', () => {
  it('scales a base duration per speed', () => {
    expect(scaleMs(100, 'normal')).toBe(100);
    expect(scaleMs(100, 'fast')).toBe(60);
    expect(scaleMs(100, 'relaxed')).toBe(160);
  });

  it('is always exactly 0 when off, for any base', () => {
    expect(scaleMs(100, 'off')).toBe(0);
    expect(scaleMs(0, 'off')).toBe(0);
    expect(scaleMs(99999, 'off')).toBe(0);
  });

  it('rounds to the nearest millisecond', () => {
    expect(scaleMs(1, 'fast')).toBe(1); // 0.6 rounds to 1
    expect(scaleMs(5, 'fast')).toBe(3); // 3.0
  });
});

describe('Motion', () => {
  function motionFor(): { store: SettingsStore; motion: Motion } {
    const store = new SettingsStore(null);
    return { store, motion: new Motion({ settings: store }) };
  }

  it('reflects the injected store, not the shared singleton', () => {
    const { store, motion } = motionFor();
    expect(motion.preference).toBe('normal');
    store.set('animationSpeed', 'relaxed');
    expect(motion.preference).toBe('relaxed');
    expect(motion.effective).toBe('relaxed');
  });

  it('ms() tracks the live effective speed', () => {
    const { store, motion } = motionFor();
    expect(motion.ms(100)).toBe(100);
    store.set('animationSpeed', 'fast');
    expect(motion.ms(100)).toBe(60);
    store.set('animationSpeed', 'off');
    expect(motion.ms(100)).toBe(0);
  });

  it('apply() mirrors the effective speed onto data-motion', () => {
    const { store, motion } = motionFor();
    const root = { dataset: {} as Record<string, string> } as unknown as HTMLElement;
    motion.apply(root);
    expect(root.dataset['motion']).toBeUndefined();
    store.set('animationSpeed', 'off');
    motion.apply(root);
    expect(root.dataset['motion']).toBe('reduce');
    store.set('animationSpeed', 'normal');
    motion.apply(root);
    expect(root.dataset['motion']).toBeUndefined();
  });
});

describe('bot pacing', () => {
  it('scales the thinking dwell and the pick highlight with the speed', () => {
    expect(botDwellMs('normal')).toBe(700);
    expect(botDwellMs('fast')).toBe(420);
    expect(botDwellMs('relaxed')).toBe(1120);
    expect(botPickMs('normal')).toBe(300);
    expect(botPickMs('fast')).toBe(180);
    expect(botPickMs('relaxed')).toBe(480);
  });

  it('keeps a short floor with animation off, so the bot can still be followed', () => {
    expect(botDwellMs('off')).toBe(300);
    expect(botPickMs('off')).toBe(300);
  });
});
