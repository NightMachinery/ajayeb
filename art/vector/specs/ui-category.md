# Spec: category glyphs, badge glyphs, crests, laurel, wonder and token frame

Set name in `sets.json`: `ui-category`. 15 files. Read [../STYLE.md](../STYLE.md) first. Acceptance: each category identifiable from its glyph alone in greyscale at the size of a 54 px card; the five affordability states distinguishable in greyscale and under each colour-vision emulation at the mini size; the crests shape-distinct; the laurel at 100% blind identification (it counts with the resources class).

## Category glyphs (7)

One glyph per card colour (`Color` in the wire types: brown, grey, blue, green, yellow, red, purple). Colour, CSS band pattern and glyph each identify the category on their own, so colour-blind players lose nothing (UX design, "Colour, pattern and glyph"). The glyph is drawn in the card colour with `--i2` a 45% mix of that colour into the card paper.

The UX design proposed a mountain, an amphora, a dome, an eight-pointed star, a coin, a shield and a crown. Three of those collide with other icons in this family (the amphora with chain-amphora, the coin with coin, the shield with shield), and a player would read a category glyph as an effect. This spec keeps the mountain, dome, star and crown, and replaces the three colliding ones.

Sizes for all seven: 15 to 17 px in the card's name row of the full tier, the discard pile and the score table; the glyph in the placeholder art (larger, on a tinted surface, with a halo); 1.1 em in the discount effect (blue). Contexts: CardFace (full tier), CardPlaceholderArt, CityBand, DiscardPile, ScoreTable, GameOver, EffectGlyph discount.

### category-brown

- Meaning: raw materials (brown cards produce wood, clay and stone).
- Metaphor: a mountain or a quarried hill.
- Silhouette: two overlapping peaks, the front one larger, with a snow line or strata as the detail tone. Wide base, 20 units.
- Must differ from: resource-stone (peaks, not a boulder), science-set-square (not a single right triangle), category-green.

### category-grey

- Meaning: manufactured goods (grey cards produce glass and papyrus).
- Metaphor: the workshop: an anvil, or a potter's kiln. Not an amphora (chain-amphora) and not a flask (resource-glass).
- Silhouette: an anvil seen from the side: a horn to one side, a flat top, a narrow waist and a broad foot.
- Correction (M9.VG, see the log): the horn is long and heavy (a wedge from the left edge to the waist, 4 to 5 units deep where it meets the face) and the waist and foot sit under the right half of the face, so the outline is plainly lopsided. A short horn read as an I-beam or a spool at 12 to 16 px.
- Must differ from: resource-glass, chain-amphora, chain-barrel.

### category-blue

- Meaning: civic buildings (blue cards give victory points).
- Metaphor: a dome on a drum, a pointed onion dome over an arched door.
- Silhouette: a dome rising to a point, on a short wall with an arched door knockout.
- Must differ from: chain-temple-facade (a dome, not a pediment over columns), wonder, chain-helmet.

### category-green

- Meaning: science buildings.
- Metaphor: the eight-pointed star (khatam) of two overlapping squares, a mark of learning in manuscript ornament.
- Silhouette: an eight-pointed star of 20 units with a round or octagonal centre as the detail tone.
- Correction (M9.V4, see the log): the star is drawn as two overlapping four-pointed stars, not two squares, so its points are sharp. The two-square khatam with a centre read as chain-gear at 12 to 16 px. The octagonal centre is painted in the detail tone over a solid star, so the silhouette has no hole.
- Must differ from: chain-sun (no disc, straight star points), chain-gear, ornament-rosette (the rosette is an ornament, not a glyph; keep the star sharper and simpler).

### category-yellow

- Meaning: commercial buildings (yellow cards: coins, trade and discounts).
- Metaphor: a merchant's purse tied at the neck, or a market stall awning. Not a coin (coin is an effect icon).
- Silhouette: a round purse with a gathered, tied neck and the tie as the detail tone.
- Must differ from: coin, coin-loss, looting-2 and looting-5 (the looting tokens may show coins; keep the purse closed and whole).

### category-red

- Meaning: military buildings (red cards give shields).
- Metaphor: two crossed spears, or a war banner. Not a shield (shield is the effect icon, shown on the same red cards).
- Silhouette: two spears crossed at the centre, points up, or a pennant on a staff.
- Correction (M9.VG, see the log): 3-unit shafts, broad leaf heads (6 units wide) with an ogee point, short butts and a round boss of radius 3 at the crossing, the whole glyph moved down half a unit. Thin shafts with small heads read as a bare X (close, wrong, or a multiplication sign) below 16 px.
- Must differ from: shield, chain-sword (two crossed weapons, not one upright blade), chain-target.

