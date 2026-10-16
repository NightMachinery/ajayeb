// The concept diagrams (ChainDiagram and the rest): each renders in its full
// and its compact form, shows its labels from the message tables with every
// placeholder filled, and keeps the caption out of sight when compact.
import type { Component } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { t } from '$lib/i18n';
import ChainDiagram from './ChainDiagram.svelte';
import DiscardDiagram from './DiscardDiagram.svelte';
import GuildDiagram from './GuildDiagram.svelte';
import SeventhWonderDiagram from './SeventhWonderDiagram.svelte';
import StarterDiagram from './StarterDiagram.svelte';

const DIAGRAMS: [string, Component<{ compact?: boolean }>, string][] = [
  ['chain', ChainDiagram, t('rules.diagram.chain.free')],
  ['discard', DiscardDiagram, t('rules.diagram.discard.example')],
  ['seventh', SeventhWonderDiagram, t('rules.diagram.seventh.out')],
  ['guild', GuildDiagram, t('rules.diagram.guild.counted')],
  ['starter', StarterDiagram, t('rules.diagram.starter.centre')]
];

describe('concept diagrams', () => {
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

  it('the seventh wonder shows one wonder leaving the game', () => {
    const body = render(SeventhWonderDiagram).body;
    expect(body.match(/data-wonder-out/g)?.length).toBe(1);
  });

  it('the starter names the Moon as the chooser when the pawn is on its side', () => {
    const body = render(StarterDiagram).body;
    expect(body).toContain(t('rules.diagram.starter.weaker', { moon: t('board.crest.moon') }));
  });
});
