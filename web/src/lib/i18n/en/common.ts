// English messages shared by the app shell and the Home screen.
//
// Conventions (see ../index.ts):
//   - Keys are flat and dotted: '<area>.<name>'. Other areas get their own
//     file next to this one (cards.ts, errors.ts ...) and are merged in
//     ../index.ts.
//   - {name} is a placeholder. t() checks at compile time that every
//     placeholder gets a value. Numbers are formatted for the locale.
//   - A message that depends on a count is an object of plural forms, chosen
//     with Intl.PluralRules on params.count: zero, one, two, few, many, other
//     ('other' is required), plus exact matches such as '=0', which win.

import type { Messages } from '../types';

export const common = {
  'app.name': 'Ajayeb',
  // The Persian name, shown as decoration next to the Latin one in every locale.
  'app.nameNative': 'عجایب',
  'app.tagline': 'A duel of wonders for two',
  'app.legal':
    'Ajayeb is an unofficial, non-commercial fan implementation of the rules of 7 Wonders Duel by Antoine Bauza and Bruno Cathala (Repos Production). It is not affiliated with or endorsed by Repos Production or Asmodee; 7 Wonders Duel is their trademark.',
  'app.getGame': 'Get the original game from its publisher',
  'app.home': 'Back to the start',

  'error.notFound': 'This page does not exist.',
  'error.generic': 'Something went wrong.',
  'error.status': 'Error {status}',

  'home.modes': 'Game modes',
  'home.playAi': 'Play vs AI',
  'home.playAiHint': 'Against the computer',
  'home.passAndPlay': 'Pass and play',
  'home.passAndPlayHint': 'Two players, one device',
  'home.playOnline': 'Play online',
  'home.playOnlineHint': 'Invite a friend with a link',
  'home.soon': 'Soon',
  'home.comingSoon':
    'Coming soon. The game is still being built, and each mode opens here as soon as it is ready.',

  'theme.label': 'Theme',
  'theme.system': 'System',
  'theme.light': 'Light',
  'theme.dark': 'Dark',

  'turn.player': "{name}'s turn",
  'cards.left': {
    '=0': 'No cards left',
    one: '{count} card left',
    other: '{count} cards left'
  },
  'coins.count': { one: '{count} coin', other: '{count} coins' },

  'dev.title': 'Developer pages',
  'dev.intro': 'These pages exist only in development builds. Nothing is listed here yet.',

  'dev.gallery.title': 'Gallery',
  'dev.gallery.intro':
    'Every hand-authored icon and shell primitive (components/shell/*), for a visual check in both themes and at phone and desktop widths. Component slices add their own section below as they land.',
  'dev.gallery.viewport': 'Viewport {width}×{height}',
  'dev.gallery.icons': 'Icons ({count})',
  'dev.gallery.shell': 'Shell primitives',
  'dev.gallery.cards': 'Cards',
  'dev.gallery.board': 'Board',
  'dev.gallery.screens': 'Screens'
} as const satisfies Messages;
