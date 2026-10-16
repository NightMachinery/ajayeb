<script lang="ts">
  // The discard rule: discarding a card gives 2 coins plus 1 for each
  // yellow card in your city, with a worked example of two yellow cards.
  // Built from the coin and a yellow card swatch; labels are translatable.
  import { t } from '$lib/i18n';
  import Coin from '../card/Coin.svelte';
  import Arrow from './Arrow.svelte';
  import CardSwatch from './CardSwatch.svelte';
  import DiagramFrame from './DiagramFrame.svelte';

  interface Props {
    /** The tooltip size (about 220 by 120 px), without the caption. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();
</script>

<DiagramFrame name="discard" {compact} caption={t('rules.diagram.discard.label')}>
  <div class="flex flex-col gap-1.5 {compact ? 'text-sm' : 'text-base'}">
    <div class="flex flex-wrap items-center gap-1.5">
      <span class="text-sm font-semibold">{t('rules.diagram.discard.action')}</span>
      <Arrow class="text-fg-muted" />
      <span dir="ltr" class="flex items-center gap-1" aria-hidden="true">
        <Coin n={2} />
        <span class="font-semibold">+</span>
        <Coin n={1} />
        <span class="font-semibold">/</span>
        <CardSwatch color="yellow" />
      </span>
    </div>
    <div class="flex flex-wrap items-center gap-1.5 border-t border-rule pt-1.5">
      <span dir="ltr" class="flex items-center gap-1" aria-hidden="true">
        <CardSwatch color="yellow" />
        <CardSwatch color="yellow" />
        <Arrow class="mx-0.5 text-fg-muted" />
        <Coin n={2} />
        <span class="font-semibold">+</span>
        <Coin n={2} />
        <span class="font-semibold">=</span>
        <Coin n={4} />
      </span>
      <span class="text-xs text-fg-muted">{t('rules.diagram.discard.example')}</span>
    </div>
  </div>
</DiagramFrame>
