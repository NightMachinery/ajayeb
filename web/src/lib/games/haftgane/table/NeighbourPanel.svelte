<script lang="ts">
  // A neighbour's trading summary: crest, name, wonder stages built and the
  // status badge; the military outlook, coins and points; the trade chips
  // (what this neighbour sells the viewer and at what price per unit, from
  // the view); the colour counts of their city; and the pass mark on the
  // inner bottom corner. The desktop layout adds the "sells you" caption,
  // the wonder's stage pills and the city's card names by colour.
  // A spectator's view carries no trading facts, so those parts stay empty.
  // With `onopen` the crest and name form a button that opens the city.
  // An admin's crown follows the name.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Neighbour, Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import ColourCounts from './ColourCounts.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import MilitaryOutlook from './MilitaryOutlook.svelte';
  import PassMark from './PassMark.svelte';
  import RoleTag from './RoleTag.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import StagePills from './StagePills.svelte';
  import StatPill from './StatPill.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import TradeChip from './TradeChip.svelte';
  import { COLOURS, design, stagesOf, type Role, type SeatStatus } from './model';

  interface Props {
    seat: Seat;
    side: 'left' | 'right';
    name: string;
    where: string;
    bot: boolean;
    status: SeatStatus;
    /** What the viewer may buy from them, when the viewer holds a seat. */
    trade?: Neighbour;
    /** The viewer's own shields, for the outlook's tooltip. */
    yourShields?: number;
    pass: string;
    /** The role of the human in the seat. */
    role?: Role | null;
    /** Opens this seat's city. */
    onopen?: () => void;
  }

  let {
    seat,
    side,
    name,
    where,
    bot,
    status,
    trade,
    yourShields = 0,
    pass,
    role = null,
    onopen
  }: Props = $props();

  const stages = $derived(stagesOf(seat));
  const crestTip = $derived(ht('hg.screen.crest.tip', { name, where }));
  const stagesTip = $derived(ht('hg.screen.stages.tip', { built: seat.stages, of: stages.length }));
  // The board's name, and its side once chosen.
  const caption = $derived.by(() => {
    const board = ht(`hg.board.${seat.board}` as 'hg.board.gizah');
    if (seat.side !== 'day' && seat.side !== 'night') return board;
    return board + ht('hg.screen.sep') + ht(`hg.screen.side.${seat.side}`);
  });
  const city = $derived(
    COLOURS.map((colour) => ({
      colour,
      names: seat.built
        .filter((slug) => design(slug)?.colour === colour)
        .map((slug) => ht(`hg.card.${slug}` as 'hg.card.marketplace'))
    })).filter((row) => row.names.length > 0)
  );
</script>

<article
  class="panel {side}"
  data-seat={seat.seat}
  data-confirmed={status === 'confirmed' ? 'true' : 'false'}
  data-relation={seat.relation}
  data-neighbour={side}
  aria-label={crestTip}
