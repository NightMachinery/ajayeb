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
  // this card under the chosen wonder. The Build a wonder button stands out
  // when the view's choices let this card go under a wonder now, and names
  // it (WonderAction.svelte, M18.card-sheet-signals).
  //
  // A covered face-up card cannot be taken yet, but the cards and both
  // cities are public, so the server sends both seats' costs for it as if
  // it were uncovered now (SlotView.costs): the sheet says what each player
  // would pay, with the server's totals only.
  //
  // Hints (M10.HS2): in the scrolling details, the card's sections ("For you",
  // then the opponent's, outlined), and muted notes beside the actions: what
  // discarding it keeps from the opponent and whether a Mausoleum could
  // rebuild it, what building or tucking a deny target does, and a short
  // build a discard would cover. All of it re-shapes the server's
  // slot.hints and costs (view/hints.ts); the game context supplies the
  // families shown, and the gallery passes `hints`.
  //
  // The decider's hints (M10.HS3): under each move button, its outcome
  // line (Choice.outcome: coins and points after, the pawn, a win, the turn
  // order); a build the seat cannot pay for that would win says how short
  // it is; a compact track shows where a pawn move would land; and the
  // "After this pick" panel says what taking the card uncovers and each
  // move's threats one turn ahead (view/outcome.ts). When the move hands
  // the opponent a win or misses one, the confirm step says so; with
  // Confirm moves off there is no confirm step, and the outcome line and
  // the panel carry it alone (owner Q3, A).
  //
  // The tutorial (M10.tutorial-client): a coach step that belongs to this
  // sheet (`coach`, from the game screen) shows as a note in the details
  // above the action area, or above the wonder step's list; the game screen's coach layer
  // rings its target. The rails (`allow`, lib/tutorial/gate.ts): a move
  // button whose key the guided turn does not ask for is disabled, its
  // action group carries data-tutorial-locked, and `lockedReason` says why,
  // once under the buttons (under the one button of a pick). The card itself and the sheet stay open to read.
  import { cardBySlug } from '$lib/catalog';
  import {
    backName,
    cardName,
    chainName,
    effectsText,
    wonderName
  } from '$lib/components/card/effect-text';
  import { ageLabel, t } from '$lib/i18n';
  import DiscardDiagram from '../rules/DiscardDiagram.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { hintHide } from '$lib/hints/local.svelte';
  import {
    cardSections,
    denyNotes,
    discardNotes,
    sessionHintScreen,
    shortByDiscard,
    type HintScreen
  } from '$lib/view/hints';
  import {
    afterPickLines,
    blunderLine,
    outcomeContext,
    outcomeTerms,
    type PickAction
  } from '$lib/view/outcome';
  import type { Choice, SlotView, View } from '$lib/wire';
  import MilitaryTrack from '../board/MilitaryTrack.svelte';
  import { tryGameContext } from '../choices/room';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import CardBack from '../card/CardBack.svelte';
  import CardFace from '../card/CardFace.svelte';
  import ActionButton from './ActionButton.svelte';
  import { cardActions, readyWonders, wonderChoicesByCard } from './card-actions';
  import AfterPickPanel from './AfterPickPanel.svelte';
  import CostBreakdown from './CostBreakdown.svelte';
  import HintLines from './HintLines.svelte';
  import OutcomeLine from './OutcomeLine.svelte';
  import { reasonText } from './reasons';
  import WonderAction from './WonderAction.svelte';
  import WonderStep from './WonderStep.svelte';
  import CoachNote from '../tutorial/CoachNote.svelte';
  import type { CoachNoteView } from '$lib/tutorial/coach.svelte';
  import { ALLOW_ALL, type Allow } from '$lib/tutorial/gate';

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
    /** The hint display of this screen; on a room page it comes from the game context. */
    hints?: HintScreen;
    onact: (key: string) => void;
    onwonders: () => void;
    onback: () => void;
    onclose: () => void;
    /** The tutorial's coach step in this sheet, as a note; null for none. */
    coach?: CoachNoteView | null;
    /** A tutorial room, including free play when no coach step is visible. */
    tutorial?: boolean;
    /** The tutorial's rails: whether a move key may be sent now. */
    allow?: Allow;
    /** Why a move the rails refuse is locked. */
    lockedReason?: string;
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
    hints,
    onact,
    onwonders,
    onback,
    onclose,
    coach = null,
    tutorial = false,
    allow = ALLOW_ALL,
    lockedReason = ''
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
  // CostBreakdown already states this shortfall when it has the paying seat's coins.
  const shortfallShown = $derived(
    actions?.buildBlocked?.reason === 'unaffordable' &&
      myTurn &&
      coins !== undefined &&
      myCost?.affordable === false
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

  const ctx = tryGameContext();
  const screen = $derived(
    view.result ? undefined : (hints ?? sessionHintScreen(ctx?.session, hintHide.value))
  );
  const hintSections = $derived(
    screen && slug && slot
      ? cardSections(slug, slot.hints, screen.perspective, screen.families, screen.names)
      : []
  );
  const sheetSeat = $derived(seat === 1 ? 1 : 0);
  const notes = $derived(
    screen && slot && slug
      ? discardNotes(slot, view, sheetSeat, screen.families, screen.names, {
          mausoleumKnown: !tutorial
        })
      : []
  );
  const deny = $derived(
    screen && slot
      ? denyNotes(slot.hints, sheetSeat, screen.families, screen.names)
      : ({} as ReturnType<typeof denyNotes>)
  );
  // The decider's hints: only on the turn of this sheet's seat, and only
  // when the server sent outcomes (it sends them to the seat to move alone).
  const octx = $derived(
    screen && myTurn && !spectator ? outcomeContext(view, screen.families, screen.names) : undefined
  );
  function termsOf(choice: Choice | undefined) {
    return octx && choice ? outcomeTerms(choice.outcome, octx) : [];
  }
  function blunderOf(choice: Choice | undefined) {
    return octx && confirm && choice ? blunderLine(choice, view.choices, octx) : undefined;
  }
  const pickActions = $derived.by((): PickAction[] => {
    if (!actions) return [];
    const out: PickAction[] = [];
    if (actions.build)
      out.push({ label: t('hints.after.moveBuild'), outcome: actions.build.outcome });
    if (actions.discard) {
      out.push({ label: t('hints.after.moveDiscard'), outcome: actions.discard.outcome });
    }
    for (const w of actions.wonders) {
      if (w.choice) out.push({ label: wonderName(w.wonder), outcome: w.choice.outcome });
    }
    return out;
  });
  const afterLines = $derived(
    octx && slot && takeable && !pickFor
      ? afterPickLines(slot, view.structure, pickActions, octx)
      : []
  );
  // A build that moves the pawn: the track with its ghost.
  const pawnGhost = $derived(
    octx?.families.has('military') ? actions?.build?.outcome?.pawn : undefined
  );
  // A build the seat cannot pay for that would win (DisabledChoice.wins).
  const shortWin = $derived.by(() => {
    const d = actions?.buildBlocked;
    if (!octx || !octx.families.has('threats') || !d?.wins || !d.cost) return undefined;
    return t('hints.outcome.shortWin', {
      count: d.cost.shortfall,
      kind: t(`hints.victory.${d.wins.kind}`)
    });
  });

  const shortNote = $derived(
    screen ? shortByDiscard(actions?.buildBlocked?.discardToAfford, screen.families) : undefined
  );
</script>

{#snippet discardDiagram()}<DiscardDiagram compact />{/snippet}

{#snippet locked(key: string)}
  {#if !allow(key)}
    <p class="text-xs text-fg-muted" data-tutorial-reason>{lockedReason}</p>
  {/if}
{/snippet}

{#snippet coachNote()}
  {#if coach}
    <CoachNote
      title={coach.title}
      progress={coach.progress}
      text={coach.text}
      step={coach.step}
      kind={coach.kind}
      textId={coach.textId}
      onpause={coach.onpause}
    />
  {/if}
{/snippet}

{#snippet footer()}
  {#if takeable && slug && actions && step === 'card'}
    {#if pickFor && pickChoice}
      <div
        class="flex flex-col gap-1"
        data-act-group
        data-wonder-pick-action={pickFor}
        data-tutorial-locked={allow(pickChoice.key) ? undefined : ''}
      >
        <ActionButton
          key={pickChoice.key}
          label={t('game.action.buildWonder', {
            wonder: wonderName(pickFor),
            count: pickChoice.cost?.total ?? 0
          })}
          {confirm}
          {connected}
          disabled={!canAct || !allow(pickChoice.key)}
          {onact}
        />
        {@render locked(pickChoice.key)}
        <OutcomeLine
          terms={termsOf(pickChoice)}
          actKey={pickChoice.key}
          blunder={blunderOf(pickChoice)}
        />
      </div>
    {:else if myTurn && !spectator}
      <div
        class="flex flex-col gap-2 short:grid short:grid-cols-3 short:items-start"
        data-card-actions
      >
        {#if actions.build}
          <div
            class="flex flex-col gap-1"
            data-act-group
            data-tutorial-locked={allow(actions.build.key) ? undefined : ''}
          >
            <ActionButton
              key={actions.build.key}
              label={t('game.action.build', { count: actions.build.cost?.total ?? 0 })}
              {confirm}
              {connected}
              disabled={!canAct || !allow(actions.build.key)}
              {onact}
            />
            <OutcomeLine
              terms={termsOf(actions.build)}
              actKey={actions.build.key}
              blunder={blunderOf(actions.build)}
            />
          </div>
        {:else if actions.buildBlocked}
          <Button variant="primary" full disabled>
            {t('game.action.build', { count: actions.buildBlocked.cost?.total ?? 0 })}
          </Button>
        {/if}
        <div class="grid grid-cols-2 items-start gap-2 short:contents">
          {#if actions.discard}
            <div
              class="flex flex-col gap-1"
              data-act-group
              data-tutorial-locked={allow(actions.discard.key) ? undefined : ''}
              {@attach tooltip({
                text: t('game.action.discard', { count: actions.discard.gain ?? 0 }),
                rulesId: 'turn',
                diagram: discardDiagram,
                actionable: canAct && connected && allow(actions.discard.key)
              })}
            >
              <ActionButton
                key={actions.discard.key}
                label={t('game.action.discard', { count: actions.discard.gain ?? 0 })}
                variant="secondary"
                {confirm}
                {connected}
                disabled={!canAct || !allow(actions.discard.key)}
                {onact}
              />
              <OutcomeLine
                terms={termsOf(actions.discard)}
                actKey={actions.discard.key}
                blunder={blunderOf(actions.discard)}
              />
            </div>
          {/if}
          {#if actions.wonders.length > 0}
            <WonderAction ready={readyWonders(actions)} onclick={onwonders} />
          {/if}
        </div>
      </div>
    {/if}
  {/if}
{/snippet}

<Sheet
  bind:open={() => open, (v) => !v && onclose()}
  {titleId}
  class="card-sheet"
  panelData={{
    'data-card-sheet':
      slot && slot.state !== 'empty' && step !== 'wonders' ? (slug ?? 'face-down') : undefined
  }}
  footer={takeable &&
  slug &&
  actions &&
  step === 'card' &&
  ((pickFor && pickChoice) || (myTurn && !spectator))
    ? footer
    : undefined}
>
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
        {coach}
        {allow}
        {lockedReason}
      />
    {:else}
      <div class="card-details flex flex-col gap-3 px-4 pt-1 pb-2 split:pt-4">
        <div class="card-overview">
          <div class="card-preview h-[216px] w-[140px] shrink-0">
            {#if slug}
              <CardFace
                {slug}
                tier="full"
                width={140}
                cost={myCost}
                tipContext={{
                  actualCost: actions?.build?.cost,
                  costSeat: seat,
                  info: view.cardTooltips.find((c) => c.card === slug),
                  names: seat === 1 ? [opponentName, ownName ?? ''] : [ownName ?? '', opponentName],
                  you: ownName === undefined ? view.viewer : undefined
                }}
                focusable
              />
            {:else}
              <CardBack focusable back={slot.back ?? 'age1'} class="h-full w-full" />
            {/if}
          </div>
          <div class="card-heading flex min-w-0 flex-col gap-1.5">
            {#if card && slug}
              <h2 id={titleId} class="font-display text-xl leading-tight text-fg">
                {cardName(slug)}
              </h2>
              <p class="text-sm text-fg-muted">
                {t('game.card.meta', {
                  category: t(`card.category.${card.color}`),
                  age: ageLabel(card.age)
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
          {#if takeable && slug && actions && !(pickFor && pickChoice)}
            {#if myCost}
              <section
                class="card-cost flex flex-col gap-1.5"
                aria-labelledby="card-sheet-cost"
                data-reason={shortfallShown ? actions?.buildBlocked?.reason : undefined}
              >
                <h3
                  id="card-sheet-cost"
                  class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
                >
                  {t('game.cost.title')}
                </h3>
                <CostBreakdown cost={myCost} coins={myTurn ? coins : undefined} />
              </section>
            {/if}
          {/if}
        </div>

        {#if !takeable || !slug}
          {@render coachNote()}
        {/if}
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
              <p class="text-sm text-fg" data-covered-cost="own">
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
              <p class="text-sm text-fg" data-covered-cost="other">
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
          <HintLines sections={hintSections} />
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
            </section>
          {:else}
            {#if pickFor}
              <p class="text-sm font-semibold text-fg">
                {t('game.pick.notThisCard', { card: cardName(slug), wonder: wonderName(pickFor) })}
              </p>
            {/if}

            {#if opponentCost}
              <p class="text-sm text-fg-muted">
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

            {@render coachNote()}
            {#if spectator}
              <p class="text-sm text-fg-muted" data-watching>{t('online.watching')}</p>
            {:else if !myTurn}
              <p class="text-sm text-fg-muted">{t('game.card.notYourTurn')}</p>
            {:else}
              <div class="flex flex-col gap-2">
                {#if actions.build}
                  {#if pawnGhost}
                    <MilitaryTrack
                      military={view.military}
                      ghost={pawnGhost}
                      leftSeat={screen?.names.you ?? sheetSeat}
                      viewerSeat={screen?.names.you ?? undefined}
                      compact
                      class="mt-0.5"
                    />
                  {/if}
                  {#if deny.build}
                    <p class="text-xs text-fg-muted" data-hint-note="deny-build">{deny.build}</p>
                  {/if}
                {:else if actions.buildBlocked}
                  {#if !shortfallShown}
                    <p class="text-sm text-fg-muted" data-reason={actions.buildBlocked.reason}>
                      {reasonText(actions.buildBlocked.reason, actions.buildBlocked.cost)}
                    </p>
                  {/if}
                  {#if shortNote}
                    <p class="text-sm text-fg-muted" data-hint-note="short-by-discard">
                      {shortNote}
                    </p>
                  {/if}
                  {#if shortWin}
                    <p class="text-sm font-semibold text-fg" data-hint-note="short-win">
                      {shortWin}
                    </p>
                  {/if}
                {/if}
                {#if notes.length > 0 && actions.discard}
                  <ul class="flex flex-col gap-0.5" data-hint-notes="discard">
                    {#each notes as note (note.id)}
                      <li class="text-xs text-fg-muted" data-hint-note={note.id}>
                        {note.text}
                      </li>
                    {/each}
                  </ul>
                {/if}
                {#if deny.wonder && actions.wonders.length > 0}
                  <p class="text-xs text-fg-muted" data-hint-note="deny-tuck">
                    {deny.wonder}
                  </p>
                {/if}
                {#if !canAct && connected}
                  <p class="text-sm text-fg-muted">{t('game.card.locked')}</p>
                {:else if canAct && ((actions.build && !allow(actions.build.key)) || (actions.discard && !allow(actions.discard.key)))}
                  <p class="text-xs text-fg-muted" data-tutorial-reason>
                    {lockedReason}
                  </p>
                {/if}
              </div>
            {/if}
            <AfterPickPanel lines={afterLines} />
            <HintLines sections={hintSections} />
          {/if}
        {/if}
      </div>
    {/if}
  {/if}
</Sheet>

<style>
  .card-overview {
    display: grid;
    grid-template-columns: 140px minmax(0, 1fr);
    align-items: start;
    gap: 0.75rem;
  }
  .card-cost {
    grid-column: 1 / -1;
  }
  @media (min-width: 64rem) {
    :global(.card-sheet) {
      --sheet-max-height: min(calc(100dvh - 2rem), 760px);
    }
  }
  @media (orientation: landscape) and (max-height: 32rem) {
    :global(.sheet.card-sheet) {
      --sheet-max-height: calc(100dvh - 1rem);
      width: min(680px, calc(100% - 2rem));
    }
    .card-details {
      padding-block-start: 0.25rem;
    }
    .card-cost :global([data-cost-breakdown]) {
      font-size: 0.75rem;
      line-height: 1rem;
    }
    .card-overview {
      column-gap: 0.75rem;
      row-gap: 0.5rem;
      grid-template-rows: min-content 1fr;
    }
    .card-preview {
      grid-column: 1;
      grid-row: 1 / span 2;
    }
    .card-heading {
      gap: 0.25rem;
    }
    .card-heading p {
      font-size: 0.75rem;
      line-height: 1rem;
    }
    .card-cost {
      grid-column: 2;
    }
  }
</style>
