<script lang="ts">
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HandCard from './HandCard.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import type { Afford } from './model';
  interface Props {
    card: string;
    locked: boolean;
    afford: Afford | null;
    onselect: (card: string) => void;
  }
  let { card, locked, afford, onselect }: Props = $props();
</script>

<div class="kept" data-kept-card={card}>
  <div class="face"><HandCard slug={card} {afford} disabled={locked} {onselect} /></div>
  <p>
    <span
      role="img"
      aria-label={ht('hg.special.kept.tip')}
      {@attach tooltip({ text: ht('hg.special.kept.tip') })}
      ><HgGlyph name={locked ? 'badge-lock' : 'wonder'} /></span
    >{ht(locked ? 'hg.special.kept.locked' : 'hg.special.kept.play')}
  </p>
</div>

<style>
  .kept {
    position: relative;
    width: 8rem;
    max-width: 100%;
    margin: auto;
  }
  .face {
    height: 10rem;
  }
  p {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.3rem;
    margin: -2rem -0.3rem 0;
    padding: 0.5rem 0.3rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font-size: 0.8rem;
    font-weight: 600;
    line-height: 1.25;
    text-align: center;
    pointer-events: none;
  }
  p span {
    pointer-events: auto;
  }
  @media (orientation: landscape) and (max-height: 32rem) {
    .face {
      height: 7rem;
    }
    .kept {
      width: 6rem;
    }
  }
</style>
