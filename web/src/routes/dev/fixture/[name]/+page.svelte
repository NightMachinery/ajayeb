<script lang="ts">
  // /dev/fixture/<name>: one recorded game state, played through the
  // fixture transport into the debug board, with no server. Screenshots
  // wait for data-ready on <html>, set once the state is on screen and the
  // fonts have loaded.
  //
  // The recorded fixtures are large, but SvelteKit splits code by route, so
  // they load only with the /dev/fixture pages. Each fixture is opened by a
  // full page load or from the list, so the game is made once, when the
  // page starts.
  import { page } from '$app/state';
  import { onDestroy } from 'svelte';
  import { t } from '$lib/i18n';
  import DebugBoard from '$lib/session/DebugBoard.svelte';
  import { setGameContext, type GameContext } from '$lib/session/context';
  import { FixtureTransport } from '$lib/session/fixture-transport';
  import { fixtureFrames } from '$lib/session/fixtures';
  import { newClientId, newToken } from '$lib/session/identity';
  import { Presenter } from '$lib/session/presenter.svelte';
  import { Session } from '$lib/session/session.svelte';
  import { installTestHook, uninstallTestHook } from '$lib/session/test-hook';

  const name = page.params.name ?? '';
  const frames = fixtureFrames(name);

  function makeGame(first: NonNullable<typeof frames>): GameContext {
    // A throwaway identity: nothing on this page reaches a server.
    const session = new Session({
      transport: new FixtureTransport({ frames: first }),
      identity: { token: newToken(), clientId: newClientId(), name: '' }
    });
    session.join(first[0]!.code);
    return setGameContext({ session, presenter: new Presenter(session) });
  }

  const game = frames ? makeGame(frames) : null;
  if (game) installTestHook(game);

  // DOM only: it writes an attribute, never state.
  $effect(() => {
    if (!game?.presenter.frame) return;
    const root = document.documentElement;
    let live = true;
    void document.fonts.ready.then(() =>
      requestAnimationFrame(() => {
        if (live) root.dataset['ready'] = '1';
      })
    );
    return () => {
      live = false;
      delete root.dataset['ready'];
    };
  });

  onDestroy(() => {
    if (!game) return;
    uninstallTestHook(game);
    game.presenter.destroy();
    game.session.close();
  });
</script>

{#if game}
  <DebugBoard session={game.session} presenter={game.presenter} />
{:else}
  <main class="flex h-full items-center justify-center px-4">
    <p class="text-fg" role="alert">{t('dev.fixture.unknown', { name })}</p>
  </main>
{/if}
