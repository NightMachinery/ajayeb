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
  import Icon from '$lib/icons/Icon.svelte';
  import { formatNumber, t } from '$lib/i18n';
  import type { Afford } from '$lib/view/afford';

  interface Props {
    afford: Afford;
    /** Glyph and number only (the mini tier). */
    compact?: boolean;
    /** Not your turn: show it at half strength. */
    dim?: boolean;
    class?: string;
  }

  let { afford, compact = false, dim = false, class: className = '' }: Props = $props();

  const STYLE = {
    chain: 'border-(color:--gold-deep) bg-gold text-card-ink',
    free: 'border-(color:--afford-ok-edge) bg-(--afford-ok) text-(--paper-bright)',
    own: 'border-(color:--afford-ok-edge) bg-(--afford-ok) text-(--paper-bright)',
    trade: 'border-(color:--afford-trade-edge) bg-saffron text-card-ink',
    short:
      'border-dashed border-(color:--afford-short) bg-(--afford-short-bg) text-(--afford-short)'
  } as const;
</script>

<span
  class="afford inline-flex h-[1.5em] items-center gap-[0.2em] rounded-full border-[1.5px] px-[0.4em]
    leading-none font-bold whitespace-nowrap tabular-nums shadow-sm {STYLE[
    afford.state
  ]} {className}"
  class:opacity-50={dim}
  data-afford={afford.state}
  data-min-font="10"
  aria-hidden="true"
>
  {#if afford.state === 'chain'}
    <Icon name="badge-chain" class="text-[1.05em] [--i2:var(--gold-deep)]" />
    {#if !compact}<span>{t('card.badge.free')}</span>{/if}
  {:else if afford.state === 'free'}
    <Icon name="badge-check" class="text-[1.05em] [--i2:var(--afford-ok)]" />
  {:else if afford.state === 'own'}
    <Icon name="coin" class="text-[1.05em] text-gold [--i2:var(--gold-deep)]" />
    <span>{formatNumber(afford.coins)}</span>
  {:else if afford.state === 'trade'}
    <Icon name="trade" class="text-[1.05em] [--i2:var(--card-ink)]" />
    <span>{formatNumber(afford.coins)}</span>
  {:else}
    <Icon name="badge-lock" class="text-[1.05em] [--i2:var(--afford-short-bg)]" />
    <span>{formatNumber(afford.coins)}</span>
  {/if}
</span>

<style>
  /* Badge colours. Fixed in both themes: badges sit on cards, which stay
     paper-coloured. The text pairs are at least 4.5:1. */
  .afford {
    --afford-ok: oklch(0.47 0.1 160);
    --afford-ok-edge: oklch(0.36 0.08 160);
    --afford-trade-edge: oklch(0.6 0.13 70);
    --afford-short: oklch(0.47 0.12 25);
    --afford-short-bg: oklch(0.93 0.015 40);
  }
</style>
