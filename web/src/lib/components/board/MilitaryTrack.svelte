<script lang="ts">
  // The military track as its 19 printed spaces (docs/rules/military-
  // science.md): a capital at each end in its seat's colour, the zones of
  // 2, 5 and 10 points shaded deeper toward each capital with a line at
  // every zone boundary, each looting token ON the space where it triggers
  // (a gold "−2" or "−5", the coins lost as soon as the pawn enters that
  // space; a dashed ring once taken), and the pawn at its live position.
  // Under the bar, apart from the coin values, each zone's points sit on a
  // green laurel tablet in the middle of the zone (decoration from the
  // printed board; the live score is always scorePreview from the view).
  // `legend` adds one line that says how looting works (the desktop
  // board). `compact` is the landscape top-bar variant: the bar alone,
  // smaller, with the same data and the same aria-label.
  //
  // The board is spatial, not text (AGENTS.md: the board region is
  // dir="ltr"), so everything on the bar is positioned with a physical
  // `left` in the style block below, not a logical inset. Which capital
  // sits at the left end is `leftSeat`: the seat drawn at the bottom or on
  // the left of the board (UX design, decision 6). For a player at the
  // Moon the track is mirrored (view/track-cells.ts, lib/ui/track.ts), and
  // the pawn, the zones and the looting tokens follow.
  //
  // `ghost` (the hints design, pawn-preview, M10.HS3) is the move being
  // considered: an outlined pawn where it would put the pawn, and a dashed
  // ring on each looting token it would take. Both are the server's
  // outcome (Choice.outcome.pawn); the track only draws them.
  //
  // A capital is an instant win: under each end of the bar a small crown
  // says so ("Reaching this space wins the game at once"), apart from the
  // zone tablets, whose tooltips say that only the player ahead scores
  // them, at the end. Board hints (M10.HS4, the military family, from the
  // page's game context unless `hints` is given): each capital's tooltip
  // says how many shields the opponent still needs and whether it can
  // still gather them (capital-distance, capital-safe), with the number on
  // the capital at 5 or fewer; the pawn's tooltip says the same for the
  // capital it leans toward; and each looting token says what it would take
  // now (looting-token-tooltip).
  import { hintHide } from '$lib/hints/local.svelte';
  import { i18n, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { trackEnds, trackMirrored, type Seat } from '$lib/ui/track';
  import { militaryLeader, militaryMargin, nextLooting } from '$lib/view/threats';
  import {
    CAPITAL_SPACE,
    TRACK_ZONES,
    cellCenter,
    zoneBoundaries,
    zoneMiddle,
    zoneSpan
  } from '$lib/view/track-cells';
  import type { MilitaryView, PawnMoveView } from '$lib/wire';
  import LootingToken from './LootingToken.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { lootText, trackHints, type TrackHints } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { tryGameContext } from '../choices/room';

  interface Props {
    military: MilitaryView;
    /** This connection's own seat, for "you"/"the opponent" wording. Left
     *  out for a spectator or a gallery demo, which fall back to naming the
     *  crests instead. */
    viewerSeat?: Seat;
    /** The seat whose capital is drawn at the left end (default the Sun). */
    leftSeat?: Seat;
    /** The landscape top-bar variant: the bar alone, smaller. */
    compact?: boolean;
    /** A line under the track explaining looting and the zone points. */
    legend?: boolean;
    /** The considered move's pawn and looting (a selection ghost). */
    ghost?: PawnMoveView;
    /** The board hints (view/board-hints.ts trackHints); else from the game context. */
    hints?: TrackHints;
    class?: string;
  }

  let {
    military,
    viewerSeat,
    leftSeat = 0,
    compact = false,
    legend = false,
    ghost,
    hints,
    class: className = ''
  }: Props = $props();

  const ctx = tryGameContext();
  const th = $derived.by((): TrackHints | undefined => {
    if (hints) return hints;
    const view = ctx?.session.view;
    const screen = sessionHintScreen(ctx?.session, hintHide.value);
    return view && screen ? trackHints(view, screen.families, screen.names) : undefined;
  });

  /** A seat's name for the zone tooltips: the hint names, else "you" or the crest. */
  function seatLabel(seat: Seat): { you: boolean; name: string } {
    if (th) return { you: th.names.you === seat, name: th.names.names[seat] };
    return { you: viewerSeat === seat, name: t(crestKey(seat)) };
  }

  /** The tooltip of a zone's points: they go to the player ahead, at the end. */
  function zoneTip(side: -1 | 1, points: number): string {
    // The pawn on seat 1's side (+) means seat 0 is ahead.
    const who = seatLabel(side > 0 ? 0 : 1);
    return who.you
      ? t('hints.zone.you', { count: points })
      : t('hints.zone.named', { count: points, name: who.name });
  }

  function capitalTip(seat: Seat): string {
    const who = seatLabel(seat);
    const base = who.you
      ? t('hints.capital.titleYou')
      : t('hints.capital.titleNamed', { name: who.name });
    return [base, t('hints.capital.win'), th?.capitals[seat]?.text ?? ''].filter(Boolean).join(' ');
  }

  const pawnTip = $derived.by(() => {
    const endangered = leader === null ? null : leader === 0 ? 1 : 0;
    const line = endangered === null ? '' : (th?.capitals[endangered]?.text ?? '');
    return [leadText, line].filter(Boolean).join(' ');
  });

  // The zone shades, nearest the centre lightest. Spelled out for Tailwind.
  const ZONE_SHADE = ['bg-fg/6', 'bg-fg/12', 'bg-fg/20'] as const;
  const SIDES = [-1, 1] as const;

  function crestKey(seat: Seat) {
    return seat === 0 ? ('board.crest.sun' as const) : ('board.crest.moon' as const);
  }

  /** A capital's position: seat 0's is at -9, seat 1's at +9. */
  function capitalAt(seat: Seat): number {
    return seat === 0 ? -CAPITAL_SPACE : CAPITAL_SPACE;
  }

  const ends = $derived(trackEnds(leftSeat));
  const leader = $derived(militaryLeader(military));
  const margin = $derived(militaryMargin(military));
  const next = $derived(nextLooting(military));
  const boundaries = $derived(zoneBoundaries(leftSeat));

  const leadText = $derived.by(() => {
    if (leader === null) return t('board.military.tied');
    if (viewerSeat !== undefined) {
      return t(leader === viewerSeat ? 'board.military.youLead' : 'board.military.opponentLeads', {
        margin
      });
    }
    return t('board.military.crestLeads', { crest: t(crestKey(leader)), margin });
  });

  const nextText = $derived.by(() => {
    if (!next) return t('board.military.noLootLeft');
    const magnitude = Math.abs(next.at);
    const nextSeat = next.seat as Seat;
    if (viewerSeat !== undefined) {
      return t(
        nextSeat === viewerSeat ? 'board.military.nextLootYou' : 'board.military.nextLootOpponent',
        { value: next.value, magnitude }
      );
    }
    return t('board.military.nextLootCrest', {
      crest: t(crestKey(nextSeat)),
      value: next.value,
      magnitude
    });
  });

  const ariaLabel = $derived(`${leadText} ${nextText}`);
</script>

{#snippet capital(seat: Seat)}
  <div
    class="capital absolute inset-y-0 grid place-items-center {seat === 0
      ? 'bg-sun/30 text-sun-ink'
      : 'bg-moon/25 text-moon-ink'}"
    style:--pos="{cellCenter(capitalAt(seat), leftSeat)}%"
    data-capital={seat}
    data-capital-safe={th?.capitals[seat]?.safe ? '' : undefined}
  >
    {#if th?.capitals[seat]?.numeral}
      <span
        class="capital-numeral rounded-[3px] bg-cinnabar px-0.5 text-[10px] leading-none font-bold text-(--paper-bright) tabular-nums"
        data-min-font="10"
        data-to-capital={th.capitals[seat]?.toCapital}>{th.capitals[seat]?.toCapital}</span
      >
    {:else}
      <!-- About 11 px, where the crests' -sm files read. -->
      <Icon name={seat === 0 ? 'crest-sun-sm' : 'crest-moon-sm'} class="text-[11px]" />
    {/if}
  </div>
{/snippet}

{#snippet crown(seat: Seat)}
  <span
    class="capital-crown absolute top-0 grid h-4 w-4 place-items-center rounded-[4px] border border-(color:--gold-deep) bg-gold text-card-ink"
    style:--pos="{cellCenter(capitalAt(seat), leftSeat)}%"
    data-capital-crown={seat}
    {@attach tooltip({ text: capitalTip(seat) })}
  >
    <Icon name="hint-crown" class="size-3" />
  </span>
{/snippet}

{#snippet tablet(points: number, cls: string)}
  <span
    class="inline-grid h-4 min-w-4 place-items-center rounded-[4px] border border-vp-edge bg-vp px-0.5 text-[10px] leading-none font-bold text-(--paper-bright) tabular-nums {cls}"
    data-min-font="10"
    data-zone-points={points}>{points}</span
  >
{/snippet}

<div
  dir="ltr"
  role="group"
  aria-label={ariaLabel}
  data-left-seat={leftSeat}
  data-mirrored={trackMirrored(leftSeat) ? '' : undefined}
  class="military-track flex flex-col gap-0.5 {compact ? 'min-w-24' : ''} {className}"
>
  <div
    class="track-bar relative rounded-[5px] border border-rule-strong bg-panel {compact
      ? 'h-4'
      : 'h-6'}"
  >
    <div class="absolute inset-0 overflow-hidden rounded-[4px]" aria-hidden="true">
      {#each SIDES as side (side)}
        {#each TRACK_ZONES as zone, i (zone.first)}
          {@const span = zoneSpan(zone, side, leftSeat)}
          <div
            class="zone absolute inset-y-0 {ZONE_SHADE[i]}"
            style:--from="{span.start}%"
            style:--to="{span.end}%"
            data-zone={zone.points}
          ></div>
        {/each}
      {/each}
      {@render capital(ends[0])}
      {@render capital(ends[1])}
    </div>
    {#each ends as seat (seat)}
      <!-- The capital's tooltip target, above the clipped shading layer. -->
      <div
        class="capital capital-hit absolute inset-y-0"
        style:--pos="{cellCenter(capitalAt(seat), leftSeat)}%"
        data-capital-tip={seat}
        {@attach tooltip({ text: capitalTip(seat) })}
      ></div>
    {/each}
    {#each boundaries as at (at)}
      <div
        class="boundary absolute -inset-y-0.5 w-px bg-fg/40"
        style:--pos="{at}%"
        aria-hidden="true"
      ></div>
    {/each}
    {#each military.looting as loot (loot.at)}
      <div
        class="token-spot absolute top-1/2"
        style:--pos="{cellCenter(loot.at, leftSeat)}%"
        data-loot-at={loot.at}
      >
        <LootingToken
          value={loot.value}
          magnitude={Math.abs(loot.at)}
          present={loot.present}
          size={compact ? 'sm' : 'md'}
          tip={th ? lootText(loot, th.wouldLose.get(loot.at), th.names) : undefined}
        />
      </div>
    {/each}
    {#if ghost}
      {#each ghost.loot as taken (taken.at)}
        <div
          class="token-spot loot-ghost absolute top-1/2 z-10 rounded-full border-2 border-dashed border-focus {compact
            ? 'size-5'
            : 'size-7'}"
          style:--pos="{cellCenter(taken.at, leftSeat)}%"
          data-ghost-loot={taken.at}
          {@attach tooltip({ text: t('hints.ghost.loot', { count: taken.lost }) })}
        ></div>
      {/each}
      {#if ghost.to !== military.position}
        <div
          class="pawn pawn-ghost absolute top-1/2 z-20 grid place-items-center rounded-full border-2 border-dashed border-focus {compact
            ? 'size-4'
            : 'size-6'}"
          style:--pos="{cellCenter(ghost.to, leftSeat)}%"
          data-ghost-pawn={ghost.to}
          {@attach tooltip({ text: t('hints.ghost.pawn') })}
        >
          <Icon name="military-pawn" class="{compact ? 'text-[10px]' : 'text-xs'} opacity-60" />
        </div>
      {/if}
    {/if}
    <div
      class="pawn absolute top-1/2 z-10 grid place-items-center rounded-full border-2 border-focus bg-panel shadow-sm {compact
        ? 'size-4'
        : 'size-6'}"
      style:--pos="{cellCenter(military.position, leftSeat)}%"
      data-pawn={military.position}
      aria-hidden="true"
      {@attach tooltip({ text: pawnTip })}
    >
      <Icon name="military-pawn" class={compact ? 'text-[10px]' : 'text-xs'} />
    </div>
  </div>

  {#if !compact}
    <!-- The points each zone is worth to the player ahead at the end:
         decoration from the printed board, never the live score. -->
    <div class="zone-values relative h-4" aria-hidden="true">
      {#each SIDES as side (side)}
        {#each TRACK_ZONES as zone (zone.first)}
          <span
            class="zone-value absolute top-0 flex h-4"
            style:--pos="{zoneMiddle(zone, side, leftSeat)}%"
            {@attach tooltip({ text: zoneTip(side, zone.points) })}
          >
            {@render tablet(zone.points, '')}
          </span>
        {/each}
      {/each}
      {@render crown(ends[0])}
      {@render crown(ends[1])}
    </div>
  {/if}

  {#if legend && !compact}
    <p
      class="mt-1 flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-fg-muted"
      dir={i18n.dir}
    >
      <span class="flex items-center gap-1.5">
        <LootingToken value={2} magnitude={3} present />
        {t('board.military.legendLoot')}
      </span>
      <span class="flex items-center gap-1.5">
        <span class="flex" aria-hidden="true">{@render tablet(5, '')}</span>
        {t('board.military.legendPoints')}
      </span>
      <span class="flex items-center gap-1.5" data-legend-capital>
        <span class="relative flex h-4 w-4" aria-hidden="true">
          <svg
            class="size-4 rounded-[4px] border border-(color:--gold-deep) bg-gold p-px text-card-ink"
            viewBox="0 0 24 24"
          >
            <path
              d="M3 8 L7.5 12 L12 5 L16.5 12 L21 8 L19 18 H5 Z"
              fill="currentColor"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linejoin="round"
            />
            <rect x="5" y="19.5" width="14" height="2" rx="1" fill="currentColor" />
          </svg>
        </span>
        {t('hints.capital.win')}
      </span>
    </p>
  {/if}
</div>

<style>
  /* physical-ok: the board is fixed LTR regardless of reading direction
     (AGENTS.md), so everything on the track sits at a physical offset
     along it, never a logical inset-inline-start. */
  .token-spot,
  .pawn {
    left: var(--pos); /* physical-ok: see the note above */
    translate: -50% -50%;
  }
  .zone-value,
  .boundary,
  .capital,
  .capital-crown {
    left: var(--pos); /* physical-ok: see the note above */
    translate: -50% 0;
  }
  .zone {
    left: var(--from); /* physical-ok: see the note above */
    width: calc(var(--to) - var(--from));
  }
  .capital {
    width: calc(100% / 19);
  }
  /* The ghost pawn stays readable over a capital or a token. */
  .pawn-ghost {
    background: color-mix(in oklch, var(--panel) 85%, transparent);
  }
</style>
