<script lang="ts">
  // A card's or a wonder's printed effects, in one of two forms:
  //   mini  the glyph row of the effect band ("wood wood", "laurel 3" ...),
  //         with the chain-out chip at its end;
  //   full  one line per effect: the glyph and its sentence from the i18n
  //         templates, then the chain line.
  // Printed data from the catalog, for display only.
  import type { CardSlug, CatalogEffect } from '$lib/catalog';
  import { t } from '$lib/i18n';
  import type { Chain } from '$lib/wire';
  import ChainChip from './ChainChip.svelte';
  import EffectGlyph from './EffectGlyph.svelte';
  import { cardName, chainName, effectSentence } from './effect-text';

  interface Props {
    effects: readonly CatalogEffect[];
    form?: 'mini' | 'full';
    /** The chain symbol this card shows, and the card it makes free. */
    chainOut?: Chain;
    chainTo?: CardSlug;
    /** Small-grid glyphs, for the mini tier. */
    small?: boolean;
    class?: string;
  }

  let {
    effects,
    form = 'mini',
    chainOut,
    chainTo,
    small = false,
    class: className = ''
  }: Props = $props();

  // In the glyph row, a guild's "coins now, points later" pair for the same
  // colours draws as one glyph (coin, laurel / colours), which keeps it in
  // one line on the smallest cards. The sentences stay separate.
  const items = $derived.by(() => {
    const out: { effect: CatalogEffect; points?: number }[] = [];
    const used: boolean[] = [];
    effects.forEach((effect, i) => {
      if (used[i]) return;
      if (effect.kind === 'coins-per-card') {
        const j = effects.findIndex(
          (other, k) =>
            !used[k] &&
            other.kind === 'points-per-card' &&
            other.scope === effect.scope &&
            (other.colors ?? []).join() === (effect.colors ?? []).join()
        );
        if (j >= 0) {
          used[j] = true;
          out.push({ effect, points: effects[j]?.n ?? 1 });
          return;
        }
      }
      out.push({ effect });
    });
    return out;
  });
</script>

{#if form === 'mini'}
  <span
    class="inline-flex max-w-full min-w-0 flex-wrap items-center justify-center gap-y-[0.1em] {small
      ? 'gap-x-[0.18em]'
      : 'gap-x-[0.3em]'} {className}"
  >
    {#each items as item, i (i)}
      <EffectGlyph effect={item.effect} points={item.points} {small} />
    {/each}
    {#if chainOut}
      <ChainChip chain={chainOut} direction="out" />
    {/if}
  </span>
{:else}
  <ul class="flex flex-col gap-[0.45em] text-start {className}">
    {#each effects as effect, i (i)}
      <li class="flex items-start gap-[0.5em]">
        <span class="inline-flex min-w-[2.2em] justify-center pt-[0.05em]">
          <EffectGlyph {effect} />
        </span>
        <span class="leading-snug">{effectSentence(effect)}</span>
      </li>
    {/each}
    {#if chainOut}
      <li class="flex items-start gap-[0.5em]">
        <span class="inline-flex min-w-[2.2em] justify-center">
          <ChainChip chain={chainOut} direction="out" />
        </span>
        <span class="leading-snug">
          {chainTo
            ? t('card.chainOut', { symbol: chainName(chainOut), card: cardName(chainTo) })
            : chainName(chainOut)}
        </span>
      </li>
    {/if}
  </ul>
{/if}
