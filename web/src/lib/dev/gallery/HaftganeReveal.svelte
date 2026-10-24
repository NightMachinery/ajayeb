<script lang="ts">
  // Haftgane's reveal and turn recap leaves (M18.reveal-and-recap), each
  // standalone with plain props: the reveal tags of every kind, the coin
  // changes, the recap of a seven-seat end of Age II seen from seat 2 (the
  // engine's discard-seat view fixture, with a seat switch, a promotion and
  // a pass added), the same recap as a spectator reads it, the top bar's
  // recap button with its unseen dot, and the recap overlay. The people are
  // made up here; the headings are dev labels, not shipped copy.
  import type { PlatformLogEntry, TableEvent } from '$lib/wire/platform.gen';
  import CoinDelta from '$lib/games/haftgane/table/CoinDelta.svelte';
  import RecapLayer from '$lib/games/haftgane/table/RecapLayer.svelte';
  import RevealTag from '$lib/games/haftgane/table/RevealTag.svelte';
  import TableTopBar from '$lib/games/haftgane/table/TableTopBar.svelte';
  import TurnRecap from '$lib/games/haftgane/table/TurnRecap.svelte';
  import { buildRecap, type CoinChange, type RecapLine } from '$lib/games/haftgane/table/recap';
  import { coinText } from '$lib/games/haftgane/table/recap-text';

  const blank: TableEvent = {
    kind: '',
    seat: 0,
    other: 0,
    age: 0,
    turn: 0,
    play: '',
    design: '',
    stage: 0,
    side: '',
    left: 0,
    right: 0,
    printed: 0,
    delta: 0,
    source: '',
    outcome: '',
    pass: ''
  };
  const ev = (e: Partial<TableEvent>): TableEvent => ({ ...blank, ...e });
  // The discard-seat fixture's batch, as seat 2 received it, plus a pass.
  const own: TableEvent[] = [
    ev({ kind: 'leftover-discarded', seat: 2, design: 'vineyard' }),
    ev({ kind: 'card-kept', seat: 4 }),
    ev({ kind: 'play', seat: 0, play: 'sell' }),
    ev({ kind: 'play', seat: 1, play: 'build', design: 'dispensary', right: 4 }),
    ev({ kind: 'play', seat: 2, play: 'stage', design: 'statue', stage: 1, right: 2 }),
    ev({ kind: 'play', seat: 3, play: 'sell' }),
    ev({ kind: 'play', seat: 4, play: 'build', design: 'aqueduct', printed: 1 }),
    ev({ kind: 'play', seat: 5, play: 'sell' }),
    ev({ kind: 'play', seat: 6, play: 'build', design: 'vineyard' }),
    ev({ kind: 'coins', seat: 0, delta: 3, source: 'sale' }),
    ev({ kind: 'coins', seat: 3, delta: 3, source: 'sale' }),
    ev({ kind: 'coins', seat: 5, delta: 3, source: 'sale' }),
    ev({ kind: 'coins', seat: 6, delta: 5, source: 'design', design: 'vineyard' }),
    ev({ kind: 'hands-passed', age: 2, turn: 4, pass: 'right' })
  ];
  // A spectator's copy of the same batch: the stage's card and the
  // discarded last card unnamed.
  const watched = own.map((e) =>
    (e.kind === 'play' && e.play !== 'build') || e.kind === 'leftover-discarded'
      ? { ...e, design: '' }
      : e
  );
  const log: PlatformLogEntry[] = [
    { kind: 'seat', at: 8, by: 'u-ana', seat: 5, control: 'human', userId: 'u-fay' },
    { kind: 'role', at: 8, by: 'u-ana', userId: 'u-fay', role: 'admin' }
  ];
  const names: Record<number, string> = {
    0: 'Ana',
    1: 'Apprentice',
    2: 'Cyrus',
    3: 'Scholar',
    4: 'Darya',
    5: 'Fay',
    6: 'Master'
  };
  const people: Record<string, string> = { 'u-ana': 'Ana', 'u-fay': 'Fay' };
  const userName = (id: string | undefined) => (id && people[id]) || 'A player who left';
  const bots = [1, 3, 6];
  const mine = buildRecap({ batch: 9, events: own }, 7, 2, 8, log);
  const theirs = buildRecap({ batch: 9, events: watched }, 7, 4, 8, log);
  const nameOf = (seat: number) => names[seat] ?? '';

  const tags: RecapLine[] = [
    {
      seat: 1,
      act: { kind: 'build', design: 'dispensary' },
      coins: [],
      kept: null,
      leftover: null
    },
    {
      seat: 2,
      act: { kind: 'stage', stage: 2, design: '' },
      coins: [],
      kept: null,
      leftover: null
    },
    {
      seat: 3,
      act: { kind: 'sell', design: '' },
      coins: [{ kind: 'sale', amount: 3 }],
      kept: null,
      leftover: null
    },
    {
      seat: 4,
      act: { kind: 'build-discarded', design: 'sawmill' },
      coins: [],
      kept: null,
      leftover: null
    },
    { seat: 5, act: { kind: 'discard-nothing' }, coins: [], kept: null, leftover: null },
    { seat: 6, act: { kind: 'side', side: 'night' }, coins: [], kept: null, leftover: null },
    {
      seat: 0,
      act: { kind: 'build', design: 'chamber-of-commerce' },
      coins: [],
      kept: null,
      leftover: null
    }
  ];
  const coins: CoinChange[] = [
    { kind: 'bank', amount: 1 },
    { kind: 'paid', seat: 3, amount: 2 },
    { kind: 'received', seat: 1, amount: 4 },
    { kind: 'sale', amount: 3 },
    { kind: 'card', design: 'vineyard', amount: 5 },
    { kind: 'stage', stage: 2, amount: 3 }
  ];
  const noop = () => {};
  let open = $state(false);
</script>

<section class="gallery" data-gallery-reveal>
  <h3>Reveal tags, coin changes and the turn recap <!-- i18n-ok --></h3>
  <div class="box row">
    {#each tags as line (line.seat)}
      <RevealTag {line} tip={nameOf(line.seat)} />
    {/each}
  </div>
  <div class="box row">
    {#each coins as change, i (i)}
      <CoinDelta {change} tip={coinText(change, nameOf)} />
    {/each}
  </div>
  <div class="box">
    <TableTopBar
      phase="turn"
      age={2}
      turn={4}
      turns={6}
      pass="right"
      passTo="Apprentice"
      finished={false}
      onmenu={noop}
      onrecap={() => (open = true)}
      recapUnseen
      spectators={2}
      onspectators={noop}
    />
  </div>
  <div class="grid">
    <div class="box">
      <TurnRecap
        recap={mine}
        {names}
        {bots}
        you={2}
        {userName}
        held={{ name: 'Scholar', count: 4 }}
      />
    </div>
    <div class="box">
      <TurnRecap recap={theirs} {names} {bots} you={null} {userName} level={3} />
    </div>
  </div>
  <button type="button" class="open" onclick={() => (open = true)}
    >Open the recap overlay <!-- i18n-ok --></button
  >
</section>

{#if open}
  <RecapLayer labelledby="gallery-recap-title" onclose={() => (open = false)}>
    <TurnRecap
      recap={mine}
      {names}
      {bots}
      you={2}
      {userName}
      held={{ name: 'Scholar', count: 4 }}
      titleId="gallery-recap-title"
    />
  </RecapLayer>
{/if}

<style>
  .gallery {
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
  .open {
    align-self: flex-start;
    min-height: 44px;
    padding-inline: 0.75rem;
    border: 1px solid var(--rule);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
  }
  @media (min-width: 48rem) {
    .grid {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
