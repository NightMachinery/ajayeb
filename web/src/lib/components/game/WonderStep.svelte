<script lang="ts">
  // The card sheet's second step (card first): every wonder this card could
  // go under, each with the server's cost and either a build button (its
  // wonder:<wonder>:<card> key) or the reason it cannot be built. Plain
  // props: the options come from card-actions.ts, which only sorts the
  // view's own choices.
  import ArrowLeft from '@lucide/svelte/icons/arrow-left';
  import { cardName, wonderName } from '$lib/components/card/effect-text';
  import { t } from '$lib/i18n';
  import { affordOf } from '$lib/view/afford';
  import WonderTile from '../wonder/WonderTile.svelte';
  import ActionButton from './ActionButton.svelte';
  import type { WonderOption } from './card-actions';
  import CostBreakdown from './CostBreakdown.svelte';
  import { reasonText } from './reasons';

  interface Props {
    card: string;
    options: readonly WonderOption[];
    /** Wonders built by both players (View.wondersBuilt). */
    wondersBuilt: number;
    /** The paying seat's coins, for the cost lines. */
    coins?: number;
    canAct: boolean;
    confirm: boolean;
    /** The connection is up; while it is down the buttons say they wait for it. */
    connected?: boolean;
    titleId: string;
    onact: (key: string) => void;
    onback: () => void;
  }

  let {
    card,
    options,
    wondersBuilt,
    coins,
    canAct,
    confirm,
    connected = true,
    titleId,
    onact,
    onback
  }: Props = $props();

  // Legal wonders first, in the order the server listed them.
  const sorted = $derived([
    ...options.filter((o) => o.choice),
    ...options.filter((o) => !o.choice)
  ]);
</script>

<div class="flex flex-col gap-3 px-4 pt-1 pb-2 split:pt-4" data-wonder-step>
  <div class="flex items-center gap-2">
    <button
      type="button"
      class="grid size-11 shrink-0 place-items-center rounded-control text-fg hover:bg-panel-sunken"
      aria-label={t('game.wonders.back')}
      title={t('game.wonders.back')}
      data-control="wonder-back"
      onclick={onback}
    >
      <ArrowLeft class="size-5 rtl:-scale-x-100" aria-hidden="true" />
    </button>
    <h2 id={titleId} class="min-w-0 font-display text-lg leading-tight text-fg">
      {t('game.wonders.title', { card: cardName(card) })}
    </h2>
  </div>

  {#if sorted.length === 0}
    <p class="text-sm text-fg-muted">{t('game.wonders.none')}</p>
  {/if}

  <ul class="flex flex-col gap-3">
    {#each sorted as option (option.key)}
      <li
        class="flex flex-col gap-2 rounded-panel border border-rule p-2"
        data-wonder-option={option.wonder}
      >
        <WonderTile
          slug={option.wonder}
          size="city"
          afford={option.cost ? affordOf(option.cost) : undefined}
        />
        {#if option.cost}
          <details class="text-sm">
            <summary class="min-h-8 cursor-pointer text-fg-muted">{t('game.cost.title')}</summary>
            <CostBreakdown cost={option.cost} {coins} class="pt-1" />
          </details>
        {/if}
        {#if option.choice}
          <ActionButton
            key={option.key}
            label={connected
              ? t('game.action.buildWonder', {
                  wonder: wonderName(option.wonder),
                  count: option.cost?.total ?? 0
                })
              : t('online.waitingConnection')}
            {confirm}
            disabled={!canAct}
            {onact}
          />
        {:else if option.reason}
          <p class="text-sm text-fg-muted">{reasonText(option.reason, option.cost)}</p>
        {/if}
      </li>
    {/each}
  </ul>

  <p class="text-xs text-fg-muted">{t('game.wonders.tuckNote')}</p>
  <p class="text-xs text-fg-muted">{t('game.wonders.sevenNote', { count: wondersBuilt })}</p>
</div>
