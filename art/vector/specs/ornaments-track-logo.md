# Spec: ornaments, track parts and the logo

Set name in `sets.json`: `ornaments-track-logo`. Ten files, none drawn yet (the favicon exists as an M5 placeholder). Read [../STYLE.md](../STYLE.md) first. These pieces carry the manuscript look of the whole app, but they must never fight the game information: an ornament is quieter than any icon next to it.

## Where these files live

Ornaments are larger and more detailed than icons, and most are used as backgrounds or frames, not as glyphs in text. They therefore stay **out of the icon sprite** (its 40 KB budget is for glyphs):

- `web/src/lib/ornaments/svg/<id>.svg` for the ornaments and the track cell (a new directory; components import them as raw SVG or as CSS background images).
- `web/src/lib/icons/svg/capital-sun.svg` and `capital-moon.svg` in the sprite, since they are glyph-sized and themed like crests.
- `web/src/lib/brand/logo-wordmark.svg` for the wordmark (a new directory).
- `web/static/favicon.svg` for the favicon (replacing the M5 placeholder in place).

Creating the two directories, adding `capital-sun` and `capital-moon` to `names.ts`, and wiring each piece into its component are coordinator changes; the drawing agent asks for them in its result.

Lint rules for the set (in `sets.json`): any viewBox, 4 KB per file, one gradient allowed, and token paints such as `var(--gold)` allowed. The capitals follow the icon rules. The wordmark may have unsnapped outlines and 8 KB. The favicon may use raw colours (it is served without the stylesheet), on a 32 grid, in 2 KB.

## ornament-rosette

- Meaning: none; the card corner ornament of the medium and full tiers.
- Metaphor: a small eight-petalled rosette, as in the corners of an illuminated page.
- Shape: a square viewBox (`0 0 24 24`), petals around a disc, in gild with `--i2` paper. It must read as decoration at 10 to 16 px and must never be mistaken for category-green (which is a sharp star) or chain-sun.
- Sizes: 10 to 16 px in the card corners.
- Contexts: CardFace medium and full tiers (an M10 or M11 integration).

## ornament-cartouche

- Meaning: none; the frame behind the effect band's paper inset.
- Metaphor: a cartouche: a horizontal panel with pointed or lobed ends.
- Shape: a wide viewBox (about 4:1) designed to stretch horizontally (draw the ends as fixed shapes; the component may split it into three slices). A thin gild outline around a paper field.
- Sizes: the width of a card's band, 54 to 220 px.
- Contexts: CardFace band.

## ornament-divider

- Meaning: none; a horizontal rule between sections of sheets and panels.
- Metaphor: a manuscript rule with a central medallion and tapering arms.
- Shape: a wide viewBox (about 12:1), the medallion at the centre, arms that thin to points. Gild or rule colour.
- Sizes: 120 to 320 px wide, 8 to 16 px tall.
- Contexts: Sheets, the score table, the end-of-age interstitial.

## ornament-border

- Meaning: none; the illuminated border (tazhib) of large panels such as the game-over screen.
- Metaphor: a running border of interlaced vines or a repeating lobed band.
- Shape: a tile meant for CSS `border-image`: corners plus one repeating edge segment, in a square viewBox, with gild on transparent. It must tile without seams.
- Sizes: borders 8 to 16 px thick.
- Contexts: GameOver, the home hero, the about page.

## ornament-girih

- Meaning: none; the faint background pattern of the table.
- Metaphor: a girih tile (the interlaced star and polygon geometry of Persian tilework).
- Shape: a seamless square tile, lines only, drawn in `var(--rule)`, so that at 6 to 10% opacity it gives the table a texture in both themes. It must tile without seams (check a 3x3 repeat).
- Sizes: a tile of 96 to 160 px.
- Contexts: the board's table surface, the home screen.

## track-cell

