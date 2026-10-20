<script lang="ts">
  // Pending choices and the completed-age dialog. The parent retains the
  // boundary snapshot independently of the starter choice, so every viewer
  // keeps the same totals until explicitly closing it. The final opponent
  // summary precedes the first opening. The new age is dealt before the
  // dialog opens (the Presenter holds that frame until its deal ends), and
  // a starter decision keeps the open summary as it is.
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
  import { restoreFocus } from '$lib/ui/dismiss';
  import type { Attachment } from 'svelte/attachments';
  import ChevronDown from '@lucide/svelte/icons/chevron-down';
  import { ageLabel, t } from '$lib/i18n';
  import type { CardTipContext } from '$lib/view/tips';
  import { settings } from '$lib/ui/settings.svelte';
  import type { AgeSummary } from '$lib/ui/age-summary.svelte';
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
    ageSummary?: AgeSummary;
    /** Existing holder seat in pass-device hot-seat; otherwise shared device. */
    ageViewer?: number | null;
    showAge?: boolean;
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
    ageSummary,
    ageViewer = null,
    showAge = true,
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
  const events = $derived(eventsProp ?? ctx?.presenter.shownEvents ?? []);
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

  // The actual free rebuild payment must belong to the frame being shown,
  // including standalone gallery scenes, not a possibly newer Session view.
  const cardTips = $derived(
    new Map<string, CardTipContext>(
      options.flatMap((choice) =>
        choice.kind === 'revive' && choice.card && choice.cost
          ? [
              [
                choice.card,
                {
                  actualCost: choice.cost,
                  costSeat: pending?.seat,
                  names,
                  you,
                  info: view.cardTooltips.find((row) => row.card === choice.card)
                }
              ] as const
            ]
          : []
      )
    )
  );

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

  // Gallery callers without a session may still render a pending summary.
  let ageMinimized = $state(false);
  const completed = $derived(
    ageSummary
      ? ageSummary.viewFor(ageViewer)
      : pending?.kind === 'chooseStarter' || events.some((event) => event.kind === 'starterChosen')
        ? view
        : null
  );
  const ageReady = $derived(!!completed && view.age >= completed.age && view.ply >= completed.ply);
  const ageOpen = $derived(
    ageReady &&
      (ageSummary?.isShown(ageViewer) || showAge) &&
      !(ageSummary?.isDismissed(ageViewer) ?? ageMinimized)
  );
  // The new age is already under way (the other side chose and took a
  // card before this recap opened): its button says Continue, not Begin.
  const ageBegun = $derived(
    !!completed &&
      completed.age === view.age &&
      view.structure.some((slot) => slot.state === 'empty')
  );
  const choosingStarter = $derived(
    pending?.kind === 'chooseStarter' && completed?.age === view.age
  );
  const startingSeat = $derived(asSeat(ageSummary?.starterFor(ageViewer) ?? view.active));
  const startsText = $derived(
    you === startingSeat
      ? t('choices.age.startsYou', { age: ageLabel(view.age) })
      : t('choices.age.starts', { name: names[startingSeat], age: ageLabel(view.age) })
  );

  function setAgeOpen(open: boolean): void {
    if (ageSummary) {
      if (open) ageSummary.reopen(ageViewer);
      else {
        const previous = ageSummary.viewFor(ageViewer);
        ageSummary.close(ageViewer);
        // Continue can reveal the next queued recap in the same dialog.
        // With no intervening popup and visible/idle presentation, it is
        // already on screen; keep it stable through later incoming frames.
        if (previous !== ageSummary.viewFor(ageViewer) && showAge) ageSummary.markShown(ageViewer);
      }
    } else ageMinimized = !open;
    if (!open) {
      requestAnimationFrame(() => {
        // Native close may retain focus inside the closed dialog until
        // its exit completes. That element is no longer a usable opener.
        const focused = document.activeElement;
        if (
          !document.querySelector('dialog:modal') &&
          (focused === document.body || focused?.closest('dialog:not([open])'))
        ) {
          const menu = document.querySelector<HTMLButtonElement>(
            '.game-screen [data-control="menu"]'
          );
          if (menu) restoreFocus(menu);
        }
      });
    }
  }

  // Replacing the chosen starter button must leave keyboard focus inside
  // the same dialog, on its explicit way forward.
  function continueFocus(choosing: boolean): Attachment<HTMLButtonElement> {
    return (button) => {
      if (choosing) return;
      const frame = requestAnimationFrame(() => {
        const dialog = button.closest('dialog');
        if (dialog?.open && !dialog.contains(document.activeElement)) restoreFocus(button);
      });
      return () => cancelAnimationFrame(frame);
    };
  }

  function act(key: string): void {
    onact(key);
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
        return t('choices.starter.title', { age: ageLabel(view.age) });
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
        return t('choices.waiting.chooseStarter', { name, age: ageLabel(view.age) });
    }
  }

  const titleId = 'choice-panel-title';
