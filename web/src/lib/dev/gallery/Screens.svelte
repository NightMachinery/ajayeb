<script lang="ts">
  // M5.S05's screens, for a visual check in both themes: Home (with and
  // without a saved game to continue), the two new-game sheets, Settings,
  // the hot-seat pass-device interstitial, the sound board and the hint
  // settings (Hints.svelte), the rules references of both games
  // (Rules.svelte, HaftganeRules.svelte), Haftgane's table screen leaves
  // (HaftganeTable.svelte), and the tutorial's components (Tutorial.svelte).
  // No live connection: the
  // sheets' Start buttons are wired to a no-op here, same as the rest of
  // this gallery page's demo controls (see dev/gallery/+page.svelte).
  import { onMount, type Component } from 'svelte';
  import { t } from '$lib/i18n';
  import Home from '$lib/components/screens/Home.svelte';
  import HotseatSheet from '$lib/components/screens/HotseatSheet.svelte';
  import NewGameSheet from '$lib/components/screens/NewGameSheet.svelte';
  import PassDevice from '$lib/components/screens/PassDevice.svelte';
  import Settings from '$lib/components/screens/Settings.svelte';
  import HaftganeRules from './HaftganeRules.svelte';
  import Hints from './Hints.svelte';
  import Online from './Online.svelte';
  import Rules from './Rules.svelte';
  import SoundBoard from './SoundBoard.svelte';
  import Tutorial from './Tutorial.svelte';
  import type { SavedRoom } from '$lib/session/rooms.svelte';

  // Haftgane's table leaves load with their section only: a static import
  // would split them into a chunk shared with Haftgane's entry, whose name
  // every page's route map then carries (docs/architecture.md, "Chunks").
  let HaftganeTable = $state.raw<Component | null>(null);
  let HaftganeEnd = $state.raw<Component | null>(null);
  onMount(() => {
    void import('./HaftganeTable.svelte').then((m) => (HaftganeTable = m.default));
    void import('./HaftganeEnd.svelte').then((m) => (HaftganeEnd = m.default));
  });

  const sampleRooms: readonly SavedRoom[] = [
    { code: 'ZAGROS41', mode: 'ai', lastSeen: Date.now() },
    { code: 'K7QX2M', mode: 'hotseat', lastSeen: Date.now() - 3600_000 },
    {
      code: 'TUTOR1',
      mode: 'ai',
      lastSeen: Date.now() - 7200_000,
      bot: 'easy',
      age: 1,
      tutorial: true
    }
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
        onJoinCode={noop}
        onSettings={noop}
        tutorialDone
        onTutorial={noop}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Home — no saved game, the tutorial offer <!-- i18n-ok -->
    </h3>
    <div class="relative h-[700px] overflow-hidden rounded-panel border border-rule">
      <Home
        rooms={[]}
        onContinue={noop}
        onPlayAi={noop}
        onPassAndPlay={noop}
        onPlayOnline={noop}
        onSettings={noop}
        tutorialOffer
        onTutorial={noop}
        onDeclineTutorial={noop}
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
    <NewGameSheet bind:open={aiOpen} onStart={noop} onTutorial={noop} />
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

  <section class="flex flex-col gap-2" data-testid="gallery-sound-section">
    <h3 class="text-sm font-semibold text-fg-muted">{t('dev.gallery.sound')}</h3>
    <SoundBoard />
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

  <Online />

  <Hints />

  <Rules />

  <HaftganeRules />

  {#if HaftganeTable}<HaftganeTable />{/if}

  {#if HaftganeEnd}<HaftganeEnd />{/if}

  <Tutorial />
</div>
