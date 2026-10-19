# Spec: effect glyphs for play again, build from discard and destroy, the small moon crest, and the concept diagrams

Set name in `sets.json`: `tt2-glyphs`, 3 files; `crest-moon-sm` is listed in `ui-category` as the `sm` of `crest-moon`. Read [../STYLE.md](../STYLE.md) first. Slice M10.icons-and-diagrams, the vector and diagram work the second tooltip pass needs. Acceptance: lint clean, the sprite under its budget, each icon legible at 12 to 16 px on a light and a dark card, the family check passed (no stroke, size or density outlier beside the other sets), and the contact sheet looked at.

Until this set, `EffectGlyph.svelte` borrowed three outline icons from Lucide. Their 2 px outline style does not belong to this family (STYLE.md section 5: solid silhouettes, not line drawings), and the ui-category spec listed them as optional replacements. D20 holds in both directions: an effect glyph never reuses a category glyph's subject (mountain, anvil, dome, eight-pointed star, purse, crossed spears, crown), and the reverse.

## Effect glyphs (3)

Sizes for all three: 1 to 1.1 em of the effect band's font, so about 10 to 12 px on a mini card, 14 to 16 px on a medium card, a wonder tile or the card sheet, and 1.1 em in tooltip lines on the panel. Drawn in the card ink on the band's paper inset (no halo), or in the text colour on the panel; no caller sets a detail tone, so all three are main tone only.

### play-again

- Meaning: "take another turn at once" (five wonders, the Theology token's effect on wonders, and the second half of "wonders play again").
- Metaphor: a looping arrow that comes back to where it started.
- Silhouette: a thick ring (3 units) open at the upper right, with a broad solid arrowhead at its top end pointing clockwise into the gap. The head is wider than the ring by 2 units on each side, so it reads as an arrow at 12 px and not as a letter C or a broken ring.
- Must differ from: trade (two straight arrows), token-frame (a closed scalloped ring), coin, badge-chain.
- Contexts: EffectGlyph `play-again` and `wonders-play-again` (after the wonder glyph).

### build-from-discard

- Meaning: "build a card from the discard pile for free" (the Mausoleum).
- Metaphor: a card lifted back up out of a pile.
- Silhouette: a pile of two flat slabs at the bottom (the discard pile seen from the side), and above it an upright card with an upward arrow cut out of it. The pile is wider than the card, so the two parts read as "from the pile, up".
- Must differ from: the swatch (a card with a category glyph), chain-book, price-tag, wonder (no plinth with wings), resource-stone.
- Contexts: EffectGlyph `build-from-discard`.

### destroy-card

- Meaning: "discard one card of this colour from your opponent's city" (the Statue of Zeus and the Circus Maximus).
- Metaphor: a card broken in two.
- Silhouette: an upright card split by a jagged crack from top to bottom into two halves, which lean slightly apart. The crack is a gap of at least 1.5 units, so the break stays open at 12 px.
- Must differ from: the colour swatch beside it (a whole card in the band colour), badge-lock, chain-book, coin-loss.
- Contexts: EffectGlyph `destroy`, drawn in cinnabar just before the colour swatch, so it reads "destroy a brown card". It replaces a Lucide cross in a small paper disc on the swatch's corner, which was about 7 px at the mini tier and covered part of the category glyph.

## crest-moon-sm

- Meaning: the Moon player, at 14 px and below; chiefly the moon capital at the end of the compact military track (11 px), beside `crest-sun-sm`.
- Metaphor (corrected in M11.ornament-integration): a plain crescent moon, tilted so its horns lean up and to the right. The man-in-the-moon face of `crest-moon` is left out at this size: at 11 px its nose and eye filled the bite and the figure read as a round blob, the same mass as the sun (review finding S1-11).
- Silhouette: the outer arc a half circle of radius 5, the inner arc of radius 14, so the back is about 4 units thick and the horns taper to points; the whole crescent rotated 20 degrees. The bite is the open half of the disc, so even without colour the Moon is a crescent and the Sun a round radiant disc.
- Must differ from: crest-sun-sm (a round radiant figure; the pair must separate by shape in greyscale), chain-moon (a thin plain crescent, never shown at this size beside the crests; ours is twice as thick and tilted).
- Contexts: the compact MilitaryTrack capital at 11 px on the track's moon-tinted end cell; the rules diagrams (StarterDiagram, GuildDiagram, SeventhWonderDiagram) at 1 em.

## Concept diagrams (5)

Not sprite icons: Svelte components in `web/src/lib/components/rules/`, built from the real card components and sprite icons, in the manner of TradeDiagram, MilitaryDiagram and ScienceDiagram. HTML and inline layout, every label through `t()`, nothing baked into an SVG. Each takes a `compact` prop: the compact variant fits a tooltip bubble (at most about 220 by 120 CSS px) and drops the caption sentence; the full variant sits in the rules reference.

- **ChainDiagram** (rules section costs, under Chains): an Age I card whose band shows a chain symbol (the out chip), an arrow, then a later card with the same symbol beside its cost (the in chip) and the cost shown as free.
- **DiscardDiagram** (rules section turn, after the three actions): the word Discard, then 2 coins plus 1 coin per yellow card, with a worked example of two yellow cards giving 4 coins.
- **SeventhWonderDiagram** (rules section wonders, under the seventh wonder): eight wonder glyphs, seven of them built (four and three, one row per player), and the eighth crossed out as leaving the game.
- **GuildDiagram** (rules section guilds): the two cities' counts of one colour side by side, the larger one marked as the one the guild counts, whoever owns it.
- **StarterDiagram** (rules section age-end): the track with the pawn off centre, the weaker player's crest choosing who starts; and the pawn on the centre, the last active player's crest choosing (docs/rules/decisions.md).
