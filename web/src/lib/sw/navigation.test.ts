import { afterEach, describe, expect, it, vi } from 'vitest';
import { networkFirst } from './navigation';

afterEach(() => vi.useRealTimers());

describe('network-first navigation', () => {
  it.each([200, 404, 500, 502, 503, 504])(
    'preserves a network response with status %i',
    async (status) => {
      const response = new Response('network', { status });
      const shell = vi.fn(async () => new Response('cached'));
      expect(await networkFirst(async () => response, shell, 4000)).toBe(response);
      expect(shell).not.toHaveBeenCalled();
    }
  );

  it('opens the cached shell when the connection fails', async () => {
    const shell = new Response('cached');
    expect(
      await networkFirst(
        () => Promise.reject(new TypeError('offline')),
        async () => shell,
        4000
      )
    ).toBe(shell);
  });

  it('opens the cached shell after the short timeout', async () => {
    vi.useFakeTimers();
    const shell = new Response('cached');
    let rejectNetwork: (reason: Error) => void = () => {};
    const result = networkFirst(
      () =>
        new Promise<Response>((_, reject) => {
          rejectNetwork = reject;
        }),
      async () => shell,
      4000
    );
    await vi.advanceTimersByTimeAsync(4000);
    expect(await result).toBe(shell);
    // A later failure is consumed, without an unhandled rejection.
    rejectNetwork(new TypeError('offline'));
    await vi.runAllTimersAsync();
  });

  it('preserves the network failure if no shell was installed', async () => {
    const error = new TypeError('offline');
    await expect(
      networkFirst(
        () => Promise.reject(error),
        async () => undefined,
        4000
      )
    ).rejects.toBe(error);
  });
});
