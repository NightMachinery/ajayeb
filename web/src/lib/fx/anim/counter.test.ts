import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { CounterTween } from './counter';

// A text node and a frame loop the test drives by hand.
let frames: (() => void)[] = [];
let now = 0;
beforeEach(() => {
  frames = [];
  now = 0;
  (globalThis as { requestAnimationFrame?: unknown }).requestAnimationFrame = (f: () => void) =>
    frames.push(f);
  (globalThis as { cancelAnimationFrame?: unknown }).cancelAnimationFrame = () => {};
});
afterEach(() => {
  delete (globalThis as { requestAnimationFrame?: unknown }).requestAnimationFrame;
  delete (globalThis as { cancelAnimationFrame?: unknown }).cancelAnimationFrame;
});

function at(t: number): void {
  now = t;
  const run = frames;
  frames = [];
  for (const f of run) f();
}

function text(value: string): Text {
  return { nodeValue: value } as Text;
}

describe('CounterTween', () => {
  it('chains its segments, then puts the server’s text back', () => {
    const node = text('13');
    let ended = 0;
    const c = new CounterTween(
      node,
      '13',
      () => now,
      () => (ended += 1)
    );
    c.add(100, 100, 20, 18);
    c.add(300, 100, 18, 13);
    expect(node.nodeValue).toBe('20');
    at(150);
    expect(Number(node.nodeValue)).toBeGreaterThan(18);
    at(250);
    expect(node.nodeValue).toBe('18');
    at(400);
    expect(node.nodeValue).toBe('13');
    expect(ended).toBe(1);
    expect(c.done).toBe(true);
  });

  it('stopped early, it shows the server’s text at once', () => {
    const node = text('9');
    const c = new CounterTween(node, '9', () => now);
    c.add(0, 450, 7, 9);
    expect(node.nodeValue).toBe('7');
    c.stop();
    expect(node.nodeValue).toBe('9');
    c.add(0, 100, 1, 2);
    expect(node.nodeValue).toBe('9');
  });
});
