<script lang="ts">
  // The host for every pending choice a card, wonder or age end opens:
  // destroying an opponent's card, building from the discard pile, taking a
  // progress token from the board or the Great Library offer, and choosing
  // who starts the next age (with the end-of-age summary).
  //
  // For the player who decides, the panel is a sheet (a centred dialog for
  // the starter choice) that can be minimized to peek at the board; a small
  // bar then brings it back. Everyone else gets a one-line bar saying what
  // is being chosen, and for the starter choice the end-of-age summary with
  // a waiting note. Once a new age has started from a choice this client
  // watched someone else make (the bot or the online opponent), a small bar
  // says who starts, with the summary one tap away, until it is dismissed
  // or the first card is taken.
  //
  // It decides nothing: the options are the view's choices, every action
  // button carries the choice's own key in data-act, and whether input is
  // open is the canAct prop. Every commit button follows the card sheet's
  // act-button rule (game/act-button.ts): the confirm step when the room
  // asks for it, and "Waiting for connection" while the connection is down.
  //
  // Its bars sit in the flow, just above the turn bar (GameScreen mounts it
  // there), never floating over the board: they push the structure up a
  // little rather than cover the takeable cards or this player's own strip.
  import ChevronDown from '@lucide/svelte/icons/chevron-down';
  import { t } from '$lib/i18n';
  import { settings } from '$lib/ui/settings.svelte';
  import type { Choice, GameEvent, PendingKind, View } from '$lib/wire';
  import { needsConfirm } from '../game/card-actions';
  import Sheet from '../shell/Sheet.svelte';
  import AgeInterstitial from '../screens/AgeInterstitial.svelte';
  import DestroyChoice from './DestroyChoice.svelte';
  import DiscardPickChoice from './DiscardPickChoice.svelte';
  import PeekBar from './PeekBar.svelte';
  import StarterChoice from './StarterChoice.svelte';
  import TokenChoice from './TokenChoice.svelte';
  import { asSeat, isYou, otherSeat, seatNames, tryGameContext, type SeatNames } from './room';

  interface Props {
    view: View;
    /** Whether this client may act now (presenter idle and session.canAct). */
    canAct: boolean;
    /** Sends one of the view's choice keys. */
    onact: (key: string) => void;
    /** Display names by seat; read from the room when not given. */
    names?: SeatNames;
    /** The events of the action that produced view; read from the frame on screen when not given. */
    events?: readonly GameEvent[];
    /** Ask for a second tap before sending; from the room's mode and the
     *  settings when not given. */
    confirm?: boolean;
    /** The connection is up; from the page's session when not given. */
    connected?: boolean;
  }

  let {
    view,
    canAct,
    onact,
    names: namesProp,
    events: eventsProp,
    confirm: confirmProp,
    connected: connectedProp
  }: Props = $props();

  const ctx = tryGameContext();
  const names = $derived(namesProp ?? seatNames(ctx?.presenter.frame?.room));
  const mode = $derived(ctx?.presenter.frame?.room.mode);
  const events = $derived(eventsProp ?? ctx?.presenter.frame?.events ?? []);
  const confirm = $derived(
    confirmProp ?? (mode ? needsConfirm(mode, settings.value.confirmMoves) : false)
  );
  const connected = $derived(connectedProp ?? (ctx ? ctx.session.conn === 'open' : true));
  const you = $derived(
    view.viewer !== undefined && isYou(view.viewer, view.viewer, mode)
      ? asSeat(view.viewer)
      : undefined
  );

  const pending = $derived(view.pending);
  const decider = $derived(pending ? asSeat(pending.seat) : null);

  const CHOICE_KIND: Record<PendingKind, Choice['kind']> = {
    destroyCard: 'destroy',
    buildFromDiscard: 'revive',
    pickBoardToken: 'token',
    pickLibraryToken: 'token',
    chooseStarter: 'starter'
  };

  const options = $derived(
    pending ? view.choices.filter((c) => c.kind === CHOICE_KIND[pending.kind]) : []
  );
  // The view only lists choices for the seat to move, so this client
  // decides exactly when the pending choice comes with options.
  const mine = $derived(options.length > 0);

  // Slug to key, in the view's order.
  const cardKeys = $derived(
    new Map(options.flatMap((c) => (c.card === undefined ? [] : [[c.card, c.key] as const])))
  );
  const tokenKeys = $derived(
    new Map(options.flatMap((c) => (c.token === undefined ? [] : [[c.token, c.key] as const])))
  );

  // One id per open choice, so a minimized panel stays minimized only for
  // the choice it was minimized on, and each panel's selection starts fresh.
  const pendingId = $derived(pending ? `${view.ply}:${pending.kind}:${pending.seat}` : null);
  let minimizedFor = $state<string | null>(null);
  const sheetOpen = $derived(pendingId !== null && minimizedFor !== pendingId);

  function setOpen(open: boolean): void {
    minimizedFor = open ? null : pendingId;
  }

  // The age that just ended: the view has already moved on to the next age
  // when its starter is chosen.
  const endedAge = $derived(Math.max(1, view.age - 1));

  // dismissedAge is the age whose start summary this client has closed, or
  // whose starter it chose itself (so it does not see the summary twice).
  let dismissedAge = $state(0);
  let summaryOpen = $state(false);

  function act(key: string): void {
    if (pending?.kind === 'chooseStarter') dismissedAge = view.age;
    onact(key);
  }

  // Whether the action that produced this view was the other player (the
  // bot, or the online opponent) choosing who starts. In a hot-seat room
  // there is no "you": whoever chose sat at this device.
  const starterChosenElsewhere = $derived(
    you !== undefined && events.some((e) => e.kind === 'starterChosen' && e.seat !== you)
  );

  // The age whose starter this client saw chosen elsewhere, kept across the
  // frames that may follow before the first card (a bot's thinking frame, a
  // resync). A plain variable, not state: the frame that sets it already
  // shows the bar through starterChosenElsewhere, and later frames re-read it.
  // A waiting starter panel counts too, for a frame whose events were lost.
  let watchedStarterAge = 0;
  $effect(() => {
    if (starterChosenElsewhere || (pending?.kind === 'chooseStarter' && !mine)) {
      watchedStarterAge = view.age;
    }
  });

  // A new age whose first card is not taken yet, after a starter chosen
  // elsewhere: a small bar says who starts, with the summary one tap away.
  const freshAge = $derived(
    !pending &&
      view.phase === 'play' &&
      view.structure.length > 0 &&
      view.structure.every((slot) => slot.state !== 'empty') &&
      (starterChosenElsewhere || watchedStarterAge === view.age) &&
      dismissedAge !== view.age
  );

  const startsText = $derived(
    you === view.active
      ? t('choices.age.startsYou', { age: view.age })
      : t('choices.age.starts', { name: names[asSeat(view.active)], age: view.age })
  );

  function beginAge(): void {
    dismissedAge = view.age;
    summaryOpen = false;
  }

  // The who-starts bar has done its job once the player turns to the board:
  // the first tap on the structure closes it, so on a short phone it gives
  // the height it took back to the strips.
  function onBoardTap(event: PointerEvent): void {
    if (!freshAge) return;
    const target = event.target;
    if (target instanceof Element && target.closest('.card-structure')) beginAge();
  }

  function title(kind: PendingKind): string {
    switch (kind) {
      case 'destroyCard':
        return t('choices.destroy.title');
      case 'buildFromDiscard':
        return t('choices.discard.title');
      case 'pickBoardToken':
        return t('choices.token.title');
      case 'pickLibraryToken':
        return t('choices.library.title');
      case 'chooseStarter':
        return t('choices.starter.title', { age: view.age });
    }
  }

  function waitingText(kind: PendingKind, seat: 0 | 1): string {
    const name = names[seat];
    switch (kind) {
      case 'destroyCard':
        return t('choices.waiting.destroyCard', { name });
      case 'buildFromDiscard':
        return t('choices.waiting.buildFromDiscard', { name });
      case 'pickBoardToken':
        return t('choices.waiting.pickBoardToken', { name });
      case 'pickLibraryToken':
        return t('choices.waiting.pickLibraryToken', {
          name,
          count: pending?.hiddenTokens ?? 0
        });
      case 'chooseStarter':
        return t('choices.waiting.chooseStarter', { name, age: view.age });
    }
  }

  const titleId = 'choice-panel-title';
