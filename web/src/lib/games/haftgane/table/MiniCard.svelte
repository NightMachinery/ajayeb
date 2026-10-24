<script lang="ts">
  // A card named in the Seen hands notebook: its colour band with the
  // category glyph (so colour is never the only cue), its name and, when it
  // stands for several, a count. The card you played from a hand carries
  // what you did with it. The tooltip says what the card does.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { effectText } from './effect-text';
  import { cardName } from './labels';
  import { design } from './model';

  interface Props {
    slug: string;
    count?: number;
    /** What you did with it, already translated ("Built"). */
    mark?: string;
    markTip?: string;
    /** Outside the notebook's filter: drawn faint. */
    dim?: boolean;
  }

  let { slug, count = 1, mark = '', markTip = '', dim = false }: Props = $props();
  const d = $derived(design(slug));
  const tip = $derived(
    ht('hg.seen.card.tip', {
      card: cardName(slug),
      effect: (d?.effects ?? []).map(effectText).join(' ')
    })
  );
</script>

<span
  class="mini"
  class:dim
  class:played={!!mark}
  style:--ink="var(--card-{d?.colour ?? 'brown'})"
  data-card={slug}
  data-played={mark ? '' : undefined}
  {@attach tooltip({ text: tip, lines: markTip ? [markTip] : [] })}
>
  <span class="band"><HgGlyph name="category-{d?.colour ?? 'brown'}-sm" /></span>
  <span class="name">{cardName(slug)}</span>
  {#if count > 1}<span class="count" dir="ltr">×{formatNumber(count)}</span>{/if}
  {#if mark}<span class="mark">{mark}</span>{/if}
</span>

<style>
  .mini {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    max-width: 100%;
    min-width: 0;
    padding-block: 0.15rem;
    padding-inline: 0.15rem 0.45rem;
    border: 1px solid var(--rule-strong);
    border-radius: 0.45rem;
    background: var(--panel);
    font-size: 0.85rem;
    line-height: 1.2;
  }
  .played {
    border: 2px solid var(--ink);
    font-weight: 600;
  }
  .dim {
    opacity: 0.35;
  }
  .band {
    display: grid;
    flex: 0 0 1.35rem;
    place-items: center;
    height: 1.35rem;
    border-radius: 0.3rem;
    background: var(--ink);
    color: var(--paper-bright);
    font-size: 0.85rem;
  }
  .name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .count {
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }
  .mark {
    padding: 0.05rem 0.35rem;
    border-radius: 9999px;
    background: var(--ink);
    color: var(--paper-bright);
    font-size: 0.7rem;
    font-weight: 700;
    white-space: nowrap;
  }
</style>
