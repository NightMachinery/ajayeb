// The rules reference's sections (sections.ts): stable, unique ids for deep
// links, every message present and every name placeholder filled, and the
// whole reference rendered with its anchors.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { en, t, type MessageKey } from '$lib/i18n';
import { rules } from '$lib/i18n/en/rules';
import RulesContent from './RulesContent.svelte';
import {
  isRuleSectionId,
  lineText,
  RULE_SECTION_IDS,
  RULE_SECTIONS,
  rulesHref,
  sectionFromHash,
  type Block,
  type Line
} from './sections';

/** Every line of text a block shows, with the keys it uses. */
function blockKeys(block: Block): { lines: Line[]; keys: MessageKey[] } {
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
        keys: block.items.flatMap((item) => [`board.color.${item.colour}` as const, item.def])
      };
    case 'diagram':
      return { lines: [], keys: [] };
  }
}

const lines = RULE_SECTIONS.flatMap((section) =>
  section.blocks.flatMap((block) => blockKeys(block).lines)
);
const usedKeys = new Set<MessageKey>(
  RULE_SECTIONS.flatMap((section) => [
    section.title,
    ...section.blocks.flatMap((block) => blockKeys(block).keys),
    ...section.blocks.flatMap((block) =>
      blockKeys(block).lines.flatMap((line) => Object.values(line.names ?? {}))
    )
  ])
);

describe('rule sections', () => {
  it('lists every section once, in the order of the ids', () => {
    expect(RULE_SECTIONS.map((section) => section.id)).toEqual([...RULE_SECTION_IDS]);
    expect(new Set(RULE_SECTION_IDS).size).toBe(RULE_SECTION_IDS.length);
  });

  it('covers the sections the rules reference promises', () => {
    for (const id of ['goal', 'setup', 'turn', 'costs', 'military', 'science'] as const) {
      expect(RULE_SECTION_IDS).toContain(id);
    }
    for (const id of ['wonders', 'tokens', 'guilds', 'age-end', 'scoring', 'glossary'] as const) {
      expect(RULE_SECTION_IDS).toContain(id);
    }
  });

  it('uses ids that work as URL fragments and element ids', () => {
    for (const id of RULE_SECTION_IDS) expect(id).toMatch(/^[a-z][a-z-]*$/);
  });

  it('uses only messages that exist', () => {
    for (const key of usedKeys) expect(en[key], key).toBeDefined();
  });

  it('fills every placeholder of every line, and names only real keys', () => {
    for (const line of lines) {
      const text = lineText(line);
      expect(text, line.key).not.toMatch(/[{}]/);
      for (const key of Object.values(line.names ?? {})) {
        expect(t(key as 'app.name'), key).not.toBe(key);
      }
    }
  });

  it('shows every message of the rules table somewhere', () => {
    // The page and panel chrome, and the diagrams' own labels, are used by
    // the components rather than by sections.ts.
    const chrome =
      /^rules\.(title|pageTitle|intro|toc|close|backHome|homeLink|menu|openPage|diagram\..+)$/;
    const unused = Object.keys(rules).filter(
      (key) => !usedKeys.has(key as MessageKey) && !chrome.test(key)
    );
    expect(unused).toEqual([]);
  });

  it('writes no em or en dash in the rules text', () => {
    for (const text of Object.values(rules)) {
      const forms = typeof text === 'string' ? [text] : Object.values(text);
      for (const form of forms) expect(form).not.toMatch(/[–—]/);
    }
  });
});

describe('deep links', () => {
  it('recognises section ids', () => {
    expect(isRuleSectionId('military')).toBe(true);
    expect(isRuleSectionId('age-end')).toBe(true);
    expect(isRuleSectionId('nope')).toBe(false);
    expect(isRuleSectionId('')).toBe(false);
    expect(isRuleSectionId(undefined)).toBe(false);
  });

  it('reads the section from a URL hash', () => {
    expect(sectionFromHash('#military')).toBe('military');
    expect(sectionFromHash('science')).toBe('science');
    expect(sectionFromHash('#unknown')).toBeUndefined();
    expect(sectionFromHash('')).toBeUndefined();
    expect(sectionFromHash(null)).toBeUndefined();
    expect(sectionFromHash('#%E0')).toBeUndefined();
  });

  it('builds the page path, at a section when given', () => {
    expect(rulesHref()).toBe('/rules');
    expect(rulesHref('military')).toBe('/rules#military');
    for (const id of RULE_SECTION_IDS)
      expect(sectionFromHash(rulesHref(id).split('#')[1])).toBe(id);
  });
});

describe('RulesContent', () => {
  it('renders every section, with its anchor when asked', () => {
    const body = render(RulesContent, { props: { anchors: true } }).body;
    for (const id of RULE_SECTION_IDS) {
      expect(body).toContain(`id="${id}"`);
      expect(body).toContain(`data-rules-section="${id}"`);
    }
    expect(body).toContain('data-rules-diagram="military"');
    expect(body).toContain('data-rules-diagram="trade"');
    expect(body).toContain('data-rules-diagram="science"');
    for (const name of ['chain', 'discard', 'seventh', 'guild', 'starter'])
      expect(body).toContain(`data-rules-diagram="${name}"`);
  });

  it('leaves the anchors off in the panel, so no id is repeated on a page', () => {
    const body = render(RulesContent, { props: { level: 3 } }).body;
    for (const id of RULE_SECTION_IDS) expect(body).not.toContain(`id="${id}"`);
  });

  it('shows names from the name tables, not placeholders', () => {
    const body = render(RulesContent).body;
    expect(body).toContain(t('wonder.name.temple-of-artemis'));
    expect(body).toContain(t('token.law'));
    expect(body).toContain(t('card.name.merchants-guild'));
    expect(body).not.toMatch(/\{[a-z]+\}/);
  });
});
