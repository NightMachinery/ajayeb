<script lang="ts">
  // A real catalog card in a chain's linked preview, without further links.
  import { ht } from '../i18n/messages.svelte';
  import CardArt from './CardArt.svelte';
  import CostPips from './CostPips.svelte';
  import EffectGlyph from './EffectGlyph.svelte';
  import { design } from './model';

  let { slug }: { slug: string } = $props();
  const card = $derived(design(slug));
</script>

{#if card}
  <div class="preview" data-hg-card-preview={slug}>
    <CostPips cost={card.cost} />
    <div class="effects">
      {#each card.effects as effect, i (i)}<EffectGlyph {effect} />{/each}
    </div>
    <div class="art"><CardArt {slug} colour={card.colour} /></div>
    <span>{ht(`hg.card.${slug}` as 'hg.card.forum')}</span>
  </div>
{/if}

<style>
  .preview {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.4rem;
    border: 1px solid var(--rule-strong);
    border-radius: 0.4rem;
    background: var(--card-paper);
    color: var(--card-ink);
  }
  .effects {
    display: flex;
    flex-wrap: wrap;
    gap: 0.2rem;
  }
  .art {
    position: relative;
    height: 4rem;
    overflow: hidden;
    border-radius: 0.2rem;
  }
</style>
