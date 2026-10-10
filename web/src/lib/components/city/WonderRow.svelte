<script lang="ts">
  // A seat's wonders: small status chips for the phone PlayerStrip
  // ("chips"), or a readable column for the tablet/desktop CityPanel and
  // the CitySheet ("list"). Wonder and card names come from the translated
  // 'wonder.name.*' and 'card.name.*' messages (components/card/effect-text).
  import { t } from '$lib/i18n';
  import { cardName, wonderName } from '$lib/components/card/effect-text';
  import Icon from '$lib/icons/Icon.svelte';
  import type { WonderView } from '$lib/wire';

  interface Props {
    wonders: readonly WonderView[];
    variant?: 'chips' | 'list';
    /** When given, each wonder still to build is a button that reports a
     *  tap (the wonder-first flow); the caller decides what it opens. */
    onselect?: (wonder: WonderView) => void;
    class?: string;
  }

  let { wonders, variant = 'chips', onselect, class: className = '' }: Props = $props();

  function actionable(wonder: WonderView): boolean {
    return onselect !== undefined && !wonder.built && !wonder.removed;
  }

  function statusKey(wonder: WonderView) {
    if (wonder.removed) return 'board.wonder.removed' as const;
    if (wonder.built) return 'board.wonder.built' as const;
    return 'board.wonder.unbuilt' as const;
  }
</script>

{#if variant === 'chips'}
  <ul class="wonder-row flex flex-wrap gap-1 {className}" aria-label={t('board.citySheet.wonders')}>
    {#each wonders as wonder (wonder.wonder)}
      {@const name = wonderName(wonder.wonder)}
      {#if actionable(wonder)}
        <li>
          <button
            type="button"
            class="grid size-8 place-items-center rounded-control border border-rule-strong bg-panel text-fg hover:border-accent"
            title="{name} · {t(statusKey(wonder))}"
            aria-label="{name} · {t(statusKey(wonder))}"
            data-wonder={wonder.wonder}
            onclick={() => onselect?.(wonder)}
          >
            <Icon name="wonder" class="text-sm" />
          </button>
        </li>
      {:else}
        <li
          class="grid {onselect
            ? 'size-8'
            : 'size-6'} place-items-center rounded-control border {wonder.built
            ? 'border-gild bg-accent-soft text-accent-soft-fg'
            : wonder.removed
              ? 'border-rule/40 text-fg-muted/40 line-through'
              : 'border-rule text-fg-muted'}"
          title="{name} · {t(statusKey(wonder))}"
        >
          <Icon name="wonder" class={onselect ? 'text-sm' : 'text-xs'} />
        </li>
      {/if}
    {/each}
  </ul>
{:else}
  <ul class="wonder-row flex flex-col gap-1 {className}" aria-label={t('board.citySheet.wonders')}>
    {#each wonders as wonder (wonder.wonder)}
      {@const name = wonderName(wonder.wonder)}
      {#if actionable(wonder)}
        <li>
          <button
            type="button"
            class="flex min-h-11 w-full items-center gap-2 rounded-control border border-rule-strong bg-panel px-2 py-1 text-start text-sm hover:border-accent"
            data-wonder={wonder.wonder}
            onclick={() => onselect?.(wonder)}
          >
            <Icon name="wonder" class="text-fg-muted" />
            <span class="flex-1 truncate" dir="auto">{name}</span>
            <span class="text-xs text-fg-muted">{t(statusKey(wonder))}</span>
          </button>
        </li>
      {:else}
        <li
          class="flex items-center gap-2 rounded-control border border-rule bg-panel px-2 py-1 text-sm
            {wonder.removed ? 'opacity-50' : ''}"
        >
          <Icon name="wonder" class={wonder.built ? 'text-gild' : 'text-fg-muted'} />
          <span class="flex-1 truncate {wonder.removed ? 'line-through' : ''}" dir="auto"
            >{name}</span
          >
          {#if wonder.tucked}
            <span class="text-xs text-fg-muted">
              {t('board.wonder.tuckedUnder', {
                card: cardName(wonder.tucked)
              })}
            </span>
          {/if}
          <span class="text-xs text-fg-muted">{t(statusKey(wonder))}</span>
        </li>
      {/if}
    {/each}
  </ul>
{/if}
