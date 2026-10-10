# ornaments-track-logo: iteration log

## Status

- ornament-rosette: done (iteration 1)
- ornament-cartouche: done (iteration 1)
- ornament-divider: done (iteration 2)
- ornament-border: done (iteration 1); see "Open points" for the tile seams at 16 px
- ornament-girih: done (iteration 1)
- track-cell, track-cell-2, track-cell-5, track-cell-10: done (iteration 1)
- capital-sun: done (iteration 2)
- capital-moon: done (iteration 2)
- logo-wordmark: done (iteration 1)
- favicon: done (iteration 1)

No file used more than 2 of its 6 iterations, and none is marked needs-human. Every file is lint clean (`node tools/art/vector/lint.mjs --set ornaments-track-logo`: 13 files, 0 errors, 0 warnings). Sizes: the capitals 446 and 465 bytes (budget 1228), the ornaments 236 to 819 bytes (budget 4096), the wordmark 7564 bytes (budget 8192), the favicon 735 bytes (budget 2048).

## Iteration 0 (2026-10-10): sketches and candidates before the first specimen

Renders: art-work/vector/ornaments-track-logo/fast-1/ to fast-3/ (librsvg strips from `fast.mjs`) and cand/ (candidate rows and sheets made with `rsvg-convert` and `magick`).

The idea of each piece, one sentence each:

- ornament-rosette: eight rounded petals around a disc, gild, with a paper eye.
- ornament-cartouche: a horizontal panel whose short sides are pointed lobes joined to the long sides by small concave cusps, a gild outline around a paper field.
- ornament-divider: a lozenge medallion with a paper eye, a bead on each side, and arms that taper to points.
- ornament-border: a border-image tile of 48 by 48 sliced at 16: an outer rule, an inner rule and a cord threaded through lozenges, with a lozenge knot in each corner.
- ornament-girih: an eight-fold girih drawn by Hankin's method (polygons in contact) on the 4.8.8 tiling of octagons and squares, lines only.
- track cells: a niche with a pointed arch, its zone told by one, two or three lozenges near the bottom.
- capitals: two squat crenellated towers and a lower wall with a pointed-arch gate, the player's crest figure above the wall.
- logo-wordmark: عجایب outlined from Noto Nastaliq Urdu, "Ajayeb" outlined from Fraunces 600 under it, centred, in gild.
- favicon: the isolated ع in gold on a lapis tile.

What the sketches showed, and what changed before the first specimen:

- Capitals, first sketch (a tall gate block with a small eight-pointed sun above it): at 16 px the crest vanished and the star sun read as a blob. The building was lowered (towers from y 8, wall top at 13) so the crest figure could grow to a disc of radius 3, and the eight-pointed sun was dropped: it also looked like category-green. The first crescent was a sliver (two nearly equal arcs); redrawn as a two-circle crescent with horns up.
- Track cells: two mark styles were compared in a 17-cell row at 16 px wide (cand/rows-tooth-1x.png, rows-loz-1x.png): teeth rising from the bottom frame blur into a saw edge; floating lozenges stay countable. Kept the lozenges and enlarged them to 6 by 8 units. The first arch (radius 17 from a springing line at y 15) read as a round tombstone; the springing line went down to y 23 with radius 22, which gives a clear pointed arch and turns the row into an arcade.
- Rosette: three candidates beside category-green and chain-sun at 10 to 48 px (cand/fast/ros.png): separate petals (kept), a scalloped disc (read as a seal), and petals around a stroked ring (read as a doughnut). Separate rounded petals are the only one that reads as a flower at 10 px, and rounded tips keep it apart from the sharp star and the separated rays of chain-sun.
- Girih: contact angles of 45, 56.25, 60, 67.5 and 72 degrees were compared as 2 by 2 repeats (cand/gcmp.png). 60 degrees gives the classic stars in a ring of kites; kept. The tile is drawn with the copies of the neighbouring tiles that cross its edges, all snapped to 0.5, so the repeat is exactly periodic.
- Favicon: the Regular ع was thin at 16 px; the Bold ع with half a unit of extra weight holds (work/favrows2.png). A stroke of that weight fails the feature rule (it is under 2.67 units on the 32 grid), so the weight is baked into the outline: the glyph is flattened, buffered by 0.5 with round joins (shapely) and simplified to 0.06 units.

