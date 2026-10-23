// Haftgane's names, keyed by engine id: the seven boards (the city and its
// wonder), the seven resources, the three science symbols, and the few
// cards whose names the rules reference needs. Card ids are the catalog's
// design slugs (docs/haftgane/engine-architecture.md, "Ids and slugs"):
// kebab-case English names, with the Age on the three names printed in two
// Ages (glassworks-age-1 and glassworks-age-2 share one name). The full card
// list arrives with the catalog; names are added here as screens need them.

import type { Messages } from '$lib/i18n';

export const names = {
  'hg.board.alexandria': 'Alexandria',
  'hg.board.babylon': 'Babylon',
  'hg.board.ephesos': 'Ephesos',
  'hg.board.gizah': 'Gizah',
  'hg.board.halikarnassos': 'Halikarnassos',
  'hg.board.olympia': 'Olympia',
  'hg.board.rhodos': 'Rhodos',

  'hg.wonder.alexandria': 'The Lighthouse',
  'hg.wonder.babylon': 'The Hanging Gardens',
  'hg.wonder.ephesos': 'The Temple of Artemis',
  'hg.wonder.gizah': 'The Pyramids',
  'hg.wonder.halikarnassos': 'The Mausoleum',
  'hg.wonder.olympia': 'The Statue of Zeus',
  'hg.wonder.rhodos': 'The Colossus',

  'hg.resource.wood': 'Wood',
  'hg.resource.clay': 'Clay',
  'hg.resource.stone': 'Stone',
  'hg.resource.ore': 'Ore',
  'hg.resource.glass': 'Glass',
  'hg.resource.papyrus': 'Papyrus',
  'hg.resource.cloth': 'Cloth',

  'hg.science.compass': 'Compass',
  'hg.science.gear': 'Gear',
  'hg.science.tablet': 'Tablet',

  'hg.card.glassworks-age-1': 'Glassworks',
  'hg.card.press-age-1': 'Press',
  'hg.card.loom-age-1': 'Loom',
  'hg.card.tavern': 'Tavern',
  'hg.card.west-trading-post': 'West Trading Post',
  'hg.card.east-trading-post': 'East Trading Post',
  'hg.card.marketplace': 'Marketplace',
  'hg.card.caravansery': 'Caravansery',
  'hg.card.forum': 'Forum',
  'hg.card.vineyard': 'Vineyard',
  'hg.card.bazar': 'Bazar',
  'hg.card.lighthouse': 'Lighthouse',
  'hg.card.haven': 'Haven',
  'hg.card.chamber-of-commerce': 'Chamber of Commerce',
  'hg.card.ludus': 'Ludus',
  'hg.card.arena': 'Arena',
  'hg.card.workers-guild': 'Workers Guild',
  'hg.card.craftsmens-guild': 'Craftsmens Guild',
  'hg.card.magistrates-guild': 'Magistrates Guild',
  'hg.card.traders-guild': 'Traders Guild',
  'hg.card.spies-guild': 'Spies Guild',
  'hg.card.philosophers-guild': 'Philosophers Guild',
  'hg.card.builders-guild': 'Builders Guild',
  'hg.card.shipowners-guild': 'Shipowners Guild',
  'hg.card.decorators-guild': 'Decorators Guild',
  'hg.card.scientists-guild': 'Scientists Guild'
} as const satisfies Messages;
