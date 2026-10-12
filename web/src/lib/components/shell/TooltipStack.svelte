<script lang="ts">
  // A tooltip's stack: the panel of linked previews attached to its bubble,
  // such as the card a chain symbol makes free. A plain leaf; the layer
  // positions it and draws the connector between it and the bubble. Card
  // previews are real full-tier card faces, so they read like the card
  // sheet. At most two items fit beside each other at 360 px.
  import CardFace from '$lib/components/card/CardFace.svelte';
  import { i18n } from '$lib/i18n';
  import type { TooltipStackItem } from '$lib/ui/tooltip.svelte';

  interface Props {
    items: readonly TooltipStackItem[];
    class?: string;
  }

  let { items, class: className = '' }: Props = $props();

  const shown = $derived(items.slice(0, 2));
</script>

<div
  dir={i18n.dir}
  class="tooltip-stack flex w-max gap-2 rounded-panel border border-rule-strong bg-panel p-2 shadow-panel {className}"
  data-tooltip-stack
>
  {#each shown as item, i (i)}
    <figure class="flex flex-col items-center gap-1.5" data-tooltip-stack-item={item.kind}>
      <figcaption
        class="max-w-[140px] text-center text-xs leading-tight font-semibold text-fg-muted"
      >
        {item.caption}
      </figcaption>
      {#if item.kind === 'card'}
        <div class="h-[216px] w-[140px]" data-card-preview={item.slug}>
          <CardFace slug={item.slug} tier="full" width={140} />
        </div>
      {:else}
        {@render item.render()}
      {/if}
    </figure>
  {/each}
</div>
