<script lang="ts">
  // M7.S01's online screens, for a visual check in both themes: the lobby
  // (with a fixed sample code, its link and the QR code of that link), the
  // name prompt, the create and join sheets, a room that cannot be opened,
  // and the connection states as the bars and strips show them. No live
  // connection: every action is a no-op. The full game screen in these
  // states (the lobby, an opponent away, a spectator's view) is in the Game
  // screen section, from the online-... fixtures.
  import { t } from '$lib/i18n';
  import ConnectionPill from '$lib/components/hud/ConnectionPill.svelte';
  import PresenceDot from '$lib/components/hud/PresenceDot.svelte';
  import SpectatorCount from '$lib/components/hud/SpectatorCount.svelte';
  import TurnBar from '$lib/components/hud/TurnBar.svelte';
  import JoinSheet from '$lib/components/screens/JoinSheet.svelte';
  import Lobby from '$lib/components/screens/Lobby.svelte';
  import NamePrompt from '$lib/components/screens/NamePrompt.svelte';
  import OnlineSheet from '$lib/components/screens/OnlineSheet.svelte';
  import RoomError from '$lib/components/screens/RoomError.svelte';
  import type { SeatInfo } from '$lib/wire';

  const CODE = 'K7QX2M';
  const URL = `https://ajayeb.example/play/${CODE}`;
  const NAMES = ['Darius', 'Roshan'] as const;

  const lobbySeats: SeatInfo[] = [
    { seat: 0, name: NAMES[0], kind: 'human', connected: true, you: true },
    { seat: 1, name: '', kind: 'open', connected: false, you: false }
  ];

  let onlineOpen = $state(false);
  let joinOpen = $state(false);

  const noop = () => {};
</script>

<div class="flex flex-col gap-10" data-gallery-online>
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Lobby — waiting, with the QR code <!-- i18n-ok -->
    </h3>
    <div class="relative h-[780px] overflow-hidden rounded-panel border border-rule bg-table">
      <Lobby
        code={CODE}
        url={URL}
        seats={lobbySeats}
        canShare
        oncopy={noop}
        onshare={noop}
        onhome={noop}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Lobby — reconnecting, no share sheet <!-- i18n-ok -->
    </h3>
    <div class="relative h-[780px] overflow-hidden rounded-panel border border-rule bg-table">
      <Lobby
        code={CODE}
        url={URL}
        seats={lobbySeats}
        conn="reconnecting"
        oncopy={noop}
        onhome={noop}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">NamePrompt <!-- i18n-ok --></h3>
    <div class="relative h-[420px] overflow-hidden rounded-panel border border-rule bg-table">
      <NamePrompt suggestion="Roxana" code={CODE} onsubmit={noop} />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">OnlineSheet and JoinSheet <!-- i18n-ok --></h3>
    <div class="flex flex-wrap gap-2">
      <button
        type="button"
        class="min-h-11 w-fit rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
        onclick={() => (onlineOpen = true)}
      >
        {t('home.playOnline')}
      </button>
      <button
        type="button"
        class="min-h-11 w-fit rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
        onclick={() => (joinOpen = true)}
      >
        {t('home.joinCode')}
      </button>
    </div>
    <OnlineSheet
      bind:open={onlineOpen}
      name={NAMES[0]}
      suggestion="Roxana"
      onStart={noop}
      onJoin={() => {
        onlineOpen = false;
        joinOpen = true;
      }}
    />
    <JoinSheet bind:open={joinOpen} initial="k7q x2" onJoin={noop} />
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">RoomError — not found <!-- i18n-ok --></h3>
    <div class="relative h-[360px] overflow-hidden rounded-panel border border-rule bg-table">
      <RoomError error="room-not-found" code={CODE} onhome={noop} />
    </div>
  </section>

  <section class="flex flex-col gap-3">
    <h3 class="text-sm font-semibold text-fg-muted">Connection states <!-- i18n-ok --></h3>
    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-3">
      <ConnectionPill conn="reconnecting" />
      <ConnectionPill conn="offline" />
      <PresenceDot presence="connected" name={NAMES[1]} labelled />
      <PresenceDot presence="away" name={NAMES[1]} />
      <SpectatorCount count={2} />
    </div>
    <div class="flex flex-col gap-2 overflow-hidden rounded-panel border border-rule">
      <TurnBar state={{ kind: 'reconnecting' }} names={NAMES} />
      <TurnBar state={{ kind: 'offline' }} names={NAMES} />
      <TurnBar state={{ kind: 'waitingOpponent', seat: 1 }} names={NAMES} />
      <TurnBar state={{ kind: 'waitingOpponent', seat: 1, away: true }} names={NAMES} />
      <TurnBar state={{ kind: 'spectating' }} names={NAMES} />
    </div>
  </section>
</div>
