// English messages for Haftgane's rules reference (games/haftgane/rules/*,
// the /rules/haftgane page and the in-game rules panel). See
// ../messages.svelte.ts for the conventions.
//
// The rules are written for players in our own words from
// docs/haftgane/rules.md, never in the rulebook's (docs/legal.md). One key
// per paragraph, list item or definition; games/haftgane/rules/sections.ts
// arranges them into sections. Board, wonder and card names are not written
// here: a {placeholder} names them, and sections.ts fills it from the name
// keys of ./names.ts. Numbers in the text are fixed rule numbers, never
// game state.

import type { Messages } from '$lib/i18n';

export const rules = {
  // The page, the panel and the entry points.
  'hg.rules.title': 'Haftgane rules',
  'hg.rules.pageTitle': 'How to play Haftgane',
  'hg.rules.intro':
    'The rules of Haftgane, the game of wonders for three to seven players, in short. The game applies every rule for you, so come back here whenever a turn surprises you.',
  'hg.rules.toc': 'Contents',
  'hg.rules.close': 'Close the rules',
  'hg.rules.backHome': 'Home',
  'hg.rules.openPage': 'Open the rules as a full page',

  // 1. The goal.
  'hg.rules.goal.title': 'The goal',
  'hg.rules.goal.p1':
    'Three to seven players each grow a city around a wonder, over three Ages. Everyone plays at the same time: on each turn you choose a card from your hand and use it, then pass the rest of the hand to a neighbour.',
  'hg.rules.goal.p2':
    'After Age III the player with the most victory points wins. Points come from your wonder, your coins, your military victories and your buildings of every colour.',
  'hg.rules.goal.cardsTitle': 'The seven colours of card',
  'hg.rules.goal.card.brown': 'Produce raw resources: wood, clay, stone and ore.',
  'hg.rules.goal.card.grey': 'Produce manufactured resources: glass, papyrus and cloth.',
  'hg.rules.goal.card.blue': 'Score victory points.',
  'hg.rules.goal.card.green': 'Carry science symbols, which score together at the end.',
  'hg.rules.goal.card.yellow':
    'Bring coins, cheaper trade or a resource of your choice. Some in Age III also score points.',
  'hg.rules.goal.card.red': 'Carry shields, your military strength.',
  'hg.rules.goal.card.purple':
    "Guilds, in Age III only. Each scores in its own way, often by counting your neighbours' cities.",

  // 2. Terms.
  'hg.rules.terms.title': 'Terms',
  'hg.rules.terms.neighbours.term': 'Neighbours',
  'hg.rules.terms.neighbours':
    'The players seated directly to your left and to your right. You trade with them and fight them, and with nobody else. With three players, your neighbours are the two others.',
  'hg.rules.terms.city.term': 'City',
  'hg.rules.terms.city':
    'Your wonder board with its built stages, and every card you have constructed. A constructed card is a building.',
  'hg.rules.terms.age.term': 'Age',
  'hg.rules.terms.age':
    'One of the three rounds of the game. Each Age has its own deck and six turns.',
  'hg.rules.terms.hand.term': 'Hand',
  'hg.rules.terms.hand':
    'The cards you hold and choose from. It starts each Age with seven cards and loses one each turn.',
  'hg.rules.terms.board.term': 'Wonder board',
  'hg.rules.terms.board':
    'Your board shows a starting resource and a row of stages. It has a day side and a night side, and you play one of them for the whole game.',
  'hg.rules.terms.stage.term': 'Stage',
  'hg.rules.terms.stage':
    'One step of your wonder. You build it by paying its cost and tucking a card under it, always the next stage from the left.',
  'hg.rules.terms.treasure.term': 'Treasure',
  'hg.rules.terms.treasure':
    'The coins you hold. Coins come from and go to the reserve, which never runs out.',
  'hg.rules.terms.discard.term': 'Discard pile',
  'hg.rules.terms.discard':
    'The face-down pile of sold cards and of the cards left over on the last turn of each Age.',
  'hg.rules.terms.production.term': 'Production',
  'hg.rules.terms.production':
    'Each resource symbol in your city gives one unit of its resource on every turn. It is never used up and cannot be saved for later.',
  'hg.rules.terms.choice.term': 'Choice producer',
  'hg.rules.terms.choice':
    'A symbol that gives one unit of one resource from a short list, picked afresh each time you use it.',
  'hg.rules.terms.chain.term': 'Chain symbol',
  'hg.rules.terms.chain':
    'A small symbol on a card. A later card that shows the same symbol as its condition is free for you once your city holds the earlier one.',
  'hg.rules.terms.strength.term': 'Military strength',
  'hg.rules.terms.strength': 'The number of shields in your city.',
  'hg.rules.terms.token.term': 'Conflict token',
  'hg.rules.terms.token':
    'What you win or lose against a neighbour at the end of each Age: a defeat, minus 1 point, or a victory worth 1, 3 or 5 points by Age.',

  // 3. Resources.
  'hg.rules.resources.title': 'Resources',
  'hg.rules.resources.p1': 'There are seven resources, in two families.',
  'hg.rules.resources.raw': 'Raw resources: wood, clay, stone and ore. Brown cards produce them.',
  'hg.rules.resources.manufactured':
    'Manufactured resources: glass, papyrus and cloth. Grey cards produce them; cloth comes from the {loom}.',
  'hg.rules.resources.p2':
    'Your wonder board produces its starting resource from the first turn. {caravansery} and {forum} each produce one resource of a family, of your choice, and some wonder stages produce too.',
  'hg.rules.resources.p3':
    'Production is never used up: the same symbols pay again on every turn. A card with two symbols gives two units.',

  // 4. Setup.
  'hg.rules.setup.title': 'Setup',
  'hg.rules.setup.seats':
    'Seats are drawn at random, unless the table keeps the order of the lobby. Your seat decides your two neighbours for the whole game.',
  'hg.rules.setup.decks':
    'Each Age has its own deck, built for the number of players: every card shows the smallest table it is used at, and the cards for larger tables stay in the box.',
  'hg.rules.setup.guilds':
    'Two more guilds than there are players are drawn unseen from the ten and shuffled into the Age III deck. Nobody knows which guilds are in the game.',
  'hg.rules.setup.boards':
    "Everyone receives a wonder board at random. Depending on the table's settings, each player chooses a side, all in secret and revealed together, or every board plays its day side, or the sides are drawn.",
  'hg.rules.setup.coins': 'Everyone starts with 3 coins.',
  'hg.rules.setup.p1': 'Every deck holds exactly seven cards for each player.',

  // 5. The structure of an Age.
  'hg.rules.age.title': 'An Age, turn by turn',
  'hg.rules.age.p1':
    'At the start of an Age everyone receives seven cards. The Age has six turns, and every turn has the same three steps.',
  'hg.rules.age.choose':
    'Choose: pick a card from your hand and what to do with it. Nobody sees your pick, only that you have made one. You can change it until the last player confirms.',
  'hg.rules.age.play': 'Play: all the picks are revealed and played at once.',
  'hg.rules.age.pass':
    'Pass: hand the cards you did not pick to a neighbour, and take the hand your other neighbour passes to you.',
  'hg.rules.age.direction':
    'Hands go to your left neighbour in Ages I and III, and to your right neighbour in Age II.',
  'hg.rules.age.sixthTitle': 'The sixth turn',
  'hg.rules.age.sixth':
    'On the sixth turn you receive two cards. You keep one and play it as usual, and the other goes face down onto the discard pile, for nothing. A night stage of {babylon} lets you play that second card too.',
  'hg.rules.age.endTitle': 'The end of an Age',
  'hg.rules.age.end':
    'After the sixth turn every player fights both neighbours (see Military conflicts). Then the next Age begins. Age III ends the game.',

  // 6. The three actions.
  'hg.rules.actions.title': 'The three actions',
  'hg.rules.actions.p1': 'With the card you picked, you do exactly one of these.',
  'hg.rules.actions.construct.term': 'Construct the building',
  'hg.rules.actions.construct':
    'Pay its cost and put it in your city, where it stays for the game. Its effect starts at once.',
  'hg.rules.actions.stage.term': 'Build a wonder stage',
  'hg.rules.actions.stage':
    'Pay the cost of your next stage and tuck the card face down under it. Any card will do, and nobody else learns which. Stages are optional, go from left to right, and can be built in any Age.',
  'hg.rules.actions.sell.term': 'Sell the card',
  'hg.rules.actions.sell':
    'Put it face down on the discard pile and take 3 coins from the reserve. You can always sell, whatever the card.',
  'hg.rules.actions.sameName':
    'A city never holds two buildings of the same name. {glassworks}, {press} and {loom} come in both Age I and Age II, but you can have only one of each.',

  // 7. Paying costs.
  'hg.rules.costs.title': 'Paying costs',
  'hg.rules.costs.p1':
    'A card or a stage is free, or costs coins, or costs resources. Age cards never cost both coins and resources.',
  'hg.rules.costs.coinsTitle': 'Coins',
  'hg.rules.costs.coins':
    'A coin cost goes from your treasure to the reserve. You can never spend more than you hold.',
  'hg.rules.costs.productionTitle': 'Your own production',
  'hg.rules.costs.production':
    "Every production symbol in your city pays one unit each turn: your board's starting resource, your brown and grey cards, your yellow choice producers and some wonder stages. What you lack, you can buy from your neighbours.",
  'hg.rules.costs.chainTitle': 'Chains',
  'hg.rules.costs.chain':
    'From Age II on, many cards show a chain symbol as their condition. If your city holds a card that provides that symbol, you may construct the card for free. Without it, you pay the usual cost.',
  'hg.rules.costs.chainSkip':
    'Two chains skip an Age, from a card of Age I straight to a card of Age III.',
  'hg.rules.costs.freeTitle': 'Free buildings from wonders',
  'hg.rules.costs.free':
    '{halikarnassos} and {olympia} can give you a building for free: no coins, no resources. Every other rule still holds, so you cannot take a second building of a name you already have.',

  // 8. Buying from neighbours.
  'hg.rules.trade.title': 'Buying from neighbours',
  'hg.rules.trade.p1':
    'When your own production falls short, buy the missing units from your two neighbours. Each unit costs 2 coins, paid to the neighbour who produces it.',
  'hg.rules.trade.forSale':
    'Only their starting resource and the symbols on their brown and grey cards are for sale. Yellow cards and wonder stages produce for their owner alone.',
  'hg.rules.trade.once':
    'Each of their symbols sells you one unit per turn at most. A card with two symbols sells two.',
  'hg.rules.trade.either': 'A card that offers a choice of two resources sells either one.',
  'hg.rules.trade.shared':
    'Your neighbour cannot refuse, and still uses their production as usual. Both of their neighbours may buy the same symbol on the same turn.',
  'hg.rules.trade.start':
    'You buy only from what stood at the start of the turn, and pay only with the coins you held then. Coins you receive during a turn wait for the next one.',
  'hg.rules.trade.now': 'Bought units are used at once; they cannot be kept.',
  'hg.rules.trade.splitTitle': 'Who receives your coins',
  'hg.rules.trade.split':
    'When a cost can be covered in several ways, the game lists each split of coins between your left and right neighbour that no other split beats, and you pick one. A split beats another when it pays no more to either neighbour and less to one of them.',
  'hg.rules.trade.discountTitle': 'Trading discounts',
  'hg.rules.trade.discount.west': 'Raw resources bought from your left neighbour cost 1 coin each.',
  'hg.rules.trade.discount.east':
    'Raw resources bought from your right neighbour cost 1 coin each.',
  'hg.rules.trade.discount.marketplace':
    'Manufactured resources bought from either neighbour cost 1 coin each.',
  'hg.rules.trade.discount.p1':
    'A discount covers every unit of its family from that neighbour, their starting resource included. It never goes below 1 coin: with both trading posts, raw resources cost 1 coin from either side. Your left neighbour is the one who receives your hand in Ages I and III.',

  // 9. The order within a turn.
  'hg.rules.order.title': 'How a turn resolves',
  'hg.rules.order.p1':
    'Everyone plays at once, so each turn is settled in a fixed order. Nothing you do on a turn can depend on what another player does on the same turn.',
  'hg.rules.order.choose':
    'Choose: everyone picks a card, an action and, when buying, a split of coins.',
  'hg.rules.order.check':
    'Check: each pick is checked against the position at the start of the turn, so a pick that was legal stays legal.',
  'hg.rules.order.pay': 'Pay: costs go to the reserve, and purchases to the neighbours.',
  'hg.rules.order.place':
    'Place: buildings enter their cities, stage cards are tucked, sold cards go to the discard pile and their sellers take 3 coins.',
  'hg.rules.order.gain':
    "Gain: the coin effects of the cards and stages just placed happen. A card that counts your neighbours' cities counts what they placed this turn too.",
  'hg.rules.order.discard':
    'Build from the discard pile: a player who has just built a stage of {halikarnassos} with that ability takes a card from the pile.',
  'hg.rules.order.pass': 'Pass: the hands move on, after turns one to five.',
  'hg.rules.order.public':
    'Everyone sees what each player did: whether they constructed, built a stage or sold, which building they constructed, and the coins they paid each neighbour. A tucked or sold card stays hidden.',

  // 10. Military conflicts.
  'hg.rules.military.title': 'Military conflicts',
  'hg.rules.military.p1':
    'At the end of each Age every player compares their military strength, the shields in their city, with each of their two neighbours.',
  'hg.rules.military.win':
    'Stronger than that neighbour: take a victory token, worth 1 point in Age I, 3 in Age II and 5 in Age III.',
  'hg.rules.military.lose': 'Weaker: take a defeat token, worth minus 1 point.',
  'hg.rules.military.tie': 'Equal: no token.',
  'hg.rules.military.p2':
    'So you fight twice in each Age, and take at most two tokens. Shields count only at these moments and are never lost. Your military points can add up to less than zero.',

  // 11. Science.
  'hg.rules.science.title': 'Science',
  'hg.rules.science.p1':
    'Green cards carry one science symbol each: the compass, the gear or the tablet. They score at the end of the game, in two parts that add up.',
  'hg.rules.science.same':
    'Same symbols: each symbol scores its count multiplied by itself. One gear scores 1, two score 4, three score 9 and four score 16.',
  'hg.rules.science.sets': 'Sets: every set of three different symbols scores 7 more.',
  'hg.rules.science.choice':
    'The {scientists} and one stage on each side of {babylon} give a symbol of your choice. The game chooses it for you at the end, whichever scores best.',

  // 12. Commercial buildings.
  'hg.rules.commerce.title': 'Commercial buildings',
  'hg.rules.commerce.p1':
    'Yellow cards bring coins from the reserve, cheaper trade or extra production. Some in Age III also score points at the end.',
  'hg.rules.commerce.tavern': 'Take coins once, when you construct it.',
  'hg.rules.commerce.westTradingPost': 'Cheaper raw resources from your left neighbour.',
  'hg.rules.commerce.eastTradingPost': 'Cheaper raw resources from your right neighbour.',
  'hg.rules.commerce.marketplace': 'Cheaper manufactured resources from both neighbours.',
  'hg.rules.commerce.caravansery':
    'One raw resource of your choice each turn. Your neighbours cannot buy it.',
  'hg.rules.commerce.forum':
    'One manufactured resource of your choice each turn. Your neighbours cannot buy it.',
  'hg.rules.commerce.vineyard':
    "Take coins for each brown card in your city and in both neighbours' cities, those built this turn included.",
  'hg.rules.commerce.bazar': 'The same for grey cards, at a higher rate.',
  'hg.rules.commerce.lighthouse':
    'Take coins for each yellow card in your city, itself included, and score points for each at the end.',
  'hg.rules.commerce.haven':
    'Take coins for each brown card in your city, and score points for each at the end.',
  'hg.rules.commerce.chamberOfCommerce':
    'Take coins for each grey card in your city, and score points for each at the end.',
  'hg.rules.commerce.ludus':
    'Take coins for each red card in your city, and score points for each at the end.',
  'hg.rules.commerce.arena':
    'Take coins for each stage of your wonder built so far, and score points for each built stage at the end.',

  // 13. Guilds.
  'hg.rules.guilds.title': 'Guilds',
  'hg.rules.guilds.p1':
    'Guilds are the purple cards of Age III, two more of them than there are players. Each scores at the end of the game.',
  'hg.rules.guilds.workers': "Points for each brown card in your two neighbours' cities.",
  'hg.rules.guilds.craftsmens': "Points for each grey card in your two neighbours' cities.",
  'hg.rules.guilds.magistrates': "Points for each blue card in your two neighbours' cities.",
  'hg.rules.guilds.traders': "Points for each yellow card in your two neighbours' cities.",
  'hg.rules.guilds.spies': "Points for each red card in your two neighbours' cities.",
  'hg.rules.guilds.philosophers': "Points for each green card in your two neighbours' cities.",
  'hg.rules.guilds.builders':
    "Points for each wonder stage built in your city and in both neighbours' cities.",
  'hg.rules.guilds.shipowners':
    'Points for each brown, grey and purple card in your own city, itself included.',
  'hg.rules.guilds.decorators': 'Points if every stage on your side of your wonder is built.',
  'hg.rules.guilds.scientists': 'One science symbol of your choice.',
  'hg.rules.guilds.p2': "The guilds that count your neighbours' cards never count your own.",

  // 14. The seven wonders.
  'hg.rules.wonders.title': 'The seven wonders',
  'hg.rules.wonders.p1':
    "Every board has one starting resource, the same on both sides. A day side has three stages: points, then the board's own effect, then more points. A night side has two to four stages, mixing effects and points.",
  'hg.rules.wonders.p2':
    "A stage's effect starts when the stage is built and lasts for the rest of the game.",
  'hg.rules.wonders.heading': '{board}: {wonder}',
  'hg.rules.wonders.start': 'Starting resource: {resource}.',
  'hg.rules.wonders.alexandria.day':
    'Day side: points, then one raw resource of your choice each turn, then points.',
  'hg.rules.wonders.alexandria.night':
    'Night side: one raw resource of your choice each turn, then one manufactured resource of your choice each turn, then points.',
  'hg.rules.wonders.alexandria.note': 'Your neighbours can never buy what your wonder produces.',
  'hg.rules.wonders.babylon.day':
    'Day side: points, then one science symbol of your choice, then points.',
  'hg.rules.wonders.babylon.night':
    'Night side: two stages and no points. First the last card, then one science symbol of your choice.',
  'hg.rules.wonders.babylon.note':
    'The last card: on the sixth turn of each Age, from the Age you build it in, you keep your second card instead of discarding it, and play it after everyone else as a turn of your own, with any of the three actions. It already works if you build it on a sixth turn. Coins you gained on the sixth turn can be spent on it.',
  'hg.rules.wonders.ephesos.day':
    'Day side: points, then coins from the reserve once, then points.',
  'hg.rules.wonders.ephesos.night': 'Night side: three stages, each with coins once and points.',
  'hg.rules.wonders.gizah.day': 'Day side: three stages of points.',
  'hg.rules.wonders.gizah.night': 'Night side: four stages of points, the only side with four.',
  'hg.rules.wonders.halikarnassos.day':
    'Day side: points, then a build from the discard pile, then points.',
  'hg.rules.wonders.halikarnassos.night':
    'Night side: three stages, each a build from the discard pile. The first two also give points.',
  'hg.rules.wonders.halikarnassos.note':
    'Build from the discard pile: at the end of the turn in which you build that stage, you look through the whole pile, the cards discarded that same turn included, and construct one card from it for free. If a card can be built, you must take one; if none can, nothing happens. It works once, on that turn only.',
  'hg.rules.wonders.olympia.day':
    'Day side: points, then the first building of each colour for free, then points.',
  'hg.rules.wonders.olympia.night':
    'Night side: the first building of each Age for free, with points; then the last building of each Age for free, with points; then points.',
  'hg.rules.wonders.olympia.colour':
    'First of each colour: from then on, your first building of each colour you do not have yet is free, once per colour. A colour you already have gains nothing.',
  'hg.rules.wonders.olympia.first':
    'First of each Age: the building you construct on the first turn of each later Age is free.',
  'hg.rules.wonders.olympia.last':
    'Last of each Age: the building you construct on the sixth turn of each Age is free, already in the Age you build the stage if you build it before the sixth turn.',
  'hg.rules.wonders.olympia.note':
    "If you build a stage or sell on such a turn, that turn's free building is lost. These abilities make buildings free, never wonder stages.",
  'hg.rules.wonders.rhodos.day': 'Day side: points, then shields, then points.',
  'hg.rules.wonders.rhodos.night':
    'Night side: two stages, each with shields, coins once and points.',

  // 15. The end of the game and scoring.
  'hg.rules.scoring.title': 'The end and the score',
  'hg.rules.scoring.p1':
    'The game ends after the military conflicts of Age III. Every player adds up seven categories.',
  'hg.rules.scoring.wonder': 'Wonder: the points of your built stages.',
  'hg.rules.scoring.treasure':
    'Treasure: 1 point for every full 3 coins. Leftover coins score nothing.',
  'hg.rules.scoring.military': 'Military: your conflict tokens, which can add up below zero.',
  'hg.rules.scoring.blue': 'Blue cards: their points.',
  'hg.rules.scoring.yellow': 'Yellow cards: the end-of-game points of your Age III yellow cards.',
  'hg.rules.scoring.green': 'Green cards: your science score.',
  'hg.rules.scoring.purple': 'Guilds: their points.',
  'hg.rules.scoring.tie':
    'The highest total wins. On a tie, the tied player with more coins wins, and players who also have the same coins share the victory.',

  // 16. The number of players.
  'hg.rules.players.title': 'Three to seven players',
  'hg.rules.players.three':
    'With three players your two neighbours are the two other players, so you trade with both and fight both in every Age.',
  'hg.rules.players.more':
    'More players bring more copies of some cards into each deck, and more guilds: always two more than the number of players.',
  'hg.rules.players.seven':
    'With seven players every board is in play, and every Age card is used except one guild.',
  'hg.rules.players.nothingElse': 'Nothing else depends on the number of players.',

  // The concept diagrams (games/haftgane/rules/*Diagram.svelte).
  'hg.rules.diagram.example': 'For example:',
  'hg.rules.diagram.you': 'You',
  'hg.rules.diagram.left': 'Left neighbour',
  'hg.rules.diagram.right': 'Right neighbour',
  // The tooltip-size diagrams have room for one short word per seat.
  'hg.rules.diagram.leftShort': 'Left',
  'hg.rules.diagram.rightShort': 'Right',
  'hg.rules.diagram.age': 'Age {age}',
  'hg.rules.diagram.trade.caption':
    'You buy one stone from your left neighbour and one glass from your right neighbour, and pay each of them 2 coins. A trading discount makes a unit 1 coin.',
  'hg.rules.diagram.passing.caption':
    'In Ages I and III every hand moves to the left, so you pass to your left neighbour and receive from your right. In Age II it is the other way round.',
  'hg.rules.diagram.shields': { one: '{count} shield', other: '{count} shields' },
  'hg.rules.diagram.conflict.defeat': 'Defeat',
  'hg.rules.diagram.conflict.victory': 'Victory',
  'hg.rules.diagram.conflict.minus': '−{count}',
  'hg.rules.diagram.conflict.plus': '+{count}',
  'hg.rules.diagram.conflict.caption':
    'At the end of Age II you have 4 shields. Your left neighbour has more, so you take a defeat token, minus 1 point. Your right neighbour has fewer, so you take the Age II victory token, 3 points.',
  'hg.rules.diagram.science.symbols': 'The three science symbols',
  'hg.rules.diagram.science.city': 'A city with two compasses, two gears and one tablet:',
  'hg.rules.diagram.science.same': 'Same symbols: {a} + {b} + {c} = {sum}',
  'hg.rules.diagram.science.sets': 'Sets: {sets} × {each} = {points}',
  'hg.rules.diagram.science.total': { one: 'Total: {count} point', other: 'Total: {count} points' },
  'hg.rules.diagram.science.caption':
    'Two compasses score 4, two gears 4 and one tablet 1, which makes 9. They also form one complete set, worth 7, for 16 points in all.'
} as const satisfies Messages;