</script>

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

{#if ageReady && completed && (choosingStarter || !(ageSummary?.isDismissed(ageViewer) ?? ageMinimized))}
  <div
    onfocusin={() => ageSummary?.markShown(ageViewer)}
    data-age-start={!choosingStarter ? completed.age : undefined}
    data-choice-host={choosingStarter ? 'chooseStarter' : undefined}
    data-decider={choosingStarter ? decider : undefined}
    data-mine={choosingStarter ? mine : undefined}
  >
    <Sheet
      bind:open={
        () => ageOpen,
        (open) => {
          if (ageOpen || open) setAgeOpen(open);
        }
      }
      centered
      titleId="age-start-title"
    >
      {#snippet footer()}
        <div class="flex flex-col gap-1">
          {#if choosingStarter && decider !== null && mine}
            <StarterChoice
              part="actions"
              age={view.age}
              chooser={decider}
              {names}
              {you}
              military={completed.military}
              choices={options}
              {canAct}
              {confirm}
              {connected}
              onact={act}
            />
          {/if}
          <!-- While this side chooses a starter, the starter buttons are the
             primary actions and Peek is secondary (outlined); otherwise
             Continue or Begin is the one way forward. -->
          <button
            type="button"
            class="inline-flex min-h-12 w-full items-center justify-center rounded-control px-3 py-2 text-sm font-semibold
            {choosingStarter && mine
              ? 'border border-rule-strong bg-panel text-fg hover:bg-panel-sunken'
              : 'bg-accent text-on-accent hover:brightness-105 active:brightness-95'}"
            data-begin-age={!choosingStarter &&
            view.phase !== 'gameOver' &&
            completed.age === view.age
              ? view.age
              : undefined}
            data-age-continue
            {@attach continueFocus(choosingStarter)}
            onclick={() => setAgeOpen(false)}
          >
            {choosingStarter && mine
              ? t('choices.peek')
              : choosingStarter || view.phase === 'gameOver' || completed.age < view.age || ageBegun
                ? t('choices.age.continue')
                : t('choices.age.begin', { age: ageLabel(view.age) })}
          </button>
        </div>
      {/snippet}
      <div class="flex flex-col gap-3 px-5 pt-4 pb-4">
        <AgeInterstitial
          age={Math.max(1, completed.age - 1)}
          {names}
          players={completed.players}
          scores={completed.scorePreview}
          military={completed.military}
          hintView={completed}
          {you}
          titleId="age-start-title"
        >
          {#if choosingStarter && decider !== null}
            <StarterChoice
              part="details"
              age={view.age}
              chooser={decider}
              {names}
              {you}
              military={completed.military}
              showDiagram={completed.military.position === 0}
              choices={options}
              {canAct}
              {confirm}
              {connected}
              onact={act}
            />
          {:else if view.phase !== 'gameOver' && completed.age === view.age}
            <p class="text-center text-sm font-semibold text-fg" role="status">{startsText}</p>
          {/if}
        </AgeInterstitial>
      </div>
    </Sheet>
    {#if !ageOpen && showAge && (choosingStarter || !ageSummary?.isDismissed(ageViewer))}
      <PeekBar
        inline
        text={choosingStarter && decider !== null
          ? mine
            ? title('chooseStarter')
            : waitingText('chooseStarter', decider)
          : startsText}
        openLabel={t('choices.age.summaryOpen')}
        onopen={() => setAgeOpen(true)}
      />
    {/if}
  </div>
{:else if pending && decider !== null}
  <div data-choice-host={pending.kind} data-decider={decider} data-mine={mine}>
    {#if mine}
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
                tipContexts={cardTips}
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
{/if}
