# Spec: Haftgane seat crests, seat colours and the ore and cloth tokens

Set name in `sets.json`: `haftgane-seat-crests`. Seven crest glyphs for Haftgane, the second game ([docs/haftgane/README.md](../../../docs/haftgane/README.md)), one per seat, plus the seven seat colours they are drawn in and the colour tokens of the ore and cloth resources. Read [../STYLE.md](../STYLE.md) first; it holds the grid, the paint convention and the loop. Slice M19.vectors-seat-crests, part of M19 "Art for Haftgane". Where the crests appear comes from the phone-layout design ([docs/haftgane/phone-layout.md](../../../docs/haftgane/phone-layout.md), the "Crest" term and "Assets") and its private mockups.

## Decided

- **Seven new crests, one per seat.** Duel's sun and moon keep their day and night meaning: Haftgane marks a wonder's day or night side with a sun or a moon ([art-reuse.md](../../../docs/haftgane/art-reuse.md)), so no crest is a sun or a moon. This is the roadmap's recommendation for M19 item 5.
- **Crests are fixed by seat position.** Seat 0 always wears crest-0, seat 6 crest-6. There is no lobby picker and so no collision handling.
- The crest sits inside a **ring** drawn by CSS, not by the SVG: a circle with a 1.5 px border in the seat colour and, in the mockups, a fill of 18% seat colour over the panel. The glyph is drawn in the seat colour too.

## Where the files live

- `web/src/lib/games/haftgane/icons/svg/crest-<n>.svg`, outside the directory that `web/src/lib/icons/sources.ts` globs, so none reaches Duel's bundle. Nothing imports them yet; M19.art-wiring does that.
- The colour tokens live in `web/src/lib/games/haftgane/theme.css`, which nothing imports yet either. `web/src/app.css` and `web/src/lib/styles/tokens.css`, which every Duel player downloads, do not change.

## Sizes, measured from the mockups

The phone-layout mockups (private kit, `kit.css` `.crest`) draw the glyph at about two thirds of its ring. Measured from the kit and checked on the 390x844 shot of seven players at DPR 2 (far chip rings 36 device px, neighbour panel rings 44):

- 11 px glyph in a 16 px ring: the seven crests of Haftgane's tile on Home.
- 12 px glyph in an 18 px ring: far chips, the pass flow in the hand header (giver's crest, hand, receiver's crest), the payment options' neighbour rows, the turn recap's lines, the military dialog's conflict lines and winner marks, the final score rows, Seen hands and the waiting and away messages. This is the size that matters most.
- 15 px glyph in a 22 px ring: neighbour panels, your own strip, the military dialog's head-to-head, the city sheet's ring tabs, the lobby's seats.
- 20 px in a 30 px ring: the city sheet's heading.
- 28 px in a 44 px ring: the final scores' verdict.

The blind check's 12 px DPR 2 and 16 px DPR 1 sheets bracket the two common sizes. A crest gets a `-sm` file only if its md drawing fails at 11 and 12 px.

## Shared rules of the set

