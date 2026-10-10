<script lang="ts">
  // A room that could not be opened: the server's error or reject code
  // (room-not-found, room-full, forbidden, busy, rate), mapped to a friendly
  // message, with what the player can do next and a way Home. A leaf: the
  // caller passes the code and the room it tried.
  import House from '@lucide/svelte/icons/house';
  import { t } from '$lib/i18n';
  import { errorMessage, rejectMessage } from '$lib/session/messages';
  import Button from '../shell/Button.svelte';

  interface Props {
    /** The server's error code, or 'rate' for a refused join. */
    error: string;
    /** The room code that was tried. */
    code?: string;
    onhome: () => void;
  }

  let { error, code, onhome }: Props = $props();

  const message = $derived(error === 'rate' ? rejectMessage(error) : errorMessage(error));
  const hint = $derived(
    error === 'room-not-found' ? t('online.error.notFoundHint') : t('online.error.retryHint')
  );
</script>

<main class="flex h-full items-center justify-center overflow-y-auto px-4 py-6">
  <div
    class="flex w-full max-w-sm flex-col items-center gap-3 rounded-panel border border-rule bg-panel p-6 text-center shadow-panel"
    role="alert"
    data-room-error={error}
  >
    <h1 class="font-display text-2xl text-fg">{t('online.error.title')}</h1>
    <p class="text-base text-fg">{t(message)}</p>
    <p class="text-sm text-fg-muted">{hint}</p>
    {#if code}
      <p class="text-xs text-fg-muted" dir="ltr">{t('game.title', { code })}</p>
    {/if}
    <Button variant="primary" full class="mt-2" onclick={onhome} data-control="error-home">
      <House class="size-4" aria-hidden="true" />
      {t('online.error.home')}
    </Button>
  </div>
</main>
