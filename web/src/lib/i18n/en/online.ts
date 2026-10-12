// English messages for online play: the create and join sheets, the name
// prompt, the lobby with its invite, presence, spectators, the online
// rematch and the room errors (components/screens/OnlineSheet, JoinSheet,
// NamePrompt, Lobby, RoomError; components/hud/PresenceDot,
// SpectatorCount; components/identity), and the names of the players.
// See ../index.ts for the conventions.

import type { Messages } from '../types';

export const online = {
  // The create sheet (Home, Play online).
  'online.create.title': 'Play online',
  'online.create.intro': 'Create a game, then send the link to a friend.',
  'online.create.submit': 'Create game',
  'online.create.haveCode': 'Have a code? Join a game',
  'online.name.label': 'Your name',
  'online.name.hint': 'Your opponent sees this name.',
  // Suggested names, comma separated; one is offered when no name is stored.
  'online.name.suggestions': 'Kaveh,Roxana,Darius,Shirin,Babak,Parisa,Cyrus,Mitra',
  'online.crest.legend': 'Your crest',
  'online.crest.random': 'Random',
  'online.crest.sun': 'Sun',
  'online.crest.moon': 'Moon',
  'online.crest.note': 'Who plays first is drawn at random.',

  // The join sheet (Home, Join with a code).
  'online.join.title': 'Join a game',
  'online.join.label': 'Game code',
  'online.join.hint': 'The six letters and digits at the end of your friend’s link.',
  'online.join.invalid': 'A code has six letters and digits, and never uses I, L, O, 0 or 1.',
  'online.join.submit': 'Join',

  // The name prompt (a first visit through an invite link).
  'online.prompt.title': 'What should we call you?',
  'online.prompt.intro': 'Your friend sees this name during the game.',
  'online.prompt.submit': 'Play as {name}',

  // The lobby: the room before the second player arrives.
  'online.lobby.title': 'Invite a friend',
  'online.lobby.intro': 'Send them the link or the code. The game starts when they join.',
  'online.lobby.code': 'Game code',
  'online.lobby.link': 'Invite link',
  'online.lobby.copy': 'Copy link',
  'online.lobby.copied': 'Link copied.',
  'online.lobby.copyFailed': 'Could not copy the link. Select it and copy it by hand.',
  'online.lobby.share': 'Share',
  'online.lobby.shareTitle': 'A game of Ajayeb',
  'online.lobby.shareText': 'Join my game of Ajayeb. The code is {code}.',
  'online.lobby.qr': 'QR code of the invite link',
  'online.lobby.scan': 'Or scan this with the other phone.',
  'online.lobby.players': 'Players',
  'online.lobby.waiting': 'Waiting for your friend…',
  'online.lobby.home': 'Home',
  'online.lobby.homeHint': 'The game stays open: continue it from Home.',

  // Presence on a player's strip.
  'online.presence.connected': 'online',
  'online.presence.away': 'away',
  'online.presence.connectedLabel': '{name} is online',
  'online.presence.awayLabel': '{name} is away',

  // Toasts in the game screen.
  'online.toast.joined': '{name} joined. The draft begins.',
  'online.toast.left': '{name} left. The game waits for them.',
  'online.toast.back': '{name} is back.',
  'online.toast.rematch': '{name} wants a rematch.',

  // Spectators.
  'online.spectatorsLabel': {
    one: '{count} spectator is watching this game',
    other: '{count} spectators are watching this game'
  },
  'online.watching': 'You are watching: only the players make moves.',

  // The connection, on move buttons.
  'online.waitingConnection': 'Waiting for connection',

  // The rematch, on the result screen.
  'online.rematch.join': 'Join the rematch',
  'online.rematch.watch': 'Watch the rematch',

  // A room that cannot be opened.
  'online.error.title': 'This game cannot be opened',
  'online.error.notFoundHint':
    'Check the code or the link. A game also closes some time after it ends.',
  'online.error.retryHint': 'Try again in a moment, or start a game of your own.',
  'online.error.home': 'Home',

  // Who is who (components/identity, lib/view/display-name): a name that
  // equals one already in the game gets a number after it, and the
  // player's own name a small mark.
  'online.identity.numbered': '{name} {number}',
  'online.identity.you': 'you',
  'online.identity.youLabel': 'This is you',

  // Device links (components/identity/DeviceLink): a link that plays one
  // seat of this game on another device.
  'online.link.button': 'Device link for {name}',
  'online.link.buttonOwn': 'Play on another device',
  'online.link.title': 'Play on another device',
  'online.link.titleOther': 'Device link for {name}',
  'online.link.intro':
    'Open this link on your other device to play your seat there. Anyone who has it can play as you in this game, so keep it to yourself.',
  'online.link.introOther':
    'Send this link to {name}: it plays their seat on another device. Anyone who has it can play that seat in this game.',
  'online.link.field': 'Device link',
  'online.link.loading': 'Getting the link…',
  'online.link.failed': 'Could not get the link. Try again.',
  'online.link.retry': 'Try again',
  'online.link.copy': 'Copy link',
  'online.link.copied': 'Link copied.',
  'online.link.copyFailed': 'Could not copy the link. Select it and copy it by hand.',
  'online.link.reset': 'Reset link',
  'online.link.resetHint': 'A reset stops every earlier link to this seat at once.',
  'online.link.resetDone': 'Link reset. Earlier links no longer work.',

  // The game log: a reset of a device link.
  'online.log.linkResetOwn': '{name} reset their device link.',
  'online.log.linkResetOther': '{name} reset the device link of {target}.',
  'online.log.linkReset': 'The device link of {target} was reset.'
} as const satisfies Messages;
