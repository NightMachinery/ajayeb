# chain: iteration log

## Status

- chain-horseshoe: done (iteration 1); M9.VG (iteration vg-1) made the nail holes plain knockouts
- chain-sword: done (iteration 2)
- chain-tower: done (iteration 1)
- chain-book: done (iteration 1)
- chain-gear: done (iteration 1)
- chain-amphora: done (iteration 2); the weakest of the set at 12 px, see "Open points"
- chain-mask: done (iteration 1)
- chain-moon: done (iteration 1)
- chain-drop: done (iteration 1)
- chain-target: done (iteration 2)
- chain-helmet: done (iteration 1, after a candidate round in iteration 0)
- chain-lyre: done (iteration 3)
- chain-lamp: done (iteration 4)
- chain-barrel: done (iteration 1)
- chain-sun: done (iteration 1)
- chain-column: done (iteration 1)
- chain-temple-facade: done (iteration 1)

No icon used more than 4 of its 6 iterations, and none is marked needs-human. No `-sm` variant was needed: every symbol reads in the 12 px chip. Every file is lint clean and between 158 and 660 bytes; the whole sprite is 21222 bytes raw and 4378 gzipped (limits 40960 and 10240).

## Iteration 0 (2026-10-10): fast-render sketches before the first specimen

Renders: art-work/vector/chain/scratch/ (a contact sheet of the whole set made with rsvg-convert: the 12 and 15 px glyph in the gold "in" chip and the paper "out" chip, a 16 px silhouette with `--i2` unset, 16 px on night, and 24 and 48 px; the small rows enlarged 400% with point filtering) and art-work/vector/chain/cand/ for candidates.

The M5 placeholders were replaced wholesale. The idea of each icon, one sentence each:

- chain-horseshoe: a thick U on its arch, opening down, flat heels, three nail holes per side knocked out and filled with the detail tone.
- chain-sword: a broad straight blade, point up, a guard, a short grip and a round pommel.
- chain-tower: a tall block under a corbelled parapet with three merlons, a pointed door (detail tone) and a window slit.
- chain-book: a closed codex, spine to the left, its page block showing as a notch at the bottom right (detail tone), a lozenge medallion on the cover (detail tone).
- chain-gear: a ring with eight square teeth, the hole filled with the detail tone.
- chain-amphora: a rim, a narrow neck, two square handles from neck to shoulder, a tapering body ending in a pointed toe.
- chain-mask: a theatre mask, wider at the brow, almond eyes (detail tone) and a smiling mouth knockout.
- chain-moon: a thick crescent opening to the upper right, pointed horns, no star (the star belongs to crest-moon).
- chain-drop: a teardrop pointing up with a quarter-ring highlight low on the left (detail tone).
- chain-target: a ring and a bullseye with an arrow stuck into the centre from the top right, the rings cut back around the shaft.
- chain-helmet: a crested helmet in profile, facing right, the crest and the bowl one silhouette, an eye notch at the front and a cheek guard.
- chain-lyre: a sound box, two arms, a crossbar, strings in the detail tone.
- chain-lamp: a low oil lamp with a lid knob, a spout to the right and a flame rising from it.
- chain-barrel: a bulging upright cask with two hoops knocked out and filled with the detail tone.
- chain-sun: a disc and eight short triangular rays, separated from the disc by a 1.5-unit gap so it is not the eight-pointed star of category-green.
- chain-column: an abacus and echinus, a shaft with two flutes (knockouts filled with the detail tone), a stepped base.
- chain-temple-facade: a pediment over four columns on a base, wider than tall.

What the sheets showed, and what changed before the first specimen:

- chain-helmet, first idea (the crest drawn in the detail tone over a separate bowl): with `--i2` unset only the bowl was left, an arch that at 16 px matched the horseshoe. Five candidates were drawn: the crest floating above the bowl, the crest merged with the bowl with a detail-tone groove between them, a front-view helmet with a T-shaped face opening and a fin, a pointed helmet with a nasal bar, and a merged crest fan. The merged crest with the groove read as a Greek helmet at every size and in silhouette; the front view also read but resembled a small door at 12 px. Kept the merged crest.
- chain-amphora: the first jar was a balloon on a stub, and its detail band across the belly read as a mouth, so the jar looked like a face. Redrawn slimmer with square handles and no band; still too much like an acorn at 12 px (see iteration 1).
- chain-column: the first flutes were painted on a solid shaft, so they vanished with `--i2` unset. They are now knockouts filled with the detail tone.
- chain-sword: optically the lightest glyph and close to a plain cross at 12 px. Blade widened from 3 to 4 units.
- chain-lamp: the flame floated far from the spout. Moved next to the spout tip.

## Iteration 1 (2026-10-10)

Renders: art-work/vector/chain/iter-1/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the three peeps; crops of every column in crops/).
Lint: clean (0 errors, 0 warnings).
Looked at: light@1x, the light@2x crops of columns C and D (bands and the in-context mocks), peep-light@1x.

