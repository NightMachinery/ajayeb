// The rules reference as data: its sections, in order, each with a stable id
// for deep links (/rules#military, openRules('military')) and a list of
// blocks that RulesContent.svelte renders. The text itself is in
// $lib/i18n/en/rules.ts; this file only arranges it, so it holds no prose.
//
// A {placeholder} in a rules message names a card, wonder or token; the
// block's `names` maps each placeholder to the existing name key, so names
// are never written twice (sections.test.ts checks that every placeholder
// is filled).
//
// The ids are part of the public surface: tooltips and hints link to them.
// Renaming one breaks those links, so add new ids rather than renaming.

import { t, type MessageKey } from '$lib/i18n';

/** Every section of the rules reference, in reading order. */
export const RULE_SECTION_IDS = [
  'goal',
  'setup',
  'turn',
  'costs',
  'military',
  'science',
  'wonders',
  'tokens',
  'guilds',
  'age-end',
  'scoring',
  'hints',
  'glossary'
] as const;

export type RuleSectionId = (typeof RULE_SECTION_IDS)[number];

/** A message key of the rules table. */
export type RulesKey = Extract<MessageKey, `rules.${string}`>;

/** Card, wonder and token names filling a message's {placeholders}. */
export type Names = Readonly<Record<string, MessageKey>>;

/** One line of text: a rules message and the names it needs. */
export interface Line {
  key: RulesKey;
  names?: Names;
}

/** The card colours, for the colour list of the goal section. */
export type CardColour = 'brown' | 'grey' | 'blue' | 'green' | 'yellow' | 'red' | 'purple';

export type Diagram = 'military' | 'trade' | 'science' | 'hintGlyphs';

export type Block =
  | { kind: 'p'; line: Line }
  | { kind: 'h'; key: RulesKey }
  | { kind: 'list'; items: readonly Line[] }
  | { kind: 'steps'; items: readonly Line[] }
  /** A definition list: a name (or a glossary term) and what it does. */
  | { kind: 'terms'; items: readonly { term: MessageKey; def: Line }[] }
  | { kind: 'colours'; items: readonly { colour: CardColour; def: RulesKey }[] }
  | { kind: 'diagram'; diagram: Diagram };

export interface RuleSection {
  id: RuleSectionId;
  title: RulesKey;
  blocks: readonly Block[];
}

const p = (key: RulesKey, names?: Names): Block => ({ kind: 'p', line: { key, names } });
const h = (key: RulesKey): Block => ({ kind: 'h', key });
const list = (...keys: RulesKey[]): Block => ({
  kind: 'list',
  items: keys.map((key) => ({ key }))
});
const steps = (...keys: RulesKey[]): Block => ({
  kind: 'steps',
  items: keys.map((key) => ({ key }))
});
const diagram = (name: Diagram): Block => ({ kind: 'diagram', diagram: name });

const WONDERS = [
  'pyramids',
  'colossus',
  'great-lighthouse',
  'temple-of-artemis',
  'mausoleum',
  'great-library',
  'piraeus',
  'hanging-gardens',
  'statue-of-zeus',
  'sphinx',
  'appian-way',
  'circus-maximus'
] as const;

const TOKENS = [
  'agriculture',
  'architecture',
  'economy',
  'law',
  'masonry',
  'mathematics',
  'philosophy',
  'strategy',
  'theology',
  'urbanism'
] as const;

const GUILDS = [
  'merchants',
  'shipowners',
  'builders',
  'magistrates',
  'scientists',
  'moneylenders',
  'tacticians'
] as const;

const GLOSSARY = [
  'accessible',
  'covered',
  'structure',
  'city',
  'discard',
  'tucked',
  'price',
  'chain',
  'pair',
  'supremacy',
  'weaker',
  'extra',
  'looting',
  'box'
] as const;

const GLOSSARY_NAMES: Partial<Record<(typeof GLOSSARY)[number], Names>> = {
  extra: { theology: 'token.theology' },
  box: { library: 'wonder.name.great-library' }
};