## Iteration 1 (2026-10-10)

Renders: art-work/vector/ornaments-track-logo/iter-1/ (specimen: light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the three peeps) and iter-1/bench/ (an in-context bench made for this set: the track at 360 and at 1280, the capitals beside crest-sun, crest-moon, wonder and chain-tower, the rosette on cards and beside category-green and chain-sun, the cartouche in three slices at 54 to 220 px, the divider natural and stretched, the border as a border-image at 8, 12 and 16 px, the girih as a 3 by 3 seam test and as table texture at 6, 8 and 10%, the wordmark at 120, 200 and 240 px, the favicon at 16, 32 and 48 px on light and dark browser chrome; light and dark at DPR 1 and 2, and grey and the three colour-vision passes in both themes).
Lint: clean (0 errors, 0 warnings).
Looked at: every PNG of the specimen (in crops: column A, column B, column C, the mocks of column D) and of the bench (in crops of each section), and the peeps.

The specimen is built for glyphs: its mini card, strip chip, badge and track-cell mocks put each ornament where it would never go, so for this set those columns only prove that nothing breaks. The bench is the in-context check.

The bench's first shots had two bugs of the bench, not of the art: the stretched slice of a three-slice cartouche or divider took the SVG default width of 300 px, and the slices drew outside their viewBox windows because the bench set `overflow: visible`. Both fixed in the bench; the art did not change.

### ornament-rosette

- Seen: a gild flower from 10 px, its petals separate at 14 px and up; beside category-green (a sharp star) and chain-sun (a disc with separate pointed rays) it never matches either, in colour or grey. In the card corners at 10 to 16 px it is quieter than the shields in the band. No change.

### ornament-cartouche

- Seen: three slices join without a seam at 54, 90, 140 and 220 px wide; the pointed lobes read at every width, the 2-unit outline is about 1 px at 14 px tall. The paper field takes the band's icons well. No change.

### ornament-divider

- Seen: medallion, beads and arms read at 8 to 16 px tall, natural and stretched. The arms (3 units at the root) are a little heavy beside body text, heavier than a rule should be.
- Changed (iteration 2): the arms start at 2 units instead of 3.

### ornament-border

- Seen: the corners and edges meet; the rules and the cord run through the tile edges. At 8 px it is a fine dotted frame, at 12 and 16 px a clear manuscript border. At 16 px with `round`, the edge tiles land on fractional pixels and faint hairline seams show across the rules (see "Open points"). No change to the art.

### ornament-girih

- Seen: the 3 by 3 repeat at 128 px per tile has no seam in either theme; at 6 to 10% on the table it is a faint texture that a card on top ignores. No change.

### track-cell, track-cell-2, track-cell-5, track-cell-10

- Seen: at 360 px (17 cells and the 2 capitals in 17 px slots), the arches are countable at DPR 1 and 2, and the 1, 2 and 3 lozenges change exactly at the zone boundaries, so the boundaries show without the numbers. At DPR 1 the lozenges are 3 px dots, still countable. In dark, the frame (`--rule-strong`) on the sunken field reads as clearly as in light. Greyscale and the colour-vision passes change nothing, since the zones are told by count, not colour. No change.

### capital-sun, capital-moon

- Seen: a fortified gate under a disc or a crescent from 16 px, in both themes and every pass; beside wonder (a dome-topped portal) and chain-tower (one tall tower) they do not match. At 16 px the crescent is the weaker crest: a thin cup of 2 pixels.
- Changed (iteration 2): the crescent's inner arc flattened (radius 3 to 3.5) so the crescent is 3.8 units thick at the bottom instead of 2.5; the sun disc grew from radius 3 to 3.5 and moved up half a unit to keep a 1.5 unit gap to the towers.

### logo-wordmark

