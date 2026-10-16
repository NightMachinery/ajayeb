<script lang="ts">
  // The visible toast list for a Toasts queue (toast-queue.svelte.ts). By
  // default it is fixed at the bottom of the viewport, above the safe-area
  // inset; the game screen anchors it to a box of its own instead (just
  // above the turn bar, or at the foot of the structure), so it covers
  // neither the opponent's strip nor the military track. A polite live
  // region, so a screen reader announces a new toast without stealing focus.
  //
  // The queue class lives in its own toast-queue.svelte.ts file, not
  // toasts.svelte.ts, because a file named toasts.svelte.ts and a component
  // named Toasts.svelte differ only in case, and TypeScript's module
  // resolution on a case-insensitive filesystem (macOS default) resolves an
  // extensionless `./toasts.svelte` import against the component file
  // instead of the class module.
  import X from '@lucide/svelte/icons/x';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { ToastKind, Toasts } from './toast-queue.svelte';

  interface Props {
    toasts: Toasts;
    /**
     * Where the list sits: fixed above the bottom edge (default) or below
     * the top bar; or anchored to its positioned parent, just above it
     * ('above') or at its foot ('inside').
     */
    placement?: 'bottom' | 'top' | 'above' | 'inside';
  }

  let { toasts, placement = 'bottom' }: Props = $props();

  const PLACEMENT_CLASS: Record<NonNullable<Props['placement']>, string> = {
    bottom: 'fixed inset-x-0 bottom-[calc(var(--safe-b)+1rem)] px-4',
    top: 'fixed inset-x-0 top-[calc(var(--safe-t)+3.25rem)] px-4',
    above: 'absolute inset-x-0 bottom-full mb-1.5 px-2',
    inside: 'absolute inset-x-0 bottom-1 px-2'
  };

  const KIND_CLASS: Record<ToastKind, string> = {
    info: 'border-rule',
    success: 'border-malachite',
    warning: 'border-saffron',
    danger: 'border-cinnabar'
  };
</script>

<div
  class="pointer-events-none z-50 flex flex-col items-center gap-2 {PLACEMENT_CLASS[placement]}"
  aria-live="polite"
  aria-atomic="false"
>
  {#each toasts.items as item (item.id)}
    <div
      class="pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-control border-s-4 bg-panel px-4 py-3 text-fg shadow-panel {KIND_CLASS[
        item.kind
      ]}"
      role="status"
      data-toast={item.kind}
    >
      <p class="flex-1 text-sm" dir="auto">{item.message}</p>
      {#if item.actionLabel}
        <button
          type="button"
          class="shrink-0 text-sm font-semibold text-accent underline underline-offset-2"
          onclick={() => {
            item.onAction?.();
            toasts.dismiss(item.id);
          }}
        >
          {item.actionLabel}
        </button>
      {/if}
      <button
        type="button"
        class="-me-2 grid size-11 shrink-0 place-items-center text-fg-muted hover:text-fg"
        aria-label={t('shell.dismiss')}
        {@attach tooltip({ text: t('shell.dismiss') })}
        onclick={() => toasts.dismiss(item.id)}
      >
        <X size={16} />
      </button>
    </div>
  {/each}
</div>
