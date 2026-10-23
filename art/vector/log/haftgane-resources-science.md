# haftgane-resources-science: iteration log

## Status

- resource-ore, resource-ore-sm: done (iteration 2)
- resource-cloth, resource-cloth-sm: done (iteration 2)
- science-compass, science-compass-sm: done (iteration 2)
- science-tablet, science-tablet-sm: done (iteration 2)
- science-gear-sm: done (iteration 1); the md `science-gear` is Duel's `chain-gear`, referenced from `sets.json`, not copied

Every file is lint clean and between 200 and 412 bytes. The files live in `web/src/lib/games/haftgane/icons/svg/`, so Duel's icon source set (27252 bytes raw, 5638 gzipped) is unchanged.

## Iteration 0 (2026-10-23): fast-render sketches before the first specimen

Renders: art-work/vector/haftgane-resources-science/fast-1/ (librsvg strips beside the neighbours, and 600% peeps) and variants-1-ore.png, variants-1-cloth.png, variants-1-comp.png (each variant with and without `--i2`).

The first drafts, each from a one-sentence idea:

- resource-ore: a front face and a separate top face, a knocked-out edge between them, the top face in the detail tone.
- resource-cloth: a bolt across the top and a fold hanging centred below it, ending in four tabs.
- science-compass: a round hinge with a handle nub and two tapered legs, the rivet in the detail tone.
- science-tablet: a slab with rounded shoulders, its face knocked out and filled with the detail tone.
- science-gear-sm: a ring with a round hole and six teeth.

What the strips showed:

- resource-ore: the knocked-out edge splits the ingot into two tiers. At 12 px the `-sm` (which had no detail tone) read as a stepped pyramid, and two horizontal courses are exactly what makes clay clay. Three variants: a single six-sided bar with the top face in the detail tone (B), a plain trapezoid with a top band (C), and an oblique bar (D). D in plain ink is a parallelogram, a brick. C is clean but reads as a block. B reads as an ingot with the detail tone and stays one mass without it. Kept B for both files.
- resource-cloth: the centred fold made a T, which read as a table or a column capital. Of three variants (the fold hanging off the right side, a two-layer folded stack, a flared centred drape), the right-hand fold reads most like cloth over a bar, and its asymmetry keeps it apart from the symmetric tablet and papyrus. The two-layer stack is two bars at 12 px, like clay's courses. Kept the right-hand fold with three tabs.
- science-compass: legs 3 units wide that taper over their whole length were mostly sub-pixel at 12 px. Variant B keeps the shanks parallel and tapers only near the points; kept it for md.
- science-gear-sm: six teeth read as a six-pointed star at 12 px. Short teeth on a larger ring cure that. Eight short teeth (as many as the md `chain-gear`, so both sizes read as the same symbol) or six both read as a gear; kept eight. Four wide teeth read as a diamond.

## Iteration 1 (2026-10-23)

Renders: art-work/vector/haftgane-resources-science/iter-1/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the peeps).
Lint: clean (0 errors, 0 warnings).
Looked at: light@1x, peep-light@1x.

### resource-ore

- Seen: from 12 px the `-sm` is a bar with a lighter top face, an ingot; on night the top face is the darker tone, which still separates it. No blobbing. No change.

### resource-cloth

- Seen: the `-sm` fringe tabs are 1 unit long, so at 12 px they are single-pixel nubs, and the shape reads as a hooked bar more than as cloth. All corners are square, so nothing says soft.
- Changed: `-sm` tabs 2 units long; md tabs longer too. The bolt's ends rounder in both files, for the soft folded edge.

### science-compass

- Seen: md reads as a compass from 16 px, with the handle nub and the rivet. The `-sm` at 12 and 14 px reads as a letter A: the 2-unit hinge is no wider than the two legs where they meet.
- Changed: `-sm` hinge radius 2 to 2.5, the handle nub dropped to make room, the rivet kept as a 1-unit hole.

### science-tablet

- Seen: at 12 to 16 px a rounded rectangle with a light face: a frame, a phone or a window. The rounded shoulders are lost, so nothing says slab.
- Changed: the shoulders rounded much wider (radius 6 md, 3 `-sm`), so the slab has an arched top above a square face, and the border is heaviest above the face.

### science-gear-sm

- Seen: eight teeth at 12 px, slightly soft on the diagonals but a gear, and clearly not the spoked wheel. No change.

## Iteration 2 (2026-10-23)

