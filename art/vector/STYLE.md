# Vector style: the rules for every hand-drawn icon and ornament

This file is the contract for the vector art of Ajayeb: the icons in the inline sprite (`web/src/lib/icons/svg/*.svg`), the ornaments, the military track parts and the logo. Every vector is hand-authored SVG, drawn by an Opus agent in the render, look and critique loop described at the end of this file. Codex is never used for vectors.

The per-set briefs are in [specs/](specs/), the iteration logs in [log/](log/), and the set membership (which files belong to which set, and how the specimen shows them in context) in [sets.json](sets.json). The tools are in [tools/art/vector/](../../tools/art/vector/) and are described in [tools/art/README.md](../../tools/art/README.md).

## 1. Files and names

- One icon per file. The file name is the icon id, which is also the id in `web/src/lib/icons/names.ts` and the symbol id `i-<id>` in the sprite. Never rename an id that `names.ts` lists: the components refer to them.
- A `-sm` file is the small-grid variant of the icon with the same stem (`resource-wood-sm` for `resource-wood`). The app uses it at 14 px and below.
- Sprite icons live in `web/src/lib/icons/svg/`. Ornaments and track parts that are not sprite icons live in `web/src/lib/ornaments/svg/`, the wordmark in `web/src/lib/brand/`, and the favicon at `web/static/favicon.svg` (see [specs/ornaments-track-logo.md](specs/ornaments-track-logo.md)).
- A new sprite icon needs three things at once: its file, its line in `names.ts`, and its entry in `sets.json`. `sprite.ts` refuses a file that `names.ts` does not list, and the reverse.

## 2. Grid and geometry

- **Canvas.** `viewBox="0 0 24 24"` for every md icon. The live area is 2 to 22 on both axes: nothing but a halo crosses it. The 2-unit margin keeps icons from touching the edge of a tile or chip.
- **Small grid.** `-sm` variants use `viewBox="0 0 12 12"`, with the live area 1 to 11. They are drawn again on their own grid, not scaled down: fewer parts, fatter parts.
- **Minimum feature.** 2 units on the md grid (1 device pixel at 12 px), 1 unit on the 12 grid (1 pixel at 12 px). That covers stroke widths, bar widths, dot diameters and the thinnest part of any shape.
- **Minimum gap.** 1.5 units on the md grid, 1 unit on the 12 grid, between any two inked parts, and for a knockout (a hole cut through the silhouette). Narrower gaps fill in at 12 to 16 px; the pixel peep shows it.
- **Snapping.** Every coordinate, radius, width and stroke width is a multiple of 0.5. On the md grid at 24 px that puts edges on half pixels at worst; at 12 and 16 px it keeps parts from smearing across two pixel columns. Curve control points snap too.
- **Transforms.** Only `translate`, `rotate` and `scale` (for mirroring), on a `<g>` or a shape, with snapped numbers. No `matrix` or `skew`. Prefer plain coordinates; use `rotate(45 12 12)` for radial repeats in stars and rosettes.
- **Optical size.** Shapes of different kinds must look equally big beside each other:
  - a circle or disc spans 18 to 20 units;
  - a square or block spans 16 to 17 units;
  - a tall or pointed shape (a shield, a drop, a figure) spans 19 to 20 units in height;
  - a wide, flat shape (a scroll, a bar) spans 19 to 20 units in width and at least 8 in height, so it does not shrink to a dash at 12 px.
- **Centre.** The visual centre sits at (12, 12). A shape whose mass sits low (a mortar, a lamp) moves up by half a unit to a unit so it does not look dropped.

## 3. Paint: the two-tone convention

Every icon is painted with two tones that the caller chooses:

- `fill="var(--i1, currentColor)"` (or `stroke=`): the main tone. It carries the silhouette, at least about 60% of the inked area. Without any caller setting, it is the text colour, so an icon in a sentence reads like text.
- `fill="var(--i2, transparent)"` (or `stroke=`): the detail tone, for one secondary layer: the inner face of a gem, the bands of a bundle, the rim of a coin. Unset, it is transparent, which turns the detail into a hole.
- `fill="none"` on any shape that is only stroked. A shape with no fill anywhere up its tree paints black (the SVG default) and ignores the theme.

Rules that follow from it:

- **Identifiable without `--i2`.** With `--i2` unset (transparent) the icon must still be recognisable; the specimen's silhouette column shows that case. The detail tone may help recognition, never carry it. This is what keeps icons legible in greyscale and for colour-blind players.
- **No raw colours.** No hex, `rgb()`, `oklch()` or named colours, and no other custom properties, in an icon. Colours come from the caller, which takes them from `tokens.css` (resource tiles, band colours, player colours). The only exceptions: the favicon, which is a standalone file served without the stylesheet, and the ornament files, which may name a token directly (for example `var(--gold)`).
- **Two tones, not three.** A third tone shows up as noise at 14 px. If an icon seems to need one, its idea is too complicated for the size.
- **How callers set the tones.** `color` sets `--i1` through `currentColor`; `--i2` is set on the element or an ancestor (`[--i2:var(--gold-deep)]` in a Tailwind class, `style:--i2=` in Svelte). The contexts in the specs say what each caller sets today.

### The halo

A halo is a stroke drawn under the fill in the colour of the surface behind the icon, so the icon stays separate from a busy or similar-coloured background:

```svg
<path d="..." fill="var(--i1, currentColor)" stroke="var(--i-halo, transparent)"
      stroke-width="3" stroke-linejoin="round" paint-order="stroke"/>
```

- Use it only on icons that sit **directly** on a patterned or saturated surface without a paper inset: category glyphs on a card band or on the placeholder art, crests on the military track, the pawn on its track cell. Icons inside the paper inset of the effect band, inside chips, tiles or badges do not need one.
- Put it on the outer silhouette only, 2 to 3 units wide on the md grid (1 to 1.5 on the 12 grid), with round joins. It may cross the live area margin; that is what the margin is for.
- The caller sets `--i-halo` to the surface colour (the band colour, the track cell colour). Unset, it falls back to transparent and costs nothing.
- `paint-order="stroke"` is required, on the element or a parent `<g>`, or the halo covers the fill.

## 4. Structure and budgets

- Allowed elements: `svg`, `g`, `path`, `circle`, `ellipse`, `rect`, `line`, `polyline`, `polygon`, `defs`, `clipPath`, `mask` and `use`. Ornaments may add one `linearGradient` or `radialGradient` with its `stop`s, used subtly (a gold sheen, never a shading that carries meaning).
- Forbidden: `<text>` (draw numerals and letters as outlines), `<image>` and any raster, filters, `<style>`, `<title>` and `<desc>` (names come from the i18n tables), `<symbol>` (sprite.ts wraps each file), `<pattern>` (card patterns are CSS), scripts, animation elements, editor attributes (`inkscape:`, `sodipodi:` and so on), `style` and `class` attributes (the sprite is inlined into the page, so page CSS would reach them), and comments in sprite icons (they ship in the sprite; notes go in the log).
- The root is `<svg xmlns="http://www.w3.org/2000/svg" viewBox="...">` with no `width` or `height`: icons are sized in em by CSS.
- Ids, when an icon needs them (a clip path, a mask, a gradient), start with the file name: `science-wheel-hub`. The sprite puts every symbol in one document, so a bare `id="a"` would collide.
- **Budgets.** 1.2 KB (1228 bytes) per sprite icon, md or sm. The whole sprite stays under 40 KB raw and 10 KB gzipped (it is about 20 KB raw and 3.3 KB gzipped with the M5 placeholders). Ornaments: 4 KB each. The wordmark: 8 KB. The favicon: 2 KB.
- Write compact SVG by hand: one line is fine, no XML declaration, no default attributes, short path data (relative commands, no trailing zeros). If you run SVGO, keep the viewBox (`removeViewBox: false`) and turn off colour conversion so `var()` survives; SVGO is not installed here and not required.

`node tools/art/vector/lint.mjs` checks every rule in this section and in sections 2 and 3 that a program can check: the viewBox, forbidden elements and attributes, the paint convention and raw colours, implicit black fills, the halo's paint order, snapping, the stroke and part minimums, the live area (a warning), ids and references, and the budgets. It cannot check gaps, optical size or meaning; the specimen and your eyes do that.

## 5. Visual language

