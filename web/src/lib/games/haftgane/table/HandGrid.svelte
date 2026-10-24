<script lang="ts">
  // The viewer's hand: four columns and two rows on a phone in portrait
  // and on a desktop (capped at 38rem), one row of seven on a short
  // landscape screen. Card sizes come from the space the grid has, by
  // container query units, so the hand never scrolls and never overflows.
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
    locked?: boolean;
    onselect?: (slug: string) => void;
  }

  let { hand, choices, disabled, selected, picked, locked = false, onselect }: Props = $props();
</script>

<div class="hand" data-hand>
  <div class="grid">
    {#each hand as slug, i (i)}
      <HandCard
        {slug}
        afford={afford(cardChoices(slug, choices, disabled))}
        selected={selected === slug}
        picked={picked?.design === slug ? (picked.confirmed ? 'confirmed' : 'draft') : null}
        disabled={locked}
        {onselect}
      />
    {/each}
  </div>
</div>

<style>
  .hand {
    container-type: size;
    flex: 1 1 auto;
    min-height: 0;
  }
  .grid {
    --gap: 0.4rem;
    --cw: min(calc((100cqw - 3 * var(--gap)) / 4), calc((100cqh - var(--gap)) / 2 * 0.66), 9.5rem);
    display: grid;
    grid-template-columns: repeat(4, var(--cw));
    grid-auto-rows: calc(var(--cw) / 0.66);
    justify-content: center;
    align-content: start;
    gap: var(--gap);
  }
  @media (orientation: landscape) and (max-height: 32rem) {
    .grid {
      --cw: min(calc((100cqw - 6 * var(--gap)) / 7), calc(100cqh * 0.66));
      grid-template-columns: repeat(7, var(--cw));
    }
  }
</style>
