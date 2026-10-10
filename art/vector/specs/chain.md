# Spec: chain symbols

Set name in `sets.json`: `chain`. 17 files, one per chain symbol. Read [../STYLE.md](../STYLE.md) first. Acceptance: blind identification 90% or better at 16 px DPR 1; the "in" chip and the "out" chip read as the same symbol; legible inside the mini card's chip at about 12 px.

## Shared rules of the set

- A chain symbol printed on a card's effect band ("out") makes one later card free for the player who owns it; the later card shows the same symbol in its cost row ("in"). The player must match the two across ages, so each symbol needs one memorable silhouette.
- The list below comes from the server catalog (`Chain` in `server/internal/engine/catalog/types.go`, the TypeScript `Chain` type in `web/src/lib/wire/view.gen.ts`). It matches `names.ts` exactly: 17 symbols, `chain-<value>`. No id is missing and none is extra.
- Every glyph sits in a chip (`ChainChip.svelte`) of 1.3 em with the glyph at 0.95 em: a gold chip for "in", a paper chip with a gold-deep border for "out". The glyph is card ink with `--i2` gold-deep, in both themes (cards stay paper). So the main tone carries everything; the detail tone is a gold accent.
- There are no `-sm` variants yet. Add one (and its `names.ts` line and `sets.json` field) only for a symbol that fails at 12 px after its iterations.
- Groups of related symbols must still differ: five come from red cards (horseshoe, sword, tower, target, helmet), six from blue cards (mask, moon, drop, sun, column, temple-facade), four from green cards (book, gear, lyre, lamp) and two from yellow cards (amphora, barrel). Inside each group, vary the silhouette family (round, tall, wide, pointed).

## The symbols

For each: the card that shows it, then the card it makes free; then the metaphor, silhouette and must-differ-from list. Sizes for all: about 12 px in the mini chip, 14 to 16 px in the medium tier and the band tier, 20 px in sheets. Contexts for all: the cost row (in), the effect band and the band tier of the city sheet (out), and the cost breakdown in the card sheet.

## chain-horseshoe

- Cards: Stable (Age I, red) to Horse Breeders (Age II, red).
- Metaphor: a horseshoe, open end down, with nail holes.
- Silhouette: a thick U, the opening at the bottom, nail holes as knockouts (two or three per side at md, none at 12 px).
- Must differ from: chain-lyre (no strings, open at the bottom), chain-moon, chain-helmet.

## chain-sword

- Cards: Garrison (I, red) to Barracks (II, red).
- Metaphor: an upright straight sword or a curved shamshir, point up.
- Silhouette: a vertical blade with a clear crossguard and pommel.
- Must differ from: science-quill (crossguard, no vane), chain-column (thin blade, not a shaft with capitals), chain-tower.

## chain-tower

- Cards: Palisade (I, red) to Fortifications (III, red).
- Metaphor: a watchtower with crenellations.
- Silhouette: a tall block with a crenellated top and a door or window knockout.
- Must differ from: chain-column (crenels, not a capital), chain-temple-facade (no pediment), category-red.

## chain-book

- Cards: Scriptorium (I, green) to Library (II, green).
- Metaphor: a bound codex, closed, seen from the front, with a clasp; or an open book.
- Silhouette: an upright rectangle with a spine and a clasp or a centre medallion as the detail tone.
- Must differ from: resource-papyrus (not rolled), chain-tower, science-set-square.

## chain-gear

- Cards: Pharmacist (I, green) to Dispensary (II, green).
- Metaphor: a toothed cog.
- Silhouette: a disc with six to eight square teeth and a round hole; the teeth at least 2 units wide.
- Must differ from: science-wheel (teeth on the rim), chain-sun (square teeth, not rays), chain-target.

## chain-amphora

- Cards: Tavern (I, yellow) to Lighthouse (III, yellow).
- Metaphor: a two-handled amphora, wine jar.
- Silhouette: a tall jar with two looped handles at the neck and a pointed foot.
- Must differ from: resource-glass (handles), chain-barrel, category-grey (if category-grey is a vessel, make them different vessels; see ui-category), chain-drop.

## chain-mask

- Cards: Theater (I, blue) to Statue (II, blue).
- Metaphor: a theatre mask.
- Silhouette: a face-shaped mask with two eye knockouts and a mouth knockout, slightly wider at the top.
- Must differ from: military-pawn, chain-helmet (no crest, eye holes), crest-sun.

## chain-moon

- Cards: Altar (I, blue) to Temple (II, blue).
- Metaphor: a crescent moon.
- Silhouette: a thick crescent opening to the right, with a small star as an optional detail.
- Must differ from: crest-moon (the Moon player's crest; the crest must carry a frame or a star, see ui-category), chain-horseshoe, choice.

## chain-drop

- Cards: Baths (I, blue) to Aqueduct (II, blue).
- Metaphor: a water drop.
- Silhouette: a teardrop pointing up, 19 units tall, with a highlight as the detail tone.
- Must differ from: resource-glass (no neck), chain-amphora, chain-lamp (no spout).

## chain-target

- Cards: Archery Range (II, red) to Siege Workshop (III, red).
- Metaphor: an archery target with an arrow, or concentric rings.
- Silhouette: a disc with two rings and a centre; an arrow across it helps separate it from other discs.
- Must differ from: coin, science-wheel, chain-gear, chain-sun, token-frame.

## chain-helmet

- Cards: Parade Ground (II, red) to Circus (III, red).
- Metaphor: a crested helmet or a pointed Persian helmet with a nasal bar.
- Silhouette: a dome with a point or crest on top and cheek guards.
- Must differ from: chain-mask, chain-temple-facade (no columns), shield.

## chain-lyre

- Cards: School (II, green) to University (III, green).
- Metaphor: a lyre or a small harp.
- Silhouette: a U-shaped frame open at the top, two curved arms, three strings in the detail tone.
- Must differ from: chain-horseshoe (opening up, strings), science-scales.

## chain-lamp

- Cards: Laboratory (II, green) to Observatory (III, green).
- Metaphor: an oil lamp with a flame.
- Silhouette: a low lamp body with a spout to one side and a flame rising from it, the flame in the detail tone.
- Must differ from: science-mortar, chain-drop, chain-amphora.

## chain-barrel

- Cards: Brewery (II, yellow) to Arena (III, yellow).
- Metaphor: a barrel or cask, upright.
- Silhouette: a bulging upright barrel with two hoops as knockouts or detail bands.
- Must differ from: resource-wood (one body, hoops), chain-amphora, chain-column.

## chain-sun

- Cards: Temple (II, blue) to Pantheon (III, blue).
- Metaphor: a sun with rays.
- Silhouette: a disc with eight short, broad, pointed rays.
- Must differ from: crest-sun (the Sun player's crest; give the crest a face or frame and keep this one plain), chain-gear (pointed rays, not square teeth), science-wheel, science-sundial.

## chain-column

- Cards: Statue (II, blue) to Gardens (III, blue).
- Metaphor: a single column with capital and base.
- Silhouette: a tall shaft with a wider capital and base, with fluting as the detail tone.
- Must differ from: chain-tower, chain-temple-facade, chain-sword.

## chain-temple-facade

- Cards: Rostrum (II, blue) to Senate (III, blue).
- Metaphor: a temple front: a pediment over columns, or an iwan arch.
- Silhouette: a triangle or pointed-arch roof over three or four columns on a base, wider than tall.
- Must differ from: chain-column (several columns and a roof), wonder (the wonder glyph must not be a temple front; see ui-category), category-blue.

## Ids the inventory needs that names.ts lacks

None for this set. The 17 catalog values match the 17 `chain-*` names.
