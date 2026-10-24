<script lang="ts">
  // One adjacent pair's conflict at the end of an Age: both seats, the
  // token each took (the winner's victory with its points, the loser's
  // defeat, or "=" for a tie) and both shield counts, all from the server's
  // `military` event.
  import { ageLabel, formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import ConflictToken from './ConflictToken.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import { resultFor, type Conflict } from './end';

  interface Side {
    seat: number;
    name: string;
    /** Shields at the conflict. */
    shields: number;
  }

  interface Props {
    conflict: Conflict;
    age: number;
    a: Side;
    b: Side;
    you?: number | null;
  }

  let { conflict, age, a, b, you = null }: Props = $props();

  const numeral = $derived(ageLabel(age));
  function tokenTip(side: Side): string {
    const r = resultFor(conflict, side.seat);
    return r === 'victory'
      ? ht('hg.end.token.victory', { name: side.name, age: numeral, count: conflict.worth })
      : r === 'defeat'
        ? ht('hg.end.token.defeat', { name: side.name })
        : ht('hg.end.token.tie', { a: a.name, b: b.name });
  }
  const shieldsTip = $derived(
    ht('hg.end.pair.shields', {
      a: a.name,
      b: b.name,
      aCount: a.shields,
      bCount: b.shields,
      age: numeral
    })
  );
</script>

<li
  class="pair"
  data-conflict={`${conflict.seat}:${conflict.other}`}
  data-outcome={conflict.outcome}
  data-mine={you !== null && (a.seat === you || b.seat === you) ? '' : undefined}
>
  {#each [a, b] as side, i (side.seat)}
    {@const r = resultFor(conflict, side.seat) ?? 'tie'}
    <span class="who" class:end={i === 1}>
      <SeatCrest seat={side.seat} tip={side.name} />
      <span class="name" dir="auto">{side.name}</span>
    </span>
    <span class="tok" class:end={i === 1}>
      {#if r !== 'tie'}
        <ConflictToken
          kind={r}
          {age}
          tip={tokenTip(side)}
          value={r === 'victory' ? conflict.worth : undefined}
        />
      {/if}
    </span>
    {#if i === 0}
      <span
        class="mid"
        role="img"
        aria-label={shieldsTip}
        data-shields={`${a.shields}:${b.shields}`}
        {@attach tooltip({ text: shieldsTip })}
      >
        <span dir="ltr">{formatNumber(a.shields)} : {formatNumber(b.shields)}</span>
      </span>
    {/if}
  {/each}
</li>

<style>
  .pair {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 2.2rem auto 2.2rem minmax(0, 1fr);
    align-items: center;
    gap: 0.3rem;
    min-height: 2.3rem;
    padding-inline: 0.4rem;
    border-radius: var(--shape-control);
  }
  .pair[data-mine] {
    background: var(--panel-sunken);
  }
  /* Every part names its column, so all of them share the one row. */
  .pair > * {
    grid-row: 1;
  }
  .who {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    min-width: 0;
  }
  .who.end {
    flex-direction: row-reverse;
    grid-column: 5;
  }
  .tok {
    display: flex;
    justify-content: center;
    grid-column: 2;
  }
  .tok.end {
    grid-column: 4;
  }
  .who:not(.end) {
    grid-column: 1;
  }
  /* Two lines before the cut, so a numbered name keeps its number. */
  .name {
    display: -webkit-box;
    min-width: 0;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    font-size: 0.875rem;
    line-height: 1.2;
    overflow-wrap: break-word;
  }
  .who.end .name {
    text-align: end;
  }
  .mid {
    grid-column: 3;
    min-width: 2.6rem;
    color: var(--fg-muted);
    font-size: 0.875rem;
    font-variant-numeric: tabular-nums;
    font-weight: 700;
    text-align: center;
  }
</style>
