<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import type { TransportStatus } from '$lib/session/transport';
  import type { Component } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import RoomError from '$lib/components/screens/RoomError.svelte';
  import { t } from '$lib/i18n';
  import { RoomBootstrap, snapshotKind } from '$lib/entry/bootstrap';
  import { loadEntry } from '$lib/entry/registry';
  import { entryIdentity, isDirectDuel } from '$lib/entry/identity';
  import { savedRooms } from '$lib/session/saved-rooms.svelte';
  interface Props {
    code: string;
    claim: string | null;
  }
  let { code, claim }: Props = $props();
  let Entry = $state.raw<Component<{ connection: RoomBootstrap | null; snapshot: unknown }> | null>(
    null
  );
  let snapshot = $state.raw<unknown>();
  let failure = $state<string | null>(null);
  let status = $state<TransportStatus['state']>('connecting');
  let connection = $state.raw<RoomBootstrap | null>(null);
  let disposed = false;
  let loading = false;
  let activeKind: ReturnType<typeof snapshotKind> = null;
  let accepts: ((value: unknown, code: string) => boolean) | null = null;
  function fail(error: string): void {
    if (disposed) return;
    failure = error;
    connection?.close();
    if (error === 'room-not-found' || error === 'room-archived') savedRooms().forget(code);
    if (error === 'bad-claim') savedRooms().dropClaim(code);
  }
  async function discover(): Promise<void> {
    if (disposed || loading || !connection) return;
    const kind = snapshotKind(connection.latest);
    if (!kind) return;
    loading = true;
    try {
      const module = await loadEntry(kind);
      if (disposed) return;
      if (snapshotKind(connection.latest) !== kind) {
        return;
      }
      if (!module.accepts(connection.latest, code)) {
        fail('protocol');
        return;
      }
      if (kind === 'duel') {
        connection.close(); // no leave, same identity/claim, ordinary v1 rejoin
        connection = null;
        snapshot = undefined; // no bootstrap events enter Duel's Presenter
      } else {
        snapshot = connection.latest;
        if (kind === 'haftgane')
          savedRooms().remember(code, 'online', Date.now(), claim, 'haftgane');
      }
      accepts = module.accepts;
      activeKind = kind;
      Entry = module.Component;
    } catch {
      fail('busy');
    } finally {
      loading = false;
      if (
        !disposed &&
        !failure &&
        !Entry &&
        connection?.latest &&
        snapshotKind(connection.latest) !== kind
      )
        void discover();
    }
  }
  onMount(() => {
    if (isDirectDuel(code)) {
      void loadEntry('duel')
        .then((module) => {
          if (!disposed) Entry = module.Component;
        })
        .catch(() => fail('busy'));
      return;
    }
    connection = new RoomBootstrap({
      code,
      claim,
      identity: entryIdentity(),
      snapshot: () => {
        if (Entry && snapshotKind(connection?.latest) === activeKind) {
          if (!accepts?.(connection?.latest, code)) {
            fail('protocol');
            return;
          }
          snapshot = connection?.latest;
          return;
        }
        Entry = null;
        void discover();
      },
      error: fail,
      status: (next) => {
        status = next.state;
      }
    });
  });
  onDestroy(() => {
    disposed = true;
    connection?.close();
  });
</script>

{#if failure}
  <RoomError error={failure} {code} onhome={() => void goto(resolve('/'))} />
{:else if Entry}
  <Entry {connection} {snapshot} />
{:else}
  <main class="flex h-full items-center justify-center p-4" aria-busy="true" data-game-entry>
    <p role="status" data-conn={status}>
      {t(
        status === 'offline'
          ? 'session.conn.offline'
          : status === 'reconnecting'
            ? 'session.conn.reconnecting'
            : 'session.conn.connecting'
      )}
    </p>
  </main>
{/if}
