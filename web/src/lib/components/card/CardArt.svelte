<script lang="ts">
  // The art of a building card: the card's approved bitmap (4:3, feathered
  // into the paper with a CSS mask) when the generated art index has one,
  // and the procedural CardPlaceholderArt otherwise or if the file fails to
  // load. The mini and medium tiers show the thumbnail (the mini tier only
  // as a strip below the 40% line, with a narrower fade and no placeholder
  // glyph); the full tier picks by width. Fills its container; decorative.
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { artId, cardArtSizes } from '$lib/art/lookup';
  import type { Color } from '$lib/wire';
  import CardPlaceholderArt from './CardPlaceholderArt.svelte';

  interface Props {
    slug: string;
    color: Color;
    tier: 'mini' | 'medium' | 'full';
    /** The card's width in px, when known: the full tier then lets the
     *  browser pick the shipped size by width and pixel ratio. */
    width?: number;
    class?: string;
  }

  let { slug, color, tier, width, class: className = '' }: Props = $props();
</script>

<span
  class="card-art relative block overflow-hidden bg-card-paper {className}"
  data-card-art={slug}
>
  <ArtImage
    id={artId('card', slug)}
    size={tier === 'full' ? 'default' : 'thumb'}
    sizes={tier === 'full' && width !== undefined ? cardArtSizes(width) : undefined}
    feather={tier === 'mini' ? 'edge' : 'soft'}
    class="absolute inset-0 h-full w-full"
  >
    {#snippet fallback()}
      <CardPlaceholderArt {color} glyph={tier !== 'mini'} class="absolute inset-0 h-full w-full" />
    {/snippet}
  </ArtImage>
</span>
