<script lang="ts">
  // The game screen's coach layer (the tutorial design, sections 2.4 and
  // 3): it shows the coach step the game screen computed (lib/tutorial/
  // steps.ts over the presented frame's tutorial object), as a bubble on the
  // board (CoachMark) and as the ring on its target. Inside a modal sheet
  // the sheet itself shows the step as a note (CoachNote); the layer still
  // rings the target there.
  //
  // When it shows: the caller's `active` (the player's own turn on the
  // frame on screen, the Presenter idle, the move summary closed, the
  // tutorial not paused here), then two animation frames later, so the
  // target's box is final. The bubble hides while any modal is open (a
  // sheet, the rules on a phone, the forfeit confirm, a popup): the page
  // behind a modal dialog is inert. A DOM observer tells when one opens.
  //
  // The ring is the attribute data-coach-target, written on exactly one
  // element (with data-coach-region on a region) and removed when the step
  // changes; the global style below draws it. The target also gets
  // aria-describedby pointing at the step's text. On an info step the focus
  // moves quietly to Next, on a tap step to the target; it goes back where
  // it was when the bubble goes. A polite live region announces each new
  // step's text once.
  //
  // Effects here only touch the DOM or schedule frames; state is written
  // from those frame and observer callbacks, never during an effect's run.
  import { onDestroy } from 'svelte';
  import { restoreFocus } from '$lib/ui/dismiss';
  import { findAnchor, revealInScroller } from '$lib/tutorial/anchors';
  import { COACH_TEXT_ID, setCoachSource } from '$lib/tutorial/coach.svelte';
  import type { CoachStep } from '$lib/tutorial/steps';
  import type { CoachText } from '$lib/tutorial/text';
  import { modalOpen } from '../game/undo-keys';
  import CoachMark from './CoachMark.svelte';

  interface Props {
    /** The step due now, or null. */
    step: CoachStep | null;
    /** Its words. */
    text: CoachText | null;
    /** Whether a step may show now (see above). */
    active: boolean;
    /** Next, on an info step. */
    onnext: (step: CoachStep) => void;
    /** Play on my own. */
    onpause: () => void;
  }

  let { step, text, active, onnext, onpause }: Props = $props();

  // The step whose two frames have passed.
  let readyKey = $state<string | null>(null);
  // Whether a modal (a dialog, a popup) is open over the page.
  let modal = $state(false);
  // The step's target on the page, once ready.
  let anchorEl = $state<HTMLElement | null>(null);
  let nextEl = $state<HTMLButtonElement | undefined>();

  const due = $derived(active && step ? step : null);
  const ready = $derived(due !== null && readyKey === due.key);
  const bubble = $derived(ready && due?.surface === 'board' && !modal);

  // The targets already scrolled into view, with when: each element once,
  // so a sheet opened again (a new element) is scrolled again. An image that
  // loads soon after (a card sheet's art) can push the target back out, so
  // within REVEAL_SETTLE_MS of the first scroll a load scrolls it again;
  // later, the player's own scrolling is left alone.
  const revealed = new WeakMap<Element, number>();
  const REVEAL_SETTLE_MS = 2000;

  /** Finds the step's target now; writes state from callbacks only. */
  function resolve(): void {
    const s = ready ? due : null;
    const el = s?.target ? findAnchor(s.target) : null;
    if (el !== anchorEl) anchorEl = el;
    if (el && !revealed.has(el)) {
      revealed.set(el, performance.now());
      revealInScroller(el);
    }
  }

  /** A load just after the first scroll: scroll the target in again. */
  function revealAgain(): void {
    const el = anchorEl;
    const at = el ? revealed.get(el) : undefined;
    if (el && at !== undefined && performance.now() - at < REVEAL_SETTLE_MS) {
      revealInScroller(el);
    }
  }

  // DOM-only: wait two frames once a step is due, then resolve its target
  // (scrolling its scroll containers to it once).
  $effect(() => {
    const key = due?.key ?? null;
    if (key === null) return;
    let second = 0;
    const first = requestAnimationFrame(() => {
      second = requestAnimationFrame(() => {
        readyKey = key;
        resolve();
      });
    });
    return () => {
      cancelAnimationFrame(first);
      cancelAnimationFrame(second);
    };
  });

  // DOM-only: watch the page for modals opening and closing and for the
  // target being redrawn, at most once a frame.
  $effect(() => {
    let frame = 0;
    const check = (): void => {
      frame = 0;
      const open = modalOpen(document);
      if (open !== modal) modal = open;
      resolve();
    };
    const schedule = (): void => {
      if (frame === 0) frame = requestAnimationFrame(check);
    };
    const observer = new MutationObserver(schedule);
    observer.observe(document.body, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['open', 'data-fx-hidden', 'hidden']
    });
    schedule();
    window.addEventListener('resize', schedule);
    // An image's load does not bubble; capture it.
    window.addEventListener('load', revealAgain, true);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      window.removeEventListener('resize', schedule);
      window.removeEventListener('load', revealAgain, true);
    };
  });

  // DOM-only: the ring and the description on the target.
  $effect(() => {
    const el = ready ? anchorEl : null;
    const s = due;
    if (!el || !s) return;
    const described = el.getAttribute('aria-describedby');
    el.setAttribute('data-coach-target', '');
    if (s.region) el.setAttribute('data-coach-region', '');
    if (text?.text) {
      el.setAttribute(
        'aria-describedby',
        described ? `${described} ${COACH_TEXT_ID}` : COACH_TEXT_ID
      );
    }
    return () => {
      el.removeAttribute('data-coach-target');
      el.removeAttribute('data-coach-region');
      if (described === null) el.removeAttribute('aria-describedby');
      else el.setAttribute('aria-describedby', described);
    };
  });

  // DOM-only: the quiet focus move of a bubble step, and back when it goes.
  $effect(() => {
    if (!bubble || !due) return;
    const target = due.kind === 'info' ? nextEl : anchorEl;
    if (!target) return;
    const before = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    restoreFocus(target);
    return () => {
      const now = document.activeElement;
      const lost = !now || now === document.body || !now.isConnected || now === target;
      if (lost && before && before !== target && before.isConnected) restoreFocus(before);
    };
  });

  // The test hook's coach(): the step on screen (window.__ajayeb.coach()).
  setCoachSource(() => {
    if (!ready || !due) return null;
    if (due.surface === 'board' && !bubble) return null;
    return {
      step: due.id,
      kind: due.kind,
      surface: due.surface,
      selector: due.target ? due.target.join(', ') : null
    };
  });
  onDestroy(() => setCoachSource(null));

  const announce = $derived(ready && text ? text.text : '');
