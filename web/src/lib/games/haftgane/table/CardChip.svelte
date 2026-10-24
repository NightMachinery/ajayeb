<script lang="ts">
  // A card named in a list: its colour's category glyph, its name and, for
  // a card that appears more than once, the count. The tooltip names the
  // card, its colour and what it does, all printed facts from the catalog.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { cardName } from './labels';
  import { effectText } from './effect-text';
  import { design } from './model';

  interface Props {
    slug: string;
    count?: number;
  }

  let { slug, count = 1 }: Props = $props();

  const d = $derived(design(slug));
  const name = $derived(cardName(slug));
  const tip = $derived(
    d
      ? [name, ht(`hg.screen.colourName.${d.colour}`), ...d.effects.map(effectText)].join('. ')
      : name
  );
</script>

<span class="chip" data-card={slug} data-count={count} {@attach tooltip({ text: tip })}>
  {#if d}
    <span class="cat cat-{d.colour}"><HgGlyph name="category-{d.colour}-sm" /></span>
  {/if}
  <span class="name">{name}</span>
  {#if count > 1}
    <span class="count" dir="ltr">×{formatNumber(count)}</span>
  {/if}
</span>

<style>
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    max-width: 100%;
    min-height: 1.75rem;
    padding-inline: 0.25rem 0.5rem;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--panel);
    font-size: 0.8125rem;
  }
  .cat {
    display: inline-grid;
    flex: 0 0 auto;
    place-items: center;
    width: 1.3rem;
    height: 1.3rem;
    border-radius: 9999px;
    background: var(--panel-sunken);
    font-size: 0.9rem;
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
    font-weight: 700;
  }
</style>
