<script lang="ts">
  // Science scoring, as a worked example: the three symbols with their
  // names, then a city of two compasses, two gears and one tablet, scored in
  // its two parts. Same symbols: 4 + 4 + 1 = 9. Sets: 1 × 7 = 7. Total 16.
  // The arithmetic is the rule's, fixed, never game state. `compact` is the
  // tooltip size (the science counter): the city and its total only.
  import { ht } from '../i18n/messages.svelte';
  import DiagramFrame from './DiagramFrame.svelte';
  import ScienceGlyph, { type HgScienceSymbol } from './ScienceGlyph.svelte';

  interface Props {
    /** The small variant, for a tooltip bubble. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const SYMBOLS: readonly HgScienceSymbol[] = ['compass', 'gear', 'tablet'];

  /** The example city: how many of each symbol it holds. */
  const CITY: Record<HgScienceSymbol, number> = { compass: 2, gear: 2, tablet: 1 };

  const SET_POINTS = 7;
  const squares = SYMBOLS.map((symbol) => CITY[symbol] * CITY[symbol]);
  const same = squares.reduce((sum, n) => sum + n, 0);
  const sets = Math.min(...SYMBOLS.map((symbol) => CITY[symbol]));
  const total = same + sets * SET_POINTS;
</script>

{#snippet city()}
  <span
    class="flex flex-wrap items-center text-card-green-ink {compact
      ? 'gap-x-1.5 text-base'
      : 'gap-x-2.5 text-xl'}"
    aria-hidden="true"
  >
    {#each SYMBOLS as symbol (symbol)}
      <span class="flex items-center gap-0.5" data-science-count={CITY[symbol]}>
        {#each { length: CITY[symbol] }, i (i)}
          <ScienceGlyph {symbol} />
        {/each}
      </span>
    {/each}
  </span>
{/snippet}

<DiagramFrame name="science" caption={ht('hg.rules.diagram.science.caption')} {compact} example>
  {#if !compact}
    <span class="text-xs font-semibold text-fg-muted">{ht('hg.rules.diagram.science.symbols')}</span
    >
    <ul class="flex flex-wrap gap-x-4 gap-y-1.5">
      {#each SYMBOLS as symbol (symbol)}
        <li class="flex items-center gap-2 text-sm text-fg">
          <ScienceGlyph {symbol} class="text-xl text-card-green-ink" />
          {ht(`hg.science.${symbol}`)}
        </li>
      {/each}
    </ul>
    <div class="flex flex-col gap-1.5 border-t border-rule pt-2 text-sm text-fg">
      <span class="text-fg-muted">{ht('hg.rules.diagram.science.city')}</span>
      {@render city()}
      <span class="tabular-nums" data-science-part="same"
        >{ht('hg.rules.diagram.science.same', {
          a: squares[0]!,
          b: squares[1]!,
          c: squares[2]!,
          sum: same
        })}</span
      >
      <span class="tabular-nums" data-science-part="sets"
        >{ht('hg.rules.diagram.science.sets', {
          sets,
          each: SET_POINTS,
          points: sets * SET_POINTS
        })}</span
      >
      <strong class="font-semibold tabular-nums" data-science-part="total"
        >{ht('hg.rules.diagram.science.total', { count: total })}</strong
      >
    </div>
  {:else}
    {@render city()}
    <span class="text-[11px] leading-tight font-semibold text-fg tabular-nums"
      >{ht('hg.rules.diagram.science.total', { count: total })}</span
    >
  {/if}
</DiagramFrame>
