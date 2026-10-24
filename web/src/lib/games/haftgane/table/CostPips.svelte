<script lang="ts">
  // A printed cost from the catalog: a coin pip with its count, then one
  // tile per resource with its count. Each pip names what it is in its
  // tooltip. What the viewer would pay is the server's (the badge and the
  // pick list); this is only what the card prints.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { HgCost } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import ResourceTile from './ResourceTile.svelte';

  interface Props {
    cost: HgCost;
  }

  let { cost }: Props = $props();

  const groups = $derived.by(() => {
    const out: { res: string; n: number }[] = [];
    for (const r of cost.res ?? []) {
      const last = out.at(-1);
      if (last?.res === r) last.n += 1;
      else out.push({ res: r, n: 1 });
    }
    return out;
  });
</script>

<span class="pips" data-cost>
  {#if cost.coins}
    {@const tip = ht('hg.screen.cost.coins', { count: cost.coins })}
    <span class="coin" role="img" aria-label={tip} {@attach tooltip({ text: tip })}
      >{formatNumber(cost.coins)}</span
    >
  {/if}
  {#each groups as g (g.res)}
    {@const tip = ht('hg.screen.cost.res', {
      count: g.n,
      resource: ht(`hg.resource.${g.res}` as 'hg.resource.wood')
    })}
    <span class="res" role="img" aria-label={tip} {@attach tooltip({ text: tip })}>
      <ResourceTile res={g.res} size="xs" />
      {#if g.n > 1}<span>{formatNumber(g.n)}</span>{/if}
    </span>
  {/each}
</span>

<style>
  .pips {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.1em 0.2em;
    font-weight: 700;
    line-height: 1;
  }
  .coin {
    display: inline-grid;
    place-items: center;
    min-width: 1.15em;
    height: 1.15em;
    border: 1px solid var(--gold-deep);
    border-radius: 9999px;
    background: var(--gold);
    color: var(--card-ink);
    font-size: 0.85em;
  }
  .res {
    display: inline-flex;
    align-items: center;
    gap: 0.05em;
  }
  .res :global([data-res]) {
    width: 1.1em;
    height: 1.1em;
    font-size: 0.8em;
  }
</style>
