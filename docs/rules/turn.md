# A turn

This page covers what a player does on a turn, which cards can be taken, when face-down cards are turned over, and how extra turns work. Terms such as *accessible* and *pending choice* are defined in the [README](README.md#terms-used-throughout).

## Whose turn it is

The players alternate. The first player starts Age I. Who starts Age II and Age III is decided at the end of the previous age (see [ages-scoring.md](ages-scoring.md#who-starts-the-next-age)). Some wonders and the Theology token give a player an **extra turn**, which breaks the alternation (see [Extra turns](#extra-turns) below).

## Taking a card

On your turn you must take exactly one **accessible** card from the structure. You cannot pass, and you cannot take a card that another card still overlaps. Every accessible card is face up when your turn begins, because face-down cards are turned over at the end of the turn that frees them (see [The end of a turn](#the-end-of-a-turn)).

You then do exactly one of the following three things with the card.

## 1. Construct it as a building

You pay the card's cost and put it into your city.

- The cost may be nothing, some coins, some resources, or coins and resources together. Resources you do not produce can be bought from the bank. A card can also be constructed for free through a **chain**. All of this is explained in [economy.md](economy.md).
- You may construct the card only if you can pay its whole cost now.
- Once in your city, the card's effects start:
  - production, fixed prices and flexible production work from now on;
  - shields move the conflict pawn at once;
  - a science symbol may complete a pair or a scientific supremacy at once;
  - a coin reward is paid once, now;
  - victory points are counted at the end of the game;
  - a chain symbol on the card lets you construct its matching later card for free.
- Every building exists only once, so no city ever holds two copies of a card.

## 2. Discard it for coins

You put the card on the discard pile and take coins from the bank.

- This is always allowed, whatever the card and whatever your coins.
- You gain **2 coins, and 1 more for every yellow card your city holds.** The discarded card itself is not in your city, so a discarded yellow card does not count.
- The discard pile is open to both players. The Mausoleum can later bring a card back from it (see [wonders-tokens.md](wonders-tokens.md#the-mausoleum)).

## 3. Build a wonder

You use the card to build one of your own wonders.

- Choose one of your wonders that is not yet built and that has not been removed by the seven-wonder rule (see [wonders-tokens.md](wonders-tokens.md#the-seven-wonder-rule)). Pay **the wonder's cost**, not the card's.
- Tuck the card under the wonder. **Any accessible card can be used**, including one you could not afford to construct. The tucked card has no effect of any kind: it is not part of your city, it is not on the discard pile, and its chain symbol does nothing. Both players can still see which card it was.
- The wonder's effects start at once; its victory points are counted at the end. Some wonders grant an extra turn, and some open a special choice.
- Once seven wonders have been built in total, by either player, no further wonder can be built.

## The end of a turn

After the action, the turn finishes in this order:

1. **Pending choices are settled.** The action may have opened a choice: a progress token for a completed science pair, the opponent's card destroyed by the Statue of Zeus or Circus Maximus, the card rebuilt by the Mausoleum, or the token kept from the Great Library. The active player makes these choices. One choice can open another; for example, a green card rebuilt by the Mausoleum can complete a pair and open a token choice.
2. **Newly accessible cards are revealed.** Every face-down card that the turn has freed is turned face up.
3. **The age may end.** If the structure is now empty, the age is over (see [ages-scoring.md](ages-scoring.md#the-end-of-an-age)). An extra turn earned during this turn is lost.
4. **Otherwise play continues.** If you earned an extra turn, you play again. If not, your opponent's turn begins.

A supremacy does not wait for the end of the turn. The game ends the moment the conflict pawn reaches a capital or a player gets a sixth science symbol, even in the middle of an action (see [decisions.md](decisions.md#supremacy-timing)). The reveal comes after the pending choices on purpose: the choices are made with the information the active player had when taking the card (see [decisions.md](decisions.md#reveal-timing)).

## Extra turns

- **Sources:** the Temple of Artemis, Piraeus, the Hanging Gardens, the Sphinx and the Appian Way each grant an extra turn when built. With the Theology token, every wonder you build afterwards grants one too.
- **Never doubled:** a wonder grants at most one extra turn, even if it already grants one and you own Theology.
- **Timing:** the extra turn begins after the newly accessible cards have been revealed, so you can use what was just turned over.
- **Chaining:** if you build another such wonder during your extra turn, you earn yet another extra turn.
- **Lost at the end of an age:** if the card you took was the last card of the age, the extra turn is lost. It does not carry over into the next age, whose starter is chosen in the normal way.

## There is always a move

Discarding is always possible, so a player always has at least one legal move and the game can never stall.

## For developers

- **Action keys** in the play phase:
  - `construct:<card>`, for example `construct:baths`;
  - `discard:<card>`, for example `discard:baths`;
  - `wonder:<wonder>:<card>`, wonder first and then the tucked card, for example `wonder:pyramids:baths`.
- **The legal set** in the play phase is, for every accessible card: its discard key always; its construct key if the mover can pay the card; and one wonder key for each of the mover's wonders that is unbuilt, not removed and affordable, provided fewer than seven wonders have been built. The server accepts only keys from this set.
- **Payment is never part of a key.** The engine pays the cheapest way (see [economy.md](economy.md#the-cheapest-payment)) and reports the breakdown in the view.
- **The end-of-turn procedure** runs after the action and after any pending choice phases, in this order: reveal newly accessible cards (one event per card), check supremacy, then either end the age (dropping and reporting a lost extra turn), or give the same player an extra turn, or pass the move to the opponent.
- **Two seats can matter at once.** The *active* seat is the one whose turn it is. The seat *to move* is the one that must act now. They can differ only while a player other than the active one chooses who starts the next age.
