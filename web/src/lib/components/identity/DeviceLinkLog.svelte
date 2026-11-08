<script lang="ts">
  import { t as lobbyT } from '$lib/lobby/i18n/messages';
  import { t, formatNumber } from '$lib/i18n';
  import type { PlatformHuman, PlatformLogEntry } from '$lib/wire/platform.gen';
  let {
    entries,
    humans
  }: { entries: readonly PlatformLogEntry[]; humans: readonly PlatformHuman[] } = $props();
  const links = $derived(entries.filter((e) => e.kind === 'link-copy' || e.kind === 'link'));
  function name(user: string | undefined): string {
    const h = humans.find((h) => h.userId === user);
    return h
      ? h.nameNo > 1
        ? `${h.name} ${formatNumber(h.nameNo)}`
        : h.name
      : lobbyT('lobby.deviceMember');
  }
</script>

{#if links.length}
  <section
    class="rounded-panel border border-rule bg-panel p-3"
    data-device-link-log
    aria-label={lobbyT('lobby.deviceActivity')}
  >
    <h3 class="font-semibold text-fg">{lobbyT('lobby.deviceActivity')}</h3>
    <ol class="flex flex-col gap-1 text-sm text-fg-muted" aria-live="polite">
      {#each links as entry, i (i)}
        <li data-link-event={entry.kind} dir="auto">
          {entry.kind === 'link-copy'
            ? lobbyT('lobby.deviceCopied', { name: name(entry.by), target: name(entry.userId) })
            : t('online.log.linkResetOther', {
                name: name(entry.by),
                target: name(entry.userId ?? entry.by)
              })}
        </li>
      {/each}
    </ol>
  </section>
{/if}
