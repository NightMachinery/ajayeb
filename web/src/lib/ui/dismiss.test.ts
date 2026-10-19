import { describe, expect, it } from 'vitest';
import { holdAction, isQuietFocus, restoreFocus, shouldRestoreFocus, tabTarget } from './dismiss';

describe('tabTarget: Tab stays inside an open panel', () => {
  it('wraps from the last element to the first, and back with Shift+Tab', () => {
    expect(tabTarget(3, 2, false)).toBe(0);
    expect(tabTarget(3, 0, true)).toBe(2);
  });

  it('lets the browser take a step that stays inside', () => {
    expect(tabTarget(3, 0, false)).toBeNull();
    expect(tabTarget(3, 2, true)).toBeNull();
    expect(tabTarget(3, 1, true)).toBeNull();
  });

  it('enters at the first (or, backwards, the last) element from the panel itself', () => {
    expect(tabTarget(3, -1, false)).toBe(0);
    expect(tabTarget(3, -1, true)).toBe(2);
  });

  it('keeps the focus on the panel when it has nothing focusable', () => {
    expect(tabTarget(0, -1, false)).toBe(-1);
    expect(tabTarget(0, -1, true)).toBe(-1);
  });
});

describe('shouldRestoreFocus: the opener gets the focus back', () => {
  it('when the focus was inside the panel or fell to the page', () => {
    expect(shouldRestoreFocus({ onBody: true, inPanel: false, openerConnected: true })).toBe(true);
    expect(shouldRestoreFocus({ onBody: false, inPanel: true, openerConnected: true })).toBe(true);
  });

  it('not when the focus has moved on, or the opener is gone', () => {
    expect(shouldRestoreFocus({ onBody: false, inPanel: false, openerConnected: true })).toBe(
      false
    );
    expect(shouldRestoreFocus({ onBody: true, inPanel: false, openerConnected: false })).toBe(
      false
    );
  });
});

describe('restoreFocus: the opener gets the focus back quietly', () => {
  it('marks the element as quietly focused only while it takes the focus', () => {
    let during = false;
    const el = {
      focus: () => {
        during = isQuietFocus(el as unknown as Element);
      }
    } as unknown as HTMLElement;
    restoreFocus(el);
    expect(during).toBe(true);
    expect(isQuietFocus(el)).toBe(false);
  });
});

describe('holdAction: long-pressable tooltips in an overlay (the move summary)', () => {
  const press = { mouse: false, swallowed: false, onBubble: false, onAnchor: false };

  it('holds a touch on a tooltip anchor in the panel until its release', () => {
    expect(holdAction({ ...press, onAnchor: true })).toBe('hold');
  });

  it('dismisses at once on any other press, and on a mouse press on an anchor', () => {
    expect(holdAction(press)).toBe('dismiss');
    expect(holdAction({ ...press, mouse: true, onAnchor: true })).toBe('dismiss');
  });

  it('leaves the overlay to a press that closed a tooltip or landed on it', () => {
    expect(holdAction({ ...press, swallowed: true, onAnchor: true })).toBe('ignore');
    expect(holdAction({ ...press, onBubble: true })).toBe('ignore');
    expect(holdAction({ ...press, mouse: true, onBubble: true })).toBe('ignore');
  });
});
