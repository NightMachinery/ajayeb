# Spec: resources, economy and military

Set name in `sets.json`: `resources-economy-military`. 19 files: five resources with their `-sm` variants, five economy markers and four military markers. Read [../STYLE.md](../STYLE.md) first; it holds the grid, the paint convention and the loop. Acceptance: blind identification 100% at 12 px DPR 2 and 16 px DPR 1, distinct in greyscale, legible on all 7 bands, no blobbing in the pixel peep, light and dark parity.

These are the most frequent glyphs in the game: every cost row and most strips show them. They must be the clearest icons of the whole family.

## Shared rules of the set

- The five resources must differ by **silhouette family**, not by colour: one stacked, one rectilinear, one irregular, one faceted and pointed, one wide and rolled. In greyscale and in plain ink every pair must still separate.
- Each resource has an `-sm` file drawn again on the 12 grid for the mini card (about 9 px inside a 10 px tile) and the 12 to 14 px strips. The `-sm` drawing keeps the md silhouette and drops inner details.
- The resource tiles colour the background, not the glyph: the glyph is paper on wood, clay and stone tiles, and ink on glass and papyrus tiles. So the main tone must make a strong silhouette on a mid-tone tile.

## resource-wood (with resource-wood-sm)

- Meaning: the wood resource (raw material, brown cards produce it).
- Metaphor: a bundle of timber: two or three trimmed logs lashed together, or a stack of cut logs seen from the side.
- Silhouette: a compact vertical or diagonal stack of rounded staves, with the lashing as the detail tone. The ends of the logs may show as rounded or ring-cut tips.
- Must differ from: resource-clay (no grid of cells), resource-papyrus (not a single horizontal roll), chain-barrel (no hoops on a rounded body).
- Sizes: 9 to 12 px on tiles, 11 to 16 px in strips, 24 px in sheets.
- Contexts: resource tiles in the cost row and the effect band, ResourceChips, CostBreakdown.

## resource-clay (with resource-clay-sm)

- Meaning: the clay resource.
- Metaphor: fired bricks: two or three bricks in a running bond, or a single brick with a mould mark.
- Silhouette: rectilinear blocks with one clear mortar line, wider than tall. Square corners outside (radius 0.5), crisp seams.
- Must differ from: resource-wood (no rounded staves), resource-stone (rectilinear, not irregular), category-red (no wall battlement).
- Sizes and contexts: as resource-wood.

## resource-stone (with resource-stone-sm)

- Meaning: the stone resource.
- Metaphor: a rough quarried boulder, or two stacked uneven stones.
- Silhouette: an irregular, rounded, lumpy mass with one or two chisel facets as the detail tone. Low and broad, the weight at the bottom.
- Must differ from: resource-glass (not symmetric, not pointed, not gem-cut), resource-clay (no straight edges), crest-moon and coin (not a clean disc).
- Sizes and contexts: as resource-wood.

## resource-glass (with resource-glass-sm)

- Meaning: the glass resource (manufactured, grey cards produce it).
- Metaphor: a small blown-glass flask or bottle, or a cut crystal. Prefer the flask: it separates glass from stone by idea, not only by cut.
- Silhouette: a symmetric vessel with a narrow neck and a round or pear-shaped body, with a single highlight sweep as the detail tone.
- Must differ from: resource-stone (symmetric and pointed or necked), chain-amphora and chain-drop (no handles; no teardrop without a neck), science-mortar.
- Sizes and contexts: as resource-wood.

## resource-papyrus (with resource-papyrus-sm)

- Meaning: the papyrus resource (manufactured, grey cards produce it).
- Metaphor: a rolled sheet: a scroll with both ends curled.
- Silhouette: a wide shape at least 8 units tall (md), the rolled ends as two clear curls, the sheet face as the detail tone. Not a thin bar: the M5 placeholder vanishes at 12 px.
- Must differ from: resource-wood (one sheet, not staves), chain-book (no spine, no pages), science-quill.
- Sizes and contexts: as resource-wood. Its tile is pale (papyrus yellow), so the ink glyph needs weight.

## coin

