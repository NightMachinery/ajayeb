// fitStructure: how big the cards of an age's structure can be in a box.
//
// The structure is a pyramid of overlapping rows. Each card is r times as
// tall as it is wide (the card ratio). A row sits `step` card heights below
// the one above it and covers the rest of that row, so a covered card shows
// its top `step` of height. Columns are in half-card widths (the server's
// SlotView.col), so a full row of `cols` cards is cols·w + (cols − 1)·gap
// wide, and the whole structure is h + (rows − 1)·step·h tall.
//
// The rules (from the UX design, "Card-structure geometry"):
//   - The card is as wide as the box allows, never wider than wMax.
//   - If the height binds, the step drops toward stepMin before the card
//     shrinks; at stepMin the card shrinks until the structure fits.
//   - If the width binds, the step rises up to stepMax (as far as the height
//     allows), so more of each covered card shows.
//   - If wMax binds, the step is the preferred one (or less, if the height
//     cannot take it).
//   - The tier (which CardFace layout to draw) follows from the card width.
//
// Pure and synchronous: CardStructure calls it from a ResizeObserver, and
// the unit tests check it against the design's table of results.

/** How many cards wide and how many rows tall a structure is. */
export interface StructureShape {
  /** The most cards side by side in any row (a full row's card count). */
  cols: number;
  /** The number of rows. */
  rows: number;
}

export interface FitOptions {
  /** Space between two cards of one row, in px. */
  gap: number;
  /** Card height divided by card width. */
  ratio: number;
  stepMin: number;
  stepMax: number;
  stepPref: number;
  /** The widest a card may be, in px. */
  wMax: number;
}

/** The CardFace render tiers. `band` is only for city stacks, never chosen by width. */
export type CardTier = 'band' | 'mini' | 'medium' | 'full';

export interface Fit {
  /** Card width and height in whole px. */
  w: number;
  h: number;
  /** Row offset as a fraction of the card height. */
  step: number;
  gap: number;
  tier: Exclude<CardTier, 'band'>;
  /** What bound the card size: the box width, the box height, or wMax. */
  bound: 'width' | 'height' | 'cap';
  /** The size of the whole structure, in px (at most the box). */
  width: number;
  height: number;
}

/** The card ratio of the physical cards (about 44 × 68 mm). */
export const CARD_RATIO = 1.545;

export const DEFAULT_FIT: FitOptions = {
  gap: 4,
  ratio: CARD_RATIO,
  stepMin: 0.42,
  stepMax: 0.62,
  stepPref: 0.5,
  wMax: 132
};

/** Card widths under this draw the mini tier. */
export const MINI_BELOW = 72;
/** Card widths under this (and at least MINI_BELOW) draw the medium tier. */
export const MEDIUM_BELOW = 140;

/** The tier for a card of this width in px. */
export function tierFor(width: number): Exclude<CardTier, 'band'> {
  if (width < MINI_BELOW) return 'mini';
  if (width < MEDIUM_BELOW) return 'medium';
  return 'full';
}

/** Rounds down to two decimals, so a rounded step never overflows the box. */
function floor2(x: number): number {
  return Math.floor(x * 100 + 1e-9) / 100;
}

/**
 * The largest card size (and the row step) at which a structure of `shape`
 * fits a W × H box. Sizes are whole px, rounded down, so the result always
 * fits. A box with no room returns a zero-size fit.
 */
export function fitStructure(
  W: number,
  H: number,
  shape: StructureShape,
  options: Partial<FitOptions> = {}
): Fit {
  const o = { ...DEFAULT_FIT, ...options };
  const cols = Math.max(1, shape.cols);
  const rows = Math.max(1, shape.rows);
  const spans = rows - 1;

  const wByWidth = (W - (cols - 1) * o.gap) / cols;
  // The widest card whose structure still fits the height at the smallest step.
  const wByHeight = H / (o.ratio * (1 + spans * o.stepMin));
  const exact = Math.min(wByWidth, wByHeight, o.wMax);
  if (!(exact > 0)) {
    return {
      w: 0,
      h: 0,
      step: o.stepMin,
      gap: o.gap,
      tier: 'mini',
      bound: 'width',
      width: 0,
      height: 0
    };
  }

  const bound: Fit['bound'] = exact === wByHeight ? 'height' : exact === wByWidth ? 'width' : 'cap';
  const w = Math.floor(exact);
  const h = Math.floor(w * o.ratio);

  // The largest step the height allows for this card.
  const stepRoom = spans > 0 ? (H / h - 1) / spans : o.stepMax;
  let step: number;
  if (bound === 'height') step = o.stepMin;
  else if (bound === 'width') step = Math.min(o.stepMax, stepRoom);
  else step = Math.min(o.stepPref, stepRoom);
  step = Math.max(o.stepMin, floor2(step));

  return {
    w,
    h,
    step,
    gap: o.gap,
    tier: tierFor(w),
    bound,
    width: cols * w + (cols - 1) * o.gap,
    height: h + spans * step * h
  };
}