</script>

{#if due && text}
  <CoachMark
    open={bubble}
    anchor={anchorEl}
    title={text.title}
    progress={text.progress}
    text={text.text}
    kind={due.kind}
    step={due.id}
    turn={due.turn}
    region={due.region}
    onnext={due.kind === 'info' ? () => onnext(due) : undefined}
    {onpause}
    textId={COACH_TEXT_ID}
    bind:nextEl
  />
{/if}

<div class="sr-only" aria-live="polite" data-coach-live>{announce}</div>

<style>
  /* The ring on the coach step's target: an accent outline with an offset
     and a soft shadow, drawn on the element itself, so it works on the
     board and inside a sheet alike. With Animation on it closes in once;
     with it off (app.css's data-motion='reduce') it is simply there. It
     never loops. */
  :global([data-coach-target]) {
    outline: 2px solid var(--accent);
    outline-offset: 3px;
    box-shadow: 0 0 0 6px color-mix(in oklch, var(--accent) 22%, transparent);
    animation: coach-ring-in 200ms var(--curve-out, ease-out) both;
  }
  :global([data-coach-target][data-coach-region]) {
    border-radius: var(--shape-panel, 0.75rem);
    outline-offset: 4px;
  }
  :global(:root[data-motion='reduce'] [data-coach-target]) {
    animation: none;
  }
  @keyframes coach-ring-in {
    from {
      outline-offset: 10px;
      outline-color: transparent;
    }
  }
</style>
