<script lang="ts">
  // A chain symbol. "in" (on the cost row): this card is free for a player
  // whose city shows the symbol; a gold chip. "out" (on the effect band):
  // this card shows the symbol and makes a later card free; an outlined
  // chip. The two differ by fill as well as position, so they read in
  // greyscale. Decorative for assistive tech (the card's own description
  // names the chain); its tooltip names the linked card and stacks a
  // preview of it (ui/chain-links.ts). With the counting hints, it adds the
  // server's chainTargets as status lines: which city shows the symbol, the
  // card it makes free there and where that card is (view/tips.ts
  // chainLines), from the page's game context or `lines`.
  import Icon from '$lib/icons/Icon.svelte';
  import { hintHide } from '$lib/hints/local.svelte';
  import type { Chain } from '$lib/wire';
  import { chainTooltip } from '$lib/ui/chain-links';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { sessionHintScreen } from '$lib/view/hints';
  import { chainLines } from '$lib/view/tips';
  import { tryGameContext } from '../choices/room';

  interface Props {
    chain: Chain;
    direction: 'in' | 'out';
    /** The status lines; else from the game context. */
    lines?: readonly string[];
    class?: string;
  }

  let { chain, direction, lines, class: className = '' }: Props = $props();

  const ctx = tryGameContext();
  const shown = $derived(
    lines ??
      chainLines(chain, ctx?.session.view?.players, sessionHintScreen(ctx?.session, hintHide.value))
  );
</script>

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
  {@attach tooltip(chainTooltip(chain, direction, shown))}
>
  <Icon name={`chain-${chain}`} class="text-[length:max(0.95em,12px)]" />
</span>
