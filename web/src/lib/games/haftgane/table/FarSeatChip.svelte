<script lang="ts">
  // A seat beyond the viewer's neighbours: crest, name, points, coins and
  // shields, with its status badge on the top corner. The desktop layout
  // adds the colours it holds.
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
  }

  let { seat, name, where, bot, status }: Props = $props();
</script>

<article
  class="far"
  data-seat={seat.seat}
  data-confirmed={status === 'confirmed' ? 'true' : 'false'}
  data-relation={seat.relation}
  aria-label={ht('hg.screen.crest.tip', { name, where })}
>
  <div class="head">
    <SeatCrest
      seat={seat.seat}
      tip={ht('hg.screen.crest.tip', { name, where })}
      {bot}
      botTip={ht('hg.screen.botBadge')}
    />
    <span class="name" dir="auto">{name}</span>
  </div>
  <div class="stats">
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
  </div>
  <div class="colours"><ColourCounts built={seat.built} hideZero /></div>
  <span class="status"><StatusBadge {status} {name} /></span>
</article>

<style>
  .far {
    position: relative;
    display: flex;
    flex: 1 1 0;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
    padding: 0.25rem 0.35rem;
    border: 1px solid var(--rule);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow);
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
