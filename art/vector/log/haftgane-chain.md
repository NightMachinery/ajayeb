# haftgane-chain: iteration log

## Status

- chain-awning: done (iteration 1)
- chain-camel: done (iteration 1)
- chain-hammer: done (iteration 1)
- chain-lighthouse: done (iteration 2)
- chain-lightning: done (iteration 1)
- chain-planet: done (iteration 4)
- chain-saw: done (iteration 3)
- chain-scroll: done (iteration 1)
- chain-star: done (iteration 1)
- chain-torch: done (iteration 2)
- The 14 reused drawings (chain-barrel, chain-book, chain-column, chain-drop, chain-helmet, chain-horseshoe, chain-lamp, chain-lyre, chain-mask, chain-target, chain-tower, and science-mortar, science-scales and science-quill as chain-mortar, chain-scales and chain-quill): accepted at chip size unchanged (inspection below). Nothing was copied; `sets.json` references the Duel files.

No new drawing used more than 4 of its 6 iterations, and none is marked needs-human. No `-sm` file was needed. The ten new files are lint clean and between 146 and 468 bytes, 3088 bytes together, in `web/src/lib/games/haftgane/icons/svg/`, outside Duel's bundle. Duel's icon source set is unchanged (27252 bytes raw, 5638 gzipped).

## Reused drawings at chip size (2026-10-23)

Renders: art-work/vector/haftgane-chain/scratch/reused-a.png, reused-b.png and reused-sci.png (each glyph at 12 px in the gold "in" chip and the paper "out" chip, at 16 px on paper, as a 16 px silhouette with `--i2` unset, at 16 px on night and at 24 px; DPR 1 enlarged 4x with point filtering beside DPR 2 enlarged 2x; drawn with librsvg).

- All fourteen read in both chips at 12 px, in silhouette and on night. The eleven chain drawings are as Duel's chain log left them.
- science-mortar, science-scales and science-quill were drawn for the green effect band, not for a chip; in the chip they read as well as the chain drawings. The md file is the one to use, as for every chain glyph. science-scales-sm is a little cleaner at 12 px DPR 1 (the md pans are triangles that blur), so it stays the fallback if the app's chip reading asks for it.
- Pairs that become neighbours only in Haftgane: mortar and lamp (both low bowls with something rising to the right; the mortar's flat rim bar and diagonal pestle against the lamp's whale body and lens flame keep them apart), quill against the new hammer and torch (a feather against a block on a stick and an upright flame), column against tower and the new lighthouse. Each pair differs in silhouette at 12 px.

## Iteration 0 (2026-10-23): fast-render sketches before the first specimen

Renders: art-work/vector/haftgane-chain/iter-0/ (new-a.png, new-b.png and cand-1.png from the chip sheet described above; big-all.png and grid64.png at 48 and 64 px) and candidates in art-work/vector/haftgane-chain/cand/.

The idea of each icon, one sentence each:

- chain-awning: a canopy wider at the bottom, a scalloped lower edge, two of five stripes knocked out and filled with the detail tone, on two posts.
- chain-camel: a dromedary facing right, one hump, a neck rising to a small head, front and back legs as one thick leg each.
- chain-hammer: a block head and a short handle, drawn upright and turned 45 degrees.
- chain-lighthouse: three stepped tiers under a lantern with a lit window (detail tone) and a pointed cap.
- chain-lightning: one thick zigzag bolt, upper right to a point at the lower left.
- chain-planet: a disc and a tilted ring.
- chain-saw: a straight blade with three teeth and a handle with a grip opening.
- chain-scroll: an upright sheet between two horizontal rolls.
- chain-star: a broad five-point star, inner radius about half the outer, main tone only.
- chain-torch: a tapering handle, a cup, and a flame whose tip leans right, with an inner flame in the detail tone.

What the sheets showed, and what changed before the first specimen:

- chain-star, chain-lightning, chain-camel, chain-hammer: read at every size from the first sketch.
- chain-awning: without stripes the canopy on posts was a table. Stripes added (knocked out, filled with the detail tone, so the canopy stays striped with `--i2` unset). Two candidates, posts alone and posts with a counter bar; the counter turns it into a market stall and away from a temple front with columns. Kept the counter.
- chain-scroll: rolls of the same width above and below a sheet made a capital "I", too close to chain-column. Small spiral knockouts at the roll ends vanished at 16 px. Redrawn with open curls: each roll is a hook that curls back on itself, the top one at the left, the bottom one at the right. The curls show as gaps down to 12 px.
- chain-saw: a level saw with a rectangular handle and a slot read as a key. Tilted 30 degrees with a tapering blade it read as a saw from 16 px.
- chain-planet: the first ring passed in front of the disc with a clipped gap; the clip was too thick and the disc came out as a squashed lens. Replaced by a disc with an open stroked ring whose ends tuck behind it.
- chain-torch: the flame too small and the handle too thin; both enlarged.
- chain-lighthouse: the tiers had 0.5-unit gaps between them (the minimum gap is 1.5); merged into one stepped silhouette. Light beams added as wedges from the lantern's sides.

## Iteration 1 (2026-10-23)

Renders: art-work/vector/haftgane-chain/iter-1/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, the three peeps, crops/peep-new.png).
Lint: clean (24 files, 0 errors, 0 warnings).
Looked at: peep-light@1x (the whole strip and a crop of the ten new glyphs).

- chain-awning, chain-camel, chain-hammer, chain-lightning, chain-scroll, chain-star: read at 12, 14 and 16 px on paper and on night. No change.

### chain-lighthouse

- Seen: at 12 and 14 px the stepped tiers under the cap and the two beams at shoulder height make a robed figure with its arms held out.
- Changed (iteration 2): three candidates (cand/lh-a, lh-b, lh-c): a straight tapering tower with a gallery and a band (detail tone) and no beams; the same with beams; the stepped tower with long beams to the top corners (it read as a figure cheering). Kept lh-b: the narrow tapering tower with a band, a gallery, a lantern under a pointed cap and two wedge beams. The steps that remain are the gallery and the base.

### chain-planet

- Seen: at 12 px DPR 1 the disc and its ring are a lumpy ellipse.
- Changed (iteration 2): three candidates; kept a disc with a ring drawn as one open stroke that sweeps out on both sides (cand/pl-c).

### chain-torch

- Seen: at 12 px a thin stick with a blob, closer to a match than a torch.
- Changed (iteration 2): the flame widened to 10 units and lengthened, the cup widened, the handle 4 units at the top and 2 at the foot.

## Iteration 2 (2026-10-23)

Renders: art-work/vector/haftgane-chain/iter-2/ (as iteration 1, plus crops/light1x-bottom.png: the specimen rows of the last fourteen icons with their in-context mocks).
Lint: one warning: chain-planet's ring reached 0 to 24.5, past the live area; pulled in, then clean.
Looked at: peep-light@1x (crop of the new glyphs), light@1x (rows and mocks of the last fourteen icons).

- Mocks: on the mini card the gold in chip in the cost row and the paper out chip on the band show the same symbol for all 24; the strip chip, the badge and the seven bands at 14 px all read, in light and dark.
- chain-lighthouse: a beacon tower with beams at every size; the figure is gone.
- chain-torch: a flame on a handle from 12 px.

### chain-saw

- Seen: at 12 and 14 px the tilted saw is still a key: the handle loop is the bow and the toothed blade the bit.
- Changed (iteration 3): three candidates (cand/saw-c, saw-d, saw-e): level saws whose blade tapers from a deep heel at the handle to a narrow tip, with three deep teeth, and the same tilted 20 degrees. Kept saw-d, the level saw with the deeper blade: a toothed wedge with a handle, no longer a key.

## Iteration 3 (2026-10-23)

Renders: art-work/vector/haftgane-chain/iter-3/ (as before, plus crops/dark-bottom.png and crops/grey-bottom.png).
Lint: clean.
Looked at: peep-light@1x (crop of the new glyphs), dark@1x (rows and mocks of the last fourteen icons), grey-light@1x.

- chain-saw: a saw at every size.
- Dark and greyscale: nothing changes; the set is ink on gold or paper, so it does not lean on colour.

### chain-planet