- The mini card mock shows the gold "in" chip in the cost row and the paper "out" chip on the band at the same time: every symbol reads as the same symbol in both chips.
- On all seven bands at 14 px every symbol separates from the band through its chip.

### chain-amphora

- Seen: at 12 to 16 px the square handles and the rim merge into one wide top over a tapering body: a heraldic shield or an acorn, not a jar.
- Changed (iteration 2): three candidates. Round-cornered handles starting below the rim so the neck shows, a rounder belly and a pointed toe (kept); the same with a detail band (the band again read as a mouth); large looped handles with a flat foot (a vase, and the spec asks for a pointed foot).

### chain-sword

- Seen: still a thin cross with a dot at 12 px.
- Changed (iteration 2): blade 5 units wide with a 3.5-unit point, the guard curved down into quillons, the pommel unchanged. A fuller in the detail tone was tried and dropped: a 1-unit fuller is under the minimum, and it turned to a gold speck.

### chain-target

- Seen: two rings, a centre and the arrow cuts are too many parts for 16 px; at 12 px they mash into a grey disc.
- Changed (iteration 2): one ring and a solid bullseye (the "two rings" of the spec become a ring and a disc), a longer shaft and larger fletching.

### chain-lyre

- Seen: at 12 to 16 px the straight arms and three gold strings read as a bucket or a grille.
- Kept for one more round to compare against the others.

### The rest

- chain-horseshoe, chain-tower, chain-book, chain-gear, chain-mask, chain-moon, chain-drop, chain-helmet, chain-barrel, chain-sun, chain-column, chain-temple-facade: read at every size, on paper and on night, in both chips. No change.

## Iteration 2 (2026-10-10)

Renders: art-work/vector/chain/iter-2/.
Lint: clean.
Looked at: peep-light@1x, the dark@2x crops of columns A to C, grey-light@1x.

- chain-amphora: neck, handles and toe now read from 14 px; at 12 px it is a jar with ears. Accepted.
- chain-sword: reads as a sword from 12 px; the quillons separate it from a cross.
- chain-target: ring, bullseye and arrow read at 16 px; at 12 px the arrow is a notch in a ring, still unlike any other disc of the set.
- Greyscale: the set is ink on gold or paper, so greyscale changes nothing that matters.

### chain-lyre

- Changed (iteration 3): redrawn with arms as 2-unit strokes that bow outward and curl out at the top, a crossbar across the curls, a round sound box, and two strings in the detail tone (three strings at the 2-unit minimum with 1.5-unit gaps need a 12-unit opening and made the grille). Spec corrected; see below.

## Iteration 3 (2026-10-10)

Renders: art-work/vector/chain/iter-3/.
Lint: clean.
Looked at: light@1x, peep-light@1x, peep-light@2x, the dark@2x crop of columns C and D, deuteranopia and tritanopia.

- chain-lyre: the curled arms make it a lyre at every size; it no longer looks like a bucket.

### chain-lamp

- Seen: in the peep the flame is a round-bottomed drop above a whale-shaped body. A reader who knows chain-drop could read "a drop", which the spec forbids.
- Changed (iteration 4): three candidates; kept a pointed lens flame (pointed at both ends) on a slightly larger body. The lens is a flame, and the drop has a round bottom.

## Iteration 4 (2026-10-10)

Renders: art-work/vector/chain/iter-4/: light@1x.png, light@2x.png, dark@1x.png, dark@2x.png, grey-light@1x.png, deuteranopia-light@1x.png, protanopia-light@1x.png, tritanopia-light@1x.png, peep-light@1x.png, peep-light@2x.png, peep-grey-light@1x.png, and the crops in crops/ (columns A and B in three parts and columns C and D in two parts, light and dark).
Lint: clean (0 errors, 0 warnings).
Looked at: every PNG listed above.

- Light and dark: parity is good; the detail tone (gold-deep) shows on paper and on night.
- Bands at 14 px: every symbol is legible on all seven bands inside its chip.
- Mocks: in the mini card the in chip and the out chip show the same symbol; strip chip, badge and track cell all read.
- Greyscale and the three colour-vision passes: nothing changes; the set does not rely on colour.
- Peeps at 12 to 16 px: no blobbing. The weakest at 12 px DPR 1 are the amphora (a jar with ears), the lamp (the lens flame is 1 px wide) and the target (the arrow is a notch). Each is still unlike every other symbol of the set.

Group check (inside each colour group of the spec, the silhouette families differ):

- Red: horseshoe (an arch open at the bottom), sword (a thin upright with a crossbar), tower (a crenellated block), target (a ring with an arrow), helmet (an asymmetric profile with a crest).
- Blue: mask (a face, wide at the top), moon (a crescent), drop (a teardrop), sun (a spiked disc), column (a capital "I"), temple front (a wide triangle over columns).
- Green: book (a rectangle with a notch and a medallion), gear (a toothed ring), lyre (a horned frame), lamp (a low wide body with a flame).
- Yellow: amphora (a jar with handles and a toe), barrel (a bulging cask with hoops).