### category-purple

- Meaning: guilds (purple cards score by the cards and wonders of the cities).
- Metaphor: a crown.
- Silhouette: a crown with three to five points ending in balls or pointed leaves, on a solid band, wider than tall.
- Must differ from: chain-tower (points, not crenels), wonder, chain-temple-facade.

### Category -sm variants (added in M9.VF1)

Correction (M9.VF1, see the log): each category glyph has a `-sm` file on the 12 grid, for the colour swatches of `EffectGlyph.svelte` ("per card of colour X" on commerce cards and guilds, and the destroy effect of two wonders). The swatch is a small card shape in the band colour with the glyph in the band ink, about 9 px on a mini card and 15 px on a medium one, so the glyph, not the hue, names the colour. The `-sm` files keep the md ideas with fewer, fatter parts: the anvil gets a long wedge horn so its outline is asymmetric, the star loses its centre, and the spears get broad heads and a boss at the crossing so they do not read as an X. The md files stay in use everywhere else.

## Affordability badge glyphs (3, with coin and trade)

The badge on a takeable card (`AffordBadge.svelte`) shows one of five states; three glyphs live here, and the other two are `coin` (own) and `trade` (trade) from the resources set. All five must separate by shape alone, since the pills' colours collapse in greyscale (malachite and saffron become similar greys). Sizes: about 10.5 px on a mini card (1.05 em of the 10 px badge), 14 to 16 px in the card sheet's cost breakdown and the token picker.

### badge-check

- Meaning: "free": the card costs nothing.
- Metaphor: a check mark.
- Silhouette: a heavy check, 2.5 to 3 units thick, filling the live area.
- Correction (M9.VG, see the log): the check is about 4.5 units thick at the vertex, with a square-cut short arm and a long arm that tapers to a 2.5-unit wedge end, so it has the solid weight of the family rather than the weight of a line icon.
- Must differ from: trade, badge-chain.
- Contexts: AffordBadge free (paper glyph on a malachite pill), CostBreakdown, CardPickGrid, TokenChoice.

### badge-chain

- Meaning: "chain": free thanks to a chain symbol you own; also the chain-bonus effect.
- Metaphor: two interlocked links.
- Silhouette: two oval links overlapping at a diagonal, each with a knockout of at least 2 units; at 10.5 px the knockouts must stay open.
- Correction (M9.VG, see the log): each link is 9 units tall with a 3-unit rim and a 3-unit slot, so the pair has the family's solid weight; the slots stay open at 10.5 px.
- Must differ from: the chain symbols themselves (this is the idea of chaining, not a symbol), badge-lock (no shackle on a body), trade.
- Contexts: AffordBadge chain (card ink on a gold pill, `--i2` gold-deep), CostBreakdown, EffectGlyph chain-bonus.

### badge-lock

- Meaning: "short": you cannot afford the card (discarding it and using it for a wonder remain possible).
- Metaphor: a padlock.
- Silhouette: a solid body with a keyhole knockout and a shackle of 2 units; the shackle may be the detail tone, but the body alone must read as a lock.
- Must differ from: price-tag, chain-book, badge-chain.
- Contexts: AffordBadge short (desaturated red glyph on a pale pill with a dashed border), TokenChoice (a locked token).

## Crests (2)

