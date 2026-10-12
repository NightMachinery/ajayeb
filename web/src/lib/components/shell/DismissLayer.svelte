<script lang="ts">
  // A floating panel that any tap closes, the tap being swallowed so it
  // never acts on what lies beneath (ui/dismiss.ts, the same behaviour as an
  // open tooltip). Escape closes it too. Made for the opponent move summary
  // and the tutorial coach marks.
  //
  //   <DismissLayer bind:open label={t('...')} class="top-16 bottom-auto">
  //     ...content...
  //   </DismissLayer>
  //
  // It is a manual popover, so it shows above an open sheet as well. By
  // default it is centred in the viewport (inset 0 with auto margins; give
  // it a width); pass position classes, such as "top-16 bottom-auto", to
  // place it elsewhere. With `keepInside`, a tap inside the panel reaches its
  // target as usual (a coach mark's Next button) and only taps outside
  // close it.
  //
  // As a dialog (the default role) it manages the keyboard focus: the panel
  // takes the focus when it opens, Tab and Shift+Tab stay inside it while it
  // is open, and when it closes the focus goes back to the element that had
  // it before (the button that opened it), unless the focus has meanwhile
  // moved on to something else. A status or note panel leaves the focus
  // alone.
  import type { Snippet } from 'svelte';
  import { installDismiss, trapFocus } from '$lib/ui/dismiss';

  interface Props {
    /** Bindable: true shows the panel; a tap or Escape sets it false. */
    open?: boolean;
    /** Called after a tap or Escape closed the panel. */
    onclose?: (reason: 'tap' | 'escape') => void;
    /** Taps inside the panel act normally instead of closing it. */
    keepInside?: boolean;
    /** The panel's accessible name. */
    label?: string;
    labelledby?: string;
    role?: 'dialog' | 'status' | 'note';
    class?: string;
    children: Snippet;
  }

  let {
    open = $bindable(false),
    onclose,
    keepInside = false,
    label,
    labelledby,
    role = 'dialog',
    class: className = '',
    children
  }: Props = $props();

  let panelEl: HTMLDivElement | undefined = $state();

  // DOM-only: the popover enters the top layer when the panel mounts.
  $effect(() => {
    const el = panelEl;
    if (el && typeof el.showPopover === 'function' && !el.matches(':popover-open')) {
      el.showPopover();
    }
  });

  // DOM-only: the focus moves in when a dialog panel mounts, stays inside,
  // and goes back to its opener when the panel goes away.
  $effect(() => {
    const el = panelEl;
    if (!el || role !== 'dialog') return;
    return trapFocus(el);
  });

  // The dismiss listeners, only while open. They write `open` from events,
  // never from this effect's own run.
  $effect(() => {
    if (!open) return;
    return installDismiss({
      onDismiss: (reason) => {
        open = false;
        onclose?.(reason);
      },
      keep: () => (keepInside ? [panelEl] : [])
    });
  });
</script>

{#if open}
  <div
    bind:this={panelEl}
    popover="manual"
    {role}
    tabindex="-1"
    aria-label={label}
    aria-labelledby={labelledby}
    class="dismiss-layer fixed inset-0 m-auto h-fit max-w-[calc(100vw-2rem)] rounded-panel border border-rule-strong bg-panel p-4 text-fg shadow-panel {className}"
    data-dismiss-layer
  >
    {@render children()}
  </div>
{/if}

<style>
  .dismiss-layer {
    animation: dismiss-in var(--dur-base) ease-out;
  }
  /* The panel itself takes the focus only so the keyboard is inside it;
     its border already shows where it is. */
  .dismiss-layer:focus {
    outline: none;
  }
  @keyframes dismiss-in {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
  }
</style>
