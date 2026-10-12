// The keyboard undo of the game screen: U, Ctrl+Z, or Cmd+Z on a Mac. A
// display rule only: whether the game has undo and whether there is a move
// to take back stay the caller's (the room's allowUndo and the view's
// canUndo).
//
// The shortcut belongs to the board, so it does nothing while something
// else has the keyboard: a text field, or a modal sheet or dialog on top of
// the board (the Hints sheet, Settings, the city sheet, the opponent's move
// summary). A press there must never take back the move underneath.

/** The fields of a keydown event the rule reads. */
export interface UndoKey {
  key: string;
  ctrlKey: boolean;
  metaKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
  repeat: boolean;
  defaultPrevented: boolean;
}

/** An open modal layer over the board: a sheet (a modal <dialog>), any
 *  other modal dialog, or a dismiss layer (components/shell/DismissLayer). */
export const MODAL_OPEN_SELECTOR = 'dialog:modal, [aria-modal="true"], [data-dismiss-layer]';

/** Whether a key press is the undo shortcut. */
export function isUndoKey(event: UndoKey): boolean {
  if (event.defaultPrevented || event.repeat || event.altKey || event.shiftKey) return false;
  const key = event.key.toLowerCase();
  const plainU = key === 'u' && !event.ctrlKey && !event.metaKey;
  const ctrlZ = key === 'z' && (event.ctrlKey || event.metaKey);
  return plainU || ctrlZ;
}

/** Whether a key press reaches the board's undo: the shortcut, typed
 *  neither in a text field nor under an open modal layer. */
export function boardUndoKey(
  event: UndoKey,
  where: { typing: boolean; modalOpen: boolean }
): boolean {
  return !where.typing && !where.modalOpen && isUndoKey(event);
}

/** Whether a modal layer is open in the document (DOM-only). */
export function modalOpen(root: Pick<ParentNode, 'querySelector'>): boolean {
  return root.querySelector(MODAL_OPEN_SELECTOR) !== null;
}
