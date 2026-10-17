<script lang="ts">
  // The tutorial's coach mark (the tutorial design, section 2.4): a bubble
  // with an arrow that points at the coach step's target on the board,
  // with the step's text, "Lesson 3 of 10", Next (info steps only) and
  // "Play on my own".
  //
  // It is a manual popover, so it sits in the top layer, and it is not
  // modal: it installs no dismiss handler, swallows no tap and takes the
  // pointer only on the bubble itself, so a tap on the target, or on any
  // other card, works as usual. That is why it is not a DismissLayer (the
  // first-time tip is one, and would eat the very tap the step asks for).
  //
  // It is placed in viewport pixels inside a dir="ltr" overlay, as the
  // tooltip layer is (lib/tutorial/place.ts on ui/tooltip-place.ts), with
  // the text in the locale's direction: above or below the target, never
  // over an element target and over at most 40% of a region target; when
  // neither side has room it is pinned to the viewport edge farther from
  // the target, with no arrow. It is at most min(20rem, 100vw - 1rem) wide;
  // on a short screen its text is one size smaller and its body scrolls
  // inside 40% of the viewport's height. It re-measures on resize, on
  // scroll and when the target or the bubble changes size.
  //
  // A leaf with plain props: the container (CoachLayer.svelte) decides the
  // step and finds the target; the gallery renders it standalone with any
  // element as the anchor.
  import type { Attachment } from 'svelte/attachments';
  import GraduationCap from '@lucide/svelte/icons/graduation-cap';
  import { i18n, t } from '$lib/i18n';
  import { placeCoach } from '$lib/tutorial/place';
  import type { CoachKind } from '$lib/tutorial/steps';

  interface Props {
    open: boolean;
    /** The element the bubble points at; centred at the top without one. */
    anchor: HTMLElement | null;
    /** "Tutorial". */
    title: string;
    /** "Lesson 3 of 10", or null outside the guided part. */
    progress?: string | null;
    text: string;
    kind: CoachKind;
    /** The lesson id (data-coach-step). */
    step: string;
    /** The guided turn (data-coach-turn). */
    turn?: number;
    /** The target is a region the bubble may partly cover. */
    region?: boolean;
    /** Next, on an info step. */
    onnext?: () => void;
    /** Play on my own. */
    onpause?: () => void;
    /** The text's id, for the target's aria-describedby. */
    textId?: string;
    /** Next's element, for the container's quiet focus. */
    nextEl?: HTMLButtonElement | undefined;
  }

  let {
    open,
    anchor,
    title,
    progress = null,
    text,
    kind,
    step,
    turn = 0,
    region = false,
    onnext,
    onpause,
    textId,
    nextEl = $bindable()
  }: Props = $props();

  const uid = $props.id();
  const titleId = `${uid}-title`;

  let popEl: HTMLDivElement | undefined = $state();
  let probeEl: HTMLDivElement | undefined = $state();

  // DOM-only: show the popover while open (again for a new step, so it
  // lands above anything shown since), hide it when closed.
  $effect(() => {
    const el = popEl;
    if (!el || typeof el.showPopover !== 'function') return;
    if (open && !el.matches(':popover-open')) el.showPopover();
    if (!open && el.matches(':popover-open')) el.hidePopover();
  });

  /** Places the bubble next to the anchor; writes styles only, never state. */
  function place(target: HTMLElement | null): Attachment<HTMLElement> {
    return (bubble) => {
      const arrow = bubble.querySelector<HTMLElement>('[data-coach-arrow]');
      const layout = (): void => {
        if (bubble.offsetWidth === 0) return;
        const safe = probeEl?.getBoundingClientRect();
        const bounds = safe
          ? { x: safe.x, y: safe.y, width: safe.width, height: safe.height }
          : { x: 0, y: 0, width: window.innerWidth, height: window.innerHeight };
        const size = { width: bubble.offsetWidth, height: bubble.offsetHeight };
        const r = target?.getBoundingClientRect();
        const p = r
          ? placeCoach({
              anchor: { x: r.x, y: r.y, width: r.width, height: r.height },
              bubble: size,
              bounds,
              region
            })
          : {
              placement: 'pinned' as const,
              x: bounds.x + Math.max(8, (bounds.width - size.width) / 2),
              y: bounds.y + 8
            };
        bubble.style.transform = `translate(${Math.round(p.x)}px, ${Math.round(p.y)}px)`;
        bubble.dataset['coachPlacement'] = p.placement;
        if (arrow) {
          if (p.placement === 'pinned' || p.arrowX === undefined) {
            arrow.hidden = true;
          } else {
            arrow.hidden = false;
            arrow.style.insetInlineStart = `${p.arrowX - 8}px`;
          }
        }
        bubble.dataset['placed'] = '';
      };
      layout();
      const observer = new ResizeObserver(layout);
      observer.observe(bubble);
      if (target) observer.observe(target);
      let frame = requestAnimationFrame(() => {
        layout();
        frame = requestAnimationFrame(layout);
      });
      window.addEventListener('resize', layout);
      window.addEventListener('scroll', layout, { capture: true, passive: true });
      return () => {
        cancelAnimationFrame(frame);
        observer.disconnect();
        window.removeEventListener('resize', layout);
        window.removeEventListener('scroll', layout, { capture: true });
      };
    };
  }
