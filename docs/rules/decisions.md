# Rulings

This page records how Ajayeb settles the points that the published rules leave ambiguous or do not cover, and a few places where the engine plays a rule in one particular way. Each ruling gives what Ajayeb does, the reason in one line, and the alternative we did not choose.

The project-level decision log in [../dev/decisions.md](../dev/decisions.md) repeats the most visible of these (the centred pawn, the coin supply and the guild back) together with the engineering decisions. Where the two differ, this page is the authority on game rules.

A ruling changes only with a deliberate decision. Changing one changes how games play, so it also changes the engine's rules version, and stored games and golden tests are checked again.

## The main rulings

### Centred pawn

- **Ruling.** When an age ends with the conflict pawn on the centre space, the last active player chooses who starts the next age. When the pawn is off centre, the weaker player chooses, as the rules say. In both cases the next age's structure is laid out before the choice, so the chooser sees its face-up cards.
- **Why.** The rulebook only says that the last active player begins the next age, which can be read either way; the licensed online version (Board Game Arena) lets that player choose, which is what most online players expect, and it matches how the off-centre case works.
- **Alternative.** The last active player simply starts the next age, with no choice. The engine keeps this literal reading as a room option.

### Guild back

- **Ruling.** Guild cards have their own back, different from the Age III back, and it is public. A face-down guild in the Age III structure shows the guild back, so both players know a guild is there but not which one.
- **Why.** Board Game Arena draws face-down guilds with a separate fourth back, and the physical guild cards are reported to have their own back (still to be confirmed against a physical copy; see [sources.md](sources.md#open-questions)).
- **Alternative.** Guilds share the Age III back and cannot be told apart while face down. The back is a single constant in the catalog, so switching is a one-line change.

### Unlimited coins

- **Ruling.** The bank never runs out of coins. A player's coins never go below zero: a loss takes at most what the player has, as the rules say.
- **Why.** The rulebook sets no limit on the bank and Board Game Arena has none, while a cap would need rules for an empty bank that the game never defines.
- **Alternative.** Cap the bank at the coins in the physical box (14 pieces worth 1, 10 worth 3 and 7 worth 6, for 86 coins in total).

### Forced choices

- **Ruling.** Whenever a choice has at least one option, it must be made:
  - after a science pair, a token must be taken if the board has any;
  - the Mausoleum must rebuild a card if the discard pile has any;
  - the Statue of Zeus and Circus Maximus must destroy a card if the opponent has one of that colour;
  - the Great Library's pick always happens, because the box always holds five tokens then.

  When a choice has no option, it is skipped and the wonder is still built.
- **Why.** The rules state the three wonder effects as instructions, not options. The science pair is worded as a permission, but no progress token can ever hurt the player who takes it, so a "decline" button would only add a step. Board Game Arena plays all four the same way.
- **Alternative.** Follow the permissive wording of the science-pair rule literally and offer a "take nothing" option after a pair.

### Effect order

- **Ruling.** Effects happen in a fixed order.
  - **A building:** the payment (to the bank, and the trading part to an opponent who owns Economy); the card enters the city; coin rewards (first Urbanism's chain bonus, then the card's own coins); shields, with looting and a possible military supremacy; the science symbol, with a possible scientific supremacy and then a possible pair.
  - **A wonder:** the payment, as above; the wonder is built, its production starts and the seven-wonder rule is applied; coins; shields; the extra turn is noted; the Appian Way's loss for the opponent; a supremacy check; the special choice.
- **Why.** This is the order Board Game Arena uses, and it has a visible consequence: an opponent who owns Economy receives the trading coins before a looting token or the Appian Way takes coins from them.
- **Alternative.** Apply losses first and the Economy income afterwards, which would let an Economy owner with no coins escape a loss entirely.

### Reveal timing

- **Ruling.** Face-down cards freed during a turn are turned face up at the end of that turn: after every pending choice of the turn, and before the end-of-age check and any extra turn.
- **Why.** The rules require the reveal to happen before an extra turn begins, and Board Game Arena reveals after the choices, so a Mausoleum rebuild or a token pick is made with the information the player had when taking the card.
- **Alternative.** Reveal the moment a card is taken, before any choice, which would give the chooser a little more information.

### Supremacy timing

- **Ruling.** A supremacy ends the game the moment it happens. A military supremacy is checked inside the pawn's move, so nothing later in the action happens (for example, the Statue of Zeus's destroy choice). A scientific supremacy is checked right after a building's effects and right after a token is taken. The end of the turn checks once more as a safety net. A supremacy on the last card of an age ends the game with no age change, and on the last card of Age III it wins without civilian scoring.
- **Why.** The rules make supremacy an immediate win, Board Game Arena checks at the same points, and nothing that could follow in the same turn could undo a supremacy.
- **Alternative.** Check only at the end of the turn, after the pending choices. The winner would be the same, but effects such as the destroy choice would still happen first.

### Reveal at game end

- **Ruling.** When the game ends, every view shows what was hidden: the cards removed from each age deck and the guilds, wonders and progress tokens left in the box. The seed and the full action log stay on the server, in the room's file, so the game can be replayed there; no view ever carries the seed. This is an engine option, on by default.
- **Why.** Hidden information has no game value once the game is over, and seeing it helps players review the game and learn.
- **Alternative.** Keep it hidden forever, as at a real table, where nobody looks into the box.

## Smaller rulings

### Hidden until needed

- **Ruling.** The Age I structure is laid out only when the wonder draft ends, and the four wonders of the second draft round are shown only when the first round ends.
- **Why.** The rules lay out each age's structure when that age begins, and Board Game Arena deals Age I after the draft as well, so the draft is decided on the wonders alone, with the same information for both players.
- **Alternative.** Lay out Age I before the draft, so the picks can react to its face-up cards.

### Automatic payment

- **Ruling.** The player never chooses how to pay. The engine always applies the cheapest payment, including which units a flexible producer covers and which two units Masonry or Architecture waive (see [economy.md](economy.md#the-cheapest-payment)).
- **Why.** The cheapest payment is never worse for the payer: the printed coins are fixed, so a smaller total means less trading and also fewer coins for an opponent who owns Economy.
- **Alternative.** Let the player assign resources and discounts by hand, as with physical cards.

### The fixed-price delay

- **Ruling.** The four fixed-price cards (the three Reserves and the Customs House) take effect at once, with no one-turn delay.
- **Why.** These cards cost only coins, so no resource can be bought on the turn one is constructed, and the delay printed on the physical cards never changes anything.
- **Alternative.** Model the delay explicitly, which adds state without ever changing a result.

### The Great Library draw

- **Ruling.** The order of the five tokens in the box is shuffled when the game is created, and the Great Library takes the first three.
- **Why.** All chance must be resolved at setup so that a game is fully described by its seed and its action keys, and a pre-shuffled order gives the same odds as a fresh draw.
- **Alternative.** Draw three tokens at random when the Great Library is built, which would need randomness during play.

### Extra turns in a row

- **Ruling.** Extra turns chain. A wonder that grants an extra turn, built during an extra turn, grants another one.
- **Why.** The rules do not forbid it, and Board Game Arena grants an extra turn for every such wonder.
- **Alternative.** Allow at most one extra turn in a row.

### Urbanism and the Mausoleum

- **Ruling.** Urbanism's 4-coin bonus applies only to free constructions through a chain. A card rebuilt for free by the Mausoleum does not earn it.
- **Why.** The token rewards chains specifically, and Board Game Arena pays the bonus only for chain constructions.
- **Alternative.** Pay the bonus for any free construction, the Mausoleum included.

### The open discard pile

- **Ruling.** The discard pile is shown openly to both players at all times.
- **Why.** In the physical game the pile is face down but either player may look through it whenever they like, so showing it changes nothing but convenience.
- **Alternative.** Hide it behind a "look at the discard pile" control.

### Forfeit

- **Ruling.** A player may forfeit at any time before the game is over, in any phase and whether or not it is their turn (in a hot-seat game, the player to move forfeits). The game ends at once and the other player wins, as a forfeit victory. A choice still open is dropped: the tokens the Great Library drew go back to the box unseen. The score sheet records the points of the moment, for the record only, and the end-of-game reveal shows the hidden cards as for any other end. A forfeit cannot be undone.
- **Why.** The rulebook has no forfeit, but a player who wants to stop needs a clean way out that names the winner, and online players expect resignation to end the game on the spot. In the engine it is the action `resign:<seat>`, recorded in the game log like any other, so replays reproduce it. No game recorded before it contains one, so the rules version did not change.
- **Alternative.** Score the game on points as if it ended there, or let the other player choose between taking the win and playing on alone against nobody.
