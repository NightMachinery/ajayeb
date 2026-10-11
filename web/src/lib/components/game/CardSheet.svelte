<script lang="ts">
  // The card sheet (UX design section 5): what a tapped structure slot is,
  // what it costs and what can be done with it. Everything comes from the
  // view: the actions are the view's choices for this card (card-actions.ts
  // only sorts them), the price is the matching choice's CostView, the
  // opponent's price is the slot's cost for the other seat, and a disabled
  // choice shows the server's reason. Each action button carries its key as
  // data-act.
  //
  // Steps: the card itself, then (Build a wonder) the wonders it could go
  // under. In wonder-first mode the sheet instead asks to confirm placing
  // this card under the chosen wonder.
  //
  // A covered face-up card cannot be taken yet, but the cards and both
  // cities are public, so the server sends both seats' costs for it as if
  // it were uncovered now (SlotView.costs): the sheet says what each player
  // would pay, with the server's totals only.
  import { cardBySlug } from '$lib/catalog';
  import {
    backName,
    cardName,
    chainName,
    effectsText,
    wonderName
  } from '$lib/components/card/effect-text';
  import { t } from '$lib/i18n';
  import type { SlotView, View } from '$lib/wire';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import CardBack from '../card/CardBack.svelte';
  import CardFace from '../card/CardFace.svelte';
  import ActionButton from './ActionButton.svelte';
  import { cardActions, wonderChoicesByCard } from './card-actions';
  import CostBreakdown from './CostBreakdown.svelte';
  import { reasonText } from './reasons';
  import WonderStep from './WonderStep.svelte';

  interface Props {
    open: boolean;
    view: View;
    slot: SlotView | undefined;
    /** The seat whose costs and coins the sheet shows (the seat to act). */
    seat: number | undefined;
    /** The other seat's display name, for its price. */
    opponentName: string;
    /** The sheet's own seat by name rather than "you" (a hot-seat device,
     *  where both players sit, or a spectator). */
    ownName?: string;
    /** The view offers this client moves now (its turn, in a mode it plays). */
    myTurn: boolean;
    /** Input is open: presenter idle and the session may act. */
    canAct: boolean;
    /** Ask for a second tap before sending (online rooms by default). */
    confirm: boolean;
    /** The connection is up; while it is down the buttons say they wait for it. */
    connected?: boolean;
    /** This connection watches (an online spectator): nothing to play. */
    spectator?: boolean;
    step: 'card' | 'wonders';
    /** Wonder-first mode: the wonder waiting for a card. */
    pickFor: string | null;
    onact: (key: string) => void;
    onwonders: () => void;
    onback: () => void;
    onclose: () => void;
  }

  let {
    open,
    view,
    slot,
    seat,
    opponentName,
    ownName,
    myTurn,
    canAct,
    confirm,
    connected = true,
    spectator = false,
    step,
    pickFor,
    onact,
    onwonders,
    onback,
    onclose
  }: Props = $props();

  const titleId = 'card-sheet-title';

  const slug = $derived(slot?.state === 'faceUp' ? slot.card : undefined);
  const card = $derived(slug ? cardBySlug(slug) : undefined);
  const actions = $derived(slug ? cardActions(view, slug) : undefined);
  const takeable = $derived(!!slot && slot.state === 'faceUp' && slot.accessible);
  const coins = $derived(seat === 0 || seat === 1 ? view.players[seat].coins : undefined);
  const myCost = $derived(
    actions?.build?.cost ??
      actions?.buildBlocked?.cost ??
      (seat === 0 || seat === 1 ? slot?.costs?.[seat] : undefined)
  );
  const opponentCost = $derived(
    seat === 0 || seat === 1 ? slot?.costs?.[seat === 0 ? 1 : 0] : undefined
  );
  // A covered card's prices for both seats, as if it were uncovered now.
  const coveredCosts = $derived(
    slot && slug && !takeable && (seat === 0 || seat === 1) && slot.costs
      ? { own: slot.costs[seat], other: slot.costs[seat === 0 ? 1 : 0] }
      : undefined
  );
  const pickChoice = $derived(
    pickFor && slug ? wonderChoicesByCard(view, pickFor).get(slug) : undefined
  );
</script>

