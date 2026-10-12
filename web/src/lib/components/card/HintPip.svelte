<script lang="ts">
  // The one hint pip a card may carry (the hints design, "hint-glyph-slot"):
  // a small round mark that overhangs the card's top-end corner, outside the
  // cost row and the effect band, so a half-covered mini card still shows
  // it. Shape carries the meaning, fill the side:
  //
  //   win   a crown
  //   pair  the progress-token frame with the card's science symbol inside
  //   loot  the coin-loss glyph
  //
  //   solid    the viewer's own fact (gold; the mover's crest on a hot-seat
  //            device)
  //   outline  the opponent's fact, drawn in their crest colour: a deny target
  //   tint     a spectator's view, each seat in its crest tint
  //
  // A second hint adds a hairline dot beside the pip, never a second glyph.
  // The pip's tooltip lists every hint on the card, one glyph per line, in
  // priority order. The pip is aria-hidden: the card's accessible name
  // carries the same words (view/hints.ts pipLabel).
  //
  // A plain leaf: it takes the pip model (view/hints.ts cardPip) and
  // positions itself absolutely; the parent must be positioned.
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { CardPip, PipEntry } from '$lib/view/hints';

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

  function symbolIcon(e: PipEntry): IconName | undefined {
    return e.symbol ? (`science-${e.symbol}-sm` as IconName) : undefined;
  }
</script>

{#snippet crown(cls: string)}
  <svg class={cls} viewBox="0 0 24 24" aria-hidden="true">
    <path
      d="M3 8 L7.5 12 L12 5 L16.5 12 L21 8 L19 18 H5 Z"
      fill="currentColor"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linejoin="round"
    />
    <rect x="5" y="19.5" width="14" height="2" rx="1" fill="currentColor" />
  </svg>
{/snippet}

{#snippet glyph(e: PipEntry, cls: string)}
  {#if e.code === 'win'}
    {@render crown(cls)}
  {:else if e.code === 'pair'}
    <span class="relative inline-grid place-items-center {cls}">
      <Icon name="token-frame" class="col-start-1 row-start-1 size-full" />
      {#if symbolIcon(e)}
        <Icon name={symbolIcon(e)!} class="col-start-1 row-start-1 size-[56%]" />
      {/if}
    </span>
  {:else}
    <Icon name="coin-loss" class={cls} />
  {/if}
{/snippet}

{#snippet lines()}
  <ul class="flex flex-col gap-1 text-start text-[13px] text-fg" data-hint-lines>
    {#each pip.entries as e (e.code + e.seat)}
      <li class="flex items-start gap-1.5" data-hint-line={e.code} data-hint-side={e.side}>
        <span class="hint-dot mt-0.5 shrink-0" data-style={e.style} data-tone={e.tone}>
          {@render glyph(e, 'size-3')}
        </span>
        <span>{e.text}</span>
      </li>
    {/each}
  </ul>
{/snippet}

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
  .hint-pip,
  .hint-dot {
    --pip-crest: var(--gold);
    --pip-edge: var(--gold-deep);
    color: var(--card-ink);
  }
  .hint-dot {
    display: inline-grid;
    place-items: center;
    width: 1.1rem;
    height: 1.1rem;
    border-radius: 999px;
    border: 1.5px solid var(--pip-edge);
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
