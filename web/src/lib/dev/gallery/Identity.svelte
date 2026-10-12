<script lang="ts">
  // M10.ID1: who is who, for a visual check in both themes. Two players who
  // both call themselves Sara ("Sara" and "Sara 2", from the recorded
  // state-names-numbered frame, seen by the second), the player's own seat
  // marked in the strips, the city panels, the lobby and the log, and the
  // device link: its compact button and its sheet in each state. No live
  // connection: the sheet's actions only change what it shows.
  import { t } from '$lib/i18n';
  import CityPanel from '$lib/components/city/CityPanel.svelte';
  import PlayerStrip from '$lib/components/city/PlayerStrip.svelte';
  import DeviceLinkButton from '$lib/components/identity/DeviceLinkButton.svelte';
  import DeviceLinkSheet from '$lib/components/identity/DeviceLinkSheet.svelte';
  import SelfMark from '$lib/components/identity/SelfMark.svelte';
  import Lobby from '$lib/components/screens/Lobby.svelte';
  import LogSheet from '$lib/components/sheets/LogSheet.svelte';
  import { seatDisplayName } from '$lib/view/display-name';
  import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
  import type { GameEvent, SeatInfo, State } from '$lib/wire';

  function stateFixture(name: (typeof fixtureNames)[number]): State {
    const frame = fixtures[fixtureNames.indexOf(name)];
    if (frame?.t !== 'state') throw new Error(`no state fixture named ${name}`);
    return frame;
  }

  const frame = stateFixture('state-names-numbered');
  const view = frame.view;
  const names = [
    seatDisplayName(frame.room.seats[0], t('board.crest.sun')),
    seatDisplayName(frame.room.seats[1], t('board.crest.moon'))
  ] as const;
  const LINK = `https://ajayeb.example/play/${frame.code}#claim=q3Vb8Kx0Zr7TnW2mYc5JhA`;
  const lobbySeats: SeatInfo[] = frame.room.seats.map((s) => ({ ...s }));
  const logEvents: GameEvent[] = stateFixture('state-play').events;

  type SheetState = {
    own: boolean;
    status: 'loading' | 'ready' | 'failed';
    copy: 'idle' | 'copied' | 'failed';
    reset: boolean;
  };
  const SHEETS: Record<string, SheetState> = {
    copied: { own: true, status: 'ready', copy: 'copied', reset: false },
    copyFailed: { own: true, status: 'ready', copy: 'failed', reset: false },
    loading: { own: true, status: 'loading', copy: 'idle', reset: false },
    failed: { own: true, status: 'failed', copy: 'idle', reset: false },
    adminReset: { own: false, status: 'ready', copy: 'copied', reset: true }
  };
  let sheet = $state<string | null>(null);
  let sheetOpen = $state(false);
  let logOpen = $state(false);
  const shown = $derived(sheet ? SHEETS[sheet] : undefined);

  function show(name: string): void {
    sheet = name;
    sheetOpen = true;
  }

  const noop = () => {};
</script>

<div class="flex flex-col gap-10" data-gallery-identity>
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Strips: two Saras, the second is you <!-- i18n-ok -->
    </h3>
    <div class="flex max-w-md flex-col gap-2" data-gallery-identity-strips>
      {#each [0, 1] as const as seat (seat)}
        <PlayerStrip
          name={names[seat]}
          {seat}
          self={seat === 1}
          player={view.players[seat]}
          military={view.military}
          score={view.scorePreview[seat]}
          presence="connected"
          onopen={noop}
          openLabel={names[seat]}
          onwonders={noop}
        />
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">City panels <!-- i18n-ok --></h3>
    <div class="grid max-w-3xl grid-cols-2 gap-3">
      {#each [0, 1] as const as seat (seat)}
        <CityPanel
          name={names[seat]}
          {seat}
          self={seat === 1}
          player={view.players[seat]}
          military={view.military}
          score={view.scorePreview[seat]}
        />
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      The device link button and its sheet <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-3">
      <span class="flex items-center gap-1.5 text-sm font-semibold text-fg" dir="auto">
        {names[1]}
        <SelfMark />
        <DeviceLinkButton
          seat={1}
          label={t('online.link.buttonOwn')}
          onclick={() => show('copied')}
        />
      </span>
      <span class="flex items-center gap-1.5 text-sm font-semibold text-fg" dir="auto">
        {names[0]}
        <DeviceLinkButton
          seat={0}
          label={t('online.link.button', { name: names[0] })}
          onclick={() => show('adminReset')}
        />
      </span>
    </div>
    <div class="flex flex-wrap gap-2">
      {#each Object.keys(SHEETS) as name (name)}
        <button
          type="button"
          class="min-h-11 w-fit rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
          data-gallery-sheet={name}
          onclick={() => show(name)}
        >
          {name}
        </button>
      {/each}
    </div>
    {#if shown}
      <DeviceLinkSheet
        bind:open={sheetOpen}
        seat={shown.own ? 1 : 0}
        name={shown.own ? names[1] : names[0]}
        own={shown.own}
        link={shown.status === 'ready' ? LINK : null}
        status={shown.status}
        copy={shown.copy}
        reset={shown.reset}
        oncopy={noop}
        onreset={noop}
        onretry={noop}
      />
    {/if}
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Lobby with numbered names, and the log <!-- i18n-ok -->
    </h3>
    <button
      type="button"
      class="min-h-11 w-fit rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
      data-gallery-log
      onclick={() => (logOpen = true)}
    >
      {t('board.logSheet.title')}
    </button>
    <LogSheet bind:open={logOpen} events={logEvents} {names} selfSeat={1} />
    <div class="relative h-[780px] overflow-hidden rounded-panel border border-rule bg-table">
      <Lobby
        code={frame.code}
        url={`https://ajayeb.example/play/${frame.code}`}
        seats={lobbySeats}
        oncopy={noop}
        onhome={noop}
      />
    </div>
  </section>
</div>
