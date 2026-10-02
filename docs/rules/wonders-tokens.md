# Wonders and progress tokens

This page describes every wonder and every progress token, the seven-wonder rule, and the special choices that some wonders open. How to build a wonder is explained in [turn.md](turn.md#3-build-a-wonder). Terms such as *pending choice*, *extra turn* and *the box* are defined in the [README](README.md#terms-used-throughout).

## Wonders in general

- Each player drafts four wonders before Age I (see [setup.md](setup.md#the-wonder-draft)).
- A wonder can be built in any age and in any order. To build one, you take an accessible card, pay the wonder's cost, and tuck the card under the wonder.
- A built wonder stays built for the rest of the game. Nothing can destroy it.
- Its effects happen once, at the moment it is built, except production, which works on every turn from then on. Its victory points are counted at the end.
- The Architecture token reduces the cost of your later wonders, and the Theology token gives each of your later wonders an extra turn.

## The seven-wonder rule

Only seven wonders can be built in a game. **When the seventh wonder is built, by either player, the one wonder still unbuilt goes back to the box at once**, whichever player drafted it.

- If all seven are built, one player ends with four built wonders and the other with three.
- The removed wonder counts for nothing: no points, no effect, and it does not count for the Builders Guild.
- From then on, no player can build a wonder.

## The twelve wonders

Each entry gives the wonder's slug, its effect when built, and its victory points at the end.

- **The Pyramids** (`pyramids`): no effect. 9 points.
- **The Colossus** (`colossus`): 2 shields. 3 points.
- **The Great Lighthouse** (`great-lighthouse`): from now on, produces one unit of wood, clay or stone each turn. Like all flexible production, it does not raise the opponent's trade prices. 4 points.
- **The Temple of Artemis** (`temple-of-artemis`): take 12 coins from the bank, and take an extra turn. 0 points.
- **The Mausoleum** (`mausoleum`): construct one card of your choice from the discard pile, for free (see [below](#the-mausoleum)). 2 points.
- **The Great Library** (`great-library`): draw three of the five progress tokens in the box, keep one and return the other two (see [below](#the-great-library)). 4 points.
- **Piraeus** (`piraeus`): from now on, produces one unit of glass or papyrus each turn, without raising the opponent's prices. Take an extra turn. It pays no coins and has no special link to any token. 2 points.
- **The Hanging Gardens** (`hanging-gardens`): take 6 coins from the bank, and take an extra turn. 3 points.
- **The Statue of Zeus** (`statue-of-zeus`): 1 shield. Then destroy one brown card in your opponent's city (see [below](#the-statue-of-zeus-and-circus-maximus)). 3 points.
- **The Sphinx** (`sphinx`): take an extra turn. 6 points.
- **The Appian Way** (`appian-way`): take 3 coins from the bank; your opponent loses 3 coins to the bank; take an extra turn. 3 points.
- **Circus Maximus** (`circus-maximus`): 1 shield. Then destroy one grey card in your opponent's city. 3 points.

## Special choices

Three wonders open a **pending choice** that their builder settles before the turn ends. Ajayeb never offers an empty choice: when there is nothing to choose from, the choice is skipped and the wonder is still built. When there is something, the choice must be made; it cannot be declined (see [decisions.md](decisions.md#forced-choices)).

### The Statue of Zeus and Circus Maximus

- The builder picks one card of the named colour in the opponent's city: brown for the Statue of Zeus, grey for Circus Maximus. The card goes to the discard pile, so the Mausoleum can bring it back later.
- If the opponent has no card of that colour, nothing is destroyed.
- The opponent loses that card's production at once, so the builder's trade price for that resource drops at once.
- No chain is ever broken, because brown and grey cards carry no chain symbols.
- The wonder's shield moves the pawn before the choice. If the shield carries the pawn into the opponent's capital, the game ends and the choice never happens.

### The Mausoleum

- **The pool** is every card on the discard pile: cards discarded for coins and cards destroyed by the Statue of Zeus or Circus Maximus, from any age and from either player. It never includes the cards removed at setup or the cards tucked under wonders.
- The builder picks one card and constructs it **for free**: no coins and no resources.
- The card enters the builder's city with **all its effects**: shields (with Strategy's bonus), its science symbol (which can complete a pair or a scientific supremacy), coin rewards from yellow cards and guilds, production, chain symbols and victory points.
- It is not a construction through a chain, so Urbanism pays nothing for it.
- If the discard pile is empty, nothing happens.

### The Great Library

- Three of the five tokens in the box are drawn at random. **Only the builder sees them.** The builder keeps one, which then sits with the builder's other tokens in full view like any token. The other two go back to the box, and the opponent never learns what they were.
- The kept token takes effect at once: Agriculture and Urbanism pay their 6 coins, and Law adds the scales symbol, which can complete a scientific supremacy.
- The box always holds five tokens at this point, because nothing else draws from it, so this choice always happens.
- In a hot-seat game both people may see the offer on the shared screen. That is accepted.

## The order of a wonder's effects

When a wonder is built, its effects happen in this order:

1. the cost is paid, and an opponent who owns Economy receives the trading coins;
2. the wonder is marked as built, its production (if any) starts, and the seven-wonder rule is applied if this is the seventh;
3. coins are taken from the bank;
4. shields move the pawn, which can trigger looting or a military supremacy;
5. the extra turn, if any, is noted;
6. the Appian Way takes 3 coins from the opponent;
7. the special choice, if any, opens.

The reasons for this order, and its consequences, are in [decisions.md](decisions.md#effect-order).

## Progress tokens

There are ten progress tokens. At setup, five go face up on the board and five go to the box. A player gains a token in two ways:

- by completing a science pair, which lets them pick one token from the board (see [military-science.md](military-science.md#pairs));
- by building the Great Library, which offers three tokens from the box.

A token is kept for the rest of the game. One-off effects happen at once; lasting effects apply to everything the player does afterwards, never to what they did before.

## The ten tokens

- **Agriculture** (`agriculture`): take 6 coins from the bank at once. 4 points at the end.
- **Architecture** (`architecture`): every wonder you build afterwards costs 2 fewer resource units (see [economy.md](economy.md#discounts-masonry-and-architecture)).
- **Economy** (`economy`): the coins your opponent pays for trading come to you instead of the bank (see [economy.md](economy.md#the-economy-token)).
- **Law** (`law`): counts as the scales, a seventh science symbol (see [military-science.md](military-science.md#scientific-supremacy)).
- **Masonry** (`masonry`): every blue card you construct afterwards costs 2 fewer resource units.
- **Mathematics** (`mathematics`): at the end, 3 points for each progress token you own, Mathematics included.
- **Philosophy** (`philosophy`): 7 points at the end.
- **Strategy** (`strategy`): every red building you construct afterwards gives 1 extra shield. Wonders get no extra shield.
- **Theology** (`theology`): every wonder you build afterwards grants an extra turn. A wonder that already grants one still grants only one.
- **Urbanism** (`urbanism`): take 6 coins from the bank at once. Afterwards, each building you get for free through a chain also pays you 4 coins. A free construction through the Mausoleum does not count.

## For developers

- **Action keys** for the special choices:
  - `destroy:<card>`, for example `destroy:quarry`; the legal keys are the opponent's cards of the named colour;
  - `revive:<card>`, for example `revive:garrison`; the legal keys are all the cards on the discard pile;
  - `token:<token>`, for example `token:law`; during a Great Library pick, the legal keys are the three drawn tokens.
- **Each choice is its own phase,** with at most one open at a time: destroy a card, build from the discard pile, pick a board token, pick a library token. The engine never enters a choice phase that has no legal key.
- **The Great Library draw is not random at play time.** The order of the box was shuffled when the game was created, and the Great Library takes its first three tokens, which has the same distribution as a fresh draw. The two tokens not chosen go back to the bottom of the box, unannounced. The offer appears only in the builder's view, and the event announcing the draw is filtered the same way.
- **A real game's box always holds enough tokens for a full draw of three,** because nothing else draws from it. Only a scenario can start the box shorter: the Great Library then offers whatever is left, and if the box is empty the choice is skipped like any other special choice with no option (see "Special choices" above).
- **The extra turn a wonder grants is reported by `EvExtraTurn`,** whether the wonder itself grants one or the builder's Theology token does; since the two never stack, only one event is emitted either way.
- **The seven-wonder rule** is applied by a counter of built wonders: when it reaches seven, the remaining unbuilt wonder is marked as removed, and an event reports it.
