<script lang="ts">
  // A city's wonder stages in the city sheet: each stage's numeral, what it
  // does and either "built with a hidden card" or its printed cost. The view
  // lists the cards under a wonder only as a set (the viewer's own, or every
  // city's once the end reveals them), never which stage holds which, so
  // they follow the stages as one line.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { HgStage } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import CostPips from './CostPips.svelte';
  import EffectGlyph from './EffectGlyph.svelte';
  import { effectText } from './effect-text';
  import { cardName } from './labels';

  interface Props {
    stages: readonly HgStage[];
    built: number;
    compact?: boolean;
    /** The cards under this wonder, when the viewer may know them. */
    under?: { cards: readonly string[]; own: boolean } | null;
  }

  let { stages, built, under = null, compact = false }: Props = $props();

  const numeral = (i: number) =>
    ht(`hg.screen.stageNumeral.${Math.min(i + 1, 4)}` as 'hg.screen.stageNumeral.1');
</script>

<ol class="stages" class:compact style:--stages={stages.length} dir="ltr" data-city-stages>
  {#each stages as st, i (i)}
    {@const done = i < built}
    {@const label = ht('hg.city.stage.label', { stage: numeral(i) })}
    <li class="stage" class:done data-stage={i + 1} data-stage-built={done ? 'true' : 'false'}>
      <span class="numeral" role="img" aria-label={label} {@attach tooltip({ text: label })}
        >{numeral(i)}</span
      >
      <span class="text">
        <span class="effect">
          {#if compact}{#each st.effects as fx, j (j)}<EffectGlyph
                effect={fx}
              />{/each}{:else}{st.effects.map(effectText).join(' ')}{/if}
        </span>
        <span class="state">{done ? ht('hg.city.stage.built') : ht('hg.city.stage.notBuilt')}</span>
      </span>
      {#if done}
        <span
          class="mark"
          role="img"
          aria-label={ht('hg.city.stage.built')}
          {@attach tooltip({ text: ht('hg.city.stage.built') })}
          ><span class="tuck" aria-hidden="true"></span></span
        >
      {/if}
      {#if !done || compact}<span class="cost"><CostPips cost={st.cost} /></span>{/if}
    </li>
  {/each}
</ol>
{#if under && under.cards.length > 0}
  <p class="under" data-under={under.own ? 'own' : 'revealed'}>
    {ht(under.own ? 'hg.city.under.own' : 'hg.city.under.revealed', {
      cards: under.cards.map(cardName).join(ht('hg.city.row.sep'))
    })}
  </p>
{/if}

<style>
  .stages {
    display: grid;
    gap: 0.4rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .stage {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    min-width: 0;
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
  }
  .done {
    border-color: var(--gold-deep);
    background: color-mix(in oklch, var(--gold) 14%, var(--panel));
  }
  .numeral {
    flex: 0 0 1.6rem;
    color: var(--gold-deep);
    font: 700 1.1rem var(--type-display);
    text-align: center;
  }
  .text {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
  }
  .effect {
    font-weight: 600;
    line-height: 1.3;
    overflow-wrap: anywhere;
  }
  .state {
    color: var(--fg-muted);
    font-size: 0.85rem;
  }
  .mark {
    flex-shrink: 0;
    color: var(--fg);
    font-size: 1.1rem;
  }
  .cost {
    flex-shrink: 0;
  }
  .tuck {
    display: block;
    width: 1.2rem;
    height: 0.65rem;
    border: 2px solid var(--rule-strong);
    border-radius: 0.15rem;
    background: var(--panel-sunken);
  }
  .compact {
    grid-template-columns: repeat(var(--stages), minmax(0, 1fr));
    gap: 0;
    border-block-start: 1px solid var(--rule-strong);
  }
  .compact .stage {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    align-content: start;
    justify-items: center;
    gap: 0.4rem;
    padding: 0.5rem 0.2rem;
    border: 0;
    border-inline-end: 1px solid var(--rule);
    border-radius: 0;
  }
  .compact .stage:last-child {
    border-inline-end: 0;
  }
  .compact .text {
    align-items: center;
    text-align: center;
  }
  .compact .effect {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 0.2rem;
  }
  .compact .state {
    font-size: 0.7rem;
  }
  .compact .cost {
    font-size: 0.9rem;
  }
  .under {
    margin: 0.4rem 0 0;
    color: var(--fg-muted);
    font-size: 0.85rem;
    line-height: 1.4;
  }
</style>
