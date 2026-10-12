// English messages for the rules reference (components/rules/*, the /rules
// page and the in-game rules panel). See ../index.ts for the conventions.
//
// The rules are written in our own words from docs/rules/*.md (never the
// rulebook's text; docs/legal.md). One key per paragraph, list item or
// definition; components/rules/sections.ts arranges them into sections.
// Card, wonder and token names are not written here: a {placeholder} names
// them, and sections.ts fills it from the existing name keys
// ('card.name.*', 'wonder.name.*', 'token.*'), so a renamed card is renamed
// in the rules too. Numbers in the text are fixed rule numbers, never game
// state.

import type { Messages } from '../types';

export const rules = {
  // The page, the panel and the entry points.
  'rules.title': 'Rules',
  'rules.pageTitle': 'How to play',
  'rules.intro':
    'The rules of the base game for two players, in short. The game applies every rule for you, so you can come back here whenever a move surprises you.',
  'rules.toc': 'Contents',
  'rules.close': 'Close the rules',
  'rules.backHome': 'Home',
  'rules.homeLink': 'Rules',
  'rules.menu': 'Rules',
  'rules.openPage': 'Open the rules as a full page',

  // 1. The goal and the three ways to win.
  'rules.goal.title': 'The goal',
  'rules.goal.p1':
    'Two players build rival civilizations over three ages. On each turn you take one card from a shared layout and use it to grow your city, fill your treasury or raise a wonder.',
  'rules.goal.winTitle': 'Three ways to win',
  'rules.goal.win.military':
    "Military supremacy: push the conflict pawn into your opponent's capital and you win at once.",
  'rules.goal.win.science':
    'Scientific supremacy: own six different science symbols and you win at once.',
  'rules.goal.win.civilian':
    'Civilian victory: if neither supremacy happens, the game ends after the last card of Age III, and the player with more victory points wins.',
  'rules.goal.cardsTitle': 'The seven kinds of card',
  'rules.goal.card.brown': 'Produce raw goods: wood, clay and stone.',
  'rules.goal.card.grey': 'Produce manufactured goods: glass and papyrus.',
  'rules.goal.card.blue': 'Are worth victory points.',
  'rules.goal.card.green': 'Carry science symbols, and many are worth points too.',
  'rules.goal.card.yellow':
    'Bring coins, cheaper trading or a choice of resources. Those of Age III are worth points.',
  'rules.goal.card.red':
    "Carry shields, which push the conflict pawn toward your opponent's capital.",
  'rules.goal.card.purple':
    'The guilds. They appear only in Age III and reward what the two cities hold.',

  // 2. Setup and the wonder draft.
  'rules.setup.title': 'Setup and the wonder draft',
  'rules.setup.p1':
    'Each player starts with 7 coins. The conflict pawn starts on the centre of the military track, with the four looting tokens on their spaces. Five of the ten progress tokens are laid out face up; the other five stay in the box, unseen.',
  'rules.setup.p2':
    'Who plays first is drawn at random, unless the game was set up with a fixed first player. That player starts the draft and Age I.',
  'rules.setup.draftTitle': 'The wonder draft',
  'rules.setup.draftIntro':
    'Before Age I the players draft four wonders each, in two rounds of four.',
  'rules.setup.draft1':
    'Round 1: four wonders are shown. The first player takes one, the second player takes two, and the first player gets the last one.',
  'rules.setup.draft2':
    'Round 2: four new wonders are shown. Now the second player takes one, the first player takes two, and the second player gets the last one.',
  'rules.setup.agesTitle': 'The ages',
  'rules.setup.ages1':
    'Each age deals 20 cards into an overlapping layout called the structure, with some rows face up and some face down. Age I is a pyramid with its wide row at the bottom, Age II is the same pyramid upside down, and Age III is two pyramids joined in the middle.',
  'rules.setup.ages2':
    'Three cards of every age are set aside unseen, so nobody can be sure which cards will turn up. Three of the seven guilds are shuffled into Age III; a guild dealt face down shows the guild back.',

  // 3. A turn.
  'rules.turn.title': 'A turn',
  'rules.turn.p1':
    'On your turn you must take one accessible card: a card that no other card in the structure overlaps. You cannot pass. Then you do exactly one of three things with it.',
  'rules.turn.build': 'Build it: pay its cost and put it in your city. Its effects start at once.',
  'rules.turn.discard':
    'Discard it: put it on the discard pile and take 2 coins, plus 1 more for each yellow card in your city. You can always do this.',
  'rules.turn.wonder':
    "Build a wonder with it: pay the cost of one of your unbuilt wonders, not the card's, and tuck the card under the wonder. Any accessible card will do, even one you could not afford. The tucked card itself has no effect.",
  'rules.turn.p2':
    'When the turn is over, every face-down card that nothing overlaps any more is turned face up. Then your opponent plays, unless you earned an extra turn.',
  'rules.turn.p3':
    'A city never holds two copies of the same building. The discard pile is open: both players can look through it at any time.',

  // 4. Costs, resources and trading.
  'rules.costs.title': 'Costs, resources and trading',
  'rules.costs.p1':
    'A cost can be nothing, some coins, some resources, or coins and resources together. Printed coins go to the bank. Each unit of a resource you produce covers one unit of every cost, turn after turn: production is never used up.',
  'rules.costs.tradeTitle': 'Trading',
  'rules.costs.trade':
    'Any resource you lack can be bought from the bank. One unit costs 2 coins, plus 1 coin for each unit of that resource your opponent produces with brown and grey cards.',
  'rules.costs.trade.only':
    "Only your opponent's brown and grey cards raise your price. Cards and wonders that offer a choice of resources never do.",
  'rules.costs.trade.bank':
    'The coins go to the bank, not to your opponent, unless your opponent owns the {economy} token.',
  'rules.costs.trade.free':
    'Your opponent cannot refuse, and your purchase uses up nothing of theirs.',
  'rules.costs.fixedTitle': 'Yellow cards that fix prices',
  'rules.costs.fixed':
    'Four yellow cards set your price for some resources to 1 coin per unit, whatever your opponent produces. They help only their owner.',
  'rules.costs.fixed.stone': 'Stone for 1 coin.',
  'rules.costs.fixed.clay': 'Clay for 1 coin.',
  'rules.costs.fixed.wood': 'Wood for 1 coin.',
  'rules.costs.fixed.customs': 'Glass and papyrus for 1 coin each.',
  'rules.costs.choiceTitle': 'A choice of resources',
  'rules.costs.choice':
    '{forum} and {piraeus} each produce one glass or papyrus per turn; {caravansery} and the {lighthouse} each produce one wood, clay or stone. Each covers whichever unit saves you the most.',
  'rules.costs.chainTitle': 'Chains',
  'rules.costs.chain':
    'Many cards show a chain symbol. A card of a later age that shows the same symbol beside its cost is free for you, no coins and no resources, when your city holds the earlier card. Only your own city counts, and a card tucked under a wonder does not.',
  'rules.costs.payTitle': 'The game pays for you',
  'rules.costs.pay':
    'You never choose how to pay. The game always finds the cheapest way and shows its breakdown on the card before you commit.',

  // 5. The military track.
  'rules.military.title': 'The military track',
  'rules.military.p1':
    "Each shield you gain moves the conflict pawn one space toward your opponent's capital. Both players push the same pawn, so its place shows who is ahead and by how much. Shields come from red cards, from three wonders and from the {strategy} token.",
  'rules.military.zonesTitle': 'Zones',
  'rules.military.zone2': 'Spaces 1 and 2 from the centre: 2 points at the end.',
  'rules.military.zone5': 'Spaces 3 to 5: 5 points.',
  'rules.military.zone10': 'Spaces 6 to 8: 10 points.',
  'rules.military.zoneCapital': 'Space 9 is the capital: reaching it wins the game at once.',
  'rules.military.points':
    'At the end, only the player who is ahead scores the points of the zone where the pawn stands. With the pawn on the centre, nobody scores.',
  'rules.military.lootTitle': 'Looting',
  'rules.military.loot':
    "When the pawn first enters space 3 on a player's side, that player loses 2 coins; at space 6, 5 coins. The coins go to the bank, a player never goes below zero, and each looting token works only once. One big push can take both tokens of a side.",

  // 6. Science.
  'rules.science.title': 'Science',
  'rules.science.p1':
    'Each green card shows one of six science symbols, and each symbol appears on exactly two cards. A seventh symbol, the scales, comes only from the {law} token.',
  'rules.science.pairTitle': 'Pairs',
  'rules.science.pair':
    'When your city holds both cards of one symbol, that is a pair, and you take one progress token from the board at once. Each pair can happen only once in a game. If no tokens are left on the board, the pair gives nothing.',
  'rules.science.winTitle': 'Scientific supremacy',
  'rules.science.win':
    'Own six different symbols, out of the seven, and you win at once. The scales of {law} count as one of them.',

  // 7. Wonders.
  'rules.wonders.title': 'Wonders',
  'rules.wonders.p1':
    "Each player has the four wonders of the draft. A wonder can be built in any age and in any order, by tucking an accessible card under it and paying the wonder's cost. A built wonder stays built. Its effects happen when it is built, and its points count at the end.",
  'rules.wonders.seventhTitle': 'The seventh wonder',
  'rules.wonders.seventh':
    'Only seven wonders can be built in a game. When the seventh is built, by either player, the last unbuilt wonder leaves the game and is worth nothing.',
  'rules.wonders.againTitle': 'Play again',
  'rules.wonders.again':
    'Five wonders give you another turn at once: the {artemis}, {piraeus}, the {gardens}, the {sphinx} and the {appian}. Built during an extra turn, such a wonder gives yet another one. An extra turn earned on the last card of an age is lost.',
  'rules.wonders.listTitle': 'The twelve wonders',
  'rules.wonders.pyramids': '9 points.',
  'rules.wonders.colossus': '2 shields. 3 points.',
  'rules.wonders.great-lighthouse': 'Produces one wood, clay or stone each turn. 4 points.',
  'rules.wonders.temple-of-artemis': 'Take 12 coins. Play again. No points.',
  'rules.wonders.mausoleum':
    'Build one card of your choice from the discard pile, for free. 2 points.',
  'rules.wonders.great-library':
    'Draw three progress tokens from the box, keep one and put the others back. Only you see them. 4 points.',
  'rules.wonders.piraeus': 'Produces one glass or papyrus each turn. Play again. 2 points.',
  'rules.wonders.hanging-gardens': 'Take 6 coins. Play again. 3 points.',
  'rules.wonders.statue-of-zeus':
    "1 shield, then destroy one brown card in your opponent's city. 3 points.",
  'rules.wonders.sphinx': 'Play again. 6 points.',
  'rules.wonders.appian-way':
    'Take 3 coins, and your opponent loses 3 coins. Play again. 3 points.',
  'rules.wonders.circus-maximus':
    "1 shield, then destroy one grey card in your opponent's city. 3 points.",
  'rules.wonders.choices':
    'A destroyed card goes to the discard pile. A choice that has something to choose from must be made; one with nothing to choose from is skipped, and the wonder is still built.',

  // 8. Progress tokens.
  'rules.tokens.title': 'Progress tokens',
  'rules.tokens.p1':
    'You take a progress token when you complete a science pair, or by building the {library}. A token stays yours for the rest of the game. Its lasting effects apply to what you do afterwards, never to what you did before.',
  'rules.tokens.agriculture': 'Take 6 coins. 4 points.',
  'rules.tokens.architecture': 'Each wonder you build afterwards costs 2 fewer resources.',
  'rules.tokens.economy':
    'The coins your opponent pays for trading come to you instead of the bank.',
  'rules.tokens.law': 'Counts as the scales, a seventh science symbol.',
  'rules.tokens.masonry': 'Each blue card you build afterwards costs 2 fewer resources.',
  'rules.tokens.mathematics': '3 points for each progress token you own, this one included.',
  'rules.tokens.philosophy': '7 points.',
  'rules.tokens.strategy': 'Each red card you build afterwards gives 1 extra shield.',
  'rules.tokens.theology':
    'Each wonder you build afterwards lets you play again. A wonder never gives more than one extra turn.',
  'rules.tokens.urbanism':
    'Take 6 coins. Afterwards, each card you build for free through a chain also pays you 4 coins.',
  'rules.tokens.discounts':
    '{architecture} and {masonry} waive the two units that would cost you the most. Printed coins are never reduced, and a discount you cannot use is lost.',

  // 9. Guilds.
  'rules.guilds.title': 'Guilds',
  'rules.guilds.p1':
    "Three of the seven guilds join Age III. A guild looks at both cities: most count the cards of one colour in whichever city has more of them, even your opponent's. A guild pays its coins once, when you build it, from the cities at that moment, and scores its points at the end, from the cities at the end.",
  'rules.guilds.merchants': 'Yellow cards: 1 coin and 1 point for each.',
  'rules.guilds.shipowners':
    'Brown and grey cards together, counted in one city: 1 coin and 1 point for each.',
  'rules.guilds.builders':
    'Wonders: 2 points for each wonder built by the player who built the most. No coins.',
  'rules.guilds.magistrates': 'Blue cards: 1 coin and 1 point for each.',
  'rules.guilds.scientists': 'Green cards: 1 coin and 1 point for each.',
  'rules.guilds.moneylenders':
    'Coins: 1 point for every 3 coins the richer player holds at the end. No coins.',
  'rules.guilds.tacticians': 'Red cards: 1 coin and 1 point for each.',

  // 10. The end of an age.
  'rules.ageEnd.title': 'The end of an age',
  'rules.ageEnd.p1':
    'An age ends when its last card has been taken and that turn is over. Everything carries over to the next age: coins, cities, wonders, progress tokens and the pawn.',
  'rules.ageEnd.p2':
    "The next age's cards are laid out first. Then the weaker player, the one on whose side the pawn stands, chooses who starts the next age. With the pawn on the centre, the player who took the last card chooses.",
  'rules.ageEnd.p3': 'Starting matters: in Ages II and III only two cards can be taken at first.',

  // 11. The end of the game and scoring.
  'rules.scoring.title': 'The end of the game and scoring',
  'rules.scoring.p1':
    'If nobody wins by supremacy, the game ends once the last card of Age III has been taken, and each player adds up points in eight categories:',
  'rules.scoring.blue': 'Blue cards.',
  'rules.scoring.green': 'Green cards.',
  'rules.scoring.yellow': 'Yellow cards (only those of Age III have points).',
  'rules.scoring.guilds': 'Guilds.',
  'rules.scoring.wonders': 'The wonders you built.',
  'rules.scoring.tokens': 'Progress tokens.',
  'rules.scoring.coins': 'Coins: 1 point for every 3 coins.',
  'rules.scoring.military': 'Military: 2, 5 or 10 points for the player who is ahead on the track.',
  'rules.scoring.tie':
    'The higher total wins. On a tie, the player with more points from blue cards wins, and if that is tied too, both players share the victory.',
  'rules.scoring.preview':
    'During the game, the score shown for each player is a preview: the points they would have if the game ended now.',

  // 12. Glossary.
  'rules.glossary.title': 'Glossary',
  'rules.glossary.accessible.term': 'Accessible',
  'rules.glossary.accessible':
    'A card in the structure that no other card overlaps. Only accessible cards can be taken.',
  'rules.glossary.covered.term': 'Covered',
  'rules.glossary.covered': 'A card that other cards still overlap. It cannot be taken yet.',
  'rules.glossary.structure.term': 'Structure',
  'rules.glossary.structure': 'The 20 cards of the current age, laid out in overlapping rows.',
  'rules.glossary.city.term': 'City',
  'rules.glossary.city':
    'The buildings a player has built. Their wonders and progress tokens sit beside it.',
  'rules.glossary.discard.term': 'Discard pile',
  'rules.glossary.discard':
    'The cards discarded for coins and the cards destroyed by wonders. Both players can see it.',
  'rules.glossary.tucked.term': 'Tucked card',
  'rules.glossary.tucked': 'The card placed under a wonder to build it. It has no effect.',
  'rules.glossary.price.term': 'Trade price',
  'rules.glossary.price': 'What one unit of a resource costs you from the bank.',
  'rules.glossary.chain.term': 'Chain',
  'rules.glossary.chain': 'A symbol on an earlier card that makes a matching later card free.',
  'rules.glossary.pair.term': 'Pair',
  'rules.glossary.pair':
    'Two cards in one city with the same science symbol. A pair takes a progress token.',
  'rules.glossary.supremacy.term': 'Supremacy',
  'rules.glossary.supremacy': 'A military or scientific win, which ends the game at once.',
  'rules.glossary.weaker.term': 'Weaker player',
  'rules.glossary.weaker':
    'The player on whose side of the track the pawn stands. With the pawn on the centre, neither player is weaker.',
  'rules.glossary.extra.term': 'Extra turn',
  'rules.glossary.extra':
    'A second turn in a row for the same player, from some wonders and from the {theology} token.',
  'rules.glossary.looting.term': 'Looting token',
  'rules.glossary.looting':
    'A token on the military track that costs a player coins when the pawn first enters its space.',
  'rules.glossary.box.term': 'The box',
  'rules.glossary.box':
    'Where the cards, wonders and tokens left out of the game go. Only the {library} ever draws from it.',

  // Diagrams (components/rules/*Diagram.svelte).
  'rules.diagram.points': { one: '{count} point', other: '{count} points' },
  'rules.diagram.win': 'Win',
  'rules.diagram.military':
    'One side of the track, from the centre to a capital; the other side is the same. The looting tokens on spaces 3 and 6 take 2 and 5 coins from the player on this side.',
  'rules.diagram.trade.opponent': 'Your opponent produces 3 clay.',
  // The sum stays on one line (no-break spaces).
  'rules.diagram.trade.price': 'Each clay you buy costs 2\u00a0+\u00a03\u00a0=\u00a05 coins.',
  'rules.diagram.trade.label':
    'Example: your opponent produces 3 clay, so each clay you buy costs 5 coins.',
  'rules.diagram.science.six': 'The six symbols of the green cards',
  'rules.diagram.science.scales': 'The scales, from the {law} token only'
} as const satisfies Messages;