The look is an original **illuminated-manuscript** idiom, drawn for screens: solid, jewel-like silhouettes with a single inner detail, in the spirit of the gilded marginal devices of Persian manuscripts. It must read first as a clear game symbol and only second as ornament.

- **Solid, not outline.** Icons are filled silhouettes with knockouts or a detail tone, not line drawings. Line icons vanish on coloured tiles at 12 px; solid shapes survive. Strokes appear only as parts of a shape (a staff, a rim), at 2 units on the md grid.
- **Stroke weight.** Where a part is a stroke, it is 2 units (md) or 1 unit (sm). One weight per icon. Lucide's 2 px outline icons (menu, settings, undo and so on) are UI chrome and are not part of this family.
- **Corners.** Outer corners rounded with a radius of 0.5 to 1 unit, inner corners sharp. Pointed tips (arches, leaves, drops) end in a point, not a round, which gives the set its manuscript flavour.
- **Characteristic shapes.** The ogee (an arch that comes to a point), the eight-pointed star made of two squares, the teardrop and leaf, scalloped edges, faceted gems. Use them where they fit the meaning; never at the cost of recognition.
- **Density.** At most three distinct inner details on an md icon, and one or two on an `-sm` icon. If the 16 px render looks busy, remove a detail rather than thinning it.
- **Two-tone balance.** The main tone is the body; the detail tone is a single secondary layer. Avoid scattering small `--i2` specks: they turn to noise.
- **Consistency across sets.** The family sheet (`family.mjs`) puts every set side by side at 16 and 24 px. Stroke weights, corner radii, the optical size of discs and blocks, and visual density must match across sets.
- **Not the official game.** Ajayeb never copies or traces the official iconography, card frames or logo ([docs/legal.md](../../docs/legal.md)). Draw from the metaphors in the specs, never from a picture of the physical game. Where a metaphor is common to both (a coin, a shield), our construction, proportions and framing must be our own: no circular medallion backgrounds behind resource symbols, no copies of the printed card-corner layout. When in doubt, choose another metaphor from the spec's list.

## 6. Contexts and sizes

Where icons live in the app today (`web/src/lib/components`), and what each place sets:

- **Resource tiles** (`ResourceTile.svelte`): a rounded square in the resource colour, the glyph in paper or ink, `--i2` a 55% mix of the two. On a mini card the tile is 1 em of a 10 px font and the glyph about 9 px (the `-sm` file); in the medium tier about 12 px.
- **Cost row and effect band** of a card: the band's glyphs sit on a paper inset (92% paper over the band colour and pattern), so they need no halo. Shields are drawn in the red card colour with `--i2` paper; science symbols in the green card colour with `--i2` paper.
- **Chain chips** (`ChainChip.svelte`): 1.3 em chips with the glyph at 0.95 em, so about 12 px on a mini card. "In" is a gold chip, "out" a paper chip with a gold-deep border; `--i2` is gold-deep.
- **Affordability badges** (`AffordBadge.svelte`): pills 1.5 em tall with the glyph at 1.05 em, so about 10.5 px on a mini card and 15 px on a sheet.
- **Strips and panels** (`PlayerStrip`, `CityPanel`, `ResourceChips`, `ScienceTracker`, `WonderRow`): 11 to 20 px, on the panel surface in both themes. The resource chips colour the glyph with the resource colour.
- **The military track** (`MilitaryTrack`, `LootingToken`): the pawn and looting glyphs at 10 to 12 px inside 16 to 24 px discs; crests at the ends.
- **Large uses:** the crest on the pass-the-device screen (60 px), the progress-token frame (26 to 36 px).

The specimen reproduces these (see section 8), with the colours each icon's `sets.json` entry gives.

## 7. The loop

Each icon set goes through this loop. One agent owns one set at a time.

