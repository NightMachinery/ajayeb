# resources-economy-military: iteration log

## Status

- Every icon: placeholder from M5, not yet drawn in the M9 style.

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
