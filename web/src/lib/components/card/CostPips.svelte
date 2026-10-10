<script lang="ts">
  // A printed cost as a row of pips: coins first, then each resource once
  // with its count ("stone 2" rather than two stone pips), and the chain-in
  // chip when a chain symbol makes the card free. Groups that would not fit
  // the `room` (or more than `max` groups) collapse into "+n"; the sheet
  // always shows the whole cost. Printed data from the catalog, for display;
  // what a player actually pays is the server's cost breakdown (AffordBadge).
  import { formatNumber, t } from '$lib/i18n';
  import type { CatalogCost } from '$lib/catalog';
  import type { Chain, Resource } from '$lib/wire';
  import ChainChip from './ChainChip.svelte';
  import Coin from './Coin.svelte';
  import ResourceTile from './ResourceTile.svelte';
  import { costText, RESOURCES } from './effect-text';

  interface Props {
    cost: CatalogCost;
    chainIn?: Chain;
    /** The most pip groups to show before "+n" (the chain chip not counted). */
    max?: number;
    /** The width available, in em of this row's font size; groups that would
     *  not fit collapse into "+n". Unlimited when absent. */
    room?: number;
    /** Small-grid glyphs, for the mini tier. */
    small?: boolean;
    /** Tighter tiles and gaps, for cards in the structure (mini and medium). */
    dense?: boolean;
    class?: string;
  }

  let {
    cost,
    chainIn,
    max = 6,
    room = Infinity,
    small = false,
    dense = false,
    class: className = ''
  }: Props = $props();

  type Group = { kind: 'coins'; n: number } | { kind: 'res'; resource: Resource; n: number };

  const groups = $derived.by(() => {
    const all: Group[] = [];
    if (cost.coins > 0) all.push({ kind: 'coins', n: cost.coins });
    for (const r of RESOURCES) if (cost[r] > 0) all.push({ kind: 'res', resource: r, n: cost[r] });
    return all;
  });
  // Approximate widths in em (see Coin, ResourceTile and ChainChip), used
  // only to decide how many groups fit; the layout itself is flexbox.
  const tight = $derived(small || dense);
  const GAP = $derived(tight ? 0.12 : 0.2);
  const MORE = 1.2;
  const width = (g: Group) =>
    g.kind === 'coins' ? 1.25 : (small ? 1 : dense ? 1.05 : 1.2) + (g.n > 1 ? 0.7 : 0);

  const shown = $derived.by(() => {
    // ChainChip is 1.4 em on mini cards (at least 14 px) and 1.3 em elsewhere.
    const budget = room - (chainIn ? (small ? 1.4 : 1.3) + GAP : 0);
    const total = groups.reduce((sum, g) => sum + width(g) + GAP, -GAP);
    if (groups.length <= max && total <= budget) return groups;
    // Keep room for the "+n" marker.
    const kept: Group[] = [];
    let used = MORE;
    for (const g of groups) {
      if (kept.length >= max - 1 || used + GAP + width(g) > budget) break;
      kept.push(g);
      used += GAP + width(g);
    }
    return kept.length > 0 ? kept : groups.slice(0, 1);
  });
  const hidden = $derived(groups.length - shown.length);
  const label = $derived(t('card.cost.label', { items: costText(cost) }));
</script>

<span
  class="inline-flex min-w-0 items-center leading-none whitespace-nowrap {tight
    ? 'gap-[0.12em]'
    : 'gap-[0.2em]'} {className}"
  role="img"
  aria-label={label}
>
  {#each shown as g (g.kind === 'coins' ? 'coins' : g.resource)}
    {#if g.kind === 'coins'}
      <Coin n={g.n} />
    {:else}
      <span class="inline-flex items-center gap-[0.04em]">
        <ResourceTile
          resource={g.resource}
          {small}
          class={dense && !small ? 'size-[1.05em]!' : ''}
        />
        {#if g.n > 1}
          <span class="font-bold tabular-nums" data-min-font="10" aria-hidden="true"
            >{formatNumber(g.n)}</span
          >
        {/if}
      </span>
    {/if}
  {/each}
  {#if hidden > 0}
    <span class="font-bold text-card-ink" data-min-font="10" aria-hidden="true"
      >{t('card.cost.more', { n: hidden })}</span
    >
  {/if}
  {#if chainIn}
    <ChainChip chain={chainIn} direction="in" class="ms-auto" />
  {/if}
</span>
