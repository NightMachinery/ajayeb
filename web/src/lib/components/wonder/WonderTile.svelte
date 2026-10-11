<script lang="ts">
  // A wonder, at one of two sizes:
  //
  //   draft  the draft tile: 3:2 art, the name, the printed cost
  //          and the effect glyphs (and who took it, once taken)
  //   city   a row in a city panel: a small art thumb, the name, the effect
  //          glyphs and the state: built (with the card tucked under it), out
  //          of the game, or open with its affordability badge
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
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { artId } from '$lib/art/lookup';
  import { wonderArtCrop } from '$lib/art/crops';
  import { wonderBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import { t } from '$lib/i18n';
  import type { Afford } from '$lib/view/afford';
  import type { WonderView } from '$lib/wire';
  import AffordBadge from '../card/AffordBadge.svelte';
  import CostPips from '../card/CostPips.svelte';
  import EffectGlyphs from '../card/EffectGlyphs.svelte';
  import { cardName, wonderName } from '../card/effect-text';

  interface Props {
    slug: string;
    size?: 'draft' | 'city';
    /** The owner's WonderView, for the built, removed and tucked state. */
    state?: WonderView;
    afford?: Afford;
    /** Not your turn: dim the badge. */
    dim?: boolean;
    /** Draft: the name of the player who took it (user text). */
    takenBy?: string;
    /** Draft: a shorter 2:1 art band, for a phone's draft grid. */
    compactArt?: boolean;
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
      sizes={small ? undefined : '(min-width: 40rem) 280px, 50vw'}
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
      linear-gradient(to bottom, oklch(0.88 0.05 230), oklch(0.93 0.04 85) 70%);
  }
  .removed {
    opacity: 0.55;
    filter: grayscale(0.8);
  }
</style>
