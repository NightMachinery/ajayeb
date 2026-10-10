# ui-category: iteration log

## Status

- category-brown: done (iteration 1)
- category-grey: done (iteration 2, after candidates in iteration 0); redrawn in M9.VG (iteration vg-1): long, heavy horn
- category-blue: done (iteration 1)
- category-green: done (iteration 1, after candidates in iteration 0); spec corrected, see below
- category-yellow: done (iteration 1, after candidates in iteration 0)
- category-red: done (iteration 3); redrawn in M9.VG (iteration vg-2): 3-unit shafts, leaf heads, a boss
- category-purple: done (iteration 1)
- badge-check: done (iteration 1); redrawn in M9.VG (iteration vg-1): a filled, wedge-ended check
- badge-chain: done (iteration 1); redrawn in M9.VG (iteration vg-1): heavier links
- badge-lock: done (iteration 1)
- crest-sun: done (iteration 2, after candidates in iterations 0 and 1); spec corrected, see below; M9.VG (iteration vg-1) calmed the face to two eyes
- crest-sun-sm: done (M9.VG, iteration vg-3); new, for 14 px and below (the military track)
- crest-moon: done (iteration 2); spec corrected, see below; redrawn in M9.VG (iteration vg-2): a crescent with a face in profile, no star
- vp-laurel: done (iteration 1, after a generated candidate round in iteration 0); redrawn in M9.VG (iteration vg-1): three larger leaves per side
- wonder: done (iteration 2); redrawn in M9.VG (iteration vg-1): a pylon gateway, no towers
- token-frame: done (iteration 2); redrawn in M9.VG (iteration vg-1): eight fat lobes
- category-brown-sm, category-grey-sm, category-blue-sm, category-green-sm, category-yellow-sm, category-purple-sm: done (M9.VF1, iteration 1); new, for the colour swatches of EffectGlyph
- category-red-sm: done (M9.VF1, iteration 4); new, for the colour swatches of EffectGlyph

No icon used more than 3 of its 6 iterations, in the first drawing or in the M9.VG redraw (iterations vg-1 to vg-3, after a candidate round vg-0), and none is marked needs-human. Every file is lint clean. After M9.VG the whole sprite is 25392 bytes raw and 5271 gzipped (limits 40960 and 10240). Every blind check in this log was read by the author, so a fresh reader must still do the real one.

## Iteration 0 (2026-10-10): sketches and candidates before the first specimen

Renders: art-work/vector/ui-category/cand/ (contact sheets made with rsvg-convert: 12, 15, 16 and 24 px enlarged 600% with point filtering, 96 px, and a 16 px silhouette with `--i2` unset; `all-1a.png` to `all-4.png`, `laurel.png`). The laurel leaves were placed by a small generator script (gen/laurel.py in the same directory) that puts pointed leaves along a stem arc and snaps every point to the 0.5 grid.

The M5 placeholders were replaced wholesale. The idea of each icon, one sentence each:

- category-brown: two peaks, the front one larger, on a 20-unit base, with a zigzag snow cap in the detail tone painted over the front peak.
- category-grey: an anvil in profile, a pointed horn to the left, a flat face, a narrow waist and a splayed foot.
- category-blue: an onion dome rising to a point, on a wider wall with a round-arched door knocked out and filled with the detail tone.
- category-green: an eight-pointed star with an octagonal centre in the detail tone.
- category-yellow: a squat sack with a flared, gathered top above a tie; the tie is a detail-tone band bridging a 2-unit gap.
- category-red: two spears crossed at the centre, leaf-shaped heads up, small butt spikes.
- category-purple: a crown with three ball-tipped points over a band, separated by a 2-unit stripe in the detail tone.
- badge-check: a heavy check as a filled polygon, 2.8 units thick, both arms at 45 degrees.
- badge-chain: two stadium-shaped links of 2.5-unit rings, overlapping on the diagonal, holes 3 units wide.
- badge-lock: a solid body with a keyhole knocked out (filled with the detail tone) and a 2.5-unit shackle in the main tone.
- crest-sun: a disc with a calm face knocked out, ringed by flame-shaped rays.
- crest-moon: a crescent cradling a four-pointed star.
- vp-laurel: two stems curving up from the bottom centre, three leaves on the outside of each and an upright tip leaf, open at the top.
- wonder: an iwan, a block with a pointed-arch knockout, between two slender minarets with pointed caps, on a plinth.
- token-frame: a band from radius 6.5 to 8 with round scallops on its outer edge, empty inside.

