# science: iteration log

## Status

- science-globe: done (iteration 2)
- science-scales: done (iteration 2)
- science-sundial: done (iteration 2)
- science-mortar: done (iteration 2)
- science-set-square: done (iteration 2)
- science-quill: done (iteration 2)
- science-wheel: done (iteration 2)

Every md and `-sm` file is lint clean and between 204 and 418 bytes. The sprite is 20730 bytes raw and 3661 gzipped.

## Iteration 0 (2026-10-10): fast-render sketches before the first specimen

Renders: art-work/vector/science/fast-1/ to fast-3/ (librsvg strips from `fast.mjs`, 12 to 96 px, paper and night).

The M5 placeholders were replaced wholesale. Every icon was first drawn from a one-sentence idea:

- science-globe: a ball on a stem and a flat foot, cut by the front arc of a tilted equator ring.
- science-scales: a beam on a post with a finial and a foot, two solid bowls hanging from V-shaped cords.
- science-sundial: a dial plate on a slab with a triangular gnomon standing on it.
- science-mortar: a lipped bowl on a foot with a pestle leaning out to the top right.
- science-set-square: a right triangle, right angle at the bottom left, with a triangular knockout.
- science-quill: a broad feather on the diagonal, notched vane, the shaft ending in a nib point at the bottom left.
- science-wheel: a fat rim, six spokes, a solid hub.

What the strips showed, and what changed before the first specimen:

- science-sundial, first idea (a half disc on its straight edge, a fin rising from the top, two hour lines knocked out at 45 degrees): at every size it read as a spiked helmet with two angry eyes. A V-shaped fan of hour lines instead of the two separate lines read as a grin. A third try (a flatter half ellipse, a thin needle gnomon, a shadow slash) lost the needle below 24 px and the slashes looked random. Changed to a dial plate drawn in perspective: an elliptical face, a separate slab edge under it (a 1.5 unit knockout between them), a right-triangle gnomon standing on the face, and one shadow wedge knocked out of the face. It reads as a sundial at 16 px and up, and nothing else in the set has a flat, wide base with a fin. This departs from the spec's "flat half disc lying on its straight edge" silhouette; see the spec correction note below.
- science-quill: the first notch in the vane was lost to rounding. Redrawn through an outline generator with two explicit notches on the broad (upper left) side.
- science-scales-sm: the first version (single straight cords, small bowls) read as "TT" at 12 px. Changed the cords to V shapes.
- science-globe: the ball arcs used a radius that did not match the chord, so the ball bulged 1 unit over the live area. The arcs now use a smaller radius that the renderer scales to the exact half chord, so the ball is a true circle on the chord's midpoint.

Spec correction (science-sundial silhouette): the spec asked for "a flat half disc lying on its straight edge, with a raised triangular gnomon". Drawn that way the icon reads as a helmet (see above), so the dial is drawn as a plate in perspective (an elliptical face on a slab edge) with the gnomon standing on it. The spec file now says so.

## Iteration 1 (2026-10-10)

Renders: art-work/vector/science/iter-1/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the three peeps).
Lint: clean (0 errors, 0 warnings).
Looked at: light@1x, dark@2x, grey-light@1x, deuteranopia-light@1x, peep-light@1x, peep-light@2x.

### science-globe

- Seen: ball, stem and foot read at every size, on paper and night, and on all seven bands. The equator band is faint at 12 px DPR 1 but the silhouette carries the icon. No change.

### science-scales

- Seen: md is clear from 16 px. The `-sm` at 12 px DPR 1 still reads as a "TT" with dots: the bowls are 3 px wide and the 1-unit cords blur to a light grey.
- Changed: `-sm` bowls become solid hanging drops (cord and bowl one filled shape), so each pan is a dark mass at 12 px.

### science-sundial

- Seen: reads as a sundial from 14 px. In the `-sm` at 12 px the gnomon is a 2-pixel nub; the slab edge is a soft line.
- Changed: `-sm` gnomon taller and wider.

### science-mortar

- Seen: bowl, lip and pestle clear at every size; the lip gap shows as a white bar on the cards. No change.

### science-set-square

- Seen: the strongest silhouette of the set; the knockout stays open at 12 px. No change.

### science-quill

- Seen: a feather from 16 px up, but at 12 px DPR 1 it is a thin grain; the notches vanish below 20 px.
- Changed: broader vane in both files, so it keeps a leaf mass at 12 px.

