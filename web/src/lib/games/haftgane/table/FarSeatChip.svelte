<script lang="ts">
  // A seat beyond the viewer's neighbours: crest, name, points, coins and
  // shields, with its status badge on the top corner. The desktop layout
  // adds the colours it holds. With `onopen` the whole chip is a button
  // that opens the seat's city.
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import ColourCounts from './ColourCounts.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import StatPill from './StatPill.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import type { SeatStatus } from './model';

  interface Props {
    seat: Seat;
    name: string;
    /** Where it sits from the viewer, already translated. */
    where: string;
    bot: boolean;
    status: SeatStatus;
    /** Opens this seat's city. */
    onopen?: () => void;
  }

  let { seat, name, where, bot, status, onopen }: Props = $props();
</script>

<article
  class="far"
  data-seat={seat.seat}
  data-confirmed={status === 'confirmed' ? 'true' : 'false'}
  data-relation={seat.relation}
  aria-label={ht('hg.screen.crest.tip', { name, where })}
>
  {#if onopen}
    <button
      type="button"
      class="body open"
      data-act="city:{seat.seat}"
      aria-label={ht('hg.city.open', { name })}
      onclick={onopen}>{@render body()}</button
    >
  {:else}
    <span class="body">{@render body()}</span>
  {/if}
  <span class="status"><StatusBadge {status} {name} /></span>
</article>

{#snippet body()}
  <span class="head">
    <SeatCrest
      seat={seat.seat}
      tip={ht('hg.screen.crest.tip', { name, where })}
      {bot}
      botTip={ht('hg.screen.botBadge')}
    />
    <span class="name" dir="auto">{name}</span>
  </span>
  <span class="stats">
    <StatPill
      kind="points"
      size="xs"
      value={seat.score.total}
      tip={ht('hg.screen.points.tip', { count: seat.score.total })}
    />
    <StatPill
      kind="coins"
      size="xs"
      value={seat.coins}
      tip={ht('hg.screen.coins.tip', { count: seat.coins })}
    />
    <StatPill
      kind="shields"
      size="xs"
      value={seat.shields}
      tip={ht('hg.screen.shields.tip', { count: seat.shields })}
    />
  </span>
  <span class="colours"><ColourCounts built={seat.built} hideZero /></span>
{/snippet}

<style>
  .far {
    position: relative;
    display: flex;
    flex: 1 1 0;
    min-width: 0;
    border: 1px solid var(--rule);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow);
  }
  .body {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
    padding: 0.25rem 0.35rem;
  }
  .open {
    min-height: 44px;
    border: 0;
    border-radius: inherit;
    background: none;
    color: inherit;
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .open:hover {
    background: var(--panel-sunken);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    min-width: 0;
  }
  .name {
    overflow: hidden;
    font-size: 0.8125rem;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .stats {
    display: flex;
    flex-wrap: wrap;
    gap: 0.15rem 0.35rem;
  }
  .colours {
    display: none;
  }
  .status {
    position: absolute;
    inset-block-start: -0.6rem;
    inset-inline-end: -0.3rem;
  }
  @media (min-width: 64rem) and (min-height: 32.01rem) {
    .colours {
      display: block;
    }
  }
</style>
