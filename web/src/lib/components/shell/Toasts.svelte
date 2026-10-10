<script lang="ts">
  // The visible toast list for a Toasts queue (toast-queue.svelte.ts), fixed
  // at the bottom of the viewport, above the safe-area inset. A polite live
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
  import type { ToastKind, Toasts } from './toast-queue.svelte';

  interface Props {
    toasts: Toasts;
  }

  let { toasts }: Props = $props();

  const KIND_CLASS: Record<ToastKind, string> = {
    info: 'border-rule',
    success: 'border-malachite',
    warning: 'border-saffron',
    danger: 'border-cinnabar'
  };
</script>

<div
  class="pointer-events-none fixed inset-x-0 bottom-[calc(var(--safe-b)+1rem)] z-50 flex flex-col items-center gap-2 px-4"
  aria-live="polite"
  aria-atomic="false"
>
  {#each toasts.items as item (item.id)}
    <div
      class="pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-control border-s-4 bg-panel px-4 py-3 text-fg shadow-panel {KIND_CLASS[
        item.kind
      ]}"
      role="status"
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
        onclick={() => toasts.dismiss(item.id)}
      >
        <X size={16} />
      </button>
    </div>
  {/each}
</div>
