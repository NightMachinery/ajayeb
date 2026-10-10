<script lang="ts">
  // One colour's worth of a city, as a band-tier stack: the UX design's
  // "band" tier shows only colour, pattern and the category glyph (no
  // names, no per-card detail -- those need CardFace, which M5.S02 owns and
  // this slice does not import). A simple local stand-in: one tick per
  // card, a category icon and a count, legible well under the mini-card
  // widths the real CardFace hits on a phone.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import type { Color } from '$lib/wire';

  interface Props {
    color: Color;
    count: number;
    class?: string;
  }

  let { color, count, class: className = '' }: Props = $props();

  const CATEGORY_ICON: Record<Color, IconName> = {
    brown: 'category-brown',
    grey: 'category-grey',
    blue: 'category-blue',
    green: 'category-green',
    yellow: 'category-yellow',
    red: 'category-red',
    purple: 'category-purple'
  };

  const BAND_CLASS: Record<Color, string> = {
    brown: 'bg-card-brown',
    grey: 'bg-card-grey',
    blue: 'bg-card-blue',
    green: 'bg-card-green',
    yellow: 'bg-card-yellow',
    red: 'bg-card-red',
    purple: 'bg-card-purple'
  };

  // The glyph sits on the panel, so it takes the theme-aware card ink (the
  // band colours fall under 3:1 there: yellow and grey on paper, brown and
  // purple on the lapis panels). The ticks are fills and keep the band colour.
  const TEXT_CLASS: Record<Color, string> = {
    brown: 'text-card-brown-ink',
    grey: 'text-card-grey-ink',
    blue: 'text-card-blue-ink',
    green: 'text-card-green-ink',
    yellow: 'text-card-yellow-ink',
    red: 'text-card-red-ink',
    purple: 'text-card-purple-ink'
  };

  const ticks = $derived(Array.from({ length: Math.min(count, 10) }, (_unused, i) => i));

  function colorKey(color: Color) {
    switch (color) {
      case 'brown':
        return 'board.color.brown' as const;
      case 'grey':
        return 'board.color.grey' as const;
      case 'blue':
        return 'board.color.blue' as const;
      case 'green':
        return 'board.color.green' as const;
      case 'yellow':
        return 'board.color.yellow' as const;
      case 'red':
        return 'board.color.red' as const;
      case 'purple':
        return 'board.color.purple' as const;
    }
  }
</script>

{#if count > 0}
  <div
    class="city-band flex items-center gap-1 rounded-control border border-rule bg-panel py-1 ps-1.5 pe-2 {className}"
    title={t('board.city.colorCount', { count, color: t(colorKey(color)) })}
  >
    <Icon name={CATEGORY_ICON[color]} class="shrink-0 {TEXT_CLASS[color]}" />
    <span class="flex gap-0.5" aria-hidden="true">
      {#each ticks as tick (tick)}
        <span class="h-3 w-1.5 shrink-0 rounded-xs {BAND_CLASS[color]}"></span>
      {/each}
    </span>
    <span class="text-xs font-semibold text-fg-muted tabular-nums" data-must-fit>{count}</span>
  </div>
{/if}