Renders: art-work/vector/haftgane-resources-science/iter-2/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the peeps), and cloth-vs-anvil.png (librsvg, with and without `--i2`).
Lint: clean (0 errors, 0 warnings).
Looked at: light@1x, light@2x, dark@1x, peep-light@1x, peep-light@2x, grey-light@1x, deuteranopia-light@1x, tritanopia-light@1x.

### resource-ore

- Seen: unchanged and still an ingot at every size, on paper and night, on all seven bands and on the stone tile it shares with stone in the mini card. In greyscale the top face stays a separate tone. Done.

### resource-cloth

- Seen: the 2-unit `-sm` tabs now read as a fringe at 12 px DPR 2 and 14 px; the rounded bolt ends soften it. It is the least literal drawing of the set: at 12 px it is a fringed towel on a rail rather than a bolt. Its nearest neighbour by silhouette is the anvil (a bar across the top with a body below it, set to the right), and cloth sits on the same grey cards as the anvil. Side by side, the anvil keeps a narrow waist over a flared solid foot, and the cloth hangs as one broad body that ends in tabs, so they stay apart at 12 to 24 px with and without `--i2`. No change before the blind check.

### science-compass

- Seen: the `-sm` hinge now shows its rivet as a hole at 12 and 14 px, so it reads as dividers, not a letter A. The md keeps the handle nub and the rivet from 16 px. In greyscale the md at 12 px is thin, but the app draws the `-sm` file there. Done.

### science-tablet

- Seen: the wide shoulders give the slab an arched top above a square face from 12 px: a stela or writing tablet. It no longer reads as a frame, and it is apart from the book (square, with a spine band and a diamond) and from the wonder (a building with a door). Done.

### science-gear-sm

- Seen: unchanged, a gear at 12 and 14 px, beside the md `chain-gear` at 16 px and up; both sizes read as the same symbol. Done.

### Greyscale and colour vision

All five concepts separate by shape alone; no tone carries meaning that the emulations remove.

## Blind check (2026-10-23)

Reader: a fresh in-process subagent that had never seen the specimen, the sources or any labelled render. It was handed only the reader copies (sheets, `names.txt`, the answer template), told what each candidate name means in the game, and, for the two category glyphs whose ids name no object, what they show (an anvil, crossed spears). It was told to judge each tile on its own and never by elimination, and it confirmed it opened nothing else. The candidates were the five new concepts and the eleven `neighbours` of `sets.json`: the five Duel resources, the anvil, the crossed spears, the book, the lyre, the set square and the wheel. Haftgane's `chain-scroll` is drawn in the chains slice in parallel and was not on the sheets.

- On-card ink, art-work/vector/blind/haftgane-resources-science-2026-10-23T07-26-03-416Z/: 12 px DPR 2 (`-sm` files) 16/16, 16 px DPR 1 (md files) 16/16.
- Plain ink, shape only (`--mono`), art-work/vector/blind/haftgane-resources-science-2026-10-23T07-26-18-620Z/: 12 px DPR 2 16/16, 16 px DPR 1 16/16.

Misses: none. All nine new files were named right on every sheet. The reader's hesitations, recorded because they are the pairs to watch in context:

- Ore and stone: it read the trapezoid with a lighter top face as a metal ingot and the rough lump as a boulder, and said the reverse reading (a cut block of stone, a nugget of ore) was possible but fitted worse. On a card both sit on the same grey tile in this specimen, so the shape alone decided it.
- Cloth and papyrus: it read the bar with a fringed fold as a bolt of cloth with tassels, and the bar with two rolled ends as a scroll; each could just pass for the other.
- Tablet: a rounded-top slab with an inset panel; a tablet in the science colour, but a tombstone or a door out of context.
- The Duel neighbours: `science-wheel-sm` is a crosshair at 12 px, and `category-red` at 12 px looks more like crossed axes. Neither is a new drawing.

## Family check (2026-10-23)

art-work/vector/haftgane-resources-science/family@1x.png and family@2x.png (copies of the shared family sheet). Beside the Duel sets at 16 and 24 px, the five new glyphs have the weight and optical size of the resources and science sets. The compass is the lightest of them, an open shape like the set square's outline, but not thinner than the chain set's lyre. No outlier.

## The app

Not checked in the app: nothing imports these files yet. M19.art-wiring puts them on Haftgane's screens, picks the real tile tokens for ore and cloth (the specimen borrows stone's and papyrus's on purpose), and should look at them there, with the new `chain-scroll` beside the tablet.

Contact sheet with the neighbours: art-work/vector/haftgane-resources-science/contact-sheet.png (`svg-render.mjs`, 12, 16, 24 and 48 px on paper and night, `--i2` unset).
