<script lang="ts">
  // The small bar that stays on screen while a choice panel or the game-over
  // screen is minimized to peek at the board: what is open, and a button to
  // bring it back. Also used, without a button, to say what the other
  // player is choosing. By default it floats above the bottom edge (the
  // game-over screen's peek); inline, it sits in the flow where its parent
  // puts it (the choice host's place just above the turn bar), so it covers
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

<div
  class="flex justify-center {inline
    ? '@container border-t border-rule px-1.5 py-1'
    : 'pointer-events-none fixed inset-x-0 bottom-[calc(var(--safe-b)+3.75rem)] z-30 px-3'} {className}"
  data-peek-bar={inline ? 'inline' : 'float'}
>
  <!-- In a narrow column (a phone in landscape) the buttons wrap under the
       text rather than squeezing it away. -->
  <div
    class="pointer-events-auto flex max-w-full min-w-0 flex-wrap items-center justify-center gap-x-2 gap-y-1 rounded-[1.5rem]
      border border-rule-strong bg-panel py-1 ps-4 text-sm text-fg shadow-panel @max-3xs:ps-3 {onopen ||
    ondismiss
      ? 'pe-1'
      : 'min-h-11 pe-4'}"
    role="status"
  >
    <span class="line-clamp-2 min-w-0 flex-auto leading-snug font-semibold">{text}</span>
    {#if onopen}
      <button
        type="button"
        class="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-full bg-accent px-3.5 font-semibold text-on-accent
          hover:brightness-105 active:brightness-95"
        onclick={onopen}
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
