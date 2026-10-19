# tt2-glyphs: iteration log

The effect glyphs that replace the Lucide outlines in EffectGlyph.svelte, and `crest-moon-sm` (listed in `sets.json` as the `sm` of `crest-moon` in the ui-category set). The spec is [../specs/tt2-glyphs-diagrams.md](../specs/tt2-glyphs-diagrams.md).

## Status

- play-again: done (iteration 3)
- build-from-discard: done (iteration 4); still the weakest of the four at 12 px, see below
- destroy-card: done (iteration 4)
- crest-moon-sm: done (iteration 3); redrawn in M11.ornament-integration as a plain tilted crescent (round m11-3)
- Blind check, browser check and family check: run on 2026-10-16, see the last three sections.

## Iteration 1 (2026-10-15)

Renders: art-work/vector/tt2-glyphs/iter-1/ (fast.mjs at 12, 16, 24 and 48 px on paper and night, montaged at 400% with point filtering).
Lint: clean but for one live-area warning on crest-moon-sm (the top horn reached 0.94).

### play-again

- Seen: reads as a looping arrow from 16 px, but a light seam runs down the base of the arrowhead: the head and the ring were subpaths of one path with opposite winding, so their overlap was a hole. At 12 px the head is a small bump on a C.
- Changed: the head is its own path, its base flush with the ring's end, and it is larger and pushed further into the gap.

### build-from-discard

- Seen: the upright card on two slabs, the narrower slab on top, reads as a gravestone on a plinth or an upload icon.
- Changed: the card tilts 10 degrees, as if being picked up, and the two slabs are the same width and staggered, so they read as a pile of cards rather than steps.

### destroy-card

- Seen: a card broken in two at every size; at 12 px the crack is a clear dark zigzag. Distinct from the whole swatch card.
- Changed: nothing. Done.

### crest-moon-sm

- Seen: a fat crescent with a nose bump; the eye shows from 16 px and is a speck at 12. Clearly apart from crest-sun-sm in shape.
- Changed: only the live-area warning to fix.

## Iteration 2 (2026-10-15)

