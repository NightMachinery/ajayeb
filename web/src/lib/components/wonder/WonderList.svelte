<script lang="ts">
  // A seat's wonders as a list of rows (WonderTile size "row"): the wonder
  // sheet's body and the desktop city panel's Wonders section, so both show
  // the same thing. A wonder still to build is a button when `onselect` is
  // given (the wonder-first flow: it opens that wonder's own sheet), and it
  // carries data-wonder, the driver's hook. `buildable` comes from
  // view/wonders.buildableWonders, the view's own choices.
  //
  // Board hints (M10.HS4): each row shows what building it would do now,
  // the hourglass and the ready mark (view/board-hints.ts seatWonderHints),
  // given as `hints` or read from the page's game context.
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import { seatWonderHints, wonderOwner, type WonderHint } from '$lib/view/board-hints';
  import { describeWonder } from '$lib/view/describe';
  import { sessionHintScreen } from '$lib/view/hints';
  import { outcomeText } from '$lib/view/outcome';
  import type { WonderView } from '$lib/wire';
  import { tryGameContext } from '../choices/room';
  import WonderTile from './WonderTile.svelte';

  interface Props {
    wonders: readonly WonderView[];
    /** Wonders a choice of the view can build now. */
    buildable?: ReadonlySet<string>;
    /** When given, each wonder still to build is a button reporting the tap. */
    onselect?: (wonder: WonderView) => void;
    /** The board hints by wonder slug; else from the game context. */
    hints?: ReadonlyMap<string, WonderHint>;
    class?: string;
  }

  let { wonders, buildable, onselect, hints, class: className = '' }: Props = $props();

  const ctx = tryGameContext();
  const wh = $derived.by((): ReadonlyMap<string, WonderHint> | undefined => {
    if (hints) return hints;
    const view = ctx?.session.view;
    const screen = sessionHintScreen(ctx?.session, hintHide.value);
    const first = wonders[0];
    if (!view || !screen || !first) return undefined;
    const seat = wonderOwner(view, first.wonder);
    return seat === undefined ? undefined : seatWonderHints(view, seat, screen, buildable);
  });

  function actionable(wonder: WonderView): boolean {
    return onselect !== undefined && !wonder.built && !wonder.removed;
  }

  function label(wonder: WonderView): string {
    const can = buildable?.has(wonder.wonder) ?? false;
    const h = wh?.get(wonder.wonder);
    return [
      describeWonder(wonder.wonder, wonder),
      can ? t('wonder.canBuild') : '',
      h ? outcomeText(h.terms) : '',
      h?.note ?? '',
      h?.hourglass?.tip ?? '',
      h?.ready?.tip ?? ''
    ]
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
          <WonderTile
            slug={wonder.wonder}
            size="row"
            state={wonder}
            buildable={can}
            hint={wh?.get(wonder.wonder)}
          />
        </button>
      </li>
    {:else}
      <li
        class="flex items-center rounded-control border border-rule bg-panel p-1.5"
        data-wonder-item={wonder.wonder}
      >
        <WonderTile
          slug={wonder.wonder}
          size="row"
          state={wonder}
          buildable={can}
          hint={wh?.get(wonder.wonder)}
        />
        <span class="sr-only">{label(wonder)}</span>
      </li>
    {/if}
  {/each}
</ul>
