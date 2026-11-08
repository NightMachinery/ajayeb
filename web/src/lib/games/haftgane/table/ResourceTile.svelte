<script lang="ts">
  // A resource's tile: its glyph, paper-bright on the resource colour, as
  // Duel's resource tiles draw them. Ore and cloth take Haftgane's own
  // colours and drawings (theme.css). Each tile owns its tooltip; callers
  // may supply its cost, production or trade context.
  import { ht } from '../i18n/messages.svelte';
  import type { HgRuleSectionId } from '../rules/sections';
  import { hgTooltip } from './tooltip';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    res: string;
    size?: 'xs' | 'sm' | 'md';
    tip?: string;
    lines?: readonly string[];
    rulesId?: HgRuleSectionId;
  }

  let { res, size = 'sm', tip, lines, rulesId = 'resources' }: Props = $props();
  const text = $derived(tip ?? ht(`hg.resource.${res}` as 'hg.resource.wood'));
</script>

<span
  class="tile {size}"
  style:--res="var(--res-{res})"
  data-res={res}
  role="img"
  aria-label={text}
  {@attach hgTooltip({ text, lines, rulesId })}
>
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
