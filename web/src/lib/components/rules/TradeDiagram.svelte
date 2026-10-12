<script lang="ts">
  // The trade price rule as a worked example, for the rules reference: an
  // opponent who produces 3 clay makes each clay cost 2 + 3 = 5 coins. Built
  // from the card components (ResourceTile, Coin) with fixed props, so it
  // looks like the board; every label is translatable text.
  import { t } from '$lib/i18n';
  import Coin from '../card/Coin.svelte';
  import ResourceTile from '../card/ResourceTile.svelte';
</script>

<figure
  class="flex flex-col gap-2 rounded-panel border border-rule bg-panel-sunken p-3"
  data-rules-diagram="trade"
>
  <div class="flex items-center gap-3">
    <span dir="ltr" class="flex shrink-0 gap-1 text-xl" aria-hidden="true">
      <ResourceTile resource="clay" />
      <ResourceTile resource="clay" />
      <ResourceTile resource="clay" />
    </span>
    <span class="text-sm text-fg">{t('rules.diagram.trade.opponent')}</span>
  </div>
  <div class="flex items-center gap-3">
    <span dir="ltr" class="flex shrink-0 items-center gap-1 text-lg text-fg" aria-hidden="true">
      <Coin n={2} />
      <span class="font-semibold">+</span>
      <Coin n={3} />
      <span class="font-semibold">=</span>
      <Coin n={5} />
    </span>
    <span class="text-sm text-fg">{t('rules.diagram.trade.price')}</span>
  </div>
  <figcaption class="sr-only">{t('rules.diagram.trade.label')}</figcaption>
</figure>
