<script lang="ts">
  // One first-time tip (the hints design, "first-time-tips", M10.HS5): a
  // small callout beside the hint mark it explains, with "Got it" and "More
  // in Rules". It is a DismissLayer that keeps taps inside it, so its
  // buttons work and any tap elsewhere closes it and is swallowed, like a
  // tooltip; Escape closes it too.
  //
  // A leaf with plain props: the container (FirstTips.svelte) decides which
  // tip is due; the gallery renders it standalone. `anchor` is a CSS
  // selector of the mark: the panel sits above it (below when there is no
  // room), with a small arrow pointing at it, or centred when it is absent.
  import type { Attachment } from 'svelte/attachments';
  import Lightbulb from '@lucide/svelte/icons/lightbulb';
  import { t } from '$lib/i18n';
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
    /** A stable id for tests (data-first-tip). */
    id?: string;
  }

  let { open, text, anchor = null, onclose, onmore, id }: Props = $props();

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
        const r = box.getBoundingClientRect();
        const rem = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
        const gap = 0.5 * rem;
        const vw = document.documentElement.clientWidth;
        const vh = window.innerHeight;
        const width = Math.min(18 * rem, vw - 2 * gap);
        const height = panel.offsetHeight;
        const left = Math.min(Math.max(gap, r.left + r.width / 2 - width / 2), vw - width - gap);
        // Above the mark when it fits, so the card it sits on stays in view.
        const below = r.top - gap - height < gap && r.bottom + gap + height <= vh - gap;
        const top = below ? r.bottom + gap : Math.max(gap, r.top - gap - height);
        panel.style.inset = 'auto';
        panel.style.margin = '0';
        panel.style.top = `${top}px`;
        panel.style.left = `${left}px`; // physical-ok: viewport pixels from the anchor's box
        panel.style.width = `${width}px`;
        panel.dataset['side'] = below ? 'below' : 'above';
        // The arrow points at the mark's centre.
        const arrow = Math.min(Math.max(r.left + r.width / 2 - left, 12), width - 12);
        panel.style.setProperty('--tip-arrow', `${arrow - 6}px`); // the arrow's start edge
      };
      layout();
      // Again once the panel is shown and laid out (it has no height before).
      let frame = requestAnimationFrame(() => {
        layout();
        frame = requestAnimationFrame(layout);
      });
      // The board scrolls inside its own container on a short screen.
      window.addEventListener('resize', layout);
      window.addEventListener('scroll', layout, true);
      return () => {
        cancelAnimationFrame(frame);
        window.removeEventListener('resize', layout);
        window.removeEventListener('scroll', layout, true);
      };
    };
  }
</script>

<DismissLayer
  bind:open={() => open, (v) => !v && onclose()}
  keepInside
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
  </div>
</DismissLayer>

<style>
  /* A small arrow on the side of the mark, its start edge at --tip-arrow
     (viewport pixels, so physical). */
  :global(.first-tip[data-side])::before {
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