- Meaning: one space of the military track.
- Metaphor: a small tile of the board, with a lobed top edge.
- Shape: a viewBox of about 2:3 (narrow and tall), a frame that tells the zones apart by an inner mark: plain in the centre zone, one notch in the 2-point zone, two in the 5-point zone, three in the 10-point zone. The values themselves are text drawn by the component.
- Sizes: at 360 px wide the track has 19 cells in about 340 px, so a cell is 16 to 18 px wide.
- Acceptance: at 360 px the cells are countable and the zone boundaries visible.
- Contexts: MilitaryTrack.

## capital-sun and capital-moon

- Meaning: the two capitals at the ends of the military track; the pawn reaching one wins the game for the other player.
- Metaphor: a city gate with a pointed arch and two towers, crowned by the player's crest figure (a sun disc, a crescent).
- Shape: sprite icons on the 24 grid, following every icon rule. The gate is shared; the crest figure on top differs, so each reads as "this player's city".
- Must differ from: wonder (the wonder has no crest figure, and the capital has no minarets), crest-sun and crest-moon (the capital is a building), chain-tower.
- Sizes: 16 to 24 px at the track ends.
- Contexts: MilitaryTrack ends. New ids: `capital-sun` and `capital-moon` must be added to `names.ts`.

## logo-wordmark

- Meaning: the Ajayeb logo: the Persian word عجایب ("wonders") above or beside the Latin "Ajayeb".
- Source font: **Noto Nastaliq Urdu**, SIL Open Font License 1.1. The copy at `/System/Library/Fonts/NotoNastaliq.ttc` on macOS declares OFL 1.1 in its name table (license field and license URL), copyright Google. Prefer the upstream release from the Noto project (github.com/notofonts/nastaliq) for provenance and record its version in the log. Only outlines ship, never the font, so the logo is artwork made with the font, which OFL allows. Another installed Nastaliq font, IranNastaliq, has a proprietary license and must not be used.
- Making the outlines: `hb-view --font-file=NotoNastaliq.ttc --face-index=0 --font-size=200 --output-format=svg --output-file=wordmark-fa.svg "عجایب"` writes each glyph outline once as a path inside `<defs>` and places it with `<use>`; resolve the uses into plain paths at their positions, crop the viewBox to the ink, and round coordinates to 0.1 in a viewBox about 200 units wide. A probe render on 2026-10-10 joined the letters correctly. The Latin "Ajayeb" is set the same way from `web/font-sources/fraunces-600.ttf` (OFL), then outlined.
- Design: the Nastaliq word is the hero; it may be redrawn by hand from the outlines (kashida length, dot shapes) as long as the letters stay correct and readable. The Latin line is smaller and centred under it. Paint with `var(--i1, currentColor)`, with an optional gild accent through `var(--gild)`.
- Never use the typeface or lettering of the official game's logo.
- Sizes: 120 to 240 px wide on the home screen and the about page.
- Acceptance: elegant at 200 px in both themes, letters correct (a Persian reader should read عجایب without hesitation).

## favicon

- Meaning: the browser tab and home-screen icon.
- Metaphor: the first letter ع of the wordmark, in gold on a lapis tile, or the whole Nastaliq word squeezed into a roundel if it still reads at 32 px.
- Shape: `viewBox="0 0 32 32"`, a rounded lapis square (radius 6 to 7) with the gold letterform. Raw colours are allowed here: use the hex equivalents of `--lapis` and `--gold` from `web/src/lib/styles/tokens.css`, and say so in the log.
- Sizes: 16 and 32 px in tabs, and the source of the 180, 192 and 512 px PNG app icons (made later by the bitmap pipeline).
- Acceptance: recognisable at 32 px and identifiable at 16 px, on light and dark browser chrome.
- Contexts: `web/src/app.html` links it as `favicon.svg`.

## Ids the inventory needs that names.ts lacks

- `capital-sun` and `capital-moon` (sprite icons).
- The ornament, track-cell, wordmark and favicon ids are files outside the sprite, so they never go in `names.ts`; `sets.json` names their paths.
