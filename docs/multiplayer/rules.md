# 7 Wonders, second edition: the base-game rules

Draft for M12.rules-inventory, by @Opus5.5 (2026-10-21). Facts from the publisher's second-edition rulebook and sheets, stated in our own words; see [sources.md](sources.md).

This page states the rules of the base game of 7 Wonders, second edition (Repos Production, 2020), for 3 to 7 players, with no expansions. It is written for two readers: a player who wants to know exactly what happens, and the engine designer, who should never need the rulebook. Where the published rules leave a point open, the page gives a provisional ruling; every one of them is collected under [Ambiguities](#ambiguities) at the end.

Per-card numbers (costs, point values, player-count markers) are data, not rules. They live in the private draft tables `cards-draft.csv` and `wonders-draft.csv` until the data milestone verifies them. This page names a card only where a rule depends on it, and describes each wonder ability without its stage costs.

The second edition differs from the first (2010) in many card and wonder details and in a few rules. Our list of those differences is a working note, held back until the blind data verification in M13; until then, check which edition a source describes before trusting it (see [sources.md](sources.md)).

## Terms used throughout

- **Player count (N):** the number of players, 3 to 7. There is no two-player game in this edition.
- **Neighbours:** the two players seated directly to your left and right. Players sit in a ring. With 3 players your two neighbours are the two other players.
- **City:** everything a player has built: their wonder board with its built stages, and the buildings (cards) they have constructed.
- **Age:** one of the three rounds (Ages I, II and III). Each has its own deck.
- **Age card:** any card of the three decks. A constructed Age card is a **building**.
- **Hand:** the cards a player holds and chooses from on a turn.
- **Turn:** one step of an Age in which every player plays one card at the same time. An Age has six turns, plus one extra turn for a player with the Babylon night ability.
- **Wonder board:** a player's board. It shows a **starting resource** and a row of **stages**. Each board is printed on two faces, the **day side** and the **night side**; a player uses one side for the whole game.
- **Stage:** one construction step of a wonder, with a cost and an effect. Stages are built strictly left to right.
- **Treasure:** the coins a player holds. The **reserve** is the common supply of coins.
- **Discard pile:** the face-down pile of sold cards and of cards discarded on the last turn of each Age.
- **Production symbol:** one resource symbol in a city. Each symbol supplies one unit of its resource once per turn.
- **Tradeable production:** the production symbols a neighbour may buy (see [Buying from neighbours](#buying-from-neighbours)).
- **Choice producer:** a production symbol that supplies one unit of one resource from a listed set, chosen each time it is used.
- **Chain symbol:** a small symbol printed on a card. A card provides the chain symbols shown in its top right corner. A card whose free-construction condition shows a chain symbol is free to construct for a player whose city already holds a card that provides that symbol.
- **Military strength:** the total number of shields in a city.
- **Conflict token:** a military defeat token (minus 1 point) or a military victory token (1, 3 or 5 points, by Age).
- **Science symbol:** one of three symbols, which we call **compass**, **gear** and **tablet**.

## Resources

There are seven resources in two families:

- **raw resources** (the rulebook calls them common): wood, clay, stone and ore, produced by brown cards;
- **manufactured resources** (the rulebook calls them rare): glass, papyrus and cloth, produced by grey cards. The card that produces cloth is the Loom.

Resources also come from the starting resource of each wonder board, from two yellow cards (Caravansery and Forum, which produce one resource of a family by choice), and from some wonder stages (Alexandria).

Our data notation uses one letter per resource unit: W wood, C clay, S stone, O ore, G glass, P papyrus, L cloth, and `Nc` for N coins.

## Components

- 7 wonder boards, each printed on both faces (day and night).
- 148 Age cards: 49 for Age I, 49 for Age II, and 50 for Age III (40 Age III cards and 10 guilds).
- Coins worth 1 and coins worth 3, 78 pieces in all (126 coins of value).
- 48 conflict tokens: 24 defeat tokens and, for each Age, 8 victory tokens.
- A score pad, a sheet describing every effect symbol, and a sheet listing the cards and their chains.

Each Age card shows its cost, its effect, its name, a colour-blind symbol for its colour, the chain symbols it provides (top right), the chain symbol that makes it free if there is one, and the smallest player count at which it is used. Its back shows its Age and the direction in which hands pass during that Age.

## Setup

1. **Build the three decks for N players.** Every printed card carries a minimum player count (3+ to 7+). Keep the cards whose minimum is at most N and return the rest to the box. A card design printed with several minimums (for example 3+ and 5+) is several physical copies, one per minimum.
2. **Add the guilds.** Shuffle the 10 guilds face down, draw N + 2 of them without looking, and shuffle them into the Age III deck. The other guilds return to the box unseen. Nobody knows which guilds are in the game.
3. **Shuffle each deck.**
4. **Wonder boards.** Each player receives a random board and picks which side to play. The rulebook suggests day sides for first games and allows players to agree to pick boards instead of dealing them.
5. **Coins.** Each player starts with 3 coins. The rest form the reserve.
6. **Conflict tokens** go to a common supply.

The cards per Age come out at exactly 7 per player:

- 3 players: 21 cards per Age (Age III: 16 Age III cards and 5 guilds).
- 4 players: 28 (Age III: 22 and 6 guilds).
- 5 players: 35 (Age III: 28 and 7 guilds).
- 6 players: 42 (Age III: 34 and 8 guilds).
- 7 players: 49 (Age III: 40 and 9 guilds).

## The structure of an Age

When an Age begins, every player is dealt 7 cards from that Age's deck, face down. The Age then has six turns. Each turn has three steps, always in this order: everyone chooses a card, everyone plays it, everyone passes their remaining hand.

### Choosing

Each player looks at their hand, picks one card and lays it face down in front of them. The others see that a choice has been made, but not which card. The rest of the hand is set aside.

### Playing

Once every player has chosen, all chosen cards are played at the same time. Each player does exactly one of the three actions with their card (see [The three actions](#the-three-actions)). The rulebook suggests that beginners play one after another to watch each other, but the game itself is simultaneous: no player's action on a turn can depend on another player's action on the same turn (see [Order within a turn](#order-within-a-turn)).

### Passing

Each player hands the cards they did not choose to a neighbour:

- in Ages I and III, to the neighbour on their left (so each player receives from the neighbour on their right);
- in Age II, to the neighbour on their right.

Hands shrink by one card per turn: 7 cards on the first turn, 2 on the sixth.

### The sixth turn

At the start of the sixth turn each player holds the 2 cards just received. They choose 1 to play and discard the other face down onto the discard pile, gaining nothing for it. The chosen card is then played as on any turn. (A player with the Babylon night ability may keep and play the second card instead; see [Babylon](#babylon-the-hanging-gardens).)

### End of the Age

After the sixth turn, and after any extra turn from the Babylon night ability, the Age ends with the military conflicts (see [Military conflicts](#military-conflicts)). Then the next Age begins. After Age III the game ends.

## The three actions

### Construct the building

Pay the card's cost (see [Paying costs](#paying-costs)) and place the card face up in your city. A building stays for the rest of the game.

- **No two buildings with the same name.** A city never holds two cards with the same name. This matters because Glassworks, Press and Loom exist in both Age I and Age II under the same names: a city may hold only one Glassworks, one Press and one Loom.
- The card's effect starts at once.

### Construct a wonder stage

Pay the cost of the leftmost stage you have not yet built, and slide the card face down under your board at that stage. The card's own cost, effect and colour do not matter: any card can be used for any stage. The stage's effect is yours for the rest of the game.

- Stages are built in order, first to last, and only one per action.
- Building stages is optional. Stages may be built in any Age.
- A built stage never comes back, and its tucked card stays face down: other players do not learn which card was used.

### Sell the card

Put the card face down on the discard pile and receive 3 coins from the reserve. Selling is always allowed, whatever the card.

## Paying costs

A building or a stage costs nothing, coins, resources, or (for buildings only) can be free through a chain. The second edition's Age cards cost either coins or resources, never both. Some wonder abilities make a building completely free.

### Coins

A coin cost is paid from your treasure to the reserve. In the base game only some brown cards cost coins.

### Your own production

- Every production symbol in your city supplies one unit of its resource on every turn: the starting resource on your board, each symbol on your brown and grey cards, each choice producer on your yellow cards, and the production of your built wonder stages.
- A card with two identical symbols supplies two units.
- A choice producer supplies one unit of one resource from its set; you choose afresh each time you use it.
- Production is never used up. The same symbols pay again on the next turn.
- Production cannot be stored, given away or used across turns.

### Buying from neighbours

When your own production does not cover a cost, you may buy the missing units from your two neighbours.

- **Price.** Each unit costs 2 coins, paid to the neighbour who owns the symbol, unless a commerce discount applies (below).
- **What can be bought.** Only a neighbour's tradeable production: the starting resource on their board, and the symbols on their brown and grey cards. Production from yellow cards (Caravansery, Forum) and from wonder stages (Alexandria) can never be bought.
- **Choice symbols.** A neighbour's brown card that offers a choice of two resources may be bought as either resource, whatever the neighbour uses it for that turn.
- **Once per symbol per buyer per turn.** You may buy each of a neighbour's production symbols at most once per turn. A double symbol card counts as two symbols, so it can sell two units.
- **No exclusivity.** Both of a player's neighbours may buy the same symbol on the same turn. The owner cannot refuse a sale and still uses their own production as normal.
- **Only for this turn.** Bought units must be used on the turn they are bought.
- **Only what existed at the start of the turn.** You cannot buy from a building your neighbour constructs on the same turn.
- **Only coins you held at the start of the turn.** You pay for purchases with the coins you had when the turn began. Coins you receive during the turn (from neighbours buying from you, from selling, from card effects) cannot be spent until a later turn. The publisher's FAQ confirms this.
- You may buy from either neighbour or both in one payment, and may combine bought units with your own production.

### Commerce discounts

- **West Trading Post:** raw resources bought from your left neighbour cost 1 coin each.
- **East Trading Post:** raw resources bought from your right neighbour cost 1 coin each.
- **Marketplace:** manufactured resources bought from either neighbour cost 1 coin each.

The discount applies to every unit of the named family bought from the named neighbour, whichever of their symbols supplies it (see [Ambiguities](#ambiguities), item 9). The discounts do not stack below 1 coin; owning both trading posts simply discounts raw resources from both sides.

### Chains

- From Age II on, a card may show a chain symbol as a free-construction condition. If your city already holds a card that provides that symbol (shown in its top right corner), you may construct the card for free, paying neither coins nor resources.
- A chain is an option, not an obligation: without the symbol you can still pay the card's normal cost.
- A card may provide two chain symbols.
- In the base game there are 24 chain symbols. Each one is provided by an Age I or Age II card (one symbol is provided by two cards), and each one makes exactly one later card free. Two chains run from Age I straight to Age III.

### Free construction from wonder abilities

The Halikarnassos and Olympia abilities construct a building for free: no coins, no resources, no buying. A free building still follows every other rule: it must not share a name with a building already in the city, and its effect and chain symbols work as usual.

### What a player may never do

- Spend more coins than they hold, or go into debt.
- Construct a building whose name is already in their city.
- Build a stage out of order, or a stage beyond the last one on their side.

## Order within a turn

The rulebook describes the turn in three steps; the engine needs an exact order. This order follows the rulebook's constraints (simultaneous play, coins from the start of the turn, no buying from same-turn buildings, coins for cards built by neighbours this turn).

1. **Choose.** Every player commits one card and one action, including, for a construction or a stage, how they pay: the coins that go to each neighbour, picked from the outcomes the server offers (see [Ambiguities](#ambiguities), item 10). Choices stay hidden until all are in.
2. **Check.** Each choice is checked against the state at the start of the turn: the player's own treasure and city, and the neighbours' cities. Because nothing that happens this turn can change those inputs, a choice that was legal when made is still legal at the reveal. Simultaneous play therefore never needs conflict resolution.
3. **Pay.** Each player pays coin costs to the reserve and trade prices to the neighbours. Coins a player receives in this step are credited but cannot be spent this turn.
4. **Place.** Constructed buildings enter their cities, stage cards go under their boards, sold cards go to the discard pile and sellers take 3 coins.
5. **Immediate effects.** Coin effects of the cards and stages placed this turn are resolved (for example Tavern, Vineyard, Lighthouse, Arena, Ephesos stages). Effects that count cards in neighbours' cities count the cards those neighbours placed in step 4 (the effects sheet says so for Vineyard and Bazar). Shields and production simply exist from now on.
6. **End-of-turn effects.** A player who built a Halikarnassos stage with the discard ability this turn now takes the discard pile, picks one card and constructs it for free (see [Halikarnassos](#halikarnassos-the-mausoleum)).
7. **Pass** the remaining hands (on turns 1 to 5).

On the sixth turn, the leftover second card of each player is discarded in step 1, when the player chooses, so it is already in the discard pile at step 6.

## Military conflicts

At the end of each Age:

1. Each player counts their military strength: every shield on their buildings and on their built wonder stages.
2. Each pair of neighbours compares strengths once. For each of your two neighbours:
   - equal strength: neither takes a token;
   - lower strength: you take a defeat token (minus 1 point);
   - higher strength: you take the victory token of the current Age: 1 point in Age I, 3 in Age II, 5 in Age III.

So with N players there are N conflicts per Age (one per adjacent pair of seats), and each player is in exactly two. With 3 players every player fights both others. A player can collect at most two tokens per Age. Defeat tokens are the same in every Age.

Shields count only at these comparisons. They do not move anything during the Age, and they are never lost.

The token supply always suffices: an Age hands out at most N victory tokens (8 exist per Age) and the whole game at most 3N defeat tokens (24 exist).

## Science

Green cards carry one science symbol each: compass, gear or tablet. At the end of the game, the science score of a city is the sum of two parts:

- **Same symbols:** for each of the three symbols, the number of that symbol in the city, squared (1, 4, 9, 16, 25, 36 points for 1 to 6 identical symbols, and so on).
- **Sets:** 7 points for each complete set of three different symbols (the number of sets is the smallest of the three counts).

**Choice symbols.** The Scientists Guild and one stage on each side of the Babylon board each give one science symbol of the owner's choice, decided at the end of the game. Each choice symbol is chosen separately. The engine picks the choices that maximise the player's score (see [Ambiguities](#ambiguities), item 13).

## Commercial buildings (yellow)

- **Tavern** (Age I): take coins from the reserve once, when built.
- **West Trading Post, East Trading Post, Marketplace** (Age I): commerce discounts, described above.
- **Caravansery** (Age II): a choice producer of one raw resource each turn. **Forum** (Age II): a choice producer of one manufactured resource each turn. Neither can be bought by neighbours.
- **Vineyard** (Age II): when built, take coins for every brown card in your city and in both neighbours' cities, including cards the neighbours built this turn. **Bazar** (Age II): the same for grey cards, at a higher rate per card.
- **Lighthouse, Haven, Chamber of Commerce, Ludus** (Age III): when built, take coins for each card of one colour in your own city (yellow, brown, grey and red respectively); at the end of the game, score points for each card of that colour in your city. The Lighthouse counts itself.
- **Arena** (Age III): when built, take coins for each stage of your wonder already built; at the end of the game, score points for each built stage.

Coins from these effects come from the reserve. Points from them are scored in the yellow category at the end.

## Guilds (purple)

Guilds appear only in Age III, N + 2 of the 10 per game. Each scores at the end of the game:

- **Workers, Craftsmens, Magistrates, Traders, Spies, Philosophers Guilds:** points for each card of one colour (brown, grey, blue, yellow, red and green respectively) in your two neighbours' cities, not your own.
- **Builders Guild:** points for each wonder stage built in your city and in both neighbours' cities.
- **Shipowners Guild:** points for every card in your own city that is brown, grey or purple, counting itself.
- **Decorators Guild:** points only if every stage on your side of your wonder board is built.
- **Scientists Guild:** one science symbol of your choice (see [Science](#science)).

## Wonder boards

There are seven boards. Each board has one starting resource, the same on both sides. A day side always has three stages: the first gives points, the second gives the board's special effect (on Gizah, more points), the third gives more points. Night sides vary: two, three or four stages, mixing effects and points.

The abilities below are described by what they do and when they apply. The stage costs are in the private draft table.

### Alexandria: the Lighthouse

- Starting resource: glass.
- Day: stage 1 points; stage 2 a choice producer of one raw resource each turn; stage 3 points.
- Night: stage 1 a choice producer of one raw resource; stage 2 a choice producer of one manufactured resource; stage 3 points.
- Wonder production can be used by its owner on every turn from the stage's construction on, but neighbours can never buy it.

### Babylon: the Hanging Gardens

- Starting resource: wood.
- Day: stage 1 points; stage 2 one science symbol of your choice at the end of the game; stage 3 points.
- Night (two stages, no points): stage 1 the **last card ability**; stage 2 one science symbol of your choice.
- **Last card ability.** In every Age from the one in which this stage is built, the owner does not discard the second card of the sixth turn. After every player has played their sixth card, the owner plays that card as an extra turn of their own, with any of the three actions and all the usual rules (paying, buying from neighbours, chains). The extra turn comes before the military conflicts. If the ability is built during the sixth turn itself, it already works for that Age (see [Ambiguities](#ambiguities), item 1). Built in Age I, it gives three extra cards over the game.

### Ephesos: the Temple of Artemis

- Starting resource: papyrus.
- Day: stage 1 points; stage 2 a one-time coin gain from the reserve; stage 3 points.
- Night: each of the three stages gives a one-time coin gain and points.

### Gizah: the Pyramids

- Starting resource: stone.
- Day: three stages of points.
- Night: four stages of points. Gizah night is the only side with four stages.

### Halikarnassos: the Mausoleum

- Starting resource: cloth.
- Day: stage 1 points; stage 2 the **discard ability**; stage 3 points.
- Night: every one of the three stages gives the discard ability; the first two also give points.
- **Discard ability.** At the end of the turn in which the stage is built, after every player's card of that turn has been placed and every sold card is in the discard pile, the owner looks through the whole discard pile, picks one Age card and constructs it for free. The ability fires once per stage, on that turn only; it is not stored. If no card in the pile can legally be constructed (the pile is empty, or every card shares a name with a building already in the city), nothing happens. See [Ambiguities](#ambiguities), items 2 and 3.

### Olympia: the Statue of Zeus

- Starting resource: clay.
- Day: stage 1 points; stage 2 the **first of each colour** ability; stage 3 points.
- Night: stage 1 the **first of each Age** ability and points; stage 2 the **last of each Age** ability and points; stage 3 points.
- **First of each colour** (day stage 2). From then on, constructing a building of a colour your city does not hold yet is free. Each of the seven colours can benefit once. A colour already in the city when the stage is built gets nothing.
- **First of each Age** (night stage 1). The building you construct on the first turn of each Age is free. In the base game the stage cannot be built before the first turn of Age I, so this first applies on the first turn of the next Age. If you build a stage or sell on that first turn, the ability is lost for that Age.
- **Last of each Age** (night stage 2). The building you construct on the sixth turn of each Age (the card you keep, not the one you discard) is free. It already applies in the Age in which the stage is built, if it was built before the sixth turn. If you build a stage or sell on the sixth turn, the ability is lost for that Age.
- These abilities make Age cards free only: never wonder stages.

### Rhodos: the Colossus

- Starting resource: ore.
- Day: stage 1 points; stage 2 shields; stage 3 points.
- Night (two stages): each stage gives shields, a one-time coin gain and points.

## End of the game and scoring

The game ends after the military conflicts of Age III. Each player adds up seven categories, in the order of the score pad:

1. **Wonder:** the points on the built stages of their wonder.
2. **Treasure:** 1 point per full 3 coins in their treasure; leftover coins score nothing.
3. **Military:** the sum of their conflict tokens. It can be negative.
4. **Blue cards:** the points on their blue cards.
5. **Yellow cards:** the end-of-game points of their Age III yellow cards.
6. **Green cards:** the science score, including choice symbols.
7. **Purple cards:** the guilds' points.

The highest total wins. On a tie, the tied player with the most coins wins. If they also have the same coins, they share the victory.

## Special to some player counts

- **3 players:** each player's two neighbours are the two other players. Every player trades with both and fights both each Age. Age III uses 5 guilds.
- **4 to 7 players:** more card copies enter each deck, and more guilds (N + 2).
- **7 players:** all 7 boards are in play, all 148 Age cards are used except one guild.
- No rule other than the deck contents and the guild count depends on N.

## For developers

### Sizes

- Seats: 3 to 7. Hand: 7 cards at the start of an Age, one fewer each turn. Turns per Age: 6, plus one extra turn for a Babylon night player with stage 1.
- A city can hold at most 20 buildings: a Babylon night player who builds only stage 1 in Age I plays 21 cards and uses 1 for the stage. Under every other board the bound is 18. A board holds at most 4 tucked cards (Gizah night).
- Data size: 80 card designs (70 Age cards, with Glassworks, Press and Loom present in two Ages, and 10 guilds), 77 distinct names, 148 physical cards; 7 boards, 14 sides, 41 stages.

### Hidden information

- **Hidden from everyone but the owner:** a player's hand; the card a player has chosen, until the reveal; the identity of the card tucked under a stage; the identity of a sold or discarded card.
- **Hidden from everyone:** the order of the undealt cards, and which guilds were drawn into the Age III deck (and so which ones are absent).
- **Seen once:** the Halikarnassos owner sees the whole discard pile when the ability fires.
- **Public:** every city's buildings, built stages, starting resource and chosen side; every treasure; every conflict token; the action each player took on a turn (construct, stage or sell) and the building constructed; the number of cards in every hand and in the discard pile; whose hand passes to whom.
- Players also remember the hands that passed through them. A bot or the client may track those cards; the server must not reveal more than the rules do.

### Determinism

All randomness is resolved at setup: deck order for each Age, the guild draw, the board deal, and the seating. The side choice is a player decision taken before Age I. Everything after that follows from the players' actions.

## Ambiguities

Each item gives a provisional ruling and its reason. None of these is settled by the publisher's FAQ unless it says so.

1. **Babylon night, stage 1 built on the sixth turn.** *Ruling:* the ability already works in that Age: a player who commits the sixth card to building stage 1 keeps the second card instead of discarding it (ruling 25), and plays it after everyone's sixth card. *Reason:* the effects sheet says the ability works once the stage is built, and the extra card is played after all other sixth cards, which is after the stage exists; the fan wiki reports the publisher's own app plays it this way. The FAQ is silent. *Alternative:* only from the next Age.
2. **Halikarnassos: which cards it can take.** *Ruling:* every Age card in the discard pile at step 6 of the turn, which includes cards sold that turn by any player and, on the sixth turn, the discarded second cards (but not a Babylon owner's kept card). *Reason:* the ability fires at the end of the turn, and the sixth-turn leftovers are discarded at its start. *Alternative:* only cards discarded before the turn began.
3. **Halikarnassos: declining.** *Ruling:* if a legal card exists, the owner must take one. *Reason:* the effect says to choose one and build it, and in the base game a building never lowers a score, so a refusal is never useful. *Alternative:* allow declining, at the cost of one more decision node.
4. **Babylon's extra card: the allowed actions.** *Ruling:* any of the three actions, including a wonder stage (which could be Babylon's own stage 2) and selling. *Reason:* the effects sheet says the card is played under the usual rules as a new turn. *Alternative:* construction only.
5. **Babylon's extra turn and coins.** *Ruling:* it is a separate turn, so coins gained on the sixth turn can be spent on it, and the neighbours' sixth-turn buildings can be bought from. *Reason:* the sheet calls it a new turn, and the coin and same-turn restrictions are stated per turn. *Alternative:* treat it as part of the sixth turn.
6. **Olympia day: colours already present.** *Ruling:* a colour that is already in the city when stage 2 is built gives nothing; each other colour makes its first building free once. *Reason:* the publisher's FAQ says the free card is the first card of a colour not yet in your city.
7. **Olympia night, stage 1 timing.** *Ruling:* it applies on the first turn of every Age after the one in which it was built. It is never stored. *Reason:* the publisher's FAQ.
8. **Olympia night, stage 2 timing.** *Ruling:* it applies to the card played on the sixth turn of the Age in which the stage was built (if built earlier) and of every later Age, never to the discarded card. *Reason:* the publisher's FAQ.
9. **Commerce discounts and starting resources.** *Ruling:* a trading post or the Marketplace discounts every unit of its family bought from the named neighbour, including the neighbour's starting resource. *Reason:* the effects sheet defines the discount by resource family and neighbour, not by card colour. *Alternative:* card symbols only.
10. **Choosing the payment.** *Ruling:* the player picks one payment outcome from a list the server offers, and the server accepts exactly the outcomes it offered. A payment outcome is the pair of coin amounts that go to the left and to the right neighbour; different ways of covering a cost that end in the same pair are one outcome. For each building or wonder stage the player can afford, the list holds only the outcomes that no other outcome beats for the buyer, where one outcome beats another if it pays no more to either neighbour and less to at least one. Buying a unit that the buyer's own production could supply is never offered: the rules allow buying only missing units, and such a plan is beaten anyway. *Reason:* which rival receives the coins can matter, so the choice among unbeaten outcomes stays with the player. Leaving out beaten outcomes is a deliberate restriction of what the rules allow, not a claim that they never matter: extra coins can change what a neighbour buys or builds later, and a player might want that. We give up that rare play because offering every pair would mean up to 225 moves for one construction. A short canonical list keeps the server's check a plain membership test (D02) and bounds the number of moves offered. *Alternative:* always take the cheapest split, ties broken toward the left neighbour.
11. **Coins gained on a turn.** *Ruling:* coins received during a turn are credited at once but cannot be spent until the next turn; a payment's affordability is checked against the start-of-turn treasure. *Reason:* the rulebook and the FAQ both say so.
12. **Same-turn counts.** *Ruling:* immediate coin effects count the cards placed in step 4 of the same turn, including the neighbours' (the effects sheet says so for Vineyard and Bazar), but not a Halikarnassos free building of the same turn, which comes later in step 6. *Reason:* the sheet covers the neighbours' same-turn cards; the Halikarnassos ability fires at the end of the turn. *Alternative:* count the free building too.
13. **Choice science symbols.** *Ruling:* chosen automatically at scoring to give the owner the highest science score. *Reason:* the choice is made only for scoring and the best choice is always computable, so asking the player adds nothing. *Alternative:* ask the player at the end of the game.
14. **Unlimited coins.** *Ruling:* the reserve never runs out. *Reason:* the rulebook sets no limit, and no rule covers an empty reserve. *Alternative:* cap at the 126 coins in the box.
15. **Seating.** *Ruling:* seats are randomised at game creation from the seed; a host may arrange them before the start. *Reason:* the rulebook assumes players sit around a table and gives no order; seat order decides neighbours and so matters.
16. **Wonder sides.** *Ruling:* a room option: each player picks their side after seeing their board (the rulebook's default), all day sides, or random sides. *Reason:* the rulebook gives the first and recommends the second for beginners; random sides are useful for bots and quick games.
17. **What the action reveals.** *Ruling:* at the reveal, every player sees each player's action kind and any constructed building, but not which card was tucked under a stage or sold. *Reason:* the rulebook has those cards placed face down.
18. **Same name across ages.** *Ruling:* a city holds at most one card of each name, so the Age I and Age II Glassworks (and Press and Loom) exclude each other. *Reason:* the rulebook's rule is stated by name.
19. **The Decorators Guild on short sides.** *Ruling:* "every stage" means every stage of the side in play (2 on Babylon and Rhodos night, 4 on Gizah night). *Reason:* that is the only reading that works for every side.
20. **The tie-break.** *Ruling:* compare coins held at the end of the game; players still tied all win. *Reason:* the rulebook.
21. **A free construction is offered only as free.** *Ruling:* when a chain or an Olympia ability makes a building free, the only payment offered for it is the free one: no coins to the reserve, no resources, no purchases. *Reason:* the rules present a chain as an option, but paying instead only gives coins away and saves nothing for later. Olympia's day ability depends on whether the colour is already in the city, and the first building of that colour fills it however it was paid for; the night abilities belong to one particular turn. *Alternative:* offer the paid outcomes as well.
22. **Babylon's kept card cannot be declined.** *Ruling:* a Babylon night owner who keeps the second card of the sixth turn must play it with one of the three actions; selling is always possible. *Reason:* the effects sheet says the owner may play the card, but declining would only throw it away, which is never better than selling it for 3 coins in the base game, where coins never lower a score. This simplifies an optional ability for the base game only; an expansion may need a pass. *Alternative:* allow a pass that discards the card for nothing.
23. **Halikarnassos with nothing to build.** *Ruling:* when the discard ability fires, the owner looks through the whole pile even if no card in it can be built; then nothing is built and play goes on. Every player sees that the ability fired and that no building came of it. *Reason:* the owner learns the pile by searching it either way, so what the owner knows should not depend on whether a legal card happened to be there. The empty outcome is public because the cities are public, and when the pile was not empty it tells the others that every card in it shares a name with one of the owner's buildings.
24. **Halikarnassos and its owner's own leftover.** *Ruling:* a Halikarnassos owner who builds a discard stage on the sixth turn may take its own leftover second card from the pile and construct it for free, so that owner plays both cards of the sixth hand. *Reason:* it follows from ruling 2: the leftover is in the pile by step 6, and the rules make no exception for the owner's own card. It may surprise players, and it rests on the timing of ruling 25. *Alternative:* exclude the owner's own leftover (no rule supports this).
25. **When the sixth-turn leftover is discarded.** *Ruling:* at the start of the sixth turn, when the player chooses, as the rulebook describes; rulings 1, 2 and 24 rest on this. *Reason:* the rulebook gives the sixth turn as a concrete sequence (receive two cards, keep one, discard the other, play), and the French rulebook agrees. The effects sheet, describing Babylon's night ability, speaks of the card a player would otherwise throw away when the Age ends; we read that as a loose way of naming the same card, not as a timing rule. The fan wiki's report that the publisher's app lets Babylon use a stage built on the sixth turn supports ruling 1. The point stays provisional. *Alternative:* discard the leftovers after the sixth turn; then Halikarnassos could not take them on that turn, and ruling 24 would not arise.
26. **How sides are chosen.** *Ruling:* under the default option of ruling 16, every player chooses a side at the same time, after all boards are visible and before Age I is dealt; the choices stay hidden until all are in and are then shown together. *Reason:* the rulebook has each player pick a side for the board received but does not say whether openly or in turn. Choosing at once removes any advantage from choosing last and needs only one decision. This is our convention for online play, not a rule the rulebook states. *Alternative:* open choices in seat order, with the first seat always choosing blind.
27. **Payments are public.** *Ruling:* for every play, every player sees how many coins it paid to each neighbour, including for a wonder stage, whose tucked card stays hidden. *Reason:* at the table the buyer hands the coins over in plain view, and treasures are public; with several transfers in one turn the treasures alone would not always show who paid whom. *Alternative:* show only the resulting treasures.
28. **What a free building from the discard counts.** *Ruling:* a building constructed by the Halikarnassos ability resolves its own coin effect when it is placed in step 6, counting the cities as they stand then: every card placed in step 4 of that turn, the neighbours' included, and the building itself where its effect counts its own colour. Coin effects already resolved in step 5 are not recomputed. *Reason:* it is placed after everything else that turn, and ruling 12 already settles what the step 5 effects count. *Alternative:* count the cities as they stood at the start of the turn.
29. **The sixth turn's discard build comes before Babylon's kept card.** *Ruling:* on the sixth turn, a Halikarnassos discard build (step 6) happens before the Babylon night owner's extra turn. That extra turn therefore sees the free building, and may buy its production if it belongs to a neighbour and is tradeable (brown or grey). *Reason:* the discard ability belongs to the turn in which its stage is built, and the effects sheet puts the kept card after every other player's last card. *Alternative:* play Babylon's card first.
30. **Left, right and the trading posts.** *Ruling:* a player's left neighbour is the one who receives their hand in Ages I and III. The West Trading Post discounts raw resources bought from the left neighbour, and the East Trading Post those bought from the right neighbour. *Reason:* the passing rule defines left; the effects sheet says an arrow on each trading post sets its direction, and the second-edition fan wiki gives West as left and East as right. *Alternative:* none; the printed arrows decide (see Open below).

### Open

The engine design and its review agree on every rules question they raised, so no question is left open for lack of agreement. One point needs the printed cards:

- **The trading-post arrows.** Ruling 30 rests on the fan wiki and on the effects sheet's statement that an arrow sets the direction. The engine design and its review both ask the blind data tables to record each trading post's direction from the printed card, so that the reading is checked rather than assumed. A screen layout must also never reverse the meaning of left and right.
