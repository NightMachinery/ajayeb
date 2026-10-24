<script lang="ts">
  // The end-of-Age military dialog (docs/haftgane/phone-layout.md, section
  // 6): first the viewer's two conflicts side by side, then every conflict
  // around the table, one line per adjacent pair, then each seat's military
  // points so far. Every outcome is a `military` event the server sent; the
  // shield counts show only while they are the ones those conflicts
  // compared (the newest end of Age, nothing resolved since), and the points
  // are each seat's score from the view. Closing it is local: the next
  // Age's first decision is already open behind it.
  import { ageLabel, formatNumber } from '$lib/i18n';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import ConflictRow from './ConflictRow.svelte';
  import ConflictToken from './ConflictToken.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import { conflictBetween, resultFor, type Conflict } from './end';

  interface Props {
    age: number;
    /** Every Age whose conflicts are known so far, oldest first. */
    ages: readonly number[];
    conflicts: readonly Conflict[];
    seats: readonly Seat[];
    names: Readonly<Record<number, string>>;
    /** The viewer's seat, or null for a spectator. */
    you: number | null;
    /** The seats' shields are still the ones these conflicts compared. */
    shieldsKnown: boolean;
    /** These are the newest conflicts, so each seat's military score is its
     *  total right after them. */
    latest: boolean;
    onage: (age: number) => void;
    onclose: () => void;
  }

  let { age, ages, conflicts, seats, names, you, shieldsKnown, latest, onage, onclose }: Props =
    $props();

  let open = $state(true);
  const MINUS = '−';
  const signed = (n: number) =>
    n > 0 ? `+${formatNumber(n)}` : n < 0 ? MINUS + formatNumber(-n) : formatNumber(0);
  const numeral = $derived(ageLabel(age));
  const seatOf = (n: number) => seats.find((s) => s.seat === n);
  const shieldsOf = (n: number) => (shieldsKnown ? seatOf(n)?.shields : undefined);
  const nameOf = (n: number) => names[n] ?? '';
  const mine = $derived(you === null ? undefined : seatOf(you));
  const neighbours = $derived(
    you === null
      ? []
      : (['left', 'right'] as const).flatMap((side) => {
          const s = seats.find((x) => x.relation === side);
          const c = s ? conflictBetween(conflicts, you, s.seat) : undefined;
          return s && c ? [{ side, seat: s, conflict: c, result: resultFor(c, you)! }] : [];
        })
  );
  function resultTip(result: 'victory' | 'defeat' | 'tie', them: string): string {
    return result === 'victory'
      ? ht('hg.end.token.youWin', { name: them, age: numeral })
      : result === 'defeat'
        ? ht('hg.end.token.youLose', { name: them })
        : ht('hg.end.token.youTie', { name: them });
  }
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-military-title"
  class="hg-military"
  panelData={{ 'data-age-military': String(age) }}
