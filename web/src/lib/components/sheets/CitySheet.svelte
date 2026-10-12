<script lang="ts">
  // The phone city sheet (UX design sec. 5 "City sheet"): either player's
  // full city in the Sheet primitive, switched by two tabs. Reuses
  // CityPanel for the tab's content, so the sheet and the tablet/desktop
  // column never disagree about what a city shows.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { BotLevel, MilitaryView, Pair, PlayerView, ScoreView } from '$lib/wire';
  import Sheet from '../shell/Sheet.svelte';
  import CityPanel from '../city/CityPanel.svelte';

  interface Props {
    open?: boolean;
    names: readonly [string, string];
    players: Pair<PlayerView>;
    military: MilitaryView;
    scores?: Pair<ScoreView>;
    /** Each seat's bot level, when a bot plays it (its avatar shows). */
    bots?: readonly [BotLevel | undefined, BotLevel | undefined];
    /** Which seat's tab is open; bindable so the caller can open the sheet
     *  straight to one player's city (tapping that player's own strip). */
    activeSeat?: 0 | 1;
    titleId?: string;
  }

  let {
    open = $bindable(false),
    names,
    players,
    military,
    scores,
    bots,
    activeSeat = $bindable(0),
    titleId = 'city-sheet-title'
  }: Props = $props();

  const seats = [0, 1] as const;
</script>

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-5">
    <h2 id={titleId} class="sr-only">{t('board.citySheet.title')}</h2>
    <div class="flex gap-1.5" role="tablist" aria-label={t('board.citySheet.title')}>
      {#each seats as seat (seat)}
        <button
          type="button"
          role="tab"
          aria-selected={activeSeat === seat}
          onclick={() => (activeSeat = seat)}
          class="flex min-h-11 flex-1 items-center justify-center gap-1.5 rounded-control border px-2 text-sm
            font-semibold {activeSeat === seat
            ? 'border-accent bg-accent-soft text-accent-soft-fg'
            : 'border-rule text-fg-muted'}"
        >
          <Icon name={seat === 0 ? 'crest-sun' : 'crest-moon'} />
          <span class="truncate" dir="auto">{names[seat]}</span>
        </button>
      {/each}
    </div>

    <CityPanel
      name={names[activeSeat]}
      seat={activeSeat}
      player={players[activeSeat]}
      {military}
      score={scores?.[activeSeat]}
      bot={bots?.[activeSeat]}
      class="border-0 p-0"
    />
  </div>
</Sheet>
