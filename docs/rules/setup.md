# Setup

This page covers what the game contains, how it is set up, how the players draft their wonders, and how each age's cards are laid out. Terms such as *structure*, *accessible* and *the box* are defined in the [README](README.md#terms-used-throughout).

## Components

- **73 building cards:** 23 for Age I, 23 for Age II, 20 for Age III, and 7 guilds that join Age III.
  - Age I: 6 brown, 2 grey, 4 red, 4 green, 4 yellow and 3 blue cards.
  - Age II: 3 brown, 2 grey, 5 red, 4 green, 4 yellow and 5 blue cards.
  - Age III: 5 red, 4 green, 5 yellow and 6 blue cards. Age III has no brown or grey cards, so in Age III the only way to gain new fixed production is to rebuild a discarded brown or grey card with the Mausoleum.
- **12 wonders.**
- **10 progress tokens.**
- **4 looting tokens** on the military track: on each side, one that costs its victim 2 coins and one that costs 5.
- **1 conflict pawn** on the military track.
- **Coins.** The physical game has a limited number of coin pieces, but Ajayeb treats the bank as unlimited (see [decisions.md](decisions.md#unlimited-coins)).

## Setup steps

1. Put the conflict pawn on the centre space of the military track and the four looting tokens on their spaces (see [military-science.md](military-science.md#the-track-and-its-zones)).
2. Shuffle the ten progress tokens. Five go face up on the board. The other five go to the box unseen; the Great Library is the only way to reach them later.
3. Give each player 7 coins.
4. Pick the **first player**. Ajayeb picks at random unless the room was created with a fixed choice.
5. Draft the wonders (next section).
6. Prepare the three age decks:
   - Shuffle the Age I deck and remove 3 cards unseen. The other 20 form the Age I deck.
   - Do the same for Age II.
   - For Age III, remove 3 of the 20 Age III cards unseen, then draw 3 of the 7 guilds at random and shuffle them in, for 20 cards. The other 4 guilds go to the box unseen.
7. Lay out the Age I structure. The first player takes the first turn of Age I.

The removed cards are never seen during play. So even a player who remembers every card cannot be sure which cards of an age will appear.

## The wonder draft

The twelve wonders are shuffled. The draft uses eight of them, in two rounds of four, and the other four are never seen.

- **Round 1:** four wonders are shown. The first player takes one, the second player takes two, and the first player receives the last one.
- **Round 2:** four new wonders are shown. This time the second player takes one, the first player takes two, and the second player receives the last one.

```
round 1:  first player, second player, second player, first player (automatic)
round 2:  second player, first player, first player, second player (automatic)
```

Each player ends the draft with four wonders. The four wonders of round 2 stay hidden until round 1 is over, and the Age I structure is laid out only when the draft ends (see [decisions.md](decisions.md#hidden-until-needed)).

## Card structures

Each age lays its 20 cards out in a different shape. Rows are numbered from the top. Columns are counted in half-card steps, so two neighbouring cards in a row are two columns apart, and a card in the row below sits half a card to the side.

**The overlap rule.** The card at row *r*, column *c* is overlapped by any card still present at row *r* + 1, column *c* - 1 or *c* + 1. A card is **accessible** when neither of those spaces holds a card. Taking cards from the lower rows therefore frees the cards above them.

**Face-up and face-down rows.** Rows 1, 3, 5 and 7 are dealt face up, and rows 2, 4 and 6 face down. Every age has 12 face-up and 8 face-down cards.

In the diagrams below, `[up]` is a card dealt face up and `[##]` a card dealt face down.

### Age I: a pyramid, wide at the bottom

```
row 1              [up]  [up]
row 2           [##]  [##]  [##]
row 3        [up]  [up]  [up]  [up]
row 4     [##]  [##]  [##]  [##]  [##]
row 5  [up]  [up]  [up]  [up]  [up]  [up]
```

The six cards of row 5 are accessible at the start. For example, taking the two leftmost cards of row 5 frees the leftmost card of row 4, which is then turned face up.

### Age II: an upside-down pyramid, narrow at the bottom

```
row 1  [up]  [up]  [up]  [up]  [up]  [up]
row 2     [##]  [##]  [##]  [##]  [##]
row 3        [up]  [up]  [up]  [up]
row 4           [##]  [##]  [##]
row 5              [up]  [up]
```

Only the two cards of row 5 are accessible at the start. The cards at the ends of row 1 are each overlapped by a single card of row 2.

### Age III: two pyramids joined in the middle

```
row 1              [up]  [up]
row 2           [##]  [##]  [##]
row 3        [up]  [up]  [up]  [up]
row 4           [##]        [##]
row 5        [up]  [up]  [up]  [up]
row 6           [##]  [##]  [##]
row 7              [up]  [up]
```

Only the two cards of row 7 are accessible at the start. Row 4 holds just two face-down cards with a gap between them. The left one alone overlaps the two left cards of row 3, and the right one alone overlaps the two right cards of row 3.

The three guilds can land anywhere in the Age III structure. A guild in a face-down space shows the guild back instead of the Age III back, so both players can see that a guild is there, but not which one (see [decisions.md](decisions.md#guild-back)).

## What is hidden from both players

- The cards removed from each age deck, and the decks of the ages still to come.
- The face-down cards in the structure (only their backs are visible).
- The four guilds, four wonders and five progress tokens in the box. Which tokens are in the box can be worked out by elimination, as at a real table, but not their order.
- The round 2 wonders, until round 1 of the draft is over.

The only secret that one player holds and the other does not is the Great Library offer (see [wonders-tokens.md](wonders-tokens.md#the-great-library)). When the game ends, Ajayeb reveals the removed cards and the contents of the box (see [decisions.md](decisions.md#reveal-at-game-end)).

## For developers

- **All chance is resolved when the game is created**, from its seed: the wonder order (eight in draft order, four for the box), which tokens go on the board and the order of the five in the box, the three age deals and their removed cards, the guild draw, and the first player. Applying actions never uses randomness, so a game is fully described by its seed, its options and its action keys.
- **Draft keys** have the form `draft:<wonder>`, for example `draft:pyramids`. The legal draft keys are the wonders of the current round that are still on offer. The automatic last pick of each round is made by the engine itself; no key is sent for it.
- **Layout data** gives each of an age's 20 spaces its row and half-card column, exactly as in the diagrams above. A space records its back as well as its card, so a face-down space can be shown without its card.
- **Structure checks** that the tests pin down: 12 face-up and 8 face-down spaces per age; 6, 2 and 2 accessible cards at the start of Ages I, II and III; and in Age III, each card of row 3 is overlapped only by the row 4 card on its side.
