// English messages for the board chrome (components/board/*, components/
// token/*, components/city/*, components/hud/*, components/sheets/{City,Log}
// Sheet.svelte) and their view mappers (lib/view/{threats,turn,log-format}.ts).
// See ../index.ts for the conventions.
//
// Card and wonder names live in card.ts ('card.name.*', 'wonder.name.*').
// The 10 progress token names are keyed here by engine id, along with the
// science symbols, card colours and resources as the board chrome shows
// them.

import type { Messages } from '../types';

export const board = {
  // Military track (MilitaryTrack.svelte, LootingToken.svelte; data from
  // lib/view/threats.ts). "you"/"the opponent" wording is used when the
  // component is given the viewer's own seat; otherwise it names the
  // crests, which reads sensibly for a spectator or in the gallery.
  'board.crest.sun': 'Sun',
  'board.crest.moon': 'Moon',
  'board.military.youLead': 'Military: you lead by {margin}.',
  'board.military.opponentLeads': 'Military: the opponent leads by {margin}.',
  'board.military.crestLeads': 'Military: {crest} leads by {margin}.',
  'board.military.tied': 'Military: even.',
  'board.military.nextLootYou': 'Next looting: you lose {value} coins at {magnitude}.',
  'board.military.nextLootOpponent':
    'Next looting: the opponent loses {value} coins at {magnitude}.',
  'board.military.nextLootCrest': 'Next looting: {crest} loses {value} coins at {magnitude}.',
  'board.military.noLootLeft': 'No looting tokens remain; the capital is next.',
  'board.military.lootCell': '{value} coins at {magnitude}',
  'board.military.lootTaken': '{value} coins at {magnitude}, already claimed',
  'board.military.capitalYou': 'Your capital',
  'board.military.capitalOpponent': "The opponent's capital",
  'board.military.capitalCrest': "{crest}'s capital",

  // Progress tokens (ProgressToken.svelte, ProgressTokenRow.svelte).
  'token.agriculture': 'Agriculture',
  'token.architecture': 'Architecture',
  'token.economy': 'Economy',
  'token.law': 'Law',
  'token.masonry': 'Masonry',
  'token.mathematics': 'Mathematics',
  'token.philosophy': 'Philosophy',
  'token.strategy': 'Strategy',
  'token.theology': 'Theology',
  'token.urbanism': 'Urbanism',
  'board.token.take': 'Take the {name} token',
  'board.token.owned': '{name} (owned)',
  'board.tokens.boardLabel': 'Progress tokens on the board',
  'board.tokens.empty': 'No progress tokens remain on the board.',

  // Science symbols (ScienceTracker.svelte) and the 7 card colours
  // (CityBand.svelte and elsewhere).
  'board.science.globe': 'Globe',
  'board.science.scales': 'Scales',
  'board.science.sundial': 'Sundial',
  'board.science.mortar': 'Mortar',
  'board.science.set-square': 'Set Square',
  'board.science.quill': 'Quill',
  'board.science.wheel': 'Wheel',
  'board.science.label': '{count} of 6 distinct science symbols',
  'board.science.danger': 'Close to a scientific victory',
  'board.color.brown': 'Brown',
  'board.color.grey': 'Grey',
  'board.color.blue': 'Blue',
  'board.color.green': 'Green',
  'board.color.yellow': 'Yellow',
  'board.color.red': 'Red',
  'board.color.purple': 'Purple',
  'board.city.colorCount': {
    one: '{count} {color} card',
    other: '{count} {color} cards'
  },

  // Resources (ResourceChips.svelte).
  'board.resource.wood': 'Wood',
  'board.resource.clay': 'Clay',
  'board.resource.stone': 'Stone',
  'board.resource.glass': 'Glass',
  'board.resource.papyrus': 'Papyrus',
  'board.resource.chip': '{resource}: produce {count}, buy at {price} coins each',
  'board.resource.choiceHint': 'Also available from a choice producer',

  // Wonders (WonderRow.svelte).
  'board.wonder.built': 'Built',
  'board.wonder.unbuilt': 'Not yet built',
  'board.wonder.removed': 'Left the game unbuilt',
  'board.wonder.tuckedUnder': 'Built using {card}',

  // PlayerStrip.svelte / CityPanel.svelte.
  'board.player.militaryDanger': 'Military danger: the pawn is close to this capital',
  'board.player.scienceDanger': 'Close to a scientific victory',
  'board.player.points': { one: '{count} point', other: '{count} points' },
  'board.player.scorePreview': 'If the game ended now: {total} points',
  'board.player.chains': 'Builds free: {count}',
  // The calm indicator on a bot's strip or panel while it thinks.
  'board.player.thinking': 'thinking',

  // TurnBar (lib/view/turn.ts supplies the state; this file supplies the
  // wording). 'turn.player' (common.ts) covers the plain "{name}'s turn".
  'board.turn.yourTurn': 'Your turn',
  // The whole extra turn a wonder grants; a hot-seat device names the player.
  'board.turn.playAgain': 'Play again',
  'board.turn.playAgainWonder': 'Play again ({wonder})',
  'board.turn.playAgainName': '{name} plays again',
  'board.turn.playAgainNameWonder': '{name} plays again ({wonder})',
  'board.turn.aiThinking': '{name} is thinking…',
  'board.turn.waitingFor': 'Waiting for {name}…',
  'board.turn.waitingAway': '{name} is away. Waiting for them to come back…',
  'board.turn.sent': 'Sending…',
  'board.turn.sentSlow': 'Still waiting for the server…',
  'board.turn.retry': 'Retry',
  'board.turn.gameOver': 'Game over',
  'board.turn.spectating': 'Watching',
  'board.turn.connecting': 'Connecting…',
  'board.turn.pending.chooseStarter': 'Choose who starts the next age',
  'board.turn.pending.destroyCard': 'Choose a card to destroy',
  'board.turn.pending.buildFromDiscard': 'Choose a card to build from the discard pile',
  'board.turn.pending.pickBoardToken': 'Choose a progress token',
  'board.turn.pending.pickLibraryToken': 'Choose a token from the Great Library',

  // TopBar.svelte.
  'board.age': 'Age {age}',
  'board.age.draft': 'Wonder draft',
  'board.topbar.undo': 'Undo',
  'board.topbar.log': 'Log',
  'board.topbar.settings': 'Settings',

  // CitySheet.svelte.
  'board.citySheet.title': 'City',
  'board.citySheet.resources': 'Resources',
  'board.citySheet.science': 'Science',
  'board.citySheet.wonders': 'Wonders',
  'board.citySheet.tokens': 'Tokens',
  'board.citySheet.city': 'City',
  'board.citySheet.chains': 'Builds free',

  // LogSheet.svelte and lib/view/log-format.ts. {name} is a display name,
  // interpolated as data like a card or wonder name, never a server string.
  'board.logSheet.title': 'Log',
  'board.logSheet.empty': 'Nothing has happened yet.',
  'board.logSheet.age': 'Age {age}',
  'board.logSheet.draftAge': 'Wonder draft',
  'board.log.draft': '{name} drafted {wonder}.',
  'board.log.draftRound2': 'The second round of wonders is revealed.',
  'board.log.ageStart': 'Age {age} begins.',
  'board.log.pawn': 'The military pawn moves toward {crest}.',
  'board.log.build': '{name} built {card}.',
  'board.log.discard': '{name} discarded {card} for {amount} coins.',
  'board.log.wonder': '{name} built the wonder {wonder}, using {card}.',
  'board.log.loot': '{name} loses {amount} coins to looting.',
  'board.log.pair': '{name} completes a pair of {symbol} symbols.',
  'board.log.token': '{name} takes the {token} token.',
  'board.log.libraryDraw': '{name} draws from the Great Library.',
  'board.log.destroy': '{name} destroys {card}.',
  'board.log.rebuild': '{name} builds {card} from the discard pile.',
  'board.log.wonderRemoved': "{name}'s unbuilt wonder {wonder} leaves the game.",
  'board.log.effectSkipped': '{name} has no option for that choice.',
  'board.log.extraTurn': '{name} plays again.',
  'board.log.extraTurnLost': "{name}'s extra turn is lost at the end of the age.",
  'board.log.starterChosen': '{name} chooses {starter} to start the next age.'
} as const satisfies Messages;
