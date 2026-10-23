<script lang="ts">
  // Haftgane's lazy entry, for a table or a shared lobby: the Session takes
  // over the bootstrap's welcomed v2 connection and its newest snapshot,
  // then asks for a resync, since the bootstrap kept only that snapshot and
  // it may carry no history. The minimal development screen shows it until
  // the table and lobby screens arrive.
  import { onDestroy, onMount, untrack } from 'svelte';
  import { entryIdentity } from '$lib/entry/identity';
  import type { EntryProps } from '$lib/entry/types';
  import { savedRooms } from '$lib/session/saved-rooms.svelte';
  import TableDev from './dev/TableDev.svelte';
  import { TablePresenter } from './session/presenter.svelte';
  import { TableSession } from './session/session.svelte';
  import { installTableHook } from './session/test-hook';

  let { connection, code, claim }: EntryProps = $props();

  // The connection, code and claim are fixed for this entry's life: a new
  // code or claim mounts a new entry over a new connection.
  const game = untrack(() => {
    if (!connection || !code) return null;
    const session = new TableSession({
      connection,
      code,
      claim: claim ?? null,
      clientId: entryIdentity().clientId,
      rooms: savedRooms()
    });
    return { session, presenter: new TablePresenter(session) };
  });
  const removeHook = game ? installTableHook(game.session, game.presenter) : () => {};
  onMount(() => game?.session.resync());
  onDestroy(() => {
    removeHook();
    game?.presenter.destroy();
    game?.session.close();
  });
</script>

{#if game}
  <TableDev session={game.session} presenter={game.presenter} />
{/if}
