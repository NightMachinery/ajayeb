// Where the toasts sit in phone landscape (M11.board-layout-fixes). There
// the own column is a strip's width, so a long sentence at its foot grew up
// over the strip, and the structure fills the height between the top bar
// and the bottom edge, so no fixed place is clear of the cards: its foot
// holds the cards to take for most of an age, its head near the end. The
// toast list therefore tries lanes in order and takes the first where it
// covers no card that can be taken (an accessible slot, whoever is to move)
// and neither strip nor the turn bar: the structure's head, its foot, the
// free foot of the opponent's column, that of the own column, and last the
// top bar above the structure (over the track). When none is clear the list
// waits off screen, where it covers nothing and its live region still
// speaks; the game log has the same line.
//
// A placement is only as good as the geometry it measured, and the
// structure fits its cards to its box asynchronously (CardStructure's own
// ResizeObserver), so after a resize or a rotation the cards move a frame
// or more after the box. Whenever any measured box may change (the window,
// the structure, its fitted cards, a strip, the turn bar, which cards can
// be taken), the list leaves the screen and comes back only once that
// geometry has held still for two frames, in a lane checked against it. A
// new toast alone, on a settled board, is placed at once.
//
// The move summary takes the same lanes in phone landscape (summaryLane,
// M11.round3-board-fixes). It used to sit over the structure's head
// whatever lay there, so it covered a card the viewer's own move had just
// uncovered. Unlike a toast it must stay on screen (the player asked to see
// it), so it takes the first lane where it fits whole, else the lane with
// the most clear height (at least MIN_PANEL; it scrolls inside), else the
// structure's head as before, marked data-summary-lane="none". It stays
// where it is while the board moves and is placed again once it holds
// still.
//
// clear() and fitPanel() are the geometry alone; toastLane and summaryLane
// are the attachments. DOM only, no reactive state.

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

export type LaneName = 'head' | 'foot' | 'away' | 'home' | 'top';

