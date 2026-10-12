import { describe, expect, it } from 'vitest';
import { boardUndoKey, isUndoKey, MODAL_OPEN_SELECTOR, modalOpen, type UndoKey } from './undo-keys';

const press = (key: string, mods: Partial<UndoKey> = {}): UndoKey => ({
  key,
  ctrlKey: false,
  metaKey: false,
  shiftKey: false,
  altKey: false,
  repeat: false,
  defaultPrevented: false,
  ...mods
});

const board = { typing: false, modalOpen: false };

describe('the undo shortcut', () => {
  it('is U, Ctrl+Z and Cmd+Z', () => {
    expect(isUndoKey(press('u'))).toBe(true);
    expect(isUndoKey(press('U'))).toBe(true);
    expect(isUndoKey(press('z', { ctrlKey: true }))).toBe(true);
    expect(isUndoKey(press('z', { metaKey: true }))).toBe(true);
  });

  it('is not a plain Z, a modified U, a redo, a repeat or a handled press', () => {
    expect(isUndoKey(press('z'))).toBe(false);
    expect(isUndoKey(press('u', { ctrlKey: true }))).toBe(false);
    expect(isUndoKey(press('u', { metaKey: true }))).toBe(false);
    expect(isUndoKey(press('z', { ctrlKey: true, shiftKey: true }))).toBe(false);
    expect(isUndoKey(press('u', { altKey: true }))).toBe(false);
    expect(isUndoKey(press('u', { repeat: true }))).toBe(false);
    expect(isUndoKey(press('u', { defaultPrevented: true }))).toBe(false);
  });

  it('does nothing in a text field or under an open sheet or dialog', () => {
    expect(boardUndoKey(press('u'), board)).toBe(true);
    expect(boardUndoKey(press('u'), { ...board, typing: true })).toBe(false);
    expect(boardUndoKey(press('u'), { ...board, modalOpen: true })).toBe(false);
    expect(boardUndoKey(press('z', { metaKey: true }), { ...board, modalOpen: true })).toBe(false);
  });

  it('finds an open modal layer with one selector', () => {
    const seen: string[] = [];
    const root = (found: boolean) => ({
      querySelector: (sel: string) => {
        seen.push(sel);
        return found ? ({} as Element) : null;
      }
    });
    expect(modalOpen(root(true))).toBe(true);
    expect(modalOpen(root(false))).toBe(false);
    expect(seen).toEqual([MODAL_OPEN_SELECTOR, MODAL_OPEN_SELECTOR]);
    expect(MODAL_OPEN_SELECTOR).toContain('dialog:modal');
    expect(MODAL_OPEN_SELECTOR).toContain('[data-dismiss-layer]');
  });
});
