<script lang="ts">
  // A seat's reveal tag (phone-layout.md section 4): what it did in the
  // turn just revealed, as a small label with a glyph band. A building
  // shows its colour's category glyph and its name; a wonder stage says
  // "Stage II" and a sale "Sold +3", because the card under a stage and a
  // sold card stay face down for everyone but their owner. The tooltip
  // carries the seat's whole recap line. With `pop` it appears with a
  // short pop, which the Animation setting makes instant.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { design } from './model';
  import type { RecapLine } from './recap';
  import { saleOf } from './recap';
  import { tagText } from './recap-text';

  interface Props {
    line: RecapLine;
    /** The seat's recap sentence and coin changes, already translated. */
    tip: string;
    lines?: readonly string[];
    pop?: boolean;
  }

  let { line, tip, lines = [], pop = false }: Props = $props();

  const act = $derived(line.act);
  const card = $derived(
    act && (act.kind === 'build' || act.kind === 'build-discarded') ? act.design : ''
  );
  const glyph = $derived.by(() => {
    switch (act?.kind) {
      case 'build':
        return `category-${design(card)?.colour ?? 'brown'}-sm`;
      case 'build-discarded':
      case 'discard-nothing':
        return 'build-from-discard';
      case 'stage':
        return 'wonder-stage';
      case 'sell':
        return 'coin';
      default:
        return 'wonder';
    }
  });
  const ink = $derived(
    act?.kind === 'build' || act?.kind === 'build-discarded'
      ? `var(--card-${design(card)?.colour ?? 'brown'})`
      : act?.kind === 'sell'
        ? 'var(--card-yellow)'
        : 'var(--fg-muted)'
  );
  const label = $derived(tagText(line, saleOf(line)));
</script>

{#if act}
  <span
    class="tag"
    class:pop
    style:--ink={ink}
    data-reveal-tag={line.seat}
    data-reveal-act={act.kind}
    data-reveal-card={card || undefined}
    {@attach tooltip({ text: tip, lines: [...lines] })}
  >
    <span class="band"><HgGlyph name={glyph} /></span>
    <span class="label" dir="auto">{label}</span>
  </span>
{/if}

<style>
  .tag {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    max-width: 100%;
    min-width: 0;
    padding-block: 0.1rem;
    padding-inline: 0.1rem 0.4rem;
    border: 1.5px solid var(--ink);
    border-radius: 0.45rem;
    background: var(--panel);
    box-shadow: var(--shadow);
    color: var(--fg);
    font: 600 0.75rem/1.2 var(--type-display);
    white-space: nowrap;
  }
  .band {
    display: grid;
    flex: 0 0 1.1rem;
    place-items: center;
    height: 1.1rem;
    border-radius: 0.3rem;
    background: var(--ink);
    color: var(--paper-bright);
    font-size: 0.75rem;
  }
  .label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .pop {
    animation: tag-pop calc(var(--dur-base) * 1.6) cubic-bezier(0.2, 1.4, 0.4, 1) both;
  }
  @keyframes tag-pop {
    from {
      opacity: 0;
      transform: translateY(0.4rem) scale(0.7);
    }
  }
</style>
