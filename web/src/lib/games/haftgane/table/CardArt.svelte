<script lang="ts">
  // A card's picture: its approved painting when Haftgane's art index has
  // one (art/available-haftgane.gen.ts), otherwise a procedural placeholder
  // in the card's colour, with a pattern and the category glyph so the
  // colour is never the only cue. Decorative: the card's name and tooltip
  // describe it.
  import { art } from '$lib/art/available-haftgane.gen';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    slug: string;
    colour: string;
  }

  let { slug, colour }: Props = $props();

  // Art ids name the painting, so the Age suffix of a name printed in two
  // Ages is dropped (glassworks-age-1 and -age-2 share one).
  const entry = $derived(art[`card/${slug.replace(/-age-\d$/, '')}`]);
  const variant = $derived(entry?.variants['thumb'] ?? entry?.variants['default']);
</script>

<span class="art {colour}" style:--band="var(--card-{colour})" data-card-art>
  {#if variant}
    <img
      src={variant.path}
      width={variant.width}
      height={variant.height}
      alt=""
      loading="lazy"
      decoding="async"
    />
  {:else}
    <span class="glyph"><HgGlyph name="category-{colour}" /></span>
  {/if}
</span>

<style>
  .art {
    position: relative;
    display: grid;
    place-items: center;
    overflow: hidden;
    background-color: color-mix(in oklch, var(--band) 14%, var(--card-paper));
  }
  /* A band pattern per colour, so the placeholder never relies on hue. */
  .brown {
    background-image: repeating-linear-gradient(
      45deg,
      transparent 0 6px,
      color-mix(in oklch, var(--band) 16%, transparent) 6px 8px
    );
  }
  .grey {
    background-image: radial-gradient(
      color-mix(in oklch, var(--band) 30%, transparent) 1px,
      transparent 1.5px
    );
    background-size: 7px 7px;
  }
  .blue {
    background-image: repeating-linear-gradient(
      0deg,
      transparent 0 6px,
      color-mix(in oklch, var(--band) 16%, transparent) 6px 7px
    );
  }
  .yellow {
    background-image: radial-gradient(
      circle at 50% 40%,
      color-mix(in oklch, var(--band) 30%, transparent),
      transparent 60%
    );
  }
  .red {
    background-image: repeating-linear-gradient(
      -45deg,
      transparent 0 5px,
      color-mix(in oklch, var(--band) 16%, transparent) 5px 8px
    );
  }
  .green {
    background-image:
      repeating-linear-gradient(
        45deg,
        transparent 0 7px,
        color-mix(in oklch, var(--band) 14%, transparent) 7px 8px
      ),
      repeating-linear-gradient(
        -45deg,
        transparent 0 7px,
        color-mix(in oklch, var(--band) 14%, transparent) 7px 8px
      );
  }
  .purple {
    background-image: repeating-linear-gradient(
      90deg,
      transparent 0 6px,
      color-mix(in oklch, var(--band) 16%, transparent) 6px 7px
    );
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .glyph {
    color: var(--band);
    font-size: 2.2em;
    opacity: 0.85;
  }
</style>
