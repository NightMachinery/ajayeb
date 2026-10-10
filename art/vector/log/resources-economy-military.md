# resources-economy-military: iteration log

## Status

- resource-wood and resource-wood-sm: done (iteration 4)
- resource-clay and resource-clay-sm: done (iteration 4)
- resource-stone and resource-stone-sm: done (iteration 4)
- resource-glass and resource-glass-sm: done (iteration 4)
- resource-papyrus and resource-papyrus-sm: done (iteration 4); its pale caller colour is a separate problem, see "Open points"
- coin: done (iteration 4)
- coin-loss: done (iteration 4)
- trade: done (iteration 4)
- price-tag: done (iteration 4); redrawn in M9.VG (iteration vg-2): a bigger body, no string
- choice: done (iteration 4)
- shield: done (iteration 4)
- military-pawn: done (iteration 4)
- looting-2 and looting-5: done (iteration 4)

No icon used more than 4 of its 6 iterations, and none is marked needs-human. The blind check was read by the author (see below), so a fresh reader should still confirm it.

## Iteration 0 (2026-10-10): the M5 placeholders, as a baseline

This iteration only proved the tools (slice M9.V0). Nothing was redrawn.

Renders: art-work/vector/resources-economy-military/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia and tritanopia, and the peeps).
Lint: 39 errors, all expected in the placeholders: every `--i2` falls back to a raw grey instead of `transparent`, several coordinates and stroke widths are off the 0.5 grid (2.2, 1.6 and so on), and some parts are under the 2-unit minimum.

What the baseline shows, for the agent who draws this set:

- resource-papyrus is a thin horizontal pill. In its pale tile colour it nearly vanishes on paper at 12 to 16 px, and in the mini card it reads as a dash. It needs more height (STYLE.md section 2: wide shapes at least 8 units tall).
- resource-wood and resource-clay are both blocks divided into cells; at 12 px they differ mainly by colour. In greyscale they are close. The new wood must not be a grid.
- resource-stone and resource-stone-sm are a grey hexagon; it reads as "a gem" as easily as "a stone", and glass is also a faceted gem. The two need different families of shape.
- trade (two chevrons) is light and thin; at 12 px it is two specks. Its detail tone carries half the glyph, so on the night surface, where the set's i2 was the card ink, half of it disappeared.
- looting-2 and looting-5 differ only by their dot clusters under the arrow; at 12 px the clusters are mush. The value must not depend on counting dots.
- coin and coin-loss are discs of the same size; coin-loss relies on red colour and a slash. In greyscale the slash still separates them, which is good.
- military-pawn is legible; shield is legible on every band.

In the app (`/dev/gallery` at 360x740 and 1280x800, light and dark; shots in the run directory under shots/M9/V0/):

- On the military track the looting glyphs are drawn at 10 to 12 px inside small discs; the dot clusters that tell looting-2 from looting-5 are invisible there, so today only the printed value under the disc separates them.
- In the mini cards, the papyrus tile reads as a dash, and the stone and glass tiles are small grey and blue blobs; the wood and clay tiles hold up.

## Blind check (2026-10-10): tool proof only

Reader: the author of the tools, who had seen the labelled specimen first, so this was **not** a blind reading.
Score: 12 px 14/14, 16 px 14/14. This number says nothing about the placeholders' quality; it only proves the sheet, key and scorer work. The real blind check of this set needs a fresh reader.

## Iteration 1 (2026-10-10): first drawing of all 19 files

Renders: art-work/vector/resources-economy-military/fast-1/ (librsvg quick looks only: `fast.mjs` strips, plus a 600% point-filtered peep of 9, 12 and 16 px on paper, on night and as a plain silhouette). No browser specimen this round.
Lint: clean.

The idea of each icon, one sentence each, before drawing:

