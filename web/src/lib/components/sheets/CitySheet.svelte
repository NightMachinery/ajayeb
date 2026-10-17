<script lang="ts">
  // The phone city sheet (UX design sec. 5 "City sheet"): either player's
  // full city in the Sheet primitive, switched by two tabs. Reuses
  // CityPanel for the tab's content, so the sheet and the tablet/desktop
  // column never disagree about what a city shows.
  //
  // Under the city, the tab's seat's threats (the threats hint family,
  // M10.F3, view/threats.ts threatLines): every win it would have with one
  // move on its turn, beyond the strip's strongest glow and military badge.
  // From the page's game context, or `threats` (the gallery); none when
  // the family is hidden on this screen or the game is over.
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { sessionHintScreen } from '$lib/view/hints';
  import { threatLines, type ThreatLine } from '$lib/view/threats';
  import { tryGameContext } from '../choices/room';
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
    /** Each seat's threats lines; else from the page's game context. */
    threats?: readonly [readonly ThreatLine[], readonly ThreatLine[]];
  }

  let {
    open = $bindable(false),
    names,
    players,
    military,
    scores,
    bots,
    activeSeat = $bindable(0),
    titleId = 'city-sheet-title',
    threats
  }: Props = $props();

  const seats = [0, 1] as const;
  const ctx = tryGameContext();
  const screen = $derived(sessionHintScreen(ctx?.session, hintHide.value));
  const shownThreats = $derived.by((): readonly ThreatLine[] => {
    if (threats) return threats[activeSeat];
    if (!screen) return [];
    return threatLines(players[activeSeat].hints, activeSeat, screen.families, screen.names);
  });
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

    {#if shownThreats.length > 0}
      <section
        class="flex flex-col gap-1 rounded-control border border-cinnabar/45 bg-panel-sunken px-3 py-2"
        aria-labelledby="{titleId}-threats"
        data-threats={activeSeat}
      >
        <h3
          id="{titleId}-threats"
          class="flex items-center gap-1.5 text-xs font-semibold tracking-wide text-fg-muted uppercase"
        >
          <Icon name="hint-crown" class="text-sm text-cinnabar-ink" />
          {t('hints.threats.title')}
        </h3>
        <ul class="flex flex-col gap-0.5 text-sm leading-snug text-fg">
          {#each shownThreats as line (line.kind)}
            <li data-threat={line.kind} dir="auto">{line.text}</li>
          {/each}
        </ul>
      </section>
    {/if}
  </div>
</Sheet>
