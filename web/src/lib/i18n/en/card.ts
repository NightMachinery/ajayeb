// English messages for the card components (components/card/*,
// components/wonder/*, the card structure): card and wonder names, the
// effect sentences, affordability badges and the ARIA descriptions that
// view/describe.ts builds. See ../index.ts for the conventions.
//
// Names are keyed by catalog slug ('card.name.<slug>', 'wonder.name.<slug>'),
// so t(`card.name.${slug}`) type-checks only while every slug has a name.
// Effect sentences are templates fed by the catalog's declarative effects
// (components/card/effect-text.ts); lists of resources or colours are joined
// with Intl.ListFormat, so they follow the locale too.

import type { Messages } from '../types';

export const card = {
  'card.name.lumber-yard': 'Lumber Yard',
  'card.name.stone-pit': 'Stone Pit',
  'card.name.clay-pool': 'Clay Pool',
  'card.name.logging-camp': 'Logging Camp',
  'card.name.quarry': 'Quarry',
  'card.name.clay-pit': 'Clay Pit',
  'card.name.glassworks': 'Glassworks',
  'card.name.press': 'Press',
  'card.name.stable': 'Stable',
  'card.name.garrison': 'Garrison',
  'card.name.palisade': 'Palisade',
  'card.name.guard-tower': 'Guard Tower',
  'card.name.scriptorium': 'Scriptorium',
  'card.name.workshop': 'Workshop',
  'card.name.pharmacist': 'Pharmacist',
  'card.name.apothecary': 'Apothecary',
  'card.name.tavern': 'Tavern',
  'card.name.stone-reserve': 'Stone Reserve',
  'card.name.clay-reserve': 'Clay Reserve',
  'card.name.wood-reserve': 'Wood Reserve',
  'card.name.theater': 'Theater',
  'card.name.altar': 'Altar',
  'card.name.baths': 'Baths',
  'card.name.sawmill': 'Sawmill',
  'card.name.shelf-quarry': 'Shelf Quarry',
  'card.name.brickyard': 'Brickyard',
  'card.name.glass-blower': 'Glass-Blower',
  'card.name.drying-room': 'Drying Room',
  'card.name.horse-breeders': 'Horse Breeders',
  'card.name.barracks': 'Barracks',
  'card.name.walls': 'Walls',
  'card.name.archery-range': 'Archery Range',
  'card.name.parade-ground': 'Parade Ground',
  'card.name.school': 'School',
  'card.name.laboratory': 'Laboratory',
  'card.name.dispensary': 'Dispensary',
  'card.name.library': 'Library',
  'card.name.brewery': 'Brewery',
  'card.name.forum': 'Forum',
  'card.name.caravansery': 'Caravansery',
  'card.name.customs-house': 'Customs House',
  'card.name.temple': 'Temple',
  'card.name.statue': 'Statue',
  'card.name.courthouse': 'Courthouse',
  'card.name.aqueduct': 'Aqueduct',
  'card.name.rostrum': 'Rostrum',
  'card.name.circus': 'Circus',
  'card.name.arsenal': 'Arsenal',
  'card.name.siege-workshop': 'Siege Workshop',
  'card.name.fortifications': 'Fortifications',
  'card.name.pretorium': 'Pretorium',
  'card.name.academy': 'Academy',
  'card.name.university': 'University',
  'card.name.study': 'Study',
  'card.name.observatory': 'Observatory',
  'card.name.arena': 'Arena',
  'card.name.chamber-of-commerce': 'Chamber of Commerce',
  'card.name.port': 'Port',
  'card.name.lighthouse': 'Lighthouse',
  'card.name.armory': 'Armory',
  'card.name.palace': 'Palace',
  'card.name.gardens': 'Gardens',
  'card.name.pantheon': 'Pantheon',
  'card.name.town-hall': 'Town Hall',
  'card.name.senate': 'Senate',
  'card.name.obelisk': 'Obelisk',
  'card.name.merchants-guild': 'Merchants Guild',
  'card.name.shipowners-guild': 'Shipowners Guild',
  'card.name.builders-guild': 'Builders Guild',
  'card.name.magistrates-guild': 'Magistrates Guild',
  'card.name.scientists-guild': 'Scientists Guild',
  'card.name.moneylenders-guild': 'Moneylenders Guild',
  'card.name.tacticians-guild': 'Tacticians Guild',
  'wonder.name.pyramids': 'Pyramids',
  'wonder.name.colossus': 'Colossus',
  'wonder.name.great-lighthouse': 'Great Lighthouse',
  'wonder.name.temple-of-artemis': 'Temple of Artemis',
  'wonder.name.mausoleum': 'Mausoleum',
  'wonder.name.great-library': 'Great Library',
  'wonder.name.piraeus': 'Piraeus',
  'wonder.name.hanging-gardens': 'Hanging Gardens',
  'wonder.name.statue-of-zeus': 'Statue of Zeus',
  'wonder.name.sphinx': 'Sphinx',
  'wonder.name.appian-way': 'Appian Way',
  'wonder.name.circus-maximus': 'Circus Maximus',

  // Colours, as adjectives for ARIA and effect sentences, and as categories.
  'card.color.brown': 'brown',
  'card.color.grey': 'grey',
  'card.color.blue': 'blue',
  'card.color.green': 'green',
  'card.color.yellow': 'yellow',
  'card.color.red': 'red',
  'card.color.purple': 'purple',
  'card.category.brown': 'Raw materials',
  'card.category.grey': 'Manufactured goods',
  'card.category.blue': 'Civilian building',
  'card.category.green': 'Scientific building',
  'card.category.yellow': 'Commercial building',
  'card.category.red': 'Military building',
  'card.category.purple': 'Guild',
  'card.ageLabel': 'Age {age}',
  'card.ageNumeral.1': 'I',
  'card.ageNumeral.2': 'II',
  'card.ageNumeral.3': 'III',

  // Resources, science symbols and chain symbols.
  'card.resource.wood': 'wood',
  'card.resource.clay': 'clay',
  'card.resource.stone': 'stone',
  'card.resource.glass': 'glass',
  'card.resource.papyrus': 'papyrus',
  'card.science.globe': 'globe',
  'card.science.scales': 'scales',
  'card.science.sundial': 'sundial',
  'card.science.mortar': 'mortar and pestle',
  'card.science.set-square': 'set square',
  'card.science.quill': 'quill',
  'card.science.wheel': 'wheel',
  'card.chain.horseshoe': 'horseshoe',
  'card.chain.sword': 'sword',
  'card.chain.tower': 'tower',
  'card.chain.book': 'book',
  'card.chain.gear': 'gear',
  'card.chain.amphora': 'amphora',
  'card.chain.mask': 'mask',
  'card.chain.moon': 'moon',
  'card.chain.drop': 'drop',
  'card.chain.target': 'target',
  'card.chain.helmet': 'helmet',
  'card.chain.lyre': 'lyre',
  'card.chain.lamp': 'lamp',
  'card.chain.barrel': 'barrel',
  'card.chain.sun': 'sun',
  'card.chain.column': 'column',
  'card.chain.temple-facade': 'temple',

  // Effect sentences, one per effect kind of the catalog.
  'card.effect.produce': 'Produces {n} {resource} each turn',
  'card.effect.produceChoice': 'Produces one of {resources} each turn',
  'card.effect.tradeFix': 'You buy {resources} from the bank at 1 coin per unit',
  'card.effect.shields': {
    one: '{count} shield: moves the conflict pawn {count} space',
    other: '{count} shields: move the conflict pawn {count} spaces'
  },
  'card.effect.science': 'Science symbol: {symbol}',
  'card.effect.points': { one: '{count} victory point', other: '{count} victory points' },
  'card.effect.coins': { one: 'Take {count} coin', other: 'Take {count} coins' },
  'card.effect.coinsPerCard': {
    one: 'Take {count} coin for each {colors} card in your city',
    other: 'Take {count} coins for each {colors} card in your city'
  },
  'card.effect.coinsPerCardMax': {
    one: 'Take {count} coin for each {colors} card in the city that has the most',
    other: 'Take {count} coins for each {colors} card in the city that has the most'
  },
  'card.effect.coinsPerWonder': {
    one: 'Take {count} coin for each wonder you have built',
    other: 'Take {count} coins for each wonder you have built'
  },
  'card.effect.pointsPerCard': {
    one: 'At the end, {count} point for each {colors} card in your city',
    other: 'At the end, {count} points for each {colors} card in your city'
  },
  'card.effect.pointsPerCardMax': {
    one: 'At the end, {count} point for each {colors} card in the city that has the most',
    other: 'At the end, {count} points for each {colors} card in the city that has the most'
  },
  'card.effect.pointsPerWonder': {
    one: 'At the end, {count} point for each wonder you have built',
    other: 'At the end, {count} points for each wonder you have built'
  },
  'card.effect.pointsPerWonderMax': {
    one: 'At the end, {count} point for each wonder built in the city that has the most',
    other: 'At the end, {count} points for each wonder built in the city that has the most'
  },
  'card.effect.pointsPerCoins': {
    one: 'At the end, 1 point for each coin in the richest city',
    other: 'At the end, 1 point for each {count} coins in the richest city'
  },
  'card.effect.pointsPerToken': {
    one: 'At the end, {count} point for each progress token you own',
    other: 'At the end, {count} points for each progress token you own'
  },
  'card.effect.playAgain': 'Play again right away',
  'card.effect.destroy': "Discard one of your opponent's {colors} cards",
  'card.effect.buildFromDiscard': 'Build one card from the discard pile for free',
  'card.effect.drawTokens': {
    one: 'Draw {count} progress token from the box and keep it',
    other: 'Draw {count} progress tokens from the box and keep one'
  },
  'card.effect.opponentLosesCoins': {
    one: 'Your opponent loses {count} coin',
    other: 'Your opponent loses {count} coins'
  },
  'card.effect.discountBlue': {
    one: 'Blue buildings cost {count} resource less, of your choice',
    other: 'Blue buildings cost {count} resources less, of your choice'
  },
  'card.effect.discountWonder': {
    one: 'Wonders cost {count} resource less, of your choice',
    other: 'Wonders cost {count} resources less, of your choice'
  },
  'card.effect.tradeIncome': 'The coins your opponent pays for trade go to you',
  'card.effect.extraShield': {
    one: 'Every military building gives {count} extra shield',
    other: 'Every military building gives {count} extra shields'
  },
  'card.effect.wondersPlayAgain': 'Every wonder you build lets you play again',
  'card.effect.chainBonus': {
    one: 'Take {count} coin whenever you build through a chain',
    other: 'Take {count} coins whenever you build through a chain'
  },
  'card.effect.none': 'No effect',
  'card.chainOut': 'Chain symbol {symbol}: makes {card} free',
  'card.chainIn': 'Free with the {symbol} chain symbol of {card}',
  'card.chainInShort': 'Free with {symbol}',
  // Chain chip tooltips; the stack caption heads the linked card's preview.
  'card.tip.chainOut': 'Chain symbol {symbol}: building this makes {card} free.',
  'card.tip.chainOutBare': 'Chain symbol {symbol}.',
  'card.tip.chainOutStack': 'Free through this chain',
  'card.tip.chainIn': 'Free if your city has {card}, which shows the {symbol} chain symbol.',
  'card.tip.chainInStack': 'Gives the chain symbol',
  // Tooltips of a card's micro-elements (view/tips.ts). {resource} is a
  // resource name, {token} a token name, {back} 'Age I' and the like.
  'card.tip.card': '{card}: {category}, Age {age}.',
  'card.tip.quoteChain': 'free through a chain',
  'card.tip.quoteFree': 'free',
  'card.tip.quoteShort': {
    one: '{cost} ({count} coin short)',
    other: '{cost} ({count} coins short)'
  },
  'card.tip.costYou': 'You now: {cost}.',
  'card.tip.costNamed': '{name} now: {cost}.',
  'card.tip.cityCards': 'Inspect {color} cards.',
  'card.tip.printedCost': 'Printed cost: {cost}.',
  'card.tip.laterFree': 'Later builds {card} free.',
  'card.tip.freeWith': 'Free if you own {card}.',
  'card.tip.builtBy': 'Built by {name}.',
  'card.tip.backReveal': 'Turns face up when the cards covering it are taken.',
  'card.tip.costCoins': {
    one: 'Costs {count} coin, paid to the bank.',
    other: 'Costs {count} coins, paid to the bank.'
  },
  'card.tip.costUnits': 'Needs {n} {resource}.',
  'card.tip.costMore': 'Also needs {items}.',
  'card.tip.payOwn': 'Made in the city: {n}.',
  'card.tip.payFlex': 'From a yellow card or a wonder: {n}.',
  'card.tip.payDiscount': 'Waived by {token}: {n}.',
  'card.tip.payTrade': {
    one: 'Bought from the bank: {n} at {count} coin each.',
    other: 'Bought from the bank: {n} at {count} coins each.'
  },
  'card.tip.badgeChain': 'Free: the city shows the chain symbol of {card}.',
  'card.tip.badgeChainBare': 'Free through a chain symbol.',
  'card.tip.badgeFree': 'Free to build: nothing to pay.',
  'card.tip.badgeOwn': {
    one: 'The city makes every resource it needs. Pay {count} coin.',
    other: 'The city makes every resource it needs. Pay {count} coins.'
  },
  'card.tip.badgeTrade': {
    one: 'Buy the missing resources from the bank. Pay {count} coin in all.',
    other: 'Buy the missing resources from the bank. Pay {count} coins in all.'
  },
  'card.tip.tradeLine': {
    one: '{n} {resource} at {count} coin each',
    other: '{n} {resource} at {count} coins each'
  },
  // Heads the trade rule's worked example in the badge's bubble, so its
  // numbers never read as this game's.
  'card.tip.example': 'For example:',
  'card.tip.badgeShort': {
    one: 'Costs {coins} coins: {count} coin short.',
    other: 'Costs {coins} coins: {count} coins short.'
  },
  'card.tip.backAge': 'A face-down {back} card.',
  'card.tip.backGuild': 'A face-down guild card.',

  // Costs.
  'card.cost.free': 'No cost',
  'card.cost.label': 'Cost: {items}',
  'card.cost.coins': { one: '{count} coin', other: '{count} coins' },
  'card.cost.units': '{n} {resource}',
  'card.cost.more': '+{n}',

  // Affordability badges: visible words, then the spoken forms.
  'card.badge.free': 'Free',
  'card.afford.chain': 'free through {card}',
  'card.afford.chainNoCard': 'free through a chain',
  'card.afford.free': 'free to build',
  'card.afford.own': {
    one: 'build for {count} coin',
    other: 'build for {count} coins'
  },
  'card.afford.trade': {
    one: 'build for {count} coin with trade',
    other: 'build for {count} coins with trade'
  },
  'card.afford.short': {
    one: 'cannot build: costs {count} coin, {shortfall} more than you have',
    other: 'cannot build: costs {count} coins, {shortfall} more than you have'
  },
  'card.afford.waiting': 'not your turn',

  // ARIA descriptions (view/describe.ts) and the structure.
  'card.aria.card': '{name}, {color}. {effects}',
  'card.aria.available': 'available',
  'card.aria.covered': {
    one: 'covered by {count} card',
    other: 'covered by {count} cards'
  },
  'card.aria.faceDown': 'Face-down {back} card, revealed when uncovered',
  'card.aria.empty': 'Empty place',
  'card.aria.selected': 'selected',
  'card.structure.label': 'Age {age} cards',
  'card.back.age1': 'Age I',
  'card.back.age2': 'Age II',
  'card.back.age3': 'Age III',
  'card.back.guild': 'Guild',

  // Wonders.
  'wonder.tip.heading': '{name}: wonder.',
  'wonder.tip.currentCost': 'To build now: {cost}.',
  'wonder.aria': '{name}, wonder. {effects}',
  'wonder.built': 'Built',
  'wonder.builtWith': 'Built with {card}',
  'wonder.removed': 'Out of the game',
  'wonder.open': 'Not built yet',
  // The one status of a wonder list row (WonderTile size "row").
  'wonder.notBuilt': 'Not built',
  // A wonder the view's choices let its owner build this turn.
  'wonder.canBuild': 'Can build now',
  'wonder.takenBy': 'Taken by {name}',

  // The /dev/gallery cards section.
  'dev.cards.intro':
    'Every card, wonder and back from the catalog at each render tier, the five affordability badges from recorded views, and the three age structures from recorded views at a phone and a desktop size.',
  'dev.cards.tier': 'Tier {tier}',
  'dev.cards.badges': 'Affordability badges',
  'dev.cards.badgesDim': 'Not your turn (dimmed)',
  'dev.cards.art': 'Art frames',
  'dev.cards.backs': 'Card backs',
  'dev.cards.wonders': 'Wonders',
  'dev.cards.structures': 'Age structures',
  'dev.cards.structureBox': '{fixture} in a {width}×{height} box',
  'dev.cards.loading': 'Loading recorded views…',
  'dev.cards.show': 'Show'
} as const satisfies Messages;
