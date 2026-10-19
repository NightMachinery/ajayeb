<script lang="ts">
  // The small bar that stays on screen while a choice panel or the game-over
  // screen is minimized to peek at the board: what is open, and a button to
  // bring it back. Also used, without a button, to say what the other
  // player is choosing. By default it floats at the bottom edge, over the
  // turn bar (the game-over screen's peek, where the turn bar only repeats
  // "Game over"), so the board keeps its whole height and the player's own
  // strip stays readable; inline, it sits in the flow where its parent puts
  // it (the choice host's place just above the turn bar), so it covers
  // nothing at all.
  import ChevronUp from '@lucide/svelte/icons/chevron-up';
  import X from '@lucide/svelte/icons/x';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    /** Already translated. */
    text: string;
    /** Brings the panel back; no button when absent. */
    onopen?: () => void;
    /** The button's label (default "Open"). */
    openLabel?: string;
    /** Closes the bar for good; no close button when absent. */
    ondismiss?: () => void;
    /** In the flow, rather than floating over the page. */
    inline?: boolean;
    class?: string;
  }

  let {
    text,
    onopen,
    openLabel,
    ondismiss,
    inline = false,
    class: className = ''
  }: Props = $props();
</script>

<!-- The container is this full-width wrapper, never the bar itself: an
     inline-size container takes no width from its content, so a bar that was
     one would shrink to its padding and squeeze the text away. Floating, the
     bar lies along the bottom edge in the turn bar's row, inside the turn
     bar's safe-area padding by 4 px because it is 10 px taller than the 40 px
     row, so it covers neither strip (a phone held sideways has the turn bar
     in its first column only; there the bar covers the structure's foot). -->
<div
  class="@container flex justify-center {inline
    ? 'border-t border-rule px-1.5 py-1'
    : 'pointer-events-none fixed inset-x-0 bottom-[max(0.125rem,calc(var(--safe-b)-0.25rem))] z-30 px-3'} {className}"
  data-peek-bar={inline ? 'inline' : 'float'}
>
  <!-- Inline, in a narrow column (a phone in landscape), the buttons wrap
       under the text rather than squeezing it away. Floating, it is one row
       as wide as the turn bar it covers (at most 28rem), its text cut short
       if it must be: the panel it reopens says it in full. -->
  <div
    class="pointer-events-auto flex min-w-0 items-center rounded-[1.5rem] border border-rule-strong bg-panel ps-4 text-sm
      text-fg shadow-panel {inline
      ? 'max-w-full flex-wrap justify-center gap-x-2 gap-y-1 py-1 @max-3xs:ps-3'
      : 'w-full max-w-md justify-between gap-3 py-0.5'} {onopen || ondismiss
      ? 'pe-1'
      : 'min-h-11 pe-4'}"
    role="status"
  >
    <span
      class="min-w-0 leading-snug font-semibold {inline
        ? 'line-clamp-2 flex-auto'
        : 'flex-1 truncate'}"
      data-peek-text>{text}</span
    >
    {#if onopen}
      <button
        type="button"
        class="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-full bg-accent px-3.5 font-semibold text-on-accent
          hover:brightness-105 active:brightness-95"
        onclick={onopen}
        data-peek-open
      >
        <!-- Dropped in a narrow column, so the button fits beside the close. -->
        <ChevronUp class="size-4 @max-3xs:hidden" aria-hidden="true" />
        {openLabel ?? t('choices.reopen')}
      </button>
    {/if}
    {#if ondismiss}
      <button
        type="button"
        class="grid min-h-11 min-w-11 shrink-0 place-items-center rounded-full text-fg-muted hover:bg-panel-sunken hover:text-fg"
        aria-label={t('shell.dismiss')}
        {@attach tooltip({ text: t('shell.dismiss') })}
        onclick={ondismiss}
      >
        <X class="size-4" aria-hidden="true" />
      </button>
    {/if}
  </div>
</div>
