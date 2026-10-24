<script lang="ts">
  // One adjacent pair's conflict at the end of an Age: both seats, the
  // token each took (the winner's victory, the loser's defeat, or "=" for a
  // tie) and, when the dialog knows them, both shield counts. The outcome
  // is the server's `military` event; the shields are the view's own counts
  // and are passed only while they are the ones the conflict compared.
  import { ageLabel, formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import ConflictToken from './ConflictToken.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import { resultFor, type Conflict } from './end';

  interface Side {
    seat: number;
    name: string;
    /** Shields at the conflict, when known. */
    shields?: number;
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
      ? ht('hg.end.token.victory', { name: side.name, age: numeral })
      : r === 'defeat'
        ? ht('hg.end.token.defeat', { name: side.name })
        : ht('hg.end.token.tie', { a: a.name, b: b.name });
  }
  const known = $derived(a.shields !== undefined && b.shields !== undefined);
  const shieldsTip = $derived(
    known
      ? ht('hg.end.pair.shields', {
          a: a.name,
          b: b.name,
          aCount: a.shields!,
          bCount: b.shields!,
          age: numeral
        })
      : ''
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
        <ConflictToken kind={r} {age} tip={tokenTip(side)} />
      {/if}
    </span>
    {#if i === 0}
      <span
        class="mid"
        role="img"
        aria-label={known ? shieldsTip : tokenTip(a)}
        data-shields={known ? `${a.shields}:${b.shields}` : undefined}
        {@attach tooltip({ text: known ? shieldsTip : tokenTip(a) })}
      >
        {#if known}
          <span dir="ltr">{formatNumber(a.shields!)} : {formatNumber(b.shields!)}</span>
        {:else if conflict.outcome === 'tie'}
          =
        {:else}
          ·
        {/if}
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
  .name {
    min-width: 0;
    overflow: hidden;
    font-size: 0.875rem;
    text-overflow: ellipsis;
    white-space: nowrap;
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
