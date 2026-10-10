<script lang="ts">
  // One piece of approved bitmap art, or its placeholder.
  //
  // It looks the id up in the generated index (lookup.ts) and draws an <img>
  // only when approved art exists; otherwise, and also if the file fails to
  // load, it renders the `fallback` snippet (the caller's procedural
  // placeholder), so there is never a broken image. The image is lazy and
  // decoded off the main thread, fades in once loaded (instantly under
  // reduced motion), and is decorative: the frame around it carries the name.
  //
  // `feather` blends the art into the paper behind it with a CSS mask, so
  // the art needs no alpha channel and never shows a hard rectangle:
  //   soft   a wide fade along the four edges, wider at the top and the
  //          bottom (cards, wonder panoramas)
  //   edge   a narrower even fade along the four edges (small thumbnails)
  //   none   no mask (art that is already shaped, such as tokens and backs)
  import type { Snippet } from 'svelte';
  import { pickArt, type ArtIndex, type ArtSize } from './lookup';

  interface Props {
    /** The art id, "<kind>/<slug>". */
    id: string;
    /** The preferred shipped size ('default', 'thumb', ...). */
    size?: ArtSize;
    /** When set, every shipped size goes in a srcset with this `sizes`, so
     *  the browser picks by the rendered width and the pixel ratio. */
    sizes?: string;
    feather?: 'soft' | 'edge' | 'none';
    /** Where the cover crop is anchored (CSS object-position). */
    position?: string;
    /** An index other than the generated one (the dev gallery). */
    index?: ArtIndex;
    class?: string;
    /** Drawn when there is no approved art, or the file failed to load. */
    fallback?: Snippet;
  }

  let {
    id,
    size = 'default',
    sizes,
    feather = 'soft',
    position = '50% 50%',
    index,
    class: className = '',
    fallback
  }: Props = $props();

  const pick = $derived(pickArt(id, size, index));
  // Keyed by the source, so a frame that switches to another id tries again.
  let failedSrc = $state<string | null>(null);
  let loadedSrc = $state<string | null>(null);
  const failed = $derived(pick !== undefined && failedSrc === pick.src);
</script>

{#if pick && !failed}
  <img
    src={pick.src}
    srcset={sizes ? pick.srcset : undefined}
    {sizes}
    width={pick.width}
    height={pick.height}
    alt=""
    aria-hidden="true"
    loading="lazy"
    decoding="async"
    draggable="false"
    class="art-image feather-{feather} object-cover select-none {className}"
    class:loaded={loadedSrc === pick.src}
    style:object-position={position}
    data-art={id}
    onload={() => (loadedSrc = pick.src)}
    onerror={() => (failedSrc = pick.src)}
  />
{:else}
  {@render fallback?.()}
{/if}

<style>
  .art-image {
    opacity: 0;
    /* --dur-base is 0 under reduced motion, so the art just appears. */
    transition: opacity var(--dur-base) ease-out;
  }
  .art-image.loaded {
    opacity: 1;
  }
  /* Two linear fades multiplied together: the art keeps its full strength
     in the middle and melts into the paper at every edge, with naturally
     rounder corners, and no oval vignette. */
  .feather-soft {
    -webkit-mask-image:
      linear-gradient(to bottom, transparent, #000 22%, #000 76%, transparent),
      linear-gradient(to right, transparent, #000 15%, #000 85%, transparent); /* physical-ok: symmetric fade */
    -webkit-mask-composite: source-in;
    mask-image:
      linear-gradient(to bottom, transparent, #000 22%, #000 76%, transparent),
      linear-gradient(to right, transparent, #000 15%, #000 85%, transparent); /* physical-ok: symmetric fade */
    mask-composite: intersect;
  }
  .feather-edge {
    -webkit-mask-image:
      linear-gradient(to bottom, transparent, #000 14%, #000 86%, transparent),
      linear-gradient(to right, transparent, #000 14%, #000 86%, transparent); /* physical-ok: symmetric fade */
    -webkit-mask-composite: source-in;
    mask-image:
      linear-gradient(to bottom, transparent, #000 14%, #000 86%, transparent),
      linear-gradient(to right, transparent, #000 14%, #000 86%, transparent); /* physical-ok: symmetric fade */
    mask-composite: intersect;
  }
</style>
