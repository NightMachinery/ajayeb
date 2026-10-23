<script lang="ts">
  // The conflicts at the end of an Age, as a worked example: with 4 shields
  // in Age II you are weaker than your left neighbour (6) and stronger than
  // your right one (2), so you take a defeat token from one conflict and the
  // Age II victory token from the other. Each result sits between the two
  // seats it settles. The row is dir="ltr" in every locale, because left and
  // right here are seats at the table (ruling 30). `compact` is the tooltip
  // size (a neighbour's military outlook), headed "For example:". A tooltip
  // has no room for five columns, so there the seats take one row and each
  // result sits on a second row, under the edge of the two seats it settles.
  import { i18n, t } from '$lib/i18n';
  import { ht } from '../i18n/messages.svelte';
  import DiagramFrame from './DiagramFrame.svelte';
  import SeatBox from './SeatBox.svelte';

  interface Props {
    /** The small variant, for a tooltip bubble. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const SHIELDS = { left: 6, you: 4, right: 2 } as const;
</script>

{#snippet shields(count: number)}
  <span
    class="font-semibold text-fg tabular-nums {compact ? '' : 'whitespace-nowrap'}"
    data-shields={count}>{ht('hg.rules.diagram.shields', { count })}</span
  >
{/snippet}

{#snippet result(outcome: 'defeat' | 'victory')}
  <span
    class="flex flex-col items-center gap-0.5 leading-tight {compact ? 'text-[11px]' : 'text-xs'}"
    class:at-d={outcome === 'defeat'}
    class:at-v={outcome === 'victory'}
    class:result-row={compact}
    data-conflict={outcome}
  >
    <span dir={i18n.dir} class="text-fg-muted">{ht(`hg.rules.diagram.conflict.${outcome}`)}</span>
    {#if outcome === 'defeat'}
      <span
        class="rounded-[0.45em] border border-cinnabar bg-cinnabar/15 px-1 font-bold text-cinnabar-ink tabular-nums"
        >{ht('hg.rules.diagram.conflict.minus', { count: 1 })}</span
      >
    {:else}
      <span
        class="rounded-[0.45em] border border-vp-edge bg-vp px-1 font-bold text-(--paper-bright) tabular-nums"
        >{ht('hg.rules.diagram.conflict.plus', { count: 3 })}</span
      >
    {/if}
  </span>
{/snippet}

<DiagramFrame name="conflicts" caption={ht('hg.rules.diagram.conflict.caption')} {compact} example>
  <span
    dir={i18n.dir}
    class="font-semibold text-fg-muted {compact ? 'text-[11px]' : 'text-xs'}"
    data-min-font={compact ? '10' : undefined}
    aria-hidden="true">{ht('hg.rules.diagram.age', { age: t('card.ageNumeral.2') })}</span
  >
  <div
    dir="ltr"
    class="seats grid items-center {compact ? 'gap-x-1 text-[11px]' : 'gap-x-1.5 text-sm'}"
    class:compact
    aria-hidden="true"
  >
    <div class="at-l min-w-0">
      <SeatBox
        label={ht(compact ? 'hg.rules.diagram.leftShort' : 'hg.rules.diagram.left')}
        {compact}
      >
        {@render shields(SHIELDS.left)}
      </SeatBox>
    </div>
    {@render result('defeat')}
    <div class="at-y min-w-0">
      <SeatBox label={ht('hg.rules.diagram.you')} you {compact}>
        {@render shields(SHIELDS.you)}
      </SeatBox>
    </div>
    {@render result('victory')}
    <div class="at-r min-w-0">
      <SeatBox
        label={ht(compact ? 'hg.rules.diagram.rightShort' : 'hg.rules.diagram.right')}
        {compact}
      >
        {@render shields(SHIELDS.right)}
      </SeatBox>
    </div>
  </div>
</DiagramFrame>

<style>
  /* Layout that no Duel screen uses lives here, not in Tailwind classes:
     a class would add its rule to the global stylesheet that Home and
     Duel load, and this one loads only with Haftgane's rules. */
  .seats {
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr) auto minmax(0, 1fr);
    grid-template-areas: 'l d y v r';
  }
  .seats.compact {
    grid-template-columns: repeat(6, minmax(0, 1fr));
    grid-template-areas:
      'l l y y r r'
      '. d d v v .';
    row-gap: 0.25rem;
  }
  .at-l {
    grid-area: l;
  }
  .at-y {
    grid-area: y;
  }
  .at-r {
    grid-area: r;
  }
  .at-d {
    grid-area: d;
  }
  .at-v {
    grid-area: v;
  }
  .result-row {
    flex-direction: row;
    justify-content: center;
    gap: 0.25rem;
  }
</style>
