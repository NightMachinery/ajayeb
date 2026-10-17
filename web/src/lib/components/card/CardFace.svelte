<script lang="ts">
  // The face of a building card, in one of four tiers chosen by pixel width
  // (layout/fit.ts tierFor), never by device:
  //
  //   band    a 24 px colour strip with the effect glyphs (city stacks)
  //   mini    under 72 px wide (the phone pyramid): cost row, effect band,
  //           and a small strip of the card's art thumbnail below the 40%
  //           line (the placeholder pattern until art is approved)
  //   medium  72 to 139 px: as mini, plus the name, and the card's art
  //           thumbnail (CardArt; the placeholder until art is approved)
  //   full    140 px and wider (sheets): larger art (never squeezed below
  //           30% of the card) and the category label. The chain-out
  //           symbol is the chip in the effect band; the sheet beside the
  //           card spells out what it chains to.
  //
  // Names break between words only, never inside one: a name whose longest
  // word would not fit the card's width shrinks a step instead (nameFit).
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
  //
  // Tooltips (M10.tooltip-coverage): every pip of the cost, every effect
  // glyph, the chain chips and the badge carry their own; the colour band
  // and the category glyph name the category; and a mini card, which prints
  // no name, names itself (view/tips.ts cardTip). `cost`, the server's
  // breakdown for the seat in view, lets the cost pips and the badge say how
  // the payment covers each resource.
  import { artId, hasApprovedArt } from '$lib/art/lookup';
  import { cardBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import { i18n, t } from '$lib/i18n';
  import type { CardTier } from '$lib/layout/fit';
  import type { Afford } from '$lib/view/afford';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { cardTip } from '$lib/view/tips';
  import type { CostView } from '$lib/wire';
  import AffordBadge from './AffordBadge.svelte';
  import CardArt from './CardArt.svelte';
  import ChainChip from './ChainChip.svelte';
  import CostPips from './CostPips.svelte';
  import EffectGlyphs from './EffectGlyphs.svelte';
  import { bandBackground, bandColor, categoryIcon, longestWord } from './card-style';
  import { cardName } from './effect-text';

  interface Props {
    slug: string;
    tier: CardTier;
    /** The card's width in px, when the parent does not set --cw. */
    width?: number;
    afford?: Afford;
    /** Not your turn: dim the badge (shown for planning). */
    dim?: boolean;
    /** The badge's Urbanism bonus on a chain build (AffordBadge bonus). */
    bonus?: number;
    /** The badge's coins that go to the opponent through Economy (AffordBadge flow). */
    flow?: number;
    /** Who receives `flow` (user text). */
    flowTo?: string;
    /** The badge's recap line: how its price changed (AffordBadge priceTip). */
    priceTip?: string;
    /** The server's cost breakdown for the seat in view (the tooltips). */
    cost?: CostView;
    class?: string;
  }

  let {
    slug,
    tier,
    width,
    afford,
    dim = false,
    bonus = 0,
    flow = 0,
    flowTo = '',
    priceTip,
    cost,
    class: className = ''
  }: Props = $props();

  const card = $derived(cardBySlug(slug));
  const withArt = $derived(hasApprovedArt(artId('card', slug)));
  const name = $derived(card ? cardName(card.slug) : '');
  // The category glyph sits by the name on the full tier, and on the medium
  // tier once bitmap art replaces the placeholder (see the name row below).
  const nameGlyph = $derived(tier === 'full' || withArt);
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
      dir="ltr"
      data-tier="band"
      aria-hidden="true"
      {@attach tooltip({ text: cardTip(card.slug) })}
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
      style:--glyph-ink="var(--card-{card.color}-paper-ink)"
      dir="ltr"
      data-tier={tier}
      data-color={card.color}
      aria-hidden="true"
      {@attach tooltip(tier === 'mini' ? { text: cardTip(card.slug) } : undefined)}
    >
      <!-- Top 40%: cost row, then the effect band. -->
      <span class="cost flex items-center {tier === 'full' ? 'px-[4%]' : 'px-[2.5%]'}">
        <CostPips
          cost={card.cost}
          chainIn={card.chainIn}
          {small}
          dense={tier !== 'full'}
          room={costRoom}
          pay={cost}
          class="w-full"
        />
      </span>
      <span
        class="band flex items-center justify-center px-[2.5%] [--pat:color-mix(in_oklch,var(--card-ink)_20%,transparent)]"
        style:background={bandBackground(card.color)}
        {@attach tooltip({ text: t(`card.category.${card.color}`) })}
      >
        <span
          class="flex max-w-full min-w-0 items-center justify-center rounded-[0.35em] bg-(--paper)/92 py-[0.12em] {small
            ? 'px-0 py-0!'
            : 'px-[0.3em]'}"
        >
          <EffectGlyphs
            effects={card.effects}
            chainOut={card.chainOut}
            {small}
            class={small ? 'gap-x-[0.05em]!' : ''}
          />
        </span>
      </span>

      <!-- Below the 40% line: art (with the badge on its lower edge), name. -->
      <span class="art relative min-h-0 flex-1">
        {#if tier === 'mini'}
          <!-- Mini cards show a small strip of the art thumbnail, below the
               40% line so it never competes with the cost and the effect;
               the placeholder pattern (no glyph, too small) until then. -->
          <CardArt
            slug={card.slug}
            color={card.color}
            tier="mini"
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
            <AffordBadge
              {afford}
              {dim}
              compact={tier === 'mini'}
              {bonus}
              {flow}
              {flowTo}
              {priceTip}
              {cost}
            />
          </span>
        {/if}
      </span>
      {#if tier !== 'mini'}
        <span class="name flex items-start gap-[0.25em] px-[5%] pt-[0.2em] pb-[0.3em]">
          <span
            dir={i18n.dir}
            class="name-text line-clamp-2 min-w-0 flex-1 font-display leading-[1.1] wrap-normal hyphens-manual"
            style:--name-chars={longestWord(name)}
            style:--glyph-room={nameGlyph ? 1.3 : 0}>{name}</span
          >
          <!-- The placeholder art carries the category glyph; the full tier
               repeats it by the name, where there is room, and so does the
               medium tier once bitmap art (which has no glyph) replaces the
               placeholder, so the category never rests on colour alone. On
               the paper it takes the card colour's paper ink, since the band
               colour itself (yellow, grey) is too pale there. -->
          {#if nameGlyph}
            <span
              class="mt-[0.1em] inline-flex"
              {@attach tooltip(
                tier === 'full' ? undefined : { text: t(`card.category.${card.color}`) }
              )}
            >
              <Icon
                name={categoryIcon(card.color)}
                class="text-[1.05em] text-(--glyph-ink) [--i2:color-mix(in_oklch,var(--glyph-ink)_45%,var(--card-paper))]"
              />
            </span>
          {/if}
        </span>
        {#if tier === 'full'}
          <span
            dir={i18n.dir}
            class="px-[5%] pb-[0.5em] text-[length:max(12px,0.75em)] leading-tight text-(--ink-soft)"
          >
            {t(`card.category.${card.color}`)}
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
  /* nameFit: the name keeps the row's size unless its longest word
     (--name-chars letters, at about 0.6 em each in Fraunces) would not fit
     the row (the card's width less its padding and border, and less the
     category glyph when it shows); then it shrinks to fit, down to 9 px.
     Lines break only between words. */
  .name-text {
    font-size: max(
      9px,
      min(
        1em,
        calc((var(--cw, 72px) * 0.9 - 3px - var(--glyph-room) * 1em) / (var(--name-chars) * 0.6))
      )
    );
  }
  /* A 48 by 74 px card still needs a 14 px chain chip in each row.
     Reserve its height in the cost row, taking the remainder of the
     printed top 40% for the effect band. Tighter padding and gaps keep
     the effects on one row without shrinking their 10 px numerals. */
  .card-face[data-tier='mini'] .cost {
    flex-shrink: 0;
    height: max(14px, 17%);
    padding-inline: 0;
  }
  .card-face[data-tier='mini'] .cost :global([role='img']) {
    column-gap: 0;
  }
  .card-face[data-tier='mini'] .band {
    flex-shrink: 0;
    height: calc(40% - max(14px, 17%));
    padding-inline: 0;
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
  /* The art keeps a usable height under a two-line name. */
  .full .art {
    min-height: 30%;
  }
</style>
