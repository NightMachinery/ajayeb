# Spec: science symbols

Set name in `sets.json`: `science`. 14 files: seven symbols, each with an `-sm` variant. Read [../STYLE.md](../STYLE.md) first. Acceptance: blind identification 100% at 14 px DPR 2, and no pair confusable at 12 px. Pairs are the core mechanic of science (two equal symbols take a progress token, six different ones win the game), so a player must match them at a glance across the board.

## Shared rules of the set

- In the game the symbols come from green cards (each of six symbols on two cards) and from the Law progress token (the scales). The engine's ids are `globe`, `scales`, `sundial`, `mortar`, `set-square`, `quill` and `wheel` (`ScienceSymbol` in `web/src/lib/wire/view.gen.ts`).
- They are drawn in the green card colour with `--i2` paper-bright, on the paper inset of the effect band, and in plain ink in the science tracker of the city panel.
- All seven share one construction (the same weight, the same optical size), and each has one dominant, distinct silhouette. Avoid three of them being "a disc with lines".
- The `-sm` variant is for the mini card (about 12 px at 1.2 em of a 10 px font) and the compact tracker (12 px).
- These are scholars' instruments: draw them as objects of a manuscript workshop, never as modern pictograms.

## science-globe

- Meaning: the globe symbol (astronomy and geography).
- Metaphor: a celestial globe on a short stand, or an armillary sphere with one ring.
- Silhouette: a disc of 16 to 17 units on a small foot, with one tilted meridian ring as the detail tone.
- Must differ from: science-wheel (no spokes, the foot is the key), coin, crest-sun, chain-target.
- Sizes: 12 px (sm), 14 to 17 px (md), 24 px in sheets.
- Contexts: EffectGlyph science (card-green, `--i2` paper-bright), ScienceTracker, the science pair hint in sheets.

## science-scales

- Meaning: the scales symbol (law); it comes only from the Law progress token.
- Metaphor: a balance with a beam and two hanging pans.
- Silhouette: a T-shaped post with a horizontal beam and two pans hanging low on both sides, wider than tall; the pans are the detail tone or solid bowls.
- Must differ from: science-set-square, the Lucide balance icon (do not copy it), trade.
- Sizes and contexts: as science-globe.

## science-sundial

- Meaning: the sundial symbol (timekeeping).
- Metaphor: a sundial: a half disc dial with a triangular gnomon casting a shadow line.
- Silhouette: a flat half disc lying on its straight edge, with a raised triangular gnomon. The hour lines in the detail tone, at most three.
- Must differ from: crest-sun (no rays around a disc), science-set-square (it is curved, the set square is straight), chain-sun.
- Sizes and contexts: as science-globe.

## science-mortar

- Meaning: the mortar and pestle symbol (medicine and chemistry).
- Metaphor: a mortar bowl with a pestle leaning out of it.
- Silhouette: a heavy bowl on a foot, the pestle rising diagonally out of the top right. Move the mass up half a unit (STYLE.md section 2).
- Must differ from: resource-glass (no neck), chain-lamp, chain-amphora.
- Sizes and contexts: as science-globe.

## science-set-square

- Meaning: the set-square symbol (geometry and building).
- Metaphor: a builder's square, or a triangle set square with a plumb line.
- Silhouette: a right-angled triangle or an L-shaped square with a triangular knockout, its right angle at the bottom left.
- Must differ from: science-sundial (straight edges only), science-scales, category-brown.
- Sizes and contexts: as science-globe.

## science-quill

- Meaning: the quill symbol (writing and literature).
- Metaphor: a reed pen or feather quill, with an inkpot.
- Silhouette: a long diagonal feather from bottom left to top right, the vane as a broad leaf shape with a split, the nib as a point. An inkpot at its foot is optional and helps at 12 px.
- Must differ from: chain-sword (no crossguard, a broad vane), resource-papyrus, chain-book.
- Sizes and contexts: as science-globe.

## science-wheel

- Meaning: the wheel symbol (mechanics and engineering).
- Metaphor: a spoked cart wheel.
- Silhouette: a ring of 18 to 19 units with a solid hub and four to six spokes; at 12 px the `-sm` file keeps four spokes and a fat rim.
- Must differ from: science-globe (spokes, no foot), chain-gear (a smooth rim, no teeth), chain-sun, chain-target.
- Sizes and contexts: as science-globe.

## Ids the inventory needs that names.ts lacks

None for this set.
