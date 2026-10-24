<script lang="ts">
  // The one hint pip a card may carry (the hints design, "hint-glyph-slot"):
  // a small round mark that overhangs the card's top-end corner, outside the
  // cost row and the effect band, so a half-covered mini card still shows
  // it. Shape carries the meaning, fill the side:
  //
  //   win   a crown (the hint-crown icon)
  //   pair  the progress-token rosette with an equals sign (hint-pair): the
  //         card matches a science symbol, and its tooltip line names it
  //   loot  the coin-loss glyph
  //
  //   solid    the viewer's own fact (gold; the mover's crest on a hot-seat
  //            device)
  //   outline  the opponent's fact, drawn in their crest colour: a deny target
  //   tint     a spectator's view, each seat in its crest tint
  //
  // A second hint adds a hairline dot beside the pip, never a second glyph.
  // The pip's tooltip lists every hint on the card, one glyph per line, in
  // priority order (HintPipLines.svelte, which the card sheet shows too).
  // The pip is aria-hidden: the card's accessible name carries the same
  // words (view/hints.ts pipLabel).
  //
  // A plain leaf: it takes the pip model (view/hints.ts cardPip) and
  // positions itself absolutely; the parent must be positioned.
  import Icon from '$lib/icons/Icon.svelte';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { CardPip, PipEntry } from '$lib/view/hints';
  import HintPipLines from './HintPipLines.svelte';

  interface Props {
    pip: CardPip;
    /** The card's name, for the tooltip's heading. */
    card: string;
    /** sm for the mini tier (the phone structure), md for larger cards. */
    size?: 'sm' | 'md';
    /** Half strength: a covered card, or own pips while tucking (deny mode). */
    faint?: boolean;
    /** Positioning classes; by default the top-end corner, overhanging. */
    class?: string;
  }

  let {
    pip,
    card,
    size = 'md',
    faint = false,
    class: className = 'absolute -end-1.5 -top-1.5'
  }: Props = $props();
</script>

{#snippet glyph(e: PipEntry, cls: string)}
  {#if e.code === 'win'}
    <Icon name="hint-crown" class={cls} />
  {:else if e.code === 'pair'}
    <Icon name="hint-pair" class={cls} />
  {:else}
    <Icon name="coin-loss" class={cls} />
  {/if}
{/snippet}

{#snippet lines()}<HintPipLines {pip} />{/snippet}

<span
  class="hint-pip pointer-events-auto z-10 grid place-items-center rounded-full {size === 'sm'
    ? 'size-4'
    : 'size-5'} {className}"
  class:opacity-55={faint}
  data-hint={pip.top.code}
  data-hint-side={pip.top.side}
  data-hint-seat={pip.top.seat}
  data-style={pip.top.style}
  data-tone={pip.top.tone}
  data-hint-more={pip.more ? '' : undefined}
  aria-hidden="true"
  {@attach tooltip({ text: t('hints.pip.title', { card }), diagram: lines })}
>
  {@render glyph(pip.top, size === 'sm' ? 'size-3' : 'size-4')}
  {#if pip.more}
    <span class="more absolute -start-1 -bottom-0.5 size-[5px] rounded-full" aria-hidden="true"
    ></span>
  {/if}
</span>

<style>
  /* Fill carries the side. The colours sit on cards, which stay paper-
     coloured in both themes, so they are fixed. */
  .hint-pip {
    --pip-crest: var(--gold);
    --pip-edge: var(--gold-deep);
    color: var(--card-ink);
  }
  [data-tone='sun'] {
    --pip-crest: var(--sun);
    --pip-edge: color-mix(in oklch, var(--sun) 70%, var(--card-ink));
  }
  [data-tone='moon'] {
    --pip-crest: var(--moon);
    --pip-edge: color-mix(in oklch, var(--moon) 70%, var(--card-ink));
  }
  .hint-pip {
    border: 1.5px solid var(--pip-edge);
    box-shadow: 0 1px 2px oklch(0.2 0.04 260 / 0.45);
  }
  [data-style='solid'] {
    background: var(--pip-crest);
  }
  /* Outlined: the opponent's fact, paper inside a thick ring of their crest. */
  [data-style='outline'] {
    background: var(--card-paper);
    border-width: 2px;
    border-color: var(--pip-crest);
    color: color-mix(in oklch, var(--pip-crest) 55%, var(--card-ink));
  }
  /* The Moon's crest fill (a hot-seat device) is dark enough for a paper
     glyph; ink stays on gold and on the lighter Sun crest. */
  .hint-pip[data-style='solid'][data-tone='moon'] {
    color: var(--card-paper);
  }
  /* A spectator: each seat's crest, lightly tinted, never solid or outlined. */
  [data-style='tint'] {
    background: color-mix(in oklch, var(--pip-crest) 40%, var(--card-paper));
  }
  .more {
    background: var(--card-ink);
    box-shadow: 0 0 0 1px var(--card-paper);
  }
</style>
