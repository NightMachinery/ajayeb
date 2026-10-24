// The end of an Age and the end of the game (games/haftgane/table/): the
// military dialog, the score sheet, the final board's revealed cards and
// their tooltips. Every number arrives as a param, and each is one the
// server sent: a shield count, a seat's military score, a total, a rank,
// coins. A conflict token's points are not sent yet, so a victory names
// its Age instead.

import type { Messages } from '$lib/i18n';

export const end = {
  'hg.end.you': 'you',

  'hg.end.military.title': 'Military, Age {age}',
  'hg.end.military.ages': 'Conflicts of each Age',
  'hg.end.military.yours': 'Your conflicts',
  'hg.end.military.left': 'Left neighbour',
  'hg.end.military.right': 'Right neighbour',
  'hg.end.military.shields': 'Shields: you {you}, {name} {them}',
  'hg.end.military.all': 'Every conflict around the table',
  'hg.end.military.points': 'Military points so far',
  'hg.end.military.pointsTip': '{name}: {points} military points so far',
  'hg.end.military.yourPoints': 'Your military points so far: {points}',
  'hg.end.military.close': 'Continue',
  'hg.end.result.victory': 'Victory',
  'hg.end.result.defeat': 'Defeat',
  'hg.end.result.tie': 'Tie',
  'hg.end.token.victory': '{name} takes a victory token of Age {age}.',
  'hg.end.token.defeat': '{name} takes a defeat token.',
  'hg.end.token.tie': '{a} and {b} have equal shields: no token.',
  'hg.end.token.youWin': 'You beat {name} and take a victory token of Age {age}.',
  'hg.end.token.youLose': '{name} beats you: you take a defeat token.',
  'hg.end.token.youTie': 'You and {name} have equal shields: no token.',
  'hg.end.pair.shields': 'Shields at the end of Age {age}: {a} {aCount}, {b} {bCount}.',

  'hg.end.verdict.you': 'You win',
  'hg.end.verdict.one': '{name} wins',
  'hg.end.verdict.sharedYou': 'You share the victory with {names}',
  'hg.end.verdict.shared': '{names} share the victory',
  'hg.end.verdict.none': 'Game over',
  'hg.end.tie.coins': '{names} scored {total} each. The most coins break the tie: {counts}.',
  'hg.end.tie.count': '{name} {coins}',
  'hg.end.tie.separator': ', ',
  'hg.end.tie.shared':
    '{names} scored {total} each and hold {coins} coins each, so they share the victory.',
  'hg.end.score.player': 'Player',
  'hg.end.score.col.wonder': 'Wonder',
  'hg.end.score.col.treasure': 'Coins: a point for every 3',
  'hg.end.score.col.military': 'Military',
  'hg.end.score.col.blue': 'Civic buildings',
  'hg.end.score.col.yellow': 'Commercial buildings',
  'hg.end.score.col.green': 'Science',
  'hg.end.score.col.purple': 'Guilds',
  'hg.end.score.col.total': 'Total',
  'hg.end.score.rank': 'Place {rank}',
  'hg.end.score.totalTip': '{name}: {total} points, {coins} coins at the end',
  'hg.end.download': 'Download log',
  'hg.end.download.hidden': 'This table keeps the end hidden, so its log cannot be downloaded.',
  'hg.end.board': 'Final board',
  'hg.end.home': 'Home',

  'hg.end.open.scores': 'Scores',
  'hg.end.open.military': 'Conflicts',
  'hg.end.open.reveal': 'Revealed cards',

  'hg.end.reveal.title': 'Revealed at the end',
  'hg.end.reveal.tucked': 'Cards under the wonder stages',
  'hg.end.reveal.leftovers': 'Last cards of Age {age}',
  'hg.end.reveal.kept': 'kept for the last turn',
  'hg.end.reveal.pile': {
    one: 'The discard pile: {count} card',
    other: 'The discard pile: {count} cards'
  },
  'hg.end.reveal.box': 'Guilds left in the box',
  'hg.end.reveal.none': 'None',
  'hg.end.reveal.close': 'Close'
} as const satisfies Messages;
