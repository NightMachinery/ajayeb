<script lang="ts">
  // One of the player's wonders (the wonder-first flow, UX design section
  // 5): the wonder with its effects, the server's cost for it now, and
  // "Build this wonder", which starts the pick mode in which the next card
  // tapped goes under it. The button sends nothing; the move itself is the
  // card's wonder:<wonder>:<card> choice, confirmed in the card sheet. When
  // the wonder cannot be built now the sheet says why, from the server's
  // reason.
  import { wonderBySlug } from '$lib/catalog';
  import { cardName, effectsText, wonderName } from '$lib/components/card/effect-text';
  import { t } from '$lib/i18n';
  import { affordOf } from '$lib/view/afford';
  import type { Reason, WonderView } from '$lib/wire';
  import CostBreakdown from '../game/CostBreakdown.svelte';
  import { reasonText } from '../game/reasons';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import WonderTile from '../wonder/WonderTile.svelte';

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
  }

  let { open, slug, state, buildable, reason, coins, canAct, onbuild, onclose }: Props = $props();

  const titleId = 'wonder-sheet-title';
  const wonder = $derived(slug ? wonderBySlug(slug) : undefined);
</script>

<Sheet bind:open={() => open, (v) => !v && onclose()} {titleId}>
  {#if slug && wonder}
    <div class="flex flex-col gap-3 px-4 pt-1 pb-2" data-wonder-sheet={slug}>
      <h2 id={titleId} class="font-display text-xl text-fg">{wonderName(slug)}</h2>
      <div class="mx-auto w-full max-w-[280px]">
        <WonderTile
          {slug}
          size="draft"
          afford={state?.cost && !state.built ? affordOf(state.cost) : undefined}
        />
      </div>
      <p class="text-sm text-fg">{effectsText(wonder.effects)}</p>

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
