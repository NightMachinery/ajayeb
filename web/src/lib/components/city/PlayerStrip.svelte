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
  // the rows wrap.
  //
  // Glows when this seat is in military or scientific danger
  // (lib/view/threats.ts), never once the game has ended. A leaf
  // component: every number comes from the view through plain props.
  import BotAvatar from '$lib/art/BotAvatar.svelte';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryDanger, scienceDanger } from '$lib/view/threats';
  import { wonderTally } from '$lib/view/wonders';
  import type { Presence } from '$lib/ui/online';
  import type { BotLevel, MilitaryView, PlayerView, ScoreView } from '$lib/wire';
  import PresenceDot from '../hud/PresenceDot.svelte';
  import ThinkingDots from '../hud/ThinkingDots.svelte';
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
    class: className = ''
  }: Props = $props();

  const crestName = $derived(seat === 0 ? ('crest-sun' as const) : ('crest-moon' as const));
  const crestTextClass = $derived(seat === 0 ? 'text-sun-ink' : 'text-moon-ink');
  const militaryThreat = $derived(militaryDanger(military, seat, ended));
  const scienceThreat = $derived(scienceDanger(player.science, ended));
  const danger = $derived(militaryThreat || scienceThreat);
  const tally = $derived(wonderTally(player));
</script>

{#snippet identity()}
  <Icon name={crestName} class="shrink-0 text-lg {crestTextClass}" />
  {#if bot}<BotAvatar level={bot} class="size-6 shrink-0 short:size-5" />{/if}
  <span class="min-w-0 truncate text-sm font-semibold text-fg" dir="auto">{name}</span>
{/snippet}

<div
  class="player-strip flex flex-col gap-1 rounded-panel border px-2 py-1.5
    {danger ? 'border-cinnabar ring-2 ring-cinnabar/25' : 'border-rule'} {className}"
  data-danger={danger ? '' : undefined}
>
  <div class="strip-row flex min-w-0 items-center gap-x-1.5 gap-y-1 short:flex-wrap">
    {#if onopen}
      <button
        type="button"
        class="-ms-1 flex min-h-9 min-w-0 shrink items-center gap-1.5 rounded-control px-1 hover:bg-panel-sunken"
        aria-label={openLabel}
        data-control="open-city"
        onclick={onopen}
      >
        {@render identity()}
      </button>
    {:else}
      <div class="flex min-h-9 min-w-0 shrink items-center gap-1.5">
        {@render identity()}
      </div>
    {/if}

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
        class="grid size-6 shrink-0 place-items-center rounded-full bg-cinnabar/15 text-cinnabar"
        role="img"
        aria-label={t('board.player.militaryDanger')}
        data-military-danger
      >
        <Icon name="shield" class="text-sm" />
      </span>
    {/if}

    <span class="flex-1 short:hidden" aria-hidden="true"></span>

    {#if player.tokens.length > 0}
      <ul class="flex shrink-0 items-center ps-1.5" aria-label={t('board.citySheet.tokens')}>
        {#each player.tokens as slug (slug)}
          <li class="-ms-1.5"><ProgressToken {slug} size="xs" owned /></li>
        {/each}
      </ul>
    {/if}

    <Tally kind="coins" value={player.coins} label={t('coins.count', { count: player.coins })} />
    {#if score}
      <Tally
        kind="points"
        value={score.total}
        label={t('board.player.scorePreview', { total: score.total })}
      />
    {/if}

    <WonderButton
      built={tally.built}
      total={tally.total}
      buildable={wonderBuildable}
      {name}
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
    <ScienceTracker science={player.science} pill {ended} />
  </div>
</div>