- **Shape carries identity, colour only helps.** Seven players must tell the crests apart in greyscale, under every colour-vision deficiency and in plain ink. Every crest is a different silhouette family, recognisable at 12 px with `--i2` unset.
- **Keep the lower right free of meaning.** A bot's seat carries a 13 px bot badge over the lower right of its ring (far chips, neighbour panels, the lobby), so no crest may depend on what it draws there.
- **One tone.** A crest is drawn in the main tone only, so it reads as one solid emblem in its seat colour on the ring's light tint. A crest may cut one knockout (an eye, a hole) but never relies on it.
- **Fill the ring evenly.** A crest sits in a circle, so its mass is centred on (12, 12) and spread to about 18 to 20 units in both directions where the shape allows; a tall shape spans 19 to 20 units in height, a wide one 19 to 20 in width.
- **Not a symbol Haftgane already uses.** Crests share the screen with chain chips, science glyphs, wonder markers and conflict tokens, so none may look like one. Excluded, by subject and by silhouette:
  - Duel's crests and capitals: sun and moon (any rayed disc or crescent).
  - Haftgane's chains (spec [haftgane-chain.md](haftgane-chain.md)): barrel, book, column, drop, helmet, horseshoe, lamp, lyre, mask, target, tower, mortar, scales, quill, awning, camel, hammer, lighthouse, lightning, planet, saw, scroll, star (five points) and torch.
  - Haftgane's science and resources ([haftgane-resources-science.md](haftgane-resources-science.md)): compass, gear, tablet, ore and cloth; Duel's wood, clay, stone, glass and papyrus.
  - The conflict tokens, the pass mark and the ability glyphs (spec [haftgane-conflicts-passing-abilities.md](haftgane-conflicts-passing-abilities.md)): the intact and the split pennant on a staff, two cards over a horizontal arrow, the stepped ziggurat, the bays with a check, a card with a plus, a palette, a card against a bracket.
  - The category glyphs, which show beside seats in the city's colour counts: the mountain, the anvil, the domed gate, the eight-point star, the purse, the crossed swords and the crown. Also `hint-crown`, `token-frame`'s rosette, `vp-laurel`'s wreath, `wonder`'s gateway, `shield`, `coin`, `choice`'s Y and `military-pawn`.
- So no crest is a star, tower, sun, moon, gear, compass, tablet, scroll, book, lyre, crown, rosette, flag, shield or quadruped.

## The seven crests

Each crest is something of the game's world (the gardens, rivers and harbours of the ancient wonders) and a different silhouette family. The colour each wears is in "Seat colours".

### crest-0: lotus (rose)

