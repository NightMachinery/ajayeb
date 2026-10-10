<script lang="ts">
  // The small bar that stays on screen while a choice panel or the game-over
  // screen is minimized to peek at the board: what is open, and a button to
  // bring it back. Also used, without a button, to say what the other
  // player is choosing. It floats above the TurnBar and never covers more
  // than one line of the board.
  import ChevronUp from '@lucide/svelte/icons/chevron-up';
  import X from '@lucide/svelte/icons/x';
  import { t } from '$lib/i18n';

  interface Props {
    /** Already translated. */
    text: string;
    /** Brings the panel back; no button when absent. */
    onopen?: () => void;
    /** The button's label (default "Open"). */
    openLabel?: string;
    /** Closes the bar for good; no close button when absent. */
    ondismiss?: () => void;
    class?: string;
  }

  let { text, onopen, openLabel, ondismiss, class: className = '' }: Props = $props();
</script>

<div
  class="pointer-events-none fixed inset-x-0 bottom-[calc(var(--safe-b)+3.75rem)] z-30 flex justify-center px-3 {className}"
  data-peek-bar
>
  <div
    class="pointer-events-auto flex max-w-full min-w-0 items-center gap-2 rounded-[1.5rem] border border-rule-strong bg-panel
      py-1 ps-4 text-sm text-fg shadow-panel {onopen ? 'pe-1' : 'min-h-11 pe-4'}"
    role="status"
  >
    <span class="line-clamp-2 min-w-0 leading-snug font-semibold" dir="auto">{text}</span>
    {#if onopen}
      <button
        type="button"
        class="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-full bg-accent px-3.5 font-semibold text-on-accent
          hover:brightness-105 active:brightness-95"
        onclick={onopen}
      >
        <ChevronUp class="size-4" aria-hidden="true" />
        {openLabel ?? t('choices.reopen')}
      </button>
    {/if}
    {#if ondismiss}
      <button
        type="button"
        class="grid min-h-11 min-w-11 shrink-0 place-items-center rounded-full text-fg-muted hover:bg-panel-sunken hover:text-fg"
        aria-label={t('shell.dismiss')}
        onclick={ondismiss}
      >
        <X class="size-4" aria-hidden="true" />
      </button>
    {/if}
  </div>
</div>
