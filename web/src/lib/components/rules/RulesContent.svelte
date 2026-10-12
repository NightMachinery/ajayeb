<script lang="ts">
  // The rules reference itself: every section of sections.ts, rendered from
  // its blocks. Shared by the /rules page and the in-game panel. A leaf with
  // plain props, so the gallery renders it standalone.
  //
  // Each section carries data-rules-section="<id>" for the panel's own
  // scrolling and the tests. With `anchors` it also carries id="<id>", the
  // deep-link target of /rules#<id>; the in-game panel leaves the ids off, so
  // a page that shows both never has two elements with one id.
  import { t } from '$lib/i18n';
  import HintGlyphLegend from '../hints/HintGlyphLegend.svelte';
  import MilitaryDiagram from './MilitaryDiagram.svelte';
  import ScienceDiagram from './ScienceDiagram.svelte';
  import TradeDiagram from './TradeDiagram.svelte';
  import {
    lineText,
    plainText,
    RULE_SECTIONS,
    type CardColour,
    type RuleSection
  } from './sections';

  interface Props {
    sections?: readonly RuleSection[];
    /** The heading level of a section title (sub-headings go one deeper). */
    level?: 2 | 3;
    /** Put id="<section id>" on each section, for /rules#<id>. */
    anchors?: boolean;
    /** Classes for each section, such as a scroll margin under a sticky header. */
    sectionClass?: string;
    class?: string;
  }

  let {
    sections = RULE_SECTIONS,
    level = 2,
    anchors = false,
    sectionClass = 'scroll-mt-4',
    class: className = ''
  }: Props = $props();

  const uid = $props.id();

  /** The colour band beside each kind of card (the theme's card fills). */
  const SWATCH: Record<CardColour, string> = {
    brown: 'bg-card-brown',
    grey: 'bg-card-grey',
    blue: 'bg-card-blue',
    green: 'bg-card-green',
    yellow: 'bg-card-yellow',
    red: 'bg-card-red',
    purple: 'bg-card-purple'
  };
</script>

<div class="rules-content flex flex-col gap-8 {className}">
  {#each sections as section (section.id)}
    <section
      id={anchors ? section.id : undefined}
      data-rules-section={section.id}
      aria-labelledby="{uid}-{section.id}"
      class="flex flex-col gap-3 {sectionClass}"
    >
      <svelte:element
        this={`h${level}`}
        id="{uid}-{section.id}"
        class="font-display text-xl leading-tight text-fg"
      >
        {plainText(section.title)}
      </svelte:element>

      {#each section.blocks as block, i (i)}
        {#if block.kind === 'p'}
          <p class="text-sm leading-relaxed text-fg">{lineText(block.line)}</p>
        {:else if block.kind === 'h'}
          <svelte:element
            this={`h${level + 1}`}
            class="mt-2 text-sm font-semibold tracking-wide text-fg-muted uppercase"
          >
            {plainText(block.key)}
          </svelte:element>
        {:else if block.kind === 'list'}
          <ul class="flex list-disc flex-col gap-1.5 ps-5 text-sm leading-relaxed text-fg">
            {#each block.items as item (item.key)}
              <li>{lineText(item)}</li>
            {/each}
          </ul>
        {:else if block.kind === 'steps'}
          <ol class="flex list-decimal flex-col gap-1.5 ps-5 text-sm leading-relaxed text-fg">
            {#each block.items as item (item.key)}
              <li>{lineText(item)}</li>
            {/each}
          </ol>
        {:else if block.kind === 'terms'}
          <dl class="flex flex-col divide-y divide-rule rounded-panel border border-rule bg-panel">
            {#each block.items as item (item.term)}
              <div class="flex flex-col gap-0.5 px-3 py-2">
                <dt class="text-sm font-semibold text-fg">{plainText(item.term)}</dt>
                <dd class="text-sm leading-snug text-fg-muted">{lineText(item.def)}</dd>
              </div>
            {/each}
          </dl>
        {:else if block.kind === 'colours'}
          <ul class="flex flex-col gap-2">
            {#each block.items as item (item.colour)}
              <li class="flex items-start gap-3">
                <span
                  class="mt-0.5 h-5 w-2 shrink-0 rounded-full {SWATCH[item.colour]}"
                  aria-hidden="true"
                ></span>
                <span class="flex min-w-0 flex-col">
                  <strong class="text-sm font-semibold text-fg">
                    {t(`board.color.${item.colour}`)}
                  </strong>
                  <span class="text-sm leading-snug text-fg-muted">{plainText(item.def)}</span>
                </span>
              </li>
            {/each}
          </ul>
        {:else if block.diagram === 'military'}
          <MilitaryDiagram />
        {:else if block.diagram === 'trade'}
          <TradeDiagram />
        {:else if block.diagram === 'hintGlyphs'}
          <HintGlyphLegend />
        {:else}
          <ScienceDiagram />
        {/if}
      {/each}
    </section>
  {/each}
</div>