- Seen: pulling the ring inside the live area (iteration 2) shortened its ends, and at 12 px the planet is now a plain disc, at 12 px in the mini card's chip a coin.
- Changed (iteration 4): eight candidates (cand/pl-d to pl-k), a full stroked ellipse around a disc at several disc sizes, ring sizes and tilts. Rings tilted 20 to 30 degrees close into small loops that fill in at 12 px; a ring tilted 40 degrees uses the diagonal and keeps open loops at both ends inside the live area. Kept pl-h: a disc of radius 6 and a ring of 11.5 by 3 units turned 40 degrees, 2-unit stroke.

## Iteration 4 (2026-10-23): final

Renders: art-work/vector/haftgane-chain/iter-4/ (light@1x, light@2x, dark@1x, dark@2x, grey-light@1x, deuteranopia-light@1x, protanopia-light@1x, tritanopia-light@1x, peep-light@1x, peep-light@2x, peep-grey-light@1x, and crops/peep-both.png: the ten new glyphs at DPR 1 and 2).
Lint: clean (24 files, 0 errors, 0 warnings).
Looked at: crops/peep-both.png, light@1x.

- chain-planet: a ringed planet with both loops open at 12 px DPR 1 and clean at DPR 2.
- All ten read at 12, 14 and 16 px at DPR 1 and 2, on paper and on night, in both chips.

Silhouette families across the 24 (the spec's shared rule): tall and narrow are column (a capital "I" with flutes), tower (crenels), drop (a round bottom), lighthouse (beams and a cap), torch (a flame on a stick), scroll (curled rolls) and lightning (a zigzag); round are target (rings and an arrow), planet (a tilted ring), mask (a face) and helmet (a crested profile); wide and low are lamp, mortar, awning, camel, saw and scales; pointed or radial are star, quill and hammer. Inside each family the distinguishing feature holds at 12 px.

## Blind check (2026-10-23)

Reader: a fresh Opus subagent that had never seen the labelled icons, handed only the reader copy (the two sheets, names.txt and the answer template) and a one-line object gloss per candidate name ("chain-saw: a saw", "category-grey: an anvil"). It answered each tile on its own, not by elimination, and could answer "?".
Sheets: art-work/vector/blind/haftgane-chain-2026-10-23T07-30-08-660Z/ (key) and art-work/vector/blind-reader/haftgane-chain-mono-1/ (reader copy and answers). `--mono` (plain ink, shape only), so the neighbours' own colours and tiles give nothing away. 32 tiles per sheet: the 24 symbols and the set's 8 neighbours (chain-sword, chain-sun, chain-temple-facade, science-globe, category-yellow, category-grey, category-green, resource-papyrus), at 12 px DPR 2 (the neighbours that have one use their `-sm` file there) and 16 px DPR 1.
Score: 12 px 32/32, 16 px 32/32; misses: none. Each new drawing was named correctly against its confusable neighbours: star against category-green and chain-sun, planet against science-globe and chain-target, scroll against resource-papyrus and chain-column, awning against category-yellow and chain-temple-facade, hammer and saw against chain-sword and category-grey.

What the reader was unsure of: chain-scroll against resource-papyrus. It told the two drawings apart (the upright scroll with curled rolls, the level roll between two rollers) but said nothing in the names tells which object is the "scroll" and which the "papyrus roll", and that its split was a coin flip it got right. That is a naming doubt, not a drawing one: the two never meet in the same place (papyrus sits on resource tiles, the scroll in chain chips). The Haftgane science tablet, which the spec also names as the scroll's neighbour, was not drawn yet when this check ran, so it is not on the sheets.

## Family check (2026-10-23)

art-work/vector/haftgane-chain/family/family@2x.png: resources-economy-military, science, chain and haftgane-chain at 16 and 24 px on paper and night. The ten new glyphs have the solid manuscript weight of Duel's chain drawings, the 2-unit stroke where a part is a stroke (the planet's ring, like the lyre's arms) and the same corner treatment. Hammer, lightning and torch are the lightest of the new ten, no lighter than Duel's sword. No outlier.

## The app (2026-10-23)

Not applicable yet: nothing imports the Haftgane icon directory, and Haftgane has no screen that draws chain chips. M19.art-wiring wires them; its gallery shots are the check in real components. The specimen's mocks (the mini card's in and out chips, the strip chip, the badge and the seven bands) stand in until then.
