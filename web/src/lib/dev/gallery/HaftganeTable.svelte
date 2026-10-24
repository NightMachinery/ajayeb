<script lang="ts">
  // Haftgane's table screen leaves (M18.table-screen), each standalone with
  // plain props, for a visual check in both themes: the crests and pills,
  // the status badges, the trade chips and military outlooks, a far chip,
  // both neighbour panels, the strip, hand cards with every badge, the
  // pick list, the pass cues and the bars. The data is made up here and
  // carries only the fields the leaves read. The headings are dev labels,
  // not shipped copy.
  import type { Choice, Neighbour, Seat, TradeUnit } from '$lib/wire/haftgane.gen';
  import { hgBoardSlugs } from '$lib/wire/haftgane-catalog.gen';
  import FarSeatChip from '$lib/games/haftgane/table/FarSeatChip.svelte';
  import HandCard from '$lib/games/haftgane/table/HandCard.svelte';
  import HandGrid from '$lib/games/haftgane/table/HandGrid.svelte';
  import MilitaryOutlook from '$lib/games/haftgane/table/MilitaryOutlook.svelte';
  import NeighbourPanel from '$lib/games/haftgane/table/NeighbourPanel.svelte';
  import PassChip from '$lib/games/haftgane/table/PassChip.svelte';
  import PassFlow from '$lib/games/haftgane/table/PassFlow.svelte';
  import PickPanel from '$lib/games/haftgane/table/PickPanel.svelte';
  import SeatCrest from '$lib/games/haftgane/table/SeatCrest.svelte';
  import StatPill from '$lib/games/haftgane/table/StatPill.svelte';
  import StatusBadge from '$lib/games/haftgane/table/StatusBadge.svelte';
  import TableTopBar from '$lib/games/haftgane/table/TableTopBar.svelte';
  import TradeChip from '$lib/games/haftgane/table/TradeChip.svelte';
  import TurnBar from '$lib/games/haftgane/table/TurnBar.svelte';
  import YourStrip from '$lib/games/haftgane/table/YourStrip.svelte';
  import type { Afford, SeatStatus } from '$lib/games/haftgane/table/model';
  import '$lib/games/haftgane/theme.css';

  const noop = () => {};

  function seat(n: number, relation: string, built: string[]): Seat {
    return {
      seat: n,
      board: hgBoardSlugs[n % hgBoardSlugs.length]!,
      side: n % 2 === 0 ? 'day' : 'night',
      stages: 1,
      built,
      coins: 3 + n,
      victories: [1, 0, 0],
      defeats: 0,
      shields: n,
      handSize: 5,
      hasKept: false,
      relative: n,
      relation,
      score: {
        wonder: 3,
        treasure: 1,
        military: 1,
        blue: 3,
        yellow: 0,
        green: 0,
        purple: 0,
        total: 8 + n
      },
      science: { counts: [1, 0, 1], choices: [], points: 2 }
    };
  }

  const you = seat(0, 'self', ['lumber-yard', 'clay-pit', 'baths', 'apothecary', 'marketplace']);
  const left = seat(1, 'left', ['stone-pit', 'loom-age-1', 'barracks', 'east-trading-post']);
  const right = seat(6, 'right', ['ore-vein', 'press-age-1', 'scriptorium', 'altar', 'stockade']);
  const far = seat(3, 'other', ['tree-farm', 'theater']);

  const units: TradeUnit[] = [
    { choice: false, producer: 1, count: 2, options: [{ resource: 'stone', price: 2 }] },
    {
      choice: false,
      producer: 1,
      count: 1,
      options: [{ resource: 'cloth', price: 1, discount: { design: 'marketplace', stage: 0 } }]
    },
    {
      choice: true,
      producer: 1,
      count: 1,
      options: [
        { resource: 'wood', price: 2 },
        { resource: 'clay', price: 2 }
      ]
    }
  ];
  const leftTrade: Neighbour = { seat: 1, direction: 'left', units, outlook: -1 };
  const rightTrade: Neighbour = {
    seat: 6,
    direction: 'right',
    units: [units[0]!],
    outlook: 3
  };

  function choice(key: string, over: Partial<Choice>): Choice {
    return {
      key,
      kind: 'build',
      design: '',
      side: '',
      stage: 0,
      left: 0,
      right: 0,
      printed: 0,
      total: 0,
      gain: 0,
      free: '',
      witness: [],
      ...over
    };
  }

  const hand = ['baths', 'library', 'aqueduct', 'guard-tower', 'marketplace', 'caravansery'];
  const choices: Choice[] = [
    choice('build:baths:2:0', { design: 'baths', left: 2, total: 2 }),
    choice('stage:baths:1:0:0', { kind: 'stage', design: 'baths', stage: 2, total: 0 }),
    choice('sell:baths', { kind: 'sell', design: 'baths', gain: 3 }),
    choice('build:library:0:0', { design: 'library', free: 'chain' }),
    choice('build:guard-tower:0:0', { design: 'guard-tower' }),
    choice('build:caravansery:0:0', { design: 'caravansery', printed: 1, total: 1 })
  ];
  const disabled = [
    {
      kind: 'build',
      design: 'aqueduct',
      stage: 0,
      reason: 'insufficient-coins',
      cheapest: 5,
      shortBy: 2
    },
    {
      kind: 'build',
      design: 'marketplace',
      stage: 0,
      reason: 'already-built',
      cheapest: 0,
      shortBy: 0
    }
  ];
  const affords: { slug: string; afford: Afford }[] = [
    { slug: 'library', afford: { kind: 'free', via: 'chain' } },
    { slug: 'baths', afford: { kind: 'paid', total: 2, trade: true } },
    { slug: 'caravansery', afford: { kind: 'paid', total: 1, trade: false } },
    { slug: 'guard-tower', afford: { kind: 'none' } },
    { slug: 'marketplace', afford: { kind: 'built' } },
    { slug: 'aqueduct', afford: { kind: 'blocked', reason: 'insufficient-coins', shortBy: 2 } }
  ];
  const STATUSES: SeatStatus[] = ['confirmed', 'bot', 'choosing', 'away', 'idle'];
