<script lang="ts">
  // Haftgane's end-of-Age and end-of-game pieces (M18.age-military-and-scores),
  // each standalone with plain props: the conflict tokens, a conflict line
  // with and without shield counts, the score table, the final board's
  // actions and a card chip, and buttons that open the three dialogs. The
  // data is made up here for 3 to 7 seats; the tie-break sentence uses the
  // shape requested from the server (`result.tieBreak`), which the live
  // view does not carry yet. With `?hgEnd=military|scores|tie|shared|reveal`
  // (and `seats=3..7`) the page shows that one dialog open, for captures.
  // The headings are dev labels, not shipped copy.
  import { onMount } from 'svelte';
  import '$lib/games/haftgane/theme.css';
  import type { Reveal, Result, Seat } from '$lib/wire/haftgane.gen';
  import { hgBoardSlugs } from '$lib/wire/haftgane-catalog.gen';
  import AgeMilitary from '$lib/games/haftgane/table/AgeMilitary.svelte';
  import CardChip from '$lib/games/haftgane/table/CardChip.svelte';
  import ConflictRow from '$lib/games/haftgane/table/ConflictRow.svelte';
  import ConflictToken from '$lib/games/haftgane/table/ConflictToken.svelte';
  import EndActions from '$lib/games/haftgane/table/EndActions.svelte';
  import FinalReveal from '$lib/games/haftgane/table/FinalReveal.svelte';
  import ScoreSheet from '$lib/games/haftgane/table/ScoreSheet.svelte';
  import ScoreTable from '$lib/games/haftgane/table/ScoreTable.svelte';
  import { scoreRows, type Conflict, type TieBreak } from '$lib/games/haftgane/table/end';

  interface Props {
    /** Show only this dialog, open (the focused capture mode). */
    focus?: string | null;
    seatCount?: number;
  }

  let { focus = null, seatCount = 7 }: Props = $props();

  const noop = () => {};
  const NAMES = ['Roshan', 'Lena', 'Apprentice', 'Darius', 'Master', 'Scholar', 'Omid'];

  function seatsOf(n: number): Seat[] {
    return Array.from({ length: n }, (_, i) => ({
      seat: i,
      board: hgBoardSlugs[i % hgBoardSlugs.length]!,
      side: i % 2 === 0 ? 'day' : 'night',
      stages: i % 3,
      built: [],
      coins: [7, 13, 4, 9, 13, 2, 11][i]!,
      victories: [1, 0, 0],
      defeats: i % 2,
      shields: [3, 1, 4, 1, 5, 2, 0][i]!,
      handSize: 0,
      hasKept: false,
      relative: i,
      relation: i === 0 ? 'self' : i === 1 ? 'left' : i === n - 1 ? 'right' : 'other',
      score: {
        wonder: [3, 7, 0, 10, 5, 3, 0][i]!,
        treasure: [2, 4, 1, 3, 4, 0, 3][i]!,
        military: [4, -2, 9, -3, 12, 1, -1][i]!,
        blue: [12, 8, 3, 15, 6, 9, 4][i]!,
        yellow: [2, 5, 0, 4, 3, 1, 6][i]!,
        green: [13, 21, 10, 2, 9, 0, 18][i]!,
        purple: [5, 0, 7, 3, 4, 2, 0][i]!,
        total: [41, 43, 30, 34, 43, 16, 30][i]!
      },
      science: { counts: [0, 0, 0], choices: [], points: 0 }
    }));
  }

  const seats = $derived(seatsOf(seatCount));
  const names = $derived(Object.fromEntries(seats.map((s) => [s.seat, NAMES[s.seat]!])));
  const conflicts = $derived<Conflict[]>(
    seats.map((s) => {
      const other = (s.seat + 1) % seats.length;
      const a = s.shields;
      const b = seats[other]!.shields;
      return {
        seat: s.seat,
        other,
        outcome: a > b ? 'seat-wins' : a < b ? 'other-wins' : 'tie',
        seatShields: a,
        otherShields: b,
        worth: a === b ? 0 : 3
      };
    })
  );
  // Ranks as the server would state them for these totals and coins.
  const result = $derived.by<Result>(() => {
    const scores = seats.map((s) => s.score);
    const rank = seats.map(
      (s) =>
        1 +
        seats.filter(
          (o) =>
            o.score.total > s.score.total || (o.score.total === s.score.total && o.coins > s.coins)
        ).length
    );
    return {
      scores,
      rank,
      winners: rank.flatMap((r, i) => (r === 1 ? [i] : [])),
      tieBreak: '',
      tied: []
    };
  });
  // Lena (13 coins) and Master (13 coins) both have 43: a shared victory
  // at five seats or more; at three or four, Lena alone has 43.
  const tieShared = $derived<TieBreak | undefined>(
    seats.length >= 5 ? { kind: 'shared', seats: [1, 4] } : undefined
  );
  // A coin tie-break: the same table with Master's coins lowered.
  const coinSeats = $derived(seats.map((s) => (s.seat === 4 ? { ...s, coins: 8 } : s)));
  const coinResult = $derived.by<Result>(() => {
    if (seats.length < 5) return result;
    const rank = [...result.rank];
    rank[4] = 2;
    return { ...result, rank, winners: [1] };
  });
  const tieCoins = $derived<TieBreak | undefined>(
    seats.length >= 5 ? { kind: 'coins', seats: [1, 4] } : undefined
  );
  const reveal = $derived<Reveal>({
    tucked: seats.map((s) =>
      s.stages > 0 ? ['lumber-yard', 'baths', 'library'].slice(0, s.stages) : []
    ),
    leftovers: [1, 2, 3].flatMap((age) =>
      seats.map((s) => ({
        age,
        seat: s.seat,
        design: ['ore-vein', 'dispensary', 'senate'][age - 1]!,
        kept: age === 3 && s.seat === 2
      }))
    ),
    pile: ['clay-pit', 'clay-pit', 'baths', 'altar', 'altar', 'altar', 'forum', 'arena'],
    box: ['shipowners-guild', 'traders-guild']
  });

  let shown = $state<string | null>(null);
  onMount(() => {
    if (focus) shown = focus;
  });
