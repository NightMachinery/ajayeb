<script lang="ts">
  // The room of /play/<code>, once the player has a name: it joins the room
  // over the app's live connection and shows the game screen (the lobby
  // first, in an online room still waiting for its second player). The
  // socket stays open when the page goes away; only the room is left. A
  // rematch this player asked for (or chose to follow) navigates here
  // again, which joins the new room.
  import { afterNavigate, goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { onDestroy } from 'svelte';
  import GameScreen from '$lib/components/game/GameScreen.svelte';
  import RoomError from '$lib/components/screens/RoomError.svelte';
  import { t } from '$lib/i18n';
  import { liveGame } from '$lib/session/app.svelte';
  import { setGameContext } from '$lib/session/context';
  import { loadName } from '$lib/session/identity';
  import { installTestHook } from '$lib/session/test-hook';

  const game = setGameContext(liveGame());
  installTestHook(game);
  const { session, presenter } = game;
  // A connection opened before the name was stored says hello again.
  session.setName(loadName());

  // Joins the page's room now (Room may mount after the navigation, once
  // the name prompt is answered) and after every navigation within this
  // page, such as /play/A to /play/B for a rematch; joining B leaves A on
  // the server.
  let joined: string | null = null;
  function joinPageRoom(): void {
    const code = page.params.code;
    if (!code || code === joined) return;
    joined = code;
    session.join(code);
  }
  joinPageRoom();
  afterNavigate(joinPageRoom);

  onDestroy(() => session.leave());

  // Screenshots wait for data-ready on <html>: set once a state is on
  // screen and the fonts have loaded. DOM only: it writes an attribute,
  // never state.
  $effect(() => {
    if (!presenter.frame || !presenter.idle) return;
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

  // The server's answer to a rematch this player asked for: its room's
  // code. Navigation only; it writes no state.
  $effect(() => {
    const code = session.rematchCode;
    if (code && session.rematchAsked && code !== session.code) {
      void goto(resolve('/play/[code]', { code }));
    }
  });

  function toHome() {
    void goto(resolve('/'));
  }
</script>

<svelte:head>
  <title>{t('game.title', { code: session.code ?? '' })} · {t('app.name')}</title>
</svelte:head>

{#if session.roomError}
  <RoomError error={session.roomError} code={session.code ?? page.params.code} onhome={toHome} />
{:else}
  <GameScreen {session} {presenter} onhome={toHome} />
{/if}
