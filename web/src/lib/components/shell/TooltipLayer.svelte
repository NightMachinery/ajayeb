<script lang="ts">
  // The one tooltip layer, mounted once in the root layout. It installs the
  // gestures (ui/tooltip.svelte.ts) and draws the open tooltip: the bubble
  // with its arrow, and the stack with its connector when the tooltip has
  // one. One tooltip at a time.
  //
  // The layer is a manual popover, so it sits in the top layer above an
  // open sheet (a modal dialog) as well; it is shown again each time a new
  // tooltip opens, which puts it above any dialog opened since. Its boxes
  // are placed in viewport pixels (ui/tooltip-place.ts) inside a dir="ltr"
  // overlay; the text in them follows the page's direction. The probe below
  // measures the viewport less the safe-area insets, which the tooltip
  // never crosses.
  import { tooltips, installTooltipGestures, TOOLTIP_ID } from '$lib/ui/tooltip.svelte';
  import { openRules } from '../rules/rules-overlay.svelte';
  import type { RuleSectionId } from '../rules/sections';
  import { placeTooltip } from '$lib/ui/tooltip-place';
  import type { Attachment } from 'svelte/attachments';
  import TooltipBubble from './TooltipBubble.svelte';
  import TooltipStack from './TooltipStack.svelte';

  /** The stack goes beside the bubble from this viewport width up. */
  const BESIDE_MIN_WIDTH = 640;

  let popEl: HTMLDivElement | undefined = $state();
  let probeEl: HTMLDivElement | undefined = $state();

  const active = $derived(tooltips.active);

  /** The bubble's "More in Rules" link: close the tooltip, open the rules. */
  function moreRules(section: RuleSectionId): void {
    tooltips.hide();
    void openRules(section);
  }

  // DOM listeners only; they write the store from events, never here.
  $effect(() => installTooltipGestures());

  // DOM-only: hide the popover when no tooltip is open. Showing it is the
  // placing attachment's job, so the boxes are measured in the open popover.
  $effect(() => {
    const el = popEl;
    if (!el || typeof el.hidePopover !== 'function') return;
    if (!active && el.matches(':popover-open')) el.hidePopover();
  });

  /** Places the group's boxes once they are measured, and again whenever
   *  one of them changes size. Writes styles only, never state. */
  const place: Attachment<HTMLElement> = (group) => {
    const tip = active;
    if (!tip) return;
    const bubble = group.querySelector<HTMLElement>('[data-tooltip-bubble]');
    const stack = group.querySelector<HTMLElement>('[data-tooltip-stack]');
    const arrow = bubble?.querySelector<SVGElement>('.arrow');
    const connector = group.querySelector<HTMLElement>('[data-tooltip-connector]');
    if (!bubble) return;

    // Show the popover (again, so it lands above any dialog opened since)
    // before measuring: a closed popover is display: none, and has no size.
    const pop = popEl;
    if (pop && typeof pop.showPopover === 'function') {
      if (pop.matches(':popover-open')) pop.hidePopover();
      pop.showPopover();
    }

    const layout = (): void => {
      // Not laid out yet (no size): the resize observer calls again.
      if (bubble.offsetWidth === 0 || (stack && stack.offsetWidth === 0)) return;
      const safe = probeEl?.getBoundingClientRect();
      const bounds = safe
        ? { x: safe.x, y: safe.y, width: safe.width, height: safe.height }
        : { x: 0, y: 0, width: window.innerWidth, height: window.innerHeight };
      // The anchor's box now: a scroll moves the tooltip with its anchor.
      const r = tip.anchor.getBoundingClientRect();
      const p = placeTooltip({
        anchor: { x: r.x, y: r.y, width: r.width, height: r.height },
        bubble: { width: bubble.offsetWidth, height: bubble.offsetHeight },
        stack: stack ? { width: stack.offsetWidth, height: stack.offsetHeight } : undefined,
        bounds,
        allowBeside: window.innerWidth >= BESIDE_MIN_WIDTH
      });
      bubble.style.transform = `translate(${p.bubble.x}px, ${p.bubble.y}px)`;
      bubble.dataset['side'] = p.side;
      if (arrow) arrow.style.insetInlineStart = `${p.arrowX - 8}px`;
      if (stack && p.stack) {
        stack.style.transform = `translate(${p.stack.x}px, ${p.stack.y}px)`;
        stack.dataset['mode'] = p.stack.mode;
      }
      if (connector && p.connector) {
        const { x1, y1, x2, y2 } = p.connector;
        const vertical = x1 === x2;
        const x = Math.min(x1, x2);
        const y = Math.min(y1, y2);
        connector.style.transform = `translate(${vertical ? x - 1 : x}px, ${vertical ? y : y - 1}px)`;
        connector.style.width = vertical ? '2px' : `${Math.abs(x2 - x1)}px`;
        connector.style.height = vertical ? `${Math.abs(y2 - y1)}px` : '2px';
        if (vertical) connector.dataset['vertical'] = '';
        else delete connector.dataset['vertical'];
      }
      group.dataset['placed'] = '';
    };

    layout();
    const observer = new ResizeObserver(layout);
    observer.observe(bubble);
    if (stack) observer.observe(stack);
    // A scroll carries the tooltip along with its anchor; once the anchor
    // has left the viewport, the tooltip closes.
    const onScroll = (): void => {
      const r = tip.anchor.getBoundingClientRect();
      const gone =
        r.bottom <= 0 || r.top >= window.innerHeight || r.right <= 0 || r.left >= window.innerWidth;
      if (gone) tooltips.hide();
      else layout();
    };
    window.addEventListener('scroll', onScroll, { capture: true, passive: true });
    return () => {
      observer.disconnect();
      window.removeEventListener('scroll', onScroll, { capture: true });
    };
  };
