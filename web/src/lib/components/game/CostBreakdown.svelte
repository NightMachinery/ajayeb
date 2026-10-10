<script lang="ts">
  // A cost breakdown, line by line, exactly as the server worked it out
  // (CostView): a chain that makes the build free, each unit the seat makes
  // itself, each unit a flexible producer covers, units a token waives, each
  // resource bought with its unit price, the printed coin cost, and the
  // total. The only sum shown here is "coins left": the seat's coins minus
  // the server's total, for reading, never for deciding anything.
  import { cardName, resourceName, wonderName } from '$lib/components/card/effect-text';
  import { formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { CostView, Resource } from '$lib/wire';
  import ResourceTile from '../card/ResourceTile.svelte';
  import { RESOURCES } from '../card/effect-text';
  import { tokenName } from './names';

  interface Props {
    cost: CostView;
    /** The paying seat's coins now, for the "left after" line. */
    coins?: number;
    class?: string;
  }

  let { cost, coins, class: className = '' }: Props = $props();

  interface Line {
    id: string;
    resource?: Resource;
    text: string;
    /** Coins this line costs, shown at the end. */
    amount?: number;
    ok?: boolean;
  }

  const lines = $derived.by(() => {
    const out: Line[] = [];
    if (cost.chainFree) {
      out.push({
        id: 'chain',
        text: cost.chainFrom
          ? t('game.cost.chain', { card: cardName(cost.chainFrom) })
          : t('game.cost.chainNoCard'),
        ok: true
      });
      return out;
    }
    for (const r of RESOURCES) {
      const n = cost.own[r];
      if (n > 0) {
        out.push({
          id: `own-${r}`,
          resource: r,
          text: t('game.cost.own', { n, resource: resourceName(r) }),
          ok: true
        });
      }
    }
    cost.producers.forEach((p, i) => {
      const source = p.card ? cardName(p.card) : p.wonder ? wonderName(p.wonder) : '';
      out.push({
        id: `producer-${i}`,
        resource: p.resource,
        text: t('game.cost.producer', { resource: resourceName(p.resource), source }),
        ok: true
      });
    });
    if (cost.discount) {
      const discount = cost.discount;
      for (const r of RESOURCES) {
        const n = discount.units[r];
        if (n > 0) {
          out.push({
            id: `discount-${r}`,
            resource: r,
            text: t('game.cost.discount', {
              n,
              resource: resourceName(r),
              token: tokenName(discount.token)
            }),
            ok: true
          });
        }
      }
    }
    for (const tr of cost.trade) {
      out.push({
        id: `trade-${tr.resource}`,
        resource: tr.resource,
        text: t(tr.fixed ? 'game.cost.tradeFixed' : 'game.cost.trade', {
          n: tr.units,
          resource: resourceName(tr.resource),
          price: tr.unitPrice
        }),
        amount: tr.coins
      });
    }
    if (cost.printedCoins > 0) {
      out.push({ id: 'printed', text: t('game.cost.printed'), amount: cost.printedCoins });
    }
    return out;
  });
</script>

<div class="flex flex-col gap-1 text-sm text-fg {className}" data-cost-breakdown>
  {#if lines.length === 0}
    <p class="text-fg-muted">{t('game.cost.nothing')}</p>
  {:else}
    <ul class="flex flex-col gap-1">
      {#each lines as line (line.id)}
        <li class="flex min-h-6 items-center gap-2">
          {#if line.resource}
            <ResourceTile resource={line.resource} class="text-[16px]" />
          {:else if line.id === 'chain'}
            <Icon name="badge-chain" class="text-[16px] text-gild" />
          {:else}
            <Icon name="coin" class="text-[16px] text-gild" />
          {/if}
          <span class="min-w-0 flex-1">{line.text}</span>
          {#if line.amount !== undefined}
            <span class="font-semibold tabular-nums" data-must-fit
              >{t('coins.count', { count: line.amount })}</span
            >
          {:else if line.ok}
            <Icon name="badge-check" class="text-[15px] text-malachite" />
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
  {#if cost.urbanismBonus > 0}
    <p class="text-fg-muted">{t('game.cost.urbanism', { count: cost.urbanismBonus })}</p>
  {/if}
  {#if cost.toOpponent > 0}
    <p class="text-fg-muted">{t('game.cost.toOpponent', { count: cost.toOpponent })}</p>
  {/if}
  <div class="mt-1 flex items-baseline justify-between gap-2 border-t border-rule pt-1.5">
    <span class="font-semibold">{t('game.cost.total')}</span>
    <span class="font-semibold tabular-nums" data-must-fit data-cost-total={cost.total}
      >{t('coins.count', { count: cost.total })}</span
    >
  </div>
  {#if coins !== undefined}
    <p class="text-fg-muted {cost.affordable ? '' : 'font-semibold text-cinnabar'}">
      {#if cost.affordable}
        {t('game.cost.left', {
          have: formatNumber(coins),
          after: formatNumber(coins - cost.total)
        })}
      {:else}
        {t('game.cost.short', { have: formatNumber(coins), count: cost.shortfall })}
      {/if}
    </p>
  {/if}
</div>
