<script lang="ts">
  // /play/<code>: every mode's room. It joins the room over the app's live
  // connection and shows the game screen. The socket stays open when the
  // page goes away; only the room is left. A rematch's new code navigates
  // here again, which joins the new room.
  import { afterNavigate, goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { onDestroy } from 'svelte';
  import GameScreen from '$lib/components/game/GameScreen.svelte';
  import { t } from '$lib/i18n';
  import { liveGame } from '$lib/session/app.svelte';
  import { setGameContext } from '$lib/session/context';
  import { errorMessage, rejectMessage } from '$lib/session/messages';
  import { installTestHook } from '$lib/session/test-hook';

  const game = setGameContext(liveGame());
  installTestHook(game);
  const { session, presenter } = game;

  // Runs on arrival and after every navigation within this page, such as
  // /play/A to /play/B for a rematch; joining B leaves A on the server.
  afterNavigate(() => {
    const code = page.params.code;
    if (code) session.join(code);
  });

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

  // The server's answer to a rematch request: its room's code. Navigation
  // only; it writes no state.
  $effect(() => {
    const code = session.rematchCode;
    if (code && code !== session.code) void goto(resolve('/play/[code]', { code }));
  });

  function toHome() {
    void goto(resolve('/'));
  }
</script>

<svelte:head>
  <title>{t('game.title', { code: session.code ?? '' })} · {t('app.name')}</title>
</svelte:head>

{#if session.roomError}
  <main class="flex h-full flex-col items-center justify-center gap-3 px-4 text-center">
    <p class="text-lg text-fg" role="alert" data-room-error={session.roomError}>
      {t(
        session.roomError === 'rate'
          ? rejectMessage(session.roomError)
          : errorMessage(session.roomError)
      )}
    </p>
    <a
      class="inline-flex min-h-11 items-center rounded-control bg-accent px-4 font-semibold text-on-accent"
      href={resolve('/')}
    >
      {t('app.home')}
    </a>
  </main>
{:else}
  <GameScreen {session} {presenter} onhome={toHome} />
{/if}
