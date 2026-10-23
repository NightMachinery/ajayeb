// Haftgane's rules reference as data: its sections, in order, each with a
// stable id for deep links (/rules/haftgane#military,
// openHaftganeRules('military')) and a list of blocks that
// RulesContent.svelte renders. The text is in ../i18n/en/rules.ts; this file
// only arranges it, so it holds no prose. It mirrors Duel's
// (components/rules/sections.ts) but shares no code with it, so nothing of
// Duel's rules reference loads with Haftgane's, nor the reverse.
//
// A {placeholder} in a rules message names a board, a wonder or a card; the
// line's `names` maps each placeholder to its name key (../i18n/en/names.ts),
// so names are never written twice (sections.test.ts checks that every
// placeholder is filled).
//
// The ids are part of the public surface: tooltips and the game menu will
// link to them. Renaming one breaks those links, so add new ids rather than
// renaming.

import { t } from '$lib/i18n';
import { ht, type HgKey } from '../i18n/messages.svelte';

/** Every section of the rules reference, in reading order. */
export const HG_RULE_SECTION_IDS = [
  'goal',
  'terms',
  'resources',
  'setup',
  'age',
  'actions',
  'costs',
  'trade',
  'turn-order',
  'military',
  'science',
  'commerce',
  'guilds',
  'wonders',
  'scoring',
  'player-counts'
] as const;

export type HgRuleSectionId = (typeof HG_RULE_SECTION_IDS)[number];

/** A message key of Haftgane's rules table. */
export type HgRulesKey = Extract<HgKey, `hg.rules.${string}`>;

/** Board, wonder and card names filling a message's {placeholders}. */
export type HgNames = Readonly<Record<string, HgKey>>;

/** One line of text: a rules message and the names it needs. */
export interface HgLine {
  key: HgRulesKey;
  names?: HgNames;
}

/** The card colours, for the colour list of the goal section. */
export type HgCardColour = 'brown' | 'grey' | 'blue' | 'green' | 'yellow' | 'red' | 'purple';

/** The seven boards, by engine id. */
export const HG_BOARDS = [
  'alexandria',
  'babylon',
  'ephesos',
  'gizah',
  'halikarnassos',
  'olympia',
  'rhodos'
] as const;

export type HgBoardId = (typeof HG_BOARDS)[number];

export type HgDiagram = 'trade' | 'passing' | 'conflicts' | 'science';

export type HgBlock =
  | { kind: 'p'; line: HgLine }
  | { kind: 'h'; key: HgRulesKey }
  | { kind: 'list'; items: readonly HgLine[] }
  | { kind: 'steps'; items: readonly HgLine[] }
  /** A definition list: a name (or a term) and what it does. */
  | { kind: 'terms'; items: readonly { term: HgKey; def: HgLine }[] }
  | { kind: 'colours'; items: readonly { colour: HgCardColour; def: HgRulesKey }[] }
  /** One wonder board: its names as the heading, then its lines. */
  | { kind: 'wonder'; board: HgBoardId; lines: readonly HgLine[] }
  | { kind: 'diagram'; diagram: HgDiagram };

export interface HgRuleSection {
  id: HgRuleSectionId;
  title: HgRulesKey;
  blocks: readonly HgBlock[];
}

const p = (key: HgRulesKey, names?: HgNames): HgBlock => ({ kind: 'p', line: { key, names } });
const h = (key: HgRulesKey): HgBlock => ({ kind: 'h', key });
const list = (...keys: HgRulesKey[]): HgBlock => ({
  kind: 'list',
  items: keys.map((key) => ({ key }))
});
const steps = (...keys: HgRulesKey[]): HgBlock => ({
  kind: 'steps',
  items: keys.map((key) => ({ key }))
});
const diagram = (name: HgDiagram): HgBlock => ({ kind: 'diagram', diagram: name });

const TERMS = [
  'neighbours',
  'city',
  'age',
  'hand',
  'board',
  'stage',
  'treasure',
  'discard',
  'production',
  'choice',
  'chain',
  'strength',
  'token'
] as const;

