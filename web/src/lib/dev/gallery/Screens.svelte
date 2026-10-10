<script lang="ts">
  // M5.S05's screens, for a visual check in both themes: Home (with and
  // without a saved game to continue), the two new-game sheets, Settings,
  // and the hot-seat pass-device interstitial. No live connection: the
  // sheets' Start buttons are wired to a no-op here, same as the rest of
  // this gallery page's demo controls (see dev/gallery/+page.svelte).
  import { t } from '$lib/i18n';
  import Home from '$lib/components/screens/Home.svelte';
  import HotseatSheet from '$lib/components/screens/HotseatSheet.svelte';
  import NewGameSheet from '$lib/components/screens/NewGameSheet.svelte';
  import PassDevice from '$lib/components/screens/PassDevice.svelte';
  import Settings from '$lib/components/screens/Settings.svelte';
  import type { SavedRoom } from '$lib/session/rooms.svelte';

  const sampleRooms: readonly SavedRoom[] = [
    { code: 'ZAGROS41', mode: 'ai', lastSeen: Date.now() },
    { code: 'K7QX2M', mode: 'hotseat', lastSeen: Date.now() - 3600_000 }
  ];

  let aiOpen = $state(false);
  let hotseatOpen = $state(false);

  const noop = () => {};
</script>

<div class="flex flex-col gap-10">
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Home — with a saved game <!-- i18n-ok --></h3>
    <div class="relative h-[700px] overflow-hidden rounded-panel border border-rule">
      <Home
        rooms={sampleRooms}
        onContinue={noop}
        onPlayAi={noop}
        onPassAndPlay={noop}
        onPlayOnline={noop}
        onSettings={noop}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Home — no saved game <!-- i18n-ok --></h3>
    <div class="relative h-[700px] overflow-hidden rounded-panel border border-rule">
      <Home
        rooms={[]}
        onContinue={noop}
        onPlayAi={noop}
        onPassAndPlay={noop}
        onPlayOnline={noop}
        onSettings={noop}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">NewGameSheet (vs AI) <!-- i18n-ok --></h3>
    <button
      type="button"
      class="min-h-11 w-fit rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
      onclick={() => (aiOpen = true)}
    >
      {t('home.playAi')}
    </button>
    <NewGameSheet bind:open={aiOpen} onStart={noop} />
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">HotseatSheet <!-- i18n-ok --></h3>
    <button
      type="button"
      class="min-h-11 w-fit rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
      onclick={() => (hotseatOpen = true)}
    >
      {t('home.passAndPlay')}
    </button>
    <HotseatSheet bind:open={hotseatOpen} onStart={noop} />
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Settings <!-- i18n-ok --></h3>
    <div class="rounded-panel border border-rule bg-panel p-5">
      <Settings />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">PassDevice — Sun <!-- i18n-ok --></h3>
    <div class="relative h-96 overflow-hidden rounded-panel border border-rule">
      <PassDevice
        nextName="Darius"
        nextSeat={0}
        summary={[t('home.demo.passSummary1'), t('home.demo.passSummary2')]}
        canUndo
        previousName="Roshan"
        onReady={noop}
        onUndo={noop}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">PassDevice — Moon <!-- i18n-ok --></h3>
    <div class="relative h-96 overflow-hidden rounded-panel border border-rule">
      <PassDevice
        nextName="Roshan"
        nextSeat={1}
        summary={[t('home.demo.passSummary1'), t('home.demo.passSummary2')]}
        canUndo={false}
        onReady={noop}
        onUndo={noop}
      />
    </div>
  </section>
</div>
