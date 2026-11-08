<script lang="ts" module>
  // Each of the catalog's 24 chains (named after the design it makes free)
  // drawn by one of the 24 chain drawings of docs/haftgane/art-reuse.md,
  // "Chains: twenty-four symbols". The presenter owns this map; the engine
  // knows chains only as values.
  const DRAWINGS: Record<string, string> = {
    'archery-range': 'chain-target',
    stables: 'chain-horseshoe',
    aqueduct: 'chain-drop',
    courthouse: 'science-scales',
    statue: 'chain-column',
    caravansery: 'chain-camel',
    forum: 'chain-awning',
    dispensary: 'science-mortar',
    laboratory: 'chain-lamp',
    library: 'chain-book',
    circus: 'chain-mask',
    fortifications: 'chain-tower',
    'siege-workshop': 'chain-hammer',
    gardens: 'chain-lyre',
    pantheon: 'chain-torch',
    senate: 'science-quill',
    arena: 'chain-helmet',
    haven: 'chain-barrel',
    lighthouse: 'chain-lighthouse',
    academy: 'chain-scroll',
    lodge: 'chain-saw',
    observatory: 'chain-planet',
    study: 'chain-lightning',
    university: 'chain-star'
  };

  export function chainDrawing(chain: string): string {
    return DRAWINGS[chain] ?? 'badge-chain';
  }
</script>

<script lang="ts">
  // A chain symbol on a card: one it provides (what it makes free later)
  // or the one that makes it free, ringed, with the card names in its
  // tooltip.
  import { hgTooltip } from './tooltip';
  import CardPreview from './CardPreview.svelte';
  import { hgDesignSlugs } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { design } from './model';

  interface Props {
    chain: string;
    direction: 'in' | 'out';
  }

  let { chain, direction }: Props = $props();

  const cardName = (slug: string) => ht(`hg.card.${slug}` as 'hg.card.forum');
  const linked = $derived(
    direction === 'out'
      ? [chain]
      : hgDesignSlugs.filter((s) => design(s)?.chainOut?.includes(chain as never))
  );
  const tip = $derived.by(() => {
    if (direction === 'out') return ht('hg.screen.chain.out', { card: cardName(chain) });
    const from = hgDesignSlugs.filter((s) => design(s)?.chainOut?.includes(chain as never));
    const names = from.map(cardName);
    const card = names.length ? names.reduce((a, b) => ht('hg.screen.trade.or', { a, b })) : '';
    return ht('hg.screen.chain.in', { card });
  });
</script>

<span
  class="chain {direction}"
  data-chain={chain}
  data-chain-dir={direction}
  role="img"
  aria-label={tip}
  {@attach hgTooltip({
    text: tip,
    rulesId: 'costs',
    stack: linked.slice(0, 2).map((slug, i) => ({
      kind: 'custom',
      caption: cardName(slug),
      render: i === 0 ? firstPreview : secondPreview
    }))
  })}
>
  <HgGlyph name={chainDrawing(chain)} />
</span>

{#snippet firstPreview()}<CardPreview slug={linked[0] ?? ''} />{/snippet}
{#snippet secondPreview()}<CardPreview slug={linked[1] ?? ''} />{/snippet}

<style>
  .chain {
    display: inline-grid;
    place-items: center;
    width: 1.25em;
    height: 1.25em;
  }
  .in {
    border: 1px dashed currentColor;
    border-radius: 9999px;
    opacity: 0.8;
  }
</style>
