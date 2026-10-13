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
  'hints.line.worth': {
    one: 'Worth now: {count} coin, {vp} VP.',
    other: 'Worth now: {count} coins, {vp} VP.'
  },
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
  'hints.cardWith': '{card}: {hint}',

  // The decider's hints (lib/view/outcome.ts, M10.HS3): the outcome line
  // under each move, the confirm step's warning, the card sheet's "After
  // this pick" panel and the starter choice. {delta} is a signed number
  // ("+3"), {card} and {list} are names, {age} a number.
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
  'hints.after.revealWonderYou': 'Built into a wonder that plays again, it lets you take it first.',
  'hints.after.revealWonderNamed':
    'Built into a wonder that plays again, it lets {name} take it first.',
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
  'hints.after.threatensNamed': 'Next turn {mover} could win with {list}, unless {name} stops it.',
  'hints.after.risk': {
    one: 'A {num} in {den} chance the card turned over lets {name} win.',
    other: 'A {num} in {den} chance a card turned over lets {name} win.'
  },
  'hints.after.byMove': '{move}: {line}',
  'hints.after.moveBuild': 'Build',
  'hints.after.moveDiscard': 'Discard',

  'hints.starter.threatYou': 'If you start, you can win at once with {list}.',
  'hints.starter.threatNamed': 'If {name} starts, they can win at once with {list}.',
  'hints.starter.opening': 'The age opens with {list}.',
  'hints.ghost.pawn': 'Where this move puts the pawn.',
  'hints.ghost.loot': {
    one: 'This move takes this looting token: {count} coin lost.',
    other: 'This move takes this looting token: {count} coins lost.'
  },
  'hints.ghost.coins': {
    one: '{count} coin after this move',
    other: '{count} coins after this move'
  },
  'hints.ghost.score': {
    one: '{count} point after this move',
    other: '{count} points after this move'
  },

  // Board-level hints (lib/view/board-hints.ts, M10.HS4): where a card is,
  // the science tracker, the military track, the wonders, the progress
  // tokens, the age chip and the age interstitial. {place} is one of the
  // hints.place.* phrases; {symbol} a symbol's name; {age} a number.
  'hints.place.faceUp': 'face up on the board',
  'hints.place.tucked': 'tucked under a wonder, out of the game',
  'hints.place.cityYou': 'in your city',
  'hints.place.cityNamed': 'in {name}’s city',
  'hints.place.discard': 'in the discard pile',
  'hints.place.unseenThisAge': 'not seen yet this age',
  'hints.place.unseenThisAgeOdds': 'not seen yet this age ({num} in {den} that it is dealt)',
  'hints.place.laterAge': 'in the Age {age} deck',
  'hints.place.laterAgeOdds': 'in the Age {age} deck ({num} in {den} that it is dealt)',
  'hints.place.removed': 'set aside, out of the game',

  'hints.science.symbol': '{symbol}.',
  'hints.science.card': '{card}: {place}.',
  'hints.science.paired': 'A pair.',
  'hints.science.live': 'One card; the other may still come.',
  'hints.science.owned': 'One card; the other can no longer come.',
  'hints.science.open': {
    one: 'None yet; {count} card may still come.',
    other: 'None yet; {count} cards may still come.'
  },
  'hints.science.dead': 'None, and no card of it can come.',
  'hints.science.scalesOwned': 'Owned, from the Law token.',
  'hints.science.scalesOpen': 'Only from the Law token, which may still come.',
  'hints.science.scalesDead': 'Only from the Law token, which can no longer come.',
  'hints.science.needYou': {
    one: 'You need {count} more different symbol; still possible: {list}.',
    other: 'You need {count} more different symbols; still possible: {list}.'
  },
  'hints.science.needNamed': {
    one: '{name} needs {count} more different symbol; still possible: {list}.',
    other: '{name} needs {count} more different symbols; still possible: {list}.'
  },
  'hints.science.unreachableYou': 'You can no longer win by science.',
  'hints.science.unreachableNamed': '{name} can no longer win by science.',
  'hints.science.lawYou': 'Law would be your sixth symbol: a pair you complete now wins.',
  'hints.science.lawNamed': 'Law would be {name}’s sixth symbol: a pair they complete now wins.',

  'hints.capital.win': 'Reaching this space wins the game at once.',
  'hints.capital.titleYou': 'Your capital.',
  'hints.capital.titleNamed': '{name}’s capital.',
  'hints.capital.yours': 'your capital',
  'hints.capital.named': '{name}’s capital',
  'hints.capital.needYou': {
    one: 'You need {count} more shield to take {capital}.',
    other: 'You need {count} more shields to take {capital}.'
  },
  'hints.capital.needNamed': {
    one: '{name} needs {count} more shield to take {capital}.',
    other: '{name} needs {count} more shields to take {capital}.'
  },
  'hints.capital.safeYou': 'You can no longer gather that many shields.',
  'hints.capital.safeNamed': '{name} can no longer gather that many shields.',
  'hints.zone.you': {
    one: 'With the pawn here at the end, you score {count} point. Only the player ahead scores.',
    other: 'With the pawn here at the end, you score {count} points. Only the player ahead scores.'
  },
  'hints.zone.named': {
    one: 'With the pawn here at the end, {name} scores {count} point. Only the player ahead scores.',
    other:
      'With the pawn here at the end, {name} scores {count} points. Only the player ahead scores.'
  },

  'hints.loot.taken': 'Already taken: it never triggers again.',
  'hints.loot.triggerYou': {
    one: 'When the pawn enters this space, you lose {count} coin.',
    other: 'When the pawn enters this space, you lose {count} coins.'
  },
  'hints.loot.triggerNamed': {
    one: 'When the pawn enters this space, {name} loses {count} coin.',
    other: 'When the pawn enters this space, {name} loses {count} coins.'
  },
  'hints.loot.clampYou': {
    one: 'You have only {count} coin now.',
    other: 'You have only {count} coins now.'
  },
  'hints.loot.clampNamed': {
    one: '{name} has only {count} coin now.',
    other: '{name} has only {count} coins now.'
  },

  'hints.glow.militaryWin': '{name} could take this capital now with {list}.',
  'hints.glow.militaryWinYou': 'You could take this capital now with {list}.',
  'hints.glow.calm': 'Close, but it can no longer happen.',

  // The city sheet's threats list (view/threats.ts threatLines): the wins a
  // seat would have with one move on its turn.
  'hints.threats.title': 'Threats',
  'hints.threats.militaryYou': 'On your turn you could take the capital with {list}.',
  'hints.threats.military': 'On their turn {name} could take the capital with {list}.',
  'hints.threats.scienceYou': 'On your turn you could win by science with {list}.',
  'hints.threats.science': 'On their turn {name} could win by science with {list}.',
  'hints.threats.civilianYou': 'On your turn you could end the game ahead on points with {list}.',
  'hints.threats.civilian': 'On their turn {name} could end the game ahead on points with {list}.',

  'hints.wonder.library':
    'Draws {count} of the {total} tokens in the box: {list}. Only its owner sees them.',
  'hints.wonder.lastSlot': 'Only one more wonder can be built.',
  'hints.wonder.removes': 'Building this one sends {name}’s {wonder} back to the box.',
  'hints.wonder.removesYour': 'Building this one sends your {wonder} back to the box.',
  'hints.wonder.affordYou': 'You can afford it now.',
  'hints.wonder.affordNamed': '{name} can afford it now.',
  'hints.wonder.readyYou': 'You could build this on your turn, at today’s prices.',
  'hints.wonder.readyNamed': '{name} could build this on their turn, at today’s prices.',
  'hints.wonder.readyCountYou': {
    one: 'You could build {count} wonder at today’s prices.',
    other: 'You could build {count} wonders at today’s prices.'
  },
  'hints.wonder.readyCountNamed': {
    one: '{name} could build {count} wonder at today’s prices.',
    other: '{name} could build {count} wonders at today’s prices.'
  },
  'hints.wonder.slotsLeft': {
    one: '{count} wonder slot left, {unbuilt} wonders unbuilt.',
    other: '{count} wonder slots left, {unbuilt} wonders unbuilt.'
  },

  'hints.token.forYou': 'For you: {value}.',
  'hints.token.forNamed': 'For {name}: {value}.',
  'hints.token.wins': 'wins the game',
  'hints.token.coins': { one: '+{count} coin', other: '+{count} coins' },
  'hints.token.vp': '{delta} VP',
  'hints.token.saves': {
    one: 'saves {count} coin on the cards and wonders in view',
    other: 'saves {count} coins on the cards and wonders in view'
  },
  'hints.token.savesNothing': 'saves nothing in view now',
  'hints.token.economy': {
    one: '{name}’s trade on the cards they can take: {count} coin',
    other: '{name}’s trade on the cards they can take: {count} coins'
  },
  'hints.token.chains': {
    one: '{count} chain still open',
    other: '{count} chains still open'
  },
  'hints.token.theology': {
    one: '{count} unbuilt wonder gains play again',
    other: '{count} unbuilt wonders gain play again'
  },
  'hints.token.strategy': 'later red cards get one more shield',
  'hints.token.nothing': 'nothing now',

  'hints.age.cardsLeft': {
    one: '{count} card left this age.',
    other: '{count} cards left this age.'
  },
  'hints.age.lastYou': 'If nobody plays again, you take the last card.',
  'hints.age.lastNamed': 'If nobody plays again, {name} takes the last card.',
  'hints.age.lastLost': 'An extra turn taken with the last card is lost.',
  'hints.age.chooserYou': 'You are the weaker player and will choose who starts Age {age}.',
  'hints.age.chooserNamed': '{name} is the weaker player and will choose who starts Age {age}.',
  'hints.age.lastStarts': 'The player who takes the last card will start Age {age}.',
  'hints.age.lastChooses': 'The player who takes the last card will choose who starts Age {age}.',
  'hints.age.deadPairYou': '{card} was set aside: your {own} will never make a pair.',
  'hints.age.deadPairNamed': '{card} was set aside: {name}’s {own} will never make a pair.',

  'hints.draft.order': '{first} picks one, {second} picks two, and the last one goes to {first}.',

  // The badge modifiers (card/AffordBadge.svelte).
  'hints.badge.bonus': '+{count}',
  'hints.badge.urbanismTip': {
    one: 'Free by chain, and Urbanism pays you {count} coin.',
    other: 'Free by chain, and Urbanism pays you {count} coins.'
  },
  'hints.badge.flowTip': {
    one: '{count} coin of this trade goes to {name} (Economy).',
    other: '{count} of these coins go to {name} (Economy).'
  },

  // The move review (lib/view/review.ts, components/game/MoveReview.svelte,
  // M10.HS5): inside the summary of the opponent's next move, what the
  // player did and whether a clearly better move was there. {move} is one
  // of the hints.review.move.* phrases.
  'hints.review.title': 'Your move',
  'hints.review.tip': 'The advisor looks at your own move after you play. Only you see it.',
  'hints.review.played.build': 'You built {card}.',
  'hints.review.played.discard': 'You discarded {card}.',
  'hints.review.played.wonder': 'You built the wonder {wonder}.',
  'hints.review.played.other': 'Your last move.',
  'hints.review.move.build': 'Building {card}',
  'hints.review.move.discard': 'Discarding {card}',
  'hints.review.move.wonder': 'Building {wonder}',
  'hints.review.move.other': 'Another move',
  'hints.review.none': 'No clearly better move was found.',
  'hints.review.wins': '{move} would have won the game.',
  'hints.review.stopsMilitary': '{move} would have kept {name} from taking your capital.',
  'hints.review.stopsWinWith': '{move} would have kept {name} from winning with {card}.',
  'hints.review.stopsWin': '{move} would have kept {name} from winning next turn.',
  'hints.review.pair': '{move} would have completed your {symbol} pair.',
  'hints.review.deniesPair': '{move} would have kept {name} from a {symbol} pair.',
  'hints.review.loot': {
    one: '{move} would have looted {count} coin from {name}.',
    other: '{move} would have looted {count} coins from {name}.'
  },
  'hints.review.points': {
    one: '{move} looked about {count} point better.',
    other: '{move} looked about {count} points better.'
  },

  // What changed since the player's last move (lib/view/recap.ts): the
  // trade prices the other side's moves changed, shown in the move summary.
  'hints.recap.title': 'Your prices',
  'hints.recap.tip': 'Trade prices that changed since your last move.',
  'hints.recap.price': 'Your {resource}: was {from}, now {to} coins.',
  'hints.recap.priceBuilt': 'Your {resource}: was {from}, now {to} coins. {name} built {card}.',
  // The board's marks of the same changes (M10.F3): the trade-price chip's
  // caret, a card's badge, the emptied slot and a card just turned face up.
  'hints.recap.chip': 'Since your last move: was {from}, now {to}.',
  'hints.recap.chipBuilt': 'Since your last move: was {from}, now {to}: {name} built {card}.',
  'hints.recap.cardPrice': 'Was {from}, now {to}.',
  'hints.recap.cardPriceBuilt': 'Was {from}, now {to}: {name} built {card}.',
  'hints.recap.emptied': 'Taken since your last move.',
  'hints.recap.revealed': 'Turned face up since your last move.',

  // First-time tips (lib/ui/tips.svelte.ts, components/hints/FirstTip.svelte):
  // each explains a hint mark once, the first time it appears on this device.
  'hints.tip.label': 'Tip',
  'hints.tip.pair':
    'Each science symbol is on two cards. Own both and you pick a progress token. This mark shows a card that completes your pair.',
  'hints.tip.denyPair':
    'This card would complete {name}’s science pair, and they would pick a progress token. Taking it keeps it from them.',
  'hints.tip.win': 'A crown marks a card that wins the game if you build it now.',
  'hints.tip.winOpp': 'An outlined crown marks a card that would win the game for {name}.',
  'hints.tip.loot':
    'This card’s shields would push the pawn onto a looting token, and the other player would lose coins.',
  'hints.tip.gotIt': 'Got it',
  'hints.tip.more': 'More in Rules',

  // The Settings screen's hint section (screens/Settings.svelte).
  'hints.settings.title': 'Hints',
  'hints.settings.tips': 'First-time tips',
  'hints.settings.tipsHint': 'Explain each hint mark the first time it appears on this device.',
  'hints.settings.tipsReset': 'Show the tips again',
  'hints.settings.tipsResetDone': 'The tips will show again.',

  // The hint-settings badge (hud/TurnBar.svelte). {summary} is one of the
  // hints.summary.* lines.
  'hints.status.label': 'Hints in this game',
  'hints.status.tip': 'Hints in this game: {summary}. {setBy}',
  'hints.status.setByYou': 'You can change them in the game menu.',
  'hints.status.setByCreator': 'The player who created the game sets them.',

  // The rules reference's "Hint glyphs" section (rules/sections.ts,
  // components/hints/HintGlyphLegend.svelte). Under rules.* so the rules
  // sections can name it; it lives here with the other hint words.
  'rules.hints.title': 'Hint glyphs',
  'rules.hints.intro':
    'Hint marks state facts about a card as the board stands now, never advice. Each card shows at most one mark at its top corner.',
  'rules.hints.win': 'Crown: building this card wins the game.',
  'rules.hints.pair':
    'Progress-token rosette with an equals sign: building this card completes a science pair, so its builder picks a progress token. Its tooltip names the symbol.',
  'rules.hints.loot':
    'Coin loss: this card’s shields push the pawn onto a looting token, and the other player loses coins.',
  'rules.hints.own': 'Filled in gold: the fact is for you.',
  'rules.hints.opp':
    'Outlined in the other player’s colour: the fact is for them, so taking or discarding the card denies it.',
  'rules.hints.seat':
    'On a shared device the filled mark takes the colour of the player to move, and a spectator sees each player’s own colour.',
  'rules.hints.more':
    'A small dot beside the mark: the card has more than one hint. Its tooltip lists them all.',
  'rules.hints.review':
    'The advisor, when the game turns it on, reviews your own move after you play: the summary of the other player’s next move says whether a clearly better move was there, and why. Only you see it.',
  'rules.hints.sampleCard': 'a card'
} as const satisfies Messages;
