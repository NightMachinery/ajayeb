<script lang="ts">
  // One printed effect in its compact glyph form, for the card's effect band
  // and for wonder tiles: "wood wood", "shield ×2", "laurel 4", "coin 2 per
  // grey card" and so on, one form per effect kind of the catalog. The
  // sentence form is effect-text.ts; EffectGlyphs shows either. Decorative:
  // the card's ARIA name carries the words.
  import ArchiveRestore from '@lucide/svelte/icons/archive-restore';
  import RotateCw from '@lucide/svelte/icons/rotate-cw';
  import X from '@lucide/svelte/icons/x';
  import Icon from '$lib/icons/Icon.svelte';
  import { formatNumber } from '$lib/i18n';
  import type { CatalogEffect } from '$lib/catalog';
  import type { Color } from '$lib/wire';
  import Coin from './Coin.svelte';
  import ResourceTile from './ResourceTile.svelte';
  import { bandColor, bandInk } from './card-style';

  interface Props {
    effect: CatalogEffect;
    /** Small-grid glyphs, for the mini tier. */
    small?: boolean;
    /** For coins-per-card: the points per card of a matching points-per-card
     *  effect, drawn in the same glyph (guilds). */
    points?: number;
  }

  let { effect, small = false, points }: Props = $props();

  const n = $derived(effect.n ?? 1);
  const max = $derived(effect.scope === 'max');
</script>

<!-- Victory points: a green laurel disc with the numeral, the counterpart of
     the gold coin disc (dark on light for coins, light on dark for points,
     so the two differ in greyscale too). -->
{#snippet laurel(value: number, stacked = false)}
  <span
    class="{stacked
      ? '-ms-[0.4em]'
      : ''} inline-grid h-[1.25em] min-w-[1.25em] shrink-0 place-items-center rounded-[0.45em] border border-(color:--vp-edge) bg-(--vp) px-[0.15em] leading-none font-bold text-(--paper-bright) tabular-nums"
    data-min-font="10">{formatNumber(value)}</span
  >
{/snippet}

