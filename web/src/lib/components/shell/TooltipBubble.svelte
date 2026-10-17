<script lang="ts">
  // One tooltip bubble: the text, optional status lines, an optional
  // concept diagram, the "More in Rules" link when the tooltip names a rules
  // section, and the arrow that points at the anchor. A plain leaf: the
  // layer (TooltipLayer) positions it, passes the arrow's offset and handles
  // the link (`onrules`); the gallery renders it standalone. The box itself
  // is dir="ltr" because the arrow's offset is a physical distance from its
  // left edge; the text inside follows the page.
  //
  // The layer is pointer-events: none so a tooltip never blocks the board;
  // only the link row takes the pointer, so neighboring cards remain
  // reachable through the bubble body. ui/tooltip.svelte.ts keeps the link
  // open. The link is not in the tab order: keyboard users open the rules from the game menu.
  import type { Snippet } from 'svelte';
  import { i18n, t } from '$lib/i18n';
  import type { RuleSectionId } from '../rules/sections';

  interface Props {
    text: string;
    /** Status lines under the text (TooltipContent.lines). */
    lines?: readonly string[];
    diagram?: Snippet;
    /** Which side of the anchor the bubble is on; the arrow is on the other.
     *  The layer updates data-side itself once it has placed the bubble. */
    side?: 'top' | 'bottom';
    /** The arrow's centre, in px from the bubble's left edge. Hidden when
     *  undefined (a standalone specimen). */
    arrowX?: number;
    id?: string;
    /** The rules section of the "More in Rules" link; no link without it. */
    rulesId?: RuleSectionId;
    /** The link was clicked or tapped (the layer opens the rules). */
    onrules?: (section: RuleSectionId) => void;
    /** Must position the bubble (relative or absolute): the arrow hangs
     *  off its edge. */
    class?: string;
  }

  let {
    text,
    lines = [],
    diagram,
    side = 'top',
    arrowX,
    id,
    rulesId,
    onrules,
    class: className = 'relative'
  }: Props = $props();
</script>

<div
  {id}
  role="tooltip"
  dir="ltr"
  class="tooltip-bubble group/tip pointer-events-none w-max max-w-[min(17rem,calc(100vw-1rem))] rounded-control border border-rule-strong bg-panel px-2.5 py-1.5 text-[13px] leading-snug text-fg shadow-panel {className}"
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
    {#if lines.length > 0}
      <ul class="mt-1 flex flex-col gap-0.5 text-[12px] text-fg-muted" data-tooltip-lines>
        {#each lines as line, i (i)}
          <li>{line}</li>
        {/each}
      </ul>
    {/if}
    {#if diagram}
      <div class="mt-1.5 flex justify-center text-fg-muted" data-tooltip-diagram>
        {@render diagram()}
      </div>
    {/if}
    {#if rulesId}
      <button
        type="button"
        tabindex="-1"
        class="pointer-events-auto -mx-1 mt-1 inline-flex min-h-8 items-center gap-1 rounded-control px-1 text-[12px] font-semibold text-accent underline underline-offset-2 hover:bg-panel-sunken"
        data-tooltip-rules={rulesId}
        onclick={() => onrules?.(rulesId)}
      >
        {t('shell.tooltip.moreRules')}
      </button>
    {/if}
  </div>
</div>
