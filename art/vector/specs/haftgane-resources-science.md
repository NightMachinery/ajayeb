# Spec: Haftgane resources and science

Set name in `sets.json`: `haftgane-resources-science`. Nine new files for Haftgane, the second game ([docs/haftgane/README.md](../../../docs/haftgane/README.md)): ore and cloth with their `-sm` variants, compass and tablet with their `-sm` variants, and a 12-grid `science-gear-sm`. Read [../STYLE.md](../STYLE.md) first; it holds the grid, the paint convention and the loop. The subjects and their must-differ-from lists come from [docs/haftgane/art-reuse.md](../../../docs/haftgane/art-reuse.md), "Resources: reuse five, add two" and "Science: two new concepts, one adapted concept".

Acceptance: blind identification 100% for all nine files at 12 px DPR 2 (the `-sm` files) and 16 px DPR 1 (the md files), among the confusable neighbours listed below; distinct in greyscale and in plain ink; no blobbing in the pixel peep; light and dark parity; lint clean.

## Where the files live and what they reuse

- The sources live in `web/src/lib/games/haftgane/icons/svg/`, outside the directory that `web/src/lib/icons/sources.ts` globs, so none of them reaches Duel's bundle. Nothing imports them yet; M19.art-wiring does that, and gives them their names list.
- Haftgane reuses Duel's wood, clay, stone, glass and papyrus, md and `-sm`, unchanged. They are referenced, never copied.
- **science-gear** is an alias: its md drawing is Duel's `chain-gear` (`web/src/lib/icons/svg/chain-gear.svg`), recorded in this set's `sets.json` entry as `file`. Only its `-sm` is new, drawn again on the 12 grid rather than scaled from the md teeth. `science-wheel` is never a substitute: its smooth rim and spokes are a different object.
- No Duel science presentation carries over: Haftgane has no science pairs, progress tokens or science victory, so nothing here is drawn as a pair or a token.

## Shared rules of the set

- Resources must differ by silhouette family, as in Duel. With ore and cloth the seven are: wood stacked and vertical, clay rectilinear in two courses, stone irregular, glass a necked vessel, papyrus wide with two curled ends, ore a single symmetric bar, wider at the foot, with a flat top, cloth a horizontal bolt with a fringed fold hanging below it.
- Never rely on colour. The specimen draws ore on the stone tile and cloth on the papyrus tile on purpose, so that the shape alone must separate the pairs that are closest in idea. M19.art-wiring picks the real tile tokens.
- Science symbols are drawn in the green card colour with `--i2` paper-bright, like Duel's, and are scholars' instruments of a manuscript workshop, not modern pictograms.
- New drawings stay original: never trace the physical game's symbols.

## resource-ore (with resource-ore-sm)

- Meaning: the ore resource (raw material, brown cards produce it).
- Metaphor: a cast metal ingot.
- Silhouette: one low, flat-topped, symmetric mass, wider at the foot than at the top: a front face whose sides slope gently and a top face whose sides slope more steeply, so the outline is a six-sided bar seen from above the front. The broad bevel is the top face, in the detail tone. One mass with no knocked-out edge, never a stack, so it cannot read as two courses of bricks or as a stepped pyramid (log, iteration 0).
- Must differ from: resource-stone (straight edges and symmetry, not a lumpy mass), resource-clay (one sloped block, no courses or seams), category-grey, the anvil (no horn, no waist, no foot), category-brown's mountain (flat top, no peaks).
- Sizes: 9 to 12 px on tiles (`-sm`), 11 to 16 px in strips, 24 px in sheets.
- Contexts: resource tiles in the cost row and the effect band, resource chips, cost breakdowns.

## resource-cloth (with resource-cloth-sm)

- Meaning: the cloth resource (manufactured, grey cards produce it).
- Metaphor: a folded bolt of cloth with one broad fold hanging from it.
- Silhouette: a wide, low bolt across the top with rounded ends, and one broad fold hanging down from its right half, flaring a little, that ends in a notched fringe of three tabs. The bolt's folded layers show as one line in the detail tone (md only). The fold hangs to one side: a centred fold makes a T that reads as a table or a column capital.
- Must differ from: resource-papyrus (no curled ends; the hanging fold and fringe are the key), resource-wood (no upright staves), science-tablet (not a single slab), Haftgane's conflict pennants (a fringe of several tabs, never one swallowtail notch).
- Sizes and contexts: as resource-ore.

## science-compass (with science-compass-sm)

- Meaning: the compass science symbol.
- Metaphor: an open drafting compass, the dividers of a geometer.
- Silhouette: two straight legs joined at a broad round hinge at the top, splayed into an upright V that is open at the bottom and ends in two points. A short handle may rise from the hinge. No magnetic compass rose, no crossbar that would turn it into a letter A.
- Must differ from: chain-lyre (straight legs that meet at the top, not curved arms rising from a sound box), category-red's crossed spears (the legs never cross), science-set-square (open, not a filled triangle).
- Sizes: 12 px (`-sm`) on mini cards and compact trackers, 14 to 17 px (md) on cards, 24 px in sheets.
- Contexts: the science effect glyph on green cards (card-green, `--i2` paper-bright), the science tracker, science choice effects composed with the choice marker.

## science-tablet (with science-tablet-sm)

- Meaning: the tablet science symbol.
- Metaphor: a single broad writing slab of clay or stone.
- Silhouette: an upright slab with rounded shoulders (the two top corners rounded so wide that the top is an arch), a square foot, and one heavy bevel: a broad border, heaviest above, around a blank square face in the detail tone. No writing, no spine, no rolled ends. With small shoulders it reads as a frame or a phone (log, iteration 1).
- Must differ from: chain-book (no spine, no page block, symmetric), Haftgane's chain-scroll (no rolled top and bottom), resource-papyrus (upright and single, no curls), resource-cloth.
- Sizes and contexts: as science-compass.

## science-gear-sm

- Meaning: the gear science symbol at 14 px and below; the md is Duel's `chain-gear`.
- Metaphor: a toothed gear wheel, as chain-gear.
- Silhouette: a solid ring with a round hole and eight short square teeth, drawn on the 12 grid for 12 px: as many teeth as the md, so both sizes read as the same symbol, but a larger ring and shorter teeth, so the gaps stay a unit wide and the teeth never read as the points of a star (log, iteration 0).
- Must differ from: science-wheel-sm (teeth outside, no spokes), science-compass's hinge, coin.
- Sizes and contexts: as science-compass.

## Blind-check neighbours

`sets.json` lists them under `neighbours`, so `blind.mjs make` shuffles them into the sheets as candidates: the five Duel resources, `category-grey` (the anvil), `category-red` (the crossed spears), `chain-book`, `chain-lyre`, `science-set-square` and `science-wheel`. Haftgane's new `chain-scroll` belongs to the chains slice and is drawn in parallel, so it can only be checked against the tablet once both slices are on `master`.