/** The yellow cards, by engine id, with the key of what each does. */
const COMMERCE = [
  ['tavern', 'tavern'],
  ['west-trading-post', 'westTradingPost'],
  ['east-trading-post', 'eastTradingPost'],
  ['marketplace', 'marketplace'],
  ['caravansery', 'caravansery'],
  ['forum', 'forum'],
  ['vineyard', 'vineyard'],
  ['bazar', 'bazar'],
  ['lighthouse', 'lighthouse'],
  ['haven', 'haven'],
  ['chamber-of-commerce', 'chamberOfCommerce'],
  ['ludus', 'ludus'],
  ['arena', 'arena']
] as const;

const GUILDS = [
  'workers',
  'craftsmens',
  'magistrates',
  'traders',
  'spies',
  'philosophers',
  'builders',
  'shipowners',
  'decorators',
  'scientists'
] as const;

/** The lines each board shows under its names, after its starting resource. */
const WONDER_LINES: Record<HgBoardId, readonly HgRulesKey[]> = {
  alexandria: [
    'hg.rules.wonders.alexandria.day',
    'hg.rules.wonders.alexandria.night',
    'hg.rules.wonders.alexandria.note'
  ],
  babylon: [
    'hg.rules.wonders.babylon.day',
    'hg.rules.wonders.babylon.night',
    'hg.rules.wonders.babylon.note'
  ],
  ephesos: ['hg.rules.wonders.ephesos.day', 'hg.rules.wonders.ephesos.night'],
  gizah: ['hg.rules.wonders.gizah.day', 'hg.rules.wonders.gizah.night'],
  halikarnassos: [
    'hg.rules.wonders.halikarnassos.day',
    'hg.rules.wonders.halikarnassos.night',
    'hg.rules.wonders.halikarnassos.note'
  ],
  olympia: [
    'hg.rules.wonders.olympia.day',
    'hg.rules.wonders.olympia.night',
    'hg.rules.wonders.olympia.colour',
    'hg.rules.wonders.olympia.first',
    'hg.rules.wonders.olympia.last',
    'hg.rules.wonders.olympia.note'
  ],
  rhodos: ['hg.rules.wonders.rhodos.day', 'hg.rules.wonders.rhodos.night']
};

/** Each board's starting resource (docs/haftgane/rules.md, "Wonder boards"). */
const STARTING_RESOURCE: Record<HgBoardId, HgKey> = {
  alexandria: 'hg.resource.glass',
  babylon: 'hg.resource.wood',
  ephesos: 'hg.resource.papyrus',
  gizah: 'hg.resource.stone',
  halikarnassos: 'hg.resource.cloth',
  olympia: 'hg.resource.clay',
  rhodos: 'hg.resource.ore'
};

