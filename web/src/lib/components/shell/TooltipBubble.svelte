<script lang="ts">
  // One tooltip bubble: the text, an optional concept diagram, and the arrow
  // that points at the anchor. A plain leaf: the layer (TooltipLayer)
  // positions it and passes the arrow's offset; the gallery renders it
  // standalone. The box itself is dir="ltr" because the arrow's offset is a
  // physical distance from its left edge; the text inside follows the page.
  import type { Snippet } from 'svelte';
  import { i18n } from '$lib/i18n';

  interface Props {
    text: string;
    diagram?: Snippet;
    /** Which side of the anchor the bubble is on; the arrow is on the other.
     *  The layer updates data-side itself once it has placed the bubble. */
    side?: 'top' | 'bottom';
    /** The arrow's centre, in px from the bubble's left edge. Hidden when
     *  undefined (a standalone specimen). */
    arrowX?: number;
    id?: string;
    rulesId?: string;
    /** Must position the bubble (relative or absolute): the arrow hangs
     *  off its edge. */
    class?: string;
  }

  let {
    text,
    diagram,
    side = 'top',
    arrowX,
    id,
    rulesId,
    class: className = 'relative'
  }: Props = $props();
</script>

<div
  {id}
  role="tooltip"
  dir="ltr"
  class="tooltip-bubble group/tip w-max max-w-[min(17rem,calc(100vw-1rem))] rounded-control border border-rule-strong bg-panel px-2.5 py-1.5 text-[13px] leading-snug text-fg shadow-panel {className}"
  data-tooltip-bubble
  data-side={side}
  data-rules-id={rulesId}
>
  {#if arrowX !== undefined}
    <!-- The arrow overlaps the border by one pixel, so the two read as one
         outline. -->
    <svg
      class="arrow pointer-events-none absolute top-full -mt-px h-2 w-4 overflow-visible
        group-data-[side=bottom]/tip:top-auto group-data-[side=bottom]/tip:bottom-full
        group-data-[side=bottom]/tip:mt-0 group-data-[side=bottom]/tip:-mb-px group-data-[side=bottom]/tip:rotate-180"
      style:inset-inline-start="{arrowX - 8}px"
      viewBox="0 0 16 8"
      aria-hidden="true"
    >
      <path d="M0 0 L8 8 L16 0" fill="var(--panel)" stroke="var(--rule-strong)" />
    </svg>
  {/if}
  <div dir={i18n.dir} class="text-start">
    <p>{text}</p>
    {#if diagram}
      <div class="mt-1.5 flex justify-center text-fg-muted" data-tooltip-diagram>
        {@render diagram()}
      </div>
    {/if}
  </div>
</div>
