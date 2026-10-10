<script lang="ts">
  // The horizontal military track: both capitals, the conflict pawn at its
  // live position, the four looting tokens with their taken state, and the
  // printed zone point values (2/5/10, from docs/rules/military-science.md
  // -- decoration only; the live score is always scorePreview from the
  // view, never recomputed here). `compact` is the landscape top-bar
  // variant from the UX design: a slimmer bar with smaller tokens, same
  // data and the same aria-label.
  //
  // The board is spatially fixed regardless of reading direction (seat 0's
  // capital is always the start, seat 1's always the end -- see AGENTS.md
  // "the board region is dir=ltr because it is spatial, not text"), so the
  // pawn and the looting tokens are positioned with a physical `left` in
  // the style block below, not a logical inset.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryLeader, militaryMargin, nextLooting } from '$lib/view/threats';
  import type { MilitaryView } from '$lib/wire';
  import LootingToken from './LootingToken.svelte';

  interface Props {
    military: MilitaryView;
    /** This connection's own seat, for "you"/"the opponent" wording. Left
     *  out for a spectator or a gallery demo, which fall back to naming the
     *  crests instead. */
    viewerSeat?: 0 | 1;
    /** The landscape top-bar variant: slimmer, smaller tokens. */
    compact?: boolean;
    class?: string;
  }

  let { military, viewerSeat, compact = false, class: className = '' }: Props = $props();

  // The printed board's zone values by space, 0 (centre) to 8; space 9 is a
  // capital, shown as the crest instead of a number.
  const ZONE_POINTS = [0, 2, 2, 5, 5, 5, 10, 10, 10] as const;

  function crestKey(seat: 0 | 1) {
    return seat === 0 ? ('board.crest.sun' as const) : ('board.crest.moon' as const);
  }

  function pawnPercent(position: number): number {
    return ((position + 9) / 18) * 100;
  }

  const leader = $derived(militaryLeader(military));
  const margin = $derived(militaryMargin(military));
  const next = $derived(nextLooting(military));

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
    const nextSeat = next.seat as 0 | 1;
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

<div
  dir="ltr"
  role="group"
  aria-label={ariaLabel}
  class="military-track flex items-center gap-1.5 {compact
    ? 'min-h-6 text-xs'
    : 'min-h-12 text-sm'} {className}"
>
  <div
    class="grid shrink-0 place-items-center rounded-full border-2 {compact
      ? 'size-5'
      : 'size-7'} {leader === 0
      ? 'border-sun bg-sun/20 text-sun-ink'
      : 'border-rule text-fg-muted'}"
    title={t(viewerSeat === 0 ? 'board.military.capitalYou' : 'board.military.capitalCrest', {
      crest: t('board.crest.sun')
    })}
  >
    <Icon name="crest-sun" />
  </div>

  <div class="track-bar relative {compact ? 'h-2' : 'h-3'} flex-1 rounded-full bg-panel-sunken">
    <div class="track-center absolute inset-y-0" aria-hidden="true"></div>
    {#each military.looting as loot (loot.at)}
      <div
        class="token-spot absolute top-1/2 flex flex-col items-center gap-0.5"
        style:--pos="{pawnPercent(loot.at)}%"
      >
        <LootingToken
          value={loot.value}
          magnitude={Math.abs(loot.at)}
          present={loot.present}
          size={compact ? 'sm' : 'md'}
        />
        {#if !compact}
          <!-- The printed zone value this token's space starts (see
               docs/rules/military-science.md): decoration only, never the
               live score, which is always scorePreview from the view. -->
          <span class="text-[10px] text-fg-muted" data-min-font="10" aria-hidden="true">
            {ZONE_POINTS[Math.abs(loot.at)]}
          </span>
        {/if}
      </div>
    {/each}
    <div
      class="pawn absolute top-1/2 grid place-items-center rounded-full border-2 border-focus bg-panel {compact
        ? 'size-4'
        : 'size-6'}"
      style:--pos="{pawnPercent(military.position)}%"
      aria-hidden="true"
    >
      <Icon name="military-pawn" class={compact ? 'text-[10px]' : 'text-xs'} />
    </div>
  </div>

  <div
    class="grid shrink-0 place-items-center rounded-full border-2 {compact
      ? 'size-5'
      : 'size-7'} {leader === 1
      ? 'border-moon bg-moon/20 text-moon-ink'
      : 'border-rule text-fg-muted'}"
    title={t(viewerSeat === 1 ? 'board.military.capitalYou' : 'board.military.capitalCrest', {
      crest: t('board.crest.moon')
    })}
  >
    <Icon name="crest-moon" />
  </div>
</div>

<style>
  /* physical-ok: the board is fixed LTR regardless of reading direction
     (AGENTS.md), so the pawn and each looting token sit at a physical
     offset along the track, never a logical inset-inline-start. */
  .token-spot,
  .pawn {
    left: var(--pos); /* physical-ok: see the note above */
    translate: -50% -50%;
  }
  .track-center {
    left: 50%; /* physical-ok: see the note above */
    width: 1px;
    background: var(--rule-strong);
  }
</style>
