// Where the toasts sit in phone landscape (M11.board-layout-fixes). There
// the own column is a strip's width, so a long sentence at its foot grew up
// over the strip, and the structure fills the height between the top bar
// and the bottom edge, so no fixed place is clear of the cards: its foot
// holds the cards to take for most of an age, its head near the end. The
// toast list therefore tries lanes in order and takes the first where it
// covers no card that can be taken (an accessible slot, whoever is to move)
// and neither strip nor the turn bar: the structure's head, its foot, the
// free foot of the opponent's column, that of the own column, and last the
// top bar (over the track). When none is clear the list waits off screen,
// where it covers nothing and its live region still speaks; the game log
// has the same line.
//
// A placement is only as good as the geometry it measured, and the
// structure fits its cards to its box asynchronously (CardStructure's own
// ResizeObserver), so after a resize or a rotation the cards move a frame
// or more after the box. Whenever any measured box may change (the window,
// the structure, its fitted cards, a strip, the turn bar), the list leaves
// the screen and comes back only once that geometry has held still for two
// frames, in a lane checked against it. A new toast alone, on a settled
// board, is placed at once.
//
// clear() is the geometry alone; toastLane is the attachment. DOM only, no
// reactive state.

import type { Attachment } from 'svelte/attachments';

export interface Box {
  top: number;
  bottom: number;
  left: number; // physical-ok: viewport coordinates
  right: number; // physical-ok: viewport coordinates
}

/** Whether two boxes share more than a hairline. */
export function overlaps(a: Box, b: Box, tolerance = 0.5): boolean {
  return (
    Math.min(a.right, b.right) - Math.max(a.left, b.left) > tolerance && // physical-ok
    Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top) > tolerance
  );
}

/** Whether a box (the list measured in a lane) covers none of the obstacles; an empty list covers nothing. */
export function clear(box: Box, obstacles: readonly Box[]): boolean {
  return box.bottom - box.top <= 0 || !obstacles.some((o) => overlaps(box, o));
}

/** A place for the list: a box it is centred in, against one of its edges. */
interface Lane {
  name: 'head' | 'foot' | 'away' | 'home' | 'top';
  box: Box & { width: number };
  edge: 'top' | 'bottom';
  /** The widest the list may be, in px. */
  width: number;
}

export interface LaneTargets {
  structure: HTMLElement | null;
  home: HTMLElement | null;
  away: HTMLElement | null;
  turn: HTMLElement | null;
  /** Reruns the placement when it changes (the frame's version). */
  version: number;
}

const GAP = 4;
const MAX_WIDTH = 384;
/** Frames the geometry must hold still before the list comes back. */
const STILL_FRAMES = 2;
/** Frames after which a board that keeps moving is placed anyway (still checked). */
const MAX_FRAMES = 60;

/** Places the list (the attached element) in `lane`. */
function place(el: HTMLElement, lane: Lane): void {
  const width = Math.max(0, Math.min(lane.width, lane.box.width - 2 * GAP));
  el.style.width = `${width}px`;
  el.style.left = `${lane.box.left + (lane.box.width - width) / 2}px`; // physical-ok: viewport pixels
  if (lane.edge === 'top') {
    el.style.top = `${lane.box.top + GAP}px`;
    el.style.bottom = 'auto';
  } else {
    el.style.top = 'auto';
    el.style.bottom = `${window.innerHeight - lane.box.bottom + GAP}px`;
  }
}

function rect(el: Element): Box & { width: number } {
  const r = el.getBoundingClientRect();
  return { top: r.top, bottom: r.bottom, left: r.left, right: r.right, width: r.width }; // physical-ok
}

/**
 * The attachment for the toast list's fixed wrapper in phone landscape. It
 * reads the targets lazily, so it can be attached before they mount; the
 * wrapper's first child is the list.
 */
