// The tooltip helper: our own replacement for the native title attribute
// (AGENTS.md, "Tooltips"). A component registers an anchor with an
// attachment,
//
//   <span {@attach tooltip({ text: t('board.resource.chip', { ... }) })}>
//
// and the one tooltip layer (components/shell/TooltipLayer.svelte, mounted
// once in the root layout) shows it:
//
// - Fine pointer: after a short hover delay, and at once on keyboard focus
//   (focus-visible only, so a mouse click does not pop one up). Escape
//   closes it.
// - Touch (model B): a tap on an element with no action shows it; an
//   actionable element keeps its tap action and shows it on a long-press,
//   whose release never acts. Any tap while one is open closes it and is
//   swallowed.
// - The innermost registered anchor under the pointer or finger wins, and
//   only one tooltip shows at a time.
//
// The content is text (through t()), with an optional diagram snippet, an
// optional stack of linked previews (a second panel attached to the bubble,
// such as the card a chain symbol makes free), and an optional rules
// section id for the rules reference's "More in Rules" link.

import { untrack, type Snippet } from 'svelte';
import type { Attachment } from 'svelte/attachments';
import { swallowClick } from './dismiss';
import {
  HOVER_DELAY_MS,
  HOVER_HIDE_MS,
  HOVER_SKIP_MS,
  isActionable,
  LONG_PRESS_MS,
  Press,
  resolveAnchor
} from './tooltip-gesture';

/** One linked preview in a tooltip's stack. */
export type TooltipStackItem =
  /** A building card, drawn as a real full-tier CardFace. */
  | { kind: 'card'; slug: string; caption: string }
  /** Anything else (a wonder, a token): the caller draws it. */
  | { kind: 'custom'; caption: string; render: Snippet };

export interface TooltipContent {
  /** The tooltip's text, already translated. Short: one or two lines. */
  text: string;
  /** A small concept diagram under the text. Its labels must be
   *  translatable text, not baked into the SVG. */
  diagram?: Snippet;
  /** Linked previews, shown in a panel attached to the bubble (beside it on
   *  wide screens, on its far side on phones). At most two items; the stack
   *  itself has no stack, so a tooltip is never more than two levels deep. */
  stack?: readonly TooltipStackItem[];
  /** The rules reference section this tooltip's topic belongs to, for a
   *  later "More in Rules" link. */
  rulesId?: string;
  /** Whether a tap on the anchor acts, when the DOM cannot tell (the
   *  default looks for a button, a link, a data-act and the like on the
   *  anchor or an ancestor). */
  actionable?: boolean;
}

export type TooltipVia = 'hover' | 'focus' | 'touch' | 'api';

export interface ActiveTooltip {
  /** Unique per showing, so the layer can restart its fade. */
  id: number;
  anchor: Element;
  content: TooltipContent;
  /** The anchor's box when it was shown. */
  rect: DOMRect;
  via: TooltipVia;
}

/** The bubble's element id, for aria-describedby. */
export const TOOLTIP_ID = 'ajayeb-tooltip';
/** The attribute every registered anchor carries. */
export const ANCHOR_ATTR = 'data-tooltip';

const FOCUSABLE =
  'a[href], button, input, select, textarea, summary, [tabindex]:not([tabindex="-1"])';

class TooltipStore {
  /** The tooltip on screen, if any. Replaced whole. */
  active: ActiveTooltip | null = $state.raw(null);

  #registry = new WeakMap<Element, TooltipContent>();
  #seq = 0;
  #closedAt = -Infinity;

  /** Registers an anchor, or updates its content (a re-run attachment). */
  register(el: Element, content: TooltipContent): void {
    this.#registry.set(el, content);
    el.setAttribute(ANCHOR_ATTR, '');
    // An open tooltip whose anchor re-registered shows the new content. The
    // write waits for a microtask, outside the attachment's effect (state is
    // never written from inside an effect).
    // untrack: the attachment that calls this must not re-run whenever a
    // tooltip opens or closes.
    if (untrack(() => this.active?.anchor === el)) {
      queueMicrotask(() => {
        const active = this.active;
        if (active?.anchor === el && this.#registry.get(el) === content) {
          this.active = { ...active, content };
        }
      });
    }
  }

