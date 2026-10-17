<script lang="ts">
  // The phone layout's player strip (UX design sec. 3), in two short rows
  // so that both strips fit a 360 px phone without scrolling:
  //
  //   1. the crest, the bot's avatar and the name (a button that opens the
  //      city sheet), presence and thinking, a military danger mark, the
  //      owned progress tokens, coins, victory points, and the wonder
  //      button ("1/4", glowing when a wonder can be built now)
  //   2. the five resource chips (production, then the price as a coin)
  //      and the science count
  //
  // The two strips have the same layout. Everything else (the science
  // slots, the city by colour, each wonder) is in the city sheet and the
  // wonder sheet. In phone landscape, where the strip is a narrow column,
  // the rows wrap: the first always breaks after the name and puts the
  // owned tokens last, so both strips have their tallies and wonder button
  // on one line of their own.
  //
  // Glows when this seat is in military or scientific danger
  // (lib/view/threats.ts), never once the game has ended. A leaf
  // component: every number comes from the view through plain props.
  //
  // The player's own seat has a ring on its crest and a "you" mark (the
  // `self` prop, else the room's SeatInfo.you from the page's game context,
  // never on a hot-seat device), and the device link button (SeatLink) sits
  // after the name where the room allows it.
  //
  // The name button draws 36 px tall to keep the strip short, but takes taps
  // over 44 px: a transparent ::after reaches 4 px above and below it (the
  // wonder button does the same).
  import BotAvatar from '$lib/art/BotAvatar.svelte';
  import { formatNumber, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Icon from '$lib/icons/Icon.svelte';
  import { isSelfSeat } from '$lib/view/display-name';
  import { hintHide } from '$lib/hints/local.svelte';
  import { sessionHintScreen } from '$lib/view/hints';
  import { threatList } from '$lib/view/outcome';
  import { coinsLines, scoreLines } from '$lib/view/tips';
  import {
    glowHints,
    glowWarns,
    militaryGlow,
    scienceGlow,
    strongerGlow,
    winsOf
  } from '$lib/view/threats';
  import { wonderTally } from '$lib/view/wonders';
  import type { Presence } from '$lib/ui/online';
  import type { BotLevel, MilitaryView, PlayerView, ScoreView } from '$lib/wire';
  import PresenceDot from '../hud/PresenceDot.svelte';
  import ThinkingDots from '../hud/ThinkingDots.svelte';
  import { tryGameContext } from '../choices/room';
  import SeatLink from '../identity/SeatLink.svelte';
  import SelfCrest from '../identity/SelfCrest.svelte';
  import SelfMark from '../identity/SelfMark.svelte';
  import ProgressToken from '../token/ProgressToken.svelte';
  import WonderButton from '../wonder/WonderButton.svelte';
  import ResourceChips from './ResourceChips.svelte';
  import ScienceTracker from './ScienceTracker.svelte';
  import Tally from './Tally.svelte';

  interface Props {
    /** The seat's display name (RoomInfo, not the view): data, like a
     *  player's own words, so it is shown as given, through dir="auto". */
    name: string;
    seat: 0 | 1;
    player: PlayerView;
    military: MilitaryView;
    score?: ScoreView;
    /** When given, the wonder button opens this seat's wonder sheet. */
    onwonders?: () => void;
    /** A wonder of this seat can be built now (from the view's choices). */
    wonderBuildable?: boolean;
    /** The game has ended: no danger warnings. */
    ended?: boolean;
    /** The bot's level when a bot plays this seat: its avatar shows by the name. */
    bot?: BotLevel;
    /** This seat's bot is thinking: a calm indicator shows by the name. */
    thinking?: boolean;
    /** The player's presence in an online room; nothing shows without it. */
    presence?: Presence;
    /** When given, the crest and name are a button that opens this city in full. */
    onopen?: () => void;
    /** The label of that button. */
    openLabel?: string;
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
    onwonders,
    wonderBuildable = false,
    ended = false,
    bot,
    thinking = false,
    presence,
    onopen,
    openLabel,
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
  const tally = $derived(wonderTally(player));
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

{#snippet identity()}
  <SelfCrest {seat} self={isSelf} class="text-lg" />
  {#if bot}<BotAvatar level={bot} class="size-6 shrink-0 short:size-5" />{/if}
  <span class="min-w-[3ch] truncate text-sm font-semibold text-fg" dir="auto">{name}</span>
  {#if isSelf}<SelfMark class="px-1! text-[9px]! tracking-normal!" />{/if}
{/snippet}

<div
  class="player-strip flex flex-col gap-1 rounded-panel border px-2 py-1.5
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
  <div class="strip-row flex min-w-0 items-center gap-x-1 gap-y-1 short:flex-wrap">
    {#if onopen}
      <button
        type="button"
        class="-ms-1 flex min-h-11 min-w-11 shrink items-center gap-1 rounded-control px-1 hover:bg-panel-sunken"
        aria-label={openLabel}
        data-control="open-city"
        data-fx-city
        onclick={onopen}
      >
        {@render identity()}
      </button>
    {:else}
      <div class="flex min-h-9 min-w-0 shrink items-center gap-1" data-fx-city>
        {@render identity()}
      </div>
    {/if}

    <SeatLink {seat} {name} class="-ms-1" />

    {#if presence}
      <PresenceDot {presence} {name} class="-ms-1" />
    {/if}

    {#if thinking}
      <span class="-ms-0.5 flex shrink-0 items-center" data-thinking={seat}>
        <ThinkingDots class={crestTextClass} />
        <span class="sr-only">{t('board.player.thinking')}</span>
      </span>
    {/if}

    {#if militaryThreat}
      <span
        class="grid size-6 shrink-0 place-items-center rounded-full bg-cinnabar/15 text-cinnabar-ink"
        role="img"
        aria-label={militaryTip}
        data-military-danger={militaryLevel}
        {@attach tooltip({ text: militaryTip })}
      >
        <Icon name="shield" class="text-sm" />
      </span>
    {/if}

    <span class="flex-1 short:h-0 short:basis-full" aria-hidden="true"></span>

    {#if player.tokens.length > 0}
      <ul
        class="flex shrink-0 items-center ps-1.5 short:order-last"
        aria-label={t('board.citySheet.tokens')}
        data-fx-tokens
      >
        {#each player.tokens as slug (slug)}
          <li class="-ms-1.5"><ProgressToken {slug} size="xs" owned /></li>
        {/each}
      </ul>
    {/if}

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

    <WonderButton
      built={tally.built}
      total={tally.total}
      buildable={wonderBuildable}
      {name}
      {seat}
      onclick={onwonders}
    />
  </div>

  <div
    class="strip-row flex min-w-0 items-center justify-between gap-x-1.5 gap-y-1 short:flex-wrap"
  >
    <ResourceChips
      production={player.production}
      tradePrices={player.tradePrices}
      compact
      class="flex-nowrap! short:flex-wrap!"
    />
    <ScienceTracker science={player.science} pill {ended} {seat} glow={scienceLevel} />
  </div>
</div>
