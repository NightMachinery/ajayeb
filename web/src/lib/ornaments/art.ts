// The ornament files the app draws (art/vector/specs/ornaments-track-logo.md).
// The wordmark is inline: our own SVG under src/lib/brand, imported as text
// at build time, so its markup never carries user or server data and can be
// placed with {@html}. The track cells are CSS masks, one element per run of
// equal spaces, so the board stays inside its DOM budget; like the girih
// (TableGirih.svelte), their URLs come from Vite, which inlines a file this
// small or names it by its content hash.

import wordmarkRaw from '$lib/brand/logo-wordmark.svg?raw';
import cell0 from './svg/track-cell.svg?url';
import cell2 from './svg/track-cell-2.svg?url';
import cell5 from './svg/track-cell-5.svg?url';
import cell10 from './svg/track-cell-10.svg?url';

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

const CELLS = { 0: cell0, 2: cell2, 5: cell5, 10: cell10 } as const;

/**
 * The URL of the cell drawn on a space of the track: plain in the centre,
 * then one, two or three lozenges in the zones of 2, 5 and 10 points
 * (`points` is the zone's value, 0 for the centre). Its field paints
 * `var(--i2, transparent)`, which is transparent in an image, so as a mask
 * it shows only the frame and the lozenges.
 */
export function trackCellUrl(points: number): string {
  return points === 2 || points === 5 || points === 10 ? CELLS[points] : CELLS[0];
}
