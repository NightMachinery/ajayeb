<script lang="ts">
  // The face of a building card, in one of four tiers chosen by pixel width
  // (layout/fit.ts tierFor), never by device:
  //
  //   band    a 24 px colour strip with the effect glyphs (city stacks)
  //   mini    under 72 px wide (the phone pyramid): cost row, effect band,
  //           the placeholder pattern (never bitmap art)
  //   medium  72 to 139 px: as mini, plus the name, and the card's art
  //           thumbnail (CardArt; the placeholder until art is approved)
  //   full    140 px and wider (sheets): larger art, the category label and
  //           the chain-out line
  //
  // Everything a decision needs (cost, chain-in, effects, chain-out) sits in
  // the top 40% of the card, so a card half covered by the row below still
  // reads, as on the physical table. The affordability badge sits low on the
  // card, because only uncovered cards can be taken. Cards stay paper-
  // coloured in both themes.
  //
  // A plain leaf component: it takes a slug and the display state, fills its
  // container (give the container the card's size, or pass `width`), and is
  // decorative; the slot button around it carries the ARIA name.
  import { artId, hasApprovedArt } from '$lib/art/lookup';
  import { cardBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import { t } from '$lib/i18n';
  import type { CardTier } from '$lib/layout/fit';
  import type { Afford } from '$lib/view/afford';
  import AffordBadge from './AffordBadge.svelte';
  import CardArt from './CardArt.svelte';
  import CardPlaceholderArt from './CardPlaceholderArt.svelte';
  import ChainChip from './ChainChip.svelte';
  import CostPips from './CostPips.svelte';
  import EffectGlyphs from './EffectGlyphs.svelte';
  import { bandBackground, bandColor, categoryIcon } from './card-style';
  import { cardName, chainName } from './effect-text';

  interface Props {
    slug: string;
    tier: CardTier;
    /** The card's width in px, when the parent does not set --cw. */
    width?: number;
    afford?: Afford;
    /** Not your turn: dim the badge (shown for planning). */
    dim?: boolean;
    class?: string;
  }

  let { slug, tier, width, afford, dim = false, class: className = '' }: Props = $props();

  const card = $derived(cardBySlug(slug));
  const withArt = $derived(hasApprovedArt(artId('card', slug)));
  const small = $derived(tier === 'mini');
  // The cost row's room in em, so CostPips can collapse what would not fit.
  // Mirrors the font sizes and paddings in the style block below.
  const costRoom = $derived.by(() => {
    if (width === undefined) return undefined;
    const font =
      tier === 'full'
        ? Math.min(20, Math.max(14, width * 0.1))
        : Math.min(15, Math.max(10, width * 0.16));
    return (width * (tier === 'full' ? 0.92 : 0.95) - 3) / font;
  });
</script>

{#if card}
  {#if tier === 'band'}
    <span
      class="flex h-6 w-full items-center gap-1 overflow-hidden rounded-[4px] border border-black/25 px-1 text-[12px] text-card-ink
        [--pat:color-mix(in_oklch,var(--card-ink)_18%,transparent)] {className}"
      style:background={bandBackground(card.color)}
      data-tier="band"
      aria-hidden="true"
    >
      <span class="flex min-w-0 flex-1 justify-start rounded-[3px] bg-(--paper)/90 px-1 py-px">
        <EffectGlyphs effects={card.effects} small class="flex-nowrap! justify-start!" />
      </span>
      {#if card.chainOut}
        <ChainChip chain={card.chainOut} direction="out" />
      {/if}
    </span>
  {:else}
    <span
      class="card-face relative flex h-full w-full flex-col overflow-hidden rounded-[7%/4.5%] bg-card-paper text-card-ink {className}"
      class:full={tier === 'full'}
      style:--cw={width === undefined ? undefined : `${width}px`}
      style:--band={bandColor(card.color)}
      data-tier={tier}
      data-color={card.color}
      aria-hidden="true"
    >
      <!-- Top 40%: cost row, then the effect band. -->
      <span class="cost flex items-center {tier === 'full' ? 'px-[4%]' : 'px-[2.5%]'}">
        <CostPips
          cost={card.cost}
          chainIn={card.chainIn}
          {small}
          dense={tier !== 'full'}
          room={costRoom}
          class="w-full"
        />
      </span>
      <span
        class="band flex items-center justify-center px-[2.5%] [--pat:color-mix(in_oklch,var(--card-ink)_20%,transparent)]"
        style:background={bandBackground(card.color)}
      >
        <span
          class="flex max-w-full min-w-0 items-center justify-center rounded-[0.35em] bg-(--paper)/92 py-[0.12em] {small
            ? 'px-[0.15em]'
            : 'px-[0.3em]'}"
        >
          <EffectGlyphs effects={card.effects} chainOut={card.chainOut} {small} />
        </span>
      </span>

      <!-- Below the 40% line: art (with the badge on its lower edge), name. -->
      <span class="relative min-h-0 flex-1">
        {#if tier === 'mini'}
          <!-- Mini cards never show bitmap art: the phone pyramid covers
               most of them, and the pattern keeps the colour readable. -->
          <CardPlaceholderArt
            color={card.color}
            glyph={false}
            class="absolute inset-0 h-full w-full"
          />
        {:else}
          <CardArt
            slug={card.slug}
            color={card.color}
            {tier}
            {width}
            class="absolute inset-0 h-full w-full"
          />
        {/if}
        {#if afford}
          <span class="pointer-events-none absolute inset-x-0 bottom-[0.3em] flex justify-center">
            <AffordBadge {afford} {dim} compact={tier === 'mini'} />
          </span>
        {/if}
      </span>
      {#if tier !== 'mini'}
        <span class="name flex items-start gap-[0.25em] px-[5%] pt-[0.2em] pb-[0.3em]">
          <span
            class="line-clamp-2 min-w-0 flex-1 font-display leading-[1.1] wrap-anywhere hyphens-auto"
            >{cardName(card.slug)}</span
          >
          <!-- The placeholder art carries the category glyph; the full tier
               repeats it by the name, where there is room, and so does the
               medium tier once bitmap art (which has no glyph) replaces the
               placeholder, so the category never rests on colour alone. -->
          {#if tier === 'full' || withArt}
            <Icon
              name={categoryIcon(card.color)}
              class="mt-[0.1em] text-[1.05em] text-(--band) [--i2:color-mix(in_oklch,var(--band)_45%,var(--card-paper))]"
            />
          {/if}
        </span>
        {#if tier === 'full'}
          <span
            class="flex flex-col gap-[0.15em] px-[5%] pb-[0.5em] text-[length:max(12px,0.75em)] leading-tight text-(--ink-soft)"
          >
            <span>{t(`card.category.${card.color}`)}</span>
            {#if card.chainOut && card.chainTo}
              <span
                >{t('card.chainOut', {
                  symbol: chainName(card.chainOut),
                  card: cardName(card.chainTo)
                })}</span
              >
            {/if}
          </span>
        {/if}
      {/if}
    </span>
  {/if}
{/if}

<style>
  /* Sizes follow the card width (--cw, set by the structure or the width
     prop). Numerals keep a 10 px floor (data-min-font marks them). */
  .card-face {
    font-size: clamp(10px, calc(var(--cw, 72px) * 0.16), 15px);
    border: 1.5px solid color-mix(in oklch, var(--band) 75%, var(--card-ink));
    box-shadow: inset 0 0 0 1px color-mix(in oklch, var(--card-paper) 70%, transparent);
  }
  .cost {
    height: 16%;
  }
  .band {
    height: 24%;
  }
  .name {
    font-size: max(12px, 0.88em);
  }
  /* The full tier (sheets, 150 px and up) is roomier. */
  .card-face.full {
    font-size: clamp(14px, calc(var(--cw, 180px) * 0.1), 20px);
  }
  .full .cost {
    height: 11%;
  }
  .full .band {
    height: 17%;
  }
  .full .name {
    font-size: max(16px, 1.05em);
  }
</style>
