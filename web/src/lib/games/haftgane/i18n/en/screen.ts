// The table screen's messages (games/haftgane/table/): the top bar, the
// seat ring, your strip, the hand, the turn bar and every tooltip on them.
// Numbers and names arrive as params; every number shown is one the server
// sent or a count of what the frame holds.

import type { Messages } from '$lib/i18n';

export const screen = {
  'hg.screen.menu': 'Game menu',
  'hg.screen.menu.title': 'Table {code}',
  'hg.screen.menu.close': 'Close the menu',
  'hg.screen.menu.rules': 'Rules',
  'hg.screen.menu.toBot': 'Give seat {seat} to a bot',
  'hg.screen.menu.seatUser': 'Seat {name} in seat {seat}',
  'hg.screen.menu.undo': 'Undo',
  'hg.screen.menu.resign': 'Resign',
  'hg.screen.notice.dismiss': 'Dismiss',
  'hg.screen.age': 'Age {age}',
  'hg.screen.turn': '{turn}/{of}',
  'hg.screen.turn.tip': 'Turn {turn} of {of} in this Age',
  'hg.screen.sides': 'Sides',
  'hg.screen.sides.tip': 'Everyone chooses the side of their wonder board before Age I.',
  'hg.screen.over': 'Game over',

  'hg.screen.pass.left': 'Left',
  'hg.screen.pass.right': 'Right',
  'hg.screen.pass.tip.left': 'This Age, hands pass to the left: you give yours to {name}.',
  'hg.screen.pass.tip.right': 'This Age, hands pass to the right: you give yours to {name}.',
  'hg.screen.pass.from': 'Hands pass from {name} to you this Age.',
  'hg.screen.pass.to': 'You pass your hand to {name} this Age.',

  'hg.screen.you': 'You',
  'hg.screen.you.tip': 'Your city',
  'hg.screen.bot': '{name} (bot)',
  'hg.screen.botName': 'Bot',
  'hg.screen.seat': 'Seat {seat}',
  'hg.screen.where.self': 'you',
  'hg.screen.where.left': 'your left neighbour',
  'hg.screen.where.right': 'your right neighbour',
  'hg.screen.where.farLeft': {
    one: '{count} seat to your left',
    other: '{count} seats to your left'
  },
  'hg.screen.where.farRight': {
    one: '{count} seat to your right',
    other: '{count} seats to your right'
  },
  'hg.screen.where.seat': 'seat {seat}',
  'hg.screen.crest.tip': '{name}, {where}',
  'hg.screen.botBadge': 'A bot plays this seat.',

  'hg.screen.status.confirmed': '{name} has confirmed a choice.',
  'hg.screen.status.bot': '{name} is choosing.',
  'hg.screen.status.choosing': '{name} is still choosing.',
  'hg.screen.status.away': '{name} is away. The table waits for them or for an admin.',
  'hg.screen.status.awayWord': 'away',

  'hg.screen.points.tip': {
    one: 'Points so far: {count}',
    other: 'Points so far: {count}'
  },
  'hg.screen.coins.tip': { one: '{count} coin', other: '{count} coins' },
  'hg.screen.shields.tip': { one: '{count} shield', other: '{count} shields' },
  'hg.screen.conflict.tip': 'Conflict tokens: {sum} points so far',
  'hg.screen.stages': '{built}/{of}',
  'hg.screen.stages.tip': 'Wonder stages built: {built} of {of}',

  'hg.screen.outlook.win':
    'Military with {name}: you {you}, {name} {them}. If the Age ended now, you would take a +{token} token.',
  'hg.screen.outlook.lose':
    'Military with {name}: you {you}, {name} {them}. If the Age ended now, you would take a −1 token.',
  'hg.screen.outlook.tie':
    'Military with {name}: you {you}, {name} {them}. If the Age ended now, neither of you would take a token.',

  'hg.screen.trade.one': '{name} sells you {resource} for {price} each.',
  'hg.screen.trade.many': '{name} sells you up to {count} {resource} for {price} each.',
  'hg.screen.trade.choice': '{name} sells you one of {options}, {price} each.',
  'hg.screen.trade.or': '{a} or {b}',
  'hg.screen.trade.discountCard': 'Cheaper through your {card}.',
  'hg.screen.trade.discountStage': 'Cheaper through stage {stage} of your wonder.',
  'hg.screen.trade.none': '{name} makes nothing you can buy now.',
  'hg.screen.sellsYou': 'Sells you',
  'hg.screen.perUnit': 'price per unit',

  'hg.screen.colour.brown': { one: '{count} brown card', other: '{count} brown cards' },
  'hg.screen.colour.grey': { one: '{count} grey card', other: '{count} grey cards' },
  'hg.screen.colour.blue': { one: '{count} blue card', other: '{count} blue cards' },
  'hg.screen.colour.yellow': { one: '{count} yellow card', other: '{count} yellow cards' },
  'hg.screen.colour.red': { one: '{count} red card', other: '{count} red cards' },
  'hg.screen.colour.green': { one: '{count} green card', other: '{count} green cards' },
  'hg.screen.colour.purple': { one: '{count} guild', other: '{count} guilds' },
  'hg.screen.colourName.brown': 'Raw materials',
  'hg.screen.colourName.grey': 'Manufactured goods',
  'hg.screen.colourName.blue': 'Civic buildings',
  'hg.screen.colourName.yellow': 'Commercial buildings',
  'hg.screen.colourName.red': 'Military buildings',
  'hg.screen.colourName.green': 'Science buildings',
  'hg.screen.colourName.purple': 'Guilds',

  'hg.screen.making.tip': 'What your city makes each turn:',
  'hg.screen.making.one': '{count} × {resource}',
  'hg.screen.making.choice': 'One of {options}',
  'hg.screen.science.tip': 'Science: {compass} compass, {gear} gear, {tablet} tablet',
  'hg.screen.wonder.tip': '{wonder}, {side} side. Stages built: {built} of {of}.',
  'hg.screen.wonder.noSide': '{wonder}. No side chosen yet.',
  'hg.screen.side.day': 'day',
  'hg.screen.side.night': 'night',
  'hg.screen.stage.tip': 'Stage {stage} of your wonder',
  'hg.screen.stageNumeral.1': 'I',
  'hg.screen.stageNumeral.2': 'II',
  'hg.screen.stageNumeral.3': 'III',
  'hg.screen.stageNumeral.4': 'IV',
  'hg.screen.stage.built': 'Built.',
  'hg.screen.stage.open': 'You can build it this turn with a card from your hand.',
  'hg.screen.stage.later': 'Not this turn.',

  'hg.screen.hand': 'Your hand',
  'hg.screen.hand.size': { one: '{count} card', other: '{count} cards' },
  'hg.screen.hand.from': 'Your hand comes from {name}.',
  'hg.screen.hand.to': 'After this turn, your hand goes to {name}.',
  'hg.screen.hand.empty': 'No cards in your hand.',
  'hg.screen.watching': 'You are watching this table.',

  'hg.screen.afford.free.chain': 'Free: you own the card that chains to it.',
  'hg.screen.afford.free.first-colour': 'Free: the first card of its colour, through your wonder.',
  'hg.screen.afford.free.first-turn': 'Free: the first card of the Age, through your wonder.',
  'hg.screen.afford.free.last-turn': 'Free: the last card of the Age, through your wonder.',
  'hg.screen.afford.free.discard': 'Free: built from the discard pile.',
  'hg.screen.afford.freeWord': 'Free',
  'hg.screen.afford.trade': {
    one: 'The cheapest way costs {count} coin, buying from your neighbours.',
    other: 'The cheapest way costs {count} coins, buying from your neighbours.'
  },
  'hg.screen.afford.coins': {
    one: 'Costs {count} coin.',
    other: 'Costs {count} coins.'
  },
  'hg.screen.afford.built': 'You already built one. Sell it or use it for your wonder.',
  'hg.screen.afford.builtWord': 'Built',
  'hg.screen.afford.cannot-cover': 'You cannot get the resources it needs.',
  'hg.screen.afford.insufficient-coins': {
    one: 'You are {count} coin short.',
    other: 'You are {count} coins short.'
  },

  'hg.screen.cost.coins': { one: 'Costs {count} coin', other: 'Costs {count} coins' },
  'hg.screen.cost.res': 'Needs {count} × {resource}',
  'hg.screen.cost.free': 'No cost',
  'hg.screen.chain.out': 'Makes {card} free later.',
  'hg.screen.chain.in': 'Free if you own {card}.',

  'hg.screen.effect.produce': 'Makes {count} × {resource} each turn.',
  'hg.screen.effect.produce-choice': 'Makes one of {options} each turn.',
  'hg.screen.effect.trade-discount': 'Buy {options} from {who} for 1 coin each.',
  'hg.screen.effect.who.left': 'your left neighbour',
  'hg.screen.effect.who.right': 'your right neighbour',
  'hg.screen.effect.who.both': 'both neighbours',
  'hg.screen.effect.shields': { one: '{count} shield.', other: '{count} shields.' },
  'hg.screen.effect.science': 'A {symbol} science symbol.',
  'hg.screen.effect.science-choice': 'One science symbol of your choice at the end.',
  'hg.screen.effect.coins': { one: '{count} coin when built.', other: '{count} coins when built.' },
  'hg.screen.effect.coins-per-card': '{count} coins for each {colour} card in {where}, when built.',
  'hg.screen.effect.coins-per-stage': '{count} coins for each wonder stage in {where}, when built.',
  'hg.screen.effect.points': { one: '{count} point.', other: '{count} points.' },
  'hg.screen.effect.points-per-card': '{count} points for each {colour} card in {where}.',
  'hg.screen.effect.points-per-stage': '{count} points for each wonder stage in {where}.',
  'hg.screen.effect.points-if-wonder-complete': '{count} points if your wonder is complete.',
  'hg.screen.effect.play-last-card': 'You may play the last card of each Age too.',
  'hg.screen.effect.build-from-discard': 'Build one card from the discard pile for free.',
  'hg.screen.effect.free-build.first-colour': 'Build the first card of each colour for free.',
  'hg.screen.effect.free-build.first-turn': 'Build the first card of each Age for free.',
  'hg.screen.effect.free-build.last-turn': 'Build the last card of each Age for free.',
  'hg.screen.effect.where.self': 'your city',
  'hg.screen.effect.where.neighbours': 'your neighbours’ cities',
  'hg.screen.effect.where.all': 'your city and your neighbours’',

  'hg.screen.bar.choose': 'Choose a card',
  'hg.screen.bar.side': 'Choose your wonder side',
  'hg.screen.bar.discard': 'Choose a card from the discard pile',
  'hg.screen.bar.lastCard': 'Play your last card',
  'hg.screen.bar.confirm': 'Confirm your choice',
  'hg.screen.bar.confirmed': 'Confirmed. Waiting for the others.',
  'hg.screen.bar.waiting': 'Waiting for the others',
  'hg.screen.bar.waitingFor': 'Waiting for {name}',
  'hg.screen.bar.over': 'The game is over.',
  'hg.screen.bar.chosen': '{done}/{of}',
  'hg.screen.bar.chosen.tip': '{done} of {of} players have confirmed.',
  'hg.screen.confirm': 'Confirm',
  'hg.screen.change': 'Change',

  'hg.screen.pick.title': 'What to do with {card}',
  'hg.screen.pick.close': 'Close',
  'hg.screen.pick.build': 'Build',
  'hg.screen.pick.buildFor': {
    one: 'Build for {count} coin',
    other: 'Build for {count} coins'
  },
  'hg.screen.pick.buildFree': 'Build for free',
  'hg.screen.pick.split': 'Left {left}, right {right}',
  'hg.screen.pick.stage': 'Wonder stage {stage}',
  'hg.screen.pick.stageFor': {
    one: 'Wonder stage {stage} for {count} coin',
    other: 'Wonder stage {stage} for {count} coins'
  },
  'hg.screen.pick.sell': {
    one: 'Sell for {count} coin',
    other: 'Sell for {count} coins'
  },
  'hg.screen.pick.side': 'Play the {side} side',
  'hg.screen.pick.tag': 'Your pick',
  'hg.screen.pick.picked': 'Your pick: {what}',
  'hg.screen.pick.named': '{card}: {action}'
} as const satisfies Messages;