>
  <div class="head">
    {#if onopen}
      <button
        type="button"
        class="who open"
        data-act="city:{seat.seat}"
        aria-label={ht('hg.city.open', { name })}
        onclick={onopen}>{@render who()}</button
      >
    {:else}
      <span class="who">{@render who()}</span>
    {/if}
    {#if role}<RoleTag {role} {name} compact />{/if}
    <StatusBadge {status} {name} />
  </div>
  <div class="stats">
    {#if trade}
      <MilitaryOutlook outlook={trade.outlook} theirs={seat.shields} yours={yourShields} {name} />
    {:else}
      <StatPill
        kind="shields"
        value={seat.shields}
        tip={ht('hg.screen.shields.tip', { count: seat.shields })}
      />
    {/if}
    <StatPill
      kind="coins"
      value={seat.coins}
      tip={ht('hg.screen.coins.tip', { count: seat.coins })}
    />
    <StatPill
      kind="points"
      value={seat.score.total}
      tip={ht('hg.screen.points.tip', { count: seat.score.total })}
    />
  </div>
  {#if trade}
    <p class="caption wide">
      <span class="strong">{ht('hg.screen.sellsYou')}</span>
      {ht('hg.screen.perUnit')}
    </p>
  {/if}
  <div class="goods">
    {#if trade}
      <span class="trade" data-trade-chips>
        {#each trade.units as unit, i (i)}
          <TradeChip {unit} {name} />
        {/each}
      </span>
    {/if}
    {#if stages.length > 0}
      <span class="stages" role="img" aria-label={stagesTip} {@attach tooltip({ text: stagesTip })}>
        <HgGlyph name="wonder-stage" />
        <span dir="ltr">{ht('hg.screen.stages', { built: seat.stages, of: stages.length })}</span>
      </span>
    {/if}
    <span class="colours"><ColourCounts built={seat.built} hideZero /></span>
  </div>
  {#if stages.length > 0}
    <div class="wide wonder">
      <span class="caption">
        {caption}
      </span>
      <StagePills {stages} built={seat.stages} />
    </div>
  {/if}
  {#if city.length > 0}
    <ul class="wide city" data-city>
      {#each city as row (row.colour)}
        {@const tip = `${ht(`hg.screen.colourName.${row.colour}`)}: ${ht(`hg.screen.colour.${row.colour}`, { count: row.names.length })}`}
        <li style:--ink="var(--card-{row.colour})">
          <span class="glyph" role="img" aria-label={tip} {@attach tooltip({ text: tip })}>
            <HgGlyph name="category-{row.colour}-sm" />
          </span>
          <span class="names">{row.names.join(ht('hg.city.row.sep'))}</span>
          <span class="n">{formatNumber(row.names.length)}</span>
        </li>
      {/each}
    </ul>
  {/if}
  <PassMark {side} {pass} {name} />
</article>

{#snippet who()}
  <SeatCrest seat={seat.seat} tip={crestTip} {bot} botTip={ht('hg.screen.botBadge')} size="md" />
  <span class="name" dir="auto">{name}</span>
{/snippet}

<style>
  .panel {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.45rem 0.5rem 0.8rem;
    border: 1px solid var(--rule);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    min-width: 0;
  }
  .who {
    display: flex;
    flex: 1 1 auto;
    align-items: center;
    gap: 0.35rem;
    min-width: 0;
  }
  .open {
    min-height: 44px;
    margin-block: -0.35rem;
    margin-inline-start: -0.3rem;
    padding-inline: 0.3rem;
    border: 0;
    border-radius: var(--shape-control);
    background: none;
    color: inherit;
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .open:hover {
    background: var(--panel-sunken);
  }
  .name {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .stages {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    gap: 0.15rem;
    height: 1.5rem;
    padding-inline: 0.35rem;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    font-size: 0.8125rem;
    font-weight: 700;
  }
  .stats,
  .goods,
  .trade {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.25rem;
  }
  .caption {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.75rem;
  }
  .strong {
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .wide {
    display: none;
  }
  .wonder {
    flex-direction: column;
    gap: 0.25rem;
  }
  .city {
    flex-direction: column;
    gap: 0.25rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .city li {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    min-height: 2rem;
    padding: 0.2rem 0.4rem;
    border: 1px solid var(--rule);
    border-radius: 0.5rem;
    font-size: 0.8125rem;
  }
  .city .glyph {
    display: inline-grid;
    place-items: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 0.3rem;
    background: var(--ink);
    color: var(--paper-bright);
  }
  .city .names {
    flex: 1 1 auto;
    min-width: 0;
  }
  .city .n {
    font-weight: 700;
  }
  /* The stacked layout with little height keeps each neighbour's status and leaves what
     they sell and their wonder to the city view, one tap away, and to each
     card's cost (M18.short-screen-status). */
  @media (orientation: portrait) and (max-height: 40rem),
    (min-height: 32.01rem) and (max-height: 40rem) {
    .goods {
      display: none;
    }
  }
  /* Desktop: full side panels (phone-layout.md, "Desktop"). */
  @media (min-width: 64rem) and (min-height: 32.01rem) {
    .wide {
      display: flex;
    }
    p.wide {
      display: block;
    }
  }
</style>
