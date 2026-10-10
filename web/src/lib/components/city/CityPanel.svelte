<script lang="ts">
  // The tablet/desktop column for one seat (UX design sec. 3): name, crest,
  // coins and score, city cards as band stacks grouped by colour, wonders
  // as a readable list, science and owned tokens. The same data as
  // PlayerStrip, laid out as a column instead of a wrapping row.
  import { t } from '$lib/i18n';
  import { cardBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryDanger, scienceDanger } from '$lib/view/threats';
  import type { Color, MilitaryView, PlayerView, ScoreView, WonderView } from '$lib/wire';
  import ProgressToken from '../token/ProgressToken.svelte';
  import CityBand from './CityBand.svelte';
  import ResourceChips from './ResourceChips.svelte';
  import ScienceTracker from './ScienceTracker.svelte';
  import WonderRow from './WonderRow.svelte';

  interface Props {
    name: string;
    seat: 0 | 1;
    player: PlayerView;
    military: MilitaryView;
    score?: ScoreView;
    /** When given, the wonders still to build are buttons (the wonder-first flow). */
    onwonder?: (wonder: WonderView) => void;
    class?: string;
  }

  let { name, seat, player, military, score, onwonder, class: className = '' }: Props = $props();

  const COLORS = [
    'brown',
    'grey',
    'blue',
    'green',
    'yellow',
    'red',
    'purple'
  ] as const satisfies readonly Color[];

  const cityCounts = $derived.by(() => {
    const counts: Record<Color, number> = {
      brown: 0,
      grey: 0,
      blue: 0,
      green: 0,
      yellow: 0,
      red: 0,
      purple: 0
    };
    for (const slug of player.city) {
      const color = cardBySlug(slug)?.color;
      if (color) counts[color] += 1;
    }
    return counts;
  });

  const crestName = $derived(seat === 0 ? ('crest-sun' as const) : ('crest-moon' as const));
  const crestTextClass = $derived(seat === 0 ? 'text-sun-ink' : 'text-moon-ink');
  const militaryThreat = $derived(militaryDanger(military, seat));
  const scienceThreat = $derived(scienceDanger(player.science));
  const danger = $derived(militaryThreat || scienceThreat);
</script>

<div
  class="city-panel flex w-full flex-col gap-2.5 rounded-panel border p-3
    {danger ? 'border-cinnabar ring-2 ring-cinnabar/25' : 'border-rule'} {className}"
>
  <div class="flex items-center justify-between gap-2">
    <div class="flex min-w-0 items-center gap-1.5">
      <Icon name={crestName} class="shrink-0 text-xl {crestTextClass}" />
      <span class="min-w-0 truncate font-display text-base text-fg" dir="auto">{name}</span>
    </div>
    <div class="flex items-center gap-2 text-sm font-semibold tabular-nums">
      <span class="flex items-center gap-0.5" title={t('coins.count', { count: player.coins })}>
        <Icon name="coin" class="text-gild" />
        <span data-must-fit>{player.coins}</span>
      </span>
      {#if score}
        <span
          class="flex items-center gap-0.5 text-fg-muted"
          title={t('board.player.scorePreview', { total: score.total })}
        >
          <Icon name="vp-laurel" />
          <span data-must-fit>{score.total}</span>
        </span>
      {/if}
    </div>
  </div>

  {#if militaryThreat || scienceThreat}
    <div class="flex flex-wrap gap-1">
      {#if militaryThreat}
        <span
          class="rounded-control bg-cinnabar/15 px-1.5 py-0.5 text-[11px] font-semibold text-cinnabar"
        >
          {t('board.player.militaryDanger')}
        </span>
      {/if}
      {#if scienceThreat}
        <span
          class="rounded-control bg-cinnabar/15 px-1.5 py-0.5 text-[11px] font-semibold text-cinnabar"
        >
          {t('board.player.scienceDanger')}
        </span>
      {/if}
    </div>
  {/if}

  <ResourceChips production={player.production} tradePrices={player.tradePrices} />
  <ScienceTracker science={player.science} />

  <div class="flex flex-col gap-1">
    <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
      {t('board.citySheet.city')}
    </h3>
    <div class="flex flex-wrap gap-1.5" aria-label={t('board.citySheet.city')}>
      {#each COLORS as color (color)}
        <CityBand {color} count={cityCounts[color]} />
      {/each}
    </div>
  </div>

  <div class="flex flex-col gap-1">
    <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
      {t('board.citySheet.wonders')}
    </h3>
    <WonderRow wonders={player.wonders} variant="list" onselect={onwonder} />
  </div>

  {#if player.tokens.length > 0}
    <div class="flex flex-col gap-1">
      <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
        {t('board.citySheet.tokens')}
      </h3>
      <ul class="flex flex-wrap gap-1" aria-label={t('board.citySheet.tokens')}>
        {#each player.tokens as slug (slug)}
          <li><ProgressToken {slug} size="sm" taken /></li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