</script>

<div class="flex flex-col gap-6" data-testid="gallery-haftgane-table">
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Haftgane table: crests, pills, badges <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-center gap-3">
      {#each [0, 1, 2, 3, 4, 5, 6] as n (n)}
        <SeatCrest seat={n} tip="Seat" bot={n === 3} botTip="Bot" size="md" />
      {/each}
      <StatPill kind="points" value={12} tip="12" />
      <StatPill kind="coins" value={5} tip="5" />
      <StatPill kind="shields" value={2} tip="2" />
      <StatPill kind="conflict" value={-1} tip="-1" />
      {#each STATUSES as s (s)}
        <span
          class="relative inline-block h-8 w-16 rounded-control border border-dashed border-rule"
        >
          <StatusBadge status={s} name="Lena" />
        </span>
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Haftgane table: trade and military <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-center gap-3">
      {#each units as unit, i (i)}
        <TradeChip {unit} name="Omid" />
      {/each}
      {#each [3, -1, 0, 5] as outlook (outlook)}
        <MilitaryOutlook {outlook} theirs={2} yours={3} name="Omid" />
      {/each}
      <PassChip pass="left" to="Omid" />
      <PassChip pass="right" to="Lena" />
      <PassFlow from={{ seat: 6, name: 'Lena' }} to={{ seat: 1, name: 'Omid' }} handSize={6} />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Haftgane table: seats <!-- i18n-ok --></h3>
    <div class="grid max-w-[24rem] grid-cols-2 gap-2 pt-3">
      <FarSeatChip seat={far} name="Darius" where="3 seats away" bot={false} status="choosing" />
      <FarSeatChip seat={far} name="Scholar" where="3 seats away" bot status="bot" />
    </div>
    <div class="grid max-w-[24rem] grid-cols-2 gap-2" dir="ltr">
      <NeighbourPanel
        seat={left}
        side="left"
        name="Omid"
        where="left"
        bot={false}
        status="confirmed"
        trade={leftTrade}
        yourShields={0}
        pass="left"
      />
      <NeighbourPanel
        seat={right}
        side="right"
        name="Master"
        where="right"
        bot
        status="away"
        trade={rightTrade}
        yourShields={0}
        pass="left"
      />
    </div>
    <div class="max-w-[24rem]">
      <YourStrip seat={you} name="Sara" where="you" own bot={false} status="choosing" stageOpen />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Haftgane table: hand cards <!-- i18n-ok --></h3>
    <div class="flex flex-wrap gap-2">
      {#each affords as a (a.slug)}
        <div class="w-24">
          <HandCard
            slug={a.slug}
            afford={a.afford}
            selected={a.slug === 'baths'}
            picked={a.slug === 'library' ? 'confirmed' : a.slug === 'guard-tower' ? 'draft' : null}
          />
        </div>
      {/each}
    </div>
    <div class="flex h-[19rem] max-w-[24rem] flex-col">
      <HandGrid
        {hand}
        {choices}
        {disabled}
        selected="baths"
        picked={{ design: 'library', confirmed: false }}
        onselect={noop}
      />
    </div>
  </section>

  <section class="flex max-w-[24rem] flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Haftgane table: bars and the pick list <!-- i18n-ok -->
    </h3>
    <TableTopBar
      phase="turn"
      age={2}
      turn={3}
      turns={6}
      pass="right"
      passTo="Lena"
      finished={false}
      onmenu={noop}
    />
    <PickPanel
      card="baths"
      choices={choices.filter((c) => c.design === 'baths')}
      pendingKey="build:baths:2:0"
      locked={false}
      onpick={noop}
      onclose={noop}
    />
    <TurnBar
      text="Confirm your choice"
      chosen={{ done: 3, of: 7 }}
      pending={{ key: 'build:baths:2:0', confirm: false }}
      pendingLabel="Your pick: Baths: Build for 2 coins"
      canAct
      onconfirm={noop}
      onwithdraw={noop}
    />
    <TurnBar
      text="Confirmed. Waiting for the others."
      chosen={{ done: 6, of: 7 }}
      pending={{ key: 'build:baths:2:0', confirm: true }}
      canAct
      onconfirm={noop}
      onwithdraw={noop}
    />
  </section>
</div>
