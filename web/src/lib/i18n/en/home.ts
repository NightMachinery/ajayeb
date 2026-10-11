// English messages for the Home and setup screens
// (components/screens/Home, NewGameSheet, HotseatSheet, Settings,
// PassDevice). See ../index.ts for the conventions.

import type { Messages } from '../types';

export const home = {
  // Home: the game modes.
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

  // Home: the saved-rooms "Continue" list and the Settings entry point.
  'home.continueLabel': 'Continue a game',
  'home.continueEntry': 'Continue: {mode} · {code}',
  'home.mode.online': 'Online',
  'home.mode.hotseat': 'Pass and play',
  'home.mode.ai': 'vs AI',
  'home.settings': 'Settings',

  // Home: joining an online game by its code.
  'home.joinCode': 'Join with a code',

  // NewGameSheet (vs AI).
  'home.ai.title': 'Choose your rival',
  'home.ai.levelLegend': 'Difficulty',
  'home.ai.apprentice': 'Apprentice',
  'home.ai.apprenticeHint': 'Learning the ropes',
  'home.ai.scholar': 'Scholar',
  'home.ai.scholarHint': 'A solid player',
  'home.ai.master': 'Master',
  'home.ai.masterHint': 'Merciless',
  'home.ai.seatLegend': 'Your crest',
  'home.ai.seatRandom': 'Random',
  'home.ai.seatSun': 'Sun',
  'home.ai.seatMoon': 'Moon',
  'home.ai.firstRandomNote': 'Who plays first is drawn at random.',
  'home.ai.start': 'Start game',

  // HotseatSheet (pass and play).
  'home.hotseat.title': 'Pass and play',
  'home.hotseat.player1Label': 'Player 1 (Sun crest)',
  'home.hotseat.player1Default': 'Player 1',
  'home.hotseat.player2Label': 'Player 2 (Moon crest)',
  'home.hotseat.player2Default': 'Player 2',
  'home.hotseat.firstRandomNote': 'Who plays first is drawn at random.',
  'home.hotseat.passDeviceToggle': 'Pass-device screen',
  'home.hotseat.passDeviceHint':
    'Between turns, a screen names the next player and sums up the last move.',
  'home.hotseat.orientationLegend': 'Orientation',
  'home.hotseat.orientationFace': 'Face active player',
  'home.hotseat.orientationFixed': 'Fixed',
  'home.hotseat.start': 'Start game',

  // Settings (the sheet and the /settings route share these).
  'home.settings.title': 'Settings',
  'home.settings.animationSpeedLegend': 'Animation speed',
  'home.settings.speedNormal': 'Normal',
  'home.settings.speedFast': 'Fast',
  'home.settings.speedRelaxed': 'Relaxed',
  'home.settings.speedOff': 'Off',
  'home.settings.confirmMovesLegend': 'Confirm moves',
  'home.settings.confirmOnlineOnly': 'Online only',
  'home.settings.confirmAlways': 'Always',
  'home.settings.confirmNever': 'Never',
  'home.settings.colorPatterns': 'Colour patterns',
  'home.settings.colorPatternsHint': 'Adds a pattern to each card colour, for colour-blind play.',
  'home.settings.cardTextLegend': 'Card text',
  'home.settings.cardTextNormal': 'Normal',
  'home.settings.cardTextLarge': 'Large',

  // PassDevice (the hot-seat interstitial; M5.S04 wires it into the game).
  'home.pass.title': 'Pass to {name}',
  'home.pass.ready': "I'm ready",
  'home.pass.undo': "Undo {name}'s move",

  // /dev/gallery demo content for this slice's screens.
  'home.demo.continueCode': 'Sample saved games',
  'home.demo.passSummary1': 'Darius built Lumber Yard (−4 coins)',
  'home.demo.passSummary2': 'Pawn moved 1 toward you'
} as const satisfies Messages;