<Sheet bind:open={() => open, (v) => !v && onclose()} {titleId}>
  {#if slot && slot.state !== 'empty'}
    {#if step === 'wonders' && slug && actions}
      <WonderStep
        card={slug}
        options={actions.wonders}
        wondersBuilt={view.wondersBuilt}
        {coins}
        {canAct}
        {confirm}
        {connected}
        {titleId}
        {onact}
        {onback}
      />
    {:else}
      <div
        class="flex flex-col gap-3 px-4 pt-1 pb-2 split:pt-4"
        data-card-sheet={slug ?? 'face-down'}
      >
        <div class="flex gap-3">
          <div class="h-[216px] w-[140px] shrink-0">
            {#if slug}
              <CardFace {slug} tier="full" width={140} />
            {:else}
              <CardBack back={slot.back ?? 'age1'} class="h-full w-full" />
            {/if}
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-1.5">
            {#if card && slug}
              <h2 id={titleId} class="font-display text-xl leading-tight text-fg">
                {cardName(slug)}
              </h2>
              <p class="text-sm text-fg-muted">
                {t('game.card.meta', {
                  category: t(`card.category.${card.color}`),
                  age: t(`card.ageNumeral.${card.age}`)
                })}
              </p>
              <p class="text-sm text-fg">{effectsText(card.effects)}</p>
              {#if card.chainIn && card.chainFrom}
                <p class="text-xs text-fg-muted">
                  {t('card.chainIn', {
                    symbol: chainName(card.chainIn),
                    card: cardName(card.chainFrom)
                  })}
                </p>
              {/if}
              {#if card.chainOut && card.chainTo}
                <p class="text-xs text-fg-muted">
                  {t('card.chainOut', {
                    symbol: chainName(card.chainOut),
                    card: cardName(card.chainTo)
                  })}
                </p>
              {/if}
            {:else}
              <h2 id={titleId} class="font-display text-xl leading-tight text-fg">
                {t('card.aria.faceDown', { back: backName(slot.back ?? 'age1') })}
              </h2>
              <p class="text-sm text-fg-muted">{t('game.card.faceDown')}</p>
            {/if}
          </div>
        </div>

        {#if slug && !takeable}
          <p class="text-sm text-fg-muted" data-covered={slot.coveredBy.length}>
            {t('game.card.covered', { count: slot.coveredBy.length })}
          </p>
          {#if coveredCosts}
            <section class="flex flex-col gap-1" aria-labelledby="card-sheet-covered-cost">
              <h3
                id="card-sheet-covered-cost"
                class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
              >
                {t('game.card.coveredCostTitle')}
              </h3>
              <p class="text-sm text-fg" dir="auto" data-covered-cost="own">
                {#if ownName !== undefined}
                  {coveredCosts.own.affordable
                    ? t('game.card.opponentPays', { name: ownName, count: coveredCosts.own.total })
                    : t('game.card.opponentShort', {
                        name: ownName,
                        count: coveredCosts.own.total
                      })}
                {:else}
                  {coveredCosts.own.affordable
                    ? t('game.card.youPay', { count: coveredCosts.own.total })
                    : t('game.card.youShort', { count: coveredCosts.own.total })}
                {/if}
              </p>
              <p class="text-sm text-fg" dir="auto" data-covered-cost="other">
                {coveredCosts.other.affordable
                  ? t('game.card.opponentPays', {
                      name: opponentName,
                      count: coveredCosts.other.total
                    })
                  : t('game.card.opponentShort', {
                      name: opponentName,
                      count: coveredCosts.other.total
                    })}
              </p>
            </section>
          {/if}
        {/if}

        {#if takeable && slug && actions}
          {#if pickFor && pickChoice}
            <section class="flex flex-col gap-2" data-wonder-pick={pickFor}>
              <h3 class="font-display text-lg text-fg">
                {t('game.pick.confirm', { card: cardName(slug), wonder: wonderName(pickFor) })}
              </h3>
              <p class="text-sm text-fg-muted">{t('game.pick.effectLost')}</p>
              {#if pickChoice.cost}
                <CostBreakdown cost={pickChoice.cost} {coins} />
              {/if}
              <ActionButton
                key={pickChoice.key}
                label={t('game.action.buildWonder', {
                  wonder: wonderName(pickFor),
                  count: pickChoice.cost?.total ?? 0
                })}
                {confirm}
                {connected}
                disabled={!canAct}
                {onact}
              />
            </section>
          {:else}
            {#if pickFor}
              <p class="text-sm font-semibold text-fg">
                {t('game.pick.notThisCard', { card: cardName(slug), wonder: wonderName(pickFor) })}
              </p>
            {/if}

            {#if myCost}
              <section class="flex flex-col gap-1.5" aria-labelledby="card-sheet-cost">
                <h3
                  id="card-sheet-cost"
                  class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
                >
                  {t('game.cost.title')}
                </h3>
                <CostBreakdown cost={myCost} coins={myTurn ? coins : undefined} />
              </section>
            {/if}
            {#if opponentCost}
              <p class="text-sm text-fg-muted" dir="auto">
                {opponentCost.affordable
                  ? t('game.card.opponentPays', {
                      name: opponentName,
                      count: opponentCost.total
                    })
                  : t('game.card.opponentShort', {
                      name: opponentName,
                      count: opponentCost.total
                    })}
              </p>
            {/if}

            {#if spectator}
              <p class="text-sm text-fg-muted" data-watching>{t('online.watching')}</p>
            {:else if !myTurn}
              <p class="text-sm text-fg-muted">{t('game.card.notYourTurn')}</p>
            {:else}
              <div class="flex flex-col gap-2">
                {#if actions.build}
                  <ActionButton
                    key={actions.build.key}
                    label={t('game.action.build', { count: actions.build.cost?.total ?? 0 })}
                    {confirm}
                    {connected}
                    disabled={!canAct}
                    {onact}
                  />
                {:else if actions.buildBlocked}
                  <Button variant="primary" full disabled>
                    {t('game.action.build', { count: actions.buildBlocked.cost?.total ?? 0 })}
                  </Button>
                  <p class="text-sm text-fg-muted" data-reason={actions.buildBlocked.reason}>
                    {reasonText(actions.buildBlocked.reason, actions.buildBlocked.cost)}
                  </p>
                {/if}
                <div class="grid grid-cols-2 gap-2">
                  {#if actions.discard}
                    <ActionButton
                      key={actions.discard.key}
                      label={t('game.action.discard', { count: actions.discard.gain ?? 0 })}
                      variant="secondary"
                      {confirm}
                      {connected}
                      disabled={!canAct}
                      {onact}
                    />
                  {/if}
                  {#if actions.wonders.length > 0}
                    <Button
                      variant="secondary"
                      full
                      class="h-auto! min-h-12 py-2 leading-tight"
                      data-control="wonder-step"
                      onclick={onwonders}
                    >
                      {t('game.action.wonder')}
                    </Button>
                  {/if}
                </div>
                {#if !canAct && connected}
                  <p class="text-sm text-fg-muted">{t('game.card.locked')}</p>
                {/if}
              </div>
            {/if}
          {/if}
        {/if}
      </div>
    {/if}
  {/if}
</Sheet>
