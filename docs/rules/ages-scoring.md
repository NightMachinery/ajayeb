# Ages and scoring

This page covers the end of an age, who starts the next one, how the game ends, the eight scoring categories (including the guilds), the tie-break and the shared victory. Terms such as *last active player* and *weaker player* are defined in the [README](README.md#terms-used-throughout).

## The end of an age

An age ends when the last card of its structure has been taken and that turn has finished, including any pending choices it opened.

- An extra turn earned on that last turn is lost.
- Nothing is reset between ages. Coins, cities, wonders, progress tokens, the pawn's position and the looting tokens still on the track all carry over, and the progress tokens left on the board stay available.
- If a supremacy happens on the last card of an age, the game is over at once and no new age begins.

## Who starts the next age

1. The next age's structure is laid out first, so everyone can see its face-up cards before the starter is decided.
2. **The weaker player chooses** who starts the next age: themselves or their opponent. The weaker player is the one on whose side of the track the pawn stands.
3. **If the pawn is on the centre space,** neither player is weaker, and the **last active player** chooses. This is Ajayeb's ruling on a point the rulebook leaves unclear (see [decisions.md](decisions.md#centred-pawn)).

The choice matters because the starter takes the first card from the new structure, and in Ages II and III only two cards are accessible at the start.

## The end of the game

The game ends in one of two ways.

- **By supremacy:** a military or scientific supremacy ends the game the moment it happens, with a single winner and no scoring (see [military-science.md](military-science.md)).
- **By civilian scoring:** otherwise, the game ends when the last card of Age III has been taken and that turn has finished, including its pending choices. An extra turn earned on that turn is lost. The players then count their victory points.

## The eight scoring categories

Each player adds up points in eight categories:

1. **Blue cards:** the points printed on the blue cards in their city.
2. **Green cards:** the points printed on their green cards.
3. **Yellow cards:** the points printed on their yellow cards. Only the Age III yellow cards have points, 3 each.
4. **Purple cards (guilds):** see [Guilds](#guilds) below.
5. **Wonders:** the points of each wonder they have built. A wonder that was never built, or was removed by the seven-wonder rule, is worth nothing.
6. **Progress tokens:** Agriculture is worth 4, Philosophy 7, and Mathematics 3 for every token the player owns, Mathematics included. The other tokens are worth no points.
7. **Coins:** 1 point for every full 3 coins. Leftover coins are worth nothing, so 14 coins give 4 points.
8. **Military:** 2, 5 or 10 points for the player who is ahead on the track, depending on the pawn's zone, and nothing for the other player (see [military-science.md](military-science.md#military-points-at-the-end)).

The total of the eight categories is the player's score.

## Guilds

The three guilds in a game are drawn at random from seven. Unlike the other cards, a guild looks at **both cities**. Most guilds count the cards of one kind in whichever city has the most of them, which may be the opponent's city. When both cities have the same number, the count is the same either way.

A guild acts at two moments:

- **When it is constructed,** it may pay coins, based on the cities at that moment.
- **At the end of the game,** it scores points, based on the cities at the end. The city with the most cards of that kind can be a different one by then.

The seven guilds follow. In each entry, the coins are paid when the guild is constructed and the points are scored at the end, each from the counts at that moment.

- **Merchants Guild:** yellow cards. 1 coin and 1 point per yellow card in the city with the most yellow cards.
- **Shipowners Guild:** brown and grey cards together. 1 coin and 1 point per brown or grey card, counted in the single city with the most brown and grey cards combined. Brown cards from one city cannot be added to grey cards from the other.
- **Builders Guild:** 2 points per wonder built by the player who has built the most wonders. It pays no coins.
- **Magistrates Guild:** blue cards. 1 coin and 1 point per blue card in the city with the most blue cards.
- **Scientists Guild:** green cards. 1 coin and 1 point per green card in the city with the most green cards.
- **Moneylenders Guild:** 1 point per full 3 coins held by the richer player at the end. It pays no coins.
- **Tacticians Guild:** red cards. 1 coin and 1 point per red card in the city with the most red cards.

No guild counts purple cards.

Example: at the end, A has built 4 wonders and B has built 3. Whoever owns the Builders Guild scores 8 points for it. If A has 14 coins and B has 5, whoever owns the Moneylenders Guild scores 4 points for it.

## Tie-break and shared victory

- The player with the higher total wins.
- If the totals are equal, the player with more points from blue cards (category 1) wins.
- If those are equal too, **both players share the victory.**

Supremacies need no tie-break: each one has exactly one winner, and no single action can produce both (see [military-science.md](military-science.md#both-supremacies-at-once)).

## The score preview

During the game, Ajayeb shows each player's score as if the game ended now: military points from the current pawn position, guild points from the current cities, and so on. It is a preview only; supremacy can still end the game before any scoring.

## For developers

- **The starter choice** is its own phase. The key is `starter:<seat>`, with seat 0 or 1, for example `starter:1`. The seat to move is the chooser; the active seat stays the last active player until the choice is made.
- **The chooser** is the seat on whose side the pawn stands. With the pawn in the centre it is the last active seat, under the default option. A room option provides the literal reading of the rulebook, in which the last active player simply starts the next age and no choice phase opens; an off-centre pawn still gives the choice to the weaker player.
- **The next age is dealt before the choice.** The view shows the new structure while the chooser decides.
- **Scores** are kept per category in the same eight fields as above, plus the total. The result records the kind of victory (military, science or civilian) and the winner, or a shared victory.
- **Guild and Age III yellow coins** are computed after the card has entered the city, which is why the Lighthouse counts itself. Guild points are computed from the end state only.