</script>

<!-- The safe area, measured: the viewport less the safe-area insets, which
     are physical by nature (top, right, bottom, left). -->
<div
  bind:this={probeEl}
  class="tooltip-probe pointer-events-none invisible fixed"
  style:inset="var(--safe-t) var(--safe-r) var(--safe-b) var(--safe-l)"
  aria-hidden="true"
></div>

<div
  bind:this={popEl}
  popover="manual"
  dir="ltr"
  class="tooltip-layer pointer-events-none fixed inset-0 z-[90] m-0 size-full max-h-none max-w-none overflow-visible border-0 bg-transparent p-0"
  data-tooltip-layer
>
  {#if active}
    {#key active.id}
      <div class="tooltip-group" {@attach place} data-tooltip-open={active.via}>
        {#if active.content.stack && active.content.stack.length > 0}
          <span
            class="connector absolute start-0 top-0 rounded-full bg-(--gold-deep)"
            data-tooltip-connector
          ></span>
          <TooltipStack items={active.content.stack} class="absolute start-0 top-0" />
        {/if}
        <TooltipBubble
          id={TOOLTIP_ID}
          text={active.content.text}
          lines={active.content.lines}
          diagram={active.content.diagram}
          rulesId={active.content.rulesId}
          onrules={moreRules}
          arrowX={0}
          class="absolute start-0 top-0"
        />
      </div>
    {/key}
  {/if}
</div>

<style>
  /* data-placed is set by the placing attachment, so Svelte cannot see it
     (hence :global). Hidden until placed, so it never flashes at the corner; then a fade
     that follows the Animation setting (instant when it is off, through
     app.css's data-motion rule and --dur-fast). */
  .tooltip-group:not(:global([data-placed])) {
    visibility: hidden;
  }
  .tooltip-group:global([data-placed]) {
    animation: tooltip-in var(--dur-fast) ease-out;
  }
  @keyframes tooltip-in {
    from {
      opacity: 0;
    }
  }
  /* A long-press on an anchor shows its tooltip, so it must not also
     select text or open the browser's callout. */
  :global([data-tooltip]) {
    -webkit-touch-callout: none;
    -webkit-user-select: none;
    user-select: none;
  }
  .tooltip-layer:popover-open {
    display: block;
  }
  .connector::before,
  .connector::after {
    content: '';
    position: absolute;
    width: 6px;
    height: 6px;
    border-radius: 9999px;
    background: var(--gold-deep);
    inset-inline-start: -2px;
    inset-block-start: -3px;
  }
  .connector::after {
    inset-block-start: auto;
    inset-block-end: -3px;
  }
  .connector:not([data-vertical])::before {
    inset-inline-start: -3px;
    inset-block-start: -2px;
  }
  .connector:not([data-vertical])::after {
    inset-inline-start: auto;
    inset-inline-end: -3px;
    inset-block-start: -2px;
    inset-block-end: auto;
  }
</style>
