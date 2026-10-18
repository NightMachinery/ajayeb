import { describe, expect, it, vi } from 'vitest';
import { listenForImage } from './image-events';

function image(complete = false, naturalWidth = 0): HTMLImageElement {
  return Object.assign(new EventTarget(), { complete, naturalWidth }) as HTMLImageElement;
}

describe('art image lifecycle', () => {
  it('settles a cached success that completed before hydration', async () => {
    const loaded = vi.fn(),
      failed = vi.fn();
    const stop = listenForImage(image(true, 800), loaded, failed);
    await Promise.resolve();
    expect(loaded).toHaveBeenCalledTimes(1);
    expect(failed).not.toHaveBeenCalled();
    stop();
  });

  it('settles a failure that completed before hydration', async () => {
    const loaded = vi.fn(),
      failed = vi.fn();
    const stop = listenForImage(image(true), loaded, failed);
    await Promise.resolve();
    expect(failed).toHaveBeenCalledTimes(1);
    expect(loaded).not.toHaveBeenCalled();
    stop();
  });

  it('waits for ordinary lazy image events and removes listeners on cleanup', () => {
    const pending = image();
    const loaded = vi.fn(),
      failed = vi.fn();
    const stop = listenForImage(pending, loaded, failed);
    expect(loaded).not.toHaveBeenCalled();
    expect(failed).not.toHaveBeenCalled();
    pending.dispatchEvent(new Event('load'));
    expect(loaded).toHaveBeenCalledTimes(1);
    stop();
    pending.dispatchEvent(new Event('load'));
    pending.dispatchEvent(new Event('error'));
    expect(loaded).toHaveBeenCalledTimes(1);
    expect(failed).not.toHaveBeenCalled();
  });

  it('cancels stale completion and listeners when an image frame changes source', async () => {
    const reused = image(true);
    const previous = vi.fn(),
      current = vi.fn();
    const stopOld = listenForImage(reused, previous, previous);
    stopOld();
    // The same element now holds a new, still-loading source.
    Object.assign(reused, { complete: false });
    const stopNew = listenForImage(reused, current, current);
    await Promise.resolve();
    expect(previous).not.toHaveBeenCalled();
    expect(current).not.toHaveBeenCalled();
    reused.dispatchEvent(new Event('load'));
    expect(current).toHaveBeenCalledTimes(1);
    expect(previous).not.toHaveBeenCalled();
    stopNew();
  });
});
