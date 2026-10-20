<script lang="ts">
  // One first-time tip (the hints design, "first-time-tips", M10.HS5): a
  // small callout beside the hint mark it explains, with "Got it" and "More
  // in Rules". It is a DismissLayer that keeps taps inside it, so its
  // buttons work. On touch an enabled action outside closes the tip and
  // keeps that action; other outside taps are swallowed. Tooltips keep
  // their stricter any-tap-swallowed rule. Escape closes this tip too.
  //
  // A leaf with plain props: the container (FirstTips.svelte) decides which
  // tip is due; the gallery renders it standalone. `anchor` is a CSS
  // selector of the mark: the panel sits above it (below when there is no
  // room), with a small arrow pointing at it, or centred when it is absent.
  // It keeps clear of the cards to take and the strips where it can
  // (placeTip's obstacles).
  import type { Attachment } from 'svelte/attachments';
  import Lightbulb from '@lucide/svelte/icons/lightbulb';
  import { t } from '$lib/i18n';
  import { placeTip } from '$lib/ui/tips.svelte';
  import DismissLayer from '../shell/DismissLayer.svelte';

  interface Props {
    open: boolean;
    /** The tip's text, already translated. */
    text: string;
    /** A selector of the mark it explains; centred without one. */
    anchor?: string | null;
    /** Closed by Got it, a tap elsewhere or Escape. */
    onclose: () => void;
    /** "More in Rules"; the link is left out without it. */
    onmore?: () => void;
    /** Turns all first-time tips off on this device. */
    onoff?: () => void;
    /** A stable id for tests (data-first-tip). */
    id?: string;
  }

  let { open, text, anchor = null, onclose, onmore, onoff, id }: Props = $props();

  /** Places the panel next to the anchor's box, in viewport pixels. DOM only. */
  function placeNear(selector: string | null): Attachment<HTMLElement> {
    return (content) => {
      // Looked up on each layout: the attachment may run before the content
      // is inside the panel, which the next frame's layout then finds.
      const layout = (): void => {
        const panel = content.closest<HTMLElement>('[data-dismiss-layer]');
        if (!panel) return;
        const box = selector ? document.querySelector<HTMLElement>(selector) : null;
        if (!box) return;
        const card = box.closest<HTMLElement>('.card-structure [data-slot], [data-card]');
        const r = box.getBoundingClientRect();
        const cr = (card ?? box).getBoundingClientRect();
        const rem = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
        const vw = document.documentElement.clientWidth;
        const vh = window.innerHeight;
        // Kept clear where it can: the cards to take and both strips, but
        // not the strip the mark itself sits in.
        const obstacles = [
          ...document.querySelectorAll('.card-structure [data-accessible], .player-strip')
        ]
          .filter((el) => !el.contains(box))
          .map((el) => el.getBoundingClientRect());
        const placed = placeTip({
          mark: r,
          card: cr,
          panelHeight: panel.offsetHeight,
          vw,
          vh,
          rem,
          obstacles
        });
        panel.style.inset = 'auto';
        panel.style.margin = '0';
        panel.style.top = `${placed.top}px`;
        panel.style.left = `${placed.left}px`; // physical-ok: viewport pixels from the anchor's box
        panel.style.width = `${placed.width}px`;
        panel.dataset['side'] = placed.side;
        // The arrow points at the mark's centre.
        panel.style.setProperty('--tip-arrow', `${placed.arrow}px`); // the arrow's start edge
      };
      layout();
      const observer =
        typeof ResizeObserver !== 'undefined' ? new ResizeObserver(layout) : undefined;
      // Again once the panel is shown and laid out (it has no height before).
      let frame = requestAnimationFrame(() => {
        const panel = content.closest<HTMLElement>('[data-dismiss-layer]');
        if (panel) observer?.observe(panel);
        layout();
        frame = requestAnimationFrame(layout);
      });
      // The board scrolls inside its own container on a short screen.
      window.addEventListener('resize', layout);
      window.addEventListener('scroll', layout, true);
      return () => {
        cancelAnimationFrame(frame);
        observer?.disconnect();
        window.removeEventListener('resize', layout);
        window.removeEventListener('scroll', layout, true);
      };
    };
  }
</script>

<DismissLayer
  bind:open={() => open, (v) => !v && onclose()}
  keepInside
  passTouchActions
  role="note"
  label={t('hints.tip.label')}
  class="first-tip w-72 overflow-visible p-3"
>
  <div class="flex flex-col gap-2" data-first-tip={id} {@attach placeNear(anchor)}>
    <p class="flex items-center gap-1.5 text-xs font-semibold tracking-wide text-accent uppercase">
      <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
      {t('hints.tip.label')}
    </p>
    <p class="text-sm leading-snug text-fg">{text}</p>
    <div class="flex items-center justify-end gap-2">
      {#if onmore}
        <button
          type="button"
          class="min-h-11 rounded-control px-3 text-sm font-semibold text-accent hover:bg-panel-sunken"
          data-tip-more
          onclick={() => {
            onclose();
            onmore?.();
          }}
        >
          {t('hints.tip.more')}
        </button>
      {/if}
      <button
        type="button"
        class="min-h-11 rounded-control bg-accent px-4 text-sm font-semibold text-on-accent"
        data-tip-ok
        onclick={onclose}
      >
        {t('hints.tip.gotIt')}
      </button>
    </div>
    <div class="flex items-center justify-between gap-2 border-t border-rule pt-1.5">
      <span class="text-xs leading-tight text-fg-muted" data-tip-off-hint>
        {t('hints.tip.offHint')}
      </span>
      <button
        type="button"
        class="min-h-9 shrink-0 rounded-control px-2.5 text-xs font-semibold text-fg-muted underline decoration-rule-strong underline-offset-2 hover:bg-panel-sunken hover:text-fg"
        data-tip-off
        onclick={() => {
          if (onoff) onoff();
          else onclose();
        }}
      >
        {t('hints.tip.off')}
      </button>
    </div>
  </div>
</DismissLayer>

<style>
  /* A small arrow on the side of the mark, its start edge at --tip-arrow
     (viewport pixels, so physical). */
  :global(.first-tip[data-side='above'])::before,
  :global(.first-tip[data-side='below'])::before {
    content: '';
    position: absolute;
    left: var(--tip-arrow, 50%); /* physical-ok: matches the panel's viewport placement */
    width: 12px;
    height: 12px;
    background: var(--panel);
    border: 1px solid var(--rule-strong);
    transform: rotate(45deg);
  }
  :global(.first-tip[data-side='below'])::before {
    top: -7px;
    border-right: none; /* physical-ok: the arrow turns with the viewport placement */
    border-bottom: none;
  }
  :global(.first-tip[data-side='above'])::before {
    bottom: -7px;
    border-left: none; /* physical-ok: the arrow turns with the viewport placement */
    border-top: none;
  }
</style>
