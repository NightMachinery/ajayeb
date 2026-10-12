// English messages of the connection layer (src/lib/session): connection
// states, the server's reject and error codes, and the temporary debug board
// (src/lib/session/DebugBoard.svelte) with the dev room pages, which M5.S04
// replaces with the real game screen.

import type { Messages } from '../types';

export const session = {
  'session.conn.connecting': 'Connecting…',
  'session.conn.open': 'Connected',
  'session.conn.reconnecting': 'Reconnecting…',
  'session.conn.offline': 'Offline. Trying to reconnect…',
  'session.pendingSlow': 'Waiting for the server…',
  'session.catalogStale':
    'The server plays with newer card data than this page shows. Reload the page to update it.',

  'session.reject.stale': 'The game moved on before your move arrived. Please choose again.',
  'session.reject.illegal': 'That move is not allowed now.',
  'session.reject.notYourTurn': 'It is not your turn.',
  'session.reject.rate': 'Too many requests at once. Wait a moment and try again.',
  'session.reject.badRequest': 'The server could not use that request.',
  'session.reject.notAdmin': 'Only the player who created the game can change its hints.',
  'session.reject.gameOver': 'The game is over, so its hints no longer change.',

  'session.error.roomNotFound': 'No game has that code.',
  'session.error.roomFull': 'This game takes no more connections.',
  'session.error.forbidden': 'You cannot do that in this game.',
  'session.error.badHello': 'The server could not identify this browser. Reload the page.',
  'session.error.protocol': 'A newer version of the game is out. Reload the page to continue.',
  'session.error.busy': 'The server is full right now. Try again in a few minutes.',
  'session.error.badClaim':
    'This device link no longer works. Ask the player for a new one, or open the game without it.',
  'session.error.unknown': 'Something went wrong.',

  'debug.title': 'Room {code}',
  'debug.note': 'A plain debug view of the game. The real game screen replaces it.',
  'debug.loading': 'Waiting for the game…',
  'debug.connection': 'Connection',
  'debug.mode': 'Mode',
  'debug.status': 'Status',
  'debug.version': 'Version',
  'debug.phase': 'Phase',
  'debug.age': 'Age',
  'debug.toMove': 'To move',
  'debug.active': 'Active seat',
  'debug.extraTurn': 'Plays again',
  'debug.yes': 'yes',
  'debug.no': 'no',
  'debug.none': 'none',
  'debug.seat': 'Seat {seat}',
  'debug.you': 'you',
  'debug.coins': 'Coins',
  'debug.points': 'Points',
  'debug.military': 'Conflict pawn',
  'debug.city': 'City',
  'debug.wonders': 'Wonders',
  'debug.built': 'built',
  'debug.tokens': 'Tokens',
  'debug.boardTokens': 'Progress tokens on the board',
  'debug.discard': 'Discard pile',
  'debug.pending': 'Open choice',
  'debug.hiddenTokens': { one: '{count} hidden token', other: '{count} hidden tokens' },
  'debug.structure': 'Structure',
  'debug.accessible': 'takeable',
  'debug.choices': 'Choices',
  'debug.noChoices': 'Nothing to choose now.',
  'debug.cost': { '=0': 'free', one: 'pays {count} coin', other: 'pays {count} coins' },
  'debug.gain': { one: 'gains {count} coin', other: 'gains {count} coins' },
  'debug.undo': 'Undo',
  'debug.resync': 'Refresh',
  'debug.result': 'Result',
  'debug.winner': 'Winner',
  'debug.shared': 'shared victory',
  'debug.rematch': 'Ask for a rematch',
  'debug.toRematch': 'Go to the rematch',
  'debug.thinking': 'Seat {seat} is thinking…',

  'dev.new.title': 'New room',
  'dev.new.creating': 'Creating a {mode} room…',
  'dev.new.failed': 'Could not create the room: {code}',
  'dev.new.badMode': 'Unknown mode. Use mode=hotseat, ai or online.',
  'dev.fixture.title': 'Recorded fixtures',
  'dev.fixture.unknown': 'No recorded fixture is called {name}.'
} as const satisfies Messages;