1. **Read** the set's spec in [specs/](specs/), this file and the set's log. Never look at official game art.
2. **Draw** or edit the SVG files. Keep each icon's idea in one sentence before you draw it.
3. **Lint**: `node tools/art/vector/lint.mjs --set <set>`. Fix every error before rendering.
4. **Quick look** (no browser, milliseconds): `node tools/art/vector/fast.mjs <files>` renders each file at 12 to 96 px on paper and on night with both tones substituted, through librsvg. Use it for every small change.
5. **Specimen** (once per iteration batch, one Chrome): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/shoot.mjs --set <set> --iter <n>`. It renders the specimen in light and dark at DPR 1 and 2, then greyscale and the deuteranopia, protanopia and tritanopia emulations, and writes the set's lint report next to the PNGs.
6. **Pixel peep**: `node tools/art/vector/peep.mjs --set <set> --iter <n>` enlarges the 12 to 16 px strip 800% with point filtering, so you see the real pixels.
7. **Look at every PNG yourself** with the Read tool: `light@1x`, `light@2x`, `dark@1x`, `dark@2x`, the greyscale and the three colour-vision PNGs, and the peeps. Check against the spec: silhouette, must-differ-from, legibility at each size and on each band, light and dark parity, greyscale, blobbing in the peep, budgets.
8. **Write the critique and the changes** in `art/vector/log/<set>.md` (format in [log/README.md](log/README.md)), then go back to step 2.
9. **Stop** when the icon meets its acceptance criteria, or after **6 iterations** of that icon. Then keep the best candidate and mark it `needs-human` in the log, with what is still wrong.

Before a set is finished:

- **Blind check.** `python3 tools/heavy.py run --kind browser -- node tools/art/vector/blind.mjs make --set <set>` writes a shuffled, numbered, unlabelled sheet at 12 px (DPR 2) and another at 16 px (DPR 1), plus the candidate names, under the run directory. The key is in the same directory, in a file named `KEY-do-not-open-before-answering.json`. Read the sheets, write your answer for every tile in `answers.txt` **before** you open the key, then score it with `node tools/art/vector/blind.mjs score --dir <dir> --answers <dir>/answers.txt` and record the score honestly in the log. Better still, give the sheets and `names.txt` to a fresh agent that has never seen the labelled specimen; a reader who drew the icons, or saw them labelled, is not blind, and the log must say so.
- **Family check.** `python3 tools/heavy.py run --kind browser -- node tools/art/vector/family.mjs` montages every set at 16 and 24 px in both themes. Look for a stroke, size or density outlier.
- **The app.** Build the test bundle, serve it with the Go server, and screenshot `/dev/gallery` at 360x740 and 1280x800 in light and dark (see the chrome-cdp skill). The icons must work in the real components, not only in the specimen.

## 8. Acceptance per class

- **Resources, coin, shield, laurel:** blind identification 100% at 12 px DPR 2 and 16 px DPR 1; distinct in greyscale; legible on all 7 bands; no blobbing in the peep; light and dark parity.
- **Science (7):** blind identification 100% at 14 px DPR 2 (pairs are the core mechanic of science); no two symbols confusable at 12 px.
- **Chain (17):** blind identification 90% or better at 16 px DPR 1; the in chip and the out chip read as the same symbol; legible in the mini card's chip at 12 px.
- **Category glyphs:** each category identifiable from its glyph alone in greyscale at the size of a 54 px wide card.
- **Badge glyphs:** the five badge states (chain, free, own, trade, short) distinguishable in greyscale and under each colour-vision emulation at the mini size.
- **Track:** at 360 px wide, cells countable, the zone boundaries visible, capitals identifiable.
- **Logo:** recognisable at 32 px as the favicon and elegant at 200 px, in both themes.
- **All:** lint clean, budgets met, family check passed.

## 9. sets.json

`art/vector/sets.json` lists each set's files and how the specimen shows them. Per set: `title`, optional `rules` (lint overrides for the whole set: `viewBox` ("any" to skip the check), `budget` in bytes, `gradients`, `snap`, `tokens` (allow `var(--token)` paints), `rawColors`, `liveArea` (false for a full-bleed file such as the favicon)) and `overrides` (the same per id). Per icon:

- `id`: the md file's id; `sm`: the `-sm` id, when there is one; `file`: the path, when the file does not live in the sprite directory.
- `ink`: the colour of the icon on a theme surface (strips, panels, the specimen's columns A and B); `cardInk`: the colour on a card, when it differs (cards are paper in both themes); `i2`: the detail tone, when the app sets one (otherwise the specimen mixes the ink 45% into the surface).
- `tile`: `{bg, ink}` for icons drawn on a resource tile; `chip`: true for chain symbols; `badge`: the badge state whose pill the icon sits in; `band`: the card colour whose band the mini card mock uses.
