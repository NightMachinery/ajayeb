<script lang="ts">
  // The anchor seat's strip: crest and name ("You" for the viewer's own),
  // points, coins, shields and the sum of its conflict tokens; one chip for
  // everything the city makes and one for its science symbols; its colour
  // counts (the colours it holds); and its wonder with one pill per stage.
  // For a spectator it is seat 0's city, without the "You" tag.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import ColourCounts from './ColourCounts.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import ResourceTile from './ResourceTile.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import StagePills from './StagePills.svelte';
  import StatPill from './StatPill.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import { makings, stagesOf, type SeatStatus } from './model';

  interface Props {
    seat: Seat;
    name: string;
    where: string;
    own: boolean;
    bot: boolean;
    status: SeatStatus;
    /** The next stage may be built this turn (the view offers a stage). */
    stageOpen?: boolean;
  }

  let { seat, name, where, own, bot, status, stageOpen = false }: Props = $props();

  const stages = $derived(stagesOf(seat));
  const made = $derived(makings(seat));
  const resName = (r: string) => ht(`hg.resource.${r}` as 'hg.resource.wood');
  const makingLines = $derived(
    made.map((m) =>
      m.choice
        ? ht('hg.screen.making.choice', {
            options: m.res.map(resName).reduce((a, b) => ht('hg.screen.trade.or', { a, b }))
          })
        : ht('hg.screen.making.one', { count: m.n, resource: resName(m.res[0] ?? '') })
    )
  );
  const scienceTip = $derived(
    ht('hg.screen.science.tip', {
      compass: seat.science.counts[0] ?? 0,
      gear: seat.science.counts[1] ?? 0,
      tablet: seat.science.counts[2] ?? 0
    })
  );
  const wonderName = $derived(ht(`hg.wonder.${seat.board}` as 'hg.wonder.gizah'));
  const sideName = $derived(
    seat.side === 'day'
      ? ht('hg.screen.side.day')
      : seat.side === 'night'
        ? ht('hg.screen.side.night')
        : ''
  );
  const wonderTip = $derived(
    sideName
      ? ht('hg.screen.wonder.tip', {
          wonder: wonderName,
          side: sideName,
          built: seat.stages,
          of: stages.length
        })
      : ht('hg.screen.wonder.noSide', { wonder: wonderName })
  );
  const SYMBOLS = ['compass', 'gear', 'tablet'] as const;
</script>

<section
  class="strip"
  class:own
  style:--seat="var(--seat-{seat.seat})"
  data-seat={seat.seat}
  data-confirmed={status === 'confirmed' ? 'true' : 'false'}
  data-relation={seat.relation}
  aria-label={ht('hg.screen.crest.tip', { name, where })}
>
  <div class="row head">
    <SeatCrest
      seat={seat.seat}
      tip={ht('hg.screen.crest.tip', { name, where })}
      {bot}
      botTip={ht('hg.screen.botBadge')}
      size="md"
    />
    <span class="name" dir="auto">{name}</span>
    {#if own}<span class="you">{ht('hg.screen.you')}</span>{/if}
    <StatusBadge {status} {name} />
    <span class="pills">
      <StatPill
        kind="points"
        value={seat.score.total}
        tip={ht('hg.screen.points.tip', { count: seat.score.total })}
      />
      <StatPill
        kind="coins"
        value={seat.coins}
        tip={ht('hg.screen.coins.tip', { count: seat.coins })}
      />
      <StatPill
        kind="shields"
        value={seat.shields}
        tip={ht('hg.screen.shields.tip', { count: seat.shields })}
      />
      <StatPill
        kind="conflict"
        value={seat.score.military}
        tip={ht('hg.screen.conflict.tip', { sum: seat.score.military })}
      />
    </span>
  </div>
  <div class="row">
    <span
      class="chip"
      data-making
      role="img"
      aria-label={[ht('hg.screen.making.tip'), ...makingLines].join(' ')}
      {@attach tooltip({ text: ht('hg.screen.making.tip'), lines: makingLines })}
    >
      {#each made as m, i (i)}
        {#if m.choice}
          <span class="choice">
            {#each m.res as r (r)}<ResourceTile res={r} size="xs" />{/each}
          </span>
        {:else}
          <ResourceTile res={m.res[0] ?? ''} />
          {#if m.n > 1}<span class="n">{formatNumber(m.n)}</span>{/if}
        {/if}
      {/each}
    </span>
    <span
      class="chip"
      data-science
      role="img"
      aria-label={scienceTip}
      {@attach tooltip({ text: scienceTip })}
    >
      {#each SYMBOLS as s, i (s)}
        <span class="sym"
          ><HgGlyph name="science-{s}-sm" />{formatNumber(seat.science.counts[i] ?? 0)}</span
        >
      {/each}
    </span>
    <span class="colours"><ColourCounts built={seat.built} hideZero /></span>
  </div>
  <div class="row">
    <span class="wonder" role="img" aria-label={wonderTip} {@attach tooltip({ text: wonderTip })}>
      <HgGlyph name="wonder" />
      <span class="wonder-text">
        <span class="wonder-name">{ht(`hg.board.${seat.board}` as 'hg.board.gizah')}</span>
        {#if sideName}
          <span class="wonder-side"
            >{sideName} · {ht('hg.screen.stages', { built: seat.stages, of: stages.length })}</span
          >
        {/if}
      </span>
    </span>
    <StagePills {stages} built={seat.stages} open={own && stageOpen} />
  </div>
</section>

<style>
  /* The rows group the markup; on screen their pieces flow together and
     wrap, so the strip takes as few lines as its width allows. */
  .strip {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.3rem 0.4rem;
    min-width: 0;
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--rule);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow);
  }
  .own {
    border: 2px solid var(--seat);
  }
  .row {
    display: contents;
  }
  .name {
    min-width: 0;
    overflow: hidden;
    font-size: 1.05rem;
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .you {
    padding: 0.1rem 0.4rem;
    border-radius: 9999px;
    background: var(--lapis);
    color: var(--paper-bright);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
  }
  .pills {
    display: inline-flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    margin-inline-start: auto;
  }
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    min-height: 1.6rem;
    padding: 0.1rem 0.35rem;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    font-size: 0.8125rem;
    font-weight: 700;
  }
  .choice {
    display: inline-flex;
    gap: 0.05rem;
    padding: 0.05rem;
    border: 1px dashed var(--rule-strong);
    border-radius: 0.25rem;
  }
  .sym {
    display: inline-flex;
    align-items: center;
    gap: 0.1rem;
  }
  .wonder {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    min-height: 2.25rem;
    padding: 0.2rem 0.5rem;
    border: 1px solid var(--rule-strong);
    border-radius: 0.6rem;
    background: var(--panel-sunken);
  }
  .wonder > :global(svg) {
    font-size: 1.3rem;
    color: var(--gild);
  }
  .wonder-text {
    display: flex;
    flex-direction: column;
    line-height: 1.1;
  }
  .wonder-name {
    font-family: var(--type-display);
    font-weight: 600;
  }
  .wonder-side {
    color: var(--fg-muted);
    font-size: 0.75rem;
  }
  /* A short landscape screen leaves the colour counts to the city view. */
  @media (orientation: landscape) and (max-height: 32rem) {
    .strip {
      padding: 0.3rem 0.4rem;
    }
    .colours {
      display: none;
    }
  }
</style>
