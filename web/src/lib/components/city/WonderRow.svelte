<script lang="ts">
  // A seat's wonders: small status chips for the phone PlayerStrip
  // ("chips"), or a readable column for the tablet/desktop CityPanel and
  // the CitySheet ("list"). Wonder *names* come straight from the
  // presentation catalog, the same way a player's own display name does
  // (see lib/view/log-format.ts's file comment): they are not retranslated,
  // since 12 wonders is M5.S02/M5.S04 territory, not this slice's.
  import { t } from '$lib/i18n';
  import { cardBySlug, wonderBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import type { WonderView } from '$lib/wire';

  interface Props {
    wonders: readonly WonderView[];
    variant?: 'chips' | 'list';
    class?: string;
  }

  let { wonders, variant = 'chips', class: className = '' }: Props = $props();

  function statusKey(wonder: WonderView) {
    if (wonder.removed) return 'board.wonder.removed' as const;
    if (wonder.built) return 'board.wonder.built' as const;
    return 'board.wonder.unbuilt' as const;
  }
</script>

{#if variant === 'chips'}
  <ul class="wonder-row flex flex-wrap gap-1 {className}" aria-label={t('board.citySheet.wonders')}>
    {#each wonders as wonder (wonder.wonder)}
      {@const name = wonderBySlug(wonder.wonder)?.name ?? wonder.wonder}
      <li
        class="grid size-6 place-items-center rounded-control border {wonder.built
          ? 'border-gild bg-accent-soft text-accent-soft-fg'
          : wonder.removed
            ? 'border-rule/40 text-fg-muted/40 line-through'
            : 'border-rule text-fg-muted'}"
        title="{name} · {t(statusKey(wonder))}"
      >
        <Icon name="wonder" class="text-xs" />
      </li>
    {/each}
  </ul>
{:else}
  <ul class="wonder-row flex flex-col gap-1 {className}" aria-label={t('board.citySheet.wonders')}>
    {#each wonders as wonder (wonder.wonder)}
      {@const name = wonderBySlug(wonder.wonder)?.name ?? wonder.wonder}
      <li
        class="flex items-center gap-2 rounded-control border border-rule bg-panel px-2 py-1 text-sm
          {wonder.removed ? 'opacity-50' : ''}"
      >
        <Icon name="wonder" class={wonder.built ? 'text-gild' : 'text-fg-muted'} />
        <span class="flex-1 truncate {wonder.removed ? 'line-through' : ''}" dir="auto">{name}</span
        >
        {#if wonder.tucked}
          <span class="text-xs text-fg-muted">
            {t('board.wonder.tuckedUnder', {
              card: cardBySlug(wonder.tucked)?.name ?? wonder.tucked
            })}
          </span>
        {/if}
        <span class="text-xs text-fg-muted">{t(statusKey(wonder))}</span>
      </li>
    {/each}
  </ul>
{/if}
