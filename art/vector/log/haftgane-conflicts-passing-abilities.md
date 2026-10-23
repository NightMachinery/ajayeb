# haftgane-conflicts-passing-abilities: iteration log

## Status

- conflict-victory: done (iteration 1)
- conflict-defeat: done (iteration 2); the weakest reading of the set: at 16 px DPR 1 the fallen half shrinks to a chevron, which two readers saw as a K or an hourglass before settling on a torn flag
- pass-hand: done (iteration 1)
- wonder-stage: done (iteration 2)
- wonder-complete: done (iteration 1)
- play-last-card: done (iteration 4); metaphor changed from the spec's first proposal, see the spec
- free-first-colour: done (iteration 4); metaphor changed from the spec's first proposal, see the spec
- free-age-boundary: done (iteration 1)

Blind check: 16/16 for the eight new glyphs, from each of two fresh readers in turn (checks 3 and 4), against the nine Duel neighbours. Every file is lint clean and between 211 and 881 bytes. Duel's icon source set is unchanged (27252 bytes raw, 5638 gzipped in the specimen envelope).

## Iteration 0 (2026-10-23): fast-render sketches before the first specimen

Renders: art-work/vector/haftgane-conflicts-passing-abilities/fast-1/ to fast-4/ and ws-var/ (librsvg strips from `fast.mjs`, 12 to 96 px, paper and night). The Duel neighbours were rendered beside them (neigh/) to check against.

Every icon was first drawn from a one-sentence idea:

- conflict-victory: a staff with a round finial, a broad swallowtail pennant flying right, a lozenge emblem knocked out of it (the detail tone when set).
- conflict-defeat: the same staff; the pennant torn across, the hoist half left on the staff with a jagged edge, the notched fly half broken off and hanging tilted down.
- pass-hand: two fanned cards over a heavy arrow that runs the full width and points right.
- wonder-stage: a stepped construction bay with the end of a tucked card showing beneath it.
- wonder-complete: three equal bays on one plinth, a heavy check above.
- play-last-card: a card low on the left, a short end post on the right, the second card lifted up over the post and overlapping the first card's corner.
- free-first-colour: three positions, the first an empty slot with a new card dropping into it, the other two holding cards.
- free-age-boundary: a heavy bracket-shaped end-stop the full height of the icon, one card standing against its open side.

What the strips showed, and what changed before the first specimen:

- conflict-victory, pass-hand, wonder-complete, free-age-boundary: read as intended from the first sketch at every size. No change.
- conflict-defeat: the first fallen half (tilted 22 degrees, three tear teeth) looked splintered rather than fallen. Now two broad tear teeth and a 34 degree tilt with a further drop, so the fly half hangs clearly below the hoist half.
- wonder-stage: the first two-tier block with a narrow card end under it read as a cross or a mushroom at 12 to 16 px, and beside Duel's anvil (category-grey, a T shape) it was too close. A wider, flatter two-tier block over a slanted card read as a hat. Four alternatives were drawn side by side (ws-var/): a three-step ziggurat over a slanted card, the same with a doorway, a stepped bar over a hanging card (a T again), and a ziggurat on a plain block (a pedestal). The three-step ziggurat with the tucked card slanting out below is the most clearly a monument tier and keeps clear of the T; kept.
- free-first-colour: the first row of three thin cards (one hollow) with a small card above read as a bar chart. A rotated three-card fan read as a blob at 12 px. Now a full-height left column (the new card above a U-shaped empty slot) beside a pair of overlapping cards, so the empty slot keeps a visible opening at 12 px.

An authoring trap found on the way: a clip path referenced by an element that has its own `transform` is evaluated in that element's rotated space, so a rotated back card clipped by the front card's outline got a skewed gap. Each clipped, rotated card is now wrapped in a `<g clip-path>`, which keeps the clip in the icon's own space.

## Iteration 1 (2026-10-23)

Renders: art-work/vector/haftgane-conflicts-passing-abilities/iter-1/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the three peeps).
Lint: clean (0 errors, 0 warnings). Files 211 to 561 bytes.
Looked at: light@1x, peep-light@1x, peep-light@2x.

### conflict-victory

- Seen: a flag at every size from 12 px, on paper, night and all seven bands; the lozenge shows from 14 px and is a dot at 12 px, which does no harm. No change.

### conflict-defeat

- Seen: reads as a torn flag from 16 px, but at 12 px DPR 1 the break between the two halves closes to a grey seam, so it is a ragged flag rather than a broken one, too close to the victory flag.
- Changed: one tear tooth instead of two; the fly half dropped 3.5 units and tilted 20 degrees about its own corner, so the narrowest gap is about 1.8 units and the widest about 4. Three tilts were compared first (ws-var/dA to dC); 20 degrees keeps the swallowtail recognisable and stays inside the live area.