/** The rules reference, section by section. */
export const RULE_SECTIONS: readonly RuleSection[] = [
  {
    id: 'goal',
    title: 'rules.goal.title',
    blocks: [
      p('rules.goal.p1'),
      h('rules.goal.winTitle'),
      list('rules.goal.win.military', 'rules.goal.win.science', 'rules.goal.win.civilian'),
      h('rules.goal.cardsTitle'),
      {
        kind: 'colours',
        items: (['brown', 'grey', 'blue', 'green', 'yellow', 'red', 'purple'] as const).map(
          (colour) => ({ colour, def: `rules.goal.card.${colour}` as const })
        )
      }
    ]
  },
  {
    id: 'setup',
    title: 'rules.setup.title',
    blocks: [
      p('rules.setup.p1'),
      p('rules.setup.p2'),
      h('rules.setup.draftTitle'),
      p('rules.setup.draftIntro'),
      steps('rules.setup.draft1', 'rules.setup.draft2'),
      h('rules.setup.agesTitle'),
      p('rules.setup.ages1'),
      p('rules.setup.ages2')
    ]
  },
  {
    id: 'turn',
    title: 'rules.turn.title',
    blocks: [
      p('rules.turn.p1'),
      steps('rules.turn.build', 'rules.turn.discard', 'rules.turn.wonder'),
      p('rules.turn.p2'),
      p('rules.turn.p3')
    ]
  },
  {
    id: 'costs',
    title: 'rules.costs.title',
    blocks: [
      p('rules.costs.p1'),
      h('rules.costs.tradeTitle'),
      p('rules.costs.trade'),
      diagram('trade'),
      {
        kind: 'list',
        items: [
          { key: 'rules.costs.trade.only' },
          { key: 'rules.costs.trade.bank', names: { economy: 'token.economy' } },
          { key: 'rules.costs.trade.free' }
        ]
      },
      h('rules.costs.fixedTitle'),
      p('rules.costs.fixed'),
      {
        kind: 'terms',
        items: [
          { term: 'card.name.stone-reserve', def: { key: 'rules.costs.fixed.stone' } },
          { term: 'card.name.clay-reserve', def: { key: 'rules.costs.fixed.clay' } },
          { term: 'card.name.wood-reserve', def: { key: 'rules.costs.fixed.wood' } },
          { term: 'card.name.customs-house', def: { key: 'rules.costs.fixed.customs' } }
        ]
      },
      h('rules.costs.choiceTitle'),
      p('rules.costs.choice', {
        forum: 'card.name.forum',
        piraeus: 'wonder.name.piraeus',
        caravansery: 'card.name.caravansery',
        lighthouse: 'wonder.name.great-lighthouse'
      }),
      h('rules.costs.chainTitle'),
      p('rules.costs.chain'),
      h('rules.costs.payTitle'),
      p('rules.costs.pay')
    ]
  },
  {
    id: 'military',
    title: 'rules.military.title',
    blocks: [
      p('rules.military.p1', { strategy: 'token.strategy' }),
      diagram('military'),
      h('rules.military.zonesTitle'),
      list(
        'rules.military.zone2',
        'rules.military.zone5',
        'rules.military.zone10',
        'rules.military.zoneCapital'
      ),
      p('rules.military.points'),
      h('rules.military.lootTitle'),
      p('rules.military.loot')
    ]
  },
  {
    id: 'science',
    title: 'rules.science.title',
    blocks: [
      p('rules.science.p1', { law: 'token.law' }),
      diagram('science'),
      h('rules.science.pairTitle'),
      p('rules.science.pair'),
      h('rules.science.winTitle'),
      p('rules.science.win', { law: 'token.law' })
    ]
  },
  {
    id: 'wonders',
    title: 'rules.wonders.title',
    blocks: [
      p('rules.wonders.p1'),
      h('rules.wonders.seventhTitle'),
      p('rules.wonders.seventh'),
      h('rules.wonders.againTitle'),
      p('rules.wonders.again', {
        artemis: 'wonder.name.temple-of-artemis',
        piraeus: 'wonder.name.piraeus',
        gardens: 'wonder.name.hanging-gardens',
        sphinx: 'wonder.name.sphinx',
        appian: 'wonder.name.appian-way'
      }),
      h('rules.wonders.listTitle'),
      {
        kind: 'terms',
        items: WONDERS.map((slug) => ({
          term: `wonder.name.${slug}` as const,
          def: { key: `rules.wonders.${slug}` as const }
        }))
      },
      p('rules.wonders.choices')
    ]
  },
  {
    id: 'tokens',
    title: 'rules.tokens.title',
    blocks: [
      p('rules.tokens.p1', { library: 'wonder.name.great-library' }),
      {
        kind: 'terms',
        items: TOKENS.map((slug) => ({
          term: `token.${slug}` as const,
          def: { key: `rules.tokens.${slug}` as const }
        }))
      },
      p('rules.tokens.discounts', {
        architecture: 'token.architecture',
        masonry: 'token.masonry'
      })
    ]
  },
  {
    id: 'guilds',
    title: 'rules.guilds.title',
    blocks: [
      p('rules.guilds.p1'),
      {
        kind: 'terms',
        items: GUILDS.map((guild) => ({
          term: `card.name.${guild}-guild` as const,
          def: { key: `rules.guilds.${guild}` as const }
        }))
      }
    ]
  },
  {
    id: 'age-end',
    title: 'rules.ageEnd.title',
    blocks: [p('rules.ageEnd.p1'), p('rules.ageEnd.p2'), p('rules.ageEnd.p3')]
  },
  {
    id: 'scoring',
    title: 'rules.scoring.title',
    blocks: [
      p('rules.scoring.p1'),
      steps(
        'rules.scoring.blue',
        'rules.scoring.green',
        'rules.scoring.yellow',
        'rules.scoring.guilds',
        'rules.scoring.wonders',
        'rules.scoring.tokens',
        'rules.scoring.coins',
        'rules.scoring.military'
      ),
      p('rules.scoring.tie'),
      p('rules.scoring.preview')
    ]
  },
  {
    // The board's hint marks (components/hints/HintGlyphLegend.svelte);
    // its words live with the other hint messages (i18n/en/hints.ts).
    id: 'hints',
    title: 'rules.hints.title',
    blocks: [p('rules.hints.intro'), diagram('hintGlyphs')]
  },
  {
    id: 'glossary',
    title: 'rules.glossary.title',
    blocks: [
      {
        kind: 'terms',
        items: GLOSSARY.map((term) => ({
          term: `rules.glossary.${term}.term` as const,
          def: { key: `rules.glossary.${term}` as const, names: GLOSSARY_NAMES[term] }
        }))
      }
    ]
  }
];

