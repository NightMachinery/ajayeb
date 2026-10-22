<script lang="ts">
  // D61: the ordinary v1 Duel path first; all discovery code is lazy.
  import { onMount, onDestroy, untrack, flushSync, type Component } from 'svelte';
  import { afterNavigate, goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import RoomError from '$lib/components/screens/RoomError.svelte';
  import { errorMessage } from '$lib/session/messages';
  import { knownEntryGame, rememberEntryGame } from '$lib/entry/identity';
  import type { Discovery } from '$lib/entry/discovery';
  import type { EntryProps } from '$lib/entry/types';
  import type { TransportStatus } from '$lib/session/transport';
  import { savedRooms } from '$lib/session/saved-rooms.svelte';
  interface Props {
    code: string;
    claim: string | null;
  }
  let { code, claim }: Props = $props();
  let Entry = $state.raw<Component<EntryProps> | null>(null);
  let entryProps = $state.raw<EntryProps>({});
  let failure = $state<string | null>(null);
  let status = $state<TransportStatus['state']>('connecting');
  let discovery: Discovery | undefined;
  let kind: 'duel' | 'haftgane' | 'lobby' | null = null;
  let key = '';
  let generation = 0;
  let disposed = false;
  let started = false;
  function fail(error: string): void {
    if (disposed) return;
    failure = error;
    discovery?.close();
    if (error === 'room-not-found' || error === 'room-archived') savedRooms().forget(code);
    if (error === 'bad-claim') savedRooms().dropClaim(code);
  }
  async function fallback(): Promise<void> {
    const request = ++generation;
    discovery?.close();
    // Unregister Room's navigation callback before it could join a known
    // non-Duel target. A wrong-game refusal has no admitted room to leave.
    if (Entry)
      flushSync(() => {
        Entry = null;
      });
    try {
      const { Discovery } = await import('$lib/entry/discovery');
      if (disposed || request !== generation) return;
      discovery = new Discovery({
        code,
        claim,
        ready: (entry, props, next) => {
          if (disposed || request !== generation) return;
          kind = next;
          entryProps = props;
          Entry = entry;
        },
        error: fail,
        status: (next) => {
          status = next.state;
        }
      });
    } catch {
      fail('busy');
    }
  }
  function wrongGame(refused: string): void {
    if (refused !== code) return;
    rememberEntryGame(code, 'other');
    void fallback();
  }
  function enter(): void {
    if (disposed) return;
    const nextKey = code + '#' + (claim ?? '');
    if (nextKey === key) return;
    key = nextKey;
    failure = null;
    const game =
      knownEntryGame(code) ??
      untrack(() => savedRooms().list.find((room) => room.code === code)?.game);
    if (game && game !== 'duel') {
      void fallback();
      return;
    }
    // Keep the actual Room instance across Duel rematches and hash changes.
    // Its existing handlers join the new code/claim on the same v1 socket.
    if (kind === 'duel' && Entry) return;
    const request = ++generation;
    discovery?.close();
    discovery = undefined;
    void import('$lib/games/duel/Entry.svelte')
      .then((module) => {
        if (disposed || request !== generation) return;
        kind = 'duel';
        entryProps = {};
        Entry = module.default;
      })
      .catch(() => fail('busy'));
  }
  onMount(() => {
    started = true;
    enter();
  });
  afterNavigate(() => {
    if (started) enter();
  });
  onDestroy(() => {
    disposed = true;
    generation++;
    discovery?.close();
  });
</script>

{#if failure === 'protocol' || failure === 'bad-hello'}
  <main
    class="flex h-full flex-col items-center justify-center gap-3 p-4"
    data-entry-fatal={failure}
  >
    <p role="status" data-conn="offline">{t('session.conn.offline')}</p>
    <p role="alert" data-notice={failure}>{t(errorMessage(failure))}</p>
  </main>
{:else if failure}
  <RoomError error={failure} {code} onhome={() => void goto(resolve('/'))} />
{:else if Entry}
  <Entry
    {...entryProps}
    onwronggame={wrongGame}
    onstate={(joined) => discovery?.firstDuelState(joined)}
  />
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