### pass-hand

- Seen: two cards and an arrow at every size; the back card is a sliver at 12 px but the pair still reads as a hand. No change.

### wonder-stage

- Seen: the three-step ziggurat over a slanted card end reads as a bell or a mushroom at 12 to 16 px; the card end is too small to be a card.
- Changed: the card moves beside the ziggurat: a full portrait card slid under its lower right corner (the ziggurat clipped 1.5 units around it), so the icon says "a stepped monument with a card tucked under it". Three more layouts were compared first (ws-var/wC to wE): a card end under the base (too small again) and a stage block with a card standing in a slot (an "A").

### wonder-complete

- Seen: three bays and a check at every size; the check stays clear of the bays at 12 px. No change.

### play-last-card

- Seen: a low card, a lifted card and the small post; the post is a one-pixel stroke at 12 px but stays visible. No change until the blind check says otherwise.

### free-first-colour

- Seen: the U-shaped slot keeps its opening at 12 px DPR 1; the whole is busy (five parts) but no part blobs. No change until the blind check says otherwise.

### free-age-boundary

- Seen: the clearest of the set, a bracket and a card at every size. No change.

## Iteration 2 (2026-10-23)

Renders: art-work/vector/haftgane-conflicts-passing-abilities/iter-2/ (light and dark at DPR 1 and 2, grey, deuteranopia, protanopia, tritanopia, and the three peeps).
Lint: clean (0 errors, 0 warnings). Files 211 to 567 bytes.
Looked at: light@2x, peep-light@2x, and grey, deuteranopia, protanopia and tritanopia side by side.

### conflict-defeat

- Seen: the break stays open at 12 px DPR 1 and DPR 2, on paper and night and on every band. The fallen half reads as a chevron at 12 px and as a notched pennant from 20 px; beside the intact victory flag the two never look alike, in greyscale or any colour-vision pass. Done.

### wonder-stage

- Seen: a stepped ziggurat with a card slid under its corner from 16 px; at 12 px a staircase and a card. Nothing like the Duel gateway (`wonder`), the anvil or the mountain in the family sheet. Done pending the blind check.

### The rest

- Seen: unchanged from iteration 1 and still clear. Every icon is drawn in one ink, so the greyscale and colour-vision passes match the light column exactly; no meaning rests on colour.

## Family check (2026-10-23)

Renders: art-work/vector/haftgane-conflicts-passing-abilities/family/ (this set beside resources-economy-military, ui-category, tt2-glyphs and chain, 16 and 24 px, paper and night).

- The weight and density match `build-from-discard`, `destroy-card` and the chain symbols; the cards share one corner radius with Duel's card glyphs.
- play-last-card is the lightest of the set (two cards and a thin post leave the most empty space), but not an outlier beside `play-again` or `chain-lamp`. No change.
- The wonder-stage ziggurat and category-brown's mountain sit in different rows here; the steps and the card keep them apart at 16 px.

## Blind check 1 (2026-10-23)

Sheets: 12 px DPR 2 and 16 px DPR 1, the eight icons of iteration 2 shuffled with the nine Duel neighbours (17 tiles a sheet), in each icon's specimen ink. Key: art-work/vector/blind/haftgane-conflicts-passing-abilities-2026-10-23T07-27-02-130Z/.
Reader: a fresh in-process Opus subagent that had never seen the labelled specimen, given only the two sheets, `names.txt` and a one-line meaning per name (what each stands for in the game, nothing about its look), and told to name each tile on its own.
Score: all tiles 12 px 16/17, 16 px 16/17; the eight new glyphs 14/16 (87.5%), under the 90% pass mark. Miss: play-last-card read as destroy-card on both sheets.

- play-last-card: "a standing card, a second card tilted up and away, and a small upside-down T where it seems to have stood": a card knocked out of a city. The tilt reads as falling, not as being played, and the post reads as a stand.
- Right but uncertain: free-first-colour (low: "colour columns where the first slot matters"), free-age-boundary (medium-low: "a card at the start of a span"), conflict-defeat (medium), wonder-stage (medium: "a stepped pyramid with a card tucked against its lower right"). conflict-victory, pass-hand and wonder-complete were medium-high to high.
- The reader also hesitated between coin-loss and conflict-defeat for the cracked disc; it named both correctly.

## Iteration 3 (2026-10-23)

### play-last-card

