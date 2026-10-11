// Swipe down to dismiss a bottom sheet (Sheet.svelte). A drag starts on the
// handle at any time, or anywhere in the panel when the panel is scrolled
// to its top and the finger moves down; an upward move, or a downward one
// while the panel can still scroll up, stays an ordinary scroll. While it
// drags, the sheet follows the finger and the page under it neither
// overscrolls nor pulls to refresh. On release it closes when it travelled
// far enough or was flicked, and otherwise springs back; both moves use the
// sheet's own CSS transition, so the Animation setting governs them.
//
// Touch events, not pointer events: a pointer drag inside a vertically
// scrollable panel is taken over by the browser's pan (pointercancel), and
// only a non-passive touchmove can claim it with preventDefault.

/** Finger travel in px before a downward move counts as a drag. */
export const DRAG_START_PX = 6;

/** When this media query matches, Sheet shows a centred panel, not a bottom
 *  sheet (it mirrors @custom-variant split in app.css), so there is
 *  nothing to swipe. */
export const SPLIT_QUERY = '(min-width: 64rem), (orientation: landscape) and (max-height: 32rem)';

/** Whether a released drag closes the sheet: it travelled a quarter of the
 *  panel's height (at most 120 px), or a flick of at least 32 px moved
 *  faster than 0.5 px per millisecond. */
export function shouldDismiss(offsetPx: number, elapsedMs: number, panelHeightPx: number): boolean {
  if (offsetPx <= 0) return false;
  if (offsetPx >= Math.min(120, panelHeightPx * 0.25)) return true;
  return offsetPx >= 32 && offsetPx / Math.max(elapsedMs, 1) >= 0.5;
}

/** Wires the gesture to a sheet's dialog and its scrolling panel. canDrag
 *  is asked at the start of every touch (false for a centred panel).
 *  Returns the cleanup. */
export function attachSheetDrag(
  dialog: HTMLDialogElement,
  panel: HTMLElement,
  canDrag: () => boolean
): () => void {
  let tracking = false;
  let dragging = false;
  let fromHandle = false;
  let startY = 0;
  let startTime = 0;
  let offset = 0;

  function onStart(event: TouchEvent): void {
    tracking = event.touches.length === 1 && dialog.open && canDrag();
    dragging = false;
    if (!tracking) return;
    startY = event.touches[0]!.clientY;
    startTime = event.timeStamp;
    offset = 0;
    fromHandle =
      event.target instanceof Element && event.target.closest('[data-sheet-handle]') !== null;
  }

  function onMove(event: TouchEvent): void {
    if (!tracking || event.touches.length !== 1) return;
    const dy = event.touches[0]!.clientY - startY;
    if (!dragging) {
      if (dy <= -DRAG_START_PX) tracking = false; // an upward scroll
      if (dy < DRAG_START_PX) return;
      if (!fromHandle && panel.scrollTop > 0) {
        tracking = false; // the panel scrolls back up first
        return;
      }
      dragging = true;
      dialog.style.transition = 'none';
    }
    event.preventDefault();
    offset = Math.max(0, dy - DRAG_START_PX);
    dialog.style.translate = `0 ${offset}px`;
  }

  function onEnd(event: TouchEvent): void {
    if (!tracking) return;
    tracking = false;
    if (!dragging) return;
    dragging = false;
    const close =
      event.type === 'touchend' &&
      shouldDismiss(offset, event.timeStamp - startTime, panel.getBoundingClientRect().height);
    // Dropping both inline values in one go lets the CSS transition run
    // from where the finger left the sheet: off screen when closing (the
    // closed sheet's translate is 100%), back to 0 otherwise.
    dialog.style.removeProperty('transition');
    dialog.style.removeProperty('translate');
    if (close) dialog.close(); // its 'close' event syncs Sheet's open prop
  }

  panel.addEventListener('touchstart', onStart, { passive: true });
  panel.addEventListener('touchmove', onMove, { passive: false });
  panel.addEventListener('touchend', onEnd);
  panel.addEventListener('touchcancel', onEnd);
  return () => {
    panel.removeEventListener('touchstart', onStart);
    panel.removeEventListener('touchmove', onMove);
    panel.removeEventListener('touchend', onEnd);
    panel.removeEventListener('touchcancel', onEnd);
  };
}
