<script lang="ts">
  // A chain symbol. "in" (on the cost row): this card is free for a player
  // whose city shows the symbol; a gold chip. "out" (on the effect band):
  // this card shows the symbol and makes a later card free; an outlined
  // chip. The two differ by fill as well as position, so they read in
  // greyscale. Decorative for assistive tech (the card's own description
  // names the chain); its tooltip names the linked card and stacks a
  // preview of it (ui/chain-links.ts). Public quotes and whereabouts come
  // explicitly from the displayed view, independently of optional hints.
  import ChainDiagram from '../rules/ChainDiagram.svelte';
  import Icon from '$lib/icons/Icon.svelte';
  import type { Chain } from '$lib/wire';
  import { chainTooltip } from '$lib/ui/chain-links';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { CardTipContext } from '$lib/view/tips';

  interface Props {
    chain: Chain;
    direction: 'in' | 'out';
    /** Optional standalone status lines, superseded by public linked details. */
    lines?: readonly string[];
    context?: CardTipContext;
    class?: string;
  }

  let { chain, direction, lines, context, class: className = '' }: Props = $props();
  const details = $derived(direction === 'out' ? context?.info?.chainOut : context?.info?.chainIn);
</script>

{#snippet chainDiagram()}<ChainDiagram compact />{/snippet}

<!-- The chip is 1.3 em with the glyph at 0.95 em, but never under 14 px
     with a 12 px glyph: on a mini card (a 10 px font) 0.95 em is 9.5 px,
     too small for the mask, lyre, amphora and barrel to read. -->
<span
  class="inline-grid size-[max(1.3em,14px)] shrink-0 place-items-center rounded-[0.35em] border
    {direction === 'in'
    ? 'border-(color:--gold-deep) bg-gold text-card-ink'
    : 'border-(color:--gold-deep) bg-paper text-card-ink'} {className}"
  style:--i2="var(--gold-deep)"
  aria-hidden="true"
  data-chain={chain}
  data-chain-direction={direction}
  {@attach tooltip({
    ...chainTooltip(chain, direction, lines, context),
    diagram: details ? undefined : chainDiagram,
    rulesId: 'costs'
  })}
>
  <Icon name={`chain-${chain}`} class="text-[length:max(0.95em,12px)]" />
</span>
