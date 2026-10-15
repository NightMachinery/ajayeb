# tt2-glyphs: iteration log

The effect glyphs that replace the Lucide outlines in EffectGlyph.svelte, and `crest-moon-sm` (listed in `sets.json` as the `sm` of `crest-moon` in the ui-category set). The spec is [../specs/tt2-glyphs-diagrams.md](../specs/tt2-glyphs-diagrams.md).

## Status

- play-again: done (iteration 3)
- build-from-discard: done (iteration 2); weakest of the four at 12 px, see below
- destroy-card: done (iteration 1)
- crest-moon-sm: done (iteration 3)
- Browser check, family check and blind check: not yet run (the browser lock was held by another agent's Playwright run when the slice paused).

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