/** A place for the list: a box it is centred in, against one of its edges. */
export interface Lane {
  name: LaneName;
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
/** The narrowest lane the move summary takes, in px. */
const MIN_PANEL_WIDTH = 224;
/** The least clear height the move summary takes, scrolling inside, in px. */
export const MIN_PANEL = 112;
/** Frames the geometry must hold still before the list comes back. */
const STILL_FRAMES = 2;
/** Frames after which a board that keeps moving is placed anyway (still checked). */
const MAX_FRAMES = 60;

/** The width a list or panel takes in `lane`. */
function laneWidth(lane: Lane): number {
  return Math.max(0, Math.min(lane.width, lane.box.width - 2 * GAP));
}

/** Places the list (the attached element) in `lane`. */
function place(el: HTMLElement, lane: Lane): void {
  const width = laneWidth(lane);
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

/**
 * The clear height in `lane` for a box spanning `left` to `right`, from the
 * lane's edge to the nearest obstacle in that span (or the lane's far
 * edge). Zero or less when an obstacle reaches the edge itself.
 */
export function clearSpan(
  lane: Lane,
  left: number,
  right: number,
  obstacles: readonly Box[]
): { top: number; bottom: number } {
  const across = obstacles.filter(
    (o) => Math.min(right, o.right) - Math.max(left, o.left) > 0.5 // physical-ok
  );
  let top = lane.box.top + GAP;
  let bottom = lane.box.bottom - GAP;
  if (lane.edge === 'top') {
    for (const o of across) if (o.bottom > top + 0.5) bottom = Math.min(bottom, o.top - GAP);
  } else {
    for (const o of across) if (o.top < bottom - 0.5) top = Math.max(top, o.bottom + GAP);
  }
  return { top, bottom };
}

export interface PanelFit {
  lane: LaneName | 'none';
  top: number;
  left: number;
  width: number;
  /** The panel's height there: its natural height, or less when it scrolls. */
  height: number;
}

/**
 * Where the move summary goes among `lanes` (in order): the first lane where
 * it fits whole, else the lane with the most clear height of at least
 * `min`, else null (the caller keeps its old place). `natural` is the
 * panel's height at a width.
 */
export function fitPanel(
  lanes: readonly Lane[],
  obstacles: readonly Box[],
  natural: (width: number) => number,
  min = MIN_PANEL
): PanelFit | null {
  let best: PanelFit | null = null;
  for (const lane of lanes) {
    const width = laneWidth(lane);
    if (width < MIN_PANEL_WIDTH) continue;
    const left = lane.box.left + (lane.box.width - width) / 2;
    const span = clearSpan(lane, left, left + width, obstacles);
    const room = span.bottom - span.top;
    const want = natural(width);
    const height = Math.min(want, room);
    const top = lane.edge === 'top' ? span.top : span.bottom - height;
    if (room >= want) return { lane: lane.name, top, left, width, height };
    if (room >= min && (!best || height > best.height)) {
      best = { lane: lane.name, top, left, width, height };
    }
  }
  return best;
}

function rect(el: Element): Box & { width: number } {
  const r = el.getBoundingClientRect();
  return { top: r.top, bottom: r.bottom, left: r.left, right: r.right, width: r.width }; // physical-ok
}

/** The fitted cards: the structure's sized child, which CardStructure resizes. */
function fitted(structure: HTMLElement): Element | null {
  return structure.querySelector('.card-structure > div');
}

/** The lanes in order and the boxes nothing may cover; `el` carries --safe-t. */
function measure(t: LaneTargets, el: HTMLElement): { lanes: Lane[]; obstacles: Box[] } | null {
  const { structure, home, away, turn } = t;
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
  // Over the top bar, above the structure only: centred on the whole width,
  // a 24rem list or summary reached a strip at 740x360.
  const safeTop = parseFloat(getComputedStyle(el).getPropertyValue('--safe-t')) || 0;
  lanes.push({
    name: 'top',
    box: { top: safeTop, bottom: window.innerHeight, left: s.left, right: s.right, width: s.width }, // physical-ok
    edge: 'top',
    width: wide
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
}

/**
 * Watches the board's geometry and calls `move` whenever it may change,
 * then `still` once it has held still for STILL_FRAMES frames (or after
 * MAX_FRAMES). Returns `settle` (to start a wait by hand) and the cleanup.
 */
function watchBoard(
  t: LaneTargets,
  el: HTMLElement,
  move: () => void,
  still: () => void
): { settle: () => void; stop: () => void } {
  let frame = 0;
  let settling = false;
  let held = 0;
  let frames = 0;
  let last = '';

  /** The geometry the placement depends on, rounded, to tell when it holds still. */
  const signature = (): string => {
    const m = measure(t, el);
    if (!m) return '';
    const round = (b: Box): string =>
      [b.top, b.bottom, b.left, b.right].map((n) => Math.round(n)).join(','); // physical-ok
    return [...m.lanes.map((l) => round(l.box)), ...m.obstacles.map(round)].join(';');
  };

  const tick = (): void => {
    frame = 0;
    const now = signature();
    held = now === last ? held + 1 : 0;
    last = now;
    if (held >= STILL_FRAMES || ++frames >= MAX_FRAMES) {
      settling = false;
      still();
      return;
    }
    frame = requestAnimationFrame(tick);
  };

  const settle = (): void => {
    if (!settling) {
      settling = true;
      move();
    }
    held = 0;
    frames = 0;
    last = '';
    if (!frame) frame = requestAnimationFrame(tick);
  };

  const { structure, home, away, turn } = t;
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
    watch(structure?.querySelector('.card-structure'));
    if (structure) watch(fitted(structure));
    watch(home?.querySelector('.player-strip'));
    watch(away?.querySelector('.player-strip'));
    watch(turn);
  };
  watchAll();
  // The fitted child mounts once the box has a size, its card size is
  // mirrored on the structure's data attributes, and a move uncovers cards
  // (data-accessible) after the frame that carried it.
  const fit = new MutationObserver(() => {
    watchAll();
    settle();
  });
  const cards = structure?.querySelector('.card-structure');
  if (cards) {
    fit.observe(cards, {
      attributes: true,
      attributeFilter: ['data-card-w', 'data-card-h', 'data-accessible'],
      childList: true,
      subtree: true
    });
  }
  window.addEventListener('resize', settle);
  return {
    settle,
    stop: () => {
      if (frame) cancelAnimationFrame(frame);
      board.disconnect();
      fit.disconnect();
      window.removeEventListener('resize', settle);
    }
  };
}

/**
 * The attachment for the toast list's fixed wrapper in phone landscape. It
 * reads the targets lazily, so it can be attached before they mount; the
 * wrapper's first child is the list.
 */
export function toastLane(targets: () => LaneTargets): Attachment<HTMLElement> {
  return (el) => {
    let settling = false;

    /** Off screen: covers nothing, takes no tap, still read out. */
    const offscreen = (): void => {
      el.style.translate = '0 -200vh';
    };

    /** Takes the first clear lane for the geometry measured now, or waits off screen. */
    const layout = (): void => {
      const list = el.firstElementChild as HTMLElement | null;
      const m = measure(t, el);
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

    // Its own moves must take effect at once to be measured: with Animation
    // off every element transitions for 0.01 ms (app.css), so a box read
    // right after a change would still be the old, off-screen one.
    el.style.transitionProperty = 'none';
    const t = targets(); // also reads the version
    const watcher = watchBoard(
      t,
      el,
      () => {
        settling = true;
        el.dataset.toastSettling = '';
        offscreen();
      },
      () => {
        settling = false;
        delete el.dataset.toastSettling;
        layout();
      }
    );
    /** The list changed size (a toast came or went): place it now unless the board is moving. */
    const listObserver = new ResizeObserver(() => {
      if (!settling) layout();
    });
    if (el.firstElementChild) listObserver.observe(el.firstElementChild);
    watcher.settle();
    return () => {
      watcher.stop();
      listObserver.disconnect();
    };
  };
}

/**
 * Places the move summary's panel (the DismissLayer around the attached
 * content) in the first lane where it covers no card to take, neither strip
 * nor the turn bar (fitPanel), in phone landscape. Without a clear lane it
 * sits over the structure's head, data-summary-lane="none". DOM only.
 */
export function summaryLane(targets: () => LaneTargets): Attachment<HTMLElement> {
  return (content) => {
    const panel = content.closest<HTMLElement>('[data-dismiss-layer]');
    if (!panel) return;
    const t = targets(); // also reads the version

    /** The panel's height at `width`, unclipped. */
    const natural = (width: number): number => {
      panel.style.width = `${width}px`;
      panel.style.maxHeight = 'none';
      return panel.offsetHeight;
    };

    const layout = (): void => {
      const m = measure(t, panel);
      if (!m) return;
      let fit = fitPanel(m.lanes, m.obstacles, natural);
      if (!fit) {
        // Nothing is clear: the structure's head, as on a portrait phone.
        const head = m.lanes[0]!;
        const width = laneWidth(head);
        fit = {
          lane: 'none',
          top: head.box.top + GAP,
          left: head.box.left + (head.box.width - width) / 2,
          width,
          height: Math.max(MIN_PANEL, window.innerHeight - head.box.top - 2 * GAP)
        };
      }
      panel.style.inset = 'auto';
      panel.style.margin = '0';
      panel.style.top = `${fit.top}px`;
      panel.style.left = `${fit.left}px`; // physical-ok: viewport pixels from the lane's box
      panel.style.width = `${fit.width}px`;
      panel.style.maxHeight = `${fit.height}px`;
      panel.dataset.summaryLane = fit.lane;
    };

    panel.style.transitionProperty = 'none';
    layout();
    const watcher = watchBoard(t, panel, () => {}, layout);
    // A modal dialog opened in the same update (a starter choice): show the
    // panel again so it lands above it.
    const frame = requestAnimationFrame(() => {
      if (document.querySelector('dialog:modal') && panel.matches(':popover-open')) {
        panel.hidePopover();
        panel.showPopover();
      }
      layout();
    });
    const resize = new ResizeObserver(() => watcher.settle());
    resize.observe(content);
    return () => {
      cancelAnimationFrame(frame);
      resize.disconnect();
      watcher.stop();
      delete panel.dataset.summaryLane;
    };
  };
}
