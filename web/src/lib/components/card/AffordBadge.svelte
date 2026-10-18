<script lang="ts">
  // The affordability badge on a takeable card: chain, free, own, trade or
  // short (view/afford.ts maps the server's cost breakdown to these). Each
  // state has its own glyph as well as its own colour, so the badges stay
  // apart in greyscale and for colour-blind players:
  //
  //   chain  gold pill, chain glyph ("Free" beside it when there is room)
  //   free   malachite pill, check
  //   own    malachite pill, coin and the coins to pay
  //   trade  saffron pill, trade arrows and the coins to pay
  //   short  muted dashed pill, lock and the coins it would cost
  //
  // When it is not your turn the badge is dimmed but still shown, for
  // planning. Decorative: the card's ARIA name says the same in words.
  //
  // Two modifiers of the prices hints (M10.HS2), each with its own tooltip:
  //   bonus  a chain build pays Urbanism's coins: "+4" beside the chain
  //          glyph (only the glyph on the mini tier; the tooltip says it)
  //   flow   part of the trade goes to the opponent (Economy): a small
  //          outlined notch on the badge's end edge
  //
  // The badge's tooltip says why (view/tips.ts affordTip): the chain and
  // the card that gives it, free, the city's own production, what a trade
  // buys at what price (with the trade diagram of the rules, compact), or
  // how many coins it is short; the chain, trade and short states link the
  // rules' costs section. The recap (price-changed-tick, M10.F3) adds a line
  // when the card's price changed since the viewer's last move ("Was 4, now
  // 5: Moon built Clay Pit.", `priceTip`, data-recap-price).
  import Icon from '$lib/icons/Icon.svelte';
  import { formatNumber, i18n, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Afford } from '$lib/view/afford';
  import { affordTip } from '$lib/view/tips';
  import type { CardTipContext } from '$lib/view/tips';
  import { chainTooltip } from '$lib/ui/chain-links';
  import type { Chain, CostView } from '$lib/wire';
  import TradeDiagram from '../rules/TradeDiagram.svelte';

  interface Props {
    afford: Afford;
    /** Glyph and number only (the mini tier). */
    compact?: boolean;
    /** Not your turn: show it at half strength. */
    dim?: boolean;
    /** Urbanism's coins for this chain build (CostView.urbanismBonus), or 0. */
    bonus?: number;
    /** Coins of this payment that go to the opponent (CostView.toOpponent), or 0. */
    flow?: number;
    /** The opponent who receives `flow` (user text). */
    flowTo?: string;
    /** How the price changed since the viewer's last move (the recap). */
    priceTip?: string;
    /** The server's cost breakdown behind the badge, for its tooltip. */
    cost?: CostView;
    chain?: Chain;
    chainContext?: CardTipContext;
    class?: string;
  }

  let {
    afford,
    compact = false,
    dim = false,
    bonus = 0,
    flow = 0,
    flowTo = '',
    priceTip,
    cost,
    chain,
    chainContext,
    class: className = ''
  }: Props = $props();

  const tip = $derived.by(() => {
    const words = affordTip(afford, cost);
    const linked =
      afford.state === 'chain' && chain ? chainTooltip(chain, 'in', [], chainContext) : undefined;
    return {
      ...linked,
      ...words,
      lines: [...(linked?.lines ?? words.lines ?? []), ...(priceTip ? [priceTip] : [])],
      diagram: afford.state === 'trade' ? tradeDiagram : undefined
    };
  });

  const STYLE = {
    chain: 'border-(color:--gold-deep) bg-gold text-card-ink',
    free: 'border-(color:--afford-ok-edge) bg-(--afford-ok) text-(--paper-bright)',
    own: 'border-(color:--afford-ok-edge) bg-(--afford-ok) text-(--paper-bright)',
    trade: 'border-(color:--afford-trade-edge) bg-saffron text-card-ink',
    short:
      'border-dashed border-(color:--afford-short) bg-(--afford-short-bg) text-(--afford-short)'
  } as const;
</script>

{#snippet tradeDiagram()}
  <TradeDiagram compact />
{/snippet}

<span
  class="afford pointer-events-auto relative inline-flex h-[1.5em] items-center gap-[0.2em] rounded-full border-[1.5px] px-[0.4em]
    leading-none font-bold whitespace-nowrap tabular-nums shadow-sm {STYLE[
    afford.state
  ]} {className}"
  class:opacity-50={dim}
  data-afford={afford.state}
  data-recap-price={priceTip ? '' : undefined}
  {@attach tooltip(tip)}
  data-min-font="10"
  aria-hidden="true"
>
  {#if afford.state === 'chain'}
    <Icon name="badge-chain" class="text-[1.05em] [--i2:var(--gold-deep)]" />
    {#if bonus > 0}
      <span
        class="pointer-events-auto"
        data-afford-bonus={bonus}
        {@attach tooltip({ text: t('hints.badge.urbanismTip', { count: bonus }) })}
        >{compact ? '' : t('hints.badge.bonus', { count: bonus })}</span
      >
    {:else if !compact}<span dir={i18n.dir}>{t('card.badge.free')}</span>{/if}
  {:else if afford.state === 'free'}
    <Icon name="badge-check" class="text-[1.05em] [--i2:var(--afford-ok)]" />
  {:else if afford.state === 'own'}
    <Icon name="coin" class="text-[1.05em] text-gold [--i2:var(--gold-deep)]" />
    <span>{formatNumber(afford.coins)}</span>
  {:else if afford.state === 'trade'}
    <Icon name="trade" class="text-[1.05em] [--i2:var(--card-ink)]" />
    <span>{formatNumber(afford.coins)}</span>
    {#if flow > 0}
      <span
        class="flow pointer-events-auto absolute -end-[3px] top-1/2 h-[0.7em] w-[5px] -translate-y-1/2 rounded-[2px]"
        data-afford-flow={flow}
        {@attach tooltip({ text: t('hints.badge.flowTip', { count: flow, name: flowTo }) })}
      ></span>
    {/if}
  {:else}
    <Icon name="badge-lock" class="text-[1.05em] [--i2:var(--afford-short-bg)]" />
    <span>{formatNumber(afford.coins)}</span>
  {/if}
</span>

<style>
  /* Badge colours: the --afford-* tokens (styles/tokens.css), fixed in
     both themes because badges sit on cards, which stay paper-coloured. */
  /* The Economy notch: an outlined tab on the end edge. */
  .flow {
    background: var(--card-paper);
    border: 1.5px solid var(--card-ink);
  }
</style>
