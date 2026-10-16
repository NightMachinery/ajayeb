<script lang="ts">
  // The trade price rule as a worked example, for the rules reference: an
  // opponent who produces 3 clay makes each clay cost 2 + 3 = 5 coins. Built
  // from the card components (ResourceTile, Coin) with fixed props, so it
  // looks like the board; every label is translatable text. `compact` is
  // the tooltip size (the trade badge's bubble): about 220 by 100 px, headed
  // "For example:" so its numbers never read as the game's own.
  import { t } from '$lib/i18n';
  import Coin from '../card/Coin.svelte';
  import ResourceTile from '../card/ResourceTile.svelte';

  interface Props {
    /** The small variant, for a tooltip bubble. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();
</script>

<figure
  class="flex flex-col rounded-panel border border-rule bg-panel-sunken {compact
    ? 'max-w-[13.5rem] gap-1 p-1.5'
    : 'gap-2 p-3'}"
  data-rules-diagram="trade"
  data-compact={compact ? '' : undefined}
>
  {#if compact}
    <span class="text-[11px] leading-tight font-semibold text-fg-muted"
      >{t('card.tip.example')}</span
    >
  {/if}
  <div class="flex items-center {compact ? 'gap-2' : 'gap-3'}">
    <span
      dir="ltr"
      class="flex shrink-0 {compact ? 'gap-0.5 text-sm' : 'gap-1 text-xl'}"
      aria-hidden="true"
    >
      <ResourceTile resource="clay" />
      <ResourceTile resource="clay" />
      <ResourceTile resource="clay" />
    </span>
    <span class="{compact ? 'text-[11px] leading-tight' : 'text-sm'} text-fg"
      >{t('rules.diagram.trade.opponent')}</span
    >
  </div>
  <div class="flex items-center {compact ? 'gap-2' : 'gap-3'}">
    <span
      dir="ltr"
      class="flex shrink-0 items-center gap-1 text-fg {compact ? 'text-xs' : 'text-lg'}"
      aria-hidden="true"
    >
      <Coin n={2} />
      <span class="font-semibold">+</span>
      <Coin n={3} />
      <span class="font-semibold">=</span>
      <Coin n={5} />
    </span>
    <span class="{compact ? 'text-[11px] leading-tight' : 'text-sm'} text-fg"
      >{t('rules.diagram.trade.price')}</span
    >
  </div>
  <figcaption class="sr-only">{t('rules.diagram.trade.label')}</figcaption>
</figure>
