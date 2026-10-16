// English messages for the in-room game screen (components/game/*,
// components/sheets/WonderSheet.svelte, routes/play). The draft, the
// choice panels and the game-over screen keep their own keys in
// choices.ts. See ../index.ts for the conventions.

import type { Messages } from '../types';

export const game = {
  'game.title': 'Game {code}',
  'game.loading': 'Waiting for the game…',
  'game.board': 'Game board',
  'game.openCity': "Show {name}'s city",

  // The card sheet.
  'game.card.meta': '{category} · Age {age}',
  'game.card.covered': {
    one: 'Covered by {count} card. It can be taken once nothing covers it.',
    other: 'Covered by {count} cards. It can be taken once nothing covers it.'
  },
  'game.card.faceDown': 'Face down. It is revealed when nothing covers it any more.',
  'game.card.notYourTurn': 'Not your turn: you can look, but not play.',
  'game.card.locked': 'Wait until your move has been answered.',
  'game.card.nothing': 'Nothing can be done with this card now.',
  'game.card.opponentPays': {
    '=0': '{name} would build it for free.',
    one: '{name} would pay {count} coin for it.',
    other: '{name} would pay {count} coins for it.'
  },
  'game.card.opponentShort': {
    one: '{name} could not afford it ({count} coin).',
    other: '{name} could not afford it ({count} coins).'
  },
  // A covered card: what each player would pay once it is uncovered, from
  // the server's costs for both seats (the opponent's line reuses the two
  // messages above).
  'game.card.coveredCostTitle': 'If it were uncovered now',
  'game.card.youPay': {
    '=0': 'You would build it for free.',
    one: 'You would pay {count} coin for it.',
    other: 'You would pay {count} coins for it.'
  },
  'game.card.youShort': {
    one: 'You could not afford it ({count} coin).',
    other: 'You could not afford it ({count} coins).'
  },

  // Action buttons. Each carries data-act with the server's key.
  'game.action.build': {
    '=0': 'Build for free',
    one: 'Build for {count} coin',
    other: 'Build for {count} coins'
  },
  'game.action.discard': {
    one: 'Discard for {count} coin',
    other: 'Discard for {count} coins'
  },
  'game.action.wonder': 'Build a wonder',
  'game.action.buildWonder': {
    '=0': 'Build {wonder} for free',
    one: 'Build {wonder} for {count} coin',
    other: 'Build {wonder} for {count} coins'
  },

  // Why an action is not available (the server's Reason codes).
  'game.reason.gameOver': 'The game is over.',
  'game.reason.wrongPhase': 'The game is waiting for another kind of move.',
  'game.reason.malformed': 'That move is not possible.',
  'game.reason.notInDraft': 'That wonder is not on offer.',
  'game.reason.notAccessible': 'That card cannot be taken now.',
  'game.reason.wonderUnavailable': 'That wonder cannot be built.',
  'game.reason.unaffordable': {
    one: 'You are {count} coin short.',
    other: 'You are {count} coins short.'
  },
  'game.reason.unaffordableNoCount': 'You cannot afford it.',
  'game.reason.notTarget': 'That card cannot be chosen.',
  'game.reason.notInDiscard': 'That card is not in the discard pile.',
  'game.reason.tokenUnavailable': 'That token is not on offer.',

  // The cost breakdown, read from the server's CostView.
  'game.cost.title': 'Cost',
  'game.cost.own': '{n} {resource}: you make it',
  'game.cost.producer': '1 {resource}: from {source}',
  'game.cost.discount': '{n} {resource}: waived by {token}',
  'game.cost.trade': '{n} {resource}: bought at {price} each',
  'game.cost.tradeFixed': '{n} {resource}: bought at the fixed price of {price}',
  'game.cost.printed': 'Coin cost of the card',
  'game.cost.chain': 'Free through the chain symbol of {card}',
  'game.cost.chainNoCard': 'Free through a chain symbol',
  'game.cost.urbanism': {
    one: 'You receive {count} coin (Urbanism)',
    other: 'You receive {count} coins (Urbanism)'
  },
  'game.cost.toOpponent': {
    one: '{count} coin of it goes to your opponent (Economy)',
    other: '{count} coins of it go to your opponent (Economy)'
  },
  'game.cost.nothing': 'Nothing to pay',
  'game.cost.total': 'Total',
  'game.cost.short': {
    one: 'You have {have}: {count} coin short',
    other: 'You have {have}: {count} coins short'
  },

  // The wonder step of the card sheet (card first).
  'game.wonders.title': 'Use {card} to build a wonder',
  'game.wonders.back': 'Back to the card',
  'game.wonders.tuckNote': 'The card goes face down under the wonder, and its own effect is lost.',
  'game.wonders.sevenNote': {
    one: '{count} of 7 wonders is built. When the seventh is built, the last one still unbuilt leaves the game.',
    other:
      '{count} of 7 wonders are built. When the seventh is built, the last one still unbuilt leaves the game.'
  },
  'game.wonders.none': 'None of your wonders can be built with this card now.',

  // The wonder sheet (wonder first) and the pick mode it starts.
  'game.wonder.build': 'Build this wonder',
  'game.wonder.pickHint': 'Next, pick a card from the structure to place under it.',
  'game.wonder.cannot': 'It cannot be built now.',
  'game.pick.prompt': 'Pick a card for {wonder}',
  'game.pick.cancel': 'Cancel',
  'game.pick.confirm': 'Use {card} to build {wonder}?',
  'game.pick.effectLost': 'Its effect is lost: the card goes face down under the wonder.',
  'game.pick.notThisCard': '{card} cannot go under {wonder} now.',

  // The bot's persona, by level (the room's seat names a bot by it).
  'game.persona.easy': 'Apprentice',
  'game.persona.normal': 'Scholar',
  'game.persona.hard': 'Master',

  // The game menu in the top bar.
  'game.menu.open': 'Game menu',
  'game.menu.home': 'Home',
  'game.menu.homeHint': 'You can continue this game from Home.',
  'game.menu.settings': 'Settings',
  'game.menu.log': 'Log',
  'game.menu.replay': 'Download replay',
  'game.menu.forfeit': 'Forfeit the game',
  // The menu's undo item, and why it is greyed out when it is.
  'game.menu.undo': 'Undo last move',
  'game.menu.undoOnline': 'Undo is unavailable in online games.',
  'game.menu.undoDisabled': 'Undo is off for this game.',
  'game.menu.undoSpectator': 'Spectators cannot undo moves.',
  'game.menu.undoNothing': 'No move of yours to take back yet.',
  'game.menu.undoOver': 'The game is over.',
  'game.menu.undoWaiting': 'Waiting for the server…',
  'game.menu.undoOffline': 'Waiting for connection.',

  // The forfeit's confirm step (ForfeitConfirm.svelte).
  'game.forfeit.title': 'Forfeit the game?',
  'game.forfeit.titleName': 'Forfeit the game for {name}?',
  'game.forfeit.consequence': '{name} wins at once.',
  'game.forfeit.final': 'A forfeit ends the game for good: it cannot be undone.',
  'game.forfeit.confirm': 'Forfeit',
  'game.forfeit.cancel': 'Keep playing',

  // The settings sheet inside a game.
  'game.settings.title': 'Settings',

  // The /dev/gallery game screen section.
  'game.gallery.title': 'Game screen',
  'game.gallery.intro':
    'The game screen on recorded game states, each in a frame of a phone or desktop size. Each frame is the /dev/fixture page of that state.',
  'game.gallery.frame': '{scene} at {width}×{height}',
  'game.gallery.open': 'Open full size',
  'game.gallery.scene.draft': 'Wonder draft',
  'game.gallery.scene.ageOne': 'Age I start',
  'game.gallery.scene.ageTwo': 'Mid Age II',
  'game.gallery.scene.ageTwoLate': 'Late Age II',
  'game.gallery.scene.ageThree': 'Age III',
  'game.gallery.scene.pending': 'A pending choice',
  'game.gallery.scene.gameOver': 'Game over',
  'game.gallery.scene.lobby': 'Online lobby',
  'game.gallery.scene.away': 'Opponent away',
  'game.gallery.scene.watching': 'Spectator',
  'game.gallery.summary': 'Opponent move summary',
  'game.gallery.summaryIntro':
    'The popup of what the other side just did, on recorded moves. Open one, then tap anywhere to close it.',
  'game.gallery.summaryOpen': 'Show: {scene}',
  'game.gallery.summary.bot': 'A bot reply with a science pair and a token',
  'game.gallery.summary.extra': 'An extra turn with trade, the pawn and looting',
  'game.gallery.summary.special':
    'A wonder that destroys, a wonder leaving the game and an age change',

  // The opponent move summary (MoveSummary.svelte): what the other side
  // did since the viewer last moved, one line per move. The player is named
  // in the title, so the lines leave them out.
  'game.summary.title': "{name}'s move",
  'game.summary.open': "Show the opponent's last move",
  'game.summary.close': 'Tap anywhere to close',
  'game.summary.build': 'Built {card}',
  'game.summary.discard': {
    one: 'Discarded {card} for {count} coin',
    other: 'Discarded {card} for {count} coins'
  },
  'game.summary.wonder': 'Built the wonder {wonder}, using {card}',
  'game.summary.wonderPlain': 'Built the wonder {wonder}',
  'game.summary.draft': 'Took {wonder} in the draft',
  'game.summary.draftAuto': '{wonder}, the last of the round, went to {name}',
  'game.summary.token': 'Took the {token} token',
  'game.summary.tokenLibrary': 'Took the {token} token from the Great Library',
  'game.summary.destroy': "Destroyed {name}'s {card}",
  'game.summary.rebuild': 'Built {card} from the discard pile',
  'game.summary.starter': 'Chose {name} to start the next age',
  'game.summary.resign': 'Forfeited the game',
  'game.summary.ageStart': 'Age {age} begins',
  'game.summary.extraTurn': 'Plays again',
  'game.summary.extraTurnLost': 'The extra turn is lost at the end of the age',
  'game.summary.wonderRemoved': "{name}'s {wonder} leaves the game",
  'game.summary.wonderRemovedTip': 'Seven wonders are built, so the eighth leaves the game.',
  'game.summary.wins': '{name} wins: {kind}',
  'game.summary.paid': {
    one: 'Paid {count} coin',
    other: 'Paid {count} coins'
  },
  'game.summary.trade': {
    one: '{count} coin of it went to {name} for trade',
    other: '{count} coins of it went to {name} for trade'
  },
  'game.summary.gains': {
    one: '{name} gains {count} coin',
    other: '{name} gains {count} coins'
  },
  'game.summary.loses': {
    one: '{name} loses {count} coin',
    other: '{name} loses {count} coins'
  },
  'game.summary.pawn': {
    one: 'The conflict pawn moves {count} space toward {name}',
    other: 'The conflict pawn moves {count} spaces toward {name}'
  },
  'game.summary.pawnShort': '{count} toward {name}',
  'game.summary.loot': {
    one: '{name} loses {count} coin to looting',
    other: '{name} loses {count} coins to looting'
  },
  'game.summary.pair': 'A pair of {symbol} symbols: a progress token to take',
  'game.summary.pairShort': 'Pair'
} as const satisfies Messages;
