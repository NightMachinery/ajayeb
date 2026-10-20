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
  // beneath; Escape closes it too. On touch, a long-press on a picture or a
  // chip shows its tooltip instead and keeps the popup (longPressTips). It sits where it hides the least: given
  // an `anchor` (the structure on a phone, the player's own column on wide
  // screens), it is placed over the top of that box, below the strips;
  // given `lanes` (phone landscape), in the first lane clear of the cards to
  // take, the strips and the turn bar (ui/toast-lane.ts, summaryLane);
  // without either (the gallery) it is centred. When it is too short for
  // its lines it scrolls inside, and a fade at its foot shows that more lies
  // below, the card sheet's cue (shell/sheet-scroll.ts).
  //
  // After the moves come the viewer's own trade prices that those moves
  // changed (`prices`, the recap family: lib/view/recap.ts), and last the
  // `review` slot, where the game screen puts the move review of the
  // viewer's own move before these (MoveReview, the advisor family).
  //
  // A leaf: it shows the summary it is given and decides nothing.
  import type { Snippet } from 'svelte';
  import type { Attachment } from 'svelte/attachments';
  import { t } from '$lib/i18n';
  import type { MoveSummary } from '$lib/view/move-summary';
  import type { PriceTick } from '$lib/view/recap';
  import ChevronDown from '@lucide/svelte/icons/chevron-down';
  import ChevronUp from '@lucide/svelte/icons/chevron-up';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import { cardName, resourceName } from '../card/effect-text';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { summaryLane, type LaneTargets } from '$lib/ui/toast-lane';
  import DismissLayer from '../shell/DismissLayer.svelte';
  import { attachSheetScroll } from '../shell/sheet-scroll';
  import MoveSummaryLine from './MoveSummaryLine.svelte';

  interface Props {
    /** Whether the popup shows (it needs a summary too). */
    open: boolean;
    summary: MoveSummary | null;
    /** The moves arrived during a reload, reconnect or hidden page. */
    away?: boolean;
    names: readonly [string, string];
    /** The box to place the popup over; centred in the viewport without one. */
    anchor?: HTMLElement | null;
    /** The board to find a clear lane in (phone landscape); takes precedence over `anchor`. */
    lanes?: LaneTargets | null;
    /** A tap or Escape closed it. */
    onclose: () => void;
    /** The viewer's trade prices these moves changed (the recap family). */
    prices?: readonly PriceTick[];
    /** The move review of the viewer's own move before these (MoveReview). */
    review?: Snippet;
  }

  let {
    open,
    summary,
    names,
    away = false,
    prices = [],
    anchor = null,
    lanes = null,
    onclose,
    review
  }: Props = $props();

  function priceText(p: PriceTick): string {
    const resource = resourceName(p.resource);
    return p.cause
      ? t('hints.recap.priceBuilt', {
          resource,
          from: p.from,
          count: p.to,
          name: names[p.cause.seat],
          card: cardName(p.cause.card)
        })
      : t('hints.recap.price', { resource, from: p.from, count: p.to });
  }

  function resourceIcon(p: PriceTick): IconName {
    return `resource-${p.resource}-sm` as IconName;
  }

  /** Marks the panel (the scroller) data-scroll-more while lines lie below its foot. */
  const scrollCue: Attachment<HTMLElement> = (fade) => {
    const panel = fade.closest<HTMLElement>('[data-dismiss-layer]');
    return panel ? attachSheetScroll(panel) : undefined;
  };

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
    longPressTips
    class="move-summary w-80 overflow-y-auto overscroll-contain p-3"
  >
    <div
      class="flex flex-col gap-2"
      data-move-summary={summary.seat}
      data-plies={summary.plies.join(' ')}
      {@attach lanes ? summaryLane(() => lanes) : placeOver(anchor)}
    >
      <h2 class="font-display text-base leading-tight font-semibold text-fg">{title}</h2>
      {#if away}<p class="text-xs font-semibold text-fg-muted" data-summary-away>
          {t('game.summary.away')}
        </p>{/if}
      <ol class="flex flex-col gap-2">
        {#each summary.lines as line, i (`${line.ply}:${i}`)}
          <MoveSummaryLine {line} seat={summary.seat} {names} />
        {/each}
      </ol>
      {#if prices.length > 0}
        <section class="flex flex-col gap-1" data-summary-prices>
          <h3
            class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
            {@attach tooltip({ text: t('hints.recap.tip') })}
          >
            {t('hints.recap.title')}
          </h3>
          <ul class="flex flex-col gap-1">
            {#each prices as p (p.resource)}
              <li
                class="flex items-start gap-1.5 text-sm leading-snug text-fg"
                data-price-tick={p.resource}
                data-price-dir={p.to > p.from ? 'up' : 'down'}
              >
                <Icon name={resourceIcon(p)} class="mt-0.5 shrink-0" />
                {#if p.to > p.from}
                  <ChevronUp class="mt-0.5 size-4 shrink-0 text-fg" aria-hidden="true" />
                {:else}
                  <ChevronDown class="mt-0.5 size-4 shrink-0 text-fg" aria-hidden="true" />
                {/if}
                <span>{priceText(p)}</span>
              </li>
            {/each}
          </ul>
        </section>
      {/if}
      <div class="empty:hidden" data-summary-review>{@render review?.()}</div>
      <p class="text-xs text-fg-muted">{t('game.summary.close')}</p>
    </div>
    <div class="summary-more" aria-hidden="true" {@attach scrollCue}></div>
  </DismissLayer>
{/if}

<style>
  /* The fade over the last visible lines while the panel has more below
     (data-scroll-more, set by attachSheetScroll), like the card sheet's.
     Sticky at the scroller's foot, it takes no room of its own. */
  .summary-more {
    position: sticky;
    inset-block-end: 0;
    height: 1.5rem;
    margin-block-start: -1.5rem;
    pointer-events: none;
    background: linear-gradient(transparent, var(--panel));
    opacity: 0;
  }
  :global([data-scroll-more]) > .summary-more {
    opacity: 1;
  }
</style>
