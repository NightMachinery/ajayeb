// The city sheet's and the Seen hands sheet's messages
// (games/haftgane/table/City*.svelte, SeenHands*.svelte). Every number is
// one the server sent or a count of what the frame holds.

import type { Messages } from '$lib/i18n';

export const city = {
  'hg.city.open': 'Open the city of {name}',
  'hg.city.close': 'Close the city',
  'hg.city.tabs': 'Cities around the table',
  'hg.city.wonder': '{wonder}, {side} side',
  'hg.city.wonder.noSide': '{wonder}, no side chosen yet',

  'hg.city.score': 'Points so far',
  'hg.city.score.wonder': 'Wonder',
  'hg.city.score.treasure': 'Coins',
  'hg.city.score.military': 'Conflicts',
  'hg.city.score.blue': 'Civic',
  'hg.city.score.yellow': 'Commerce',
  'hg.city.score.green': 'Science',
  'hg.city.score.purple': 'Guilds',
  'hg.city.victories.tip': {
    one: 'Age {age}: {count} victory token',
    other: 'Age {age}: {count} victory tokens'
  },
  'hg.city.defeats.tip': { one: '{count} defeat token', other: '{count} defeat tokens' },
  'hg.city.science.tip': {
    one: 'Science: {compass} compass, {gear} gear, {tablet} tablet, worth {count} point',
    other: 'Science: {compass} compass, {gear} gear, {tablet} tablet, worth {count} points'
  },
  'hg.city.science.chosen': {
    one: '{count} of them from a symbol of choice, counted where it scores most',
    other: '{count} of them from symbols of choice, counted where they score most'
  },

  'hg.city.stages': 'Wonder stages',
  'hg.city.stage.label': 'Stage {stage}',
  'hg.city.stage.built': 'Built with a hidden card',
  'hg.city.stage.notBuilt': 'Not built',
  'hg.city.stage.cost': 'Cost',
  'hg.city.under.own': 'Under your wonder, seen only by you: {cards}',
  'hg.city.under.revealed': 'Under the wonder, revealed at the end: {cards}',
  'hg.city.kept.own': 'You keep {card} to play at the end of the Age.',
  'hg.city.kept.other': 'Keeps a card to play at the end of the Age.',

  'hg.city.production': 'Production',
  'hg.city.production.own': 'your two neighbours can buy it',
  'hg.city.production.other': 'only the two neighbours of {name} can buy it',
  'hg.city.buildings': {
    one: 'City · {count} building',
    other: 'City · {count} buildings'
  },
  'hg.city.row.none': 'none',
  // In place of the colour rows, while a city has no building.
  'hg.city.row.nothing': 'Nothing built yet',
  // Between two card names in a list.
  'hg.city.row.sep': ', ',
  'hg.city.last': 'Last turn: {what}',
  'hg.city.last.build': 'built {card}',
  'hg.city.last.build-discarded': 'built {card} from the discard pile',
  'hg.city.last.stage': 'built wonder stage {stage}',
  'hg.city.last.sell': 'sold a card',

  'hg.seen.open': 'Seen hands',
  'hg.seen.open.tip': 'The hands you have held and the pile you have looked through',
  'hg.seen.title': 'Seen hands',
  'hg.seen.close': 'Close Seen hands',
  'hg.seen.intro': 'Only you see this notebook. Bots remember exactly as much.',
  'hg.seen.ages': 'Ages',
  'hg.seen.turn': 'Turn {turn}',
  'hg.seen.lastCard': 'Last card',
  'hg.seen.pile': 'The discard pile you looked through, turn {turn}',
  'hg.seen.played.build': 'Built',
  'hg.seen.played.stage': 'Under a stage',
  'hg.seen.played.sell': 'Sold',
  'hg.seen.played.build-discarded': 'Built',
  'hg.seen.played.tip.build': 'You built this card.',
  'hg.seen.played.tip.stage': 'You put this card under a wonder stage.',
  'hg.seen.played.tip.sell': 'You sold this card.',
  'hg.seen.played.tip.build-discarded': 'You built this card from the discard pile.',
  'hg.seen.empty': 'Nothing yet. Each hand you hold appears here once its turn is revealed.',
  'hg.seen.emptyAge': 'Nothing from this Age yet.',

  'hg.seen.summary': 'What you know',
  'hg.seen.circulating': 'Cards from hands you passed that may still be going round',
  'hg.seen.circulating.none': 'None of the cards you passed can still be going round.',
  'hg.seen.bounds.exact': '{count} for certain',
  'hg.seen.bounds.atMost': 'at most {count}',
  'hg.seen.bounds.range': '{certain} for certain, at most {atMost}',
  'hg.seen.colour.tip': '{colour}: {bounds}. Tap to show these cards.',
  'hg.seen.picks': 'Cards your neighbours took from hands you passed',
  'hg.seen.pick': '{name} took {card} (Age {age}, turn {turn})',
  'hg.seen.pick.tip': '{name} took {card}. Tap to show where you saw it.',
  'hg.seen.returning': 'Cards that can still come back to you this Age',
  'hg.seen.returning.tip': '{card}: {bounds}. Tap to show where you saw it.',
  'hg.seen.filter': 'Showing: {what}',
  'hg.seen.filter.clear': 'Show all',
  'hg.seen.card.tip': '{card}: {effect}'
} as const satisfies Messages;
