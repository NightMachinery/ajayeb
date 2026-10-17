import { describe, expect, it } from 'vitest';
import { findAnchor, isDrawn, tut, TUT_ANCHORS } from './anchors';

/** A fake element: a box, an optional data-fx-hidden ancestor, a visibility. */
interface Fake {
  name: string;
  width: number;
  height: number;
  fxHidden?: boolean;
  visibility?: string;
}

function el(f: Fake): Element {
  return {
    name: f.name,
    getBoundingClientRect: () => ({ width: f.width, height: f.height }),
    closest: (sel: string) => (sel === '[data-fx-hidden]' && f.fxHidden ? {} : null),
    visibility: f.visibility ?? 'visible'
  } as unknown as Element;
}

function root(map: Record<string, Element[]>): ParentNode {
  return { querySelectorAll: (sel: string) => map[sel] ?? [] } as unknown as ParentNode;
}

const style = (e: Element) => ({ visibility: (e as unknown as { visibility: string }).visibility });
const visible = (e: Element) => isDrawn(e as Element & Parameters<typeof isDrawn>[0], style);

describe('the tutorial anchors', () => {
  it('name every data-tut id', () => {
    expect(TUT_ANCHORS).toContain('structure');
    expect(tut('turn-bar')).toBe('[data-tut="turn-bar"]');
  });

  it('take the first visible match, in selector order', () => {
    const hidden = el({ name: 'sheet track', width: 0, height: 0 });
    const shown = el({ name: 'board track', width: 120, height: 24 });
    const r = root({ '[data-tut="track"]': [hidden, shown], '.other': [] });
    expect(findAnchor(['[data-tut="track"]'], r, visible)).toBe(shown);
    expect(findAnchor(['.other', '[data-tut="track"]'], r, visible)).toBe(shown);
    expect(findAnchor(['.other'], r, visible)).toBeNull();
  });

  it('skip a box hidden by visibility or by a flying move clone', () => {
    const a = el({ name: 'a', width: 40, height: 40, visibility: 'hidden' });
    const b = el({ name: 'b', width: 40, height: 40, fxHidden: true });
    const c = el({ name: 'c', width: 40, height: 40 });
    expect(findAnchor(['x'], root({ x: [a, b, c] }), visible)).toBe(c);
    expect(findAnchor(['x'], root({ x: [a, b] }), visible)).toBeNull();
  });
});
