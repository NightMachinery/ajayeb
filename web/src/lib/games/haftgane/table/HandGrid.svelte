<script lang="ts">
  // The viewer's hand: four columns and up to two rows on a phone in
  // portrait and on a desktop (capped at 38rem), one row of up to seven on
  // a short landscape screen. Card sizes come from the space the grid has,
  // by container query units, so the hand never scrolls and never
  // overflows. The hand asks the screen for room for its rows at a usable
  // card (4.5rem wide, its tags readable) in portrait and on a desktop,
  // and at a 44 px card on a short landscape screen.
  import type { Choice, Disabled } from '$lib/wire/haftgane.gen';
  import HandCard from './HandCard.svelte';
  import { afford, cardChoices } from './model';

  interface Props {
    hand: readonly string[];
    choices: readonly Choice[];
    disabled: readonly Disabled[];
    selected: string | null;
    /** The design of the pending pick and whether it is confirmed. */
    picked: { design: string; confirmed: boolean } | null;
    /** The design the pending pick keeps for an extra turn (its `kept`). */
    keeps?: string | null;
    locked?: boolean;
    onselect?: (slug: string) => void;
  }

  let {
    hand,
    choices,
    disabled,
    selected,
    picked,
    keeps = null,
    locked = false,
    onselect
  }: Props = $props();
  // The card that stays: with two copies of one design, the one not picked.
  const keepAt = $derived.by(() => {
    if (!keeps) return -1;
    const first = hand.indexOf(keeps);
    return keeps === picked?.design ? hand.indexOf(keeps, first + 1) : first;
  });
  // The card the pick plays: with two copies of one design, one tag only.
  const pickAt = $derived(hand.findIndex((slug, i) => slug === picked?.design && i !== keepAt));
</script>

<div
  class="hand"
  data-hand
  style:--hand-rows={Math.max(1, Math.ceil(hand.length / 4))}
  style:--hand-cols={Math.max(1, Math.min(hand.length, 7))}
>
  <div class="grid">
    {#each hand as slug, i (i)}
      <HandCard
        {slug}
        afford={afford(cardChoices(slug, choices, disabled))}
        selected={selected === slug}
        picked={picked && i === pickAt ? (picked.confirmed ? 'confirmed' : 'draft') : null}
        keeps={i === keepAt}
        disabled={locked}
        {onselect}
      />
    {/each}
  </div>
</div>

<style>
  .hand {
    --gap: 0.4rem;
    --rows: var(--hand-rows);
    --min-cw: 4.5rem;
    container-type: size;
    flex: 1 1 auto;
    min-height: calc(var(--rows) * var(--min-cw) / 0.66 + (var(--rows) - 1) * var(--gap));
  }
  /* The width decides first, so the grid never overflows sideways; then
     the height, but never below the usable card the hand made room for. */
  .grid {
    --cw: min(
      calc((100cqw - 3 * var(--gap)) / 4),
      9.5rem,
      max(var(--min-cw), calc((100cqh - (var(--rows) - 1) * var(--gap)) / var(--rows) * 0.66))
    );
    display: grid;
    grid-template-columns: repeat(4, var(--cw));
    grid-auto-rows: calc(var(--cw) / 0.66);
    justify-content: center;
    align-content: start;
    gap: var(--gap);
  }
  @media (orientation: landscape) and (max-height: 32rem) {
    .hand {
      --rows: 1;
      --min-cw: 2.75rem;
    }
    .grid {
      --cw: min(
        calc((100cqw - (var(--hand-cols) - 1) * var(--gap)) / var(--hand-cols)),
        max(var(--min-cw), calc(100cqh * 0.66))
      );
      grid-template-columns: repeat(var(--hand-cols), var(--cw));
    }
  }
</style>
