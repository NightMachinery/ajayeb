<script lang="ts">
  // Home: composes the Home screen with the sheets it opens. The actual
  // session wiring (creating a room, navigating once it exists) happens
  // here rather than inside the screen components, so Home, NewGameSheet
  // and HotseatSheet stay renderable with no live connection (the gallery).
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Home from '$lib/components/screens/Home.svelte';
  import HotseatSheet from '$lib/components/screens/HotseatSheet.svelte';
  import JoinSheet from '$lib/components/screens/JoinSheet.svelte';
  import NewGameSheet from '$lib/components/screens/NewGameSheet.svelte';
  import OnlineSheet from '$lib/components/screens/OnlineSheet.svelte';
  import Settings from '$lib/components/screens/Settings.svelte';
  import { liveGame, savedRooms } from '$lib/session/app.svelte';
  import { loadName, saveName } from '$lib/session/identity';
  import { suggestName } from '$lib/ui/online';
  import { tips } from '$lib/ui/tips.svelte';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import type { BotLevel, SeatChoice } from '$lib/wire';

  const rooms = savedRooms();

  // ?play=ai (the tutorial's New game) opens the vs-AI sheet.
  let aiOpen = $state(page.url.searchParams.get('play') === 'ai');
  let hotseatOpen = $state(false);
  let onlineOpen = $state(false);
  let joinOpen = $state(false);
  let settingsOpen = $state(false);

  // The stored display name, read once; the online sheet saves a new one.
  let storedName = $state(loadName());
  const suggestion = suggestName(t('online.name.suggestions'));

  function toRoom(code: string): Promise<void> {
    return goto(resolve('/play/[code]', { code }));
  }

  async function startAi(opts: { bot: BotLevel; seat: SeatChoice }) {
    const code = await liveGame().session.create({ mode: 'ai', bot: opts.bot, seat: opts.seat });
    aiOpen = false;
    await toRoom(code);
  }

  /** The tutorial: a guided ai room against the Easy bot, from seat 0. */
  async function startTutorial(): Promise<void> {
    const code = await liveGame().session.create({
      mode: 'ai',
      bot: 'easy',
      seat: 0,
      tutorial: true
    });
    aiOpen = false;
    await toRoom(code);
  }

  // The query is dropped once read, so a reload does not open it again.
  onMount(() => {
    if (page.url.searchParams.has('play')) {
      void goto(resolve('/'), { replaceState: true, keepFocus: true, noScroll: true });
    }
  });

  /** Home's offer and row: a failure leaves the player on Home. */
  function tutorialFromHome(): void {
    startTutorial().catch((err: unknown) => {
      if (import.meta.env.DEV) console.warn('tutorial: create failed', err);
    });
  }

  async function startHotseat(opts: { names: [string, string] }) {
    const code = await liveGame().session.create({ mode: 'hotseat', names: opts.names });
    hotseatOpen = false;
    await toRoom(code);
  }

  async function startOnline(opts: { name: string; seat: SeatChoice }) {
    storedName = saveName(opts.name);
    const { session } = liveGame();
    // The hello name is the one a joined seat gets; the create names it too.
    session.setName(storedName);
    const code = await session.create({
      mode: 'online',
      seat: opts.seat,
      names: [storedName, storedName]
    });
    onlineOpen = false;
    await toRoom(code);
  }

  function joinRoom(code: string): Promise<void> {
    joinOpen = false;
    return toRoom(code);
  }

  function openJoin(): void {
    onlineOpen = false;
    joinOpen = true;
  }

  function continueRoom(code: string): Promise<void> {
    return toRoom(code);
  }
</script>

<svelte:head>
  <title>{t('app.name')}</title>
</svelte:head>

<Home
  rooms={rooms.list}
  onContinue={continueRoom}
  onPlayAi={() => (aiOpen = true)}
  onPassAndPlay={() => (hotseatOpen = true)}
  onPlayOnline={() => (onlineOpen = true)}
  onJoinCode={() => (joinOpen = true)}
  onSettings={() => (settingsOpen = true)}
  tutorialOffer={rooms.list.length === 0 &&
    tips.value.tutorial.offer === 'new' &&
    !tips.value.tutorial.done}
  tutorialDone={tips.value.tutorial.done}
  onTutorial={tutorialFromHome}
  onDeclineTutorial={() => tips.declineOffer()}
/>

<NewGameSheet bind:open={aiOpen} onStart={startAi} onTutorial={tutorialFromHome} />
<HotseatSheet bind:open={hotseatOpen} onStart={startHotseat} />

<OnlineSheet
  bind:open={onlineOpen}
  name={storedName}
  {suggestion}
  onStart={startOnline}
  onJoin={openJoin}
/>
<JoinSheet bind:open={joinOpen} onJoin={joinRoom} />

<Sheet bind:open={settingsOpen} titleId="home-settings-title">
  <div class="flex flex-col gap-4 px-5 pt-1 pb-5 split:pt-5">
    <h2 id="home-settings-title" class="font-display text-xl text-fg">
      {t('home.settings.title')}
    </h2>
    <Settings />
  </div>
</Sheet>
