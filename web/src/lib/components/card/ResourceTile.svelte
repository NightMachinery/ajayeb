<script lang="ts">
  // One resource as a small coloured tile with its glyph: wood, clay, stone,
  // glass or papyrus. The tile colour and the glyph shape both identify the
  // resource, so it reads in greyscale too. Decorative (aria-hidden).
  // `tip` gives it a tooltip of its own; inside a cost row or an effect
  // glyph the surrounding group carries the tooltip instead.
  import Icon from '$lib/icons/Icon.svelte';
  import { tooltip, type TooltipContent } from '$lib/ui/tooltip.svelte';
  import type { Resource } from '$lib/wire';
  import { resourceIcon, resourceTile } from './card-style';

  interface Props {
    resource: Resource;
    /** Use the small-grid glyph (for a tile under about 14 px). */
    small?: boolean;
    /** Its own tooltip, when no surrounding element has one. */
    tip?: TooltipContent;
    class?: string;
  }

  let { resource, small = false, tip, class: className = '' }: Props = $props();

  const tile = $derived(resourceTile(resource));
</script>

<span
  class="inline-grid shrink-0 place-items-center rounded-[0.25em] {small
    ? 'size-[1em]'
    : 'size-[1.2em]'} {className}"
  style:background={tile.bg}
  style:color={tile.ink}
  style:--i2="color-mix(in oklch, {tile.ink} 55%, {tile.bg})"
  aria-hidden="true"
  {@attach tooltip(tip)}
>
  <Icon name={resourceIcon(resource, small)} class="text-[0.9em]" />
</span>
