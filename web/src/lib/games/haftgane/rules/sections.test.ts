// Haftgane's rules reference sections (sections.ts): stable, unique ids for
// deep links, every message present and every name placeholder filled, and
// the whole reference rendered with its anchors.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { rules } from '../i18n/en/rules';
import { hgEn, ht, type HgKey } from '../i18n/messages.svelte';
import RulesContent from './RulesContent.svelte';
import {
  HG_BOARDS,
  HG_RULE_SECTION_IDS,
  HG_RULE_SECTIONS,
  hgLineText,
  hgRulesHref,
  hgSectionFromHash,
  isHgRuleSectionId,
  type HgBlock,
  type HgLine
} from './sections';

/** Every line of text a block shows, with the keys it uses. */
function blockKeys(block: HgBlock): { lines: HgLine[]; keys: HgKey[] } {
  switch (block.kind) {
    case 'p':
      return { lines: [block.line], keys: [block.line.key] };
    case 'h':
      return { lines: [], keys: [block.key] };
    case 'list':
    case 'steps':
      return { lines: [...block.items], keys: block.items.map((item) => item.key) };
    case 'terms':
      return {
        lines: block.items.map((item) => item.def),
        keys: block.items.flatMap((item) => [item.term, item.def.key])
      };
    case 'colours':
      return {
        lines: block.items.map((item) => ({ key: item.def })),
        keys: block.items.map((item) => item.def)
      };
    case 'wonder':
      return {
        lines: [...block.lines],
        keys: [
          'hg.rules.wonders.heading',
          `hg.board.${block.board}`,
          `hg.wonder.${block.board}`,
          ...block.lines.map((line) => line.key)
        ]
      };
    case 'diagram':
      return { lines: [], keys: [] };
  }
}

const lines = HG_RULE_SECTIONS.flatMap((section) =>
  section.blocks.flatMap((block) => blockKeys(block).lines)
);
const usedKeys = new Set<HgKey>(
  HG_RULE_SECTIONS.flatMap((section) => [
    section.title,
    ...section.blocks.flatMap((block) => blockKeys(block).keys),
    ...lines.flatMap((line) => Object.values(line.names ?? {}))
  ])
);

describe('Haftgane rule sections', () => {
  it('lists every section once, in the order of the ids', () => {
    expect(HG_RULE_SECTIONS.map((section) => section.id)).toEqual([...HG_RULE_SECTION_IDS]);
    expect(new Set(HG_RULE_SECTION_IDS).size).toBe(HG_RULE_SECTION_IDS.length);
  });

  it('covers what the brief asks for', () => {
    for (const id of [
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
    ] as const) {
      expect(HG_RULE_SECTION_IDS).toContain(id);
    }
  });

  it('keeps the ids stable: they are deep links', () => {
    expect(HG_RULE_SECTION_IDS).toEqual([
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
    ]);
    for (const id of HG_RULE_SECTION_IDS) expect(id).toMatch(/^[a-z][a-z-]*$/);
  });

  it('describes all seven boards', () => {
    const boards = HG_RULE_SECTIONS.flatMap((section) =>
      section.blocks.flatMap((block) => (block.kind === 'wonder' ? [block.board] : []))
    );
    expect(boards).toEqual([...HG_BOARDS]);
  });

  it('uses only messages that exist', () => {
    for (const key of usedKeys) expect(hgEn[key], key).toBeDefined();
  });

  it('fills every placeholder of every line, and names only real keys', () => {
    for (const line of lines) {
      const text = hgLineText(line);
      expect(text, line.key).not.toMatch(/[{}]/);
      for (const key of Object.values(line.names ?? {})) {
        expect(ht(key as 'hg.rules.toc'), key).not.toBe(key);
      }
    }
  });

  it('shows every message of the rules table somewhere', () => {
    // The page and panel chrome and the diagrams' own labels are used by
    // the components rather than by sections.ts.
    const chrome = /^hg\.rules\.(title|pageTitle|intro|toc|close|backHome|openPage|diagram\..+)$/;
    const unused = Object.keys(rules).filter(
      (key) => !usedKeys.has(key as HgKey) && !chrome.test(key)
    );
    expect(unused).toEqual([]);
  });
});

describe('deep links', () => {
  it('recognises section ids', () => {
    expect(isHgRuleSectionId('military')).toBe(true);
    expect(isHgRuleSectionId('turn-order')).toBe(true);
    expect(isHgRuleSectionId('tokens')).toBe(false);
    expect(isHgRuleSectionId('')).toBe(false);
    expect(isHgRuleSectionId(undefined)).toBe(false);
  });

  it('reads the section from a URL hash', () => {
    expect(hgSectionFromHash('#military')).toBe('military');
    expect(hgSectionFromHash('trade')).toBe('trade');
    expect(hgSectionFromHash('#unknown')).toBeUndefined();
    expect(hgSectionFromHash('')).toBeUndefined();
    expect(hgSectionFromHash(null)).toBeUndefined();
    expect(hgSectionFromHash('#%E0')).toBeUndefined();
  });

  it('builds the page path, at a section when given', () => {
    expect(hgRulesHref()).toBe('/rules/haftgane');
    expect(hgRulesHref('military')).toBe('/rules/haftgane#military');
    for (const id of HG_RULE_SECTION_IDS)
      expect(hgSectionFromHash(hgRulesHref(id).split('#')[1])).toBe(id);
  });
});

describe('RulesContent', () => {
  it('renders every section, with its anchor when asked, and the four diagrams', () => {
    const body = render(RulesContent, { props: { anchors: true } }).body;
    for (const id of HG_RULE_SECTION_IDS) {
      expect(body).toContain(`id="${id}"`);
      expect(body).toContain(`data-rules-section="${id}"`);
    }
    for (const name of ['trade', 'passing', 'conflicts', 'science'])
      expect(body).toContain(`data-rules-diagram="${name}"`);
    for (const board of HG_BOARDS) expect(body).toContain(`data-rules-wonder="${board}"`);
  });

  it('leaves the anchors off in the panel, so no id is repeated on a page', () => {
    const body = render(RulesContent, { props: { level: 3 } }).body;
    for (const id of HG_RULE_SECTION_IDS) expect(body).not.toContain(`id="${id}"`);
  });

  it('shows names from the name table, not placeholders', () => {
    const body = render(RulesContent).body;
    expect(body).toContain(ht('hg.card.scientists-guild'));
    expect(body).toContain(ht('hg.card.west-trading-post'));
    expect(body).toContain(`${ht('hg.board.rhodos')}: ${ht('hg.wonder.rhodos')}`);
    expect(body).not.toMatch(/\{[a-z]+\}/);
  });
});
