<script lang="ts">
  // One unit a neighbour sells the viewer, exactly as the view states it:
  // the resource (or the choice of resources) on one symbol, how many such
  // units, and the price per unit, green when a discount applies, with the
  // discount's source in the tooltip. The client prices nothing.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { TradeUnit } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import ResourceTile from './ResourceTile.svelte';

  interface Props {
    unit: TradeUnit;
    /** The selling neighbour's display name. */
    name: string;
  }

  let { unit, name }: Props = $props();

  const first = $derived(unit.options[0]);
  const resName = (r: string) => ht(`hg.resource.${r}` as 'hg.resource.wood');
  const discounted = $derived(unit.options.some((o) => o.discount));
  const text = $derived.by(() => {
    if (!first) return '';
    const price = first.price;
    if (unit.choice) {
      const names = unit.options.map((o) => resName(o.resource));
      const options = names.reduce((a, b) => ht('hg.screen.trade.or', { a, b }));
      return ht('hg.screen.trade.choice', { name, options, price });
    }
    const resource = resName(first.resource);
    return unit.count > 1
      ? ht('hg.screen.trade.many', { name, count: unit.count, resource, price })
      : ht('hg.screen.trade.one', { name, resource, price });
  });
  const lines = $derived.by(() => {
    const d = unit.options.find((o) => o.discount)?.discount;
    if (!d) return [];
    return [
      d.design
        ? ht('hg.screen.trade.discountCard', {
            card: ht(`hg.card.${d.design}` as 'hg.card.marketplace')
          })
        : ht('hg.screen.trade.discountStage', { stage: d.stage })
    ];
  });
</script>

{#if first}
  <span
    class="chip"
    data-trade={unit.options.map((o) => o.resource).join('/')}
    data-price={first.price}
    role="img"
    aria-label={[text, ...lines].join(' ')}
    {@attach tooltip({ text, lines })}
  >
    <span class="res">
      {#each unit.options as o, i (o.resource)}
        {#if i > 0}<span class="slash" aria-hidden="true">/</span>{/if}
        <ResourceTile res={o.resource} />
      {/each}
    </span>
    {#if unit.count > 1}<span class="count" dir="ltr">×{formatNumber(unit.count)}</span>{/if}
    <span class="price" class:discounted>{formatNumber(first.price)}</span>
  </span>
{/if}

<style>
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    height: 1.625rem;
    padding-inline: 0.2rem;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--panel);
    font-size: 0.8125rem;
    font-weight: 700;
    line-height: 1;
    white-space: nowrap;
  }
  .res {
    display: inline-flex;
    align-items: center;
    gap: 0.05rem;
  }
  .slash {
    color: var(--fg-muted);
    font-weight: 400;
  }
  .count {
    font-variant-numeric: tabular-nums;
  }
  .price {
    display: inline-grid;
    place-items: center;
    min-width: 1.15rem;
    height: 1.15rem;
    border: 1px solid var(--gold-deep);
    border-radius: 9999px;
    background: var(--gold);
    color: var(--card-ink);
    font-size: 0.75rem;
  }
  .discounted {
    border-color: var(--afford-ok-edge);
    background: var(--afford-ok);
    color: var(--paper-bright);
  }
</style>