- resource-wood: three upright rounded staves, the middle one taller, lashed by a band in the detail tone (stacked, vertical).
- resource-clay: a running-bond wall of five bricks in two courses with 2-unit mortar gaps (rectilinear, wider than tall).
- resource-stone: a low, rounded boulder with one chisel facet in the detail tone (irregular).
- resource-glass: a flask with a lip, a narrow neck and a round body, with a quarter-ring highlight knocked out of the body and filled with the detail tone.
- resource-papyrus: an open scroll, two tall rounded rolls joined by a sheet with knocked-out text lines (wide and rolled, 16 units tall).
- coin: a disc of 19 units with a square hole knocked out and filled with the detail tone (our own construction: a holed cash coin, no numeral, no medallion).
- coin-loss: the coin in two halves split by a zigzag crack, the halves offset.
- trade: two heavy straight arrows in opposite directions, 3-unit shafts and 10-unit heads, both in the main tone.
- price-tag: a tag tilted 45 degrees with a punched hole and a string in the detail tone.
- choice: a disc split into a solid half and a hollow half.
- shield: a heater shield whose top rises to a point, with a round boss knocked out and filled with the detail tone.
- military-pawn: one path for head, neck, flared body and base, with the halo stroke for the track.
- looting-2 and looting-5: a pile of large coins, two for looting-2 and three for looting-5, the front coin carrying the coin's square hole; each coin behind is cut by a clip path so a 2-unit gap separates the coins.

Seen and judged:

- resource-stone: the low rounded mass reads as a dome, a bell or a helmet, not a rock; it is too symmetric (the spec asks for irregular, not symmetric).
- coin-loss: at 9 and 12 px the zigzag closes up and it reads as a disc with a vertical slit.
- choice: at 9 and 12 px the hollow half fills in and it also reads as a disc with a vertical slit, so coin-loss and choice are confusable at the smallest sizes.
- resource-wood: reads as staves or a fence, distinct from the bricks by orientation. Acceptable.
- The other icons read at every size; looting-5's three coins merge into a cluster at 9 px but still differ from looting-2's two.

## Iteration 2 (2026-10-10)

Renders: art-work/vector/resources-economy-military/iter-2/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, the three peeps). The candidates of this round (stone-a to stone-f, wood-a, wood-b, coin-loss-a, choice-a, choice-b) were peeped with librsvg in a scratch directory only.
Lint: clean.

### resource-stone

- Seen: dome or helmet, symmetric (iteration 1). Candidates tried: two stacked angular stones (reads as an arch at 12 px, because the gap between them dominates), a single polygon with a leaf-shaped facet (still a dome), a polygon with two facet holes (reads as a skull), and an asymmetric boulder with a stepped top.
- Changed: the asymmetric boulder: a broad flat base, a low shoulder on the left, a higher crown right of centre, and the lit top face in the detail tone. The -sm file keeps the same silhouette with no facet.

### resource-wood

- Seen: acceptable. Candidates tried: the staves fanned out from the lashing (reads as a crown or a hand at 9 px, and a crown is the purple category glyph) and three log ends with growth rings in a 2 plus 1 pile (the most "wood" at 24 px, but at 12 px three ringed discs, too close to the looting coin piles).
- Changed: nothing; the upright staves stay.

### coin-loss

- Changed: each half is rotated 15 degrees outward about the bottom of the crack, so the break opens into a wedge at the top. It now reads as broken at 9 px.

### choice

- Changed: the split disc became a solid Y fork (branches 3.5 units wide, the stem 4). It reads at 9 px and cannot be mistaken for any disc. The spec offers both metaphors; the fork is the one that survives the 8 px dot on the resource chips.

### The first browser specimen

- Light and dark: every icon reads in its column; parity is good. On the night surface resource-wood (the wood token, a mid brown) has low contrast at 12 px, which is the colour, not the shape.
- Bands at 14 px: every glyph separates from all seven bands.
- Greyscale: all fourteen shapes are distinct; resource-papyrus is very pale on paper because its token is pale (see "Open points").
- Deuteranopia and protanopia: the wood and clay inks collapse to the same olive, but the staves and the bricks still separate them, which is the purpose of the silhouette families. Tritanopia changes nothing that matters.
- Peeps at 12 to 16 px: no blobbing. The pawn's head joins the body at 12 px DPR 1 but the pawn still reads. The price tag looks optically smaller than its neighbours.