  unregister(el: Element, content: TooltipContent): void {
    // A re-run attachment registers its new content right after this
    // cleanup; only a stale entry is removed.
    if (this.#registry.get(el) !== content) return;
    this.#registry.delete(el);
    el.removeAttribute(ANCHOR_ATTR);
    // The anchor left the page (or lost its tooltip): close after the
    // current update settles, unless it registered again meanwhile.
    queueMicrotask(() => {
      if (this.active?.anchor === el && (!el.isConnected || !this.#registry.has(el))) this.hide();
    });
  }

  contentOf(el: Element): TooltipContent | undefined {
    return this.#registry.get(el);
  }

  /** The innermost registered anchor at or above `target`. */
  anchorFor(target: EventTarget | null): Element | null {
    const start = target instanceof Element ? target : null;
    return resolveAnchor<Element>(
      start,
      (node) => node.parentElement,
      (node) => this.#registry.has(node)
    );
  }

  show(anchor: Element, via: TooltipVia): void {
    const content = this.#registry.get(anchor);
    if (!content) return;
    if (this.active && this.active.anchor !== anchor) this.#undescribe(this.active.anchor);
    this.active = {
      id: ++this.#seq,
      anchor,
      content,
      rect: anchor.getBoundingClientRect(),
      via
    };
    // A focusable anchor points at the bubble while it shows. Micro-elements
    // inside a focusable card are not focusable themselves; the card's own
    // accessible description covers them.
    if (anchor.matches(FOCUSABLE)) {
      const ids = (anchor.getAttribute('aria-describedby') ?? '').split(/\s+/).filter(Boolean);
      if (!ids.includes(TOOLTIP_ID)) {
        anchor.setAttribute('aria-describedby', [...ids, TOOLTIP_ID].join(' '));
      }
    }
  }

  hide(): void {
    if (!this.active) return;
    this.#undescribe(this.active.anchor);
    this.active = null;
    this.#closedAt = performance.now();
  }

  /** Whether a tooltip closed so recently that the next one should skip
   *  the hover delay. */
  get warm(): boolean {
    return this.active !== null || performance.now() - this.#closedAt < HOVER_SKIP_MS;
  }

  #undescribe(anchor: Element): void {
    const ids = (anchor.getAttribute('aria-describedby') ?? '')
      .split(/\s+/)
      .filter((id) => id && id !== TOOLTIP_ID);
    if (ids.length > 0) anchor.setAttribute('aria-describedby', ids.join(' '));
    else anchor.removeAttribute('aria-describedby');
  }
}

/** The app's one tooltip store. */
export const tooltips = new TooltipStore();

/**
 * The attachment that gives an element a tooltip. Pass undefined to give it
 * none (a conditional tooltip). When the content changes, Svelte re-runs
 * the attachment and an open tooltip updates in place.
 */
export function tooltip(content: TooltipContent | undefined): Attachment {
  return (el) => {
    if (!content) return;
    untrack(() => tooltips.register(el, content));
    return () => tooltips.unregister(el, content);
  };
}

/**
 * Wires the gestures to the store: pointer, touch, focus, Escape and
 * resize. The layer installs it once, from an effect; returns the
 * cleanup.
 */
export function installTooltipGestures(store: TooltipStore = tooltips): () => void {
  let hoverTimer: ReturnType<typeof setTimeout> | undefined;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;
  let pressTimer: ReturnType<typeof setTimeout> | undefined;
  let hovered: Element | null = null;
  let press: { state: Press; anchor: Element; pointerId: number } | null = null;

  const clearHover = (): void => {
    clearTimeout(hoverTimer);
    clearTimeout(hideTimer);
    hoverTimer = hideTimer = undefined;
  };
  const endPress = (): void => {
    clearTimeout(pressTimer);
    pressTimer = undefined;
    press = null;
  };

  const onOver = (e: PointerEvent): void => {
    if (e.pointerType !== 'mouse') return;
    const anchor = store.anchorFor(e.target);
    if (anchor === hovered) return;
    hovered = anchor;
    clearHover();
    if (!anchor) {
      if (store.active?.via === 'hover') hideTimer = setTimeout(() => store.hide(), HOVER_HIDE_MS);
      return;
    }
    // A keyboard focus tooltip stays until focus moves.
    if (store.active?.via === 'focus') return;
    if (store.warm) store.show(anchor, 'hover');
    else hoverTimer = setTimeout(() => store.show(anchor, 'hover'), HOVER_DELAY_MS);
  };

  const onOut = (e: PointerEvent): void => {
    if (e.pointerType !== 'mouse' || e.relatedTarget !== null) return;
    // The pointer left the window.
    hovered = null;
    clearHover();
    if (store.active?.via === 'hover') store.hide();
  };

  const onDown = (e: PointerEvent): void => {
    clearHover();
    endPress();
    if (e.pointerType === 'mouse') {
      // A click acts as usual; it only closes a hover tooltip.
      if (store.active && store.active.via !== 'focus') store.hide();
      return;
    }
    // Touch or pen. Any tap while a tooltip is open closes it, and is
    // swallowed: neither the press nor its click reaches the page.
    if (store.active) {
      e.stopPropagation();
      swallowClick(e);
      store.hide();
      return;
    }
    const anchor = store.anchorFor(e.target);
    if (!anchor) return;
    const content = store.contentOf(anchor);
    const state = new Press(
      e.clientX,
      e.clientY,
      e.timeStamp,
      isActionable(anchor, content?.actionable)
    );
    press = { state, anchor, pointerId: e.pointerId };
    pressTimer = setTimeout(() => {
      if (press?.state !== state) return;
      if (state.due(Math.max(performance.now(), state.downAt + LONG_PRESS_MS))) {
        store.show(anchor, 'touch');
        swallowClick();
      }
    }, LONG_PRESS_MS);
  };

  const onMove = (e: PointerEvent): void => {
    if (!press || e.pointerId !== press.pointerId) return;
    press.state.move(e.clientX, e.clientY);
    if (press.state.cancelled) endPress();
  };

  const onUp = (e: PointerEvent): void => {
    if (!press || e.pointerId !== press.pointerId) return;
    const { state, anchor } = press;
    endPress();
    const outcome = state.up(e.clientX, e.clientY, e.timeStamp);
    if (outcome === 'show') {
      store.show(anchor, 'touch');
      swallowClick();
    } else if (outcome === 'swallow') {
      if (!store.active) store.show(anchor, 'touch');
      swallowClick();
    }
  };

  const onCancel = (e: PointerEvent): void => {
    if (!press || e.pointerId !== press.pointerId) return;
    press.state.cancel();
    endPress();
  };

  // The long-press must not open the browser's own menu or callout.
  const onContextMenu = (e: Event): void => {
    if (press || store.active?.via === 'touch') e.preventDefault();
  };

  const onKey = (e: KeyboardEvent): void => {
    if (e.key !== 'Escape' || !store.active) return;
    // Close the tooltip only, not the sheet beneath it.
    e.preventDefault();
    e.stopPropagation();
    store.hide();
  };

  const onFocusIn = (e: FocusEvent): void => {
    const el = e.target;
    if (!(el instanceof Element)) return;
    if (store.contentOf(el) && el.matches(':focus-visible')) {
      clearHover();
      store.show(el, 'focus');
    } else if (store.active?.via === 'focus') store.hide();
  };

  const onFocusOut = (e: FocusEvent): void => {
    if (store.active?.via === 'focus' && store.active.anchor === e.target) store.hide();
  };

  // A resize (a rotation, the keyboard opening) closes it; a scroll moves
  // it with its anchor (the layer follows the anchor).
  const onMoveAway = (): void => {
    if (store.active) store.hide();
  };

  const opts = { capture: true } as const;
  const passive = { capture: true, passive: true } as const;
  window.addEventListener('pointerover', onOver, passive);
  window.addEventListener('pointerout', onOut, passive);
  window.addEventListener('pointerdown', onDown, opts);
  window.addEventListener('pointermove', onMove, passive);
  window.addEventListener('pointerup', onUp, opts);
  window.addEventListener('pointercancel', onCancel, opts);
  window.addEventListener('contextmenu', onContextMenu, opts);
  window.addEventListener('keydown', onKey, opts);
  window.addEventListener('focusin', onFocusIn, opts);
  window.addEventListener('focusout', onFocusOut, opts);
  window.addEventListener('resize', onMoveAway, passive);
  return () => {
    clearHover();
    endPress();
    window.removeEventListener('pointerover', onOver, passive);
    window.removeEventListener('pointerout', onOut, passive);
    window.removeEventListener('pointerdown', onDown, opts);
    window.removeEventListener('pointermove', onMove, passive);
    window.removeEventListener('pointerup', onUp, opts);
    window.removeEventListener('pointercancel', onCancel, opts);
    window.removeEventListener('contextmenu', onContextMenu, opts);
    window.removeEventListener('keydown', onKey, opts);
    window.removeEventListener('focusin', onFocusIn, opts);
    window.removeEventListener('focusout', onFocusOut, opts);
    window.removeEventListener('resize', onMoveAway, passive);
  };
}
