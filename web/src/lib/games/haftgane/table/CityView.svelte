<script lang="ts">
  // One city in the city sheet (phone-layout.md section 8): its crest, name,
  // place at the table, wonder and side; its points, coins, shields,
  // conflict tokens, stages and science; its points so far by category; its
  // last turn; its wonder stages; what it produces; and its buildings by
  // colour. Every number is the view's. Only the viewer's own city shows
  // what only its owner may know (the cards under the wonder, a kept
  // card); another city shows its public view, plus the cards under its
  // wonder once the end reveals them.
  import type { Snippet } from 'svelte';
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import CityRows from './CityRows.svelte';
  import CityStages from './CityStages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import ResourceTile from './ResourceTile.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import StatPill from './StatPill.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import { cardName } from './labels';
  import { makings, stagesOf, type SeatStatus } from './model';

  interface Props {
    seat: Seat;
    name: string;
    /** Where it sits from the viewer, already translated. */
    where: string;
    bot: boolean;
    status: SeatStatus;
    /** The id its name heading takes, for the sheet's label. */
    titleId: string;
    /** The viewer's own city: its private facts follow. */
    own?: boolean;
    /** The viewer's cards under this wonder (own city only). */
    tucked?: readonly string[];
    /** The viewer's kept card (own city only). */
    kept?: string;
    /** The cards the end revealed under this wonder, if it did. */
    revealed?: readonly string[] | null;
    /** Controls at the end of the header, such as the sheet's close button. */
    actions?: Snippet;
  }

  let {
    seat,
    name,
    where,
    bot,
    status,
    titleId,
    own = false,
    tucked = [],
    kept = '',
    revealed = null,
    actions
  }: Props = $props();

  const stages = $derived(stagesOf(seat));
  const made = $derived(makings(seat));
  const resName = (r: string) => ht(`hg.resource.${r}` as 'hg.resource.wood');
  const making = (m: (typeof made)[number]) =>
    m.choice
      ? ht('hg.screen.making.choice', {
          options: m.res.map(resName).reduce((a, b) => ht('hg.screen.trade.or', { a, b }))
        })
      : ht('hg.screen.making.one', { count: m.n, resource: resName(m.res[0] ?? '') });
  const boardName = $derived(ht(`hg.board.${seat.board}` as 'hg.board.gizah'));
  const wonderLine = $derived(
    seat.side === 'day' || seat.side === 'night'
      ? ht('hg.city.wonder', {
          wonder: boardName,
          side: ht(seat.side === 'day' ? 'hg.screen.side.day' : 'hg.screen.side.night')
        })
      : ht('hg.city.wonder.noSide', { wonder: boardName })
  );
  const numeral = (i: number) =>
    ht(`hg.screen.stageNumeral.${Math.min(i + 1, 4)}` as 'hg.screen.stageNumeral.1');
  const SYMBOLS = ['compass', 'gear', 'tablet'] as const;
  const chosen = $derived(seat.science.choices.reduce((a, b) => a + b, 0));
  const scienceTip = $derived(
    ht('hg.city.science.tip', {
      compass: seat.science.counts[0] ?? 0,
      gear: seat.science.counts[1] ?? 0,
      tablet: seat.science.counts[2] ?? 0,
      count: seat.science.points
    })
  );
  const scienceLines = $derived(
    chosen > 0 ? [ht('hg.city.science.chosen', { count: chosen })] : []
  );
  const stagesTip = $derived(ht('hg.screen.stages.tip', { built: seat.stages, of: stages.length }));
  const CATEGORIES = [
    'wonder',
    'treasure',
    'military',
    'blue',
    'yellow',
    'green',
    'purple'
  ] as const;
  const MINUS = '−';
  const signed = (n: number) => (n < 0 ? MINUS + formatNumber(-n) : formatNumber(n));
  const last = $derived.by(() => {
    const p = seat.last;
    if (!p) return '';
    const what =
      p.kind === 'build'
        ? ht('hg.city.last.build', { card: cardName(p.design) })
        : p.kind === 'build-discarded'
          ? ht('hg.city.last.build-discarded', { card: cardName(p.design) })
          : p.kind === 'stage'
            ? ht('hg.city.last.stage', { stage: numeral(p.stage - 1) })
            : p.kind === 'sell'
              ? ht('hg.city.last.sell')
              : '';
    return what ? ht('hg.city.last', { what }) : '';
  });
  const under = $derived(
    own && tucked.length > 0
      ? { cards: tucked, own: true }
      : revealed && revealed.length > 0
        ? { cards: revealed, own: false }
        : null
  );
  const keptLine = $derived(
    own && kept
      ? ht('hg.city.kept.own', { card: cardName(kept) })
      : !own && seat.hasKept
        ? ht('hg.city.kept.other')
        : ''
  );
</script>

<article
  class="city"
  style:--seat="var(--seat-{seat.seat})"
  data-city={seat.seat}
  data-city-own={own ? 'true' : 'false'}
