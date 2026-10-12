// English messages for a game's hint settings (docs/dev/decisions.md D23):
// the family names and descriptions, the settings editor and its local
// hide (components/hints/), the new-game section, the lobby summary, the
// game menu item and the log line of a change (lib/hints/log.ts). See
// ../index.ts for the conventions.

import type { Messages } from '../types';

export const hints = {
  // The families, each with a one-line description (lib/hints/families.ts).
  'hints.family.prices': 'Prices',
  'hints.family.prices.hint': 'What cards and wonders cost, for both players.',
  'hints.family.economy': 'Economy',
  'hints.family.economy.hint': 'How a card changes production and trading prices.',
  'hints.family.science': 'Science',
  'hints.family.science.hint': 'Science symbols, pairs, and which are still in play.',
  'hints.family.military': 'Military',
  'hints.family.military.hint': 'Where the pawn would move, and what it would loot.',
  'hints.family.threats': 'Threats',
  'hints.family.threats.hint': 'When a move wins the game, or hands a win over.',
  'hints.family.tempo': 'Tempo',
  'hints.family.tempo.hint': 'Which cards a move uncovers, and who starts the next age.',
  'hints.family.wonders': 'Wonders',
  'hints.family.wonders.hint': 'What each wonder would do now, and the race for the seventh.',
  'hints.family.tokens': 'Progress tokens',
  'hints.family.tokens.hint': 'What each progress token is worth now.',
  'hints.family.scoring': 'Scoring',
  'hints.family.scoring.hint': 'Where each player’s points come from.',
  'hints.family.counting': 'Card counting',
  'hints.family.counting.hint': 'Which cards are still unseen, from the public card lists.',
  'hints.family.recap': 'Recap',
  'hints.family.recap.hint': 'What changed in the last move.',
  'hints.family.advisor': 'Advisor',
  'hints.family.advisor.hint':
    'After your move, whether a better one was there, and why. Only you see it.',

  // The editor (HintSettingsEditor.svelte).
  'hints.title': 'Hints',
  'hints.sameForBoth': 'Both players play with the same hints.',
  'hints.readOnly': 'Only the player who created the game can change them.',
  'hints.on': 'On',
  'hints.off': 'Off',
  'hints.saving': 'Saving…',

  // The local hide (HintLocalHide.svelte).
  'hints.local.title': 'On my screen',
  'hints.local.intro': 'Hide hints on this screen only. It never shows a hint the game turned off.',
  'hints.local.boardMarks': 'Hide hint marks on the board',
  'hints.local.boardMarksHint': 'The sheets still show the hints.',
  'hints.local.hideFamily': 'Hide {family} on my screen',

  // The read-only summary (HintsSummary.svelte), and the collapsed new-game
  // section. {families} is a list of family names.
  'hints.summary.all': 'All hints on',
  'hints.summary.none': 'No hints',
  'hints.summary.allBut': 'All hints on except {families}',
  'hints.summary.only': 'Only {families}',
  'hints.lobby.title': 'Hints in this game',

  // The game menu item and its sheet (HintsMenuItem.svelte, GameHintsSheet.svelte).
  'hints.menu': 'Hints',
  'hints.sheet.title': 'Hints in this game',
  // The game's undo setting, read-only under the hints (hot-seat and bot games).
  'hints.sheet.undoTitle': 'Undo',
  'hints.sheet.undoOn': 'Undo is allowed in this game.',
  'hints.sheet.undoOff': 'Undo is off in this game.',
  'hints.sheet.undoFixed': 'Set when the game was created; it cannot be changed.',

  // A change in the game log (lib/hints/log.ts). {families} is a list of
  // family names; {changes} joins the "on" and "off" parts.
  'hints.log.on': 'turned on {families}',
  'hints.log.off': 'turned off {families}',
  'hints.log.separator': '; ',
  'hints.log.changedBy': '{name} changed the hints: {changes}.',
  'hints.log.changed': 'The hints changed: {changes}.',

  // Per-card hints (lib/view/hints.ts, M10.HS2): the pip on a card, its
  // tooltip lines, and the card sheet's hint sections and notes. {symbol}
  // and {resource} are lower-case names (card.symbol.*, card.resource.*).
  'hints.pip.title': 'Hints on {card}',
  'hints.pip.win.you': 'You win if you build this.',
  'hints.pip.win.named': '{name} wins by building this.',
  'hints.pip.win.lawYou':
    'You win with it: it completes a pair, and Law would be your sixth symbol.',
  'hints.pip.win.lawNamed':
    '{name} wins with it: it completes a pair, and Law would be their sixth symbol.',
  'hints.pip.pair.you': 'Completes your {symbol} pair: you pick a progress token.',
  'hints.pip.pair.named': 'Completes {name}’s {symbol} pair: they would pick a progress token.',
  'hints.pip.loot.you': {
    one: '{name} loses {count} coin if you build this.',
    other: '{name} loses {count} coins if you build this.'
  },
  'hints.pip.loot.youLose': {
    one: 'You lose {count} coin if {name} builds this.',
    other: 'You lose {count} coins if {name} builds this.'
  },
  'hints.pip.loot.named': {
    one: '{other} loses {count} coin if {name} builds this.',
    other: '{other} loses {count} coins if {name} builds this.'
  },
  'hints.pip.aria': '{label}. {hints}',

  'hints.victory.military': 'military',
  'hints.victory.science': 'science',
  'hints.victory.civilian': 'points',
  'hints.victory.forfeit': 'forfeit',

  'hints.section.you': 'For you',
  'hints.section.named': 'For {name}',
  'hints.line.win': 'Building it wins the game ({kind}).',
  'hints.line.winShort': 'It would win the game ({kind}), but cannot be paid for now.',
  'hints.line.winLaw': 'Completes a pair, and Law then wins the game.',
  'hints.line.pair': 'Completes the {symbol} pair: pick a progress token.',
  'hints.line.pairNoToken': 'Completes the {symbol} pair, but no tokens are left.',
  'hints.line.newSymbol': {
    one: 'A new symbol, {symbol}: {count} different of 6.',
    other: 'A new symbol, {symbol}: {count} different of 6.'
  },
  'hints.line.pawn': {
    one: 'Pawn {count} space toward {name}’s capital.',
    other: 'Pawn {count} spaces toward {name}’s capital.'
  },
  'hints.line.pawnYou': {
    one: 'Pawn {count} space toward your capital.',
    other: 'Pawn {count} spaces toward your capital.'
  },
  'hints.line.loot': {
    one: 'Looting: {name} loses {count} coin.',
    other: 'Looting: {name} loses {count} coins.'
  },
  'hints.line.lootYou': {
    one: 'Looting: you lose {count} coin.',
    other: 'Looting: you lose {count} coins.'
  },
  'hints.line.price.you': 'Your {resource}: {from} to {to} coins per unit.',
  'hints.line.price.named': '{name}’s {resource}: {from} to {to} coins per unit.',
  'hints.line.saves': {
    one: 'Saves {count} coin on the cards and wonders in view, now.',
    other: 'Saves {count} coins on the cards and wonders in view, now.'
  },
  'hints.line.savesNothing': 'Saves nothing in view right now.',
  'hints.line.lastCertain': 'No other {resource} producer can still come.',
  'hints.line.lastVisible': 'The last {resource} producer in view; unseen cards may hold another.',
  'hints.line.worth': 'Worth now: {coins} coins, {vp} VP.',
  'hints.line.worthCoins': {
    one: 'Worth {count} coin to build today.',
    other: 'Worth {count} coins to build today.'
  },
  'hints.line.onlySource': '{name}’s only {resource}.',
  'hints.line.onlySourceYou': 'Your only {resource}.',

  'hints.note.deniesChain': 'Also keeps it from {name} (free for them by chain).',
  'hints.note.deniesChainForNow':
    'Keeps it from {name} for now (free for them by chain): their Mausoleum could still rebuild it.',
  'hints.note.deniesCheap': {
    one: 'Also keeps it from {name}, who would pay only {count} coin.',
    other: 'Also keeps it from {name}, who would pay only {count} coins.'
  },
  'hints.note.deniesCheapForNow': {
    one: 'Keeps it from {name} for now ({count} coin for them): their Mausoleum could still rebuild it.',
    other:
      'Keeps it from {name} for now ({count} coins for them): their Mausoleum could still rebuild it.'
  },
  'hints.note.mausoleumOpp':
    '{name}’s Mausoleum could rebuild it for free. Tucked under a wonder, it is out of reach.',
  'hints.note.mausoleumYou': 'Your Mausoleum could rebuild it later.',
  'hints.note.mausoleumNamed': '{name}’s Mausoleum could rebuild it later.',
  'hints.note.denyBuild': 'Building it takes it from {name}.',
  'hints.note.denyTuck': 'Tucked under a wonder, it is out of the game for {name}.',
  'hints.note.shortByDiscard': {
    one: 'A discard (+{count} coin) would cover it at today’s prices.',
    other: 'A discard (+{count} coins) would cover it at today’s prices.'
  },

  'hints.discard.poolYou': 'Your Mausoleum could rebuild',
  'hints.discard.poolNamed': '{name}’s Mausoleum could rebuild',
  'hints.discard.empty': 'Nothing in the pile would do anything special.',
  'hints.cardWith': '{card}: {hint}',

  // The decider's hints (lib/view/outcome.ts, M10.HS3): the outcome line
  // under each move, the confirm step's warning, the card sheet's "After
  // this pick" panel and the starter choice. {delta} is a signed number
  // ("+3"), {card} and {list} are names, {age} a number.
  'hints.outcome.label': 'What this move does',
  'hints.outcome.separator': '; ',
  'hints.outcome.win': 'Wins the game ({kind}).',
  'hints.outcome.winLaw': 'Wins: it completes a pair, and Law is the sixth symbol.',
  'hints.outcome.winVia': 'Wins with the follow-up: {card}.',
  'hints.outcome.winTwoStep': 'Wins in the extra turn, with {card}.',
  'hints.outcome.endYou': 'Ends the game: you win on points, {a} to {b}.',
  'hints.outcome.endNamed': 'Ends the game: {name} wins on points, {a} to {b}.',
  'hints.outcome.endBlueYou': 'Ends the game: {a} points each, and you win on blue points.',
  'hints.outcome.endBlueNamed': 'Ends the game: {a} points each, and {name} wins on blue points.',
  'hints.outcome.endShared': 'Ends the game in a shared victory, {a} points each.',
  'hints.outcome.vp': '{delta} VP',
  'hints.outcome.vpNamed': '{name}: {delta} VP',
  'hints.outcome.pawn': {
    one: 'Pawn {count} space toward {name}’s capital',
    other: 'Pawn {count} spaces toward {name}’s capital'
  },
  'hints.outcome.pawnYou': {
    one: 'Pawn {count} space toward your capital',
    other: 'Pawn {count} spaces toward your capital'
  },
  'hints.outcome.loot': {
    one: '{name} loses {count} coin to looting',
    other: '{name} loses {count} coins to looting'
  },
  'hints.outcome.lootYou': {
    one: 'You lose {count} coin to looting',
    other: 'You lose {count} coins to looting'
  },
  'hints.outcome.coins': {
    one: '{count} coin after ({delta})',
    other: '{count} coins after ({delta})'
  },
  'hints.outcome.coinsNamed': {
    one: '{name}: {count} coin after ({delta})',
    other: '{name}: {count} coins after ({delta})'
  },
  'hints.outcome.distinct': {
    one: '{count} different science symbol',
    other: '{count} different science symbols'
  },
  'hints.outcome.playAgain': 'Play again',
  'hints.outcome.playAgainTheology': 'Play again (Theology)',
  'hints.outcome.extraLost': 'The age ends: no extra turn',
  'hints.outcome.starterYou': 'You choose who starts Age {age}',
  'hints.outcome.starterNamed': '{name} chooses who starts Age {age}',
  'hints.outcome.opens.chooseStarter': 'Then choose who starts',
  'hints.outcome.opens.destroyCard': 'Then pick a card to destroy',
  'hints.outcome.opens.buildFromDiscard': 'Then rebuild a discarded card',
  'hints.outcome.opens.pickBoardToken': 'Then pick a progress token',
  'hints.outcome.opens.pickLibraryToken': 'Then pick from three tokens',
  'hints.outcome.shortWin': {
    one: 'Short {count} coin of a win ({kind}).',
    other: 'Short {count} coins of a win ({kind}).'
  },

  'hints.blunder.gives': 'This hands {name} a win: {list}.',
  'hints.blunder.missedYou': 'You could win this turn instead: {list}.',
  'hints.blunder.missedNamed': '{name} could win this turn instead: {list}.',

  'hints.after.title': 'After this pick',
  'hints.after.freesNothing': 'Uncovers no card.',
  'hints.after.frees': {
    one: 'Uncovers {list}.',
    other: 'Uncovers {count} cards: {list}.'
  },
  'hints.after.faceDown': {
    one: '{count} face-down card',
    other: '{count} face-down cards'
  },
  'hints.after.reveal': {
    one: 'The card turned over is {name}’s to take first.',
    other: 'The cards turned over are {name}’s to take first.'
  },
  'hints.after.revealWonder': 'With the extra turn, you see the turned card first.',
  'hints.after.giftWin': '{card} would then win the game for {name}.',
  'hints.after.giftPair': '{card} would then give {name} a science pair.',
  'hints.after.giftChain': '{card} would then be free for {name} by chain.',
  'hints.after.giftPays': {
    one: '{card} would then cost {name} {count} coin.',
    other: '{card} would then cost {name} {count} coins.'
  },
  'hints.after.gives': '{name} could then win with {list}.',
  'hints.after.blocks': 'Takes away {name}’s win with {list}.',
  'hints.after.threatensYou': 'Next turn you could win with {list}, unless {name} stops it.',
  'hints.after.threatensNamed':
    'Next turn {mover} could win with {list}, unless {name} stops it.',
  'hints.after.risk': {
    one: 'A {num} in {den} chance the card turned over lets {name} win.',
    other: 'A {num} in {den} chance a card turned over lets {name} win.'
  },
  'hints.after.byMove': '{move}: {line}',

  'hints.starter.threatYou': 'If you start, you can win at once with {list}.',
  'hints.starter.threatNamed': 'If {name} starts, they can win at once with {list}.',
  'hints.starter.opening': 'The age opens with {list}.',

  // The badge modifiers (card/AffordBadge.svelte).
  'hints.badge.bonus': '+{count}',
  'hints.badge.urbanismTip': {
    one: 'Free by chain, and Urbanism pays you {count} coin.',
    other: 'Free by chain, and Urbanism pays you {count} coins.'
  },
  'hints.badge.flowTip': {
    one: '{count} coin of this trade goes to {name} (Economy).',
    other: '{count} of these coins go to {name} (Economy).'
  }
} as const satisfies Messages;
