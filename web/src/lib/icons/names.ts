// The typed list of icon names, one per source file in ./svg/*.svg. Keep this
// in step with that directory; sprite.ts checks at build time that every name
// here has a source file, and that every source file is named here.
//
// Placeholder set for M5 (see docs/dev/roadmap.md and AGENTS.md "Assets"):
// simple, clean, geometric SVGs that stand in for the hand-authored vector
// set the M9 milestone builds through its own Opus render-critique loop.
// Never copy or trace official iconography, here or later.

export const ICON_NAMES = [
  // Resources (5), each with a -sm 12-grid variant for 14px and below.
  'resource-wood',
  'resource-wood-sm',
  'resource-clay',
  'resource-clay-sm',
  'resource-stone',
  'resource-stone-sm',
  'resource-glass',
  'resource-glass-sm',
  'resource-papyrus',
  'resource-papyrus-sm',

  // Economy and markers.
  'coin',
  'coin-loss',
  'trade',
  'price-tag',
  'choice',

  // Military.
  'shield',
  'military-pawn',
  'looting-2',
  'looting-5',

  // Science symbols (7), each with a -sm variant (pairs are the core mechanic).
  'science-globe',
  'science-globe-sm',
  'science-scales',
  'science-scales-sm',
  'science-sundial',
  'science-sundial-sm',
  'science-mortar',
  'science-mortar-sm',
  'science-set-square',
  'science-set-square-sm',
  'science-quill',
  'science-quill-sm',
  'science-wheel',
  'science-wheel-sm',

  // Chain symbols (17): the in/out markers that make a later card free.
  'chain-horseshoe',
  'chain-sword',
  'chain-tower',
  'chain-book',
  'chain-gear',
  'chain-amphora',
  'chain-mask',
  'chain-moon',
  'chain-drop',
  'chain-target',
  'chain-helmet',
  'chain-lyre',
  'chain-lamp',
  'chain-barrel',
  'chain-sun',
  'chain-column',
  'chain-temple-facade',

  // Scoring and players.
  'vp-laurel',
  'wonder',
  'token-frame',
  'crest-sun',
  'crest-moon',

  // The two capitals at the ends of the military track.
  'capital-sun',
  'capital-moon',

  // Card-category glyphs (7), one per Color in the catalog, each with a -sm
  // variant for the colour swatches of "per card of colour X" effects.
  'category-brown',
  'category-brown-sm',
  'category-grey',
  'category-grey-sm',
  'category-blue',
  'category-blue-sm',
  'category-green',
  'category-green-sm',
  'category-yellow',
  'category-yellow-sm',
  'category-red',
  'category-red-sm',
  'category-purple',
  'category-purple-sm',

  // Affordability badge glyphs. "trade" above doubles as the trade badge.
  'badge-check',
  'badge-chain',
  'badge-lock'
] as const;

export type IconName = (typeof ICON_NAMES)[number];
