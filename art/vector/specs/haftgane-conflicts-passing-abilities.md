# Spec: Haftgane conflict tokens, the pass mark and the wonder ability glyphs

Set name in `sets.json`: `haftgane-conflicts-passing-abilities`. 8 files, all md (24 grid), no `-sm` variants. Read [../STYLE.md](../STYLE.md) first. Slice M19.vectors-conflicts-passing-abilities, part of M19 "Art for Haftgane". The subjects come from [docs/haftgane/art-reuse.md](../../../docs/haftgane/art-reuse.md) ("Military: reuse shields, add conflict tokens", "Categories, passing and backs" and "Ability glyphs beyond the requested core sets"); the rules behind them from [docs/haftgane/rules.md](../../../docs/haftgane/rules.md).

## Shared rules of the set

- **Where the files live.** `web/src/lib/games/haftgane/icons/svg/<id>.svg`, outside Duel's icon directory, so nothing here reaches a Duel player's bundle. No screen uses them yet; M19.art-wiring does that. Each `sets.json` entry names its file.
- **Reused, not redrawn.** Strength stays Duel's `shield`; the Halikarnassos discard ability stays Duel's `build-from-discard`; whole-wonder context stays `wonder`. This set draws only what Duel has no glyph for.
- **No baked numbers or words.** A conflict token's signed value (+1, +3, +5, -1), a turn number, a colour name and a neighbour's name are live, translated text beside the glyph, never outlines in the SVG.
- **Colour never carries meaning.** Victory and defeat, and the two pass directions, must read apart in greyscale and as plain ink. The specimen draws every icon in the panel ink (`--fg`) or the card ink, so only the shape counts.
- **Main tone first.** Each icon is recognisable with `--i2` unset. Where a detail tone appears it is one secondary layer, never the only difference between two icons.
- **One card shape.** Every card in this set (pass-hand, the three timing abilities, the card tucked under the ziggurat and the palette) is the same portrait rounded rectangle, corners 1 unit. The card glyphs differ by the one thing beside the card (an arrow, a plus, an end-stop, a palette, a ziggurat), never by drawing the card differently. The three timing abilities all speak of a first or a last, so each carries a cue that belongs to it alone: the plus for an extra card, the palette for colour, the bracket for the start of an Age.
- **Sizes.** 12 to 16 px on a card's effect band, a wonder stage tile and a tooltip line; 16 to 24 px for a conflict token beside its value in the city panel and the Age-end summary; 14 to 20 px for the pass mark beside the hand. The blind check uses 12 px at DPR 2 and 16 px at DPR 1.
- **Confusable neighbours** (the blind check's extra candidates, `neighbours` in `sets.json`): `shield`, `chain-helmet`, `coin-loss`, `destroy-card`, `trade`, `play-again`, `build-from-discard`, `wonder` and `badge-check`.

## Acceptance

- Lint clean, each file within the 1228-byte budget, no aggregate change to Duel's icon budget.
- **Blind check, pass mark 90%:** at least 15 of the 16 tiles of the eight new glyphs (8 per sheet, sheets 12 px DPR 2 and 16 px DPR 1) named correctly by a fresh reader who sees only the sheets and the candidate list with one meaning line per name, and no new glyph missed on both sheets. A neighbour tile answered with a new glyph's name also counts as a failure of that new glyph, because it is the same confusion seen from the other side.
- Victory and defeat distinct in greyscale and under every colour-vision emulation at 12 px; the pass mark legible mirrored.
- Family check passed: no stroke, size or density outlier beside the Duel sets.

## conflict-victory

- Meaning: a military victory token from an Age-end conflict with a neighbour (worth 1, 3 or 5 points by Age; the value is live text).
- Metaphor (art-reuse.md): an intact, broad, notched pennant.
- Silhouette: a staff with a round finial on the left, and a broad swallowtail pennant flying from its top to the right, its fly end cut by a deep V notch. The pennant is about two thirds of the icon's width and half its height, so it reads as a flag at 12 px; one emblem as the detail tone, a knockout when unset.
- Must differ from: shield (no heater outline), chain-helmet, conflict-defeat (intact, level), vp-laurel.
- Contexts: the conflict token component, beside its signed value; the Age-end conflict summary; tooltips.

## conflict-defeat

- Meaning: a military defeat token (minus 1 point in every Age; the value is live text).
- Metaphor (art-reuse.md): the same pennant split, with a large open break.
- Silhouette: the same staff and finial. The pennant is torn across: the hoist half stays on the staff with a jagged edge, and the notched fly half has broken away and hangs tilted down beyond a wide, open gap (at least 1.5 units at its narrowest and about 4 at its widest, so the break stays open at 12 px). The tilt of the fallen half, not only the gap, carries the reading.
- Must differ from: conflict-victory (by the gap and the fallen half, without colour), destroy-card (a card split top to bottom, no staff), coin-loss (a split disc: a defeat must not look like a coin penalty), shield.
- Contexts: as conflict-victory.

## pass-hand

- Meaning: the direction in which hands pass this Age (to the left neighbour in Ages I and III, to the right in Age II); the receiving neighbour's name is live text.
- Metaphor (art-reuse.md): two overlapping cards and one heavy horizontal arrow.
- Silhouette: two fanned cards above a heavy arrow that runs the full width and points right. The caller mirrors the icon (`scale(-1, 1)`) for the other direction, so the drawing must read the same mirrored and must not lean on anything that only works one way round. Which way it points comes from the stable seat ring and the server's pass state, never from the document direction.
- Must differ from: trade (two opposed arrows and no cards), play-last-card, build-from-discard, play-again.
- Contexts: the hand header and the Age banner, 14 to 20 px, mirrored as needed; the Age-start tooltip. With Animation off the static arrow is the whole message.

## wonder-stage

- Meaning: one built stage of a wonder; used where an effect counts stages (Arena, Builders Guild) and for board progress.
- Metaphor (art-reuse.md): one stepped construction bay, with a card tucked under it.
- Silhouette: a stepped ziggurat (flat top, square steps, each step a stage) across the upper three quarters, and one whole card slid under its lower right corner at a slant, the ziggurat cut 1.5 units around the card. (Corrected in iteration 2: a stepped block with only the end of a card showing beneath it read as a cross, a mushroom or a hat, and as a T beside Duel's anvil; the card end was too small to be a card.) No arch: the arch belongs to `wonder`.
- Must differ from: wonder (the whole gateway with its arch), wonder-complete (bays and a check, no card), build-from-discard (card over a pile), category-grey (the anvil, a T), category-brown (the mountain: steps, not peaks).
- Contexts: effect lines that count stages, the stage strip of a wonder board, tooltips.

## wonder-complete

- Meaning: every stage of your wonder built; used by completion scoring (Decorators Guild).
- Metaphor (art-reuse.md): a short row of stage bays capped by a check.
- Silhouette: three equal solid bays standing on one plinth across the bottom half, and a broad check mark above them. The check is as heavy as `badge-check`'s and sits clear of the bays.
- Must differ from: badge-check (the bays), wonder-stage (three bays and a check, no tucked card), wonder.
- Contexts: the Decorators Guild's effect line, tooltips.

## play-last-card

- Meaning: Babylon's night ability: keep the last card of each Age instead of discarding it, and play it as an extra turn.
- Metaphor: one more card. (Corrected in iteration 4. art-reuse.md proposed "two overlapping cards, one lifted beyond a small end marker". Drawn with a tilted card over a post, a blind reader took it for a card knocked out of a city; drawn upright beyond a dashed end line, another took it for the Age-boundary ability. Not Duel's looping play-again arrow either, which suggests another timing.)
- Silhouette: one upright card on the left and a heavy plus at its upper right, clear of it by 1.5 units. The plus says "an extra card", the only glyph of the set with one.
- Must differ from: play-again (no loop), pass-hand (no arrow), free-age-boundary (no stop or boundary line), build-from-discard (no pile, no arrow).
- Contexts: Babylon's night stage tile and its tooltip.

## free-first-colour

- Meaning: Olympia's day ability: the first building of each colour you construct is free. The colours themselves are composed beside it from the category swatches and live text.
- Metaphor: a painter's palette with a card. (Corrected in iteration 4. art-reuse.md proposed "a single card entering the first empty position in a three-card fan". Drawn as a card dropping into an empty slot beside two cards, both blind readers read "the first of something", and the second gave it to the Age-boundary meaning; nothing in it said colour, which is what sets this ability apart.)
- Silhouette: a kidney-shaped palette with a thumb hole and three paint wells (knockouts, the detail tone when set), and one card slid under its lower right corner at a slant, the palette cut 1.5 units around it.
- Must differ from: wonder-stage (the same "subject plus tucked card" layout, but a round palette, not a stepped ziggurat), destroy-card (whose meaning also names a colour: no crack), category glyphs.
- Contexts: Olympia's day stage tile, its tooltip, the colour swatches it composes with.

## free-age-boundary

- Meaning: Olympia's night abilities: the building you construct on the first turn of each Age (or, mirrored, on its last turn) is free. Which turn it is comes from the live label.
- Metaphor (art-reuse.md): a card beside a heavy end-stop. Mirror for first and last turn.
- Silhouette: a heavy bracket-shaped end-stop, the full height of the icon, and one card standing right against it on its open side. Drawn as the first-turn form (the stop on the left); the caller mirrors it for the last turn. No turn numbers in the shape.
- Must differ from: play-last-card (one card, a heavy full-height stop), free-first-colour, badge-lock, category glyphs.
- Contexts: Olympia's night stage tiles and their tooltips.

## Ids the inventory needs that names.ts lacks

None. These ids are Haftgane's and are not in Duel's `names.ts`; Haftgane's own icon list arrives with M19.art-wiring.