>
  <header class="head">
    <SeatCrest
      seat={seat.seat}
      tip={ht('hg.screen.crest.tip', { name, where })}
      {bot}
      botTip={ht('hg.screen.botBadge')}
      size="md"
    />
    <div class="who">
      <h2 id={titleId} dir="auto">{name}</h2>
      <p class="where">{where} · {wonderLine}</p>
    </div>
    <StatusBadge {status} {name} />
    {@render actions?.()}
  </header>

  <div class="pills">
    <StatPill
      kind="points"
      size="md"
      value={seat.score.total}
      tip={ht('hg.screen.points.tip', { count: seat.score.total })}
    />
    <StatPill
      kind="coins"
      size="md"
      value={seat.coins}
      tip={ht('hg.screen.coins.tip', { count: seat.coins })}
    />
    <StatPill
      kind="shields"
      size="md"
      value={seat.shields}
      tip={ht('hg.screen.shields.tip', { count: seat.shields })}
    />
    {#each seat.victories as count, age (age)}
      {#if count > 0}
        {@const tip = ht('hg.city.victories.tip', { age: numeral(age), count })}
        <span
          class="token victory"
          data-victories={age + 1}
          role="img"
          aria-label={tip}
          {@attach tooltip({ text: tip })}
          ><HgGlyph name="conflict-victory" /><span dir="ltr"
            >{numeral(age)}{count > 1 ? ` ×${formatNumber(count)}` : ''}</span
          ></span
        >
      {/if}
    {/each}
    {#if seat.defeats > 0}
      {@const tip = ht('hg.city.defeats.tip', { count: seat.defeats })}
      <span
        class="token defeat"
        data-defeats={seat.defeats}
        role="img"
        aria-label={tip}
        {@attach tooltip({ text: tip })}
        ><HgGlyph name="conflict-defeat" /><span dir="ltr">×{formatNumber(seat.defeats)}</span
        ></span
      >
    {/if}
    {#if stages.length > 0}
      <span class="chip" role="img" aria-label={stagesTip} {@attach tooltip({ text: stagesTip })}>
        <HgGlyph name="wonder-stage" />
        <span dir="ltr">{ht('hg.screen.stages', { built: seat.stages, of: stages.length })}</span>
      </span>
    {/if}
    <span
      class="chip"
      data-science
      role="img"
      aria-label={[scienceTip, ...scienceLines].join(' ')}
      {@attach tooltip({ text: scienceTip, lines: scienceLines })}
    >
      {#each SYMBOLS as s, i (s)}
        <span class="sym"
          ><HgGlyph name="science-{s}-sm" />{formatNumber(seat.science.counts[i] ?? 0)}</span
        >
      {/each}
    </span>
  </div>

  <section class="part" aria-labelledby="{titleId}-score">
    <h3 id="{titleId}-score">{ht('hg.city.score')}</h3>
    <ul class="score" data-city-score>
      {#each CATEGORIES as c (c)}
        <li data-score={c}>
          <span class="label">{ht(`hg.city.score.${c}`)}</span>
          <span class="value" dir="ltr">{signed(seat.score[c])}</span>
        </li>
      {/each}
    </ul>
  </section>

  {#if last || keptLine}
    <div class="notes">
      {#if last}<p data-city-last>{last}</p>{/if}
      {#if keptLine}<p data-city-kept>{keptLine}</p>{/if}
    </div>
  {/if}

  {#if stages.length > 0}
    <section class="part" aria-labelledby="{titleId}-stages">
      <h3 id="{titleId}-stages">{ht('hg.city.stages')}</h3>
      <CityStages {stages} built={seat.stages} {under} />
    </section>
  {/if}

  <section class="part" aria-labelledby="{titleId}-production">
    <h3 id="{titleId}-production">
      {ht('hg.city.production')}
      <span class="note"
        >{own ? ht('hg.city.production.own') : ht('hg.city.production.other', { name })}</span
      >
    </h3>
    <div class="makings" data-city-production>
      {#each made as m, i (i)}
        {@const tip = making(m)}
        <span class="chip" role="img" aria-label={tip} {@attach tooltip({ text: tip })}>
          {#if m.choice}
            {#each m.res as r, j (r)}{#if j > 0}<span class="slash">/</span>{/if}<ResourceTile
                res={r}
                size="xs"
              />{/each}
          {:else}
            <ResourceTile res={m.res[0] ?? ''} />
            {#if m.n > 1}<span dir="ltr">×{formatNumber(m.n)}</span>{/if}
          {/if}
        </span>
      {/each}
    </div>
  </section>

  <section class="part" aria-labelledby="{titleId}-city">
    <h3 id="{titleId}-city">{ht('hg.city.buildings', { count: seat.built.length })}</h3>
    <CityRows built={seat.built} />
  </section>
</article>

<style>
  .city {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    min-width: 0;
    color: var(--fg);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    min-width: 0;
    font-size: 1.4rem;
  }
  .who {
    flex: 1;
    min-width: 0;
  }
  h2 {
    margin: 0;
    overflow: hidden;
    font: 600 1.6rem/1.15 var(--type-display);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .where {
    margin: 0.1rem 0 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
    line-height: 1.3;
  }
  .pills,
  .makings {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
  }
  .token,
  .chip {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    gap: 0.2em;
    height: 1.625rem;
    padding-inline: 0.5rem;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--panel);
    font-size: 0.9375rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .defeat {
    border-color: var(--cinnabar);
  }
  .sym {
    display: inline-flex;
    align-items: center;
    gap: 0.1em;
  }
  .slash {
    color: var(--fg-muted);
    font-weight: 400;
  }
  .part {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    min-width: 0;
  }
  h3 {
    margin: 0;
    color: var(--fg-muted);
    font: 600 0.95rem var(--type-display);
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .note {
    font-family: inherit;
    font-weight: 400;
    letter-spacing: 0;
    text-transform: none;
  }
  .score {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .score li {
    display: inline-flex;
    align-items: baseline;
    gap: 0.35rem;
    padding: 0.2rem 0.55rem;
    border-radius: 0.45rem;
    background: var(--panel-sunken);
    font-size: 0.875rem;
  }
  .value {
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .notes p {
    margin: 0 0 0.25rem;
    color: var(--fg-muted);
    font-size: 0.9rem;
    line-height: 1.4;
  }
</style>
