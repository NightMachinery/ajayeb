<script lang="ts">
  // A wonder, at one of two sizes:
  //
  //   draft  the draft tile: 3:2 art, the name, the printed cost
  //          and the effect glyphs (and who took it, once taken)
  //   city   a row in a city panel: a small art thumb, the name, the effect
  //          glyphs and the state: built (with the card tucked under it), out
  //          of the game, or open with its affordability badge
  //   row    a line of a wonder list (the wonder sheet, the desktop city
  //          panel), drawn on the panel rather than on card paper: the art
  //          thumb, the name, a one-line effect summary in words, the
  //          printed cost and ONE status: "Built with Palisade", "Out of
  //          the game", "Not built", or, when the caller says a choice of
  //          the view can build it now, "Can build now" in the accent
  //
  // The art is the wonder's approved bitmap (3:2, feathered into the paper)
  // when the generated art index has one, and a procedural sky with the
  // wonder glyph otherwise or if the file fails to load. A wonder whose
  // subject reaches the top of its painting (artCrop) anchors its crop
  // there and keeps a narrow top fade, so the subject is never cut off.
  //
  // A plain leaf component: printed data from the catalog, the state from
  // the server's WonderView, the badge from view/afford.ts. It renders no
  // button of its own; the parent wraps it in one (with describeWonder for
  // the name) when the tile is actionable.
  //
  // Hints (M10.HS4, `hint`, view/board-hints.ts wonderHint): on the row
  // and city sizes, the hourglass once only one more wonder can be built
  // (solid when its owner can afford it now), and the ready mark on a
  // wonder its owner could build at today's prices though no choice of the
  // view builds it now (solid for this screen's own seat, outlined in the
  // opponent's crest colour). The row also shows what building it would do
  // now, as an outcome line.
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { artId, WONDER_ART_SIZES } from '$lib/art/lookup';
  import { wonderArtCrop } from '$lib/art/crops';
  import { wonderBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import { t } from '$lib/i18n';
  import type { Afford } from '$lib/view/afford';
  import type { WonderHint } from '$lib/view/board-hints';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { WonderView } from '$lib/wire';
  import OutcomeLine from '../game/OutcomeLine.svelte';
  import Hourglass from './Hourglass.svelte';
  import AffordBadge from '../card/AffordBadge.svelte';
  import CostPips from '../card/CostPips.svelte';
  import EffectGlyphs from '../card/EffectGlyphs.svelte';
  import { cardName, effectsText, wonderName } from '../card/effect-text';

  interface Props {
    slug: string;
    size?: 'draft' | 'city' | 'row';
    /** The owner's WonderView, for the built, removed and tucked state. */
    state?: WonderView;
    afford?: Afford;
    /** Not your turn: dim the badge. */
    dim?: boolean;
    /** Draft: the name of the player who took it (user text). */
    takenBy?: string;
    /** Draft: a shorter 2:1 art band, for a phone's draft grid. */
    compactArt?: boolean;
    /** Row: a wonder choice of the view can build it now. */
    buildable?: boolean;
    /** The board hints about it (row and city sizes). */
    hint?: WonderHint;
    class?: string;
  }

  let {
    slug,
    size = 'draft',
    state,
    afford,
    dim = false,
    takenBy,
    compactArt = false,
    buildable = false,
    hint,
    class: className = ''
  }: Props = $props();

  const wonder = $derived(wonderBySlug(slug));
  const crop = $derived(wonderArtCrop(slug));
  const built = $derived(state?.built ?? false);
  const removed = $derived(state?.removed ?? false);
</script>

{#snippet art(cls: string, small: boolean)}
  <span class="relative flex items-center justify-center overflow-hidden bg-card-paper {cls}">
    <ArtImage
      id={artId('wonder', slug)}
      size={small ? 'thumb' : 'default'}
      sizes={small ? undefined : WONDER_ART_SIZES}
      feather={small ? 'edge' : crop.feather}
      position={crop.position}
      class="absolute inset-0 h-full w-full"
    >
      {#snippet fallback()}
        <span class="wonder-art absolute inset-0 flex items-center justify-center">
          <Icon
            name="wonder"
            class="text-[length:min(56cqh,40cqw)] text-(--lapis) opacity-85 [--i2:var(--gold)]"
          />
        </span>
      {/snippet}
    </ArtImage>
  </span>
{/snippet}

{#snippet marks()}
  {#if hint?.hourglass}
    <span
      class="hint-mark grid size-5 shrink-0 place-items-center rounded-full border text-fg
        {hint.hourglass.solid
        ? 'border-cinnabar bg-cinnabar/15 text-cinnabar'
        : 'border-rule-strong'}"
      data-hint="hourglass"
      data-hourglass={hint.hourglass.solid ? 'solid' : 'hollow'}
      {@attach tooltip({ text: hint.hourglass.tip })}
    >
      <Hourglass solid={hint.hourglass.solid} class="size-3" />
    </span>
  {/if}
  {#if hint?.ready}
    <span
      class="hint-mark ready-mark grid size-5 shrink-0 place-items-center rounded-full"
      data-hint="ready"
      data-style={hint.ready.style}
      data-tone={hint.ready.tone}
      {@attach tooltip({ text: hint.ready.tip })}
    >
      <Icon name="badge-check" class="text-[13px]" />
    </span>
  {/if}
{/snippet}

{#if wonder}
  {#if size === 'draft'}
    <span
      class="wonder-tile relative flex w-full flex-col overflow-hidden rounded-[10px] border-[1.5px] border-(color:--gold-deep) bg-card-paper text-card-ink shadow-panel {className}"
      class:opacity-60={takenBy !== undefined}
      data-wonder={slug}
      data-size="draft"
      aria-hidden="true"
    >
      {@render art(
        `${compactArt ? 'aspect-[2/1]' : 'aspect-[3/2]'} w-full [container-type:size]`,
        false
      )}
      <span class="flex flex-col gap-[0.35em] px-[0.6em] pt-[0.45em] pb-[0.6em] text-[13px]">
        <span class="font-display text-[15px] leading-tight">{wonderName(slug)}</span>
        <span class="flex min-h-[1.4em] items-center justify-between gap-2">
          <CostPips cost={wonder.cost} />
          {#if afford}<AffordBadge {afford} {dim} />{/if}
        </span>
        <EffectGlyphs effects={wonder.effects} class="justify-start! gap-x-[0.5em]" />
        {#if takenBy !== undefined}
          <span class="text-[12px] text-(--ink-soft)" dir="auto"
            >{t('wonder.takenBy', { name: takenBy })}</span
          >
        {/if}
      </span>
    </span>
  {:else if size === 'row'}
    <span
      class="wonder-tile relative flex w-full items-center gap-2.5 text-start text-fg {className}"
      class:removed
      data-wonder-row={slug}
      data-size="row"
      data-built={built}
      data-buildable={buildable ? '' : undefined}
      aria-hidden="true"
    >
      {@render art(
        `aspect-[3/2] w-16 shrink-0 rounded-[6px] border [container-type:size] ${built ? 'border-(color:--gold-deep)' : 'border-rule'}`,
        true
      )}
      <span class="flex min-w-0 flex-1 flex-col gap-0.5">
        <span class="line-clamp-2 font-display text-[15px] leading-tight wrap-normal"
          >{wonderName(slug)}</span
        >
        <span class="line-clamp-2 text-xs leading-snug text-fg-muted"
          >{effectsText(wonder.effects)}</span
        >
        {#if hint && hint.terms.length > 0}
          <OutcomeLine terms={hint.terms} />
        {/if}
        <span class="flex min-h-5 items-center gap-1.5 text-xs">
          {@render marks()}
          {#if built}
            <Icon
              name="badge-check"
              class="shrink-0 text-[14px] text-(--gold-deep) [--i2:var(--paper)]"
            />
            <span class="font-semibold" data-wonder-status="built"
              >{state?.tucked
                ? t('wonder.builtWith', { card: cardName(state.tucked) })
                : t('wonder.built')}</span
            >
          {:else if removed}
            <span class="text-fg-muted" data-wonder-status="removed">{t('wonder.removed')}</span>
          {:else}
            <span class="text-[13px]"><CostPips cost={wonder.cost} small /></span>
            {#if buildable}
              <!-- Still unbuilt, and buildable this turn: the one status,
                   in the accent. -->
              <span
                class="rounded-full bg-accent px-1.5 py-px text-[11px] font-semibold whitespace-nowrap text-on-accent"
                data-wonder-status="buildable"
                data-can-build
              >
                {t('wonder.canBuild')}
              </span>
            {:else}
              <span class="text-fg-muted" data-wonder-status="open">{t('wonder.notBuilt')}</span>
            {/if}
          {/if}
        </span>
      </span>
    </span>
  {:else}
    <span
      class="wonder-tile relative flex w-full items-stretch gap-2 overflow-hidden rounded-[8px] border bg-card-paper p-1 text-[12px] text-card-ink
        {built ? 'border-(color:--gold-deep)' : 'border-black/20'} {className}"
      class:removed
      data-wonder={slug}
      data-size="city"
      data-built={built}
      aria-hidden="true"
    >
      {@render art('aspect-[3/2] w-14 shrink-0 rounded-[5px] [container-type:size]', true)}
      <span class="flex min-w-0 flex-1 flex-col justify-center gap-[0.3em]">
        <span class="flex items-center gap-1">
          <!-- Two lines, broken between words, rather than an ellipsis: a
               narrow city row (a 360 px phone) still names the wonder. -->
          <span
            class="line-clamp-2 min-w-0 flex-1 font-display text-[13px] leading-tight wrap-normal"
            >{wonderName(slug)}</span
          >
          {#if built}
            <Icon
              name="badge-check"
              class="shrink-0 text-[14px] text-(--gold-deep) [--i2:var(--paper)]"
            />
          {/if}
          {@render marks()}
        </span>
        <EffectGlyphs effects={wonder.effects} small class="justify-start! gap-x-[0.4em]" />
      </span>
      <span class="flex shrink-0 flex-col items-end justify-center gap-1">
        {#if built}
          <span class="text-[12px] text-(--ink-soft)"
            >{state?.tucked ? cardName(state.tucked) : t('wonder.built')}</span
          >
        {:else if removed}
          <span class="text-[12px] text-(--ink-soft)">{t('wonder.removed')}</span>
        {:else}
          <CostPips cost={wonder.cost} small />
          {#if afford}<AffordBadge {afford} {dim} compact />{/if}
        {/if}
      </span>
    </span>
  {/if}
{/if}

<style>
  .wonder-art {
    background:
      radial-gradient(circle, var(--gold) 0 1px, transparent 1.5px) 0 0 / 10px 10px,
      linear-gradient(to bottom, var(--scene-sky), var(--scene-horizon) 70%);
  }
  /* The ready mark: the hint pip's look (card/HintPip.svelte). Solid for
     this screen's own seat, outlined in the opponent's crest colour, a
     crest tint for a spectator. */
  .ready-mark {
    --mark: var(--gold-deep);
    border: 1.5px solid var(--mark);
    color: var(--mark);
  }
  .ready-mark[data-tone='sun'] {
    --mark: var(--sun-ink);
  }
  .ready-mark[data-tone='moon'] {
    --mark: var(--moon-ink);
  }
  .ready-mark[data-style='solid'] {
    background: color-mix(in oklch, var(--mark) 18%, transparent);
  }
  .ready-mark[data-style='outline'] {
    border-style: dashed;
  }
  .removed {
    opacity: 0.55;
    filter: grayscale(0.8);
  }
</style>