/** True for a string that names a section of the rules. */
export function isRuleSectionId(value: string | null | undefined): value is RuleSectionId {
  return (RULE_SECTION_IDS as readonly string[]).includes(value ?? '');
}

/** The section a URL hash names ('#military' or 'military'), if any. */
export function sectionFromHash(hash: string | null | undefined): RuleSectionId | undefined {
  let id: string;
  try {
    id = decodeURIComponent((hash ?? '').replace(/^#/, ''));
  } catch {
    return undefined; // A malformed escape such as '#%E0' names no section.
  }
  return isRuleSectionId(id) ? id : undefined;
}

/** The path of the rules page, at a section when one is given. */
export function rulesHref(section?: RuleSectionId): `/rules${string}` {
  return section ? `/rules#${section}` : '/rules';
}

/** t() through a looser signature: the rules keys form a union that the
 *  compiler cannot narrow to "takes no params", so their placeholders are
 *  checked by sections.test.ts instead. */
const looseT = t as (key: MessageKey, params?: Record<string, string>) => string;

/** A message with no placeholders to fill, such as a title or a name. */
export function plainText(key: MessageKey): string {
  return looseT(key);
}

/** A rules message with its card, wonder and token names filled in. */
export function lineText(line: Line): string {
  const params: Record<string, string> = {};
  for (const [name, key] of Object.entries(line.names ?? {})) params[name] = looseT(key);
  return looseT(line.key, params);
}
