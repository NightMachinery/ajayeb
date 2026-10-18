<script lang="ts">
  // One of the player's wonders (the wonder-first flow, UX design section
  // 5): the wonder with its effects, the server's cost for it now, and
  // "Build this wonder", which starts the pick mode in which the next card
  // tapped goes under it. The button sends nothing; the move itself is the
  // card's wonder:<wonder>:<card> choice, confirmed in the card sheet. When
  // the wonder cannot be built now the sheet says why, from the server's
  // reason. Reached from a seat's wonder list, it has a Back button to it.
  //
  // Board hints (M10.HS4, from the page's game context): what building it
  // would do now (an outcome line with its hourglass and ready mark), how
  // many wonder slots are left once the race for the last one is near, and
  // how many cards are left at the end of an age.
  import ArrowLeft from '@lucide/svelte/icons/arrow-left';
  import { hintHide } from '$lib/hints/local.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { wonderHint, wonderOwner, wonderSheetLines } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { wonderBySlug } from '$lib/catalog';
  import { cardName, effectsText, wonderName } from '$lib/components/card/effect-text';
  import { t } from '$lib/i18n';
  import { affordOf } from '$lib/view/afford';
  import type { Reason, WonderView } from '$lib/wire';
  import CostBreakdown from '../game/CostBreakdown.svelte';
  import { reasonText } from '../game/reasons';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import OutcomeLine from '../game/OutcomeLine.svelte';
  import { tryGameContext } from '../choices/room';
  import Hourglass from '../wonder/Hourglass.svelte';
  import WonderTile from '../wonder/WonderTile.svelte';
  import { WONDER_SHEET_ART_SIZES } from '$lib/art/lookup';

  interface Props {
    open: boolean;
    /** The wonder's slug; nothing shows without one. */
    slug: string | null;
    /** The owner's WonderView (built, removed, cost). */
    state?: WonderView;
    /** Whether the view has any choice that builds this wonder now. */
    buildable: boolean;
    /** The server's reason when it cannot be built, if it gave one. */
    reason?: Reason;
    /** The owner's coins, for the cost lines. */
    coins?: number;
    /** Input is open (presenter idle and the session may act). */
    canAct: boolean;
    onbuild: (slug: string) => void;
    onclose: () => void;
    /** When given, a Back button returns to the wonder list it came from. */
    onback?: () => void;
  }

  let { open, slug, state, buildable, reason, coins, canAct, onbuild, onclose, onback }: Props =
    $props();

  const titleId = 'wonder-sheet-title';
  const wonder = $derived(slug ? wonderBySlug(slug) : undefined);

  const ctx = tryGameContext();
  const screen = $derived(sessionHintScreen(ctx?.session, hintHide.value));
  const hint = $derived.by(() => {
    const view = ctx?.session.view;
    if (!view || !screen || !slug || !state) return undefined;
    const seat = wonderOwner(view, slug);
    return seat === undefined ? undefined : wonderHint(view, state, seat, screen, buildable);
  });
  const sheetLines = $derived.by(() => {
    const view = ctx?.session.view;
    return view && screen ? wonderSheetLines(view, screen.families) : [];
  });
</script>

<Sheet bind:open={() => open, (v) => !v && onclose()} {titleId}>
  {#if slug && wonder}
    <div class="flex flex-col gap-3 px-4 pt-1 pb-2 split:pt-4" data-wonder-sheet={slug}>
      {#if onback}
        <button
          type="button"
          class="-ms-2 flex min-h-11 items-center gap-1 self-start rounded-control px-2 text-sm font-semibold text-fg-muted hover:text-fg"
          data-control="wonder-list-back"
          onclick={onback}
        >
          <ArrowLeft class="size-4 rtl:-scale-x-100" aria-hidden="true" />
          {t('board.wonders.back')}
        </button>
      {/if}
      <h2 id={titleId} class="font-display text-xl text-fg">{wonderName(slug)}</h2>
      <div class="mx-auto w-full max-w-[280px]">
        <WonderTile
          {slug}
          {state}
          focusable
          size="draft"
          artSizes={WONDER_SHEET_ART_SIZES}
          afford={state?.cost && !state.built ? affordOf(state.cost) : undefined}
        />
      </div>
      <p class="text-sm text-fg">{effectsText(wonder.effects)}</p>
      {#if hint}
        <div class="flex items-start gap-2" data-wonder-value>
          {#if hint.hourglass}
            <span
              class="grid size-6 shrink-0 place-items-center rounded-full border text-fg
                {hint.hourglass.solid ? 'border-cinnabar text-cinnabar-ink' : 'border-rule-strong'}"
              data-hint="hourglass"
              {@attach tooltip({ text: hint.hourglass.tip })}
            >
              <Hourglass solid={hint.hourglass.solid} class="size-3.5" />
            </span>
          {/if}
          <div class="flex min-w-0 flex-col gap-1">
            <OutcomeLine terms={hint.terms} />
            {#if hint.note}
              <p class="text-xs text-fg-muted" data-wonder-library>{hint.note}</p>
            {/if}
            {#if hint.hourglass}
              <p class="text-xs text-fg-muted">{hint.hourglass.tip}</p>
            {/if}
            {#if hint.ready}
              <p class="text-xs text-fg-muted" data-hint="ready">{hint.ready.tip}</p>
            {/if}
          </div>
        </div>
      {/if}
      {#each sheetLines as line (line)}
        <p class="text-xs text-fg-muted" data-wonder-sheet-line>{line}</p>
      {/each}

      {#if state?.built}
        <p class="text-sm text-fg-muted">
          {state.tucked
            ? t('wonder.builtWith', { card: cardName(state.tucked) })
            : t('wonder.built')}
        </p>
      {:else if state?.removed}
        <p class="text-sm text-fg-muted">{t('wonder.removed')}</p>
      {:else}
        {#if state?.cost}
          <section class="flex flex-col gap-1.5" aria-labelledby="wonder-sheet-cost">
            <h3
              id="wonder-sheet-cost"
              class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
            >
              {t('game.cost.title')}
            </h3>
            <CostBreakdown cost={state.cost} {coins} />
          </section>
        {/if}
        {#if buildable}
          <Button
            variant="primary"
            full
            disabled={!canAct}
            data-control="wonder-build"
            onclick={() => onbuild(slug)}
          >
            {t('game.wonder.build')}
          </Button>
          <p class="text-xs text-fg-muted">{t('game.wonder.pickHint')}</p>
        {:else}
          <p class="text-sm text-fg-muted" data-reason={reason}>
            {reason ? reasonText(reason, state?.cost) : t('game.wonder.cannot')}
          </p>
        {/if}
      {/if}
    </div>
  {/if}
</Sheet>
