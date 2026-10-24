<script lang="ts">
  // Haftgane's city sheet and Seen hands leaves (M18.city-sheet-and-notebook),
  // each standalone with plain props. The cities and notebooks are the
  // engine's own view fixtures (server/internal/haftgane/rules/testdata/views:
  // discard-seat mid-game, over-reveal-spectator at the end, turn-seat at
  // the start), copied into haftgane-city-views.json. Those games hold no
  // neighbour picks or returning cards, so the summary's sample statistics
  // are made up here. The headings are dev labels, not shipped copy.
  import type { Notebook, Seat, Statistics } from '$lib/wire/haftgane.gen';
  import { ht } from '$lib/games/haftgane/i18n/messages.svelte';
  import CitySheet from '$lib/games/haftgane/table/CitySheet.svelte';
  import CityView from '$lib/games/haftgane/table/CityView.svelte';
  import MiniCard from '$lib/games/haftgane/table/MiniCard.svelte';
  import RingTabs from '$lib/games/haftgane/table/RingTabs.svelte';
  import SeenHandsButton from '$lib/games/haftgane/table/SeenHandsButton.svelte';
  import SeenHandsSheet from '$lib/games/haftgane/table/SeenHandsSheet.svelte';
  import SeenHandsSummary from '$lib/games/haftgane/table/SeenHandsSummary.svelte';
  import { ringOrder, type SeenFilter } from '$lib/games/haftgane/table/city';
  import type { SeatStatus } from '$lib/games/haftgane/table/model';
  import { where } from '$lib/games/haftgane/table/seat-names';
  import views from './haftgane-city-views.json';

  interface Views {
    mid: {
      seats: Seat[];
      age: number;
      you: { seat: number; tucked: string[]; kept: string; notebook: Notebook };
    };
    end: { seats: Seat[]; tucked: string[][] };
    start: { age: number; notebook: Notebook };
  }
  const v: Views = views;
  const names = Object.fromEntries(
    Array.from({ length: 7 }, (_, i) => [i, ht('hg.screen.seat', { seat: i + 1 })])
  );
  const statuses: Record<number, SeatStatus> = { 0: 'confirmed', 1: 'bot', 3: 'choosing' };
  const tabs = ringOrder(v.mid.seats).map((s) => ({
    seat: s.seat,
    name: s.relation === 'self' ? ht('hg.screen.you') : names[s.seat]!,
    tip: ht('hg.screen.crest.tip', { name: names[s.seat]!, where: where(s, 7) }),
    bot: s.seat % 2 === 1
  }));
  const own = v.mid.seats.find((s) => s.relation === 'self')!;
  const left = v.mid.seats.find((s) => s.relation === 'left')!;
  const sample: Statistics = {
    circulating: [
      { colour: 'brown', bounds: { certain: 1, atMost: 2 } },
      { colour: 'red', bounds: { certain: 0, atMost: 1 } },
      { colour: 'green', bounds: { certain: 2, atMost: 2 } }
    ],
    returning: [{ design: 'apothecary', bounds: { certain: 0, atMost: 1 } }],
    neighbourPicks: [{ point: 'age-1-turn-2', seat: left.seat, design: 'stockade' }]
  };

  let tab = $state(own.seat);
  let filter = $state<SeenFilter | null>(null);
  let city = $state<number | null>(null);
  let seen = $state<'mid' | 'start' | null>(null);
</script>

<section class="gallery" data-gallery-city>
  <h3>City sheet and Seen hands <!-- i18n-ok --></h3>
  <div class="row">
    <button type="button" class="open" onclick={() => (city = own.seat)}
      >Open the city sheet <!-- i18n-ok --></button
    >
    <button type="button" class="open" onclick={() => (seen = 'mid')}
      >Open Seen hands, Age II <!-- i18n-ok --></button
    >
    <button type="button" class="open" onclick={() => (seen = 'start')}
      >Open Seen hands, empty <!-- i18n-ok --></button
    >
    <SeenHandsButton onopen={() => (seen = 'mid')} />
  </div>

  <div class="box tabs">
    <RingTabs
      {tabs}
      active={tab}
      label={ht('hg.city.tabs')}
      botTip={ht('hg.screen.botBadge')}
      panelId="gallery-city-panel"
      onselect={(s) => (tab = s)}
    />
  </div>

  <div class="grid">
    <div class="box">
      <CityView
        seat={own}
        name={names[own.seat]!}
        where={where(own, 7)}
        bot={false}
        status="confirmed"
        titleId="gallery-city-own"
        own
        tucked={v.mid.you.tucked}
        kept="dispensary"
      />
    </div>
    <div class="box">
      <CityView
        seat={left}
        name={names[left.seat]!}
        where={where(left, 7)}
        bot
        status="bot"
        titleId="gallery-city-left"
      />
    </div>
    <div class="box">
      <CityView
        seat={v.end.seats[5]!}
        name={names[5]!}
        where={where(v.end.seats[5]!, 7)}
        bot={false}
        status="idle"
        titleId="gallery-city-end"
        revealed={v.end.tucked[5]}
      />
    </div>
    <div class="box stack">
      <SeenHandsSummary statistics={sample} {names} {filter} onfilter={(f) => (filter = f)} />
      <div class="row">
        <MiniCard slug="stockade" mark={ht('hg.seen.played.build')} />
        <MiniCard slug="clay-pool" count={2} />
        <MiniCard slug="scriptorium" dim />
      </div>
    </div>
  </div>
</section>

{#if city !== null}
  <CitySheet
    seats={v.mid.seats}
    {names}
    bots={[1, 3, 5]}
    {statuses}
    own={{ seat: own.seat, tucked: v.mid.you.tucked, kept: v.mid.you.kept }}
    active={city}
    onselect={(s) => (city = s)}
    onclose={() => (city = null)}
  />
{/if}
{#if seen}
  <SeenHandsSheet
    notebook={seen === 'mid' ? { ...v.mid.you.notebook, statistics: sample } : v.start.notebook}
    age={seen === 'mid' ? v.mid.age : v.start.age}
    {names}
    onclose={() => (seen = null)}
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
  .tabs {
    max-width: 46rem;
  }
  @media (min-width: 48rem) {
    .grid {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
