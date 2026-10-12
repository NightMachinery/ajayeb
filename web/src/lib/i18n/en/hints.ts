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
  'hints.log.changed': 'The hints changed: {changes}.'
} as const satisfies Messages;
