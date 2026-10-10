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
  'game.cost.left': 'You have {have}; {after} left after',
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

  // Toasts after a move in a local room.
  'game.toast.undo': 'Undo',

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
  'game.gallery.scene.gameOver': 'Game over'
} as const satisfies Messages;
