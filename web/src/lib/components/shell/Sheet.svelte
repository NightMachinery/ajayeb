<script lang="ts">
  // A native <dialog> used as a bottom sheet on phones and a centred panel on
  // wide screens or short landscape (the "split" breakpoint; see app.css's
  // @custom-variant split, mirrored below since a plain CSS style block
  // cannot reference a Tailwind custom variant directly). showModal() gives
  // us a focus trap, Esc handling, top-layer stacking and an inert
  // background for free; @starting-style gives the slide/fade entrance with
  // no JS animation.
  //
  // Note for the next editor: never spell out the style or script tag with
  // its angle brackets inside a comment in this script block (write "the
  // style block", not the bracketed form). svelte-check's bundled
  // svelte2tsx does a naive text scan for those two bracketed tag strings
  // to find this file's section boundaries, so a stray match inside a JS
  // comment here previously made it miscompute those boundaries and fail
  // with a spurious parse error ("element was left open"), even though the
  // real Svelte compiler parses the file fine.
  import type { Snippet } from 'svelte';
  import { attachSheetDrag, SPLIT_QUERY } from './sheet-drag';
  import { attachSheetScroll } from './sheet-scroll';

  interface Props {
    /** Bindable: set true to open, false (or Esc, or a backdrop tap) to close. */
    open?: boolean;
    /** Id of the heading the caller renders inside children, for aria-labelledby. */
    titleId?: string;
    showHandle?: boolean;
    /** A centred panel at every size, not only on wide screens (dialogs
     *  such as the age interstitial and the game-over screen). */
    centered?: boolean;
    class?: string;
    children: Snippet;
    /** Actions kept outside the scrolling detail body. A sheet without one
     *  marks its primary action row `data-sheet-actions` instead, and the
     *  sheet pins that row to its bottom edge while the rest scrolls; the
     *  row must sit directly in the content's one wrapper (a form or a
     *  div), which spans the whole content. */
    footer?: Snippet;
    /** Stable caller hooks encompass both the details and the footer. */
    panelData?: Record<`data-${string}`, string | undefined>;
  }

  let {
    open = $bindable(false),
    titleId,
    showHandle = true,
    centered = false,
    class: className = '',
    children,
    footer,
    panelData = {}
  }: Props = $props();

  let dialogEl: HTMLDialogElement | undefined = $state();
  let panelEl: HTMLDivElement | undefined = $state();

  // DOM-only effect (see +layout.svelte): showModal()/close() never write
  // $state themselves, so this cannot loop with the `open` reads below.
  $effect(() => {
    const dialog = dialogEl;
    if (!dialog) return;
    if (open && !dialog.open) {
      dialog.showModal();
      focusTitle(dialog);
    } else if (!open && dialog.open) dialog.close();
  });

  // The dialog opens with the focus on its title, the WAI-ARIA dialog
  // pattern for long content. showModal() alone focuses the first focusable
  // descendant, which in a sheet with a footer is the scrolling body, and
  // with no pointer input since the load (a reload into a finished game)
  // Chrome's :focus-visible then rings the whole body. The title takes
  // focus from script only (tabindex -1, so it is no Tab stop) and draws no
  // ring, since it is not a control; Tab moves on to the next control,
  // which does. A child marked autofocus keeps the focus showModal() gave it.
  function focusTitle(dialog: HTMLDialogElement): void {
    if (!titleId || dialog.querySelector('[autofocus]')) return;
    const title = document.getElementById(titleId);
    if (!title || !dialog.contains(title)) return;
    if (!title.hasAttribute('tabindex')) title.tabIndex = -1;
    title.setAttribute('data-sheet-title', '');
    title.focus({ preventScroll: true });
  }

  // Swipe down to dismiss (sheet-drag.ts), only while this is a bottom
  // sheet: never for a centred panel. DOM listeners only, no $state writes.
  $effect(() => {
    const dialog = dialogEl;
    const panel = panelEl;
    if (!dialog || !panel) return;
    return attachSheetDrag(
      dialog,
      panel,
      () => !centered && !globalThis.matchMedia?.(SPLIT_QUERY).matches
    );
  });

  // The dialog's own 'close' fires on Esc too, so this is the single place
  // that syncs `open` back to false.
  function handleClose(): void {
    open = false;
  }

  // A click that lands on the <dialog> element itself (never a descendant)
  // is a click on the ::backdrop area; see MDN's "light-dismiss" pattern.
  function handleClick(event: MouseEvent): void {
    if (event.target === dialogEl) open = false;
  }
</script>

<dialog
  bind:this={dialogEl}
  onclose={handleClose}
  onclick={handleClick}
  aria-labelledby={titleId}
  class="sheet {className}"
  class:centered