Spec corrections (art/vector/specs/chain.md):

- chain-lyre: two strings, not three (see iteration 2).
- chain-lamp: the flame is in the main tone, a pointed lens. In the detail tone the lamp lost its flame with `--i2` unset, which STYLE.md section 3 does not allow.
- Shared rules: the detail tone is optional; seven symbols with no natural secondary layer use the main tone only.

## Blind check (2026-10-10)

Reader: the author (this agent), who drew the icons and saw them labelled in every iteration, so this was **not** a blind reading. It only shows that nothing in the set is confusable for someone who knows it. A fresh agent that has never seen the labelled specimen must do the real blind reading; the sheets and names.txt in the directories below can be handed over as they are, and a new `blind.mjs make` run gives a new shuffle.

- Chip ink (card ink with the gold-deep detail), art-work/vector/blind/chain-2026-10-10T15-57-47-956Z/: 12 px DPR 2 17/17, 16 px DPR 1 17/17.
- Plain ink, shape only (`--mono`), art-work/vector/blind/chain-2026-10-10T15-58-14-164Z/: 12 px DPR 2 17/17, 16 px DPR 1 17/17.

Misses: none.

## Family check (2026-10-10)

art-work/vector/family/family@2x.png. Beside the resources and science sets at 16 and 24 px, the chain set has the same solid manuscript weight, the same 2-unit strokes (the lyre's arms) and the same corner treatment. It is a little denser than science: the mask, the book and the barrel are large solid masses, which suits a symbol that sits alone in a chip. The sword is the lightest glyph of the set but holds its own beside the science quill. No outlier.

## The app (2026-10-10)

Shots in the run directory under shots/M9/V3/sections/: the /dev/gallery sections Icons, Affordability badges, Tier band, Tier mini, Tier medium, Tier full, Age structures and City, at 360x740 (DPR 2) and 1280x800 (DPR 1), light and dark. (The full-page captures in shots/M9/V3/ are blank: the page's autofocused name field scrolls the document while the full-page shot is taken, so each section was shot separately with focus scrolling disabled.)

- 360x740 DPR 2, mini cards: every out chip on the bands (horseshoe, sword, tower, book, gear, amphora, mask, moon, drop, target, helmet, lyre, lamp, barrel, sun, column, temple front) and every gold in chip in the cost rows reads at a glance, in light and dark.
- Tier band: the out chips at the end of each band read clearly.
- 1280x800 DPR 1, mini cards: the chip glyph is about 9.5 px here (the mini card's font is clamped to 10 px, and the glyph is 0.95 em), smaller than the 12 px the spec assumes. Horseshoe, sword, tower, moon, drop, sun, column, temple front, helmet and target still read; mask, book, barrel, lyre and amphora are recognisable only to a player who knows the set. This is a property of the mini tier at desktop width, not of the drawings; see "Open points".
- Medium and full tiers: the chips read cleanly beside the shields and science glyphs.

## Open points

- The desktop mini tier draws chain glyphs at about 9.5 px DPR 1. If the real blind reading at that size falls short, the options are a `-sm` set for the five densest symbols (mask, book, barrel, lyre, amphora) or a larger chip glyph in ChainChip; both are outside this set's files.
- chain-amphora is the weakest symbol at 12 px. If the fresh reader misses it, the next idea is larger looped handles that stand clear of the body.

## M9.VG: chain-horseshoe (2026-10-10)

The M9b family review (art-work/vector/review/family-r1.md) found the horseshoe's six 2-unit nail holes, each filled with the detail tone, to be the scattered small `--i2` specks STYLE.md section 5 warns against: gold dust at 16 px.

### Iteration vg-1 (2026-10-10): final

Renders: art-work/vector/chain/iter-vg-1/: light@1x.png, light@2x.png, dark@1x.png, dark@2x.png, grey-light@1x.png, deuteranopia-light@1x.png, protanopia-light@1x.png, tritanopia-light@1x.png, peep-light@1x.png, peep-light@2x.png, peep-grey-light@1x.png, and crops/hs-lg.png, crops/horseshoe-dark2x.png.
Lint: clean (17 files).

- Changed: the detail-tone layer is gone; the six holes stay as knockouts of the main tone, which is what the spec already asks for ("nail holes as knockouts"). The file shrinks to one path.
- Seen: at 16 px and up the holes are small, crisp dots of the chip colour instead of gold specks; at 12 px they fill in and the glyph is a plain thick U, which the spec allows ("none at 12 px"). In and out chips, the seven bands, greyscale and the three emulations all read as before.
- Self blind check (author-read, not blind), art-work/vector/blind/chain-2026-10-10T17-52-59-898Z/: 12 px DPR 2 17/17, 16 px DPR 1 17/17. A fresh reader must take the real reading.