Renders: art-work/vector/tt2-glyphs/iter-2/.
Lint: two live-area warnings (play-again's head at 1.5; crest-moon-sm at 11.27).

- play-again: seam gone; reads as a looping arrow at 12 px. Changed: head trimmed to y 2 to 9.
- build-from-discard: the tilted card and staggered pile read as "a card lifted from the pile" from 24 px; at 12 to 16 px it is a small card over two bars, legible as a shape but not self-explaining. The card sheet and the tooltip carry the words. Kept.
- crest-moon-sm: moving the top horn made the moon rounder and blunter. Changed: the crescent is redrawn on the circle of radius 5 about (6, 6), with both horn tips on it at (6, 1) and (10, 9).

## Iteration 3 (2026-10-15)

Renders: art-work/vector/tt2-glyphs/iter-3/.
Lint: clean, 0 warnings; the sprite is 27206 bytes raw and 5615 gzipped (budgets 40960 and 10240).

- play-again: clean at 12 to 48 px on paper and night. Done.
- crest-moon-sm: a fat crescent face at 12 to 48 px, the profile readable from 16 px. Done.

## Blind check (2026-10-16)

One fresh subagent saw only the four icons, unnamed, at 48 px and at their true 16 px raster enlarged, on paper, and was told only what kind of game they come from.

- play-again: "a refresh or rotate arrow"; the most recognisable of the four at 16 px. Good: in context it sits after a wonder or alone, and "again" is what a loop arrow says.
- crest-moon-sm: "a crescent moon, or a coin with a bite"; at 16 px the bite rounds off. Acceptable: it only ever appears at the track's end beside the sun, where the pair reads as the two capitals.
- destroy-card: "two tilted bars, like matchsticks". The two halves splayed apart symmetrically, so the card was lost.
- build-from-discard: "a pin or banner on a stand"; the arrow inside the card was lost at 16 px and the two slabs merged.

## Iteration 4 (2026-10-16)

Renders: art-work/vector/tt2-glyphs/iter-4/ (render-fast.sh at 12, 16 and 48 px on paper and night; the current file and two variants of each icon side by side).
Lint: clean, 0 warnings; the sprite is 27273 bytes raw and 5638 gzipped.

- destroy-card: the left half now stands upright with the card's rounded corners and only the right half falls away (4 degrees), so the outline reads as one card with a crack through it rather than two bars. A variant with both halves at 3 degrees still read as a pair. Done.
- build-from-discard: the card is larger (11 by 13), tilted 8 degrees instead of 10, with a wider arrow, and the slabs are 2 units thick. The arrow now survives at 16 px as a light notch with a stem. An upright card over a tray was tried and dropped: it reads as the generic upload icon. Done; at 12 px it is still a small card over a pile and leans on the tooltip and card sheet words.

## Browser check (2026-10-16)

Sheet: art-work/vector/tt2-glyphs/browser/sheet@2x.png (tools/chrome/svg-render.mjs, the four icons and crest-sun-sm for comparison, at 12, 16, 24 and 48 px on paper #f4ecdc, bright paper #fbf6ea, night #1c1915 and the dark theme's navy #14213d, at DPR 2).

- All four hold their shape at every size on all four grounds. crest-moon-sm and crest-sun-sm have the same mass and sit as a pair at 12 px.
- play-again and destroy-card read from 12 px; build-from-discard from 16 px.

## Family check (2026-10-16)

Sheet: art-work/vector/tt2-glyphs/family/family@1x.png and @2x (tools/art/vector/family.mjs, every set at 16 and 24 px on paper and night).

- The tt2 glyphs are solid fills like the chain and ui-category sets, with the same stroke weight in the loop of play-again as in the chain set's horseshoe. No outline icon is left among the effect glyphs.

## M11.ornament-integration: crest-moon-sm redrawn (2026-10-19)

Finding S1-11 of the M11 screens review: on the compact track (phone landscape) the two capital caps were told apart only by colour. In greyscale and under deuteranopia the 11 px crests were two dark blobs of the same mass, because the crescent face's nose and eye filled its bite. Renders: art-work/vector/ornament-integration/iter-1/ to iter-3/ (render-fast.sh at 11 px on paper, on the moon tint and on navy, an 800% point-filtered peep, a greyscale row, and 96 px), each row beside crest-sun-sm and the old file.

### Round m11-1

- Seen: the old file at 11 px is a bitten disc with a speck; beside the sun's stub-rayed disc only the colour tells them apart.
- Tried: a fat crescent opening up and right (inner arc 4 or 4.5, back about 6.8 units), the same with an eye knocked out, and a thinner one. The fat ones still read as a coin with a bite at 11 px; the eye is noise. A plain C-shaped crescent (outer half circle of radius 5, inner arc 7.5, back 3 units) was clearly a moon in every row.

### Round m11-2

- Tried: the C crescent upright and tilted by 20 and 30 degrees, with inner arcs of 8, 10 and 14. Tilted reads more like a moon and less like a bracket; the inner arc of 14 gives a back of about 4 units, so its mass at 11 px sits close to the sun's.
- Seen: the tilted top horn crossed the live area at y 0.45.

### Round m11-3

- Changed: the crescent shifted by (-0.5, 0.5) before the rotation, so it sits inside the live area (lint clean, 0 warnings) and centred on the disc of the sun.
- Seen: at 11 px a crescent of 3 to 4 pixels' thickness with two clear horns, on paper, on the moon tint and on navy; in greyscale the sun is a round bumpy disc and the moon an open crescent. Done.
- In the app (shots/ornament-integration/after/, trackgrey-* and trackdeut-*): on the compact track the two ends now differ by shape in greyscale and under deuteranopia. The full bar now draws the city gates (capital-sun, capital-moon) at 18 px instead of the crests.
