<script lang="ts">
  // The opponent move summary (M10.S03b): a compact popup of what the other
  // side did since the viewer last moved, one line per move in order
  // (MoveSummaryLine), consolidated over their consecutive turns: a bot's
  // reply, both moves of an extra turn and the choices inside a turn are one
  // summary (lib/view/move-summary.ts). The game screen opens it after a
  // live frame hands the turn back (Settings, "Show opponent move in
  // detail"), and the top bar's button opens it at any time.
  //
  // It is the shell's DismissLayer: any tap anywhere closes it and is
  // swallowed, exactly like a tooltip, so it never acts on the board
  // beneath; Escape closes it too. It sits where it hides the least: given
  // an `anchor` (the structure on a phone, the player's own column on wide
  // screens), it is placed over the top of that box, below the strips;
  // without one (the gallery) it is centred. The `review` slot at its end
  // is for the move review line a later hints slice adds.
  //
  // A leaf: it shows the summary it is given and decides nothing.
  import type { Snippet } from 'svelte';
  import type { Attachment } from 'svelte/attachments';
  import { t } from '$lib/i18n';
  import type { MoveSummary } from '$lib/view/move-summary';
  import DismissLayer from '../shell/DismissLayer.svelte';
  import MoveSummaryLine from './MoveSummaryLine.svelte';

  interface Props {
    /** Whether the popup shows (it needs a summary too). */
    open: boolean;
    summary: MoveSummary | null;
    names: readonly [string, string];
    /** The box to place the popup over; centred in the viewport without one. */
    anchor?: HTMLElement | null;
    /** A tap or Escape closed it. */
    onclose: () => void;
    /** The move review line (a later hints slice); nothing for now. */
    review?: Snippet;
  }

  let { open, summary, names, anchor = null, onclose, review }: Props = $props();

  const title = $derived(summary ? t('game.summary.title', { name: names[summary.seat] }) : '');

  /**
   * Places the panel over the top of the anchor's box, in viewport pixels
   * (the panel is a popover in the top layer), and again on a resize. When
   * a modal dialog opened in the same update (a starter choice, the result
   * screen), the panel is shown again so it lands above it. DOM only.
   */
  function placeOver(box: HTMLElement | null): Attachment<HTMLElement> {
    return (content) => {
      const panel = content.closest<HTMLElement>('[data-dismiss-layer]');
      if (!panel) return;
      const layout = (): void => {
        if (!box || !box.isConnected) return;
        const r = box.getBoundingClientRect();
        const rem = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;
        const gap = 0.5 * rem;
        const vw = document.documentElement.clientWidth;
        const vh = window.innerHeight;
        const width = Math.min(Math.max(r.width - 2 * gap, 16 * rem), 24 * rem, vw - 2 * gap);
        const left = Math.min(Math.max(gap, r.left + (r.width - width) / 2), vw - width - gap);
        const top = Math.max(gap, r.top + gap);
        panel.style.inset = 'auto';
        panel.style.margin = '0';
        panel.style.top = `${top}px`;
        panel.style.left = `${left}px`; // physical-ok: viewport pixels from the anchor's box
        panel.style.width = `${width}px`;
        panel.style.maxHeight = `${Math.max(8 * rem, vh - top - gap)}px`;
      };
      layout();
      const frame = requestAnimationFrame(() => {
        if (document.querySelector('dialog:modal') && panel.matches(':popover-open')) {
          panel.hidePopover();
          panel.showPopover();
        }
        layout();
      });
      window.addEventListener('resize', layout);
      return () => {
        cancelAnimationFrame(frame);
        window.removeEventListener('resize', layout);
      };
    };
  }
</script>

{#if summary}
  <DismissLayer
    bind:open={() => open, (v) => !v && onclose()}
    label={title}
    class="move-summary w-80 overflow-y-auto overscroll-contain p-3"
  >
    <div
      class="flex flex-col gap-2"
      data-move-summary={summary.seat}
      data-plies={summary.plies.join(' ')}
      {@attach placeOver(anchor)}
    >
      <h2 class="font-display text-base leading-tight font-semibold text-fg" dir="auto">{title}</h2>
      <ol class="flex flex-col gap-2">
        {#each summary.lines as line, i (`${line.ply}:${i}`)}
          <MoveSummaryLine {line} seat={summary.seat} {names} />
        {/each}
      </ol>
      <div class="empty:hidden" data-summary-review>{@render review?.()}</div>
      <p class="text-xs text-fg-muted">{t('game.summary.close')}</p>
    </div>
  </DismissLayer>
{/if}
