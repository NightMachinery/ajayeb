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
  /** Educational tips may let a touch press on an enabled action continue.
   * Tooltips and move summaries never opt in. */
  passTouchActions?: boolean;
  /** The overlay's panel, when its tooltips can be long-pressed (the move
   *  summary): a touch on a tooltip anchor inside it waits for the release,
   *  and dismisses only when no long-press showed the tooltip. A press that
   *  closes an open tooltip, or lands on its bubble, leaves the overlay
   *  open. */
  longPressTips?: () => Element | null | undefined;
}

/** What a press does to an overlay with long-pressable tooltips. */
export type HoldAction =
  /** It belongs to a tooltip (it closed one, or is on its bubble): leave the overlay. */
  | 'ignore'
  /** A touch on a tooltip anchor inside the panel: decide at the release. */
  | 'hold'
  /** Anything else: dismiss now, as any overlay does. */
  | 'dismiss';

/** The press rule for DismissOptions.longPressTips. */
export function holdAction(p: {
  mouse: boolean;
  /** The tooltip layer already swallowed this press to close its tooltip. */
  swallowed: boolean;
  /** The press is on the open tooltip (its bubble or its stack). */
  onBubble: boolean;
  /** The press is on a tooltip anchor inside the overlay's panel. */
  onAnchor: boolean;
}): HoldAction {
  if (p.swallowed || p.onBubble) return 'ignore';
  return !p.mouse && p.onAnchor ? 'hold' : 'dismiss';
}

/** The open tooltip, its bubble and its stack (components/shell/TooltipLayer.svelte). */
const TOOLTIP_OPEN = '[data-tooltip-layer] [data-tooltip-open]';

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
  // A held touch (longPressTips): released with no tooltip open, it was a tap.
  let held: number | null = null;
  const onRelease = (e: PointerEvent): void => {
    if (e.pointerId !== held) return;
    held = null;
    if (e.type === 'pointerup' && document.querySelector(TOOLTIP_OPEN)) return;
    swallowClick();
    opts.onDismiss('tap');
  };
  const onPress = (e: PointerEvent): void => {
    if (inside(e.target)) return;
    const panel = opts.longPressTips?.();
    if (panel) {
      const el = e.target instanceof Element ? e.target : null;
      const action = holdAction({
        mouse: e.pointerType === 'mouse',
        swallowed: armed && armedBy === e,
        onBubble: el?.closest(TOOLTIP_OPEN) != null,
        onAnchor: el !== null && panel.contains(el) && el.closest('[data-tooltip]') !== null
      });
      if (action === 'ignore') return;
      if (action === 'hold') {
        e.stopPropagation();
        held = e.pointerId;
        return;
      }
    }
    if (
      opts.passTouchActions &&
      e.pointerType !== 'mouse' &&
      e.target instanceof Element &&
      e.target.closest('button:not(:disabled), a[href], [data-act]:not(:disabled)')
    ) {
      opts.onDismiss('tap');
      return;
    }
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
  window.addEventListener('pointerup', onRelease, true);
  window.addEventListener('pointercancel', onRelease, true);
  window.addEventListener('keydown', onKey, true);
  return () => {
    window.removeEventListener('pointerdown', onPress, true);
    window.removeEventListener('pointerup', onRelease, true);
    window.removeEventListener('pointercancel', onRelease, true);
    window.removeEventListener('keydown', onKey, true);
  };
}

let quietFocus: Element | null = null;

/**
 * Gives the focus back to `el` without opening its keyboard focus tooltip
 * (the tooltip layer asks isQuietFocus). The focus that returns to an
 * opener is not the user moving it, and a tooltip popping up there would
 * also take the next tap on a touch screen.
 */
export function restoreFocus(el: HTMLElement): void {
  quietFocus = el;
  try {
    el.focus({ preventScroll: true });
  } finally {
    quietFocus = null;
  }
}

/** Whether `el` is taking the focus through restoreFocus right now. */
export function isQuietFocus(el: Element): boolean {
  return quietFocus !== null && el === quietFocus;
}

/** What can take the keyboard focus inside a panel. */
const FOCUSABLE = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
  '[contenteditable="true"]'
].join(', ');

/**
 * Where Tab (or Shift+Tab, `backwards`) goes inside a panel of `count`
 * focusable elements when the focus is on the one at `current` (-1: on the
 * panel itself or elsewhere): the index to focus, -1 for the panel itself
 * (it has nothing focusable), or null to let the browser move it (a step
 * that stays inside).
 */
export function tabTarget(count: number, current: number, backwards: boolean): number | null {
  if (count === 0) return -1;
  if (current < 0 || current >= count) return backwards ? count - 1 : 0;
  if (backwards && current === 0) return count - 1;
  if (!backwards && current === count - 1) return 0;
  return null;
}

/**
 * Whether the focus must go back to the opener once a panel has closed:
 * it was inside the panel (or fell to the page when the panel went away),
 * and the opener is still on the page. A focus that has moved on to
 * something else is left there.
 */
export function shouldRestoreFocus(opts: {
  /** The focus has no element, or is on the page's body. */
  onBody: boolean;
  /** The focus is inside the (closing) panel. */
  inPanel: boolean;
  /** The opener is still in the document. */
  openerConnected: boolean;
}): boolean {
  return opts.openerConnected && (opts.onBody || opts.inPanel);
}

/**
 * The focus management of a dialog-like panel (components/shell/
 * DismissLayer.svelte): it takes the focus now, keeps Tab and Shift+Tab
 * inside while it is open, and the cleanup gives the focus back to the
 * element that had it before, quietly (no focus tooltip opens there). A
 * panel that cannot take the focus (made inert by a modal dialog) is left
 * alone. DOM-only: install it from an effect once the panel is in the
 * document, and return its cleanup.
 */
export function trapFocus(panel: HTMLElement): () => void {
  if (typeof document === 'undefined') return () => {};
  const before = document.activeElement;
  const opener = before instanceof HTMLElement && before !== document.body ? before : null;
  if (!panel.contains(document.activeElement)) panel.focus({ preventScroll: true });
  if (!panel.contains(document.activeElement)) return () => {};

  const focusables = (): HTMLElement[] =>
    [...panel.querySelectorAll<HTMLElement>(FOCUSABLE)].filter(
      (el) => el.getClientRects().length > 0
    );
  const onKey = (e: KeyboardEvent): void => {
    if (e.key !== 'Tab' || e.defaultPrevented || e.altKey || e.ctrlKey || e.metaKey) return;
    const items = focusables();
    const active = document.activeElement;
    const current = active instanceof HTMLElement ? items.indexOf(active) : -1;
    const to = tabTarget(items.length, current, e.shiftKey);
    if (to === null) return;
    e.preventDefault();
    (to < 0 ? panel : (items[to] ?? panel)).focus({ preventScroll: true });
  };
  const onFocusIn = (e: FocusEvent): void => {
    if (e.target instanceof Node && !panel.contains(e.target)) panel.focus({ preventScroll: true });
  };
  document.addEventListener('keydown', onKey, true);
  document.addEventListener('focusin', onFocusIn, true);
  return () => {
    document.removeEventListener('keydown', onKey, true);
    document.removeEventListener('focusin', onFocusIn, true);
    const active = document.activeElement;
    const restore = shouldRestoreFocus({
      onBody: !active || active === document.body,
      inPanel: !!active && panel.contains(active),
      openerConnected: !!opener?.isConnected
    });
    if (restore && opener) restoreFocus(opener);
  };
}
