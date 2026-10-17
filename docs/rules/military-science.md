# Military and science

This page covers the military track, its zones and looting tokens, military supremacy, the science symbols and pairs, the Law token, and scientific supremacy. Terms such as *pair*, *supremacy* and *weaker player* are defined in the [README](README.md#terms-used-throughout).

## The track and its zones

The military track is a single row of 19 spaces: a centre space and nine spaces on each side. The ninth space on each side is that player's **capital**. The conflict pawn starts on the centre space.

Each **shield** a player gains moves the pawn one space toward the opponent's capital. Both players push the same pawn, so its position shows the difference between their shields so far. When the pawn stands on your side of the centre, your opponent is ahead, and you are the **weaker player**.

Spaces are counted from the centre, 1 to 9 on each side, and form zones:

- **Spaces 1 and 2:** worth 2 victory points at the end of the game to the player who is ahead.
- **Spaces 3 to 5:** worth 5 points. A 2-coin looting token sits at the start of this zone.
- **Spaces 6 to 8:** worth 10 points. A 5-coin looting token sits at the start of this zone.
- **Space 9, the capital:** reaching it wins the game at once.

```
space    9   8   7   6   5   4   3   2   1   0   1   2   3   4   5   6   7   8   9
points  win  10  10  10  5   5   5   2   2   0   2   2   5   5   5   10  10  10 win
looting              -5          -2                      -2          -5
         ^                                   ^                                   ^
    A's capital                           centre                            B's capital
```

In the diagram, player A's shields push the pawn to the right, toward B's capital, and player B's shields push it to the left.

## Where shields come from

- **Red buildings:** every Age I red card has 1 shield. In Age II, Horse Breeders and Barracks have 1, and Walls, Archery Range and Parade Ground have 2. In Age III, Circus, Siege Workshop and Fortifications have 2, and Arsenal and Pretorium have 3.
- **Wonders:** the Colossus has 2 shields; the Statue of Zeus and Circus Maximus have 1 each.
- **The Strategy token:** each red building you construct after taking it gives 1 extra shield. This includes a red building rebuilt with the Mausoleum. It does not apply to wonders or to red buildings already in your city.

## Looting tokens

When the pawn **enters space 3 or space 6 on a player's side** for the first time, and that space's looting token is still there, that player loses 2 or 5 coins, and the token leaves the game for good.

- The coins go to the bank. The player who moved the pawn gains nothing from looting.
- A player who has fewer coins loses all they have and no more.
- A token triggers only once. If the pawn retreats and enters the zone again later, nothing happens.
- **One move can trigger both tokens on a side,** one after the other. For example, with the pawn on space 2 of B's side, A constructs Arsenal (3 shields) while owning Strategy (1 more). The pawn moves 4 spaces to space 6, passing space 3 on the way: B loses 2 coins and then 5 coins.

## Military supremacy

If the pawn reaches a player's capital, **the other player wins at once**. Whatever the action would still have done after the move is skipped. For example, if the Statue of Zeus's shield carries the pawn into the capital, the game ends before its builder chooses a card to destroy; and if this happens on the last card of an age, the game ends without starting a new age.

## Military points at the end

If the game ends by civilian scoring, only the player who is ahead scores military points: 2, 5 or 10, according to the zone the pawn stands in. The other player scores nothing. If the pawn is on the centre space, neither scores.

## Science symbols

Each green card shows one of six science symbols, and each symbol appears on exactly two green cards:

- **globe:** University and Observatory (both Age III);
- **sundial:** Academy and Study (both Age III);
- **mortar:** Pharmacist (Age I) and Dispensary (Age II);
- **set square:** Workshop (Age I) and Laboratory (Age II);
- **quill:** Scriptorium (Age I) and Library (Age II);
- **wheel:** Apothecary (Age I) and School (Age II).

A seventh symbol, the **scales**, appears only on the Law progress token. Green cards are also worth victory points at the end (see [ages-scoring.md](ages-scoring.md#the-eight-scoring-categories)).

## Pairs

When your city comes to hold **both cards of one symbol**, that is a pair, and you immediately choose one progress token from the board and take it.

- Each symbol is on only two cards, so each pair can happen at most once in a game. If each player owns one card of a symbol, neither will ever have that pair.
- The scales cannot form a pair, because only Law carries them.
- If no tokens are left on the board, the pair gives nothing.
- In Ajayeb the pick is not optional: you must take a token when one is available (see [decisions.md](decisions.md#forced-choices)).
- A pair completed by a card rebuilt with the Mausoleum counts like any other.
- The token is chosen before newly accessible cards are revealed, as part of finishing the turn (see [turn.md](turn.md#the-end-of-a-turn)).
- Pairs never break in the base game, since no effect can remove a green card from a city. A token, once taken, is kept for the rest of the game in any case.

## Scientific supremacy

A player who owns **six different science symbols** out of the seven wins at once. The scales from Law count as one of the seven. The sixth symbol can come from:

- constructing a green card;
- rebuilding a green card with the Mausoleum;
- taking the Law token, either from the board after a pair or from the Great Library offer.

A card that completes a pair repeats a symbol you already had, so it never adds a sixth different symbol itself. The token you then pick can, if it is Law.

## Both supremacies at once

No single action can produce both. Shields come only from red buildings and three wonders, symbols only from green cards and Law, and none of these supplies both.

## For developers

- **The pawn** is one signed integer from -9 to +9. Positive values point toward seat 1's capital, so a positive pawn means seat 0 is ahead and seat 1 is the weaker player.
- **Pawn moves are walked space by space.** Each step away from the centre that lands on space 3 or 6 checks that side's looting token; reaching 9 ends the game immediately with a military victory, inside the move itself. The engine emits a pawn event for the move and a loot event for each token taken.
- **Science** is kept as a count per symbol for each player, with Law adding the scales. A count reaching 2 opens the board token pick if the board is not empty; a sixth distinct symbol ends the game with a science victory.
- **The token pick** uses keys of the form `token:<token>`, for example `token:law`. During a board pick, the legal keys are the tokens still on the board.
- **The victory kinds** are military, science, civilian and forfeit, each with a winner, or a shared victory for tied civilian scoring (see [ages-scoring.md](ages-scoring.md#the-end-of-the-game)).
