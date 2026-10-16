// Colour, pattern and glyph per card colour (UX design, "Colour, pattern and
// glyph"). Every colour also has its own pattern and category glyph, so a
// colour-blind player (or a greyscale screen) can still tell the categories
// apart. Patterns are CSS gradients, drawn over the colour on the card's band
// and in the placeholder art; they cost nothing to ship.

import { stripBidiMarks } from '$lib/i18n';
import type { IconName } from '$lib/icons/names';
import type { Back, Color, Resource } from '$lib/wire';

/** The CSS colour of a card colour's band. */
export function bandColor(color: Color): string {
  return `var(--card-${color})`;
}

/** The category glyph of a card colour (an icon in the sprite). */
export function categoryIcon(color: Color): IconName {
  return `category-${color}`;
}

/**
 * The pattern layers of a card colour, as a CSS background list that uses
 * --pat for its ink (set it to a translucent colour on the element). A
 * colour's layers go before its base colour in a `background` shorthand.
 */
export function patternLayers(color: Color): string {
  switch (color) {
    case 'brown': // diagonal hatch
      return 'repeating-linear-gradient(45deg, var(--pat) 0 1.5px, transparent 1.5px 6px)';
    case 'grey': // dots
      return 'radial-gradient(circle, var(--pat) 0 1.3px, transparent 1.6px) 0 0 / 6px 6px';
    case 'blue': // horizontal rules
      return 'repeating-linear-gradient(0deg, var(--pat) 0 1.5px, transparent 1.5px 5px)';
    case 'green': // diamond lattice
      return (
        'repeating-linear-gradient(45deg, var(--pat) 0 1px, transparent 1px 7px), ' +
        'repeating-linear-gradient(-45deg, var(--pat) 0 1px, transparent 1px 7px)'
      );
    case 'yellow': // plain, with a gold double rule at both edges
      return (
        'linear-gradient(var(--pat), var(--pat)) 0 2px / 100% 1px no-repeat, ' +
        'linear-gradient(var(--pat), var(--pat)) 0 4px / 100% 1px no-repeat, ' +
        'linear-gradient(var(--pat), var(--pat)) 0 calc(100% - 2px) / 100% 1px no-repeat, ' +
        'linear-gradient(var(--pat), var(--pat)) 0 calc(100% - 4px) / 100% 1px no-repeat'
      );
    case 'red': // chevrons
      return (
        'linear-gradient(135deg, var(--pat) 25%, transparent 25%) -4px 0 / 8px 8px, ' +
        'linear-gradient(225deg, var(--pat) 25%, transparent 25%) -4px 0 / 8px 8px'
      );
    case 'purple': // checker
      return 'repeating-conic-gradient(var(--pat) 0 25%, transparent 0 50%) 0 0 / 8px 8px';
  }
}

/** A `background` value: the colour's pattern over its band colour. */
export function bandBackground(color: Color): string {
  return `${patternLayers(color)}, ${bandColor(color)}`;
}

/** Ink for text and glyphs drawn straight on a band of this colour. */
export function bandInk(color: Color): string {
  return color === 'yellow' || color === 'grey' ? 'var(--card-ink)' : 'var(--paper-bright)';
}

/** The resource tile colours: the resource colour, and a glyph colour that reads on it. */
export function resourceTile(resource: Resource): { bg: string; ink: string } {
  const light = resource === 'glass' || resource === 'papyrus';
  return {
    bg: `var(--res-${resource})`,
    ink: light ? 'var(--card-ink)' : 'var(--paper-bright)'
  };
}

/** The icon of a resource, in the small-grid variant for tiny sizes. */
export function resourceIcon(resource: Resource, small: boolean): IconName {
  return small ? `resource-${resource}-sm` : `resource-${resource}`;
}

/** The age numeral key of a back (guild backs belong to Age III). */
export function backAge(back: Back): 1 | 2 | 3 {
  if (back === 'age1') return 1;
  if (back === 'age2') return 2;
  return 3;
}

/** The length in characters of a name's longest word, which CardFace uses to
 *  shrink a name that would otherwise have to break inside a word. At least 1.
 *  The invisible direction marks of a right-to-left locale do not count. */
export function longestWord(name: string): number {
  let longest = 1;
  for (const word of stripBidiMarks(name).split(/\s+/)) {
    longest = Math.max(longest, [...word].length);
  }
  return longest;
}