### science-wheel

- Seen: md is a clear spoked wheel. The `-sm` at 12 px reads as a crosshair: the hub (3 units) eats most of the four holes.
- Changed: `-sm` hub radius 1.5 to 1, so the four holes open up.

## Iteration 2 (2026-10-10)

Renders: art-work/vector/science/iter-2/ (light@1x, light@2x, dark@1x, dark@2x, grey, deuteranopia, protanopia and tritanopia in light and dark, and the three peeps).
Lint: clean (0 errors, 0 warnings).
Looked at: every PNG of the iteration (the colour-vision and grey passes as light and dark pairs side by side).

One fix came from the fast render before the shoot: the first `-sm` sundial drew its gnomon in the same path as the face, and where the two overlapped the winding cancelled and cut a hole in the face. The gnomon is now its own path.

### science-globe

- Seen: unchanged and still clear. The ball is 15 units across, a little under the spec's 16 to 17, because the stem and foot need the bottom 5 units; the globe still looks as heavy as its neighbours in the family sheet. No change.

### science-scales

- Seen: the solid drop-shaped pans make the `-sm` read as a balance at 12 px DPR 1 and DPR 2; in greyscale and in every colour-vision pass the pans stay dark masses.

### science-sundial

- Seen: the larger `-sm` gnomon reads at 12 px. The md shadow wedge is a single light fleck at 16 px; it helps at 24 px and up and does no harm below.

### science-mortar, science-set-square

- Seen: unchanged, clear at every size and on every band.

### science-quill

- Seen: the broader vane keeps a leaf mass at 12 px; from 16 px the notches show it is a feather, not a leaf.

### science-wheel

- Seen: with the smaller hub the `-sm` shows four open holes at 12 px and reads as a wheel, not a crosshair. md keeps six spokes.

### Pairs at 12 px

No two symbols share a silhouette: a ball on a foot (globe), a beam with two hanging pans (scales), a flat plate with a fin (sundial), a bowl under a bar with a stick (mortar), a solid right triangle (set-square), a diagonal leaf (quill) and the only open ring (wheel). The closest pair is sundial and mortar (both have a wide base and something rising from it), but the sundial's fin is a solid triangle above a convex plate and the mortar's pestle is a thin diagonal over a concave bowl. They stayed apart in every pass, greyscale included.

## Blind check (2026-10-10)

Reader: the author, not blind. I drew the icons and had seen them labelled in the specimen, so this only shows that nothing is confusable for someone who knows the set. No fresh agent was available to this slice. The sheets and names.txt in the two directories below can be handed to a fresh reader as they are; a new `blind.mjs make` run gives a new shuffle if the first reader has seen these.

The blind tool shoots fixed sheets: 12 px at DPR 2 with the `-sm` files, and 16 px at DPR 1 with the md files. The acceptance target for science is 14 px at DPR 2. The app uses the `-sm` files at 14 px and below, so the 12 px DPR 2 sheet tests the same files at a smaller size, which is the stricter test.

- On-card ink (green with paper detail), art-work/vector/blind/science-2026-10-10T15-08-09-212Z/: 12 px 7/7, 16 px 7/7.
- Plain ink, shape only (`--mono`), art-work/vector/blind/science-2026-10-10T15-08-34-018Z/: 12 px 7/7, 16 px 7/7.

Misses: none.

## Family check (2026-10-10)

art-work/vector/family/family@2x.png. Beside the other sets at 16 and 24 px, the science set matches the weight and optical size of the resource and military glyphs. It is a touch denser than the chain set, which suits symbols that must be matched at a glance. No outlier.

## The app (2026-10-10)

Shots in the run directory under shots/M9/V2/: the /dev/gallery sections Icons, Tier band, Tier mini, Tier medium, Tier full, Age structures and City, at 360x740 (DPR 2) and 1280x800 (DPR 1), light and dark.

- Mini cards: the `-sm` glyphs in the green band's paper inset (about 12 px) are all identifiable: wheel, set-square, mortar, quill, globe and sundial each read at a glance.
- Medium and full cards: the md glyphs sit well beside the chain chips and the coin.
- Science tracker (city panel): the unowned slots draw the md icons at 40% opacity in the muted ink with no `--i2`, so every detail is a hole. All seven stay recognisable. The compact tracker draws the md files at 12 px; they hold up, but the `-sm` files are drawn for that size (see the slice's requests).
