<script lang="ts">
  // The tablet/desktop column for one seat (UX design sec. 3): name, crest,
  // coins and score, city cards as band stacks grouped by colour, wonders
  // as a list of rows (art, name, effect, cost and one status, and "Can
  // build now" from the view's choices), science and owned tokens, and a
  // calm thinking indicator while this seat's bot thinks. The same data as
  // PlayerStrip, laid out as a column instead of two rows. A bot seat shows
  // its persona's avatar between the crest and the name. No danger warning
  // once the game has ended. The player's own seat has a ring on its crest
  // and a "you" mark (`self`, else the room's SeatInfo.you from the page's
  // game context, never on a hot-seat device), and the device link button
  // (SeatLink) follows the name where the room allows it.
  import type { Presence } from '$lib/ui/online';
  import BotAvatar from '$lib/art/BotAvatar.svelte';
  import { formatNumber, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { cardBySlug } from '$lib/catalog';
  import { isSelfSeat } from '$lib/view/display-name';
  import { hintHide } from '$lib/hints/local.svelte';
  import { sessionHintScreen } from '$lib/view/hints';
  import { threatList } from '$lib/view/outcome';
  import { coinsLines, guildLines, scoreLines } from '$lib/view/tips';
  import {
    glowHints,
    glowWarns,
    militaryGlow,
    scienceGlow,
    strongerGlow,
    winsOf
  } from '$lib/view/threats';
  import type { BotLevel, Color, MilitaryView, PlayerView, ScoreView, WonderView } from '$lib/wire';
  import PresenceDot from '../hud/PresenceDot.svelte';
  import ThinkingDots from '../hud/ThinkingDots.svelte';
  import { tryGameContext } from '../choices/room';
  import SeatLink from '../identity/SeatLink.svelte';
  import SelfCrest from '../identity/SelfCrest.svelte';
  import SelfMark from '../identity/SelfMark.svelte';
  import ProgressToken from '../token/ProgressToken.svelte';
  import WonderList from '../wonder/WonderList.svelte';
  import CityBand from './CityBand.svelte';
  import ResourceChips from './ResourceChips.svelte';
  import ScienceTracker from './ScienceTracker.svelte';
  import Tally from './Tally.svelte';

  interface Props {
    name: string;
    seat: 0 | 1;
    player: PlayerView;
    military: MilitaryView;
    score?: ScoreView;
    /** When given, the wonders still to build are buttons (the wonder-first flow). */
    onwonder?: (wonder: WonderView) => void;
    /** Wonders a choice of the view can build now (view/wonders). */
    buildable?: ReadonlySet<string>;
    /** The game has ended: no danger warnings. */
    ended?: boolean;
    /** The bot's level when a bot plays this seat: its avatar shows by the name. */
    bot?: BotLevel;
    /** This seat's bot is thinking: a calm indicator shows by the name. */
    thinking?: boolean;
    /** Online presence, beside the name within the header. */
    presence?: Presence;
    /** This is the player's own seat; by default the room's SeatInfo.you. */
    self?: boolean;
    /** The selected move's coins and score total after it (a selection
     *  ghost, the hints design hud-ghosts): shown beside the tallies when
     *  they differ. */
    ghost?: { readonly coins?: number; readonly score?: number };
    class?: string;
  }

  let {
    name,
    seat,
    player,
    military,
    score,
    onwonder,
    buildable,
    ended = false,
    bot,
    thinking = false,
    presence,
    self,
    ghost,
    class: className = ''
  }: Props = $props();

  const ctx = tryGameContext();
  const isSelf = $derived(
    self ??
      isSelfSeat(
        ctx?.session.frame?.room.seats.find((s) => s.seat === seat),
        ctx?.session.frame?.room.mode
      )
  );

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

  const crestTextClass = $derived(seat === 0 ? 'text-sun-ink' : 'text-moon-ink');
  // The glows read the board hints when this screen shows them (M10.HS4,
  // lib/view/threats.ts): calm, soft or strong; else the coarse rules.
  const hintScreen = $derived(ended ? undefined : sessionHintScreen(ctx?.session, hintHide.value));
  const glowIn = $derived(glowHints(ctx?.session.view, hintScreen));
  const militaryLevel = $derived(militaryGlow(military, seat, ended, glowIn));
  const scienceLevel = $derived(scienceGlow(player.science, seat, ended, glowIn));
  const glow = $derived(strongerGlow(militaryLevel, scienceLevel));
  const militaryThreat = $derived(glowWarns(militaryLevel));
  const scienceThreat = $derived(glowWarns(scienceLevel));
  const danger = $derived(militaryThreat || scienceThreat);
  // The tallies' tooltips: the scoring hints as status lines (M10.HS4's
  // coinsToNextVp and previewWinner, view/tips.ts).
  const coinsTip = $derived({
    text: t('coins.count', { count: player.coins }),
    lines: coinsLines(player.hints, hintScreen)
  });
  const scoreTip = $derived.by(() => {
    const lines = scoreLines(ctx?.session.view?.hints, hintScreen);
    return {
      text: t('board.player.scorePreview', { total: score?.total ?? 0 }),
      lines,
      rulesId: lines.length > 0 ? ('scoring' as const) : undefined
    };
  });
  const militaryTip = $derived.by(() => {
    const wins = winsOf(glowIn?.hints[seat === 0 ? 1 : 0], 'military');
    if (militaryLevel !== 'strong' || wins.length === 0) return t('board.player.militaryDanger');
    const attacker = seat === 0 ? 1 : 0;
    return hintScreen?.names.you === attacker
      ? t('hints.glow.militaryWinYou', { list: threatList(wins) })
      : t('hints.glow.militaryWin', {
          name: hintScreen?.names.names[attacker] ?? '',
          list: threatList(wins)
        });
  });
</script>

{#snippet ghostTally(kind: 'coins' | 'points', value: number)}
  <span
    class="ghost-tally inline-flex h-5 shrink-0 items-center rounded-full border border-dashed border-focus px-1.5 text-[13px] leading-none font-bold text-fg tabular-nums"
    data-ghost-tally={kind}
    {@attach tooltip({
      text:
        kind === 'coins'
          ? t('hints.ghost.coins', { count: value })
          : t('hints.ghost.score', { count: value })
    })}
  >
    <span class="sr-only">
      {kind === 'coins'
        ? t('hints.ghost.coins', { count: value })
        : t('hints.ghost.score', { count: value })}
    </span>
    <span aria-hidden="true">{formatNumber(value)}</span>
  </span>
{/snippet}

<div
  class="city-panel flex w-full flex-col gap-2.5 rounded-panel border p-3
    {glow === 'strong'
    ? 'border-cinnabar ring-[3px] ring-cinnabar/45'
    : danger
      ? 'border-cinnabar ring-2 ring-cinnabar/25'
      : glow === 'calm'
        ? 'border-cinnabar/45'
        : 'border-rule'} {className}"
  data-danger={danger ? '' : undefined}
  data-glow={glow}
>
  <div class="flex flex-wrap items-center justify-between gap-x-2 gap-y-1">
    <div class="flex min-w-[7rem] flex-1 items-center gap-1.5">
      <SelfCrest {seat} self={isSelf} class="text-xl" />
      {#if bot}<BotAvatar level={bot} class="size-7" />{/if}
      <span class="min-w-0 truncate font-display text-base text-fg" dir="auto">{name}</span>
      {#if isSelf}<SelfMark />{/if}
      {#if presence}<PresenceDot {presence} {name} />{/if}
      <SeatLink {seat} {name} class="-ms-1" />
      {#if thinking}
        <span
          class="flex shrink-0 items-center gap-1 text-xs font-semibold text-fg-muted"
          data-thinking={seat}
        >
          <ThinkingDots class={crestTextClass} />
          {t('board.player.thinking')}
        </span>
      {/if}
    </div>
    <div class="ms-auto flex shrink-0 items-center gap-1.5">
      <Tally
        kind="coins"
        value={player.coins}
        label={t('coins.count', { count: player.coins })}
        tip={coinsTip}
      />
      {#if ghost?.coins !== undefined && ghost.coins !== player.coins}
        {@render ghostTally('coins', ghost.coins)}
      {/if}
      {#if score}
        <Tally
          kind="points"
          value={score.total}
          label={t('board.player.scorePreview', { total: score.total })}
          tip={scoreTip}
        />
        {#if ghost?.score !== undefined && ghost.score !== score.total}
          {@render ghostTally('points', ghost.score)}
        {/if}
      {/if}
    </div>
  </div>

  {#if militaryThreat || scienceThreat}
    <div class="flex flex-wrap gap-1">
      {#if militaryThreat}
        <span
          class="rounded-control bg-cinnabar/15 px-1.5 py-0.5 text-[11px] font-semibold text-cinnabar-ink"
          data-military-danger={militaryLevel}
        >
          {militaryLevel === 'strong' ? militaryTip : t('board.player.militaryDanger')}
        </span>
      {/if}
      {#if scienceThreat}
        <span
          class="rounded-control bg-cinnabar/15 px-1.5 py-0.5 text-[11px] font-semibold text-cinnabar-ink"
        >
          {t('board.player.scienceDanger')}
        </span>
      {/if}
    </div>
  {/if}

  <ResourceChips production={player.production} tradePrices={player.tradePrices} />
  <ScienceTracker science={player.science} {ended} {seat} glow={scienceLevel} />

  <div class="flex flex-col gap-1">
    <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
      {t('board.citySheet.city')}
    </h3>
    <div class="flex flex-wrap gap-1.5" aria-label={t('board.citySheet.city')} data-fx-city>
      {#each COLORS as color (color)}
        <CityBand
          {color}
          count={cityCounts[color]}
          lines={color === 'purple' ? guildLines(player.hints?.guildPoints, hintScreen) : []}
        />
      {/each}
    </div>
  </div>

  <div class="flex flex-col gap-1" data-tut="wonders">
    <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
      {t('board.citySheet.wonders')}
    </h3>
    <WonderList wonders={player.wonders} {buildable} onselect={onwonder} />
  </div>

  {#if player.tokens.length > 0}
    <div class="flex flex-col gap-1">
      <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
        {t('board.citySheet.tokens')}
      </h3>
      <ul class="flex flex-wrap gap-1" aria-label={t('board.citySheet.tokens')} data-fx-tokens>
        {#each player.tokens as slug (slug)}
          <li><ProgressToken {slug} size="sm" owned /></li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
