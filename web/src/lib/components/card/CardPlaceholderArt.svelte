<script lang="ts">
  // Procedural art for a card until its bitmap exists: the category pattern
  // in the card's colour on paper, with the category glyph in the middle.
  // Pure CSS gradients and one sprite icon, so it ships nothing. Decorative.
  import Icon from '$lib/icons/Icon.svelte';
  import type { Color } from '$lib/wire';
  import { bandColor, categoryIcon, patternLayers } from './card-style';

  interface Props {
    color: Color;
    /** Draw the glyph (off for the smallest cards, where it would be noise). */
    glyph?: boolean;
    class?: string;
  }

  let { color, glyph = true, class: className = '' }: Props = $props();
</script>

<span
  class="[container-type:size] relative flex items-center justify-center overflow-hidden {className}"
  style:color={bandColor(color)}
  style:--glyph-ink="var(--card-{color}-paper-ink)"
  style:--pat="color-mix(in oklch, {bandColor(color)} 22%, transparent)"
  style:background="{patternLayers(color)}, color-mix(in oklch, {bandColor(color)} 9%, var(--card-paper))"
  aria-hidden="true"
>
  {#if glyph}
    <!-- The glyph takes the card colour's paper ink: the band colour itself
         (yellow, grey) is too pale on the paper to name the category. -->
    <Icon
      name={categoryIcon(color)}
      class="text-[length:min(50cqh,50cqw)] text-(--glyph-ink) opacity-80 [--i2:color-mix(in_oklch,currentColor_45%,var(--card-paper))]"
    />
  {/if}
  <span
    class="pointer-events-none absolute inset-0 bg-[radial-gradient(closest-side,transparent_55%,var(--card-paper))] opacity-70"
  ></span>
</span>