- Seen: the blind miss above; at 12 px the tilted card and the post look like an accident, not an extra play.
- Changed: both cards upright, nothing tilted: one card low on the left, a dashed end line the full height of the icon (four dashes of 2 by 3, so it reads as a boundary and never as the heavy bracket of free-age-boundary), and the second card raised beyond the line. The idea in one sentence: one more card, played after the Age's end line. Three layouts were compared first (ws-var/pA to pC); overlapping the cards again with two short posts under the raised one read as a card on a stand.

## Blind check 2 (2026-10-23)

Sheets: as blind check 1, with iteration 3's play-last-card. Key: art-work/vector/blind/haftgane-conflicts-passing-abilities-2026-10-23T07-32-23-176Z/.
Reader: a second fresh in-process Opus subagent (not the first reader), same instructions and meaning lines.
Score: all tiles 12 px 15/17, 16 px 15/17; the eight new glyphs 12/16 (75%). Misses: play-last-card read as free-age-boundary on both sheets ("a card, a dashed line, a second card higher on the far side: the last and first turn either side of an Age boundary"); free-age-boundary read as free-first-colour on both sheets ("a bracket with a card: the first card of a group").

What the two readings show together: the three timing abilities (play-last-card, free-first-colour, free-age-boundary) all speak of a first or a last, and all three drawings were a card in a position, so each reader matched them by the word "first" or "last" and not by the drawing. The bracket did read as "the start of a span" both times; it lost only because free-first-colour also says "first". The fix is a cue each meaning owns alone.

## Iteration 4 (2026-10-23)

### play-last-card

- Changed: one upright card and a heavy plus at its upper right (arms 3 units thick and 9 long, 1.5 units clear of the card): "one more card". Two cards and a plus was also tried (ws-var/p2) and read as "copy".

### free-first-colour

- Changed: a painter's palette (kidney outline, thumb hole, three paint wells as knockouts with the detail tone) with a card slid under its lower right, the layout of wonder-stage with another subject. A palette alone (ws-var/f2) reads even more clearly but loses the link to a building; kept the card. At 12 px DPR 1 the wells are single pixels but the round outline and the card carry it.

### free-age-boundary

- Seen: unchanged. With colour now owned by the palette, the bracket's "start of a span" should land on the Age.

Spec: play-last-card and free-first-colour metaphors corrected (the spec says why).

## Blind check 3 (2026-10-23)

Sheets: as before, with iteration 4's drawings. Key: art-work/vector/blind/haftgane-conflicts-passing-abilities-2026-10-23T07-38-26-611Z/.
Reader: a third fresh in-process Opus subagent (neither earlier reader), same instructions and meaning lines.
Score: all tiles 12 px 17/17, 16 px 17/17; the eight new glyphs 16/16 (100%), over the 90% pass mark.

- High: pass-hand; free-first-colour at 12 px ("a painter's palette with paint dots next to a card"), medium at 16 px DPR 1, where the wells fade.
- Medium to medium-high: conflict-victory, conflict-defeat (the fallen half "looks like a stray K" until you see it has broken off), wonder-complete.
- Medium-low: play-last-card ("one extra card play"; play-again was its other thought), free-age-boundary ("the card on the boundary of an Age"; the bracket says a start more than an end).
- Low: wonder-stage ("a card tucked under a stepped pyramid, which is how a stage gets built", but at 16 px DPR 1 the steps blur to a heap that could be a pile of cards).

## Blind check 4 (2026-10-23): confirmation

Sheets: the same drawings as blind check 3, shuffled with a new seed. Key: art-work/vector/blind/haftgane-conflicts-passing-abilities-2026-10-23T07-40-30-262Z/.
Reader: a fourth fresh in-process Opus subagent (none of the earlier readers), same instructions and meaning lines. Run because the first two readers disagreed with each other and the third named three glyphs with low confidence; one reading is one sample.
Score: all tiles 12 px 17/17, 16 px 17/17; the eight new glyphs 16/16 (100%).

- Medium-high: free-first-colour ("the palette reads as colour"). Medium: conflict-victory, wonder-complete, wonder-stage. Low-medium: free-age-boundary, play-last-card. Medium-low: conflict-defeat ("the detached piece also reads as an hourglass", "a < chevron at 16 px").
- Two readers in a row named every tile; the weak readings repeat, though: the defeat flag's fallen half is small at 16 px DPR 1, and the bracket and the plus each say their idea only weakly without the label. In the app all three always sit beside live text (the signed value, the stage's tooltip), which the blind sheets leave out on purpose.

## Family check, final drawings (2026-10-23)

Renders: art-work/vector/haftgane-conflicts-passing-abilities/family-final/ (the same sets as the first family check).

- The palette and the card with its plus, new since the first family check, match the weight and size of the rest of the set and of the Duel card glyphs; the palette's wells in the detail tone are no noisier than the chain book's lozenge. No outlier.
