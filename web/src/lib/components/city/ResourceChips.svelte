<script lang="ts">
  // The five resource chips of a player strip or city panel: how many of
  // each this seat produces for free (production.fixed) and what it pays
  // per unit to buy one now (tradePrices), both straight from the view's
  // PlayerView -- no arithmetic of our own. The count comes first, by the
  // resource's glyph; the price follows as a small gold coin with the
  // amount on it, the way cards print coin costs, so the two numbers never
  // read alike. A small choice glyph marks a resource at least one flexible
  // producer can also make.
  //
  // The recap (price-changed-tick, M10.F3): on the viewer's own strip or
  // panel, a trade price the other side's moves changed since the viewer's
  // last move carries a static up or down caret on the chip's bottom-end
  // corner until that seat's move ends, with the old and new price in its
  // tooltip (data-recap-tick). From `recap`, or the game screen's context
  // (view/recap.ts recapBoard) when these are that seat's prices.
  import ChevronDown from '@lucide/svelte/icons/chevron-down';
  import ChevronUp from '@lucide/svelte/icons/chevron-up';
  import { formatNumber, t } from '$lib/i18n';
  import Coin from '$lib/components/card/Coin.svelte';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import type { ProductionView, Resource, ResourceCounts } from '$lib/wire';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { getRecapBoard, type RecapChip } from '$lib/view/recap';

  interface Props {
    production: ProductionView;
    tradePrices: ResourceCounts;
    compact?: boolean;
    /** The recap's carets by resource; else from the game screen's context. */
    recap?: ReadonlyMap<Resource, RecapChip>;
    class?: string;
  }

  let { production, tradePrices, compact = false, recap, class: className = '' }: Props = $props();

  const recapBoard = getRecapBoard();
  // The context's carets belong to one seat: they show on the chips of the
  // strip or panel that draws that seat's own prices (the same object).
  const carets = $derived.by((): ReadonlyMap<Resource, RecapChip> | undefined => {
    if (recap) return recap;
    const board = recapBoard();
    return board?.marks && board.prices === tradePrices ? board.chips : undefined;
  });

  const RESOURCES = [
    'wood',
    'clay',
    'stone',
    'glass',
    'papyrus'
  ] as const satisfies readonly Resource[];

  function labelKey(resource: Resource) {
    switch (resource) {
      case 'wood':
        return 'board.resource.wood' as const;
      case 'clay':
        return 'board.resource.clay' as const;
      case 'stone':
        return 'board.resource.stone' as const;
      case 'glass':
        return 'board.resource.glass' as const;
      case 'papyrus':
        return 'board.resource.papyrus' as const;
    }
  }

  function hasChoice(resource: Resource): boolean {
    return production.producers.some((p) => p.resources.includes(resource));
  }

  // The glyph sits straight on the panel, so it takes the resource's ink
  // tone (text-res-*-ink), which keeps 3:1 against the page surfaces in both
  // themes; the plain text-res-* fills are for tiles. Tailwind's scanner
  // needs each utility spelled out literally somewhere in the file; a
  // template-interpolated class name such as `text-res-${r}-ink` would not
  // generate anything, so the five are listed once here instead.
  const RESOURCE_TEXT_CLASS: Record<Resource, string> = {
    wood: 'text-res-wood-ink',
    clay: 'text-res-clay-ink',
    stone: 'text-res-stone-ink',
    glass: 'text-res-glass-ink',
    papyrus: 'text-res-papyrus-ink'
  };

  // The chips are 11 or 12 px, so they use the small-grid (-sm) glyphs, as
  // everything drawn at 14 px and below does.
  const RESOURCE_ICON: Record<Resource, IconName> = {
    wood: 'resource-wood-sm',
    clay: 'resource-clay-sm',
    stone: 'resource-stone-sm',
    glass: 'resource-glass-sm',
    papyrus: 'resource-papyrus-sm'
  };
</script>

<ul
  class="resource-chips flex flex-wrap {compact ? 'gap-1.5' : 'gap-2'} {className}"
  aria-label={t('board.citySheet.resources')}
>
  {#each RESOURCES as resource (resource)}
    {@const count = production.fixed[resource]}
    {@const price = tradePrices[resource]}
    {@const resourceName = t(labelKey(resource))}
    {@const label = t('board.resource.chip', {
      resource: resourceName,
      count,
      coins: t('coins.count', { count: price })
    })}
    {@const caret = carets?.get(resource)}
    <li
      class="relative flex items-center rounded-control border border-rule bg-panel {compact
        ? 'h-5 gap-0.5 px-1 text-[11px]'
        : 'gap-1 px-1.5 py-0.5 text-xs'}"
      {@attach tooltip({
        text: label
      })}
      data-resource={resource}
    >
      <Icon name={RESOURCE_ICON[resource]} class={RESOURCE_TEXT_CLASS[resource]} />
      <span class="font-semibold tabular-nums" data-must-fit aria-hidden="true"
        >{formatNumber(count)}</span
      >
      <Coin n={price} class={compact ? 'text-[10px]' : 'text-[11px]'} />
      {#if caret}
        <span
          class="recap-caret absolute -end-1 -bottom-1 grid size-3 place-items-center rounded-full text-(--paper-bright)
            {caret.up ? 'bg-cinnabar' : 'bg-malachite'}"
          data-recap-tick={caret.up ? 'up' : 'down'}
          {@attach tooltip({ text: caret.text })}
        >
          {#if caret.up}
            <ChevronUp class="size-2.5" strokeWidth={3.5} aria-hidden="true" />
          {:else}
            <ChevronDown class="size-2.5" strokeWidth={3.5} aria-hidden="true" />
          {/if}
          <span class="sr-only">{caret.text}</span>
        </span>
      {/if}
      {#if hasChoice(resource)}
        <span
          data-resource-choice
          class="absolute -end-0.5 -top-1 grid size-3 place-items-center rounded-full bg-gild text-[8px] text-on-accent"
          {@attach tooltip({ text: t('board.resource.choiceHint') })}
          aria-hidden="true"
        >
          <Icon name="choice" />
        </span>
      {/if}
      <span class="sr-only">{label}</span>
    </li>
  {/each}
</ul>
