// Haftgane's concept diagrams: each renders in its full and its compact
// form, shows its labels from the message tables with every placeholder
// filled, keeps the caption out of sight when compact, and draws left and
// right as seats (dir="ltr") in every locale.
import type { Component } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { rules } from '../i18n/en/rules';
import { ht } from '../i18n/messages.svelte';
import ConflictDiagram from './ConflictDiagram.svelte';
import PassingDiagram from './PassingDiagram.svelte';
import ScienceDiagram from './ScienceDiagram.svelte';
import TradeDiagram from './TradeDiagram.svelte';

/** The diagrams' own sources, to find the labels they use. */
const SOURCES = Object.values(
  import.meta.glob('./*.svelte', { query: '?raw', import: 'default', eager: true })
) as string[];

const DIAGRAMS: [string, Component<{ compact?: boolean }>, string][] = [
  ['trade', TradeDiagram, ht('hg.resource.stone')],
  ['passing', PassingDiagram, ht('hg.rules.diagram.age', { age: 'II' })],
  ['conflicts', ConflictDiagram, ht('hg.rules.diagram.shields', { count: 4 })],
  ['science', ScienceDiagram, ht('hg.rules.diagram.science.total', { count: 16 })]
];

describe('Haftgane concept diagrams', () => {
  for (const [name, diagram, label] of DIAGRAMS) {
    it(`${name}: full and compact, labels from the tables`, () => {
      for (const compact of [false, true]) {
        const body = render(diagram, { props: { compact } }).body;
        expect(body).toContain(`data-rules-diagram="${name}"`);
        expect(body).toContain(label);
        expect(body).not.toMatch(/\{[a-z]+\}/);
        expect(body.includes('data-compact')).toBe(compact);
        expect(body).toMatch(
          compact ? /<figcaption class="sr-only"/ : /<figcaption class="text-xs/
        );
      }
    });
  }

  it('keeps left on the left: the seat rows are left to right in every locale', () => {
    for (const diagram of [TradeDiagram, PassingDiagram, ConflictDiagram]) {
      expect(render(diagram).body).toContain('dir="ltr"');
    }
  });

  it('names the neighbours in full, and in one word when compact', () => {
    for (const diagram of [TradeDiagram, PassingDiagram, ConflictDiagram]) {
      const full = render(diagram).body;
      const compact = render(diagram, { props: { compact: true } }).body;
      expect(full).toContain(ht('hg.rules.diagram.left'));
      expect(compact).not.toContain(ht('hg.rules.diagram.left'));
      expect(compact).toContain(ht('hg.rules.diagram.leftShort'));
    }
  });

  it('trade: coins go to both neighbours, resources come back', () => {
    const body = render(TradeDiagram).body;
    expect(body).toContain('data-trade="left"');
    expect(body).toContain('data-trade="right"');
    expect(body).toContain(ht('hg.resource.glass'));
  });

  it('passing: left in Ages I and III, right in Age II', () => {
    const body = render(PassingDiagram).body;
    expect(body.match(/data-pass="left"/g)?.length).toBe(4);
    expect(body.match(/data-pass="right"/g)?.length).toBe(2);
  });

  it('conflicts: a defeat on one side and a victory on the other', () => {
    const body = render(ConflictDiagram).body;
    expect(body).toContain('data-conflict="defeat"');
    expect(body).toContain('data-conflict="victory"');
    expect(body).toContain(ht('hg.rules.diagram.conflict.plus', { count: 3 }));
  });

  it('science: two compasses, two gears and one tablet make 9 + 7 = 16', () => {
    const body = render(ScienceDiagram).body;
    expect(body.match(/data-science="compass"/g)?.length).toBe(3);
    expect(body).toContain(ht('hg.rules.diagram.science.same', { a: 4, b: 4, c: 1, sum: 9 }));
    expect(body).toContain(ht('hg.rules.diagram.science.sets', { sets: 1, each: 7, points: 7 }));
  });

  it('uses every diagram label of the rules table', () => {
    // A key with a computed suffix (conflict.defeat, conflict.victory) is
    // found by its stem, followed by the start of a template expression.
    const expression = '$' + '{';
    const unused = Object.keys(rules)
      .filter((key) => key.startsWith('hg.rules.diagram.'))
      .filter((key) => !SOURCES.some((source) => source.includes(key)))
      .filter(
        (key) => !SOURCES.some((source) => source.includes(key.replace(/[^.]+$/, expression)))
      );
    expect(unused).toEqual([]);
  });
});
