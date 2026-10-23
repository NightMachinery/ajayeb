# haftgane-seat-crests: iteration log

## Status

- crest-0 (lotus): done (iteration 3)
- crest-1 (palm): done (iteration 3)
- crest-2 (scarab): done (iteration 3)
- crest-3 (falcon): done (iteration 4), replacing the eagle of iterations 1 to 3
- crest-4 (wave): done (iteration 1)
- crest-5 (anchor): done (iteration 3)
- crest-6 (fish): done (iteration 3)
- No `-sm` files: every md drawing reads at 11 and 12 px.
- Seat colours `--seat-0` to `--seat-6` and the ore and cloth tokens: done, in `web/src/lib/games/haftgane/theme.css`.

Every file is lint clean and between 335 and 533 bytes. The files live in `web/src/lib/games/haftgane/icons/svg/`, so Duel's icon source set (27252 bytes raw, 5638 gzipped) is unchanged.

The private renders are under `art-work/vector/haftgane-seat-crests/` in the run directory. Besides the specimens, iterations 3 and 4 have a `sheets/` directory made by a private script: `rings@2x.png` (every crest in its ring at the five mockup sizes, with the bot badge, a far chip, the Home tile and the pass flow, both themes), the same in greyscale and under the three colour-vision emulations (`rings-grey@2x.png`, `rings-deuteranopia@2x.png`, `rings-protanopia@2x.png`, `rings-tritanopia@2x.png`), `swatches@2x.png` (the seat colours on both panels with their contrast ratios, and the ore and cloth tiles beside Duel's five) and `contrast.txt` (the numbers below).

## Iteration 1 (2026-10-23): fast-render drafts

Renders: art-work/vector/haftgane-seat-crests/iter-1/ (librsvg strips at 11, 12, 15, 20 and 64 px on paper and night, `fast-all.png`, and the 300% enlargement `fast-all-3x.png`), and `neighbours/all-48.png`, the excluded symbols (the neighbours, the chain glyphs, the conflict pennants and the pass mark) side by side at 48 px.

The first drafts, each from the spec's sentence:

- crest-0: pointed petals rising from a cup on a stem.
- crest-1: a curved trunk and a crown of drooping fronds.
- crest-2: an oval body split down the back, a toothed head and three straight legs on each side.
- crest-3: an eagle displayed, wings raised with five feather points on each lower edge, a profile head and a fanned tail.
- crest-4: a band of water with one wave curling right into a hollow.
- crest-5: a ring, a stock, a shank and two arms with flukes.
- crest-6: a lens-shaped fish swimming right, a dorsal fin, a forked tail and a knocked-out eye.

What the strips showed:

- crest-0: reads as a chalice or a trophy: a cup on a stem.
- crest-1: reads as a palm at every size, but the fronds, drawn as subpaths of one path, overlap with opposite windings and leave hairline seams at 64 px.
- crest-2: the straight legs read as rays; at 12 px the scarab is a sun with an oval core.
- crest-3: a spiky blob at 11 and 12 px; the feather points merge into a grey fringe.
- crest-4: reads as a wave at every size and stays apart from the moon, because the curl stands on a band. No change from here on.
- crest-5: reads as an anchor, but the flukes, drawn with `evenodd` over the arms, leave holes where they overlap.
- crest-6: reads as a fish, with the same kind of flaw where the tail meets the body.

## Iteration 2 (2026-10-23): fast renders

Renders: art-work/vector/haftgane-seat-crests/iter-2/ (`fast-all-3x.png`, and `top4.png`, the doubtful crests enlarged).

- crest-0: the cup replaced by a fan of pointed petals on a short stem. Seen: reads as a cannabis leaf at 15 px and up.
- crest-2: the legs bent at the knee, pointing forward and back, with round caps (a stroke path of width 2). Seen: reads as a beetle from 11 px; nothing of the sun is left.
- crest-3: fewer, larger feathers and a narrower body. Seen: at 15 px and up the pointed wings and the round head read as a bat or an angel; at 12 px still unclear.
- The seams and holes of iteration 1 were left for the next iteration, which rebuilt the files' structure.

## Iteration 3 (2026-10-23): the first specimen

Renders: art-work/vector/haftgane-seat-crests/iter-3/ (light and dark at DPR 1 and 2, grey, the three emulations in both themes, the peeps, and `sheets/`).
Lint: clean (0 errors, 0 warnings).
Looked at: `sheets/rings@2x.png` and its full-resolution crops of the small sizes (`light-small.png`, `dark-small.png`), `rings-grey@2x.png`, `rings-deuteranopia@2x.png`, `swatches@2x.png`, `contrast.txt`.

Changed, all files: every overlapping part is its own `<path>` inside one `<g>` that carries the fill, so nonzero filling cannot cancel an overlap. `evenodd` is kept only where a hole is meant (the anchor's ring, the fish's eye, the split down the scarab's back). The seams of the palm and the holes of the anchor and the fish are gone.

Before this iteration, two scratch trials each (`scratch/`):

- Lotus: a broad yoga-style lotus with no stem, five petals leaning out from a shallow bowl (lotus-a), against a narrower fan on a pad (lotus-b). Kept lotus-a: the bowl, not a stem, says flower, and the flat outer petals make it wide, unlike the crowns.
- Eagle: raised wings with the feather fingers on the outer edge only (eagle-a), against an eagle seen from below with the wings level (eagle-b). eagle-b reads as a cross or an aeroplane. Kept eagle-a.

What the sheets showed:

- crest-0, crest-1, crest-2, crest-4, crest-5, crest-6: each reads at every size from 11 in 16 to 28 in 44, in both themes, on the far chip and on the Home tile, and still reads with the bot badge over the lower right of its ring. In greyscale and under deuteranopia the seven stay apart by shape.
- crest-3: reads as a bird from 15 px. At 11 and 12 px it is a fuzzy W with no head, and beside the lotus on the Home tile both read as a spiky fan. This is the size that matters most (far chips, the pass flow, the recap).

## Iteration 4 (2026-10-23): a falcon in place of the eagle

Renders: art-work/vector/haftgane-seat-crests/iter-4/ and `scratch/birds.png`, `birds2.png`, `birds3.png` (fast strips of the trials beside the lotus, the fish and the palm).

Three bird constructions in scratch, at 11, 12, 15, 20 and 64 px:

- bird-a, the eagle displayed with smooth raised wings and two scallops: at 11 px a figure raising its arms, the same problem as before.
- bird-b, a bird in flight seen from the side: a V over a bar at 11 px.
- bird-f, a bird standing in profile, leaning forward: reads as a bird at 11 px, because its whole outline (round head, beak, breast, tail) is one solid mass.

Then three refinements of bird-f: a broad blunt beak (reads as a duck), the legs merged into a perch block (loses the legs' "standing" at 12 px), and a hooked beak (bird-f4). bird-f4 reads as a bird of prey at 20 px and up and as a bird at 11 px. Kept bird-f4 as crest-3, a falcon of Horus, which joins the scarab and the lotus as an Egyptian emblem. The spec's crest-3 section now describes the falcon and why the eagle was dropped.

## Seat colours (2026-10-23)

Measured with a private script (OKLCH to sRGB, WCAG contrast, OKLab distance, the Machado 2009 colour-vision matrices); the numbers are in `iter-3/sheets/contrast.txt` and on `swatches@2x.png`. The surfaces are tokens.css's panel, sunken panel and table, and the ring's fill, `color-mix(in oklch, <seat> 18%, var(--panel))`.

How the values were chosen:

- Each hue sits between two of Duel's card hues (brown 55, yellow 88, green 150, blue 255, purple 315, red 27), so a seat ring next to a city's colour counts is not mistaken for a card colour: rose 352, olive 122, violet 290, amber 70, azure 243, rust 41 and teal 182.
- Lightness is spread so that every value keeps 3:1 against all four surfaces of its theme, the glyph on its own ring included.
- The order around the table was found by trying every permutation of the seven: seats that sit next to each other at 3 to 7 players (seat i and i+1, and the last and seat 0) are the most distinct pairs. The worst such pair has an OKLab distance of 0.127 in both themes. The two closest colours overall, rust and amber (0.100), are seats 5 and 3, which are never neighbours.

Contrast, light theme (panel, sunken, table, ring fill):

- seat-0 rose oklch(0.56 0.18 352): 4.92, 4.05, 4.58, 3.90
- seat-1 olive oklch(0.55 0.13 122): 4.50, 3.70, 4.18, 3.56
- seat-2 violet oklch(0.5 0.17 290): 6.17, 5.07, 5.73, 4.71
- seat-3 amber oklch(0.59 0.12 70): 4.04, 3.32, 3.75, 3.26
- seat-4 azure oklch(0.5 0.11 243): 5.68, 4.68, 5.28, 4.40
- seat-5 rust oklch(0.52 0.15 41): 5.64, 4.64, 5.24, 4.38
- seat-6 teal oklch(0.56 0.1 182): 4.23, 3.48, 3.93, 3.38

Contrast, dark theme (panel, sunken, table, ring fill):

- seat-0 rose oklch(0.76 0.15 352): 6.58, 8.04, 7.55, 4.84
- seat-1 olive oklch(0.8 0.15 125): 8.40, 10.27, 9.64, 5.82
- seat-2 violet oklch(0.74 0.14 290): 6.32, 7.72, 7.25, 4.68
- seat-3 amber oklch(0.8 0.13 66): 7.91, 9.67, 9.08, 5.69
- seat-4 azure oklch(0.7 0.12 238): 5.77, 7.06, 6.63, 4.37
- seat-5 rust oklch(0.72 0.14 41): 5.80, 7.09, 6.66, 4.42
- seat-6 teal oklch(0.82 0.11 182): 9.08, 11.10, 10.43, 6.25

Distance between seats, normal vision: at least 0.100 in both themes (rust and amber); every other pair is at least 0.117.

Nearest card colour (OKLab distance; in the dark theme against the card inks of that theme): light rose to red 0.112, olive to green 0.070, violet to purple 0.076, amber to brown 0.085, azure to blue 0.059, rust to red 0.070, teal to green 0.072; dark rose to purple 0.121, olive to green 0.087, violet to blue 0.083, amber to yellow 0.066, azure to blue 0.041, rust to red 0.058, teal to green 0.106. Dark azure is the nearest to a card colour; it is told apart by the crest it always comes with, and a seat ring never stands where a card colour count does.

Colour vision: in greyscale and under each emulation some pairs fall together (greyscale rust and azure 0.002; deuteranopia teal and rose 0.010, amber and olive 0.008 in the dark theme; protanopia amber and olive 0.005). That is expected of seven hues at similar lightness, and it is why identity is carried by the crest's shape: on `rings-grey@2x.png` and the three emulation sheets every crest is still told apart.

## Ore and cloth tokens (2026-10-23)

- `--res-ore` oklch(0.42 0.035 270), a dark blue-grey iron. Distance to Duel's tiles: wood 0.146, clay 0.237, stone 0.201, glass 0.310, papyrus 0.455. The paper-bright glyph on it: 8.13. Inks: light oklch(0.42 0.035 270) (panel 8.13, sunken 6.69, table 7.56), dark oklch(0.68 0.045 275) (5.23, 6.39, 6.00).
- `--res-cloth` oklch(0.55 0.15 345), a madder red-violet. Distance to Duel's tiles: wood 0.162, clay 0.140, stone 0.168, glass 0.280, papyrus 0.365. The paper-bright glyph on it: 5.05. Inks: light oklch(0.5 0.15 345) (panel 6.26, sunken 5.15, table 5.81), dark oklch(0.74 0.13 345) (6.18, 7.55, 7.09).
- `swatches@2x.png` shows both glyphs on their new tiles at 12, 16 and 32 px beside Duel's five, and both inks on each theme's panel.

## Family check (2026-10-23)

Renders: art-work/vector/haftgane-seat-crests/family/ (`family.mjs --sets haftgane-seat-crests,haftgane-chain,haftgane-conflicts-passing-abilities,haftgane-resources-science,ornaments-track-logo`, at 16 and 24 px on paper and night).

- The crests match the chain and conflict glyphs in optical size and density: solid single-tone silhouettes that fill the 24 grid's live area like the barrel, the helmet and the camel.
- The anchor and the palm are the lightest of the seven (2-unit strokes and fronds), about as light as `chain-sword` and the compass; the lotus, the scarab and the wave are the heaviest, about as heavy as the helmet and the pennants. Inside the ring each reads at the same strength (iteration 4 sheets), so no weight change.
- The seat inks on paper and on night sit beside the chain set's plain ink and the science set's greens without clashing.

## Blind check (2026-10-23, iteration 4)

`blind.mjs make --set haftgane-seat-crests --sizes 12@2,16@1`, once in colour (seed 11) and once with `--mono` (seed 23): the seven crests among the fifteen neighbours of `sets.json` (`crest-sun`, `crest-moon`, `chain-star`, `chain-tower`, `chain-torch`, `chain-drop`, `chain-lighthouse`, `hint-crown`, `category-purple`, `vp-laurel`, `choice`, `chain-sword`, `conflict-victory`, `pass-hand`, `military-pawn`), 22 tiles per sheet. Each run had its own fresh reader, an in-process subagent that saw only the sheets, `names.txt` and one gloss line per name (for the crests: a lotus flower, a palm tree, a scarab beetle, a falcon, a breaking wave, an anchor, a fish), and never the key.

- Colour: 12 px DPR 2 22/22, 16 px DPR 1 22/22 (100%).
- Mono: 12 px DPR 2 22/22, 16 px DPR 1 22/22 (100%).

Neither reader hesitated over a crest. Both hesitated only between `hint-crown` and `category-purple`, two Duel crowns that differ only by how the band is set off (a stripe in the detail tone, or a gap), and both readers still named them right; and the mono reader named `crest-sun` by elimination because its face is not visible at 12 or 16 px. Neither doubt touches this set.
