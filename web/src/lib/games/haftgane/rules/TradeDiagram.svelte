<script lang="ts">
  // Buying from both neighbours, as a worked example: you buy a stone from
  // your left neighbour and a glass from your right one, and pay each of
  // them 2 coins. Coins go out along the upper arrows and the resources
  // come back along the lower ones. The row is dir="ltr" in every locale,
  // because left and right here are seats at the table (ruling 30), not a
  // reading direction. `compact` is the tooltip size (a neighbour's trade
  // chip), headed "For example:" so its numbers never read as the game's.
  // A tooltip has no room for five columns, so there the seats take one row
  // and each exchange sits on a second row, under the edge of its two seats.
  import { ht } from '../i18n/messages.svelte';
  import Arrow from './Arrow.svelte';
  import CoinDisc from './CoinDisc.svelte';
  import DiagramFrame from './DiagramFrame.svelte';
  import SeatBox from './SeatBox.svelte';

  interface Props {
    /** The small variant, for a tooltip bubble. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  /** The units bought in the example, on each side. */
  const SIDES = [
    { to: 'left', resource: 'stone' },
    { to: 'right', resource: 'glass' }
  ] as const;
</script>

{#snippet exchange(side: (typeof SIDES)[number])}
  <div
    class="flex flex-col items-center {compact ? 'gap-0.5 text-[11px]' : 'gap-1 text-xs'}"
    class:at-a={side.to === 'left'}
    class:at-b={side.to === 'right'}
    data-trade={side.to}
  >
    <span class="flex items-center gap-0.5 text-fg">
      {#if side.to === 'left'}<Arrow to="left" />{/if}
      <CoinDisc n={2} />
      {#if side.to === 'right'}<Arrow to="right" />{/if}
    </span>
    <span class="flex items-center gap-0.5 text-fg-muted">
      {#if side.to === 'right'}<Arrow to="left" />{/if}
      <span
        class="chip rounded-full px-1.5 font-semibold whitespace-nowrap"
        class:chip-stone={side.resource === 'stone'}
        class:chip-glass={side.resource === 'glass'}
        data-min-font="10">{ht(`hg.resource.${side.resource}`)}</span
      >
      {#if side.to === 'left'}<Arrow to="right" />{/if}
    </span>
  </div>
{/snippet}

<DiagramFrame name="trade" caption={ht('hg.rules.diagram.trade.caption')} {compact} example>
  <div
    dir="ltr"
    class="seats grid items-center {compact ? 'gap-x-1' : 'gap-x-1.5'}"
    class:compact
    aria-hidden="true"
  >
    <div class="at-l min-w-0">
      <SeatBox
        label={ht(compact ? 'hg.rules.diagram.leftShort' : 'hg.rules.diagram.left')}
        {compact}
      />
    </div>
    {@render exchange(SIDES[0])}
    <div class="at-y min-w-0">
      <SeatBox label={ht('hg.rules.diagram.you')} you {compact} />
    </div>
    {@render exchange(SIDES[1])}
    <div class="at-r min-w-0">
      <SeatBox
        label={ht(compact ? 'hg.rules.diagram.rightShort' : 'hg.rules.diagram.right')}
        {compact}
      />
    </div>
  </div>
</DiagramFrame>

<style>
  /* Layout that no Duel screen uses lives here, not in Tailwind classes:
     a class would add its rule to the global stylesheet that Home and
     Duel load, and this one loads only with Haftgane's rules. */
  .seats {
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr) auto minmax(0, 1fr);
    grid-template-areas: 'l a y b r';
  }
  .seats.compact {
    grid-template-columns: repeat(6, minmax(0, 1fr));
    grid-template-areas:
      'l l y y r r'
      '. a a b b .';
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
  .at-a {
    grid-area: a;
  }
  .at-b {
    grid-area: b;
  }
  /* The fill is the resource colour and the label the dark card ink in
     both themes (the res-*-ink tones are for text on the page, not on a
     fill). */
  .chip {
    line-height: 1.5;
  }
  .chip-stone {
    background: var(--res-stone);
    color: var(--card-ink);
  }
  .chip-glass {
    background: var(--res-glass);
    color: var(--card-ink);
  }
</style>
