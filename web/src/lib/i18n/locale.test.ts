import { describe, expect, it, vi } from 'vitest';
import { LocaleState } from './locale.svelte';
import type { PartialLocaleTable } from './index';

function deferred() {
  let resolve!: (table: PartialLocaleTable) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<PartialLocaleTable>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

describe('on-demand locale messages', () => {
  it('keeps English and its direction until Persian is ready, then caches it', async () => {
    const pending = deferred();
    const load = vi.fn(() => pending.promise);
    const state = new LocaleState(load);
    expect(load).not.toHaveBeenCalled();
    const switched = state.set('fa');
    expect(state.locale).toBe('en');
    expect(state.requested).toBe('fa');
    expect(state.dir).toBe('ltr');
    pending.resolve({ 'app.name': 'عجایب' });
    await switched;
    expect(state.locale).toBe('fa');
    expect(state.requested).toBe('fa');
    expect(state.dir).toBe('rtl');
    expect(state.table?.['app.name']).toBe('عجایب');
    await state.set('en');
    await state.set('fa');
    expect(load).toHaveBeenCalledTimes(1);
    expect(state.locale).toBe('fa');
  });

  it('does not apply a stale Persian request after an English switch', async () => {
    const pending = deferred();
    const load = vi.fn(() => pending.promise);
    const state = new LocaleState(load);
    const first = state.set('fa');
    const second = state.set('fa');
    await state.set('en');
    pending.resolve({ 'app.name': 'عجایب' });
    await Promise.all([first, second]);
    expect(state.locale).toBe('en');
    expect(state.requested).toBe('en');
    expect(state.dir).toBe('ltr');
    expect(load).toHaveBeenCalledTimes(1);
  });

  it('keeps the working language on failure and retries a later request', async () => {
    const load = vi
      .fn<() => Promise<PartialLocaleTable>>()
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce({ 'app.name': 'عجایب' });
    const state = new LocaleState(load);
    await state.set('fa');
    expect(state.locale).toBe('en');
    expect(state.requested).toBe('en');
    expect(state.table).toBeUndefined();
    await state.set('fa');
    expect(state.locale).toBe('fa');
    expect(load).toHaveBeenCalledTimes(2);
  });
});
