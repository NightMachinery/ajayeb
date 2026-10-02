# Economy

This page covers coins, resources and costs: how trading is priced, the cards that fix prices or produce flexibly, chains, the Masonry and Architecture discounts, the Economy token, coin rewards, and how Ajayeb finds the cheapest way to pay. Terms such as *trade price* and *chain* are defined in the [README](README.md#terms-used-throughout).

## Coins

- Each player starts with 7 coins. The bank never runs out (see [decisions.md](decisions.md#unlimited-coins)).
- **Gaining coins:** discarding a card; the coin rewards of yellow cards, guilds and wonders; the Agriculture and Urbanism tokens; and, with the Economy token, the coins your opponent spends on trading.
- **Losing coins:** paying costs and trading; looting tokens on the military track (see [military-science.md](military-science.md#looting-tokens)); and the Appian Way built by your opponent.
- **Coins never go below zero.** A loss larger than what a player has takes everything they have, and nothing more.
- At the end of the game, a player scores 1 victory point for every 3 coins they hold, ignoring any remainder (see [ages-scoring.md](ages-scoring.md#the-eight-scoring-categories)).

## Resources

There are five resources in two families:

- **raw goods:** wood, clay and stone, produced by brown cards;
- **manufactured goods:** glass and papyrus, produced by grey cards.

The producers:

- Age I brown cards produce 1 unit each: Lumber Yard and Logging Camp (wood), Clay Pool and Clay Pit (clay), Quarry and Stone Pit (stone).
- Age II brown cards produce 2 units each: Sawmill (wood), Brickyard (clay) and Shelf Quarry (stone).
- Grey cards produce 1 unit each: Glassworks (Age I) and Glass-Blower (Age II) make glass; Press (Age I) and Drying Room (Age II) make papyrus.

**Production is never spent.** On every turn, each unit you produce can cover one unit of the cost you are paying. A player with a Lumber Yard and a Sawmill can cover up to 3 wood in every cost, turn after turn. Production cannot be saved up, given away or sold.

## Costs

A card's or a wonder's cost can be nothing, a number of coins, a list of resource units, or coins and resources together. To pay it:

- pay the printed coins to the bank;
- cover each resource unit with your own production, with a flexible producer, with a discount, or by buying it.

Some cards can instead be constructed for free through a chain (see [Chains](#chains)).

## Trading

Any resource unit you cannot cover yourself, you buy from the bank.

**The price rule.** One unit of a resource costs **2 coins, plus 1 coin for each unit of that resource your opponent produces with brown and grey cards.**

- A card that produces 2 units counts as 2.
- Your opponent's flexible production (yellow cards and wonders) never raises your price.
- Your own production does not change your price; it only reduces how much you need to buy.
- The coins go to the bank, not to your opponent, unless your opponent owns the Economy token.
- There is no limit on how many units you buy. Your opponent cannot refuse, and their production is not used up by your purchase.
- Prices move during the game. They rise when your opponent constructs a brown or grey card, and fall at once when the Statue of Zeus or Circus Maximus destroys one.

Example: your opponent owns a Clay Pool and a Brickyard (3 clay in total) and a Caravansery. Each clay unit costs you 2 + 3 = 5 coins. The Caravansery adds nothing to the price. A card needing 2 clay, when you produce none, costs you 10 coins of trading.

## Fixed prices

Four yellow cards set your price for some resources to **1 coin per unit**, whatever your opponent produces:

- Stone Reserve (Age I): stone.
- Clay Reserve (Age I): clay.
- Wood Reserve (Age I): wood.
- Customs House (Age II): glass and papyrus.

A fixed price helps only its owner; it does not change what the opponent pays.

The physical cards say the fixed price applies from the owner's next turn. These four cards cost only coins, so no resource is ever bought on the turn one of them is constructed, and the delay changes nothing. Ajayeb does not model it (see [decisions.md](decisions.md#the-fixed-price-delay)).

## Flexible production

Four components produce **one unit of a choice of resources** each turn:

- Forum (Age II yellow card): glass or papyrus.
- Caravansery (Age II yellow card): wood, clay or stone.
- The Great Lighthouse (wonder): wood, clay or stone.
- Piraeus (wonder): glass or papyrus.

Each covers one unit per turn, of whichever of its resources is most useful for the cost at hand, chosen afresh every time. Two flexible producers cover two units, for example Forum and Piraeus together can cover one glass and one papyrus, or two glass. Flexible production never raises the opponent's trade price.

## Chains

Many cards show a **chain symbol**, and a card of a later age shows the same symbol as an alternative to its cost. If your city holds the earlier card, you may construct the later card for free, ignoring both its coins and its resources.

- Only your own city counts. Your opponent's chain cards do nothing for you, and neither does a card tucked under a wonder.
- Chains apply to buildings only, never to wonders.
- Each chain symbol links exactly one card to exactly one card of a later age. A chained card can still be constructed by paying its normal cost.
- Chains never break: the only cards that can be destroyed are brown and grey, and none of them carries a chain symbol.
- With the Urbanism token, every free construction through a chain also pays you 4 coins.

The base game has 17 chains, listed here by the earlier card and the later card it frees:

- Age I to Age II: Stable to Horse Breeders; Garrison to Barracks; Scriptorium to Library; Pharmacist to Dispensary; Theater to Statue; Altar to Temple; Baths to Aqueduct.
- Age I to Age III: Palisade to Fortifications; Tavern to Lighthouse.
- Age II to Age III: Archery Range to Siege Workshop; Parade Ground to Circus; School to University; Laboratory to Observatory; Brewery to Arena; Temple to Pantheon; Statue to Gardens; Rostrum to Senate.

Temple and Statue appear twice: each is freed by an Age I card and in turn frees an Age III card.

## Discounts: Masonry and Architecture

Two progress tokens make future constructions cheaper:

- **Masonry:** every blue card you construct afterwards costs 2 fewer resource units.
- **Architecture:** every wonder you build afterwards costs 2 fewer resource units.

Details:

- The two units can be of any kind, the same or different.
- Printed coins are not reduced.
- Ajayeb waives the two units that would otherwise cost you the most.
- If fewer than two units remain to be paid after your own production and your flexible producers, the unused part of the discount is lost. It is never refunded in coins.

Example: the Palace costs one clay, one stone, one wood and two glass. Suppose you produce none of these, and your opponent produces one stone and one glass. Your prices are 2 for clay and wood and 3 for stone and glass, so the Palace costs 2 + 3 + 2 + 3 + 3 = 13 coins. With Masonry, two of the 3-coin units are waived, for 7 coins. With Masonry and a Forum, the Forum covers one glass and Masonry waives the stone and the other glass, leaving only the clay and the wood, for 4 coins.

## The Economy token

When your opponent buys resources, **the coins they pay for those resources come to you** instead of going to the bank.

- Only the trading part counts, at the price actually paid, including the 1-coin fixed prices and after any discount.
- Printed coin costs still go to the bank.
- The coins arrive as part of the payment, before any other effect of the card or wonder, so they count before a looting token or the Appian Way takes coins from you (see [decisions.md](decisions.md#effect-order)).

Example: you own Economy and produce one glass. Your opponent constructs a Caravansery, which costs 2 coins plus one glass and one papyrus, and they produce neither. Glass costs them 3 and papyrus 2, so they pay 7 coins in total. The bank receives the 2 printed coins and you receive 5.

## Coin rewards

Some cards pay coins once, from the bank, when they are constructed. That includes a construction through the Mausoleum. Changes to the cities afterwards do not alter a reward already paid.

- **Tavern** (Age I): 4 coins.
- **Brewery** (Age II): 6 coins.
- **Age III yellow cards** count only your own city, after the card has entered it. Each is also worth 3 victory points at the end.
  - Arena: 2 coins per wonder you have built.
  - Chamber of Commerce: 3 coins per grey card in your city.
  - Port: 2 coins per brown card in your city.
  - Lighthouse: 1 coin per yellow card you own, the Lighthouse included.
  - Armory: 1 coin per red card in your city.
- **Guilds** pay coins according to the city that has the most cards of their kind, as described in [ages-scoring.md](ages-scoring.md#guilds).

Wonders and tokens that pay coins are described in [wonders-tokens.md](wonders-tokens.md).

## The cheapest payment

Ajayeb never asks how you want to pay. The engine finds the cheapest payment and applies it.

**No other payment is ever better.** The printed coins are fixed, so a cheaper total always means less trading. Less trading also means fewer coins for an opponent who owns Economy. The cheapest payment is therefore best in every respect.

**How it is found:**

1. If your city holds the matching chain card, the construction is free (and Urbanism pays 4 coins).
2. Otherwise start from the printed coins.
3. Subtract your own brown and grey production from each resource in the cost.
4. Price each remaining unit: 1 coin with a matching fixed price, otherwise 2 plus the opponent's brown and grey production of that resource.
5. Let each flexible producer cover the most expensive remaining unit it can produce.
6. If Masonry (for a blue card) or Architecture (for a wonder) applies, waive the two most expensive remaining units.
7. Buy the rest. The construction is allowed when the total is no more than your coins.

This step-by-step method gives the true minimum for the base game because of how the flexible producers are made. Each produces either only raw goods or only manufactured goods, and the producers within one family are interchangeable, so covering the dearest unit first can never be a mistake. The engine's tests compare the method against an exhaustive search over every way to pay.

## For developers

- **The payment breakdown** that the engine reports for a construction contains: whether it is free through a chain, and from which card; the printed coins; the units covered by own production; which flexible producers covered which resources; the units waived by a discount, and by which token; the units bought, each resource with its unit price and whether the price was fixed; the trade total and the grand total; the part of the payment that goes to the opponent through Economy; the Urbanism bonus; and whether the player can afford it.
- **The view carries both seats' breakdowns** for every accessible card, each player's trade price per resource, each player's discard value, and the cost of each unbuilt wonder for its owner. The client shows these numbers and never computes a cost itself.
- **Integers only.** All of this is integer arithmetic over arrays indexed by resource; the rules code uses no floating point and no maps.
- **The precondition of the fast method** (every flexible producer covers exactly the raw goods or exactly the manufactured goods) is asserted when the catalog loads.