</script>

<svelte:window onpointerdown={onBoardTap} />

{#snippet peekButton()}
  <button
    type="button"
    class="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-control px-2 text-sm font-semibold text-fg-muted
      hover:bg-panel-sunken hover:text-fg"
    onclick={() => setOpen(false)}
    data-peek
  >
    <ChevronDown class="size-4" aria-hidden="true" />
    {t('choices.peek')}
  </button>
{/snippet}

{#if pending && decider !== null}
  <div data-choice-host={pending.kind} data-decider={decider} data-mine={mine}>
    {#if pending.kind === 'chooseStarter'}
      <!-- The end of an age: the summary for everyone, with the choice or a
           waiting note, and always Peek as a way on. -->
      <Sheet bind:open={() => sheetOpen, setOpen} centered {titleId}>
        <div class="flex flex-col gap-2 px-5 pt-4 pb-1">
          <AgeInterstitial
            age={endedAge}
            {names}
            players={view.players}
            scores={view.scorePreview}
            military={view.military}
            {you}
            {titleId}
          >
            <StarterChoice
              age={view.age}
              chooser={decider}
              {names}
              {you}
              military={view.military}
              showDiagram={view.military.position === 0}
              choices={options}
              {canAct}
              {confirm}
              {connected}
              onact={act}
            />
          </AgeInterstitial>
          <div class="flex justify-center">{@render peekButton()}</div>
        </div>
      </Sheet>
      {#if !sheetOpen}
        <PeekBar
          inline
          text={mine ? title(pending.kind) : waitingText(pending.kind, decider)}
          onopen={() => setOpen(true)}
        />
      {/if}
    {:else if mine}
      <Sheet bind:open={() => sheetOpen, setOpen} {titleId}>
        <div class="flex flex-col gap-3 px-4 pt-1">
          <header class="flex items-start justify-between gap-2">
            <h2 id={titleId} class="pt-2 font-display text-xl leading-tight text-fg">
              {title(pending.kind)}
            </h2>
            {@render peekButton()}
          </header>
          {#key pendingId}
            {#if pending.kind === 'destroyCard'}
              <DestroyChoice
                keys={cardKeys}
                color={pending.color}
                opponentName={names[otherSeat(decider)]}
                {canAct}
                {confirm}
                {connected}
                onact={act}
              />
            {:else if pending.kind === 'buildFromDiscard'}
              <DiscardPickChoice
                cards={pending.cards}
                keys={cardKeys}
                {canAct}
                {confirm}
                {connected}
                onact={act}
              />
            {:else}
              <TokenChoice
                keys={tokenKeys}
                library={pending.kind === 'pickLibraryToken'}
                {canAct}
                {confirm}
                {connected}
                onact={act}
              />
            {/if}
          {/key}
        </div>
      </Sheet>
      {#if !sheetOpen}
        <PeekBar inline text={title(pending.kind)} onopen={() => setOpen(true)} />
      {/if}
    {:else}
      <PeekBar inline text={waitingText(pending.kind, decider)} />
    {/if}
  </div>
{:else if freshAge}
  <div data-age-start={view.age}>
    <PeekBar
      inline
      text={startsText}
      openLabel={t('choices.age.summaryOpen')}
      onopen={() => (summaryOpen = true)}
      ondismiss={beginAge}
    />
    <Sheet bind:open={summaryOpen} centered titleId="age-start-title">
      <div class="flex flex-col gap-3 px-5 pt-4 pb-1">
        <AgeInterstitial
          age={endedAge}
          {names}
          players={view.players}
          scores={view.scorePreview}
          military={view.military}
          {you}
          titleId="age-start-title"
        >
          <div class="flex flex-col gap-3">
            <p class="text-center text-sm font-semibold text-fg" dir="auto">{startsText}</p>
            <button
              type="button"
              class="inline-flex h-[48px] w-full items-center justify-center rounded-control bg-accent px-5 text-[15px]
                font-semibold text-on-accent hover:brightness-105 active:brightness-95"
              onclick={beginAge}
              data-begin-age={view.age}
            >
              {t('choices.age.begin', { age: view.age })}
            </button>
          </div>
        </AgeInterstitial>
      </div>
    </Sheet>
  </div>
{/if}
