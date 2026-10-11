<script lang="ts">
  // A seat's wonders as a list of rows (WonderTile size "row"): the wonder
  // sheet's body and the desktop city panel's Wonders section, so both show
  // the same thing. A wonder still to build is a button when `onselect` is
  // given (the wonder-first flow: it opens that wonder's own sheet), and it
  // carries data-wonder, the driver's hook. `buildable` comes from
  // view/wonders.buildableWonders, the view's own choices.
  import { t } from '$lib/i18n';
  import { describeWonder } from '$lib/view/describe';
  import type { WonderView } from '$lib/wire';
  import WonderTile from './WonderTile.svelte';

  interface Props {
    wonders: readonly WonderView[];
    /** Wonders a choice of the view can build now. */
    buildable?: ReadonlySet<string>;
    /** When given, each wonder still to build is a button reporting the tap. */
    onselect?: (wonder: WonderView) => void;
    class?: string;
  }

  let { wonders, buildable, onselect, class: className = '' }: Props = $props();

  function actionable(wonder: WonderView): boolean {
    return onselect !== undefined && !wonder.built && !wonder.removed;
  }

  function label(wonder: WonderView): string {
    const can = buildable?.has(wonder.wonder) ?? false;
    return [describeWonder(wonder.wonder, wonder), can ? t('wonder.canBuild') : '']
      .filter(Boolean)
      .join('. ');
  }
</script>

<ul class="wonder-list flex flex-col gap-1.5 {className}" aria-label={t('board.citySheet.wonders')}>
  {#each wonders as wonder (wonder.wonder)}
    {@const can = buildable?.has(wonder.wonder) ?? false}
    {#if actionable(wonder)}
      <li>
        <button
          type="button"
          class="flex min-h-11 w-full items-center rounded-control border bg-panel p-1.5 hover:border-accent
            {can ? 'border-accent ring-2 ring-accent/35' : 'border-rule-strong'}"
          aria-label={label(wonder)}
          data-wonder={wonder.wonder}
          data-buildable={can ? '' : undefined}
          onclick={() => onselect?.(wonder)}
        >
          <WonderTile slug={wonder.wonder} size="row" state={wonder} buildable={can} />
        </button>
      </li>
    {:else}
      <li
        class="flex items-center rounded-control border border-rule bg-panel p-1.5"
        data-wonder-item={wonder.wonder}
      >
        <WonderTile slug={wonder.wonder} size="row" state={wonder} buildable={can} />
        <span class="sr-only">{label(wonder)}</span>
      </li>
    {/if}
  {/each}
</ul>
