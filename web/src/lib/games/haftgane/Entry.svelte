<script lang="ts">
  // Haftgane's lazy entry, for a table or a shared lobby: the Session takes
  // over the bootstrap's welcomed v2 connection and its newest snapshot,
  // then asks for a resync, since the bootstrap kept only that snapshot and
  // it may carry no history. The real lobby is lazy; running tables keep
  // the development screen until the parallel table-screen slice lands.
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { claimFragment } from '$lib/session/claim';
  import Button from '$lib/components/shell/Button.svelte';
  import { t } from '$lib/lobby/i18n/messages';
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
  $effect(() => {
    const s = game?.session;
    if (s?.rematchAsked && s.rematchCode) {
      /* eslint-disable svelte/no-navigation-without-resolve -- resolved room path plus private claim fragment */
      void goto(
        resolve('/play/[code]', { code: s.rematchCode }) +
          (s.rematchClaim ? claimFragment(s.rematchClaim) : '')
      );
      /* eslint-enable svelte/no-navigation-without-resolve */
    }
  });
</script>

{#if game}
  {#if game.session.lobby}
    {#await import('$lib/lobby/LobbyScreen.svelte') then { default: LobbyScreen }}
      <LobbyScreen session={game.session} />
    {/await}
  {:else}
    {#if game.session.table?.status === 'finished'}
      <div class="bg-panel p-3" data-rematch-actions>
        {#if game.session.rematchError}
          <p role="alert" class="mb-2 text-sm" data-rematch-error={game.session.rematchError}>
            {t(
              game.session.rematchError === 'forbidden'
                ? 'lobby.rematchParticipant'
                : 'lobby.failed'
            )}
          </p>
        {/if}
        <Button
          disabled={game.session.conn !== 'open' ||
            (game.session.rematchAsked && !game.session.rematchCode)}
          onclick={() => game.session.rematch()}
          data-act="lobby-rematch">{t('lobby.rematch')}</Button
        >
      </div>
    {/if}
    <TableDev session={game.session} presenter={game.presenter} />
  {/if}
{/if}
