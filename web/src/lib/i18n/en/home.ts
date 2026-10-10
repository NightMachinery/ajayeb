// English messages for the Home and setup screens
// (components/screens/Home, NewGameSheet, HotseatSheet, Settings,
// PassDevice). The home.modes/playAi/passAndPlay/playOnline/soon/comingSoon
// keys Home itself still uses stay in common.ts: that file is not part of
// this slice's owned files (see AGENTS.md's per-slice ownership), so the
// planned move of those keys into this one waits for a slice that owns
// both. Everything new in this slice lives here. See ../index.ts for the
// conventions.

import type { Messages } from '../types';

export const home = {
  // Home: the saved-rooms "Continue" list and the Settings entry point.
  'home.continueLabel': 'Continue a game',
  'home.continueEntry': 'Continue: {mode} · {code}',
  'home.mode.online': 'Online',
  'home.mode.hotseat': 'Pass and play',
  'home.mode.ai': 'vs AI',
  'home.settings': 'Settings',

  // The tiny "online arrives later" sheet.
  'home.online.title': 'Play online',
  'home.online.comingSoon':
    'Online play arrives in a later version. For now, invite a friend to the same device with Pass and play.',

  // NewGameSheet (vs AI).
  'home.ai.title': 'Choose your rival',
  'home.ai.levelLegend': 'Difficulty',
  'home.ai.apprentice': 'Apprentice',
  'home.ai.apprenticeHint': 'Learning the ropes',
  'home.ai.scholar': 'Scholar',
  'home.ai.scholarHint': 'A solid player',
  'home.ai.master': 'Master',
  'home.ai.masterHint': 'Merciless',
  'home.ai.seatLegend': 'Your seat',
  'home.ai.seatRandom': 'Random',
  'home.ai.seatMe': 'Me',
  'home.ai.seatRival': 'Rival',
  'home.ai.start': 'Start game',

  // HotseatSheet (pass and play).
  'home.hotseat.title': 'Pass and play',
  'home.hotseat.player1Label': 'First player',
  'home.hotseat.player1Default': 'Player 1',
  'home.hotseat.player2Label': 'Second player',
  'home.hotseat.player2Default': 'Player 2',
  'home.hotseat.firstToPlayLegend': 'First to play',
  'home.hotseat.firstRandom': 'Random',
  'home.hotseat.firstOption': 'First',
  'home.hotseat.secondOption': 'Second',
  'home.hotseat.passDeviceToggle': 'Pass-device screen',
  'home.hotseat.passDeviceHint':
    'Shows a cover screen between turns, naming the next player, so the board is not seen early.',
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
  'home.settings.speedForcedNote':
    'Animations are off because your system asks for reduced motion.',
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
