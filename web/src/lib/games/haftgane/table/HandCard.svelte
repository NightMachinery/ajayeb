<script lang="ts">
  // One card of the viewer's hand, with Duel's card anatomy: the printed
  // cost at the top and its chain symbols at the top end, the colour band
  // with its effects, the picture with the affordability badge, and the
  // name. A tap selects it (the pick list then shows the server's choices
  // for it); a long-press shows its tooltip.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ageLabel, joinPhrases } from '$lib/i18n';
  import { ht } from '../i18n/messages.svelte';
  import AffordBadge from './AffordBadge.svelte';
  import CardArt from './CardArt.svelte';
  import ChainIcon from './ChainIcon.svelte';
  import CostPips from './CostPips.svelte';
  import EffectGlyph from './EffectGlyph.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { design, type Afford } from './model';

  interface Props {
    slug: string;
    afford: Afford | null;
    selected?: boolean;
    /** This card is the seat's pending pick: drafted, or confirmed. */
    picked?: 'draft' | 'confirmed' | null;
    /** The server says the pending pick keeps this card for an extra turn. */
    keeps?: boolean;
    disabled?: boolean;
    onselect?: (slug: string) => void;
    /** Non-actionable enlarged face, retaining every micro-tooltip. */
    preview?: boolean;
    /** A server key when this face belongs to an offered special decision. */
    actKey?: string;
  }

  let {
    slug,
    afford,
    selected = false,
    picked = null,
    keeps = false,
    disabled = false,
    onselect,
    preview = false,
    actKey
  }: Props = $props();

  const d = $derived(design(slug));
  const name = $derived(ht(`hg.card.${slug}` as 'hg.card.forum'));
  const tipText = $derived(
    d
      ? joinPhrases([
          name,
          ht(`hg.screen.colourName.${d.colour}`),
          ht('hg.screen.age', { age: ageLabel(d.age) })
        ])
      : name
  );
</script>

{#snippet face()}
  <span class="top">
    {#if d}
      <span class="cost"><CostPips cost={d.cost} /></span>
      <span class="chains">
        {#if d.chainIn}<ChainIcon chain={d.chainIn} direction="in" />{/if}
        {#each d.chainOut ?? [] as c (c)}<ChainIcon chain={c} direction="out" />{/each}
      </span>
    {/if}
  </span>
  <span class="band">
    {#each d?.effects ?? [] as fx, i (i)}<EffectGlyph effect={fx} />{/each}
  </span>
  <span class="picture">
    <CardArt {slug} colour={d?.colour ?? 'grey'} />
    {#if afford}<span class="afford"><AffordBadge {afford} /></span>{/if}
    {#if picked}
      <span class="pick-tag" data-pick-tag={picked}>
        {#if picked === 'confirmed'}<HgGlyph name="badge-check" />{/if}
        <span>{ht('hg.screen.pick.tag')}</span>
      </span>
    {/if}
    {#if keeps}
      <span class="keep-tag" data-keeps-preview={slug}>
        <HgGlyph name="wonder" />
        <span>{ht('hg.special.kept.preview.tag')}</span>
      </span>
    {/if}
  </span>
  <span class="name">{name}</span>
{/snippet}
{#if preview}
  <div
    class="card preview"
    style:--band="var(--card-{d?.colour ?? 'grey'})"
    data-card={slug}
    role="img"
    aria-label={tipText}
    {@attach tooltip({ text: tipText })}
  >
    {@render face()}
  </div>
{:else}
  <button
    type="button"
    class="card"
    class:selected
    class:picked={picked !== null}
    style:--band="var(--card-{d?.colour ?? 'grey'})"
    data-card={slug}
    data-act={actKey ?? `hand:${slug}`}
    data-picked={picked ?? undefined}
    aria-pressed={selected}
    aria-label={name}
    aria-disabled={disabled}
    onclick={() => {
      if (!disabled) onselect?.(slug);
    }}
    {@attach tooltip({ text: tipText, actionable: true })}
  >
    {@render face()}
  </button>
{/if}

<style>
  .card {
    container-type: inline-size;
    display: grid;
    grid-template-rows: auto auto 1fr auto;
    width: 100%;
    height: 100%;
    min-height: 44px;
    overflow: hidden;
    padding: 0;
    border: 2px solid var(--band);
    border-radius: 0.5rem;
    background: var(--card-paper);
    box-shadow: var(--card-rest-shadow);
    color: var(--card-ink);
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .card.preview,
  .card[aria-disabled='true'] {
    cursor: default;
  }
  .card:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: 2px;
  }
  .selected {
    outline: 3px solid var(--accent);
    outline-offset: 2px;
  }
  .picked {
    box-shadow:
      0 0 0 3px var(--lapis),
      var(--card-raised-shadow);
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.1rem;
    min-height: 15cqi;
    padding: 2cqi 3cqi;
    font-size: max(10px, 11cqi);
  }
  .cost {
    min-width: 0;
  }
  .chains {
    display: inline-flex;
    flex-shrink: 0;
    color: var(--card-ink);
  }
  .band {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    gap: 2cqi;
    min-height: 18cqi;
    padding: 2cqi;
    background: var(--band);
    font-size: max(10px, 11cqi);
  }
  .picture {
    position: relative;
    display: grid;
    min-height: 0;
  }
  .picture > :global([data-card-art]) {
    grid-area: 1 / 1;
    min-height: 0;
    font-size: max(12px, 14cqi);
  }
  .afford {
    z-index: 1;
    grid-area: 1 / 1;
    align-self: end;
    justify-self: center;
    margin-block-end: 3cqi;
    font-size: max(10px, 12cqi);
  }
  .keep-tag {
    z-index: 1;
    display: inline-flex;
    grid-area: 1 / 1;
    align-items: center;
    /* Where the pick tag sits: the kept card is never the picked one, and
       the affordability badge holds the bottom. */
    align-self: start;
    justify-self: center;
    gap: 0.2em;
    margin-block-start: 3cqi;
    padding: 0.15em 0.45em;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--panel);
    color: var(--fg);
    font-size: max(10px, 10cqi);
    font-weight: 700;
    white-space: nowrap;
  }
  .pick-tag {
    z-index: 1;
    display: inline-flex;
    grid-area: 1 / 1;
    align-items: center;
    align-self: start;
    justify-self: center;
    gap: 0.2em;
    margin-block-start: 3cqi;
    padding: 0.15em 0.45em;
    border-radius: 9999px;
    background: var(--lapis);
    color: var(--paper-bright);
    font-size: max(10px, 10cqi);
    font-weight: 700;
    white-space: nowrap;
  }
  .name {
    overflow: hidden;
    padding: 2cqi 4cqi 3cqi;
    font-family: var(--type-display);
    font-size: max(10px, 12.5cqi);
    font-weight: 600;
    line-height: 1.15;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
