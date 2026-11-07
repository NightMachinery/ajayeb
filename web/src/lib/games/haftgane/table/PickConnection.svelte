<script lang="ts">
  import { i18n, t } from '$lib/i18n';
  import type { ConnState } from '../session/session.svelte';
  import { ht } from '../i18n/messages.svelte';
  let { conn }: { conn: ConnState } = $props();
</script>

{#if conn !== 'open'}
  <p class="connection" role="status" dir={i18n.dir} data-pick-connection={conn}>
    {t(`session.conn.${conn}`)}
    {ht('hg.screen.connection.kept')}
  </p>
{/if}

<style>
  .connection {
    margin: 0.25rem 0;
    font-size: 0.8125rem;
    line-height: 1.3;
  }
</style>
