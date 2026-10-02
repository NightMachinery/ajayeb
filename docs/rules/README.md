# Rules of the game

Ajayeb implements the base game of 7 Wonders Duel for two players, with no expansions. These pages state its rules in our own words. They have two readers in mind:

- **Players** who want to know exactly what the game does in a given situation.
- **Developers** who need the precise behaviour the engine must have. The engine's rules tests are written from these pages (see [AGENTS.md](../../AGENTS.md)), and each page ends with a short "For developers" section that maps its rules onto engine concepts: phases, action keys and events.

The pages describe the behaviour the engine is built and tested against. Where the published rules are ambiguous, Ajayeb makes a ruling, and every ruling is collected in [decisions.md](decisions.md).

## Pages

1. [setup.md](setup.md): the components, the setup steps, the wonder draft, and the three card structures with diagrams.
2. [turn.md](turn.md): the three things you can do on a turn, which cards you can take, when face-down cards are turned over, and extra turns.
3. [economy.md](economy.md): coins, resources, trading and its price rule, fixed prices, choice producers, chains, the Masonry and Architecture discounts, the Economy token, and how the cheapest payment is found.
4. [military-science.md](military-science.md): the military track, its zones and looting tokens, military supremacy, science symbols and pairs, the Law token, and scientific supremacy.
5. [wonders-tokens.md](wonders-tokens.md): each wonder and progress token, the seven-wonder rule, and the special choices that some of them open.
6. [ages-scoring.md](ages-scoring.md): the end of an age and who starts the next one, the end of the game, the eight scoring categories, the tie-break and the shared victory.
7. [decisions.md](decisions.md): our rulings on ambiguous points, each with its reason and the alternative we did not choose.
8. [sources.md](sources.md): where the rules and card data come from, the licences of those sources, and what we never copy.
9. [data-rulings.md](data-rulings.md): how each disagreement found by the three-way catalog data check was settled.

## The game on one page

Two players build rival civilizations over three **ages**. Each age uses 20 cards laid out in an overlapping arrangement called the **structure**, some face up and some face down. Players take turns. On your turn you take one card that nothing overlaps and do exactly one of three things with it:

- **construct** it as a building in your city, paying its cost;
- **discard** it for coins;
- use it to **build one of your wonders**, paying the wonder's cost and tucking the card under the wonder.

Costs are paid in coins and in resources. Resources come from your own buildings, and anything you lack you can buy from the bank, at a price that rises with how much of that resource your opponent produces.

Cards come in seven colours:

- **brown** cards produce raw goods (wood, clay, stone);
- **grey** cards produce manufactured goods (glass, papyrus);
- **red** cards carry shields, which push the conflict pawn toward the opponent's capital;
- **green** cards carry science symbols;
- **yellow** cards bring coins, cheaper trading, flexible production and, in Age III, points;
- **blue** cards are worth victory points;
- **purple** cards are the guilds, which appear only in Age III and reward you for what the cities contain.

Before Age I, the players draft four **wonders** each. Wonders are large one-off projects with strong effects. Only seven of the eight drafted wonders can ever be built.

Ten **progress tokens** give lasting advantages. Five are on the board at the start; you take one whenever you complete a **pair** of matching science symbols. The other five stay in the box, where only the Great Library can reach them.

There are three ways to win:

- **Military supremacy:** the conflict pawn reaches your opponent's capital. You win at once.
- **Scientific supremacy:** you own six different science symbols. You win at once.
- **Civilian victory:** if neither supremacy happens, the game ends after the last card of Age III and the players count victory points in eight categories. The higher total wins; a tie goes to the player with more points from blue cards, and if that is also tied, both players share the victory.

## Terms used throughout

These terms have a fixed meaning on every page.

- **Structure:** the 20 cards of the current age, laid out in overlapping rows.
- **Accessible:** a card in the structure is accessible when no other card in the structure overlaps it. Only accessible cards can be taken.
- **Face up and face down:** a face-up card shows its front. A face-down card shows only its back, which tells you its age, or that it is a guild.
- **City:** the buildings a player has constructed. A player's wonders and progress tokens sit beside the city.
- **The box:** where removed components go. They never return to play, with one exception: the Great Library draws from the progress tokens in the box.
- **Discard pile:** the cards discarded for coins, plus the cards destroyed by the Statue of Zeus or Circus Maximus. Both players may look at it at any time, so Ajayeb shows it openly.
- **Tucked card:** the card placed under a wonder to build it.
- **Trade price:** what one unit of a resource costs a player when bought from the bank.
- **Chain:** a symbol printed on an earlier card that lets you construct a matching later card for free.
- **Pair:** two cards in one city that show the same science symbol.
- **Supremacy:** a military or scientific win, which ends the game immediately.
- **Pending choice:** a decision that an action opens and that is settled before the turn ends, such as picking a progress token after a pair.
- **Extra turn:** a second turn in a row for the same player, granted by some wonders and by the Theology token.
- **Active player:** the player whose turn it is. The **last active player** of an age is the one who took its last card.
- **Weaker player:** the player on whose side of the military track the conflict pawn stands. When the pawn is in the centre, neither player is weaker.

## Names and ids

Card names differ between printings and between implementations, so Ajayeb identifies every card, wonder and token by an id and a slug, never by its display name. A slug is the English name in lower-case kebab case without a leading "the", for example `lumber-yard`, `merchants-guild`, `pyramids`, `great-library` or `agriculture`. Display names live in the client's translation tables.

Names to watch out for:

- The Age II blue card that costs two wood and one glass is called Courthouse here (slug `courthouse`). The 2015 English rulebook calls it Tribunal.
- The Age III red card that costs 8 coins is called Pretorium here (slug `pretorium`). The 2015 English rulebook calls it Courthouse.
- The Merchants Guild (slug `merchants-guild`) is called Traders Guild in one place in the 2015 rulebook.
- Four building names resemble wonder names but are different things: Library (Age II green card) and the Great Library (wonder); Lighthouse (Age III yellow card) and the Great Lighthouse (wonder); Circus (Age III red card) and Circus Maximus (wonder); Statue (Age II blue card) and the Statue of Zeus (wonder).

## For developers

- The engine lives in `server/internal/engine`, and the card, wonder and token data in `server/internal/engine/catalog`. Building ids follow the numbering of the Board Game Arena implementation (1 to 73) so the data can be cross-checked against it; wonder and token ids follow our own order, from `pyramids` to `circus-maximus` and from `agriculture` to `urbanism`.
- Costs, point values and other per-card numbers are data, not rules. The catalog is the single source of truth for them, and the game shows them on every card. These pages name cards only where a rule depends on them.
- What each player may see (face-down cards, the box, the Great Library offer) is described in [architecture.md](../architecture.md#hidden-information).
