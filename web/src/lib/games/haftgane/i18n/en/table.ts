// The table session's messages: what a refused request means, what
// happened to this tab's seat, and the minimal development screen that
// drives the Session until the table screen (M18.table-screen) replaces it.

import type { Messages } from '$lib/i18n';

export const table = {
  'hg.table.reject.decision-closed': 'That decision has already closed.',
  'hg.table.reject.illegal': 'That choice is not open to you now.',
  'hg.table.reject.stale': 'The table moved on before your request arrived. Choose again.',
  'hg.table.reject.bad-request': 'The server could not use that request.',
  'hg.table.reject.not-admin': 'Only an admin of this table can do that.',
  'hg.table.reject.forbidden': 'You cannot do that at this table now.',
  'hg.table.reject.rate': 'Too many requests at once. Wait a moment and try again.',
  'hg.table.reject.storage': 'The server could not save that. Try again.',
  'hg.table.reject.timeout': 'No answer came. Try again.',
  'hg.table.reject.unknown': 'That request was refused.',

  'hg.table.seat.watching': 'You no longer hold a seat. You are watching this table.',
  'hg.table.seat.seated': 'You now play seat {seat}.',

  'hg.table.dev.lobby': 'Lobby {code}',
  'hg.table.dev.table': 'Table {code}',
  'hg.table.dev.seats': 'Seats',
  'hg.table.dev.seat': 'Seat {seat}',
  'hg.table.dev.open': 'Open',
  'hg.table.dev.reserved': 'Reserved',
  'hg.table.dev.bot': 'Bot ({level})',
  'hg.table.dev.you': 'you',
  'hg.table.dev.away': 'away',
  'hg.table.dev.admin': 'admin',
  'hg.table.dev.temporary': 'temporary admin',
  'hg.table.dev.start': 'Start, filling open seats with bots',
  'hg.table.dev.point': 'Decision {id}',
  'hg.table.dev.decide': 'Your choice',
  'hg.table.dev.waiting': 'Waiting for the other seats',
  'hg.table.dev.watching': 'You are watching.',
  'hg.table.dev.picked': 'Your pick: {key}',
  'hg.table.dev.confirmed': 'Confirmed',
  'hg.table.dev.confirm': 'Confirm',
  'hg.table.dev.change': 'Change',
  'hg.table.dev.undo': 'Undo',
  'hg.table.dev.resign': 'Resign',
  'hg.table.dev.toBot': 'Seat {seat} to a bot',
  'hg.table.dev.seatUser': 'Seat {name} in seat {seat}',
  'hg.table.dev.finished': 'The game is over.',
  'hg.table.dev.dismiss': 'Dismiss',
  'hg.table.dev.create.title': 'New table',
  'hg.table.dev.create.creating': 'Creating a lobby of {count} seats…',
  'hg.table.dev.create.failed': 'Could not create the lobby: {code}',
  'hg.table.dev.create.badSeats': 'Use seats=3 to seats=7.'
} as const satisfies Messages;
