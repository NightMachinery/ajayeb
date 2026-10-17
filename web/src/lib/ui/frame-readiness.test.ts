import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { createFrameReadiness } from './frame-readiness';

let frames: Map<number, FrameRequestCallback>;
let nextFrame: number;
beforeEach(() => {
  frames = new Map();
  nextFrame = 0;
  vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
    frames.set(++nextFrame, callback);
    return nextFrame;
  });
  vi.stubGlobal('cancelAnimationFrame', (id: number) => frames.delete(id));
});
afterEach(() => vi.unstubAllGlobals());

function setup() {
  const target = new EventTarget();
  const face = { status: 'loaded' };
  const faces = [face];
  let resolve: () => void = () => {};
  let promise: Promise<unknown> = Promise.resolve();
  const getter = vi.fn(() => promise);
  const fonts = Object.assign(target, { status: 'loaded', values: () => faces.values() });
  Object.defineProperty(fonts, 'ready', { get: getter });
  const root = { dataset: {} } as HTMLElement;
  const mark = createFrameReadiness();
  return {
    root,
    fonts,
    faces,
    face,
    getter,
    mark: (locale = 'en') => mark(root, fonts as unknown as FontFaceSet, locale),
    pending() {
      fonts.status = 'loading';
      face.status = 'loading';
      promise = new Promise((done) => {
        resolve = () => done(fonts);
      });
    },
    finish() {
      fonts.status = 'loaded';
      face.status = 'loaded';
      resolve();
    }
  };
}
async function paint() {
  await Promise.resolve();
  const pending = [...frames.values()];
  frames.clear();
  for (const callback of pending) callback(0);
}

it('reuses the getter for settled faces but stamps each current frame only at rAF', async () => {
  const f = setup();
  const cancel = f.mark();
  expect(f.root.dataset['ready']).toBeUndefined();
  await paint();
  expect(f.root.dataset['ready']).toBe('1');
  cancel();
  expect(f.root.dataset['ready']).toBeUndefined();
  f.mark();
  expect(f.getter).toHaveBeenCalledTimes(1);
  expect(f.root.dataset['ready']).toBeUndefined();
  await paint();
  expect(f.root.dataset['ready']).toBe('1');
});

it('waits for pending initial fonts and cancels a stale promise completion', async () => {
  const f = setup();
  f.pending();
  const cancel = f.mark();
  await paint();
  expect(f.root.dataset['ready']).toBeUndefined();
  cancel();
  f.finish();
  await paint();
  expect(frames.size).toBe(0);
  expect(f.root.dataset['ready']).toBeUndefined();
});

it('cancels an already queued rAF when the frame becomes stale', async () => {
  const f = setup();
  const cancel = f.mark();
  await Promise.resolve();
  expect(frames.size).toBe(1);
  cancel();
  expect(frames.size).toBe(0);
  await paint();
  expect(f.root.dataset['ready']).toBeUndefined();
});

it('invalidates on locale changes and on newly registered unloaded faces', async () => {
  const f = setup();
  f.mark()();
  f.mark('fa')();
  f.faces.push({ status: 'unloaded' });
  f.mark('fa')();
  f.mark('fa')();
  expect(f.getter).toHaveBeenCalledTimes(4);
});

it('clears a live stamp for a later load and waits for its new promise', async () => {
  const f = setup();
  const cancel = f.mark();
  await paint();
  f.pending();
  f.fonts.dispatchEvent(new Event('loading'));
  expect(f.root.dataset['ready']).toBeUndefined();
  expect(f.getter).toHaveBeenCalledTimes(2);
  await paint();
  expect(f.root.dataset['ready']).toBeUndefined();
  f.finish();
  await paint();
  expect(f.root.dataset['ready']).toBe('1');
  cancel();
  f.fonts.dispatchEvent(new Event('loading'));
  expect(f.getter).toHaveBeenCalledTimes(2);
});

it('detects a load before its loading event is delivered and never stamps it ready', async () => {
  const f = setup();
  f.mark();
  await Promise.resolve();
  f.pending();
  await paint();
  expect(f.root.dataset['ready']).toBeUndefined();
  expect(f.getter).toHaveBeenCalledTimes(2);
  f.finish();
  await paint();
  expect(f.root.dataset['ready']).toBe('1');
});