- **crest-sun**: the Sun player (seat 0, turquoise). Metaphor: a sun disc with a calm face or a ring of flame-shaped rays, inside a roundel. It must differ from chain-sun (which is a plain rayed disc): give the crest its frame or face.
- **crest-moon**: the Moon player (seat 1, pomegranate). Metaphor: a crescent cradling a small star, inside a roundel (see the corrections below: the star is gone). It must differ from chain-moon (a plain crescent) and choice.
- Correction (M9.V4, see the log): neither crest has an outer roundel. A ring around the crescent read as a copyright sign at 12 to 16 px, a solid medallion blurred into a disc, and a ring around the rays read as a gear. crest-sun is a disc with twelve pointed flame rays and a calm face knocked out of it (the face and the twelve rays set it apart from chain-sun's eight separate rays); crest-moon is a crescent cradling a four-pointed star (the star sets it apart from chain-moon). The callers set no detail tone for the crests, so both are drawn in the main tone only.
- Correction (M9.VG, see the log): a crescent cradling a star is a national-flag and religious emblem, which art/style.md rules out. crest-moon is now a fat crescent whose inner edge is a calm face in profile (brow, nose, lips and chin) with one round eye knocked out, the man-in-the-moon of folk almanacs. The profile and the eye set it apart from chain-moon's plain, thinner crescent, and the face pairs it with the sun's. crest-sun's face is now calmer: two round eyes and no mouth (the smiling face read as cheerful at 60 px). crest-sun has a `-sm` file on the 12 grid (a plain disc of radius 3.5 with eight short, broad rays, no face) for 14 px and below, chiefly the crests at the ends of the military track (about 11 px), where the twelve rays and the face blurred into a fuzzball. Eight long rays on a small disc made the `-sm` sun the same spiky star as category-green at 12 px, so the disc dominates and the rays are stubs.
- The two crests must differ by shape, not only by colour: a round radiant figure against a crescent figure. Seen side by side at 15 px in greyscale, a player must still tell them apart.
- Sizes: 15 to 20 px in strips, panels and the score table; 60 px on the pass-the-device screen (where detail is welcome); about 16 px at the ends of the military track, on the track surface, with a halo.
- Contexts: PlayerStrip, CityPanel, CitySheet, Draft, AgeInterstitial, ScoreTable, GameOver, PassDevice, MilitaryTrack.

## vp-laurel

- Meaning: victory points.
- Metaphor: a laurel wreath, open at the top.
- Silhouette: two curved branches of leaves meeting at the bottom, open at the top, with the leaves as distinct points (not a blurred ring at 12 px).
- Must differ from: token-frame (open, leafy), chain-lyre.
- Sizes: 16 to 20 px. The effect band draws numbered points as a green CSS disc, not with this icon.
- Contexts: PlayerStrip and CityPanel score, AgeInterstitial, ScoreTable.

## wonder

- Meaning: a wonder (built or to build); used for wonder counts and effects that count wonders.
- Metaphor: a monumental gate: a tall pointed-arch portal (an iwan) between two slender towers.
- Silhouette: wider at the base, a central pointed arch as a knockout, two minaret-like towers at the sides.
- Correction (M9.VG, see the log): no towers. Two slender pointed towers beside an iwan read as a mosque front with minarets from 24 px, which art/style.md rules out, and as the letters "lnl" at 12 px. The wonder is now a pylon gateway: a tall central block with a cornice and the pointed-arch portal (the arch in the detail tone), flanked by two lower, slightly tapered wings, on a plinth. One solid silhouette, flat tops: no dome (category-blue), no pediment or columns (chain-temple-facade), no crenels (chain-tower), no points (category-purple).
- Must differ from: chain-temple-facade, category-blue (no dome), chain-tower, category-purple.
- Sizes: 12 to 14 px in the wonder rows and the structure slot, 1.1 em in effects, 16 px in the score table.
- Contexts: WonderTile, WonderRow, StructureSlot, Draft, EffectGlyph (coins per wonder, points per wonder, wonders play again), ScoreTable.

## token-frame

- Meaning: a progress token; also the frame drawn around a token's initial.
- Metaphor: a round medallion frame with a scalloped or beaded rim.
- Silhouette: a ring whose inside stays empty (the inner radius at least 6.5 units), so the token's initial letter, drawn by the component, fits inside at 26 to 36 px. The rim carries the ornament.
- Correction (M9.VG, see the log): an eight-lobed (scalloped) ring: a band from radius 6 to 8 with eight round lobes of radius 3 centred on radius 7, reaching radius 10, the inner disc clipped out. The inner radius is 6, not 6.5: with the 2-unit minimum feature, a 1.5-unit gap between lobes and the 2-unit live-area margin there is no room for lobes deep enough to show at 12 px around a 6.5 hole. The twelve-bead ring it replaces read as a zero beside a number. The initial still fits inside at 26 and 36 px (checked in the app's progress tokens).
- Must differ from: coin (empty centre, ornamented rim), chain-target, vp-laurel.
- Sizes: 26 and 36 px as the medallion behind the initial; 1.1 em as an effect glyph (points per token, draw tokens); 17 px in the score table.
- Contexts: ProgressToken, EffectGlyph, ScoreTable.

## Ids the inventory needs that names.ts lacks

- None are required. "trade" doubles as the trade badge and "coin" as the own badge, as `AffordBadge.svelte` does today.
- Optional: three effect glyphs that `EffectGlyph.svelte` currently borrows from Lucide (`rotate-cw` for play again, `archive-restore` for build from the discard, `x` over swatches for destroy). Their outline style does not match this family. If the family check flags them, add `play-again`, `build-from-discard` and `destroy-card` to `names.ts` and `sets.json` and switch the component (a coordinator change).
