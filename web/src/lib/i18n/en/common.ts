// English messages shared by the whole app (the shell, error pages, the
// developer pages). The Home screen's own keys live in home.ts.
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
  'text.sentence': '{text}.',
  'text.join': '{first} {second}',
  'text.phrases': '{first}. {second}',
  'text.detail': '{label}: {detail}',

  'app.name': 'Ajayeb',
  // The Persian name, shown as decoration next to the Latin one in every locale.
  'app.nameNative': 'عجایب',
  'app.tagline': 'A duel of wonders for two',
  'app.legal':
    'Ajayeb is an unofficial, non-commercial fan implementation of the rules of 7 Wonders Duel by Antoine Bauza and Bruno Cathala (Repos Production). It is not affiliated with or endorsed by Repos Production or Asmodee; 7 Wonders Duel is their trademark.',
  'app.getGame': 'Get the original game from its publisher',
  'app.foss': 'Ajayeb itself is free and open-source software, released under the MIT License.',
  'app.source': 'Source code on GitHub',
  'app.home': 'Back to the start',

  'error.notFound': 'This page does not exist.',
  'error.generic': 'Something went wrong.',
  'error.status': 'Error {status}',

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
  'dev.gallery.screens': 'Screens',
  'dev.gallery.ornaments': 'Ornaments, track and logo',
  'dev.gallery.ornaments.track': 'Military track cells at phone width',
  'dev.gallery.ornaments.capitals': 'Capitals at 16, 20, 24 and 48 px',
  'dev.gallery.ornaments.card': 'Card corner rosette and band cartouche',
  'dev.gallery.ornaments.sheet': 'Divider and panel border',
  'dev.gallery.ornaments.table': 'Girih tile: seam test and table texture',
  'dev.gallery.ornaments.logo': 'Wordmark at 200 px and the favicon',
  'dev.gallery.sound': 'Sound board',
  'dev.gallery.sound.intro':
    'One button per sound cue, played at the volume chosen here (the Sound setting does not need to be on).',
  'dev.gallery.sound.volume': 'Volume',
  'dev.gallery.sound.cue.slide': 'Card slide',
  'dev.gallery.sound.cue.coin': 'Coin clink',
  'dev.gallery.sound.cue.flip': 'Flip',
  'dev.gallery.sound.cue.drum': 'War drum',
  'dev.gallery.sound.cue.chime': 'Science chime',
  'dev.gallery.sound.cue.fanfare': 'Fanfare',
  'dev.gallery.sound.cue.tick': 'Draft tick',
  'dev.gallery.sound.all': 'Play a busy move',
  'dev.gallery.sound.allHint': 'A build that pays, moves the pawn and loots, as one frame plays it.'
} as const satisfies Messages;
