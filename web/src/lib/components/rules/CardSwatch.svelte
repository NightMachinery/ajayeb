<script lang="ts">
  // A card of one colour in miniature, for the concept diagrams: a small
  // card shape in the band colour with the colour's category glyph, the
  // same mark EffectGlyph draws for "per card of colour X". The glyph, not
  // the hue, names the colour, so it reads in greyscale too.
  import Icon from '$lib/icons/Icon.svelte';
  import type { Color } from '$lib/wire';
  import { bandColor, bandInk } from '../card/card-style';

  interface Props {
    color: Color;
    class?: string;
  }

  let { color, class: className = '' }: Props = $props();
</script>

<span
  class="inline-grid h-[1.6em] w-[1.25em] shrink-0 place-items-center rounded-[0.2em] border border-(--card-ink)/40 {className}"
  style:background={bandColor(color)}
  style:color={bandInk(color)}
  style:--i2="color-mix(in oklch, {bandInk(color)} 45%, {bandColor(color)})"
  aria-hidden="true"
  data-swatch={color}
>
  <Icon name={`category-${color}-sm`} class="text-[0.95em]" />
</span>