What the sheets showed, and what changed before the first specimen:

- category-grey, first anvil (horn and heel the same length, body centred): at 12 to 16 px an I-beam. Kept for the first specimen to see it in Chrome, then redrawn (iteration 2).
- category-green, first star (two squares, the khatam, with a round knockout): at 12 to 16 px it is the same blob as chain-gear (eight teeth and a hole). Four candidates: two squares with a solid centre (still a gear), two sharp four-pointed stars (kept), the same with a deeper notch (too spiky), two concave squares (closer to the khatam, but rounder and still gear-like at 12 px). The kept star has sharp points and a solid silhouette; the octagonal centre is painted, not knocked out, so the silhouette column shows a plain star.
- category-yellow, first purse (a narrow frill over a round body): at 12 px the same silhouette as resource-glass (a lip, a neck and a ball). Redrawn with a wide flared top with two lobes and a squat, flat-bottomed sack.
- crest-sun: the first sun (eight flame rays, a ring knockout in the disc) read as a gear at 12 px, and so did a twelve-ray version with the ring. The crests get no detail tone in the app, so any ring knockout is a real hole. Two more candidates without the ring: sixteen alternating long and short rays with a solid disc (a spiky ball), and the same with a face (kept for the first specimen).
- crest-moon: three candidates: a crescent and star inside a 2-unit ring (the spec's roundel), a large crescent cradling a star with no ring, and a solid medallion with the crescent and star knocked out. The roundel version was kept for the first specimen.
- vp-laurel: the first generated wreaths splayed their leaves outward like petals (a flower or a gear), and the leaf tips ran outside the viewBox. Leaves turned to 30 degrees off the stem, the stem radius cut to 6, the top opened by ending the stems at 125 degrees, and the whole wreath fitted to 2 to 22. Of four variants, the one with three outer leaves and a tip leaf per side read as a wreath from 12 px.
- token-frame: twelve small scallops looked like a gear at 12 to 16 px; detached beads looked like a compass. Sixteen touching scallops were kept for the first specimen.

## Iteration 1 (2026-10-10)

Renders: art-work/vector/ui-category/iter-1/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, the three peeps, and an enlarged crop of the 12 to 16 px strip in crops/).
Lint: clean (0 errors, 0 warnings; a live-area warning on the laurel's side leaves was fixed before the shoot by pulling those two tips in by half a unit).
Looked at: light@1x, peep-light@1x, crops/strip-light1x.png.

### category-grey

- Seen: still an I-beam or a spool at 12 to 14 px; the horn is the anvil's only sign and it is no longer than the heel.
- Changed (iteration 2): the waist and foot move right so the body sits under the face's right two thirds, and the horn becomes a long cone from x 2 to 9.5. The asymmetry is what makes it an anvil.

### crest-moon

- Seen: the ring with the crescent inside reads as a copyright sign at 12 to 16 px.
- Changed (iteration 2): the large crescent cradling a star, no ring. The solid-medallion candidate blurred into a dark disc at 12 px.

### crest-sun

- Seen: the sixteen-ray sun with a face reads as a spiky ball, close to a germ, at 12 to 16 px.
- Changed (iteration 2): twelve equal flame rays, a disc of radius 6.5 and the face. Twelve rays separate it from every eight-fold shape of the family (chain-gear, chain-sun, category-green).

### token-frame

- Seen: at 12 to 16 px the sixteen small scallops vanish and it is a plain "O".
- Changed (iteration 2): twelve scallops of radius 2 on a band from radius 6.5 to 8; the inner radius stays 6.5, as the spec requires.

### wonder

- Seen: at DPR 1 the 1.5-unit gaps beside the towers close, so the towers merge into the block.
- Changed (iteration 2): the gaps widen to 2 units and the towers to 3 units with taller caps; the block top drops to y 8 so the towers stand clear of it.

### The rest

- category-brown, category-blue, category-yellow, category-red, category-purple, badge-check, badge-chain, badge-lock and vp-laurel read at every size on paper and night, on all seven bands, and in the mocks. No change.

## Iteration 2 (2026-10-10)

Renders: art-work/vector/ui-category/iter-2/ (all columns as above; crops/ holds the band and mock columns at 2x in light and dark, the 12 to 16 px strip under the three colour-vision emulations, and the badge mocks under greyscale and each emulation).
Lint: clean.
Looked at: peep-light@1x, peep-grey-light@1x, crops/mocks-a-light2x.png, crops/mocks-b-dark2x.png, crops/strip-cvd.png, crops/badges-cvd.png.

- category-grey: an anvil at every size from 12 px; the horn reads.
- crest-moon: a crescent with a star from 12 px; nothing like the copyright sign.
- crest-sun: a rayed sun with a face from 14 px; at 12 px DPR 1 a spiky disc, still nothing like the crescent.
- token-frame: a scalloped medallion from 16 px; at 12 px a ring.
- wonder: the towers stand apart from the block from 14 px.
- Greyscale: the seven category glyphs separate by shape alone (peaks, an anvil, a dome, a star, a sack, crossed spears, a crown). The yellow purse is pale on paper in greyscale because the card yellow is light; its shape still reads.
- Colour vision: under deuteranopia, protanopia and tritanopia every glyph keeps its shape; the check, chain and lock pills of the specimen stay apart by glyph.

### category-red

- Seen: at 12 px DPR 1 the spearheads are two pixels, so the glyph is nearly a plain X, which could read as "close".
- Changed (iteration 3): the heads grow from 5 by 8.5 to 6 by 9.5 units.

## Iteration 3 (2026-10-10): final

Renders: art-work/vector/ui-category/iter-3/: light@1x.png, light@2x.png, dark@1x.png, dark@2x.png, grey-light@1x.png, deuteranopia-light@1x.png, protanopia-light@1x.png, tritanopia-light@1x.png, peep-light@1x.png, peep-light@2x.png, peep-grey-light@1x.png.
Lint: clean (0 errors, 0 warnings).
Looked at: peep-light@1x.png, peep-light@2x.png, dark@2x.png, grey-light@1x.png (every column: A on paper with the silhouette, B on night, C on the seven bands, D the mini card, strip chip, badge and track cell mocks).

- category-red: the leaf heads show at 12 px DPR 1; it reads as crossed spears from 14 px.
- Light and dark: parity is good; on night every glyph keeps its weight.
- Bands at 14 px: every glyph separates from all seven bands (on its own band it is drawn as paper or ink by the specimen, as the app does with its halo-free inset).
- Silhouette column (`--i2` unset): every icon is recognisable. The detail tones (the snow cap, the door, the star centre, the purse tie, the crown stripe, the keyhole, the wonder's arch) are all secondary.

Acceptance against STYLE.md section 8, item by item:

- Category glyphs, "each category identifiable from its glyph alone in greyscale at the size of a 54 px wide card": met in the specimen's greyscale pass at 12 to 16 px (grey-light@1x.png and peep-grey-light@1x.png) and in the app's full-tier cards, where the glyph by the name is about 15 px. Each colour group has its own silhouette family; the closest pair is category-green (a sharp star with a light centre) and crest-sun (a rayed disc with a face), which never share a context except the score table, where they are a row icon and a column header.
- Badge glyphs, "the five badge states distinguishable in greyscale and under each colour-vision emulation at the mini size": met; see "The app" below.
- Crests, "shape-distinct, not only colour-distinct": met. A rayed disc against an open crescent, in greyscale at 15 px and in every emulation.
- vp-laurel, "counts with the resources class": lint clean, budget met, distinct in greyscale, legible on all seven bands, no blobbing in the peep (the leaves stay separate points from 14 px; at 12 px DPR 1 the outline is a toothed U open at the top), light and dark parity. Its blind reading is below (author-read only).
- All: lint clean, budgets met, family check looked at (below).

Spec corrections (art/vector/specs/ui-category.md):

- category-green: two four-pointed stars, not two squares (see iteration 0). The two-square khatam with a centre is chain-gear at 12 px.
- crest-sun and crest-moon: no outer roundel (see iterations 0 and 1). The ring made the moon a copyright sign and the sun a gear. The sun's face and its twelve rays set it apart from chain-sun; the moon's star sets it apart from chain-moon.

## Blind check (2026-10-10)

Reader: the author (this agent), who drew the icons and saw them labelled in every iteration, so this was **not** a blind reading. It is a smoke test only. A fresh agent that has never seen the labelled specimen must do the real blind reading; the sheets and names.txt in the directories below can be handed over as they are (or `blind.mjs make --reader-dir` gives a new shuffle without the key). The answers were written into answers.txt before scoring; only the scorer read the key.

- Coloured sheets (each icon in its own ink, which gives the category glyphs away by colour), art-work/vector/blind/ui-category-2026-10-10T16-32-35-832Z/: 12 px DPR 2 15/15, 16 px DPR 1 15/15.
- Plain ink, shape only (`--mono`), art-work/vector/blind/ui-category-2026-10-10T16-33-00-497Z/: 12 px DPR 2 15/15, 16 px DPR 1 15/15.
- Per icon: every icon was named correctly on all four sheets. The pair that took the longest on the mono sheets was category-green against crest-sun (both spiky discs at 12 px; the star has a light centre and sharper points).

## Family check (2026-10-10)

art-work/vector/family/family@2x.png. Beside the resources, science and chain sets at 16 and 24 px, this set has the same solid manuscript weight, the same 2-unit stems (the laurel) and the same corner treatment. badge-check and token-frame are the lightest glyphs of the set, by design (an open stroke-like check and an empty ring); they sit with trade and science-wheel. crest-sun, chain-sun, chain-gear and category-green are the four spiky discs of the family; they differ by twelve rays and a face, eight separate rays, square teeth and a hole, and eight sharp points with a light centre. No outlier.

## The app (2026-10-10)

Shots in the run directory under shots/M9/V4/: the /dev/gallery sections Icons, Affordability badges, Tier band, Tier mini, Tier full, Board, Choices, the ScoreTable and PassDevice (Sun), at 360x740 (DPR 2) and 1280x800 (DPR 1), light and dark, each shot with `shot.mjs --selector`. Crops and the colour-vision passes are in shots/M9/V4/cvd/.

- Affordability badges: at the mini size on the desktop (about 10 px at DPR 1) the five states are a link (chain, no number, light pill), a check (free, no number, dark pill), a coin and a number (own, dark pill), arrows and a number (trade, light pill) and a lock and a number (short, dashed pill). shot.mjs has no colour-vision emulation, so the greyscale and colour-vision passes of the app were simulated on the PNG with ImageMagick (greyscale, and the Machado 2009 deuteranopia, protanopia and tritanopia matrices in linear RGB): cvd/mini-row-1x.png and cvd/desk-4passes.png. In every pass the five states stay apart. Chain and trade become similar yellows or greys, and free and own similar dark pills, but the link against the arrows and the check against the coin with its number separate them. The specimen's badge mocks went through Chrome's real emulation in iteration 2 with the same result.
- Full and medium cards: the category glyph by the name (about 15 px) and the larger glyph in the placeholder art read in both themes.
- Score table: category, wonder, token-frame and laurel row icons and both crests in the header read at 360 px in both themes.
- Military track: the crests at the two capitals (about 12 px in small discs) are a spiky sun and a crescent, identifiable in the full and compact tracks.
- Progress tokens: the scalloped frame holds the initial at 26 and 36 px with room to spare.
- Player strips and city bands: laurel, wonder chips, crests and category glyphs all read at 11 to 16 px.
- PassDevice (60 px): the sun with its face reads well; it looks cheerful rather than strictly calm.

## Open points

- The score table marks the winner with Lucide's outline crown next to the total, while the guild row uses category-purple, also a crown. A player could take the winner mark for a guild figure. A different winner mark (for example vp-laurel) would remove the clash; that is a component change, outside this set's files.
- token-frame at 12 px is a plain ring; its scallops show from 16 px. Its smallest use is 1.1 em in an effect on a mini card (about 11 px), where it still differs from the coin (a solid disc) and the target.

## Blind reading round 1 (fresh reader), M9.VF1 (2026-10-10)

Slice M9.VF1 owned only the EffectGlyph colour swatches (the "per card of colour X" chips of commerce cards, guilds and the two destroy wonders) and their `-sm` glyphs, so this section records the round 1 results for that item. The round 1 brief passed this slice an empty list of blind results: no per-sheet scores and no misses were reported for any icon of this set, so there is nothing to copy here. The scores of the fresh reader, if they exist, belong to whichever slice received them.

Critic findings received (the family review, round 1):

- must, EffectGlyph colour swatches: the swatch was hue only (its CSS pattern is invisible at the mini and medium sizes). Green, red and brown became one grey in greyscale and one olive under deuteranopia and protanopia, so a colour-blind player could not tell which colour an effect counts. Evidence: shots/M9/review-r1/crops/mini-swatch-cvd.png. Fixed below.
- should, owned by other icons of this set and not by this slice (no change here): category-red reads as a bare X below 16 px; token-frame reads as a zero; crest-moon's star is a flag emblem; wonder reads as a mosque front, and as "lnl" at 12 px; crest-sun blurs at the track size; vp-laurel speckles at 12 px; category-grey reads as an I-beam at 12 to 16 px; category-yellow and the coin are faint on paper (a token task: `--card-yellow-ink` and friends); brown, purple and pomegranate inks are faint on the dark panels; badge-check and badge-chain are line-weight. These are listed in the slice result as requests. The new `-sm` glyphs below already address the red and grey readings for the swatch context only.

### Iteration 1 (swatches): category glyph chips

Renders: shots/M9/VF1/iter1-mini-1280-light.png, iter1-mini-rows-x2.png.

- Changed: each swatch is now a small card shape in the band colour (plain, no pattern) with the colour's category glyph on it, drawn in the band ink (paper on brown, blue, green, red and purple; card ink on grey and yellow) with `--i2` a 45% mix of ink and band. The hue still says "colour"; the glyph names which one.
- Seen: green, blue, yellow and brown read, at the medium size and on the mini card. Three problems. The red chip at the mini size (glyph about 9 px) showed a white X on red: a close button, exactly the misreading the critic gave for category-red. The grey anvil was a spool. The Shipowners Guild (brown and grey) overflowed its 54 px card: the band pill clipped the coin on the left and the anvil chip on the right.

### Iteration 2 (category -sm glyphs, first drawing)

Renders: art-work/vector/swatch/fast1/ and fast2/ (fast.mjs at 9, 10, 12, 16 and 48 px).

The swatch glyph is 9 px on a mini card and 15 px on a medium one, so it is a `-sm` use (STYLE.md section 1). Seven new files on the 12 grid, each one idea:

- category-brown-sm: two peaks on a 10-unit base, a zigzag snow cap in the detail tone.
- category-grey-sm: an anvil whose face runs into a long, heavy horn to the left (a 3-unit wedge tapering to a point), a 3-unit waist set to the right and a splayed foot, so the outline is plainly asymmetric.
- category-blue-sm: the pointed dome on its wall with the arched door knocked out.
- category-green-sm: two four-pointed stars with deep inner points and no centre detail (the light centre turned the star into a gear ring at 9 px).
- category-yellow-sm: the purse, its tie a 1-unit detail band across a 1-unit gap.
- category-red-sm: see iterations 2 to 4.
- category-purple-sm: three points on a band, a ball on each point, the detail stripe between.

Seen: the first anvil (horn curving into the face, as in md) was still an I-beam at 9 and 10 px; redrawn with the long wedge horn above, which reads as an anvil at 9 px. The first red (thin shafts, small lozenge heads, a diamond boss) was a plain X at 9 to 12 px. The green with a centre was a ring.

### Iterations 3 and 4 (category-red-sm)

Renders: art-work/vector/swatch/fast3/ to fast6/.

- Tried: spears crossing low, heads up and apart (a V with short legs). At 48 px a wishbone; at 12 px still an x. Two subpaths of opposite winding also punched a hole at the crossing (nonzero fill); the second spear is now wound the same way.
- Tried: leaf-shaped heads (pointed at both ends) on a 45 degree crossing. The best spear at 48 px, but the heads vanish at 9 to 12 px and it is an X again.
- Kept: broad, near-triangular heads (about 3 units wide against 1.4-unit shafts), short butts and a round boss of radius 1.5 at the crossing. At 9 to 14 px it is top-heavy, two blades over a knot, and no longer a symmetric X. At 48 px it looks more like a bow than spears; that size never uses the `-sm` file.

### Iteration 4 (swatches, final)

Renders: shots/M9/VF1/it4-mini-*.png (Chrome's own colour-vision emulation through `Emulation.setEmulatedVisionDeficiency`, DPR 2), it4-commerce-cvd.png, it4-guild-cvd.png, it4-medium-rows.png, it4-medium-guilds.png, it4-destroy.png; the final app shots are in shots/M9/VF1/final/.

- Layout: on a mini card the chip is 0.9 by 1.2 em with the glyph at 0.9 em; on medium and larger it is 1.15 by 1.35 em with the glyph at 1 em. Several chips sit edge to edge on a mini card. A guild's paired coin and laurel now overlap by 0.4 em on mini cards (both numerals stay clear), which with the narrower chips fits Shipowners (two chips) inside its 54 px card. A probe of every mini and medium effect pill in /dev/gallery finds none overflowing.
- Destroy (Statue of Zeus, Circus Maximus): the red cross used to sit over the middle of the swatch, which would hide the glyph. It is now a small paper disc with a cinnabar cross on the swatch's bottom end corner, so mountain and anvil stay readable.
- Seen, greyscale (achromatopsia), deuteranopia and protanopia: the commerce row reads wonder, anvil, mountain, purse, spears; the guild row purse, mountain plus anvil, wonder, dome, star, spears. Brown, green and red are three different shapes (mountain, star, blades over a knot) where they used to be three equal olive or grey bars. At DPR 1 on a desktop mini card (glyph about 8 px) the mountain, anvil, purse and dome still read; the star is a plus-shaped blob and the spears an x-shaped blob, which still differ from each other and from the mountain.
- Acceptance: the must issue is fixed. Lint is clean for the set (22 files) and the whole sprite is 25635 bytes raw and 5246 gzipped.

### Self blind check (author-read, not blind), M9.VF1

Reader: the author of the `-sm` glyphs, who saw every icon labelled, so this is a smoke test, not a blind reading. art-work/vector/blind/ui-category-2026-10-10T17-17-30-303Z/: 12 px DPR 2 15/15 (the 12 px sheet now draws the category `-sm` files), 16 px DPR 1 15/15. A fresh reader must take the next round.

### Family check, M9.VF1

art-work/vector/family/family@2x.png: the md files of this set are unchanged; the family sheet shows md files only, so it does not show the new `-sm` glyphs. In the specimen peep (ui-category/iter-vf1-2/peep-light@1x.png) the `-sm` glyphs at 12 and 14 px have the weight of the resource and science `-sm` files beside them.

## M9.VG: the family critic's fixes (2026-10-10)

Slice M9.VG took the "should" items of the M9b family review (art-work/vector/review/family-r1.md) that belong to icon files: in this set crest-moon, wonder, category-red, category-grey, token-frame, crest-sun, vp-laurel, badge-check and badge-chain; in the other sets price-tag (resources-economy-military log) and chain-horseshoe (chain log). The caller-colour and token items of the review (the `-ink` tokens for yellow, grey, turquoise and gold glyphs on paper, and brown, purple and pomegranate on lapis) are not icon files and were left alone.

### Iteration vg-0: candidates (fast renders)

Renders: art-work/vector/vg/ (s1 to s12 contact sheets made with the ui-category sheet helper: 12, 15, 16 and 24 px enlarged 600% with point filtering, 96 px, and a 16 px silhouette with `--i2` unset; candidates in cand/).

- crest-moon. Critique (review): a crescent cradling a star is a national-flag and religious emblem; art/style.md rules it out. Candidates: a fat crescent whose inner edge is a face in profile with an eye knockout (moon-a, moon-b), the same thinner (moon-e, moon-f, moon-g), and a crescent with a plain disc in its lap (moon-c). The disc reads as crescent-and-dot, close to the same emblem family, so it was dropped. The thin profiles read best at 96 px, the fat one best at 12 px.
- wonder. Critique (review): minarets beside an iwan read as a mosque front at 24 px and as "lnl" at 12 px. Candidates: a stepped gate with flat wings (wonder-a), the same with a cornice and tapered pylon wings (wonder-b), a castle-like gate (wonder-c) and a stepped pyramid with a door (wonder-d). The pyramid read as category-brown's mountain at 12 px; wonder-c as chain-tower's cousin. wonder-b reads as a monumental gateway at every size and kept the pointed arch.
- category-red. Critique (review and peep): the lightest glyph of the family, a bare X below 16 px. Eighteen variants of shaft, head and boss (red-a to red-u). Broad lozenge or trapezoid heads near the crossing made a butterfly or a bow tie at 24 px; round heads read as spades. Kept for the specimen: 3-unit shafts, heads 7 units wide with an ogee point, short butts, a boss of radius 3, moved down half a unit (red-r).
- category-grey. Critique (review): the horn is a sliver, the outline an I-beam or spool. The `-sm` idea at md size (grey-a), a refined anvil with a curved waist (grey-b, grey-d) and a squarer one (grey-c). grey-d (horn from x 2 to the waist, 5 units deep at the face, a 4-unit waist under the right half) is plainly lopsided from 12 px.
- token-frame. Critique (review): a plain ring at 12 to 14 px, a zero beside a number; bead gaps 0.14 units and a 1.5-unit band. Candidates: small beads on a band (frame-a), lobes clipped by a round hole of radius 6.5 with a band to 8.5 (frame-b, frame-e: the lobes stand only 1.5 units proud and the ring stays a ring), lobes of radius 3 on radius 7.5 (frame-f: deep lobes, but they cross the live area), and lobes of radius 3 on radius 7 over a band from 6 to 8 (frame-d: 2-unit lobes, 2-unit band, gaps over 1.5). frame-d kept; the inner radius drops from 6.5 to 6 (spec corrected).
- crest-sun. Critique (review): a fuzzball at the track size; cheerful at 60 px. Faces: two round eyes and a flat mouth (sun-a, reads as a robot at 96 px), closed-eye arcs and a soft mouth (sun-b, sun-f: serene at 96 px, a smudge at 16 px), two round eyes and no mouth (sun-d), pill eyes (sun-e: a mask). sun-d kept: calm at 48 and 96 px, two clear dots at 16 px. For `-sm`: eight thin rays on a disc of 3 (sunsm-a), broad rays (sunsm-b, a blob), a disc of 3.5 with stubby rays (sunsm-c), six rays (sunsm-6: a hexagon; sunsm-6b is a hexagram, ruled out as a religious emblem).
- vp-laurel. Critique (review): eight small leaves alias into a speckled ring at 12 px. The generator of iteration 0 with three leaves per side splayed them out like petals and past the viewBox (laurel-a to laurel-h). Hand-placed leaves (laurel3.py and laurel4.py: base and tip points per leaf, vesica radius from length and width) with three leaves per side along a 7.5-radius stem: laurel-n (leaves 6 to 7 units long, 4 wide) is a wreath at 96 px and a clean U with two bumps a side at 12 px.
- badge-check. Critique (review): a line-weight check. A uniform 4.2-unit check (check-b), a hairline-tipped wedge (check-a, check-c: the tip vanishes at 12 px), and a heavy check whose long arm tapers to a 2.5-unit wedge end (check-d). check-d kept.
- badge-chain. Critique (review): outlined links. Heavier stadium links with 3-unit rims and 3-unit slots (chain-a) against a ring with an edge-on bar (chain-b, a key or a male sign at 12 px). chain-a kept.

### Iteration vg-1 (2026-10-10)

Renders: art-work/vector/ui-category/iter-vg-1/ (light@1x, light@2x, dark@1x, dark@2x, grey-light@1x, deuteranopia-light@1x, protanopia-light@1x, tritanopia-light@1x, the three peeps, and crops/red-2x.png, crests-2x.png, frame-2x.png).
Lint: clean (23 files, 0 errors, 0 warnings).

- category-red: no longer an X at 12 to 16 px: two heads over a knot. At 48 px the 7-unit heads still look like a bow tie. Changed (vg-2): heads 6 units wide.
- crest-moon (moon-f): a crescent with a face at 24 px and up, but at 12 and 14 px the thinnest and palest glyph of the row. Changed (vg-2): the profile moves right by one unit (fatter body, moon-h) and the whole glyph by one unit, since the crescent's mass sat left of centre.
- crest-sun: the specimen draws crest-sun-sm at 12 and 14 px. At 16 px and up the md sun shows two eyes, calm at 48 px.
- wonder: a gateway at every size; no "lnl", no minarets. Dense at 12 px but its own shape.
- token-frame: in the strip chip ("2 · 3") it reads as a scalloped medallion, not a zero; lobes visible from 14 px, a bumpy ring at 12 px.
- category-grey, vp-laurel, badge-check, badge-chain: as in the candidates; no change.

### Iteration vg-2 (2026-10-10)

Renders: art-work/vector/ui-category/iter-vg-2/ (the same set of PNGs).
Lint: clean.

- category-red: crossed spears with leaf heads at 48 px; at 12 to 16 px top-heavy with a boss, no longer the lightest glyph of the family.
- crest-moon: holds its weight beside the sun at 12 px; profile and eye from 16 px; distinct from chain-moon (fatter, a profile, an eye).
- Greyscale peep (peep-grey-light@1x.png): crest-sun-sm at 12 and 14 px (a disc of 3 with eight thin rays) is the same spiky star as category-green there. Changed (vg-3): crest-sun-sm becomes the disc of 3.5 with stub rays (sunsm-c).

### Iteration vg-3 (2026-10-10): final

Renders: art-work/vector/ui-category/iter-vg-3/: light@1x.png, light@2x.png, dark@1x.png, dark@2x.png, grey-light@1x.png, deuteranopia-light@1x.png, protanopia-light@1x.png, tritanopia-light@1x.png, peep-light@1x.png, peep-light@2x.png, peep-grey-light@1x.png, crops/badges-cvd.png (the badge rows under greyscale and the three emulations), crops-cvd4.png.
Lint: clean (23 files, 0 errors, 0 warnings).

- crest-sun-sm: a round sun with a bumpy edge at 12 and 14 px, unlike category-green's thin star and unlike the crescent, in colour and greyscale.
- Every column looked at: A (paper, silhouette with `--i2` unset: every redrawn icon reads without its detail tone; wonder's arch becomes a hole), B (night: parity with paper), C (the seven bands at 14 px: every glyph separates from every band), D (mini card, strip chip, badges at 10 and 14 px, track cell).
- Badges in greyscale and under deuteranopia, protanopia and tritanopia: link, check and lock stay three different shapes, the check now as heavy as the lock.

Acceptance against STYLE.md section 8 for this set, item by item:

- Category glyphs, identifiable from the glyph alone in greyscale at card size: met (peep-grey-light@1x.png, grey-light@1x.png, and the app's full tier). The anvil is lopsided and the spears are no longer an X.
- Badge glyphs, the five states apart in greyscale and under each emulation at the mini size: met in the specimen (crops/badges-cvd.png) and in the app's badge section (link, check, coin and number, arrows and number, lock and number; shots below).
- Crests, shape-distinct: met; a round radiant disc against a crescent with a profile, in every pass and at every size.
- vp-laurel (resources class): lint clean, budget met, distinct in greyscale, legible on all seven bands, no speckle in the peep (a U with two bumps a side at 12 px DPR 1, leaves from 14 px), light and dark parity. Blind: below, author-read.
- All: lint clean, budgets met, family check looked at (below).

Spec corrections (art/vector/specs/ui-category.md), each with its reason in the spec: category-grey (a long, heavy horn), category-red (the construction), badge-check and badge-chain (the weights), crest-moon (no star: a face in profile), crest-sun (a calmer face and the new `-sm`), wonder (no towers: a pylon gateway), token-frame (eight lobes, inner radius 6 instead of 6.5).

### Self blind check, M9.VG (author-read, not blind)

Reader: the author of these redraws, who saw every icon labelled; a smoke test only. The answers were written before scoring; only the scorer read the key.

- Coloured sheets, art-work/vector/blind/ui-category-2026-10-10T17-52-11-332Z/: 12 px DPR 2 15/15, 16 px DPR 1 15/15.
- Plain ink (`--mono`), art-work/vector/blind/ui-category-2026-10-10T17-52-37-364Z/: 12 px DPR 2 15/15, 16 px DPR 1 15/15.
- Per icon: every icon named correctly on all four sheets (the 12 px sheets draw crest-sun-sm and the category `-sm` files). The pair to watch is still crest-sun against category-green on the mono sheets: a solid bumpy disc against a thin star.

### Family check, M9.VG

art-work/vector/family/family@2x.png. category-red is no longer the lightest glyph of the family; badge-check and badge-chain now sit with the solid glyphs instead of with the line icons; the crescent has the weight of the sun beside it; the wonder is a single solid mass like the tower and the temple front. token-frame, an eight-lobed ring, is a cousin of ornament-rosette (eight petals and a centre); they never share a context, the rosette being an ornament. No outlier.

### The app, M9.VG

Shots in shots/M9/VG/: the /dev/gallery icon grid, Affordability badges, Tier band, Tier mini, Tier full, the military track section and the progress tokens, at 360x740 (DPR 2) and 1280x800 (DPR 1), light and dark (the tokens at 360 light and 1280 dark), with enlarged crops in crops/.

- Badges: in both themes and both viewports the chain, free, own, trade and short states read; the heavier check and links hold up at the desktop mini size.
- Full tier: the anvil by the card name and in the placeholder art is plainly an anvil; the spears on Stable read as crossed spears.
- Mini and band tiers: price-tag (resources set) is now a tag the size of the shield beside its coin.
- Military track: the moon crest is a crescent face. The track still draws the md crest-sun, because MilitaryTrack.svelte names `crest-sun` directly; at 1280 DPR 1 it is still a fuzzy disc. Switching the track (and any crest drawn at 14 px or less) to `crest-sun-sm` is a component change outside this slice, filed as a request.
- Progress tokens: the scalloped frame holds the initial at 36 and 26 px with a pixel or more to spare around wide letters (M) at 36 px.

## Open points (M9.VG)

- MilitaryTrack (and other callers at 14 px or less) should use `crest-sun-sm` for the Sun crest.
- token-frame's inner radius is 6; if a future token initial is wider than M at 36 px, the component's letter size is the knob to turn.
