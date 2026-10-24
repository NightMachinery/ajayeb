// The admin controls', role tags', resignation's and spectator view's
// messages (games/haftgane/table/Admin*.svelte, RoleTag, Spectator*.svelte,
// TableMenu). The actions are worded as phone-layout.md section 3 words
// them; every confirmation says what changes.

import type { Messages } from '$lib/i18n';

export const admin = {
  'hg.admin.role.creator': 'creator',
  'hg.admin.role.admin': 'admin',
  'hg.admin.role.temporary': 'temporary admin',
  'hg.admin.role.creator.tip': '{name} created this table and is an admin.',
  'hg.admin.role.admin.tip': '{name} is an admin: they can seat players and bots.',
  'hg.admin.role.temporary.tip':
    '{name} is a temporary admin while every admin is away. It lasts until an admin returns.',

  'hg.admin.seat.open': 'Manage the seat of {name}',
  'hg.admin.seat.title': 'Seat {seat}: {name}',
  'hg.admin.seat.close': 'Close the seat menu',
  'hg.admin.seat.player': 'Played by {name}',
  'hg.admin.seat.bot': 'Played by a bot',
  'hg.admin.seat.botLevel': 'Played by a bot: {bot}',
  'hg.admin.seat.toBot': 'Give the seat to a bot',
  'hg.admin.seat.changeLevel': 'Give the seat to another bot',
  'hg.admin.seat.seatHere': 'Seat a spectator here',
  'hg.admin.seat.noSpectators': 'No spectator is connected to seat.',
  'hg.admin.seat.hint': 'earlier in seat {seat}',
  'hg.admin.seat.hint.tip': 'Only admins see where a spectator sat earlier in this game.',
  'hg.admin.seat.roles': 'Admin role',
  'hg.admin.seat.temporary':
    'As a temporary admin you can seat players and bots, but not change who is an admin.',

  'hg.admin.confirm.toBot':
    '{bot} plays seat {seat} from now on. {name} watches, and their pick, confirmed or not, is played by the bot.',
  'hg.admin.confirm.botToBot': '{bot} plays seat {seat} from now on.',
  'hg.admin.confirm.seat':
    "{name} plays seat {seat} from now on. The bot's pick stays until {name} changes it.",
  'hg.admin.confirm.seatReplace':
    '{name} plays seat {seat} from now on, and {old} watches. Their pick stays until {name} changes it.',
  'hg.admin.confirm.promote':
    '{name} becomes an admin: they can seat players and bots and make others admins.',
  'hg.admin.confirm.demote': '{name} stops being an admin.',
  'hg.admin.confirm.stepDown':
    'You stop being an admin of this table. Only another admin can make you one again.',
  'hg.admin.confirm.resign':
    'A bot plays your seat from now on, and you watch the rest of the game. An admin can seat you again.',
  'hg.admin.confirm.ok': 'Confirm',
  'hg.admin.confirm.back': 'Back',

  'hg.admin.promote': 'Make admin',
  'hg.admin.demote': 'Remove admin',
  'hg.admin.stepDown': 'Stop being admin',
  'hg.admin.resign': 'Resign to a bot',
  'hg.admin.undo': 'Undo',
  'hg.admin.undo.tip': 'Take back your last turn',
  'hg.admin.undoOff': 'Another player took a seat, so undo is off for the rest of the game.',

  'hg.admin.seats': 'Seats',
  'hg.admin.seats.manage': 'Manage',
  'hg.admin.spectators': { one: '{count} spectator', other: '{count} spectators' },
  'hg.admin.spectators.title': 'Spectators',
  'hg.admin.spectators.none': 'Nobody is watching.',
  'hg.admin.spectators.open': {
    one: '{count} person is watching. Show who.',
    other: '{count} people are watching. Show who.'
  },
  'hg.admin.you': 'you',
  'hg.admin.away': 'away',
  'hg.admin.away.tip': '{name} is not connected.',

  'hg.watch.title': 'You are watching',
  'hg.watch.follow': 'Follow a seat',
  'hg.watch.follow.tip': 'Centre the table on {name}',
  'hg.watch.last': 'Last turn',
  'hg.watch.former.bot': 'A bot is playing your seat. An admin can seat you again.',
  'hg.watch.former.human': '{name} plays your seat now. An admin can seat you again.'
} as const satisfies Messages;
