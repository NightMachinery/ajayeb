<script lang="ts">
  // Home: composes the Home screen with the sheets it opens. The actual
  // session wiring (creating a room, navigating once it exists) happens
  // here rather than inside the screen components, so Home, NewGameSheet
  // and HotseatSheet stay renderable with no live connection (the gallery).
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Home from '$lib/components/screens/Home.svelte';
  import HotseatSheet from '$lib/components/screens/HotseatSheet.svelte';
  import NewGameSheet from '$lib/components/screens/NewGameSheet.svelte';
  import Settings from '$lib/components/screens/Settings.svelte';
  import { liveGame, savedRooms } from '$lib/session/app.svelte';
  import type { BotLevel, SeatChoice } from '$lib/wire';

  const rooms = savedRooms();

  let aiOpen = $state(false);
  let hotseatOpen = $state(false);
  let onlineOpen = $state(false);
  let settingsOpen = $state(false);

  function toRoom(code: string): Promise<void> {
    return goto(resolve('/play/[code]', { code }));
  }

  async function startAi(opts: { bot: BotLevel; seat: SeatChoice }) {
    const code = await liveGame().session.create({ mode: 'ai', bot: opts.bot, seat: opts.seat });
    aiOpen = false;
    await toRoom(code);
  }

  async function startHotseat(opts: { names: [string, string] }) {
    const code = await liveGame().session.create({ mode: 'hotseat', names: opts.names });
    hotseatOpen = false;
    await toRoom(code);
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
  onSettings={() => (settingsOpen = true)}
/>

<NewGameSheet bind:open={aiOpen} onStart={startAi} />
<HotseatSheet bind:open={hotseatOpen} onStart={startHotseat} />

<Sheet bind:open={onlineOpen} titleId="online-soon-title">
  <div class="flex flex-col gap-4 px-5 pt-1 pb-5">
    <h2 id="online-soon-title" class="font-display text-xl text-fg">{t('home.online.title')}</h2>
    <p class="text-sm text-fg-muted">{t('home.online.comingSoon')}</p>
    <Button onclick={() => (onlineOpen = false)}>{t('shell.close')}</Button>
  </div>
</Sheet>

<Sheet bind:open={settingsOpen} titleId="home-settings-title">
  <div class="flex flex-col gap-4 px-5 pt-1 pb-5">
    <h2 id="home-settings-title" class="font-display text-xl text-fg">
      {t('home.settings.title')}
    </h2>
    <Settings />
  </div>
</Sheet>
