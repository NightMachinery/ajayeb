<script lang="ts">
  // The city sheet (phone-layout.md section 8): any seat's city, opened
  // from its crest, far chip or neighbour panel, with ring tabs that move
  // between cities without closing the sheet. The viewer's own city adds
  // what only the viewer may know; the end reveal, when the table allows
  // it, adds the cards under every wonder. For an admin, Manage opens the
  // shown seat's admin menu.
  import { i18n } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import CityView from './CityView.svelte';
  import RingTabs from './RingTabs.svelte';
  import { ringOrder } from './city';
  import type { SeatStatus } from './model';
  import { where } from './seat-names';

  interface Props {
    seats: readonly Seat[];
    names: Readonly<Record<number, string>>;
    /** The seats a bot plays. */
    bots: readonly number[];
    statuses: Readonly<Record<number, SeatStatus>>;
    /** The viewer's private facts, when the viewer holds a seat. */
    own?: { seat: number; tucked: readonly string[]; kept: string } | null;
    /** The cards under each seat's wonder, once the end reveals them. */
    revealed?: readonly (readonly string[])[] | null;
    active: number;
    onselect: (seat: number) => void;
    onclose: () => void;
    /** Opens a seat's admin menu, when the viewer is an admin. */
    onmanage?: (seat: number) => void;
  }

  let {
    seats,
    names,
    bots,
    statuses,
    own = null,
    revealed = null,
    active,
    onselect,
    onclose,
    onmanage
  }: Props = $props();

  let open = $state(true);
  const order = $derived(ringOrder(seats));
  const n = $derived(seats.length);
  const nameOf = (s: Seat) => names[s.seat] ?? ht('hg.screen.seat', { seat: s.seat + 1 });
  const tabs = $derived(
    order.map((s) => ({
      seat: s.seat,
      name: s.relation === 'self' ? ht('hg.screen.you') : nameOf(s),
      tip: ht('hg.screen.crest.tip', { name: nameOf(s), where: where(s, n) }),
      bot: bots.includes(s.seat)
    }))
  );
  const seat = $derived(seats.find((s) => s.seat === active) ?? order[0]!);
  const mine = $derived(own !== null && own.seat === seat.seat);
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-city-title"
  class="hg-city-sheet"
  panelData={{ 'data-city-sheet': String(seat.seat) }}
>
  <div class="content" dir={i18n.dir}>
    <RingTabs
      {tabs}
      active={seat.seat}
      label={ht('hg.city.tabs')}
      botTip={ht('hg.screen.botBadge')}
      panelId="hg-city-panel"
      {onselect}
    />
    <div id="hg-city-panel" role="tabpanel" aria-labelledby="hg-city-title">
      <CityView
        {seat}
        name={nameOf(seat)}
        where={where(seat, n)}
        bot={bots.includes(seat.seat)}
        status={statuses[seat.seat] ?? 'idle'}
        titleId="hg-city-title"
        own={mine}
        tucked={mine ? own!.tucked : []}
        kept={mine ? own!.kept : ''}
        revealed={revealed?.[seat.seat] ?? null}
      >
        {#snippet actions()}
          {#if onmanage}
            {@const tip = ht('hg.admin.seat.open', { name: nameOf(seat) })}
            <button
              type="button"
              class="manage"
              data-act="seat-menu:{seat.seat}"
              aria-label={tip}
              onclick={() => onmanage(seat.seat)}
              {@attach tooltip({ text: tip, actionable: true })}
              >{ht('hg.admin.seats.manage')}</button
            >
          {/if}
          <button
            type="button"
            class="close"
            data-act="city-close"
            aria-label={ht('hg.city.close')}
            onclick={onclose}
            {@attach tooltip({ text: ht('hg.city.close'), actionable: true })}>×</button
          >
        {/snippet}
      </CityView>
    </div>
  </div>
</Sheet>

<style>
  :global(dialog.sheet.hg-city-sheet) {
    width: min(100%, 46rem);
    --sheet-max-height: calc(100dvh - 0.5rem);
  }
  .content {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    padding: 0.4rem 0.75rem 1.2rem;
  }
  .manage {
    flex: 0 0 auto;
    min-height: 44px;
    padding-inline: 0.75rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
  }
  .close {
    display: grid;
    flex: 0 0 44px;
    place-items: center;
    width: 44px;
    height: 44px;
    border: 0;
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 1.5rem;
    cursor: pointer;
  }
</style>
