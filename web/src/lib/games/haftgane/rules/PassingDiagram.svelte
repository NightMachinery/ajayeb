<script lang="ts">
  // Which way the hands go in each Age: the three seats once, as column
  // heads, then one row per Age with the arrows between them. In Ages I and
  // III every hand moves to the left (you pass to your left neighbour and
  // receive from your right one); in Age II, to the right. The grid is
  // dir="ltr" in every locale, because left and right here are seats at
  // the table (ruling 30). `compact` is the tooltip size (the pass cue).
  import { i18n, t } from '$lib/i18n';
  import { ht } from '../i18n/messages.svelte';
  import Arrow from './Arrow.svelte';
  import DiagramFrame from './DiagramFrame.svelte';

  interface Props {
    /** The small variant, for a tooltip bubble. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const AGES = [
    { age: 1, to: 'left' },
    { age: 2, to: 'right' },
    { age: 3, to: 'left' }
  ] as const;

  const SEATS = [
    { label: 'hg.rules.diagram.left', short: 'hg.rules.diagram.leftShort', you: false },
    { label: 'hg.rules.diagram.you', short: 'hg.rules.diagram.you', you: true },
    { label: 'hg.rules.diagram.right', short: 'hg.rules.diagram.rightShort', you: false }
  ] as const;
</script>

{#snippet hand()}
  <!-- A hand of cards: three small card backs in a fan. -->
  <span class="hand" aria-hidden="true">
    <span class="card tilt-start"></span>
    <span class="card"></span>
    <span class="card tilt-end"></span>
  </span>
{/snippet}

<DiagramFrame name="passing" caption={ht('hg.rules.diagram.passing.caption')} {compact}>
  <div
    dir="ltr"
    class="rows grid items-center gap-y-1 text-center {compact
      ? 'gap-x-1 text-[11px]'
      : 'gap-x-2 text-xs'}"
    aria-hidden="true"
  >
    <span></span>
    {#each SEATS as seat, i (seat.label)}
      {#if i > 0}<span></span>{/if}
      <span
        dir={i18n.dir}
        class="label leading-tight font-semibold {seat.you ? 'text-fg' : 'text-fg-muted'}"
        data-min-font="10">{ht(compact ? seat.short : seat.label)}</span
      >
    {/each}

    {#each AGES as row (row.age)}
      <span
        dir={i18n.dir}
        class="pe-1 text-start font-semibold whitespace-nowrap text-fg-muted"
        data-passing-age={row.age}
        >{ht('hg.rules.diagram.age', { age: t(`card.ageNumeral.${row.age}`) })}</span
      >
      {#each SEATS as seat, i (seat.label)}
        {#if i > 0}
          <span class="flex justify-center text-fg" data-pass={row.to}>
            <Arrow to={row.to} class={compact ? '' : 'text-base'} />
          </span>
        {/if}
        <span class="flex justify-center {compact ? 'text-sm' : 'text-lg'}">{@render hand()}</span>
      {/each}
    {/each}
  </div>
</DiagramFrame>

<style>
  /* Layout that no Duel screen uses lives here, not in Tailwind classes:
     a class would add its rule to the global stylesheet that Home and
     Duel load, and this one loads only with Haftgane's rules. */
  .rows {
    grid-template-columns: auto minmax(0, 1fr) auto minmax(0, 1fr) auto minmax(0, 1fr);
  }
  .label {
    overflow-wrap: break-word;
    hyphens: auto;
  }
  .hand {
    position: relative;
    display: inline-block;
    width: 1.3em;
    height: 1.1em;
  }
  .hand .tilt-start {
    rotate: -12deg;
  }
  .hand .tilt-end {
    rotate: 12deg;
  }
  .hand .card {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0.3em;
    width: 0.7em;
    border-radius: 0.12em;
    border: 1px solid var(--rule-strong);
    background: var(--panel);
    transform-origin: 50% 100%;
  }
</style>