/** Haftgane's rules reference, section by section. */
export const HG_RULE_SECTIONS: readonly HgRuleSection[] = [
  {
    id: 'goal',
    title: 'hg.rules.goal.title',
    blocks: [
      p('hg.rules.goal.p1'),
      p('hg.rules.goal.p2'),
      h('hg.rules.goal.cardsTitle'),
      {
        kind: 'colours',
        items: (['brown', 'grey', 'blue', 'green', 'yellow', 'red', 'purple'] as const).map(
          (colour) => ({ colour, def: `hg.rules.goal.card.${colour}` as const })
        )
      }
    ]
  },
  {
    id: 'terms',
    title: 'hg.rules.terms.title',
    blocks: [
      {
        kind: 'terms',
        items: TERMS.map((term) => ({
          term: `hg.rules.terms.${term}.term` as const,
          def: { key: `hg.rules.terms.${term}` as const }
        }))
      }
    ]
  },
  {
    id: 'resources',
    title: 'hg.rules.resources.title',
    blocks: [
      p('hg.rules.resources.p1'),
      {
        kind: 'list',
        items: [
          { key: 'hg.rules.resources.raw' },
          { key: 'hg.rules.resources.manufactured', names: { loom: 'hg.card.loom-age-1' } }
        ]
      },
      p('hg.rules.resources.p2', { caravansery: 'hg.card.caravansery', forum: 'hg.card.forum' }),
      p('hg.rules.resources.p3')
    ]
  },
  {
    id: 'setup',
    title: 'hg.rules.setup.title',
    blocks: [
      steps(
        'hg.rules.setup.seats',
        'hg.rules.setup.decks',
        'hg.rules.setup.guilds',
        'hg.rules.setup.boards',
        'hg.rules.setup.coins'
      ),
      p('hg.rules.setup.p1')
    ]
  },
  {
    id: 'age',
    title: 'hg.rules.age.title',
    blocks: [
      p('hg.rules.age.p1'),
      steps('hg.rules.age.choose', 'hg.rules.age.play', 'hg.rules.age.pass'),
      p('hg.rules.age.direction'),
      diagram('passing'),
      h('hg.rules.age.sixthTitle'),
      p('hg.rules.age.sixth', { babylon: 'hg.board.babylon' }),
      h('hg.rules.age.endTitle'),
      p('hg.rules.age.end')
    ]
  },
  {
    id: 'actions',
    title: 'hg.rules.actions.title',
    blocks: [
      p('hg.rules.actions.p1'),
      {
        kind: 'terms',
        items: (['construct', 'stage', 'sell'] as const).map((action) => ({
          term: `hg.rules.actions.${action}.term` as const,
          def: { key: `hg.rules.actions.${action}` as const }
        }))
      },
      p('hg.rules.actions.sameName', {
        glassworks: 'hg.card.glassworks-age-1',
        press: 'hg.card.press-age-1',
        loom: 'hg.card.loom-age-1'
      })
    ]
  },
  {
    id: 'costs',
    title: 'hg.rules.costs.title',
    blocks: [
      p('hg.rules.costs.p1'),
      h('hg.rules.costs.coinsTitle'),
      p('hg.rules.costs.coins'),
      h('hg.rules.costs.productionTitle'),
      p('hg.rules.costs.production'),
      h('hg.rules.costs.chainTitle'),
      p('hg.rules.costs.chain'),
      p('hg.rules.costs.chainSkip'),
      h('hg.rules.costs.freeTitle'),
      p('hg.rules.costs.free', {
        halikarnassos: 'hg.board.halikarnassos',
        olympia: 'hg.board.olympia'
      })
    ]
  },
  {
    id: 'trade',
    title: 'hg.rules.trade.title',
    blocks: [
      p('hg.rules.trade.p1'),
      diagram('trade'),
      list(
        'hg.rules.trade.forSale',
        'hg.rules.trade.once',
        'hg.rules.trade.either',
        'hg.rules.trade.shared',
        'hg.rules.trade.start',
        'hg.rules.trade.now'
      ),
      h('hg.rules.trade.splitTitle'),
      p('hg.rules.trade.split'),
      h('hg.rules.trade.discountTitle'),
      {
        kind: 'terms',
        items: [
          { term: 'hg.card.west-trading-post', def: { key: 'hg.rules.trade.discount.west' } },
          { term: 'hg.card.east-trading-post', def: { key: 'hg.rules.trade.discount.east' } },
          {
            term: 'hg.card.marketplace',
            def: { key: 'hg.rules.trade.discount.marketplace' }
          }
        ]
      },
      p('hg.rules.trade.discount.p1')
    ]
  },
  {
    id: 'turn-order',
    title: 'hg.rules.order.title',
    blocks: [
      p('hg.rules.order.p1'),
      {
        kind: 'steps',
        items: [
          { key: 'hg.rules.order.choose' },
          { key: 'hg.rules.order.check' },
          { key: 'hg.rules.order.pay' },
          { key: 'hg.rules.order.place' },
          { key: 'hg.rules.order.gain' },
          { key: 'hg.rules.order.discard', names: { halikarnassos: 'hg.board.halikarnassos' } },
          { key: 'hg.rules.order.pass' }
        ]
      },
      p('hg.rules.order.public')
    ]
  },
  {
    id: 'military',
    title: 'hg.rules.military.title',
    blocks: [
      p('hg.rules.military.p1'),
      diagram('conflicts'),
      list('hg.rules.military.win', 'hg.rules.military.lose', 'hg.rules.military.tie'),
      p('hg.rules.military.p2')
    ]
  },
  {
    id: 'science',
    title: 'hg.rules.science.title',
    blocks: [
      p('hg.rules.science.p1'),
      list('hg.rules.science.same', 'hg.rules.science.sets'),
      diagram('science'),
      p('hg.rules.science.choice', {
        scientists: 'hg.card.scientists-guild',
        babylon: 'hg.board.babylon'
      })
    ]
  },
  {
    id: 'commerce',
    title: 'hg.rules.commerce.title',
    blocks: [
      p('hg.rules.commerce.p1'),
      {
        kind: 'terms',
        items: COMMERCE.map(([card, def]) => ({
          term: `hg.card.${card}` as const,
          def: { key: `hg.rules.commerce.${def}` as const }
        }))
      }
    ]
  },
  {
    id: 'guilds',
    title: 'hg.rules.guilds.title',
    blocks: [
      p('hg.rules.guilds.p1'),
      {
        kind: 'terms',
        items: GUILDS.map((guild) => ({
          term: `hg.card.${guild}-guild` as const,
          def: { key: `hg.rules.guilds.${guild}` as const }
        }))
      },
      p('hg.rules.guilds.p2')
    ]
  },
  {
    id: 'wonders',
    title: 'hg.rules.wonders.title',
    blocks: [
      p('hg.rules.wonders.p1'),
      p('hg.rules.wonders.p2'),
      ...HG_BOARDS.map((board): HgBlock => ({
        kind: 'wonder',
        board,
        lines: [
          { key: 'hg.rules.wonders.start', names: { resource: STARTING_RESOURCE[board] } },
          ...WONDER_LINES[board].map((key) => ({ key }))
        ]
      }))
    ]
  },
  {
    id: 'scoring',
    title: 'hg.rules.scoring.title',
    blocks: [
      p('hg.rules.scoring.p1'),
      steps(
        'hg.rules.scoring.wonder',
        'hg.rules.scoring.treasure',
        'hg.rules.scoring.military',
        'hg.rules.scoring.blue',
        'hg.rules.scoring.yellow',
        'hg.rules.scoring.green',
        'hg.rules.scoring.purple'
      ),
      p('hg.rules.scoring.tie')
    ]
  },
  {
    id: 'player-counts',
    title: 'hg.rules.players.title',
    blocks: [
      list(
        'hg.rules.players.three',
        'hg.rules.players.more',
        'hg.rules.players.seven',
        'hg.rules.players.nothingElse'
      )
    ]
  }
];

