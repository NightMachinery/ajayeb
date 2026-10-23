# Spec: Haftgane chain symbols

Set name in `sets.json`: `haftgane-chain`. 24 chain symbols for Haftgane, Ajayeb's second game: 14 reused Duel drawings and 10 new ones. Read [../STYLE.md](../STYLE.md) and the Duel chain spec, [chain.md](chain.md), first; every shared rule there applies here unless this file says otherwise. The plan behind the list is [docs/haftgane/art-reuse.md](../../../docs/haftgane/art-reuse.md), "Chains: twenty-four symbols".

## Acceptance

- Blind identification of the 24 symbols at 90% or better on each sheet, at 12 px DPR 2 and at 16 px DPR 1, by a fresh reader who has not seen the labelled icons. The sheets also hold the out-of-set neighbours listed under "neighbours" in `sets.json` (Duel's sword, sun, globe and temple front, the purse, anvil and eight-point star category glyphs, and the papyrus resource), so a new drawing read as one of them counts as a miss.
- The "in" chip (gold) and the "out" chip (paper with a gold-deep border) read as the same symbol, with the glyph at 12 px in a 14 px chip.
- Each new drawing recognisable with `--i2` unset, on paper and on the night surface, and lint clean within the 1228-byte budget.

## Shared rules of the set

- A Haftgane card provides up to two chain symbols (top right of the card) and may show one as its free-construction condition. The engine's chain values are game data; the presenter maps each value to a drawing. The ids below are the drawings' ids, not wire values: the semantic name "columns" is drawn by `chain-column`, and "mortar", "scales" and "quill" by Duel's science drawings.
- **Reuse is by reference.** The 14 reused drawings stay where they are, in `web/src/lib/icons/svg/`, and `sets.json` points at them. Nothing is copied. `lint.mjs` refuses a file in the Haftgane directory named like a Duel icon.
- **New drawings live outside Duel's bundle,** in `web/src/lib/games/haftgane/icons/svg/`. `web/src/lib/icons/sources.ts` globs only `web/src/lib/icons/svg/`, so a file there would ship to every Duel player. Nothing imports the Haftgane directory yet; M19.art-wiring does that.
- md only (`viewBox="0 0 24 24"`), like Duel's chain set. Chain chips draw the md file at 12 px. Add a `-sm` file only for a symbol that fails at 12 px after its iterations.
- The paint and the chip colours are Duel's: the glyph is card ink with `--i2` gold-deep, in both themes, so the main tone carries everything. The detail tone is optional, and a symbol with no natural secondary layer uses the main tone only.
- **Silhouette families.** Haftgane's 24 symbols share chips on the same cards, so the set as a whole must spread over distinct families. Tall and narrow: column, tower, drop, lighthouse, torch, scroll, lightning. Round: target, planet, mask, helmet. Wide and low: lamp, mortar, awning, camel, saw, scales. Pointed or radial: star, quill, hammer. The tall group is the crowded one; inside it each symbol keeps one feature no other has (crenels, a capital, a round bottom, beams under a cap, a flame on a stick, curled ends, a zigzag).

## Reused drawings (14)

Duel's chain drawings, same id: `chain-barrel` (barrel), `chain-book` (book), `chain-column` (columns), `chain-drop` (drop), `chain-helmet` (helmet), `chain-horseshoe` (horseshoe), `chain-lamp` (lamp), `chain-lyre` (lyre), `chain-mask` (mask), `chain-target` (target), `chain-tower` (tower).

Duel's science drawings under a chain alias, in `sets.json` as `chain-mortar`, `chain-scales` and `chain-quill` with `file` pointing at `science-mortar.svg`, `science-scales.svg` and `science-quill.svg`. They are not science symbols in Haftgane; the chain chip gives them their role, and the alias belongs in the presenter, never in the engine's science enum. The chip uses the md file, as for every other chain glyph. Each has a `-sm` file from Duel's science set (`science-scales-sm` is the cleaner of the two scales at 12 px), which the presenter may use instead if the app's chip reading asks for it.

Acceptance inspection at the chip size is in the log. Pairs to watch: mortar beside lamp (both low bowls with something rising to the right), quill beside torch and hammer (diagonals), column beside tower and lighthouse.

## The ten new drawings

For each: the metaphor, the silhouette, and what it must differ from. Sizes and contexts as in [chain.md](chain.md): about 12 px in the mini card's chip, 14 to 16 px in the medium and band tiers, 20 px in sheets.

### chain-awning

- Metaphor: a market stall's striped awning.
- Silhouette: a wide canopy with a scalloped lower edge and broad stripes (knockouts filled with the detail tone), on two posts joined by a counter bar. The counter makes it a market stall rather than a roof on columns. 20 units wide.
- Must differ from: category-yellow (the purse: no tied neck, no bag), chain-temple-facade (no pediment, no row of columns), chain-column (the canopy is not a capital).

### chain-camel

- Metaphor: a dromedary, the caravan animal.
- Silhouette: one broad side view, a single hump, a long neck rising to a small head, legs simplified to two thick pairs.
- Must differ from: chain-helmet (a profile, but an animal with legs), chain-lamp (a low body with a rising neck).

### chain-hammer

- Metaphor: a builder's hammer.
- Silhouette: a heavy transverse head on a short diagonal handle; the head is the mass.
- Must differ from: chain-sword (no blade or crossguard), chain-saw (no teeth), category-grey (the anvil), chain-quill and chain-torch (other diagonals: the head is a block across the handle).

### chain-lighthouse

- Metaphor: a beacon tower, such as the Pharos.
- Silhouette: a narrow tower that tapers as it rises from a broad base, a gallery, a lantern under a pointed cap, and two wedge beams from the lantern's sides; a band across the tower is the detail tone. (Stepped tiers with the beams at their shoulders read as a robed figure with its arms out; see the log.)
- Must differ from: chain-tower (no crenels: a taper, beams and a cap), chain-column (no capital and base), chain-torch.

### chain-lightning

- Metaphor: a lightning bolt.
- Silhouette: a single thick angular bolt, a zigzag from the upper right to a point at the lower left.
- Must differ from: chain-torch (no handle, no flame), chain-quill (angular, not a feather).

### chain-planet

- Metaphor: a ringed planet.
- Silhouette: a large disc crossed by one tilted band that reaches beyond the disc on both sides, with no stand.
- Must differ from: science-globe (Duel's globe: a stand, a band inside the disc), chain-target (one band, not rings), chain-sun, coin.

### chain-saw

- Metaphor: a hand saw.
- Silhouette: a broad, short blade with three large teeth along its lower edge and a handle with a grip opening.
- Must differ from: chain-hammer, chain-sword, chain-mortar.

### chain-scroll

- Metaphor: a written scroll, standing.
- Silhouette: a tall blank sheet with a rolled top and a rolled bottom, the rolls curling the opposite ways so the ends show their spirals.
- Must differ from: resource-papyrus (Duel's horizontal roll with a roller on each side; the scroll is upright and its curls are asymmetric), chain-book (no spine), the science tablet of Haftgane (a flat slab with rounded shoulders, no rolls).

### chain-star

- Metaphor: a five-point star.
- Silhouette: a solid, broad five-point star with pointed tips.
- Must differ from: category-green (the eight-point star), chain-sun (no disc, no rays).

### chain-torch

- Metaphor: a hand torch.
- Silhouette: a straight upright handle with a cup at its top and one large flame, its tip leaning to one side.
- Must differ from: chain-lamp (no low spouted body), chain-lightning (a flame, not a zigzag), chain-drop (the flame stands on a handle), chain-lighthouse.

## Ids the inventory needs

The Haftgane catalog does not exist yet in the code. When it does, its 24 chain values map one to one onto the 24 ids above; the presenter owns that map.