>
  <div {...panelData} class="sheet-panel" class:has-footer={!!footer} bind:this={panelEl}>
    {#if showHandle && !centered}
      <div class="sheet-handle split:hidden" data-sheet-handle aria-hidden="true"></div>
    {/if}
    {#if footer}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex (This named scroll region must accept keyboard focus independently of its footer.) -->
      <div
        class="sheet-body"
        data-sheet-body
        role="region"
        aria-labelledby={titleId}
        tabindex="0"
        {@attach attachSheetScroll}
      >
        {@render children()}
      </div>
      <div class="sheet-footer" data-sheet-footer>{@render footer()}</div>
    {:else}
      {@render children()}
    {/if}
  </div>
</dialog>

<style>
  .sheet {
    position: fixed;
    inset: auto 0 0 0;
    margin: 0;
    width: 100%;
    max-width: 100%;
    max-height: var(--sheet-max-height, 85dvh);
    border: 0;
    border-radius: var(--shape-panel) var(--shape-panel) 0 0;
    background: transparent;
    padding: 0;
    padding-bottom: var(--safe-b);
    color: var(--fg);
    opacity: 1;
    translate: 0 0;
    transition:
      opacity var(--dur-base) var(--curve-out),
      translate var(--dur-base) var(--curve-out),
      display var(--dur-base) allow-discrete,
      overlay var(--dur-base) allow-discrete;
  }
  .sheet::backdrop {
    background: color-mix(in oklch, var(--ink) 45%, transparent);
    transition:
      opacity var(--dur-base) var(--curve-out),
      display var(--dur-base) allow-discrete,
      overlay var(--dur-base) allow-discrete;
    opacity: 1;
  }
  .sheet:not([open]) {
    opacity: 0;
    translate: 0 100%;
  }
  @starting-style {
    .sheet[open] {
      opacity: 0;
      translate: 0 100%;
    }
    .sheet[open]::backdrop {
      opacity: 0;
    }
  }

  .sheet-panel {
    display: flex;
    max-height: var(--sheet-max-height, 85dvh);
    flex-direction: column;
    overflow-y: auto;
    background: var(--panel);
    --sheet-bottom-padding: max(1rem, var(--safe-b));
    padding-block-end: var(--sheet-bottom-padding);
    /* A pull past the top must not chain to the page, where the browser
       would overscroll it or pull to refresh. */
    overscroll-behavior: contain;
    /* A scrolling panel can take focus when the dialog opens; its ring
       would frame the whole sheet. Its buttons keep their own rings. */
    outline: none;
  }
  .sheet-panel.has-footer {
    overflow: hidden;
    padding-block-end: 0;
  }
  .sheet-body {
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    /* Keep the keyboard ring inside the clipped scrolling body. */
    outline-offset: -2px;
  }
  /* The title holds the focus on open (focusTitle) but is not a control. */
  .sheet :global([data-sheet-title]) {
    outline: none;
  }
  .sheet-body[data-scroll-more] {
    mask-image: linear-gradient(black calc(100% - 1.25rem), transparent);
  }
  /* A short window must never hide a sheet's primary action below the
     fold (M18.short-screen-status). Pinned, the row stops at the panel's
     bottom padding; its shadows cover a ring around it and that padding
     below it, so nothing shows through as the rest scrolls under. */
  .sheet-panel:not(.has-footer) :global([data-sheet-actions]) {
    position: sticky;
    inset-block-end: 0;
    z-index: 1;
    box-shadow:
      0 0 0 0.5rem var(--panel),
      0 var(--sheet-bottom-padding) 0 0.5rem var(--panel);
  }
  /* A marked row (not a lone button, which keeps its own fill) covers what
     scrolls under the gaps between its buttons. */
  .sheet-panel:not(.has-footer) :global(:not(button)[data-sheet-actions]) {
    background: var(--panel);
  }
  .sheet-footer {
    flex-shrink: 0;
    border-block-start: 1px solid var(--rule);
    padding: 0.5rem 1rem var(--sheet-bottom-padding);
    background: var(--panel);
  }
  .sheet-handle {
    position: relative;
    margin-inline: auto;
    margin-block: 0.5rem 0.25rem;
    height: 0.375rem;
    width: 2.5rem;
    flex-shrink: 0;
    border-radius: 999px;
    background: var(--rule-strong);
    touch-action: none;
  }
  /* The handle is a small bar; give the finger a larger grip around it. */
  .sheet-handle::before {
    content: '';
    position: absolute;
    inset: -0.75rem -2.5rem;
  }

  /* Wide screens, and a short landscape screen: a centred panel instead of a
     bottom sheet. Mirrors @custom-variant split in app.css. */
  @media (min-width: 64rem), (orientation: landscape) and (max-height: 32rem) {
    .sheet {
      inset: 0;
      margin: auto;
      width: min(480px, calc(100% - 2rem));
      max-height: var(--sheet-max-height, min(85dvh, 640px));
      border-radius: var(--shape-panel);
      padding-bottom: 0;
      scale: 0.96;
    }
    .sheet[open] {
      scale: 1;
    }
    .sheet:not([open]) {
      translate: 0 0;
      scale: 0.96;
    }
    @starting-style {
      .sheet[open] {
        translate: 0 0;
        scale: 0.96;
      }
    }
    .sheet-panel {
      max-height: var(--sheet-max-height, min(85dvh, 640px));
      --sheet-bottom-padding: 1rem;
    }
  }

  /* The same centred panel at every size, when asked for. */
  .sheet.centered {
    inset: 0;
    margin: auto;
    width: min(480px, calc(100% - 2rem));
    max-height: var(--sheet-max-height, min(calc(100dvh - 2rem), 640px));
    border-radius: var(--shape-panel);
    padding-bottom: 0;
    scale: 1;
    translate: 0 0;
  }
  .sheet.centered:not([open]) {
    translate: 0 0;
    scale: 0.96;
  }
  @starting-style {
    .sheet.centered[open] {
      translate: 0 0;
      scale: 0.96;
    }
  }
  .sheet.centered .sheet-panel {
    max-height: var(--sheet-max-height, min(calc(100dvh - 2rem), 640px));
    --sheet-bottom-padding: 1rem;
  }
</style>
