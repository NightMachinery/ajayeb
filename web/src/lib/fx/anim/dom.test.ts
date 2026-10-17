// The DOM helpers of the animation runner that do arithmetic: where a box
// inside a dialog comes to rest while the dialog is still sliding in.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { settled } from './dom';

function dialogAt(rect: { x: number; y: number; w: number; h: number }) {
  const dialog = {
    getBoundingClientRect: () => ({
      x: rect.x,
      y: rect.y,
      left: rect.x,
      top: rect.y,
      width: rect.w,
      height: rect.h
    }),
    closest: () => dialog
  };
  return dialog;
}

function inside(dialog: ReturnType<typeof dialogAt>): Element {
  return { closest: () => dialog } as unknown as Element;
}

afterEach(() => vi.unstubAllGlobals());

describe('settled', () => {
  it('takes off what a sliding sheet still has to travel', () => {
    vi.stubGlobal('getComputedStyle', () => ({ translate: '0px 300px', scale: 'none' }));
    const dialog = dialogAt({ x: 0, y: 400, w: 360, h: 400 });
    const box = settled(inside(dialog), { x: 20, y: 500, w: 40, h: 40 });
    expect(box).toEqual({ x: 20, y: 200, w: 40, h: 40 });
  });

  it('reads a percentage of the dialog’s own height', () => {
    vi.stubGlobal('getComputedStyle', () => ({ translate: '0px 100%', scale: 'none' }));
    // A 400 px sheet resting at 340, still a full height lower.
    const dialog = dialogAt({ x: 0, y: 740, w: 360, h: 400 });
    const box = settled(inside(dialog), { x: 20, y: 800, w: 40, h: 40 });
    expect(box).toEqual({ x: 20, y: 400, w: 40, h: 40 });
  });

  it('undoes a centred panel’s growth about its centre', () => {
    vi.stubGlobal('getComputedStyle', () => ({ translate: 'none', scale: '0.5' }));
    // Rest box 100..300 x 100..300, shown at half size about its centre (200, 200).
    const dialog = dialogAt({ x: 150, y: 150, w: 100, h: 100 });
    const box = settled(inside(dialog), { x: 150, y: 150, w: 10, h: 10 });
    expect(box).toEqual({ x: 100, y: 100, w: 20, h: 20 });
  });

  it('leaves a settled dialog, and anything outside one, as it is', () => {
    vi.stubGlobal('getComputedStyle', () => ({ translate: '0px 0px', scale: '1' }));
    const dialog = dialogAt({ x: 0, y: 0, w: 360, h: 400 });
    const box = { x: 5, y: 6, w: 7, h: 8 };
    expect(settled(inside(dialog), box)).toBe(box);
    const loose = { closest: () => null } as unknown as Element;
    expect(settled(loose, box)).toBe(box);
  });
});
