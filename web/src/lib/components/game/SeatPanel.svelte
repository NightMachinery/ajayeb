<script lang="ts">
  // One player's side of the board, in the form the viewport class wants
  // (UX design section 3):
  //
  //   strip   phone portrait and phone landscape: the compact PlayerStrip,
  //           whose crest and name open the city sheet
  //   band    tablet portrait: the strip plus the city as band stacks
  //   column  desktop: the full CityPanel
  //
  // `lit` rings the panel in the seat's crest colour: the player to move in
  // a hot-seat game with fixed orientation. `thinking` shows the calm
  // indicator of a bot seat that is thinking. `presence` shows an online
  // player's presence: on the strip by the name, and on
  // a line above the desktop column. Wonder taps (the wonder-first flow)
  // are reported through onwonder when the caller passes it.
  import type { Color, MilitaryView, PlayerView, ScoreView, WonderView } from '$lib/wire';
  import { cardBySlug } from '$lib/catalog';
  import { t } from '$lib/i18n';
  import type { Presence } from '$lib/ui/online';
  import PresenceDot from '../hud/PresenceDot.svelte';
  import CityBand from '../city/CityBand.svelte';
  import CityPanel from '../city/CityPanel.svelte';
  import PlayerStrip from '../city/PlayerStrip.svelte';

  interface Props {
    variant: 'strip' | 'band' | 'column';
    name: string;
    seat: 0 | 1;
    player: PlayerView;
    military: MilitaryView;
    score?: ScoreView;
    lit?: boolean;
    /** This seat's bot is thinking. */
    thinking?: boolean;
    /** The player's presence in an online room. */
    presence?: Presence;
    onwonder?: (wonder: WonderView) => void;
    onopen?: () => void;
    class?: string;
  }

  let {
    variant,
    name,
    seat,
    player,
    military,
    score,
    lit = false,
    thinking = false,
    presence,
    onwonder,
    onopen,
    class: className = ''
  }: Props = $props();

  const COLORS = [
    'brown',
    'grey',
    'blue',
    'green',
    'yellow',
    'red',
    'purple'
  ] as const satisfies readonly Color[];

  const counts = $derived.by(() => {
    const out: Record<Color, number> = {
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
      if (color) out[color] += 1;
    }
    return out;
  });

  const ring = $derived(lit ? (seat === 0 ? 'lit-sun' : 'lit-moon') : '');
</script>

<div
  class="seat-panel min-w-0 rounded-panel {ring} {className}"
  data-seat={seat}
  data-lit={lit ? '' : undefined}
  data-thinking={thinking ? '' : undefined}
  data-presence={presence}
>
  {#if variant === 'column'}
    {#if presence}
      <div class="flex justify-end px-1 pb-1">
        <PresenceDot {presence} {name} labelled />
      </div>
    {/if}
    <CityPanel {name} {seat} {player} {military} {score} {onwonder} {thinking} class="bg-panel" />
  {:else}
    <PlayerStrip
      {name}
      {seat}
      {player}
      {military}
      {score}
      {onwonder}
      {thinking}
      {presence}
      {onopen}
      openLabel={t('game.openCity', { name })}
      class="bg-panel"
    />
    {#if variant === 'band' && player.city.length > 0}
      <div
        class="mt-1 flex flex-wrap gap-1.5 px-1"
        role="group"
        aria-label={t('board.citySheet.city')}
      >
        {#each COLORS as color (color)}
          <CityBand {color} count={counts[color]} />
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .lit-sun {
    box-shadow: 0 0 0 2px var(--sun);
  }
  .lit-moon {
    box-shadow: 0 0 0 2px var(--moon);
  }
</style>
