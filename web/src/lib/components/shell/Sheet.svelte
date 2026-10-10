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

  interface Props {
    /** Bindable: set true to open, false (or Esc, or a backdrop tap) to close. */
    open?: boolean;
    /** Id of the heading the caller renders inside children, for aria-labelledby. */
    titleId?: string;
    showHandle?: boolean;
    class?: string;
    children: Snippet;
  }

  let {
    open = $bindable(false),
    titleId,
    showHandle = true,
    class: className = '',
    children
  }: Props = $props();

  let dialogEl: HTMLDialogElement | undefined = $state();

  // DOM-only effect (see +layout.svelte): showModal()/close() never write
  // $state themselves, so this cannot loop with the `open` reads below.
  $effect(() => {
    const dialog = dialogEl;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    else if (!open && dialog.open) dialog.close();
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
>
  <div class="sheet-panel">
    {#if showHandle}
      <div class="sheet-handle split:hidden" aria-hidden="true"></div>
    {/if}
    {@render children()}
  </div>
</dialog>

<style>
  .sheet {
    position: fixed;
    inset: auto 0 0 0;
    margin: 0;
    width: 100%;
    max-width: 100%;
    max-height: 85dvh;
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
    max-height: 85dvh;
    flex-direction: column;
    overflow-y: auto;
    background: var(--panel);
    padding-bottom: max(1rem, var(--safe-b));
  }
  .sheet-handle {
    margin-inline: auto;
    margin-block: 0.5rem 0.25rem;
    height: 0.375rem;
    width: 2.5rem;
    flex-shrink: 0;
    border-radius: 999px;
    background: var(--rule-strong);
  }

  /* Wide screens, and a short landscape screen: a centred panel instead of a
     bottom sheet. Mirrors @custom-variant split in app.css. */
  @media (min-width: 64rem), (orientation: landscape) and (max-height: 32rem) {
    .sheet {
      inset: 0;
      margin: auto;
      width: min(480px, calc(100% - 2rem));
      max-height: min(85dvh, 640px);
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
      max-height: min(85dvh, 640px);
      padding-bottom: 1rem;
    }
  }
</style>
