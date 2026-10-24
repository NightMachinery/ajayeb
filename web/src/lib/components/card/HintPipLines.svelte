<script lang="ts">
  // The lines of a hint pip (HintPip.svelte): every hint on the card, one
  // glyph per line in the pip's own fill, in priority order, with the
  // server's fact beside it. The pip's tooltip shows them, and so does the
  // card sheet (M18.card-sheet-signals), where a phone player already
  // looks, so the two never say different things. A plain leaf: it takes
  // the pip model (view/hints.ts cardPip).
  import Icon from '$lib/icons/Icon.svelte';
  import type { CardPip } from '$lib/view/hints';

  interface Props {
    pip: CardPip;
    class?: string;
  }

  let { pip, class: className = 'text-[13px]' }: Props = $props();
</script>

<ul class="flex flex-col gap-1 text-start text-fg {className}" data-hint-lines>
  {#each pip.entries as e (e.code + e.seat)}
    <li class="flex items-start gap-1.5" data-hint-line={e.code} data-hint-side={e.side}>
      <span class="hint-dot mt-0.5 shrink-0" data-style={e.style} data-tone={e.tone}>
        {#if e.code === 'win'}
          <Icon name="hint-crown" class="size-3" />
        {:else if e.code === 'pair'}
          <Icon name="hint-pair" class="size-3" />
        {:else}
          <Icon name="coin-loss" class="size-3" />
        {/if}
      </span>
      <span>{e.text}</span>
    </li>
  {/each}
</ul>

<style>
  /* The pip's fills (HintPip.svelte), on a dot. They sit on paper in both
     themes, so they are fixed. */
  .hint-dot {
    --pip-crest: var(--gold);
    --pip-edge: var(--gold-deep);
    color: var(--card-ink);
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
  [data-style='solid'] {
    background: var(--pip-crest);
  }
  [data-style='outline'] {
    background: var(--card-paper);
    border-width: 2px;
    border-color: var(--pip-crest);
    color: color-mix(in oklch, var(--pip-crest) 55%, var(--card-ink));
  }
  [data-style='tint'] {
    background: color-mix(in oklch, var(--pip-crest) 40%, var(--card-paper));
  }
</style>
