<script lang="ts">
  // A city's cards counted by colour, each with its category glyph so that
  // colour is never the only cue. `hideZero` drops empty colours where room
  // is short (the neighbour panels); elsewhere an empty colour is dimmed.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { colourCounts } from './model';

  interface Props {
    built: readonly string[];
    hideZero?: boolean;
  }

  let { built, hideZero = false }: Props = $props();
  const counts = $derived(colourCounts(built).filter((c) => !hideZero || c.count > 0));
</script>

<span class="counts" data-colour-counts>
  {#each counts as c (c.colour)}
    {@const tip = `${ht(`hg.screen.colourName.${c.colour}`)}: ${ht(`hg.screen.colour.${c.colour}`, { count: c.count })}`}
    <span
      class="count"
      class:zero={c.count === 0}
      style:--ink="var(--card-{c.colour})"
      data-colour={c.colour}
      role="img"
      aria-label={tip}
      {@attach tooltip({ text: tip })}
    >
      <HgGlyph name="category-{c.colour}-sm" />
      <span>{formatNumber(c.count)}</span>
    </span>
  {/each}
</span>

<style>
  .counts {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.15rem 0.4rem;
  }
  .count {
    display: inline-flex;
    align-items: center;
    gap: 0.1rem;
    font-size: 0.8125rem;
    font-weight: 700;
    line-height: 1.25rem;
    font-variant-numeric: tabular-nums;
  }
  .count :global(svg) {
    color: var(--ink);
    font-size: 0.85rem;
  }
  .zero {
    color: var(--fg-muted);
    opacity: 0.6;
  }
</style>