export function toastLane(targets: () => LaneTargets): Attachment<HTMLElement> {
  return (el) => {
    let frame = 0;
    let settling = false;
    let still = 0;
    let frames = 0;
    let last = '';

    /** The fitted cards: the structure's sized child, which CardStructure resizes. */
    const fitted = (structure: HTMLElement): Element | null =>
      structure.querySelector('.card-structure > div');

    /** The lanes in order and the boxes the list must not cover. */
    const measure = (): { lanes: Lane[]; obstacles: Box[] } | null => {
      const { structure, home, away, turn } = targets();
      if (!structure) return null;
      const s = rect(structure);
      const wide = Math.min(MAX_WIDTH, s.width);
      const lanes: Lane[] = [
        { name: 'head', box: s, edge: 'top', width: wide },
        { name: 'foot', box: s, edge: 'bottom', width: wide }
      ];
      if (away) {
        const box = rect(away);
        lanes.push({ name: 'away', box, edge: 'bottom', width: box.width });
      }
      if (home) {
        const box = rect(home);
        lanes.push({ name: 'home', box, edge: 'bottom', width: box.width });
      }
      const vw = document.documentElement.clientWidth;
      const safeTop = parseFloat(getComputedStyle(el).getPropertyValue('--safe-t')) || 0;
      lanes.push({
        name: 'top',
        box: { top: safeTop, bottom: window.innerHeight, left: 0, right: vw, width: vw }, // physical-ok
        edge: 'top',
        width: Math.min(MAX_WIDTH, vw - 32)
      });
      const obstacles: Box[] = [
        ...[...structure.querySelectorAll('[data-accessible]')].map(rect),
        ...[home, away].flatMap((column) => {
          const strip = column?.querySelector('.player-strip');
          return strip ? [rect(strip)] : [];
        }),
        ...(turn ? [rect(turn)] : [])
      ];
      return { lanes, obstacles };
    };

    /** Off screen: covers nothing, takes no tap, still read out. */
    const offscreen = (): void => {
      el.style.translate = '0 -200vh';
    };

    /** Takes the first clear lane for the geometry measured now, or waits off screen. */
    const layout = (): void => {
      const list = el.firstElementChild as HTMLElement | null;
      const m = measure();
      if (!list || !m) return;
      el.style.translate = '';
      for (const lane of m.lanes) {
        place(el, lane);
        if (clear(rect(list), m.obstacles)) {
          el.dataset.toastLane = lane.name;
          return;
        }
      }
      el.dataset.toastLane = 'none';
      offscreen();
    };

    /** The geometry the placement depends on, rounded, to tell when it holds still. */
    const signature = (): string => {
      const m = measure();
      if (!m) return '';
      const round = (b: Box): string =>
        [b.top, b.bottom, b.left, b.right].map((n) => Math.round(n)).join(','); // physical-ok
      return [...m.lanes.map((l) => round(l.box)), ...m.obstacles.map(round)].join(';');
    };

    const tick = (): void => {
      frame = 0;
      const now = signature();
      still = now === last ? still + 1 : 0;
      last = now;
      if (still >= STILL_FRAMES || ++frames >= MAX_FRAMES) {
        settling = false;
        delete el.dataset.toastSettling;
        layout();
        return;
      }
      frame = requestAnimationFrame(tick);
    };

    /** Geometry may change: leave the screen until it holds still. */
    const settle = (): void => {
      if (!settling) {
        settling = true;
        el.dataset.toastSettling = '';
        offscreen();
      }
      still = 0;
      frames = 0;
      last = '';
      if (!frame) frame = requestAnimationFrame(tick);
    };

    /** The list changed size (a toast came or went): place it now unless the board is moving. */
    const relist = (): void => {
      if (!settling) layout();
    };

    // Its own moves must take effect at once to be measured: with Animation
    // off every element transitions for 0.01 ms (app.css), so a box read
    // right after a change would still be the old, off-screen one.
    el.style.transitionProperty = 'none';
    const { structure, home, away: awayEl, turn } = targets(); // also reads the version
    const board = new ResizeObserver(settle);
    const watched = new Set<Element>();
    const watch = (target: Element | null | undefined): void => {
      if (target && !watched.has(target)) {
        watched.add(target);
        board.observe(target);
      }
    };
    const watchAll = (): void => {
      watch(structure);
      const cards = structure?.querySelector('.card-structure');
      watch(cards);
      if (structure) watch(fitted(structure));
      watch(home?.querySelector('.player-strip'));
      watch(awayEl?.querySelector('.player-strip'));
      watch(turn);
    };
    watchAll();
    // The fitted child mounts once the box has a size, and its card size
    // is mirrored on the structure's data attributes.
    const fit = new MutationObserver(() => {
      watchAll();
      settle();
    });
    const cards = structure?.querySelector('.card-structure');
    if (cards) {
      fit.observe(cards, {
        attributes: true,
        attributeFilter: ['data-card-w', 'data-card-h'],
        childList: true
      });
    }
    const listObserver = new ResizeObserver(relist);
    if (el.firstElementChild) listObserver.observe(el.firstElementChild);
    window.addEventListener('resize', settle);
    settle();
    return () => {
      if (frame) cancelAnimationFrame(frame);
      board.disconnect();
      fit.disconnect();
      listObserver.disconnect();
      window.removeEventListener('resize', settle);
    };
  };
}