</script>

{#if focus}
  <div class="focus" data-gallery-end={focus} data-seats={seatCount}></div>
{:else}
  <section class="flex flex-col gap-4" data-gallery-section="haftgane-end">
    <h2 class="font-display text-2xl text-fg">Haftgane: end of Age and game <!-- i18n-ok --></h2>
    <div class="row">
      <ConflictToken kind="victory" age={2} tip="Victory token of Age II" />
      <ConflictToken kind="defeat" age={2} tip="Defeat token" />
      <ConflictToken kind="tie" age={2} tip="No token" />
      <ConflictToken kind="victory" age={3} tip="Victory, with the value sent" value={5} />
      <ConflictToken kind="defeat" age={3} tip="Defeat, with the value sent" value={-1} />
    </div>
    <ul class="pairs">
      <ConflictRow
        conflict={{
          seat: 0,
          other: 1,
          outcome: 'seat-wins',
          seatShields: 3,
          otherShields: 1,
          worth: 3
        }}
        age={2}
        you={0}
        a={{ seat: 0, name: 'Roshan', shields: 3 }}
        b={{ seat: 1, name: 'Lena', shields: 1 }}
      />
      <ConflictRow
        conflict={{ seat: 5, other: 6, outcome: 'tie', seatShields: 2, otherShields: 2, worth: 0 }}
        age={2}
        a={{ seat: 5, name: 'Scholar', shields: 2 }}
        b={{ seat: 6, name: 'A very long player name indeed', shields: 2 }}
      />
      <ConflictRow
        conflict={{
          seat: 3,
          other: 4,
          outcome: 'other-wins',
          seatShields: 0,
          otherShields: 4,
          worth: 1
        }}
        age={1}
        a={{ seat: 3, name: 'Darius', shields: 0 }}
        b={{ seat: 4, name: 'Master', shields: 4 }}
      />
    </ul>
    <div class="table-box">
      <ScoreTable
        rows={scoreRows(result)}
        {names}
        you={0}
        coins={Object.fromEntries(seats.map((s) => [s.seat, s.coins]))}
        onseat={noop}
      />
    </div>
    <EndActions onscores={noop} onconflicts={noop} onreveal={noop} />
    <div class="row">
      <CardChip slug="baths" />
      <CardChip slug="altar" count={3} />
      <CardChip slug="shipowners-guild" />
    </div>
    <div class="row">
      {#each ['military', 'scores', 'tie', 'shared', 'reveal'] as kind (kind)}
        <button type="button" class="open" onclick={() => (shown = kind)}>{kind}</button>
      {/each}
    </div>
  </section>
{/if}

{#if shown === 'military'}
  <AgeMilitary
    age={2}
    ages={[1, 2]}
    {conflicts}
    {seats}
    {names}
    you={0}
    latest
    onage={noop}
    onclose={() => (shown = null)}
  />
{:else if shown === 'scores' || shown === 'tie' || shown === 'shared'}
  <ScoreSheet
    result={shown === 'tie' ? coinResult : result}
    seats={shown === 'tie' ? coinSeats : seats}
    {names}
    you={0}
    tieBreak={shown === 'tie' ? tieCoins : shown === 'shared' ? tieShared : undefined}
    canDownload={shown !== 'shared'}
    ondownload={noop}
    onhome={noop}
    onseat={noop}
    onclose={() => (shown = null)}
  />
{:else if shown === 'reveal'}
  <FinalReveal {reveal} {seats} {names} onclose={() => (shown = null)} />
{/if}

<style>
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }
  .pairs {
    display: flex;
    flex-direction: column;
    max-width: 30rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .table-box {
    max-width: 24rem;
  }
  .open {
    min-height: 2.75rem;
    padding-inline: 0.8rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
  }
</style>
