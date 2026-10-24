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
  import HgGlyph from './HgGlyph.svelte';
  import { effectText } from './effect-text';
  import { cardName } from './labels';

  interface Props {
    stages: readonly HgStage[];
    built: number;
    /** The cards under this wonder, when the viewer may know them. */
    under?: { cards: readonly string[]; own: boolean } | null;
  }

  let { stages, built, under = null }: Props = $props();

  const numeral = (i: number) =>
    ht(`hg.screen.stageNumeral.${Math.min(i + 1, 4)}` as 'hg.screen.stageNumeral.1');
</script>

<ol class="stages" data-city-stages>
  {#each stages as st, i (i)}
    {@const done = i < built}
    {@const label = ht('hg.city.stage.label', { stage: numeral(i) })}
    <li class="stage" class:done data-stage={i + 1} data-stage-built={done ? 'true' : 'false'}>
      <span class="numeral" role="img" aria-label={label} {@attach tooltip({ text: label })}
        >{numeral(i)}</span
      >
      <span class="text">
        <span class="effect">{st.effects.map(effectText).join(' ')}</span>
        <span class="state">{done ? ht('hg.city.stage.built') : ht('hg.city.stage.notBuilt')}</span>
      </span>
      {#if done}
        <span
          class="mark"
          role="img"
          aria-label={ht('hg.city.stage.built')}
          {@attach tooltip({ text: ht('hg.city.stage.built') })}
          ><HgGlyph name="badge-check" /></span
        >
      {:else}
        <span class="cost"><CostPips cost={st.cost} /></span>
      {/if}
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
  .under {
    margin: 0.4rem 0 0;
    color: var(--fg-muted);
    font-size: 0.85rem;
    line-height: 1.4;
  }
</style>
