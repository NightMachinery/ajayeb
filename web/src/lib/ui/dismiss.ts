// "Any tap closes it, and the tap is swallowed": the dismiss behaviour that
// the tooltip layer and the floating panels share (components/shell/
// TooltipLayer.svelte and DismissLayer.svelte). A tap that closes an
// overlay must not also act on whatever lies beneath it, so the press is
// stopped before any component sees it and the click that follows it is
// cancelled.
//
// DOM-only. Install it from an effect and return its cleanup.

let armedBy: Event | null = null;
let armed = false;
let expiry: ReturnType<typeof setTimeout> | undefined;
let listening = false;

/** How long after the finger lifts a swallowed press's click may still
 *  arrive. Browsers send it within a frame or two; a press that became a
 *  scroll sends none, and the next press disarms anyway. */
const CLICK_GRACE_MS = 700;

function disarm(): void {
  armed = false;
  armedBy = null;
  if (expiry !== undefined) clearTimeout(expiry);
  expiry = undefined;
}

function onClick(e: MouseEvent): void {
  if (!armed) return;
  // A click from the keyboard (Enter or Space on a button) has no pointer
  // type; it was never part of the swallowed press.
  if ('pointerType' in e && (e as PointerEvent).pointerType === '') return;
  e.preventDefault();
  e.stopImmediatePropagation();
  disarm();
}

function onDown(e: Event): void {
  // A new press disarms an old swallow, unless this very press armed it.
  if (armed && e !== armedBy) disarm();
}

function onUp(): void {
  if (!armed) return;
  if (expiry !== undefined) clearTimeout(expiry);
  expiry = setTimeout(disarm, CLICK_GRACE_MS);
}

function listen(): void {
  if (listening || typeof window === 'undefined') return;
  listening = true;
  window.addEventListener('click', onClick, true);
  window.addEventListener('pointerdown', onDown, true);
  window.addEventListener('pointerup', onUp, true);
  window.addEventListener('pointercancel', onUp, true);
}

/** Cancels the click that ends the current press (`by` is the press's
 *  pointerdown, when the swallow starts there). */
export function swallowClick(by?: Event): void {
  listen();
  armed = true;
  armedBy = by ?? null;
  if (expiry !== undefined) clearTimeout(expiry);
  expiry = undefined;
}

export interface DismissOptions {
  /** Called once per dismissing tap or Escape. */
  onDismiss: (reason: 'tap' | 'escape') => void;
  /** Taps inside these elements are not dismissals: they reach their
   *  target as usual (a coach mark's Next button, for example). */
  keep?: () => readonly (Element | null | undefined)[];
  /** Escape dismisses too (default true). */
  escape?: boolean;
}

/**
 * Makes any press outside the kept elements dismiss the overlay and be
 * swallowed: neither the press nor its click reaches the page. Escape
 * dismisses as well. Returns the cleanup.
 */
export function installDismiss(opts: DismissOptions): () => void {
  if (typeof window === 'undefined') return () => {};
  listen();
  const inside = (target: EventTarget | null): boolean => {
    if (!(target instanceof Node)) return false;
    return (opts.keep?.() ?? []).some((el) => el?.contains(target));
  };
  const onPress = (e: PointerEvent): void => {
    if (inside(e.target)) return;
    e.stopPropagation();
    swallowClick(e);
    opts.onDismiss('tap');
  };
  const onKey = (e: KeyboardEvent): void => {
    if (e.key !== 'Escape' || opts.escape === false) return;
    e.preventDefault();
    e.stopPropagation();
    opts.onDismiss('escape');
  };
  window.addEventListener('pointerdown', onPress, true);
  window.addEventListener('keydown', onKey, true);
  return () => {
    window.removeEventListener('pointerdown', onPress, true);
    window.removeEventListener('keydown', onKey, true);
  };
}
