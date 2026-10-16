<script lang="ts">
  // The chain rule as a worked example: the Stable shows the horseshoe on
  // its band, and the Horse Breeders, which shows it beside its cost, is
  // then free. Real cards and the real chain chip; every label is
  // translatable text.
  import { t } from '$lib/i18n';
  import CardFace from '../card/CardFace.svelte';
  import ChainChip from '../card/ChainChip.svelte';
  import Arrow from './Arrow.svelte';
  import DiagramFrame from './DiagramFrame.svelte';

  interface Props {
    /** The tooltip size (about 220 by 120 px), without the caption. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const width = $derived(compact ? 46 : 64);
</script>

<DiagramFrame
  name="chain"
  {compact}
  caption={t('rules.diagram.chain.label', {
    stable: t('card.name.stable'),
    breeders: t('card.name.horse-breeders')
  })}
>
  <div dir="ltr" class="flex items-center justify-center gap-2">
    <div class="flex flex-col items-center gap-1">
      <CardFace slug="stable" tier="mini" {width} />
      <span class="text-[11px] leading-tight text-fg-muted">{t('rules.diagram.chain.earlier')}</span
      >
    </div>
    <div class="flex flex-col items-center gap-1 text-fg-muted">
      <ChainChip chain="horseshoe" direction="out" class="text-sm" />
      <Arrow class="text-lg" />
    </div>
    <div class="flex flex-col items-center gap-1">
      <CardFace slug="horse-breeders" tier="mini" {width} />
      <span
        class="rounded-full bg-malachite px-1.5 text-[11px] leading-tight font-semibold text-(--paper-bright)"
      >
        {t('rules.diagram.chain.free')}
      </span>
    </div>
  </div>
</DiagramFrame>
