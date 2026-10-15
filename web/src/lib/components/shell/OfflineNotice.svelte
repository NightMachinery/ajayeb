<script lang="ts">
  // One line, fixed above the bottom edge, while the browser is offline on
  // a page that has no connection state of its own (Home, the rules,
  // Settings). The service worker (src/service-worker.ts) opens the app
  // from its cache without a network, and this says why nothing can start.
  // A game page shows the session's own offline state instead, so the
  // layout does not pass `offline` there. Plain props, for the gallery.
  import WifiOff from '@lucide/svelte/icons/wifi-off';
  import { t } from '$lib/i18n';

  interface Props {
    offline: boolean;
  }

  let { offline }: Props = $props();
</script>

{#if offline}
  <div
    class="pointer-events-none fixed inset-x-0 bottom-[calc(var(--safe-b)+1rem)] z-40 flex justify-center px-4"
  >
    <p
      class="flex items-center gap-2 rounded-full border border-rule bg-panel px-4 py-2 text-sm text-fg shadow-panel"
      role="status"
      data-offline-notice
    >
      <WifiOff size={16} aria-hidden="true" class="shrink-0 text-fg-muted" />
      <span>{t('shell.offline')}</span>
    </p>
  </div>
{/if}