<!-- A card colour counted by an effect ("per brown card"): a small card
     shape in the band colour with the colour's category glyph on it. The
     glyph, not the hue, names the colour, so it reads in greyscale and for
     colour-blind players (green, red and brown share one olive under
     deuteranopia). -->
{#snippet swatches(colors: readonly Color[])}
  <span
    class="inline-flex items-center rounded-[0.2em] {small
      ? 'gap-0 p-[0.05em]'
      : 'gap-[0.1em] p-[0.08em]'} {max
      ? 'outline-[1.5px] outline-(--card-ink) outline-dashed'
      : ''}"
  >
    {#each colors as color (color)}
      <span
        class="inline-grid shrink-0 place-items-center rounded-[0.2em] border border-(--card-ink)/40
          {small ? 'h-[1.2em] w-[0.9em]' : 'h-[1.35em] w-[1.15em]'}"
        style:background={bandColor(color)}
        style:color={bandInk(color)}
        style:--i2="color-mix(in oklch, {bandInk(color)} 45%, {bandColor(color)})"
        data-swatch={color}
      >
        <Icon name={`category-${color}-sm`} class={small ? 'text-[0.9em]' : 'text-[1em]'} />
      </span>
    {/each}
  </span>
{/snippet}

{#snippet per()}
  <span class="font-bold" data-min-font="10" aria-hidden="true">/</span>
{/snippet}

{#snippet playAgain()}
  <RotateCw size="1em" strokeWidth={3} class="shrink-0" />
{/snippet}

<span
  class="glyph inline-flex shrink-0 items-center leading-none {small
    ? 'gap-[0.06em]'
    : 'gap-[0.12em]'}"
  data-effect={effect.kind}
  aria-hidden="true"
>
  {#if effect.kind === 'produce' && effect.resource}
    {#each { length: n }, i (i)}
      <ResourceTile resource={effect.resource} {small} />
    {/each}
  {:else if effect.kind === 'produce-choice'}
    {#each effect.resources ?? [] as r, i (r)}
      {#if i > 0}{@render per()}{/if}
      <ResourceTile resource={r} {small} />
    {/each}
  {:else if effect.kind === 'trade-fix'}
    {#each effect.resources ?? [] as r (r)}
      <ResourceTile resource={r} {small} />
    {/each}
    <Icon name="price-tag" class="text-[0.9em] text-(--gold-deep)" />
    <Coin n={1} />
  {:else if effect.kind === 'shields'}
    {#each { length: n }, i (i)}
      <Icon name="shield" class="text-[1.15em] text-(--card-red) [--i2:var(--paper-bright)]" />
    {/each}
  {:else if effect.kind === 'science' && effect.symbol}
    <Icon
      name={small ? `science-${effect.symbol}-sm` : `science-${effect.symbol}`}
      class="text-[1.2em] text-(--card-green) [--i2:var(--paper-bright)]"
    />
  {:else if effect.kind === 'points'}
    {@render laurel(n)}
  {:else if effect.kind === 'coins'}
    <Coin {n} />
  {:else if effect.kind === 'coins-per-card'}
    <!-- A guild's coins and points per card share one glyph; on mini cards
         the laurel overlaps the coin, which keeps both numerals clear and
         leaves room for two colour swatches on a 54 px card. -->
    <Coin {n} />{#if points !== undefined}{@render laurel(
        points,
        small
      )}{/if}{@render per()}{@render swatches(effect.colors ?? [])}
  {:else if effect.kind === 'coins-per-wonder'}
    <Coin {n} />{@render per()}<Icon name="wonder" class="text-[1.1em]" />
  {:else if effect.kind === 'points-per-card'}
    {@render laurel(n)}{@render per()}{@render swatches(effect.colors ?? [])}
  {:else if effect.kind === 'points-per-wonder'}
    {@render laurel(n)}{@render per()}
    <span
      class="inline-flex rounded-[0.2em] p-[0.08em] {max
        ? 'outline-[1.5px] outline-(--card-ink) outline-dashed'
        : ''}"
    >
      <Icon name="wonder" class="text-[1.1em]" />
    </span>
  {:else if effect.kind === 'points-per-coins'}
    {@render laurel(1)}{@render per()}<Coin {n} />
  {:else if effect.kind === 'points-per-token'}
    {@render laurel(n)}{@render per()}<Icon name="token-frame" class="text-[1.1em]" />
  {:else if effect.kind === 'play-again'}
    {@render playAgain()}
  {:else if effect.kind === 'destroy'}
    <!-- The cross sits on the swatch's corner, so the category glyph under
         it stays readable. -->
    <span class="relative me-[0.3em] inline-flex">
      {@render swatches(effect.colors ?? [])}
      <span
        class="absolute -end-[0.45em] -bottom-[0.3em] inline-grid size-[0.8em] place-items-center rounded-full bg-(--paper-bright) text-(--cinnabar) ring-1 ring-(--cinnabar)/50"
      >
        <X size="0.7em" strokeWidth={4} />
      </span>
    </span>
  {:else if effect.kind === 'build-from-discard'}
    <ArchiveRestore size="1.1em" strokeWidth={2.5} class="shrink-0" />
  {:else if effect.kind === 'draw-tokens'}
    <Icon name="token-frame" class="text-[1.1em]" />
    <span class="font-bold tabular-nums" data-min-font="10">×{formatNumber(n)}</span>
  {:else if effect.kind === 'opponent-loses-coins'}
    <Icon name="coin-loss" class="text-[1.1em] text-(--cinnabar)" />
    <Coin {n} loss />
  {:else if effect.kind === 'discount'}
    <span class="font-bold tabular-nums" data-min-font="10">−{formatNumber(n)}</span>
    {#if effect.target === 'wonder'}
      <Icon name="wonder" class="text-[1.1em]" />
    {:else}
      <Icon name="category-blue" class="text-[1.1em] text-(--card-blue)" />
    {/if}
  {:else if effect.kind === 'trade-income'}
    <Icon name="trade" class="text-[1.1em]" /><Icon
      name="coin"
      class="text-[1.1em] text-(--gold-deep)"
    />
  {:else if effect.kind === 'extra-shield'}
    <Icon name="shield" class="text-[1.15em] text-(--card-red) [--i2:var(--paper-bright)]" />
    <span class="font-bold tabular-nums" data-min-font="10">+{formatNumber(n)}</span>
  {:else if effect.kind === 'wonders-play-again'}
    <Icon name="wonder" class="text-[1.1em]" />{@render playAgain()}
  {:else if effect.kind === 'chain-bonus'}
    <Icon name="badge-chain" class="text-[1.1em] text-(--gold-deep)" />
    <Coin {n} />
  {/if}
</span>

<style>
  .glyph {
    --vp: oklch(0.45 0.1 150);
    --vp-edge: oklch(0.35 0.08 150);
  }
</style>