## Iteration 3 (2026-10-10)

Renders: art-work/vector/resources-economy-military/iter-3/ (all columns, as above).
Lint: clean (a live-area warning on the tag's string during the edit, fixed by ending the string at 20.5, 3.5).

### price-tag

- Seen: optically small at 12 to 16 px (iteration 2 peep).
- Changed: the tag body grows from 9 by 15 to 10 by 17.5 units before its 45 degree turn; the string starts at the new hole.

### military-pawn

- Seen: the neck is a little wide, so at 12 px the head and body form one wedge.
- Changed: the neck narrows from 5 to 4 units and the body flares to the base from y 10.

### resource-papyrus

- Seen: the two knocked-out text lines blur into one grey hole at 12 px and lighten a glyph whose caller colour is already pale.
- Changed: one heavier line (6 by 2.5 units) in the sheet, filled with the detail tone.

## Iteration 4 (2026-10-10)

Renders: art-work/vector/resources-economy-military/iter-4/ (light@1x, light@2x, dark@1x, dark@2x, grey-light@1x, deuteranopia-light@1x, protanopia-light@1x, tritanopia-light@1x, peep-light@1x, peep-light@2x, peep-grey-light@1x, and the in-context mock columns of the light and dark shots); all looked at.
Lint: clean for the set. The whole sprite is 21486 bytes raw and 3842 gzipped (limits 40960 and 10240).

### resource-stone

- Seen: in iteration 3 the boulder is 15.5 units tall and looks smaller than the flask and the scroll at 12 to 16 px.
- Changed: the crown rises by one unit and the base drops by half a unit (now 17 units tall and 19 wide); the -sm file follows on its own grid.

Final judgement of the set in iteration 4, against the spec:

- The five resources differ by silhouette family (stacked staves, a rectilinear wall, an irregular boulder, a necked vessel, a wide open scroll), in plain ink, in greyscale and under the three colour-vision emulations.
- In the mini card mock and in the app's mini cards the -sm glyphs read on their tiles at about 9 px, including at DPR 1.
- coin, coin-loss, trade, price-tag, choice, shield, pawn and looting share one weight: solid silhouettes, knockouts of 2 units or more, one detail at most.
- looting-2 and looting-5 differ by two against three large coins, which survives the 10 px track discs (checked in the app's compact track at DPR 1).

## Blind check (2026-10-10)

Reader: the author (this agent), who drew the icons and saw them labelled in every iteration, so this was **not** a blind reading. A fresh reader should repeat it.

- Coloured sheets (`blind.mjs make`, each icon in its on-card ink and tile): 12 px DPR 2: 14/14; 16 px DPR 1: 14/14. Directory: art-work/vector/blind/resources-economy-military-2026-10-10T15-07-58-346Z/.
- Plain ink sheets (`blind.mjs make --mono`, shape only): 12 px DPR 2: 14/14; 16 px DPR 1: 14/14. Directory: art-work/vector/blind/resources-economy-military-2026-10-10T15-08-25-961Z/.
- Per icon: every icon of the set was named correctly on all four sheets. The answers were written into answers.txt before scoring, and the key file was never opened by the reader (only the scorer read it).

## Family check (2026-10-10)

`family.mjs` at 16 and 24 px in both themes: this set's weight, disc size and density sit with the other sets. Two cross-set notes for the owners of the other sets:

- category-grey is a round-bellied jar and resource-glass is a flask. They live in different places (the category glyph on medium and full cards, the resource in tiles), but if the grey category glyph stays a vessel it should not grow a neck like the flask.
- chain-column is an upright "I" and resource-papyrus is a wide "H" on its side. They separate by orientation; keep it that way.

## The app (2026-10-10)

`/dev/gallery` from the test build served by the Go server, at 1280x800 (DPR 1) and 360x740 (DPR 2), light and dark; section shots in the run directory under shots/M9/V1/ (desk-light, desk-dark, phone-light, phone-dark). Looked at: the icon grid, the affordability badges, the mini tier, the resource chips with the choice dot, the player strips, the military track (full and compact) and the standalone looting tokens. The probes report no horizontal scroll, no console errors and no failed requests.

- Mini cards: the wood, clay, stone, glass and papyrus tiles read at DPR 1 in both themes.
- Badges: the coin in the "own" pill and the trade arrows in the "trade" pill read at the mini size.
- The choice Y reads inside the 12 px gild dot of the resource chips.
- Track: the pawn reads in its ring; looting-2 and looting-5 differ at 10 px in the compact track.

## Open points (not glyph problems)

- The papyrus caller colour (`--res-papyrus`) is so pale that the glyph is faint on paper wherever it is coloured with that token rather than drawn on its tile: the resource chips, the strip chips, the badges and the specimen's column A. In greyscale it nearly vanishes on paper. The glyph cannot fix this; a deeper text tone for papyrus in `RESOURCE_TEXT_CLASS` (for example the papyrus colour mixed with the card ink) would. The same holds, less severely, for `--res-wood` on the night surface.

## M9.VG: price-tag (2026-10-10)

The M9b family review (art-work/vector/review/family-r1.md) found price-tag optically the smallest glyph of this set and a brown smudge on the yellow band at the desktop mini size (EffectGlyph trade-fix draws it at 0.9 em, about 9 px there). Coin and gold glyphs being pale on paper is a caller-colour problem (a deeper ink token), not an icon change, and was left alone.

### Iteration vg-0: candidates

Renders: art-work/vector/vg/s1-badge-tag.png and s12-tag.png (12, 15, 16 and 24 px at 600%, 96 px, silhouette).

- Tried: a 12-unit body (tag-a), a 13-unit body (tag-b), the same rotated about another point (tag-c), a 14-unit body with a blunter point (tag-d, tag-e). The string went: it took the corner the body needed, and the app never shows it (the only caller sets no detail tone).

### Iteration vg-1 (2026-10-10)

Renders: art-work/vector/resources-economy-military/iter-vg-1/ (light, dark, DPR 1 and 2, grey, three CVD, peeps).
Lint: clean (19 files).

- Seen (tag-b): bigger than before, but in the app's desktop mini row (shots/M9/VG/crops/mini-yellow-tags.png) still the smallest glyph beside its coin.
- Changed (vg-2): tag-e, 14 units wide and 17.5 long with a blunter point, rotated about the centre.

### Iteration vg-2 (2026-10-10): final

Renders: art-work/vector/resources-economy-military/iter-vg-2/: light@1x.png, light@2x.png, dark@1x.png, dark@2x.png, grey-light@1x.png, deuteranopia-light@1x.png, protanopia-light@1x.png, tritanopia-light@1x.png, peep-light@1x.png, peep-light@2x.png, peep-grey-light@1x.png, and crops/tag-all.png (the price-tag row in every pass).
Lint: clean.

- Seen: the tag is now as big optically as the shield and the coin-loss beside it, reads as a tag with a hole from 14 px and as a tag shape at 12 px, keeps its shape in greyscale and under the three emulations, and separates from all seven bands.
- Acceptance (resources class) for price-tag: lint clean, budget met, distinct in greyscale, legible on all seven bands, no blobbing in the peep (the hole fills at 12 px DPR 1, the shape stays a tag), light and dark parity.
- Self blind check (author-read, not blind), art-work/vector/blind/resources-economy-military-2026-10-10T18-11-06-030Z/: 12 px DPR 2 14/14, 16 px DPR 1 14/14. A fresh reader must take the real reading.
- Spec corrected (art/vector/specs/resources-economy-military.md): no string, the body size.
