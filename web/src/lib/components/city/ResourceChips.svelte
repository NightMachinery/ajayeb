<script lang="ts">
  // The five resource chips of a player strip or city panel: how many of
  // each this seat produces for free (production.fixed) and what it pays
  // per unit to buy one now (tradePrices), both straight from the view's
  // PlayerView -- no arithmetic of our own. A small choice glyph marks a
  // resource at least one flexible producer can also make.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import type { ProductionView, Resource, ResourceCounts } from '$lib/wire';

  interface Props {
    production: ProductionView;
    tradePrices: ResourceCounts;
    compact?: boolean;
    class?: string;
  }

  let { production, tradePrices, compact = false, class: className = '' }: Props = $props();

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
  class="resource-chips flex flex-wrap {compact ? 'gap-1' : 'gap-1.5'} {className}"
  aria-label={t('board.citySheet.resources')}
>
  {#each RESOURCES as resource (resource)}
    {@const count = production.fixed[resource]}
    {@const price = tradePrices[resource]}
    {@const resourceName = t(labelKey(resource))}
    <li
      class="relative flex items-center gap-0.5 rounded-control border border-rule bg-panel px-1.5 py-0.5 {compact
        ? 'text-[11px]'
        : 'text-xs'}"
      title={t('board.resource.chip', { resource: resourceName, count, price })}
    >
      <Icon name={RESOURCE_ICON[resource]} class={RESOURCE_TEXT_CLASS[resource]} />
      <span class="font-semibold tabular-nums" data-must-fit>{count}</span>
      <span class="text-fg-muted" aria-hidden="true">·</span>
      <span class="text-fg-muted tabular-nums" data-must-fit>{price}</span>
      {#if hasChoice(resource)}
        <span
          class="absolute -end-1 -top-1 grid size-3 place-items-center rounded-full bg-gild text-[8px] text-on-accent"
          title={t('board.resource.choiceHint')}
          aria-hidden="true"
        >
          <Icon name="choice" />
        </span>
      {/if}
      <span class="sr-only"
        >{t('board.resource.chip', { resource: resourceName, count, price })}</span
      >
    </li>
  {/each}
</ul>