- Meaning: money, the coin treasury; also the "own" affordability badge (you pay only the card's coin cost).
- Metaphor: a struck coin: a disc with a raised rim and a simple centre mark (a small square hole or a star).
- Silhouette: a disc of 18 to 20 units with the rim or the centre as the detail tone. No numeral: the Coin component draws numbered coins in CSS.
- Must differ from: coin-loss (no slash), crest-sun (no rays), token-frame (not a thin ring), resource-stone.
- Sizes: 10.5 px in the mini badge, 16 to 20 px in strips and panels.
- Contexts: PlayerStrip and CityPanel treasury (gild colour), AffordBadge "own" (gold glyph on a malachite pill, `--i2` gold-deep), CostBreakdown, AgeInterstitial, EffectGlyph trade income.

## coin-loss

- Meaning: the opponent loses coins (an effect), and the looting penalty.
- Metaphor: a coin breaking: a coin with a crack or a bite out of it, or a coin falling out of a purse.
- Silhouette: the coin silhouette with a clear break that survives greyscale. Not just a red tint.
- Must differ from: coin (clearly damaged), looting-2 and looting-5 (no arrow), badge-lock.
- Sizes: 1.1 em in the effect band (about 11 to 16 px).
- Contexts: EffectGlyph "opponent loses coins" in cinnabar, before a Coin with a minus.

## trade

- Meaning: trading: buying a resource from the bank at the opponent's price; also the "trade" affordability badge and the trade-income effect.
- Metaphor: exchange: two opposed curved arrows, or two hands passing a coin. Prefer bold arrows on the 12 grid of the badge.
- Silhouette: two heavy arrows in opposite directions, each at least 2 units thick, filling the live area. Both arrows in the main tone (the placeholder put one in `--i2`, and it vanished on night).
- Must differ from: price-tag, chain symbols, Lucide's rotate icon (play again).
- Sizes: 10.5 px in the mini badge (the hardest case), 15 to 16 px in sheets.
- Contexts: AffordBadge "trade" (card ink on a saffron pill), EffectGlyph trade income (beside a coin).

## price-tag

- Meaning: fixed trading price: this resource always costs 1 coin to buy (the "=1" effect of some yellow cards).
- Metaphor: a merchant's tag on a string, or a balance-free price marker.
- Silhouette: a tag with a punched hole (a knockout of 2 units or more) and a tied string as the detail tone.
- Correction (M9.VG, see the log): no string. The tag body is 14 units wide and 17.5 long with a blunter point, set at 45 degrees, so it is as big optically as the shield beside it; the string took the corner the body needed, and the only caller (EffectGlyph trade-fix) sets no detail tone, so the string never showed in the app.
- Must differ from: badge-lock (no shackle), trade, wonder.
- Sizes: 0.9 em of the effect band (about 10 to 14 px).
- Contexts: EffectGlyph trade-fix, gold-deep, between the resource tiles and a Coin "1".

## choice

- Meaning: a production choice: this card produces one resource out of a set, chosen each turn.
- Metaphor: a fork in a path, or a split disc (half and half).
- Silhouette: a disc split into two halves of two tones, or a Y-shaped fork. At 8 px it must still read as "split".
- Must differ from: crest-moon (no crescent), coin.
- Sizes: 8 px inside a 12 px gold dot on the resource chips (the smallest use in the app), 16 px elsewhere.
- Contexts: ResourceChips choice marker (on-accent ink on a gild dot).

## shield

- Meaning: military strength (shields move the conflict pawn).
- Metaphor: a round or kite shield with a boss or a central rib.
- Silhouette: a shield of 19 to 20 units tall, pointed or rounded at the bottom, with one central detail in the detail tone. It appears in rows of up to three, so it must read when repeated at 12 px.
- Must differ from: category-red (the category glyph must not be a shield: give red cards a different glyph, see ui-category), crest-sun and crest-moon, chain-helmet.
- Sizes: 1.15 em in the effect band (about 12 to 17 px), 16 to 18 px in the score table and interstitial.
- Contexts: EffectGlyph shields and extra shield (card-red with `--i2` paper-bright), ScoreTable military row, AgeInterstitial.

## military-pawn

- Meaning: the conflict pawn on the military track.
- Metaphor: a game piece: a standing pawn or a small banner-bearing figure.
- Silhouette: a pawn with a round head, a narrowing body and a broad base, symmetric, 19 units tall.
- Must differ from: the crests, chain-mask.
- Sizes: 10 px (compact track) and 12 px.
- Contexts: MilitaryTrack, on a track cell; may use a halo.

## looting-2 and looting-5

- Meaning: the looting tokens on the military track: when the pawn reaches them, the opponent loses 2 or 5 coins.
- Metaphor: a coin pouch being plundered, or a coin stack with a grabbing hand. The value must be shown by **size or count of large parts** that survive 10 px: looting-2 with two large coins, looting-5 with a bigger heap or a distinct mark. Never by counting small dots (the placeholder's failure).
- Silhouette: a common base form for both, with a clear difference at 10 px.
- Must differ from each other at 10 px, and from coin-loss.
- Sizes: 10 px and 12 px inside a 16 or 24 px disc.
- Contexts: LootingToken on the track.

## Ids the inventory needs that names.ts lacks

None for this set.
