<script lang="ts">
  // Haftgane's admin, role and spectator leaves
  // (M18.admin-controls-resign-and-spectator), each standalone with plain
  // props: the role tags, a confirmation, the spectator list with role
  // buttons and the previous-seat hint, the spectator panel over the city
  // fixture's seats, the top bar with Undo, the seat leaves with their
  // tags, and the seat menu on a human's seat and a bot's. The people are
  // made up here; the headings are dev labels, not shipped copy.
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '$lib/games/haftgane/i18n/messages.svelte';
  import AdminConfirm from '$lib/games/haftgane/table/AdminConfirm.svelte';
  import AdminSeatMenu from '$lib/games/haftgane/table/AdminSeatMenu.svelte';
  import FarSeatChip from '$lib/games/haftgane/table/FarSeatChip.svelte';
  import RoleTag from '$lib/games/haftgane/table/RoleTag.svelte';
  import SpectatorList from '$lib/games/haftgane/table/SpectatorList.svelte';
  import SpectatorPanel from '$lib/games/haftgane/table/SpectatorPanel.svelte';
  import TableTopBar from '$lib/games/haftgane/table/TableTopBar.svelte';
  import YourStrip from '$lib/games/haftgane/table/YourStrip.svelte';
  import { botLevels, type SeatPerson } from '$lib/games/haftgane/table/admin';
  import { where } from '$lib/games/haftgane/table/seat-names';
  import views from './haftgane-city-views.json';

  const seats: Seat[] = views.mid.seats;
  const names: Record<number, string> = {
    0: 'Ana',
    1: 'Lena',
    2: 'Bo',
    3: 'Aziz',
    4: 'Mahsa',
    5: 'Cyrus',
    6: 'Wen'
  };
  const bots = [1, 3, 5];
  const people: SeatPerson[] = [
    { userId: 'u-roshan', name: 'Roshan', role: 'temporary', connected: true },
    { userId: 'u-kian', name: 'Kian', role: null, connected: true, previous: 4 },
    { userId: 'u-sara', name: 'Sara', role: 'admin', connected: false },
    { userId: 'u-you', name: 'Darya', role: null, connected: true }
  ];
  const roles = {
    'u-kian': { promote: true, demote: false },
    'u-sara': { promote: false, demote: true }
  };
  const noop = () => {};
  const own = seats.find((s) => s.relation === 'self')!;
  const far = seats.find((s) => s.relation === 'other')!;

  let follow = $state(own.seat);
  let menu = $state<'human' | 'bot' | null>(null);
</script>

<section class="gallery" data-gallery-admin>
  <h3>Admin controls, roles and the spectator view <!-- i18n-ok --></h3>
  <div class="row">
    <button type="button" class="open" onclick={() => (menu = 'human')}
      >Open a human seat's menu <!-- i18n-ok --></button
    >
    <button type="button" class="open" onclick={() => (menu = 'bot')}
      >Open a bot seat's menu <!-- i18n-ok --></button
    >
  </div>

  <div class="box row tags">
    {#each ['creator', 'admin', 'temporary'] as const as role (role)}
      <RoleTag {role} name="Ana" />
      <RoleTag {role} name="Ana" compact />
    {/each}
  </div>

  <div class="box">
    <TableTopBar
      phase="turn"
      age={1}
      turn={2}
      turns={6}
      pass="left"
      passTo="Lena"
      finished={false}
      onmenu={noop}
      onundo={noop}
      spectators={3}
      onspectators={noop}
    />
  </div>

  <div class="grid">
    <div class="box stack">
      <FarSeatChip
        seat={far}
        name={names[far.seat]!}
        where={where(far, seats.length)}
        bot={false}
        status="choosing"
        role="admin"
      />
      <YourStrip
        seat={own}
        name={names[own.seat]!}
        where={where(own, seats.length)}
        own
        bot={false}
        status="confirmed"
        role="creator"
      />
      <AdminConfirm text={ht('hg.admin.confirm.resign')} onconfirm={noop} onback={noop} />
    </div>
    <div class="box">
      <SpectatorList {people} you="u-you" {roles} onrole={noop} />
    </div>
  </div>

  <div class="panel">
    <SpectatorPanel
      {seats}
      {names}
      {bots}
      {follow}
      onfollow={(s) => (follow = s)}
      {people}
      you="u-you"
      former={{ seat: 3, by: 'bot', name: '' }}
    />
  </div>
</section>

{#if menu}
  <AdminSeatMenu
    seat={menu === 'human' ? 2 : 3}
    name={menu === 'human' ? names[2]! : names[3]!}
    crestTip={names[menu === 'human' ? 2 : 3]!}
    occupant={menu === 'human'
      ? { userId: 'u-bo', name: 'Bo', role: 'admin', connected: false }
      : null}
    levels={botLevels()}
    spectators={people.filter((p) => p.connected)}
    roles={menu === 'human' ? { promote: false, demote: true } : { promote: false, demote: false }}
    onaction={noop}
    onclose={() => (menu = null)}
  />
{/if}

<style>
  .gallery,
  .stack {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  h3 {
    margin: 0;
    font: 600 1.25rem var(--type-display);
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }
  .open {
    min-height: 44px;
    padding-inline: 0.75rem;
    border: 1px solid var(--rule);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
  }
  .grid {
    display: grid;
    gap: 1rem;
  }
  .box {
    min-width: 0;
    padding: 0.75rem;
    border-radius: var(--shape-panel);
    background: var(--panel);
  }
  .panel {
    max-width: 32rem;
  }
  @media (min-width: 48rem) {
    .grid {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
