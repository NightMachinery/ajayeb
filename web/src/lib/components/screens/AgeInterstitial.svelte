<script lang="ts">
  // The end-of-age summary: both players' coins and points and the military
  // lead, right after an age ends. A plain leaf: the caller puts it in a
  // dialog together with the starter choice (or a waiting note, or a button
  // to begin the next age) through `children`, so a player always has a way
  // to continue. Every number is the server's (coins, the score preview,
  // the pawn's position); nothing is computed here.
  //
  // Hints (M10.HS4, symbol-status): one line for each science pair the
  // ended age killed (its other card was set aside), from `notes` or the
  // page's game context (view/board-hints.ts deadPairLines).
  import type { Snippet } from 'svelte';
  import { hintHide } from '$lib/hints/local.svelte';
  import { deadPairLines } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { tryGameContext } from '../choices/room';
  import { formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryLeader, militaryMargin } from '$lib/view/threats';
  import type { MilitaryView, Pair, PlayerView, ScoreView, View } from '$lib/wire';

  interface Props {
    /** The age that just ended (1 or 2). */
    age: number;
    /** Display names by seat (user text). */
    names: readonly [string, string];
    players: Pair<PlayerView>;
    scores: Pair<ScoreView>;
    military: MilitaryView;
    /** The seat that reads as "you", if any. */
    you?: 0 | 1;
    titleId?: string;
    /** The way to continue: the starter choice, a waiting note, a button. */
    children?: Snippet;
    /** Hint lines about the ended age; else from the game context. */
    notes?: readonly string[];
    /** Frozen age-boundary hints, when this summary outlives that frame. */
    hintView?: View;
  }

  let {
    age,
    names,
    players,
    scores,
    military,
    you,
    titleId = 'age-interstitial-title',
    children,
    notes,
    hintView
  }: Props = $props();

  const ctx = tryGameContext();
  const lines = $derived.by((): readonly string[] => {
    if (notes) return notes;
    const view = hintView ?? ctx?.session.view;
    const screen = sessionHintScreen(ctx?.session, hintHide.value);
    return view && screen ? deadPairLines(view, age, screen.families, screen.names) : [];
  });

  const seats = [0, 1] as const;
  const leader = $derived(militaryLeader(military));
  const margin = $derived(militaryMargin(military));

  const militaryText = $derived.by(() => {
    if (leader === null) return t('choices.age.militaryEven');
    if (leader === you) return t('choices.age.militaryLeadYou', { margin });
    return t('choices.age.militaryLead', { name: names[leader], margin });
  });
</script>

<section class="flex flex-col gap-4" data-age-interstitial={age} aria-labelledby={titleId}>
  <h2 id={titleId} class="text-center font-display text-2xl text-fg">
    {t('choices.age.complete', { age })}
  </h2>

  <div class="grid grid-cols-2 gap-2" role="group" aria-label={t('choices.age.summary', { age })}>
    {#each seats as seat (seat)}
      <div
        class="flex min-w-0 flex-col gap-1.5 rounded-control border-2 bg-panel-sunken px-3 py-2
          {seat === 0 ? 'border-sun' : 'border-moon'}"
        data-seat-summary={seat}
      >
        <span
          class="flex min-w-0 items-center gap-1.5 text-sm font-semibold
            {seat === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
        >
          <Icon name={seat === 0 ? 'crest-sun' : 'crest-moon'} class="shrink-0 text-lg" />
          <span class="truncate" dir="auto">{names[seat]}</span>
          {#if you === seat}
            <span class="shrink-0 font-normal text-fg-muted">{t('choices.age.you')}</span>
          {/if}
        </span>
        <dl class="grid grid-cols-[auto_1fr] items-center gap-x-2 gap-y-1 text-sm text-fg">
          <dt class="flex items-center gap-1 text-fg-muted">
            <Icon name="coin" class="text-coin-ink" />
            {t('choices.age.coins')}
          </dt>
          <dd class="text-end font-semibold tabular-nums" data-must-fit>
            {formatNumber(players[seat].coins)}
          </dd>
          <dt class="flex items-center gap-1 text-fg-muted">
            <Icon name="vp-laurel" class="text-gild" />
            {t('choices.age.points')}
          </dt>
          <dd class="text-end font-semibold tabular-nums" data-must-fit>
            {formatNumber(scores[seat].total)}
          </dd>
        </dl>
      </div>
    {/each}
  </div>

  <p class="flex items-center justify-center gap-1.5 text-center text-sm text-fg">
    <Icon name="shield" class="shrink-0 text-card-red-ink" />
    <span>{militaryText}</span>
  </p>

  {#if lines.length > 0}
    <ul class="flex flex-col gap-1 text-center text-sm text-fg-muted" data-age-notes>
      {#each lines as line (line)}
        <li class="flex items-center justify-center gap-1.5" data-age-note>
          <Icon name="category-green-sm" class="shrink-0 text-card-green-ink" />
          <span>{line}</span>
        </li>
      {/each}
    </ul>
  {/if}

  {#if children}
    <div class="border-t border-rule pt-4">
      {@render children()}
    </div>
  {/if}
</section>
