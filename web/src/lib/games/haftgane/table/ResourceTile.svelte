<script lang="ts">
  // A resource's tile: its glyph, paper-bright on the resource colour, as
  // Duel's resource tiles draw them. Ore and cloth take Haftgane's own
  // colours and drawings (theme.css). Decorative: the chip or pip around it
  // names the resource in its tooltip.
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    res: string;
    size?: 'xs' | 'sm' | 'md';
  }

  let { res, size = 'sm' }: Props = $props();
</script>

<span class="tile {size}" style:--res="var(--res-{res})" data-res={res}>
  <HgGlyph name="resource-{res}-sm" />
</span>

<style>
  .tile {
    display: inline-grid;
    flex-shrink: 0;
    place-items: center;
    border-radius: 0.2rem;
    background: var(--res);
    color: var(--paper-bright);
  }
  .xs {
    width: 0.9rem;
    height: 0.9rem;
    font-size: 0.65rem;
  }
  .sm {
    width: 1.1rem;
    height: 1.1rem;
    font-size: 0.8rem;
  }
  .md {
    width: 1.4rem;
    height: 1.4rem;
    font-size: 1rem;
  }
</style>