- Seen: at 200 and 240 px elegant in both themes, the gild Latin line sits well under the ink word; at 120 px still clear. The letters match a plain `hb-view` render of the same text exactly (see "Logo provenance"). No change.

### favicon

- Seen: the ع is recognisable at 32 and 48 px and identifiable at 16 px, on the light (#dee1e6, white) and dark (#202124) tab strips; the lapis tile separates from both. No change.

## Iteration 2 (2026-10-10)

Renders: art-work/vector/ornaments-track-logo/iter-2/ and iter-2/bench/ (the same shots as iteration 1).
Lint: clean (0 errors, 0 warnings).
Looked at: column A and B crops of the specimen in both themes, the bench's track, capitals and divider sections in light and dark.

### capital-sun, capital-moon

- Seen: the thicker crescent now reads at 16 px as a crescent, not a dash; the larger sun balances it. Sun against moon stays distinct by shape in greyscale at every size.

### ornament-divider

- Seen: the lighter arms sit under body text like a rule, the medallion still carries the ornament.

## Acceptance (STYLE.md section 8)

- Track: at 360 px wide, the cells are countable (17 arches, at DPR 1 and 2), the zone boundaries are visible (the mark count changes at 1, 3 and 6 from the centre), and the capitals are identifiable (a gate under a disc, a gate under a crescent; distinct in greyscale and every colour-vision pass). Pass. Shots: iter-2/bench/light@1x.png, light@2x.png, dark@1x.png, dark@2x.png, and the app gallery shots below.
- Logo: the favicon is recognisable at 32 px and identifiable at 16 px on light and dark browser chrome; the wordmark is elegant at 200 px in both themes and its letters are correct. Pass, by the author's eye; a Persian reader should still confirm the wordmark reads عجایب without hesitation.
- All: lint clean for all 13 files, budgets met (sizes in "Status"), family check passed (below).

## Blind check (2026-10-10)

Reader: the author, not blind. I drew the set and saw it labelled, so this is a smoke test only. A fresh reader can take the key-free copy in art-work/vector/reader-V5-ornaments-track-logo/ (its key is in art-work/vector/blind/ornaments-track-logo-2026-10-10T16-41-14-086Z/).

Score (author-read), art-work/vector/blind/ornaments-track-logo-2026-10-10T16-40-34-713Z/: 12 px 13/13, 16 px 13/13; misses: none. Per file, both sheets: ornament-rosette ok, ornament-cartouche ok, ornament-divider ok, ornament-border ok, ornament-girih ok, track-cell ok, track-cell-2 ok, track-cell-5 ok, track-cell-10 ok, capital-sun ok, capital-moon ok, logo-wordmark ok, favicon ok.

At these sizes the ornaments are recognised by shape class rather than read: the divider is a dash, the girih a faint star, the wordmark a squiggle. The real question for this set is the capitals and the three cell marks, which are the files that carry meaning.

## Family check (2026-10-10)

art-work/vector/family/family@2x.png. The capitals match the solid weight and optical size of chain-tower and the resource glyphs, and their towers use the same 2-unit merlons. The rosette has the density of chain-sun. The track cells are outlines, lighter than any glyph, as an ornament should be; the girih and divider are lighter still. No outlier.

## The app (2026-10-10)

Shots in the run directory under shots/M9/V5/: gallery-360-light.png, gallery-360-dark.png (360x740, DPR 2), gallery-1280-light.png, gallery-1280-dark.png (1280x800, DPR 1), the section "Ornaments, track and logo" of /dev/gallery, served by the Go server from the test build. The probes report no horizontal scroll and no console errors.

- The track row at phone width: 17 cells and the two capitals in 340 px, the arcade countable and the zones visible in both themes.
- The cartouche in three inline slices and the divider with stretched arms render as on the bench.
- The border as a CSS mask over `--gild` (`-webkit-mask-box-image`, which Chrome supports; `mask-border` for the others) and the girih as a repeating mask over `--rule` work in both themes, with no colour baked into a URL.
- The wordmark at 200 px and the favicon at 16, 32 and 48 px look as on the bench.

## Logo provenance

- Font: Noto Nastaliq Urdu 4.000, from the upstream release NotoNastaliqUrdu-v4.000 (github.com/notofonts/nastaliq, published 2024-07-17; zip sha256 0725a7a9dfd450961b18227445cc9a5d21624d68b1f88fa52e4c6a0fe7576728), files `NotoNastaliqUrdu/unhinted/ttf/NotoNastaliqUrdu-Regular.ttf` and `-Bold.ttf`. Name table: "Copyright 2022 The Noto Project Authors (https://github.com/notofonts/nastaliq)", version 4.000, license SIL Open Font License 1.1, no Reserved Font Name. The release ships its OFL.txt (OFL 1.1). The font itself is not committed and does not ship; only outlines do.
- Latin: Fraunces SemiBold 1.000 (`web/font-sources/fraunces-600.ttf`, OFL 1.1).
- Commands: `hb-view --font-file=NotoNastaliqUrdu-Regular.ttf --font-size=200 --output-format=svg --output-file=fa-regular.svg "عجایب"` and `hb-view --font-file=fraunces-600.ttf --font-size=200 --output-format=svg "Ajayeb"`; each `<use>` resolved into its glyph path at its position, the Latin glyphs spread by 14 font units each, both scaled to the ink box (the Persian 172 units wide, the Latin 80, a 10-unit gap) in a viewBox 200 units wide, and written as relative path data rounded to 0.1. The favicon's ع is the same pipeline on the Bold face, then buffered by 0.5 units (see iteration 0). The scripts are in the run directory (art-work/vector/ornaments-track-logo/work/).
- Letter check: a plain `hb-view` render at 400 px (font/ref-regular-400t.png) and the wordmark at 900 px (work/wm-check-900.png) are the same shapes. Read right to left: initial ع (the open hook at the top right), joined to medial ج (the sweep down to the left, one dot below it), joined to final ا (the tall upright), then initial ی (the tooth left of the alef, two dots below it), joined to final ب (the long boat to the left, one dot below). `hb-shape` gives the same sequence: ain-ar.init, hah-ar.medi, alef-ar.fina, behDotless-ar.init with two dots below, behDotless-ar.fina with one dot below. The macOS system copy (`/System/Library/Fonts/NotoNastaliq.ttc`) shapes the word to the same glyph sequence.
- Never used: IranNastaliq, or the lettering of the official game's logo.
- Colours of the favicon (raw, since it is served without the stylesheet): `#224fa7` for the tile, the sRGB value of `--lapis` (oklch 0.45 0.15 262), and `#ddb049` for the letter, the sRGB value of `--gold` (oklch 0.78 0.13 85).

## Spec corrections (M9.V5)

Each is written into specs/ornaments-track-logo.md:

- track-cell became four files, one per zone (track-cell, track-cell-2, track-cell-5, track-cell-10), because one SVG cannot switch its marks; the three new ones are in sets.json. The marks are lozenges, not notches.
- The capitals' sun is a plain disc: rays fall under the minimum feature at this size.
- The favicon's letter is the Bold ع thickened by half a unit and left unsnapped (`"snap": false` in sets.json).
- Paint and use: every file uses the two-tone convention instead of naming tokens (only the wordmark's gild accent names `--gild`), and the spec now says how each piece is meant to be used (inline, three slices, or a CSS mask).

## Open points

- ornament-border at 16 px with `border-image-repeat: round`: each edge tile is scaled to a fractional pixel width, and the browser antialiases each tile's ends, so faint hairline seams cross the rules where tiles meet. They are not in the art (the 3 by 3 repeat of the tile is exact). The component can avoid them by sizing the panel's edges to whole multiples of the border width, or by using `repeat` with a fixed panel size.
- A light-theme sun capital drawn in `--sun` on a paper panel is low in contrast (turquoise on ivory); the gallery uses `--sun-ink` and `--moon-ink`, and MilitaryTrack should do the same.
- A Persian reader has not yet confirmed the wordmark, and a fresh agent has not yet taken the blind test.
