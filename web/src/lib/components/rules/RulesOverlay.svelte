<script lang="ts">
  // The in-game rules overlay, mounted once by openRules()
  // (rules-overlay.svelte.ts). It never leaves the screen it was opened on:
  //
  // - On wide screens and short landscape ones (the split breakpoint) it is
  //   a side panel, a non-modal manual popover in the top layer, below the
  //   top bar at the right edge. On a desktop it is exactly as wide as the
  //   opponent's city column, so the structure, the track, the player's own
  //   panel, the top bar and the turn bar stay visible and usable.
  // - On phones it is a bottom sheet (shell/Sheet.svelte): modal, with
  //   swipe-down, Esc and a backdrop tap to close.
  //
  // The board region is dir="ltr" (AGENTS.md), with the opponent's column
  // on the right in every locale, so the side panel sits at the physical
  // right to cover that column and never the player's own.
  import { afterNavigate } from '$app/navigation';
  import { page } from '$app/state';
  import { MediaQuery } from 'svelte/reactivity';
  import Sheet from '../shell/Sheet.svelte';
  import { SPLIT_QUERY } from '../shell/sheet-drag';
  import RulesPanel from './RulesPanel.svelte';
  import { closeRules, rulesOverlay } from './rules-overlay.svelte';

  const split = new MediaQuery(SPLIT_QUERY, false);
  const uid = $props.id();
  const titleId = `${uid}-rules-title`;

  /** Shown only on the page it was opened on. */
  const visible = $derived(rulesOverlay.open && page.url.pathname === rulesOverlay.openedOn);

  // Leaving the page closes the rules, so coming back does not reopen them.
  afterNavigate(({ to }) => {
    if (to && to.url.pathname !== rulesOverlay.openedOn) closeRules();
  });

  let drawer = $state<HTMLElement | undefined>();

  /** The element that had focus before the panel opened, to give it back. */
  let opener: HTMLElement | null = null;

  /** Places the panel just under the game's top bar, when there is one. */
  function placeDrawer(el: HTMLElement): void {
    const bar = document.querySelector('.top-bar');
    const top = bar ? Math.max(0, Math.round(bar.getBoundingClientRect().bottom)) : 0;
    el.style.setProperty('--rules-top', `${top}px`);
  }

  // DOM-only effect for the side panel: show or hide the popover, place it,
  // move focus in and back out, and close on Escape. It writes no $state;
  // the Escape listener calls closeRules() from an event, not from here.
  $effect(() => {
    const el = drawer;
    if (!el) return;
    if (!visible) {
      if (el.matches(':popover-open')) {
        el.hidePopover();
        if (opener?.isConnected) opener.focus({ preventScroll: true });
        opener = null;
      }
      return;
    }
    if (!el.matches(':popover-open')) {
      opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      placeDrawer(el);
      el.showPopover();
      el.querySelector<HTMLElement>('[data-rules-title]')?.focus({ preventScroll: true });
    }
    const onResize = () => placeDrawer(el);
    const onKey = (event: KeyboardEvent) => {
      // A modal sheet above the panel takes its own Escape first.
      if (event.key !== 'Escape' || document.querySelector('dialog:modal')) return;
      closeRules();
    };
    window.addEventListener('resize', onResize);
    document.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('resize', onResize);
      document.removeEventListener('keydown', onKey);
    };
  });
</script>

{#if split.current}
  <aside
    bind:this={drawer}
    popover="manual"
    class="rules-drawer"
    aria-labelledby={titleId}
    data-rules-overlay="drawer"
  >
    <RulesPanel
      {titleId}
      onclose={closeRules}
      section={rulesOverlay.section}
      request={rulesOverlay.request}
      shown={visible}
    />
  </aside>
{:else}
  <Sheet
    bind:open={() => visible, (open) => (open ? undefined : closeRules())}
    {titleId}
    class="rules-sheet"
  >
    <div data-rules-overlay="sheet">
      <RulesPanel
        {titleId}
        onclose={closeRules}
        section={rulesOverlay.section}
        request={rulesOverlay.request}
        shown={visible}
      />
    </div>
  </Sheet>
{/if}

<style>
  .rules-drawer {
    position: fixed;
    inset: auto;
    top: var(--rules-top, 0px);
    bottom: 0;
    right: 0; /* physical-ok: the opponent's column, see the note above */
    margin: 0;
    /* A desktop's side column (GameScreen.svelte), plus the safe area. */
    width: calc(300px + var(--safe-r, 0px));
    max-width: 100vw;
    border: 0;
    border-inline-start: 1px solid var(--rule);
    padding: 0 var(--safe-r, 0px) var(--safe-b, 0px) 0; /* physical-ok: safe-area insets */
    overflow-y: auto;
    overscroll-behavior: contain;
    scroll-behavior: smooth;
    background: var(--panel);
    color: var(--fg);
    box-shadow: var(--shadow);
    translate: 0 0;
    transition:
      translate var(--dur-base) var(--curve-out),
      display var(--dur-base) allow-discrete,
      overlay var(--dur-base) allow-discrete;
  }
  .rules-drawer:not(:popover-open) {
    display: none;
    translate: 100% 0;
  }
  @starting-style {
    .rules-drawer:popover-open {
      translate: 100% 0;
    }
  }
  @media (min-width: 100rem) {
    .rules-drawer {
      width: calc(380px + var(--safe-r, 0px));
    }
  }
  /* A phone on its side: a little wider than its narrow side column, so
     the text has room, and never more than half the screen. */
  @media (orientation: landscape) and (max-height: 32rem) {
    .rules-drawer {
      width: min(20rem, 50vw);
    }
  }

  :global(.rules-sheet .sheet-panel) {
    scroll-behavior: smooth;
  }
</style>