/** True for a string that names a section of Haftgane's rules. */
export function isHgRuleSectionId(value: string | null | undefined): value is HgRuleSectionId {
  return (HG_RULE_SECTION_IDS as readonly string[]).includes(value ?? '');
}

/** The section a URL hash names ('#military' or 'military'), if any. */
export function hgSectionFromHash(hash: string | null | undefined): HgRuleSectionId | undefined {
  let id: string;
  try {
    id = decodeURIComponent((hash ?? '').replace(/^#/, ''));
  } catch {
    return undefined; // A malformed escape such as '#%E0' names no section.
  }
  return isHgRuleSectionId(id) ? id : undefined;
}

/** The path of Haftgane's rules page, at a section when one is given. */
export function hgRulesHref(section?: HgRuleSectionId): `/rules/haftgane${string}` {
  return section ? `/rules/haftgane#${section}` : '/rules/haftgane';
}

/** ht() through a looser signature: the rules keys form a union that the
 *  compiler cannot narrow to "takes no params", so their placeholders are
 *  checked by sections.test.ts instead. */
const looseHt = ht as (key: HgKey, params?: Record<string, string>) => string;

/** A message with no placeholders to fill, such as a title or a name. */
export function hgPlainText(key: HgKey): string {
  return looseHt(key);
}

/** A rules message with its board, wonder and card names filled in. */
export function hgLineText(line: HgLine): string {
  const params: Record<string, string> = {};
  for (const [name, key] of Object.entries(line.names ?? {})) params[name] = looseHt(key);
  return looseHt(line.key, params);
}

/** A board's heading: the city and its wonder ("Rhodos: The Colossus"). */
export function hgWonderHeading(board: HgBoardId): string {
  return looseHt('hg.rules.wonders.heading', {
    board: looseHt(`hg.board.${board}`),
    wonder: looseHt(`hg.wonder.${board}`)
  });
}

/** The name of a card colour, from the shared tables. */
export function colourName(colour: HgCardColour): string {
  return t(`board.color.${colour}`);
}
