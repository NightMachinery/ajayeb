<script lang="ts">
  // The phone layout's player strip (UX design sec. 3): name, crest, coins,
  // the running score preview, the five resource chips, science, wonders
  // and owned tokens, in one compact row that wraps onto two lines at 360px
  // wide. Glows when this seat is in military or scientific danger
  // (lib/view/threats.ts), and shows a calm thinking indicator while this
  // seat's bot is thinking. In an online room the other player's strip
  // shows their presence (connected, or away once they drop). A leaf
  // component: every number comes from the view through plain props,
  // nothing computed here but the danger check. A bot seat shows its
  // persona's avatar between the crest and the name.
  import BotAvatar from '$lib/art/BotAvatar.svelte';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryDanger, scienceDanger } from '$lib/view/threats';
  import type { Presence } from '$lib/ui/online';
  import type { BotLevel, MilitaryView, PlayerView, ScoreView, WonderView } from '$lib/wire';
  import PresenceDot from '../hud/PresenceDot.svelte';
  import ThinkingDots from '../hud/ThinkingDots.svelte';
  import ProgressToken from '../token/ProgressToken.svelte';
  import ResourceChips from './ResourceChips.svelte';
  import ScienceTracker from './ScienceTracker.svelte';
  import WonderRow from './WonderRow.svelte';

  interface Props {
    /** The seat's display name (RoomInfo, not the view): data, like a
     *  player's own words, so it is shown as given, through dir="auto". */
    name: string;
    seat: 0 | 1;
    player: PlayerView;
    military: MilitaryView;
    score?: ScoreView;
    /** When given, the wonders still to build are buttons (the wonder-first flow). */
    onwonder?: (wonder: WonderView) => void;
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
    onwonder,
    bot,
    thinking = false,
    presence,
    onopen,
    openLabel,
    class: className = ''
  }: Props = $props();

  const crestName = $derived(seat === 0 ? ('crest-sun' as const) : ('crest-moon' as const));
  const crestTextClass = $derived(seat === 0 ? 'text-sun-ink' : 'text-moon-ink');
  const militaryThreat = $derived(militaryDanger(military, seat));
  const scienceThreat = $derived(scienceDanger(player.science));
  const danger = $derived(militaryThreat || scienceThreat);
</script>

<div
  class="player-strip flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-panel border px-2.5 py-1.5
    {danger ? 'border-cinnabar ring-2 ring-cinnabar/25' : 'border-rule'} {className}"
>
  {#if onopen}
    <button
      type="button"
      class="-ms-1 flex min-h-8 min-w-0 items-center gap-1.5 rounded-control px-1 hover:bg-panel-sunken"
      aria-label={openLabel}
      data-control="open-city"
      onclick={onopen}
    >
      <Icon name={crestName} class="shrink-0 text-lg {crestTextClass}" />
      {#if bot}<BotAvatar level={bot} class="size-6 short:size-5" />{/if}
      <span class="min-w-0 truncate text-sm font-semibold text-fg" dir="auto">{name}</span>
    </button>
  {:else}
    <div class="flex min-w-0 items-center gap-1.5">
      <Icon name={crestName} class="shrink-0 text-lg {crestTextClass}" />
      {#if bot}<BotAvatar level={bot} class="size-6 short:size-5" />{/if}
      <span class="min-w-0 truncate text-sm font-semibold text-fg" dir="auto">{name}</span>
    </div>
  {/if}

  {#if presence}
    <PresenceDot {presence} {name} class="-ms-1.5" />
  {/if}

  {#if thinking}
    <span
      class="-ms-1.5 flex items-center gap-1 text-xs font-semibold text-fg-muted"
      data-thinking={seat}
    >
      <ThinkingDots class={crestTextClass} />
      {t('board.player.thinking')}
    </span>
  {/if}

  <div class="flex items-center gap-2 text-sm font-semibold tabular-nums">
    <span class="flex items-center gap-0.5" title={t('coins.count', { count: player.coins })}>
      <Icon name="coin" class="text-coin-ink" />
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

  <ResourceChips production={player.production} tradePrices={player.tradePrices} compact />
  <ScienceTracker science={player.science} compact />
  <WonderRow wonders={player.wonders} variant="chips" onselect={onwonder} />

  {#if player.tokens.length > 0}
    <ul class="flex gap-1" aria-label={t('board.citySheet.tokens')}>
      {#each player.tokens as slug (slug)}
        <li><ProgressToken {slug} size="sm" owned /></li>
      {/each}
    </ul>
  {/if}
</div>
