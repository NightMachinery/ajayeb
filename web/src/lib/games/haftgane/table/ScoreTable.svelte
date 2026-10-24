<script lang="ts">
  // The final score table: one row per seat in the server's rank order, one
  // column per scoring category headed by its glyph (wonder, coins,
  // military, civic, commerce, science, guilds, total). Every number is
  // the server's; the viewer's row is highlighted and a negative value is
  // red with a minus sign, so colour is never the only cue. Seven rows and
  // eight number columns fit at 360 px: the name truncates and the numbers
  // use tabular figures.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import { SCORE_COLUMNS, type ScoreRow } from './end';

  interface Props {
    rows: readonly ScoreRow[];
    names: Readonly<Record<number, string>>;
    you: number | null;
    /** Each seat's final coins, the tie-break's measure, for the total's tooltip. */
    coins: Readonly<Record<number, number>>;
    /** Opens a seat's city; a row's name becomes its button. */
    onseat?: (seat: number) => void;
  }

  let { rows, names, you, coins, onseat }: Props = $props();

  const MINUS = '−';
  const shown = (n: number) => (n < 0 ? MINUS + formatNumber(-n) : formatNumber(n));
</script>

<table class="scores" data-score-table>
  <thead>
    <tr>
      <th scope="col" class="who">{ht('hg.end.score.player')}</th>
      {#each SCORE_COLUMNS as col (col.key)}
        {@const label = ht(`hg.end.score.col.${col.key}`)}
        <th scope="col" class="num" class:total={col.key === 'total'} data-col={col.key}>
          <span class="head" role="img" aria-label={label} {@attach tooltip({ text: label })}>
            <HgGlyph name={col.glyph} />
          </span>
        </th>
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each rows as row (row.seat)}
      {@const name = names[row.seat] ?? ''}
      {@const totalTip = ht('hg.end.score.totalTip', {
        name,
        total: row.score.total,
        coins: coins[row.seat] ?? 0
      })}
      <tr
        class:me={row.seat === you}
        class:winner={row.winner}
        data-score-row={row.seat}
        data-rank={row.rank}
        data-total={row.score.total}
      >
        <th scope="row" class="who">
          <span class="cell">
            <span
              class="rank"
              role="img"
              aria-label={ht('hg.end.score.rank', { rank: row.rank })}
              {@attach tooltip({ text: ht('hg.end.score.rank', { rank: row.rank }) })}
              >{formatNumber(row.rank)}</span
            >
            {#if onseat}
              <button
                type="button"
                class="open"
                data-act={`score-city:${row.seat}`}
                onclick={() => onseat(row.seat)}
                {@attach tooltip({ text: ht('hg.end.score.city', { name }) })}
              >
                <SeatCrest seat={row.seat} tip={name} />
                <span class="name" dir="auto">{name}</span>
              </button>
            {:else}
              <SeatCrest seat={row.seat} tip={name} />
              <span class="name" dir="auto">{name}</span>
            {/if}
          </span>
        </th>
        {#each SCORE_COLUMNS as col (col.key)}
          {@const value = row.score[col.key]}
          <td
            class="num"
            class:total={col.key === 'total'}
            class:negative={value < 0}
            data-col={col.key}
            data-value={value}
          >
            {#if col.key === 'total'}
              <span {@attach tooltip({ text: totalTip })}>{shown(value)}</span>
            {:else}
              {shown(value)}
            {/if}
          </td>
        {/each}
      </tr>
    {/each}
  </tbody>
</table>

<style>
  .scores {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
    font-size: 0.8125rem;
    font-variant-numeric: tabular-nums;
  }
  th,
  td {
    padding-block: 0.3rem;
    padding-inline: 0.1rem;
  }
  thead th {
    border-block-end: 1.5px solid var(--rule-strong);
    color: var(--fg-muted);
    font-weight: 600;
  }
  .who {
    width: auto;
    text-align: start;
  }
  tbody .who:has(.open) {
    padding-block: 0;
  }
  thead .who {
    font-size: 0.75rem;
  }
  .num {
    width: 1.6rem;
    text-align: center;
  }
  .num.total {
    width: 2.1rem;
    font-weight: 700;
  }
  @media (min-width: 30rem) {
    .num {
      width: 1.85rem;
    }
    .num.total {
      width: 2.3rem;
    }
  }
  .head {
    display: inline-grid;
    place-items: center;
    font-size: 1.05rem;
  }
  tbody tr {
    border-block-end: 1px solid var(--rule);
  }
  tbody tr.me {
    background: var(--panel-sunken);
  }
  tbody tr.me .who {
    border-inline-start: 3px solid var(--accent);
  }
  .cell {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    min-width: 0;
  }
  .rank {
    display: inline-grid;
    flex: 0 0 auto;
    place-items: center;
    min-width: 1.1rem;
    height: 1.1rem;
    border-radius: 9999px;
    background: var(--panel-sunken);
    font-size: 0.6875rem;
    font-weight: 700;
  }
  .winner .rank {
    background: var(--accent);
    color: var(--on-accent);
  }
  .open {
    display: flex;
    flex: 1 1 auto;
    align-items: center;
    gap: 0.25rem;
    min-width: 0;
    min-height: 2.75rem;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .open:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: -2px;
  }
  .name {
    min-width: 0;
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .negative {
    color: var(--cinnabar-ink);
  }
</style>
