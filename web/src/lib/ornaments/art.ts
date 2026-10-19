// The ornament files drawn inline (art/vector/specs/ornaments-track-logo.md):
// the military track's cells and the wordmark. Each is our own SVG under
// src/lib/ornaments or src/lib/brand, imported as text at build time, so its
// markup never carries user or server data and can be placed with {@html}.
// The masks (the girih) are CSS images instead: see TableGirih.svelte.

import wordmarkRaw from '$lib/brand/logo-wordmark.svg?raw';
import cell0 from './svg/track-cell.svg?raw';
import cell2 from './svg/track-cell-2.svg?raw';
import cell5 from './svg/track-cell-5.svg?raw';
import cell10 from './svg/track-cell-10.svg?raw';

/** An SVG file split into its root viewBox and the markup inside the root. */
export interface InlineArt {
  viewBox: string;
  inner: string;
}

/** The viewBox and the markup between the root's opening and closing tags. */
export function inlineArt(raw: string): InlineArt {
  const open = /<svg\b[^>]*>/.exec(raw);
  const closeAt = raw.lastIndexOf('</svg>');
  const viewBox = /viewBox="([^"]+)"/.exec(open?.[0] ?? '')?.[1];
  if (!open || closeAt < 0 || !viewBox) throw new Error('ornaments: not a standalone <svg>');
  return { viewBox, inner: raw.slice(open.index + open[0].length, closeAt) };
}

/** The Persian and Latin wordmark (logo-wordmark.svg). */
export const WORDMARK = inlineArt(wordmarkRaw);

const CELLS = {
  0: inlineArt(cell0),
  2: inlineArt(cell2),
  5: inlineArt(cell5),
  10: inlineArt(cell10)
} as const;

/**
 * The cell drawn on a space of the track: plain in the centre, then one,
 * two or three lozenges in the zones of 2, 5 and 10 points (`points` is the
 * zone's value, 0 for the centre).
 */
export function trackCellArt(points: number): InlineArt {
  return points === 2 || points === 5 || points === 10 ? CELLS[points] : CELLS[0];
}