>
  {#snippet footer()}
    <div class="foot">
      {#if latest && mine}
        <p class="mine" data-my-military={mine.score.military}>
          {ht('hg.end.military.yourPoints', { points: signed(mine.score.military) })}
        </p>
      {/if}
      <Button variant="primary" onclick={() => (open = false)} data-act="military-close">
        {ht('hg.end.military.close')}
      </Button>
    </div>
  {/snippet}
  <div class="body">
    <h2 id="hg-military-title" tabindex="-1">{ht('hg.end.military.title', { age: numeral })}</h2>
    {#if ages.length > 1}
      <div class="ages" role="group" aria-label={ht('hg.end.military.ages')}>
        {#each ages as a (a)}
          <button
            type="button"
            class="age"
            aria-pressed={a === age}
            data-act={`military-age:${a}`}
            onclick={() => onage(a)}
          >
            {ht('hg.screen.age', { age: ageLabel(a) })}
          </button>
        {/each}
      </div>
    {/if}

    {#if neighbours.length > 0}
      <section class="yours-part" aria-labelledby="hg-military-yours">
        <h3 id="hg-military-yours">{ht('hg.end.military.yours')}</h3>
        <div class="yours">
          {#each neighbours as nb (nb.side)}
            {@const them = nameOf(nb.seat.seat)}
            <div class="duel {nb.result}" data-my-conflict={nb.side} data-result={nb.result}>
              <div class="against">
                <SeatCrest seat={nb.seat.seat} tip={them} />
                <span class="label">
                  <span class="dir"
                    >{nb.side === 'left'
                      ? ht('hg.end.military.left')
                      : ht('hg.end.military.right')}</span
                  >
                  <span class="name" dir="auto">{them}</span>
                </span>
              </div>
              <div class="outcome">
                <ConflictToken kind={nb.result} {age} tip={resultTip(nb.result, them)} />
                <span class="word">{ht(`hg.end.result.${nb.result}`)}</span>
              </div>
              {#if shieldsKnown && mine}
                <p class="shields" data-shields={`${mine.shields}:${nb.seat.shields}`}>
                  {ht('hg.end.military.shields', { you: mine.shields, them: nb.seat.shields })}
                </p>
              {/if}
            </div>
          {/each}
        </div>
      </section>
    {/if}

    <section class="all-part" aria-labelledby="hg-military-all">
      <h3 id="hg-military-all">{ht('hg.end.military.all')}</h3>
      <ul class="pairs">
        {#each conflicts as c (c.seat)}
          <ConflictRow
            conflict={c}
            {age}
            {you}
            a={{ seat: c.seat, name: nameOf(c.seat), shields: shieldsOf(c.seat) }}
            b={{ seat: c.other, name: nameOf(c.other), shields: shieldsOf(c.other) }}
          />
        {/each}
      </ul>
    </section>

    {#if latest}
      <section class="points-part" aria-labelledby="hg-military-points">
        <h3 id="hg-military-points">{ht('hg.end.military.points')}</h3>
        <ul class="totals">
          {#each seats as s (s.seat)}
            {@const tip = ht('hg.end.military.pointsTip', {
              name: nameOf(s.seat),
              points: signed(s.score.military)
            })}
            <li
              class="total"
              class:me={s.seat === you}
              data-military-total={s.seat}
              data-points={s.score.military}
            >
              <SeatCrest seat={s.seat} tip={nameOf(s.seat)} />
              <span
                class="value"
                class:negative={s.score.military < 0}
                role="img"
                aria-label={tip}
                dir="ltr"
                {@attach tooltip({ text: tip })}>{signed(s.score.military)}</span
              >
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </div>
</Sheet>

<style>
  :global(dialog.sheet.hg-military) {
    width: min(100%, 36rem);
  }
  .body {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    padding: 0.4rem 1rem 0.6rem;
    color: var(--fg);
  }
  h2 {
    margin: 0;
    font: 600 1.4rem/1.2 var(--type-display);
    outline: none;
  }
  h3 {
    margin: 0 0 0.35rem;
    color: var(--fg-muted);
    font: 600 0.95rem var(--type-display);
  }
  .ages {
    display: flex;
    gap: 0.4rem;
  }
  .age {
    min-width: 2.75rem;
    min-height: 2.75rem;
    padding-inline: 0.8rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }
  .age[aria-pressed='true'] {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
  }
  .yours {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 0.5rem;
  }
  .duel {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.5rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
  }
  .duel.victory {
    border-color: var(--afford-ok-edge);
  }
  .duel.defeat {
    border-color: var(--cinnabar);
  }
  .duel.tie {
    border-style: dashed;
  }
  .against {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    min-width: 0;
  }
  .label {
    display: flex;
    flex-direction: column;
    min-width: 0;
    line-height: 1.2;
  }
  .dir {
    color: var(--fg-muted);
    font-size: 0.75rem;
  }
  .name {
    overflow: hidden;
    font-size: 0.875rem;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .outcome {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }
  .word {
    font: 600 1.05rem var(--type-display);
  }
  .shields {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.8rem;
    font-variant-numeric: tabular-nums;
  }
  .pairs,
  .totals {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .totals {
    flex-direction: row;
    flex-wrap: wrap;
    gap: 0.35rem;
  }
  .total {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding-block: 0.15rem;
    padding-inline: 0.2rem 0.45rem;
    border: 1.5px solid var(--rule);
    border-radius: 9999px;
    background: var(--panel);
  }
  .total.me {
    border-color: var(--accent);
  }
  .value {
    min-width: 1.6rem;
    font-size: 0.875rem;
    font-variant-numeric: tabular-nums;
    font-weight: 700;
  }
  .negative {
    color: var(--cinnabar-ink);
  }
  .foot {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .mine {
    margin: 0;
    font-weight: 600;
    text-align: center;
  }
  /* A short landscape screen: your conflicts and the points at the start,
     every conflict at the end, and a footer of one line. */
  @media (orientation: landscape) and (max-height: 32rem) {
    :global(dialog.sheet.hg-military) {
      --sheet-max-height: calc(100dvh - 1rem);
      width: min(100%, 52rem);
    }
    .body {
      display: grid;
      grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr);
      grid-auto-flow: row dense;
      align-content: start;
      gap: 0.6rem 1rem;
      padding-block-start: 0.75rem;
    }
    .body > * {
      grid-column: 1;
    }
    .body > .all-part {
      grid-column: 2;
      grid-row: 1 / span 4;
    }
    .foot {
      flex-direction: row;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
    }
    .mine {
      text-align: start;
    }
    .foot :global(button) {
      min-width: 12rem;
    }
  }
</style>
