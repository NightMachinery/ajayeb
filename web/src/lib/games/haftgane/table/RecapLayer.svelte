<script lang="ts">
  // The turn recap over the board (phone-layout.md section 4): it opens by
  // itself after a reveal (or from the top bar's button) and, like Duel's
  // move summary, any tap anywhere closes it and is swallowed, so it never
  // acts on the board beneath; Escape closes it too. On touch, a long-press
  // on a tag or a coin shows its tooltip and keeps the recap open. It is a
  // manual popover, so it shows above the board's own layers, and it keeps
  // the keyboard focus inside while open. It sits under the top bar on a
  // phone and over the middle column on wider screens, and scrolls inside
  // when the turn is longer than the room. A hint at its foot says how to
  // close it.
  //
  // It uses the root's dismiss and focus helpers ($lib/ui/dismiss), which
  // the root chunk already shares, rather than Duel's DismissLayer, so
  // Duel's chunks stay as they are (architecture.md, "Chunks").
  import type { Snippet } from 'svelte';
  import type { Attachment } from 'svelte/attachments';
  import { installDismiss, trapFocus } from '$lib/ui/dismiss';
  import { ht } from '../i18n/messages.svelte';

  interface Props {
    /** The recap's heading id. */
    labelledby: string;
    /** A tap or Escape closed it. */
    onclose: () => void;
    /** It opened by itself after a reveal (it pops in) rather than on request. */
    pop?: boolean;
    children: Snippet;
  }

  let { labelledby, onclose, pop = false, children }: Props = $props();

  const layer: Attachment<HTMLDivElement> = (el) => {
    if (typeof el.showPopover === 'function' && !el.matches(':popover-open')) el.showPopover();
    const untrap = trapFocus(el);
    const off = installDismiss({ onDismiss: () => onclose(), longPressTips: () => el });
    return () => {
      off();
      untrap();
    };
  };
</script>

<div
  class="layer"
  class:pop
  popover="manual"
  role="dialog"
  tabindex="-1"
  aria-labelledby={labelledby}
  data-recap-layer
  data-tooltip-actionable=""
  {@attach layer}
>
  {@render children()}
  <p class="hint">{ht('hg.recap.close')}</p>
</div>

<style>
  .layer {
    position: fixed;
    inset-block: calc(3.25rem + var(--safe-t)) auto;
    inset-inline: 0;
    box-sizing: border-box;
    width: min(34rem, calc(100vw - 1rem));
    max-height: calc(100dvh - 6.5rem - var(--safe-t));
    margin: 0 auto;
    padding: 0.6rem 0.75rem 0.75rem;
    overflow-y: auto;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow);
    color: var(--fg);
  }
  .layer:focus {
    outline: none;
  }
  .pop {
    animation: recap-in var(--dur-base) ease-out both;
  }
  .hint {
    margin: 0.4rem 0 0;
    color: var(--fg-muted);
    font-size: 0.75rem;
    text-align: center;
  }
  @keyframes recap-in {
    from {
      opacity: 0;
      transform: translateY(-0.4rem);
    }
  }
  /* A short landscape screen: over the middle column, as tall as it is. */
  @media (orientation: landscape) and (max-height: 32rem) {
    .layer {
      width: min(32rem, calc(100vw - 23rem));
      max-height: calc(100dvh - 3.75rem - var(--safe-t));
    }
  }
  @media (min-width: 64rem) and (min-height: 32.01rem) {
    .layer {
      inset-block-start: calc(4rem + var(--safe-t));
      width: 36rem;
      max-height: calc(100dvh - 8rem);
    }
  }
</style>
