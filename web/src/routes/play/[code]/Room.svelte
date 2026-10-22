<script lang="ts">
  // The room of /play/<code>, once the player has a name: it joins the room
  // over the app's live connection and shows the game screen (the lobby
  // first, in an online room still waiting for its second player). The
  // socket stays open when the page goes away; only the room is left. A
  // rematch this player asked for (or chose to follow) navigates here
  // again, which joins the new room.
  //
  // A device link (/play/<code>#claim=..., lib/session/claim.ts) joins with
  // its claim, and so does a room this browser saved with one (reopened
  // from Home without the fragment). The claim stays in the address bar, so
  // a reload plays the same seat; when this device resets its own link the
  // address bar takes the new claim.
  import { afterNavigate, goto, replaceState } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { onDestroy } from 'svelte';
  import GameScreen from '$lib/components/game/GameScreen.svelte';
  import RoomError from '$lib/components/screens/RoomError.svelte';
  import { i18n, t } from '$lib/i18n';
  import { createFrameReadiness } from '$lib/ui/frame-readiness';
  import { liveGame, savedRooms } from '$lib/session/app.svelte';
  import { claimFragment, parseClaimFragment } from '$lib/session/claim';
  import { setGameContext } from '$lib/session/context';
  import { loadName } from '$lib/session/identity';
  import { normalizeCode } from '$lib/session/session.svelte';
  import { installTestHook } from '$lib/session/test-hook';

  interface Props {
    onwronggame?: (code: string) => void;
    onstate?: (code: string) => void;
  }
  let { onwronggame, onstate }: Props = $props();

  const game = setGameContext(liveGame());
  installTestHook(game);
  const { session, presenter } = game;
  // A connection opened before the name was stored says hello again.
  // A submitted in-memory name survives blocked name storage at handoff.
  session.setName(loadName() || session.name);

  // Joins the page's room now (Room may mount after the navigation, once
  // the name prompt is answered) and after every navigation within this
  // page, such as /play/A to /play/B for a rematch; joining B leaves A on
  // the server. A device link opened over this page changes only the
  // fragment, which is no navigation to SvelteKit, so hashchange joins too.
  // The claim is read from location, which is current in both cases.
  let joined: string | null = null;
  function joinPageRoom(): void {
    const code = page.params.code;
    if (!code) return;
    const claim = parseClaimFragment(location.hash) ?? savedRooms().claimFor(normalizeCode(code));
    const key = `${code}#${claim ?? ''}`;
    if (key === joined) return;
    joined = key;
    session.join(code, claim);
  }
  joinPageRoom();
  afterNavigate(joinPageRoom);

  const offWrongGame = session.onWrongGame((code) => onwronggame?.(code));
  const offState = session.subscribe((frame) => onstate?.(frame.code));
  onDestroy(() => {
    offWrongGame();
    offState();
    if (session.roomError !== 'wrong-game') session.leave();
  });

  // Screenshots require the current idle frame, loaded fonts and a rAF stamp.
  // Reuse settled font readiness without forcing layout on every frame.
  const markFrameReady = createFrameReadiness();
  $effect(() => {
    if (!presenter.frame || !presenter.idle) return;
    return markFrameReady(document.documentElement, document.fonts, i18n.locale);
  });

  // The server's answer to a rematch this player asked for: its room's
  // code (and this device's claim there, when it plays on one).
  // Navigation only; it writes no state.
  $effect(() => {
    const code = session.rematchCode;
    if (code && session.rematchAsked && code !== session.code) {
      const claim = session.rematchClaim;
      const path = resolve('/play/[code]', { code });
      // The path is resolved; only the fragment (the claim) is added to it.
      // eslint-disable-next-line svelte/no-navigation-without-resolve
      void goto(claim ? path + claimFragment(claim) : path);
    }
  });

  // This device reset its own link: the address bar takes the new claim,
  // so a reload still plays the seat. Only a change of the session's claim
  // away from the one the address showed counts: a new fragment typed or
  // opened over the old one (joinPageRoom joins with it) must win, not be
  // put back. URL only; `synced` is a plain variable, not state.
  let synced: string | null = null;
  $effect(() => {
    const claim = session.claim;
    const shown = parseClaimFragment(page.url.hash);
    const before = synced;
    synced = claim;
    if (!claim || !shown || shown === claim || before !== shown) return;
    const path = resolve('/play/[code]', { code: page.params.code ?? '' });
    // The same page, with only the fragment (the claim) replaced.
    // eslint-disable-next-line svelte/no-navigation-without-resolve
    replaceState(path + claimFragment(claim), page.state);
  });

  function toHome() {
    void goto(resolve('/'));
  }
</script>

<svelte:window onhashchange={joinPageRoom} />

<svelte:head>
  <title>{t('game.title', { code: session.code ?? '' })} · {t('app.name')}</title>
</svelte:head>

{#if session.roomError === 'wrong-game'}
  <!-- Refused before admission; the parent switches to lazy discovery. -->
{:else if session.roomError}
  <RoomError error={session.roomError} code={session.code ?? page.params.code} onhome={toHome} />
{:else}
  <GameScreen {session} {presenter} onhome={toHome} />
{/if}
