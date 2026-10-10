# ui-category: iteration log

## Status

- category-brown: done (iteration 1)
- category-grey: done (iteration 2, after candidates in iteration 0)
- category-blue: done (iteration 1)
- category-green: done (iteration 1, after candidates in iteration 0); spec corrected, see below
- category-yellow: done (iteration 1, after candidates in iteration 0)
- category-red: done (iteration 3)
- category-purple: done (iteration 1)
- badge-check: done (iteration 1)
- badge-chain: done (iteration 1)
- badge-lock: done (iteration 1)
- crest-sun: done (iteration 2, after candidates in iterations 0 and 1); spec corrected, see below
- crest-moon: done (iteration 2); spec corrected, see below
- vp-laurel: done (iteration 1, after a generated candidate round in iteration 0)
- wonder: done (iteration 2)
- token-frame: done (iteration 2)

No icon used more than 3 of its 6 iterations, and none is marked needs-human. Every file is lint clean and between 150 and 1008 bytes. The whole sprite is 22883 bytes raw and 4769 gzipped (limits 40960 and 10240). The blind check below was read by the author, so a fresh reader must still do the real one.

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