- Metaphor: a lotus flower seen from the side, the flower of the Nile and of Persepolis' reliefs.
- Silhouette: a broad open lotus with no stem: one tall pointed petal in the middle, two broad petals leaning out on each side, the lowest nearly flat, all rising from a shallow bowl at the foot. Wide and full, a fan of petals opening upwards, never a band with points. (A cup on a stem read as a chalice, and a pointed fan on a stem as a cannabis leaf; see the log.)
- Must differ from: `hint-crown` and `category-purple`, the crowns (no band, no balls on the points: broad petals that rise from a bowl), `token-frame` (no rosette of round lobes), crest-1 the palm (petals rise from a bowl; the palm's fronds droop from a trunk).

### crest-1: palm (olive)

- Metaphor: a date palm, the tree of Babylon's gardens.
- Silhouette: a tall, slightly curving trunk that takes the lower half, and a crown of fronds that arch outwards and droop at their tips, the crown wider than the trunk is tall.
- Must differ from: `chain-torch` (no cup or flame: the crown hangs down), `chain-lighthouse` (no lantern or beams), `choice` (the fronds droop, the Y's arms rise), crest-0 the lotus.

### crest-2: scarab (violet)

- Metaphor: a scarab beetle seen from above, the emblem of Egypt's seals.
- Silhouette: a broad oval body split down the back into two wing cases, a wide shield-like head with a toothed front, and three short legs on each side.
- Must differ from: `coin` and `chain-target` (not a plain disc), `chain-mask` (no face), `military-pawn`, `shield` (legs, and a body split down the middle, rounded at the foot).

### crest-3: falcon (amber)

- Metaphor: a falcon standing in profile, the bird of Horus on Egypt's standards and seals.
- Silhouette: a bird perched and facing right, leaning forward: a round head with a hooked beak at the upper right, a full breast, the folded wing and a long tail sweeping down to the lower left, two short legs at the foot. Its mass runs on the diagonal from the upper right to the lower left; the legs, the only detail near the lower right, carry no identity.
- Must differ from: crest-6 the fish (upright and on legs, with a beak; the fish lies flat), `chain-drop` (a beak, a tail and legs; not a plain drop), `military-pawn` (it leans and has a beak; the pawn stands straight and round), `pass-hand`.
- Not an eagle displayed: wings spread to both edges read as a bat or an angel at 15 px and as a blurred W at 11 to 12 px (see the log), and the profile bird reads at every size.

### crest-4: wave (azure)

- Metaphor: a breaking wave of the sea that carries the trading ships.
- Silhouette: a full-width band of water along the foot and one tall wave rising from it whose crest curls over to the right into a hollow, its lip ending in a point.
- Must differ from: `crest-moon` and `chain-moon` (the curl stands on a band of water; a crescent stands alone), `chain-drop` (no closed drop), `chain-lightning`.

### crest-5: anchor (rust)

- Metaphor: a ship's anchor of the Mediterranean harbours.
- Silhouette: a ring at the top, a stock across under it, an upright shank, and two arms curving up from the foot of the shank into pointed flukes.
- Must differ from: `chain-sword` (the curved arms at the foot), `choice` (the arms curve up from the foot, not a fork at the top), `chain-horseshoe` (a cup with a shank, not an arch).

### crest-6: fish (teal)

- Metaphor: a fish of the river and the sea.
- Silhouette: a horizontal fish swimming to the right, a full lens-shaped body with a rounded head at the right, one fin on the back and a forked tail at the left, clear of the bot badge; the eye a knockout.
- Must differ from: Haftgane's conflict pennants (no staff; a curved body and a tail, not a flag with a notched fly), `chain-drop` (lying down, with a tail), crest-4 the wave.

## Seat colours

`--seat-0` to `--seat-6`, with a light and a dark value each, in `theme.css`. They colour the crest's ring and glyph and the outline of the player's own strip.

- **Distinct from each other in normal vision.** Measured as the OKLab distance between every pair in each theme. The order around the table is chosen so that seats that sit next to each other (with 3 to 7 players) are the most distinct pairs.
- **3:1 against the panel in both themes** (WCAG 1.4.11, a non-text element), and also against the sunken panel, the table and the ring's 18% tint, so the glyph holds on its own ring.
- **Not a card colour.** Duel's seven card colours share the screen (the city's colour counts sit on the same panels), so each seat hue sits between two category hues, and the log records each seat's nearest card colour.
- Identity never depends on colour alone: the crest's shape does that. Under colour-vision deficiencies some seat colours fall close together; the crests stay apart by shape.

## Ore and cloth tokens

`--res-ore` and `--res-cloth` (the tile fills) with `--res-ore-ink` and `--res-cloth-ink` (the theme-aware inks for the glyph straight on a page surface), light and dark, in `theme.css`, in the style of Duel's `--res-*` tokens in `tokens.css`.

- The resources-and-science slice drew ore on stone's tile and cloth on papyrus's tile as placeholders. Each new tile must be distinct from all five of Duel's tiles in normal vision.
- The glyph on the tile keeps 3:1 or more against the tile, in paper-bright or the card ink as `card-style.ts` picks for Duel's tiles. Each ink keeps 3:1 or more against every page surface of its theme.

## Acceptance

- **Blind check, 100%** at 12 px DPR 2 and 16 px DPR 1, by a fresh reader with one gloss line per name, among the neighbours below. Run twice, in colour and in `--mono`, because crests must be told apart by shape alone.
- **Neighbours** (`neighbours` in `sets.json`): `crest-sun`, `crest-moon`, `chain-star`, `chain-tower`, `chain-torch`, `chain-drop`, `chain-lighthouse`, `hint-crown`, `category-purple`, `vp-laurel`, `choice`, `chain-sword`, `conflict-victory`, `pass-hand` and `military-pawn`.
- **Colour vision:** a sheet of the seven crests in their rings at the mockup sizes, in both themes, in greyscale and under the three emulations, and a swatch sheet of the seven seat colours with their contrast ratios in both themes.
- Lint clean, each file within the 1228-byte budget, no change to Duel's icon budget or bundle. Family check passed.
