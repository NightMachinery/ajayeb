<script lang="ts">
  // The small connection-state pill for online rooms (UX design "Connection
  // states"): quiet (nothing shown) once connected, an amber pill while
  // reconnecting, red while offline. Reuses session.conn.* -- the session
  // layer already owns that wording. `compact` (the top bar) shows only the
  // icon below desktop widths, where the TurnBar already says the state in
  // words and the bar has no room for them; the words stay for screen
  // readers.
  import { t } from '$lib/i18n';
  import WifiOff from '@lucide/svelte/icons/wifi-off';
  import Loader2 from '@lucide/svelte/icons/loader-2';

  interface Props {
    conn: 'connecting' | 'open' | 'reconnecting' | 'offline';
    compact?: boolean;
    class?: string;
  }

  let { conn, compact = false, class: className = '' }: Props = $props();

  function textKey(conn: Props['conn']) {
    switch (conn) {
      case 'connecting':
        return 'session.conn.connecting' as const;
      case 'open':
        return 'session.conn.open' as const;
      case 'reconnecting':
        return 'session.conn.reconnecting' as const;
      case 'offline':
        return 'session.conn.offline' as const;
    }
  }
</script>

{#if conn !== 'open'}
  <span
    role="status"
    title={t(textKey(conn))}
    class="connection-pill inline-flex min-h-7 shrink-0 items-center gap-1 rounded-full px-2 text-xs font-semibold
      {conn === 'offline'
      ? 'bg-cinnabar/15 text-cinnabar'
      : 'bg-saffron/20 text-ink dark:text-fg'} {className}"
  >
    {#if conn === 'offline'}
      <WifiOff class="size-3.5" aria-hidden="true" />
    {:else}
      <Loader2 class="size-3.5 animate-spin" aria-hidden="true" />
    {/if}
    <span class={compact ? 'sr-only lg:not-sr-only' : ''}>{t(textKey(conn))}</span>
  </span>
{:else}
  <span class="sr-only" role="status">{t(textKey(conn))}</span>
{/if}