</script>

<!-- The safe area, measured: the viewport less the safe-area insets, which
     are physical by nature (top, right, bottom, left). -->
<div
  bind:this={probeEl}
  class="pointer-events-none invisible fixed"
  style:inset="var(--safe-t) var(--safe-r) var(--safe-b) var(--safe-l)"
  aria-hidden="true"
></div>

<div
  bind:this={popEl}
  popover="manual"
  dir="ltr"
  class="coach-layer pointer-events-none fixed inset-0 z-[80] m-0 size-full max-h-none max-w-none overflow-visible border-0 bg-transparent p-0"
  data-coach-layer
>
  {#if open}
    {#key `${step}:${kind}:${text}`}
      <div
        class="coach-bubble group/coach pointer-events-auto absolute start-0 top-0 flex w-[min(20rem,calc(100vw-1rem))] flex-col rounded-panel border border-rule-strong bg-panel text-fg shadow-panel short:w-[min(28rem,calc(100vw-1rem))]"
        role="dialog"
        aria-modal="false"
        aria-labelledby={titleId}
        data-coach
        data-coach-step={step}
        data-coach-kind={kind}
        data-coach-turn={turn}
        data-coach-placement="below"
        {@attach place(anchor)}
      >
        <svg
          class="pointer-events-none absolute top-full -mt-px h-2 w-4 overflow-visible
            group-data-[coach-placement=below]/coach:top-auto group-data-[coach-placement=below]/coach:bottom-full
            group-data-[coach-placement=below]/coach:mt-0 group-data-[coach-placement=below]/coach:-mb-px
            group-data-[coach-placement=below]/coach:rotate-180"
          viewBox="0 0 16 8"
          aria-hidden="true"
          data-coach-arrow
        >
          <path d="M0 0 L8 8 L16 0" fill="var(--panel)" stroke="var(--rule-strong)" />
        </svg>
        <div dir={i18n.dir} class="flex min-h-0 flex-col gap-2 p-3 short:gap-1.5 short:p-2.5">
          <p
            id={titleId}
            class="flex items-center gap-1.5 text-xs font-semibold tracking-wide text-accent uppercase"
          >
            <GraduationCap class="size-4 shrink-0" aria-hidden="true" />
            <span>{title}</span>
            {#if progress}
              <span class="font-normal tracking-normal text-fg-muted normal-case">· {progress}</span
              >
            {/if}
          </p>
          {#if text}
            <p
              id={textId}
              class="coach-text overflow-y-auto text-sm leading-snug text-fg short:text-xs"
              data-coach-text
            >
              {text}
            </p>
          {/if}
          <div class="flex flex-wrap items-center justify-end gap-2">
            {#if onpause}
              <button
                type="button"
                class="min-h-11 rounded-control px-3 text-sm font-semibold text-accent hover:bg-panel-sunken"
                data-coach-pause
                onclick={onpause}
              >
                {t('tutorial.coach.pause')}
              </button>
            {/if}
            {#if kind === 'info' && onnext}
              <button
                type="button"
                bind:this={nextEl}
                class="min-h-11 rounded-control bg-accent px-4 text-sm font-semibold text-on-accent"
                data-coach-next
                onclick={onnext}
              >
                {t('tutorial.coach.next')}
              </button>
            {/if}
          </div>
        </div>
      </div>
    {/key}
  {/if}
</div>

<style>
  /* On a short screen (a phone held sideways) the bubble is wider, so the
     text takes fewer lines, and a longer text scrolls inside it: the bubble
     stays under about half the viewport's height. */
  @media (max-height: 32rem) {
    .coach-text {
      max-height: calc(50vh - 6.5rem);
    }
  }
  .coach-bubble:not([data-placed]) {
    visibility: hidden;
  }
</style>
