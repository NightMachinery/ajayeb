<script lang="ts">
  // The game screen (UX design sections 3, 5 and 6): one board markup whose
  // CSS grid areas switch with the viewport class (lib/layout/breakpoints),
  // with every sheet, the wonder-first pick mode, undo, the pending and slow
  // states and the hot-seat pass-device screen.
  //
  //   phone portrait   top bar / opponent strip / track and tokens /
  //                    structure / my strip / turn bar
  //   phone landscape  top bar with the compact track and tokens, then three
  //                    columns: my strip and the turn bar / structure /
  //                    opponent strip
  //   tablet           as phone portrait, with the city's band stacks
  //   desktop          top bar, then three columns: my city panel / track,
  //                    tokens, structure and turn bar / opponent panel
  //
  // "You" are at the bottom or on the left; a spectator sees seat 0 there,
  // and a hot-seat game either faces the player to move or stays fixed
  // (settings). The board region is dir="ltr" because it is spatial; the
  // text inside the bars and panels follows the locale.
  //
  // It decides nothing about the game. The structure, the panels and the
  // sheets render the presented view; every move is one of the view's choice
  // keys, sent through the Session. Input is open only while
  // presenter.idle && session.canAct. The draft, a pending choice and the end
  // of the game are the Draft, ChoiceHost and GameOver screens, mounted with
  // their fixed props and the seat names (a bot seat is named by its
  // persona).
  //
  // Against the bot (UX design section 5, "AI thinking"): the turn bar and
  // the bot's strip or panel say it is thinking; the Presenter holds its
  // answer for the thinking dwell, then previews it, and the slot it takes
  // (or the wonder it drafts) carries data-bot-pick meanwhile. The military
  // track puts the bottom (or left) seat's capital at its left end, so a
  // player at the Moon sees it mirrored. The top bar's game menu leads Home;
  // a room stays on Home's Continue list until its game ends.
  //
  // Online (UX design section 2, "Online lobby" and "Connection states"):
  // until the second player takes their seat the screen is the Lobby, with
  // the invite; it turns into the draft on its own when they join. The other
  // player's strip shows their presence, a toast says when they leave or
  // come back, and the turn bar says when the player to move is away. While
  // the connection is down the board stays inspectable, the connection pill
  // says so and the move buttons read "Waiting for connection"; the Session
  // sends nothing until it is back. A spectator gets the same board, read
  // only. Online rooms have no undo, and moves take the confirm step the
  // settings ask for.
  import { resolve } from '$app/paths';
  import { onDestroy } from 'svelte';
  import { MediaQuery } from 'svelte/reactivity';
  import { wonderName } from '$lib/components/card/effect-text';
  import { i18n, t } from '$lib/i18n';
  import { breakpoints } from '$lib/layout/breakpoints.svelte';
  import { connMessage } from '$lib/session/messages';
  import type { Presenter } from '$lib/session/presenter.svelte';
  import type { Notice, Session } from '$lib/session/session.svelte';
  import { inviteUrl, presenceChanges, seatPresence } from '$lib/ui/online';
  import { settings } from '$lib/ui/settings.svelte';
  import { GameUi, type Acked, type Seat } from '$lib/ui/ui.svelte';
  import { badgeSeat } from '$lib/view/afford';
  import { playAgainFrom, turnState } from '$lib/view/turn';
  import type { SlotView, State, WonderView } from '$lib/wire';
  import MilitaryTrack from '../board/MilitaryTrack.svelte';
  import CardStructure from '../board/CardStructure.svelte';
  import ChoiceHost from '../choices/ChoiceHost.svelte';
  import ConnectionPill from '../hud/ConnectionPill.svelte';
  import SpectatorCount from '../hud/SpectatorCount.svelte';
  import TopBar from '../hud/TopBar.svelte';
  import TurnBar from '../hud/TurnBar.svelte';
  import Draft from '../screens/Draft.svelte';
  import GameOver from '../screens/GameOver.svelte';
  import Lobby from '../screens/Lobby.svelte';
  import PassDevice from '../screens/PassDevice.svelte';
  import Settings from '../screens/Settings.svelte';
  import CitySheet from '../sheets/CitySheet.svelte';
  import LogSheet from '../sheets/LogSheet.svelte';
  import WonderSheet from '../sheets/WonderSheet.svelte';
  import LiveRegion from '../shell/LiveRegion.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import ToastList from '../shell/Toasts.svelte';
  import { Toasts } from '../shell/toast-queue.svelte';
  import ProgressTokenRow from '../token/ProgressTokenRow.svelte';
  import CardSheet from './CardSheet.svelte';
  import {
    botPick,
    eventLines,
    homeSeat,
    needsConfirm,
    passPending,
    pendingSlot,
    pickableSlots,
    wonderStatus
  } from './card-actions';
  import { botSeat, seatNames } from './names';
  import SeatPanel from './SeatPanel.svelte';

  interface Props {
    session: Session;
    presenter: Presenter;
    /** Leaves the room for the home screen. */
    onhome: () => void;
  }

  let { session, presenter, onhome }: Props = $props();

  const ui = new GameUi();
  const toasts = new Toasts();
  const coarsePointer = new MediaQuery('(pointer: coarse)', false);

  // The request this screen sent last, to tell an act's ack from an undo's.
  let lastRequest: { seq: number; kind: 'act' | 'undo' } | null = null;
  let roomCode: string | null = null;
  // The previous frame of this room, for the presence toasts.
  let lastFrame: State | null = null;
  // The notice the player dismissed; a newer one shows again.
  let dismissedNotice = $state<Notice | null>(null);

  // Settings, with the device defaults of the hot-seat sheet.
  const passDevice = $derived(settings.value.hotseatPassDevice ?? coarsePointer.current);
  const orientation = $derived(
    settings.value.hotseatOrientation ??
      (breakpoints.viewport === 'phone-portrait' || breakpoints.viewport === 'phone-landscape'
        ? 'face'
        : 'fixed')
  );

  const frame = $derived(presenter.frame);
  const view = $derived(presenter.view);
  const mode = $derived(frame?.room.mode ?? 'online');
  const mySeats = $derived(frame?.you.seats ?? []);
  const names = $derived(seatNames(frame?.room));
  const inputOpen = $derived(presenter.idle && session.canAct);
  const myTurn = $derived(!!frame?.you.canAct);
  const confirm = $derived(needsConfirm(mode, settings.value.confirmMoves));
  const connected = $derived(session.conn === 'open');
  const spectator = $derived(!!frame && mySeats.length === 0);
  // The online room still waits for its second player.
  const lobby = $derived(frame?.room.status === 'lobby');
  const presence = $derived(seatPresence(frame?.room));
  const inviteLink = $derived(
    frame ? inviteUrl(location.origin, resolve('/play/[code]', { code: frame.code })) : ''
  );
  const canShare = typeof navigator !== 'undefined' && typeof navigator.share === 'function';
  // Undo stays available while a bot's answer is held on screen: undo()
  // shows it at once and then takes it back with the player's move.
  const canUndo = $derived(mode !== 'online' && session.canUndo);

  const layout = $derived(breakpoints.viewport);
  // Where the toasts sit in play. Above the turn bar by default (on a phone
  // that covers only this player's own strip). On a desktop the turn bar is
  // under the structure, whose bottom row holds the cards a player can take,
  // and in phone landscape the structure fills the middle down to the foot
  // of the screen, so in both they sit at the foot of this player's own
  // column instead, clear of the takeable cards, the track and the opponent.
  const toastSpot = $derived(
    !view || view.phase === 'wonderDraft'
      ? ('turn' as const)
      : layout === 'desktop' || layout === 'phone-landscape'
        ? ('home' as const)
        : ('turn' as const)
  );
  const variant = $derived(
    layout === 'desktop'
      ? ('column' as const)
      : layout === 'tablet'
        ? ('band' as const)
        : ('strip' as const)
  );

  const home = $derived(
    view
      ? homeSeat({ mode, mySeats, toMove: view.toMove, active: view.active, orientation })
      : (0 as Seat)
  );
  const away = $derived((home === 0 ? 1 : 0) as Seat);
  const badge = $derived(view ? badgeSeat(view) : undefined);
  const lit = $derived(
    mode === 'hotseat' && orientation === 'fixed' && view?.toMove !== undefined
      ? view.toMove
      : undefined
  );
  // The bot of an ai room, and the seat thinking now (the turn bar's state).
  const bot = $derived(botSeat(frame?.room));
  // The track speaks of "you" only to a player with one seat.
  const trackSeat = $derived(
    mySeats.length === 1 && mySeats[0] !== undefined ? (mySeats[0] === 1 ? 1 : 0) : undefined
  );

  const cardsLeft = $derived(view ? view.structure.filter((s) => s.state !== 'empty').length : 0);
  const pendingKey = $derived(
    session.pending?.frame.t === 'act' ? session.pending.frame.key : undefined
  );
  const pendingIndex = $derived(pendingSlot(view, pendingKey));
  const marked = $derived(view && ui.pickFor ? pickableSlots(view, ui.pickFor) : undefined);
  const selectedSlot = $derived<SlotView | undefined>(
    view && ui.selected !== null ? view.structure[ui.selected] : undefined
  );

  const turn = $derived(
    turnState({
      view,
      mode,
      mySeats: mySeats.filter((s): s is Seat => s === 0 || s === 1),
      conn: session.conn,
      pending: session.pending !== null,
      slow: session.slow,
      thinking: frame?.thinking === 0 || frame?.thinking === 1 ? frame.thinking : undefined,
      connected: [
        presence[0] === undefined ? undefined : presence[0] === 'connected',
        presence[1] === undefined ? undefined : presence[1] === 'connected'
      ],
      playAgain: playAgainFrom(ui.log)
    })
  );
  const thinkingSeat = $derived(turn.kind === 'aiThinking' ? turn.seat : undefined);
  // During the Presenter's pre-roll: what the bot's answer takes.
  const pick = $derived(botPick(view, presenter.preview, frame?.thinking));

  const showPass = $derived(
    !!view &&
      passPending({
        mode,
        enabled: passDevice,
        phase: view.phase,
        toMove: view.toMove,
        holder: ui.holder
      })
  );
  const lastLines = $derived(frame ? eventLines(frame.events, names) : []);

  // The wonder sheet's wonder, its owner and whether the view can build it now.
  const wonderOwner = $derived.by(() => {
    if (!view || !ui.wonder) return undefined;
    for (const p of view.players) {
      const w = p.wonders.find((x) => x.wonder === ui.wonder);
      if (w) return { seat: p.seat as Seat, state: w, coins: p.coins };
    }
    return undefined;
  });
  const wonderInfo = $derived(
    view && ui.wonder && wonderOwner?.seat === view.toMove
      ? wonderStatus(view, ui.wonder)
      : { buildable: false }
  );

  // The Session's frames, in order: the log, undo bookkeeping, the hot-seat
  // device holder and the "Built ... · Undo" toast. Runs outside any effect.
  function onFrame(f: State): void {
    if (f.code !== roomCode) {
      roomCode = f.code;
      lastFrame = null;
      ui.reset();
      toasts.clear();
    }
    announcePresence(lastFrame, f);
    lastFrame = f;
    let acked: Acked = null;
    if (f.ack && f.ack.clientId === session.clientId && lastRequest?.seq === f.ack.seq) {
      acked = lastRequest.kind;
    }
    // An undo toast speaks of the latest move only: a newer move or an undo
    // replaces it. In a bot room the bot's reply keeps it, since Undo there
    // takes back the player's move together with the reply.
    if (acked !== null || f.room.mode === 'hotseat') toasts.clear();
    const passOn = settings.value.hotseatPassDevice ?? coarsePointer.current;
    const holderBefore = ui.holder;
    ui.record(f, acked, passOn);
    if (acked === 'act' && f.room.mode !== 'online' && f.canUndo) {
      const passNext =
        f.room.mode === 'hotseat' &&
        passOn &&
        f.view.toMove !== undefined &&
        f.view.toMove !== holderBefore;
      const line = eventLines(f.events, seatNames(f.room))[0];
      if (line && !passNext) {
        toasts.show(line, {
          kind: 'success',
          actionLabel: t('game.toast.undo'),
          onAction: undo
        });
      }
    }
  }

  // A toast when the other player joins, leaves or comes back (online).
  function announcePresence(prev: State | null, next: State): void {
    const names = seatNames(next.room);
    for (const change of presenceChanges(prev, next)) {
      const name = names[change.seat];
      switch (change.kind) {
        case 'joined':
          toasts.show(t('online.toast.joined', { name }), { kind: 'success' });
          break;
        case 'left':
          toasts.show(t('online.toast.left', { name }), { kind: 'warning' });
          break;
        case 'back':
          toasts.show(t('online.toast.back', { name }), { kind: 'success' });
          break;
      }
    }
  }

  async function copyInvite(): Promise<void> {
    try {
      await navigator.clipboard.writeText(inviteLink);
      toasts.show(t('online.lobby.copied'), { kind: 'success', duration: 2500 });
    } catch {
      toasts.show(t('online.lobby.copyFailed'), { kind: 'warning' });
    }
  }

  function shareInvite(): void {
    if (!frame) return;
    const code = frame.code;
    navigator
      .share({
        title: t('online.lobby.shareTitle'),
        text: t('online.lobby.shareText', { code }),
        url: inviteLink
      })
      .catch(() => {
        // Dismissed, or the share sheet failed: the link is still on screen.
      });
  }

  // The rematch on the result screen. A player asks for it (or joins the
  // one the other player asked for); a spectator can only follow one the
  // players made.
  const rematchOffer = $derived.by(() => {
    if (!frame || frame.room.status !== 'finished') return undefined;
    const code = session.rematchCode;
    if (spectator) {
      return code ? { label: t('online.rematch.watch'), note: undefined } : undefined;
    }
    if (mode === 'online' && code && !session.rematchAsked) {
      const other = mySeats[0] === 1 ? 0 : 1;
      return {
        label: t('online.rematch.join'),
        note: t('online.toast.rematch', { name: names[other] })
      };
    }
    return { label: undefined, note: undefined };
  });

  // The session prop is fixed for the screen's life (a new room is a new
  // frame on the same session, handled by the code check above).
  function listen(): () => void {
    return session.subscribe(onFrame);
  }
  const unsubscribe = listen();

  onDestroy(() => {
    unsubscribe();
    toasts.clear();
    // The room is left with this screen: the next one starts blank rather
    // than with this room's last frame.
    presenter.clear();
  });

  function act(key: string): void {
    if (!inputOpen) return;
    if (session.act(key)) {
      lastRequest = { seq: session.seq, kind: 'act' };
      ui.sent();
    }
  }

  function undo(): void {
    // A bot's answer held for its dwell is shown at once, so the undo takes
    // back what the player can see.
    presenter.hurry();
    if (session.undo()) {
      lastRequest = { seq: session.seq, kind: 'undo' };
      ui.sent();
      toasts.clear();
    }
  }

  function selectSlot(slot: SlotView): void {
    ui.selectSlot(slot.index);
  }

  function selectWonder(w: WonderView): void {
    ui.openWonder(w.wonder);
  }

  /**
   * Marks the element the bot's answer takes, for the pre-roll: the slot of
   * its card, or the draft offer of its wonder. DOM only (an attribute the
   * stylesheet below draws); it never writes state.
   */
  function markBotPick(selector: string | undefined) {
    return (el: HTMLElement) => {
      if (!selector) return;
      const target = el.querySelector(selector);
      target?.setAttribute('data-bot-pick', '');
      return () => target?.removeAttribute('data-bot-pick');
    };
  }
  const pickSlot = $derived(pick?.slot !== undefined ? `[data-slot="${pick.slot}"]` : undefined);
  const pickWonder = $derived(
    pick?.wonder !== undefined ? `[data-wonder-offer="${CSS.escape(pick.wonder)}"]` : undefined
  );

  function onKey(event: KeyboardEvent): void {
    if (event.key === 'Escape' && ui.pickFor && ui.sheet === null) ui.cancelWonderPick();
  }
</script>

<svelte:window onkeydown={onKey} />

{#snippet trackAndTokens(compact: boolean)}
  {#if view}
    <MilitaryTrack
      military={view.military}
      viewerSeat={trackSeat}
      leftSeat={home}
      {compact}
      class={compact ? 'min-w-24 flex-1' : 'w-full'}
    />
    <ProgressTokenRow tokens={view.boardTokens} {compact} class="justify-center" />
  {/if}
{/snippet}

{#snippet seatPanel(seat: Seat, cls: string)}
  {#if view}
    <SeatPanel
      {variant}
      name={names[seat]}
      {seat}
      player={view.players[seat]}
      military={view.military}
      score={view.scorePreview[seat]}
      lit={lit === seat}
      thinking={thinkingSeat === seat}
      presence={mySeats.includes(seat) ? undefined : presence[seat]}
      onwonder={mySeats.includes(seat) ? selectWonder : undefined}
      onopen={() => ui.openCity(seat)}
      class={cls}
    />
  {/if}
{/snippet}

{#if lobby && frame}
  <Lobby
    code={frame.code}
    url={inviteLink}
    seats={frame.room.seats}
    conn={session.conn}
    {canShare}
    oncopy={() => void copyInvite()}
    onshare={shareInvite}
    {onhome}
  />
  <ToastList {toasts} placement="bottom" />
{:else}
  <div
    class="game-screen h-full w-full overflow-hidden"
    data-layout={layout}
    data-draft={view?.phase === 'wonderDraft' ? '' : undefined}
    data-phase={view?.phase}
    data-home={home}
    data-pick={ui.pickFor ?? undefined}
    data-bot={bot}
    data-thinking={thinkingSeat}
    dir="ltr"
    aria-label={t('game.board')}
    role="region"
  >
    <div class="area-top" dir={i18n.dir}>
      <TopBar
        age={view?.age ?? 0}
        {cardsLeft}
        {canUndo}
        onUndo={mode === 'online' ? undefined : undo}
        onOpenLog={() => ui.openLog()}
        onOpenSettings={() => ui.openSettings()}
        onHome={onhome}
        homeHint={view?.phase !== 'gameOver'}
        class="ps-[max(0.5rem,var(--safe-l))] pe-[max(0.5rem,var(--safe-r))] pt-(--safe-t)"
      >
        {#if layout === 'phone-landscape' && view?.phase !== 'wonderDraft'}
          {@render trackAndTokens(true)}
        {/if}
        {#if mode === 'online' && !spectator && frame}
          <SpectatorCount count={frame.room.spectators} />
        {/if}
        <ConnectionPill conn={session.conn} compact />
      </TopBar>
    </div>

    {#if !view}
      <div class="area-main relative grid place-items-center p-4" dir={i18n.dir}>
        <p class="text-fg-muted" role="status" data-conn={session.conn}>
          {session.conn === 'open' ? t('game.loading') : t(connMessage(session.conn))}
        </p>
      </div>
    {:else if view.phase === 'wonderDraft'}
      <div
        class="area-main relative min-h-0 overflow-y-auto"
        dir={i18n.dir}
        {@attach markBotPick(pickWonder)}
      >
        <Draft {view} canAct={inputOpen} onact={act} {names} {confirm} {connected} />
      </div>
    {:else}
      <!-- The panels scroll inside a positioned box, so nothing absolutely
         positioned in them (sr-only text) can reach past it and stretch the page. -->
      <div class="area-away relative min-h-0 min-w-0" dir={i18n.dir}>
        {@render seatPanel(away, 'relative max-h-full overflow-y-auto')}
      </div>

      {#if layout !== 'phone-landscape'}
        <div class="area-board flex min-w-0 flex-col items-center gap-1.5 px-1" dir={i18n.dir}>
          {@render trackAndTokens(false)}
        </div>
      {/if}

      <div class="area-structure relative min-h-0 min-w-0" {@attach markBotPick(pickSlot)}>
        <CardStructure
          slots={view.structure}
          age={view.age}
          seat={badge}
          toMove={view.toMove}
          selected={ui.selected ?? undefined}
          pending={pendingIndex}
          {marked}
          onselect={selectSlot}
        />
        {#if view.phase === 'gameOver'}
          <!-- GameOver is its own centred dialog with a peek bar, so it needs no frame here. -->
          <div class="contents" dir={i18n.dir}>
            <GameOver
              {view}
              {names}
              onrematch={rematchOffer ? () => session.rematch() : undefined}
              rematchLabel={rematchOffer?.label}
              rematchNote={rematchOffer?.note}
              {onhome}
            />
          </div>
        {/if}
      </div>

      <div class="area-home relative min-h-0 min-w-0" dir={i18n.dir}>
        {@render seatPanel(home, 'relative max-h-full overflow-y-auto')}
        {#if toastSpot === 'home'}
          <!-- Desktop and phone landscape: the foot of this player's own
             column, clear of the structure's takeable bottom row, the track
             and the opponent. -->
          <ToastList {toasts} placement="inside" />
        {/if}
      </div>
    {/if}

    <div class="area-turn relative flex min-w-0 flex-col" dir={i18n.dir}>
      <!-- Toasts sit just above the turn bar: on a phone they cover only this
         player's own strip, never the opponent's strip or the track, which
         matter while the other side moves. -->
      {#if toastSpot === 'turn'}
        <ToastList {toasts} placement="above" />
      {/if}
      {#if session.notice && session.notice !== dismissedNotice}
        <div
          class="flex items-center gap-2 border-s-4 border-t border-cinnabar bg-panel px-3 py-1.5 text-sm text-fg"
          role="alert"
          data-notice={session.notice.code}
        >
          <span class="min-w-0 flex-1">{t(session.notice.message)}</span>
          <button
            type="button"
            class="min-h-11 shrink-0 rounded-control px-2 font-semibold text-fg-muted hover:text-fg"
            onclick={() => (dismissedNotice = session.notice)}
          >
            {t('shell.dismiss')}
          </button>
        </div>
      {/if}
      {#if session.catalogStale}
        <p class="border-t border-rule bg-panel px-3 py-1.5 text-sm text-fg" role="alert">
          {t('session.catalogStale')}
        </p>
      {/if}
      <!-- ChoiceHost stays mounted through the play: besides a pending choice it
         shows the "who starts" bar of a new age. It waits while the pass-device
         screen is up, because its panels are modal dialogs that would cover
         that screen and make it inert. -->
      {#if view && view.phase !== 'wonderDraft' && view.phase !== 'gameOver' && !showPass}
        <ChoiceHost {view} canAct={inputOpen} onact={act} {names} />
      {/if}
      {#if ui.pickFor}
        <div
          class="flex min-h-13 items-center justify-between gap-2 border-t border-rule bg-accent-soft px-3 py-1.5 text-accent-soft-fg"
          role="status"
          data-wonder-pick-bar={ui.pickFor}
        >
          <span class="text-sm font-semibold">
            {t('game.pick.prompt', { wonder: wonderName(ui.pickFor) })}
          </span>
          <button
            type="button"
            class="min-h-11 shrink-0 rounded-control border border-rule-strong bg-panel px-3 text-sm font-semibold text-fg"
            data-control="wonder-pick-cancel"
            onclick={() => ui.cancelWonderPick()}
          >
            {t('game.pick.cancel')}
          </button>
        </div>
      {:else}
        <TurnBar
          state={turn}
          {names}
          onRetry={() => session.retry()}
          class="pb-[max(0.375rem,var(--safe-b))]"
        />
      {/if}
    </div>
  </div>
{/if}

{#if view}
  <CardSheet
    open={ui.isOpen('card') && !!selectedSlot && selectedSlot.state !== 'empty'}
    {view}
    slot={selectedSlot}
    seat={badge}
    opponentName={names[badge === 1 ? 0 : 1]}
    {myTurn}
    canAct={inputOpen}
    {confirm}
    {connected}
    {spectator}
    step={ui.step}
    pickFor={ui.pickFor}
    onact={act}
    onwonders={() => ui.showWonders()}
    onback={() => ui.backToCard()}
    onclose={() => ui.close('card')}
  />

  <WonderSheet
    open={ui.isOpen('wonder') && !!wonderOwner}
    slug={ui.wonder}
    state={wonderOwner?.state}
    buildable={wonderInfo.buildable}
    reason={wonderInfo.reason}
    coins={wonderOwner?.coins}
    canAct={inputOpen}
    onbuild={(slug) => ui.startWonderPick(slug)}
    onclose={() => ui.close('wonder')}
  />

  <CitySheet
    bind:open={() => ui.isOpen('city'), (v) => !v && ui.close('city')}
    {names}
    players={view.players}
    military={view.military}
    scores={view.scorePreview}
    bind:activeSeat={ui.citySeat}
  />

  <LogSheet
    bind:open={() => ui.isOpen('log'), (v) => !v && ui.close('log')}
    events={ui.log}
    {names}
  />
{/if}

<Sheet
  bind:open={() => ui.isOpen('settings'), (v) => !v && ui.close('settings')}
  titleId="game-settings-title"
>
  <div class="flex flex-col gap-4 px-5 pt-1 pb-5 split:pt-5">
    <h2 id="game-settings-title" class="font-display text-xl text-fg">
      {t('game.settings.title')}
    </h2>
    <Settings />
  </div>
</Sheet>

{#if showPass && view && view.toMove !== undefined}
  {@const next = view.toMove === 1 ? 1 : 0}
  <div class="fixed inset-0 z-40" data-pass-device={next}>
    <PassDevice
      nextName={names[next]}
      nextSeat={next}
      summary={lastLines}
      {canUndo}
      previousName={ui.holder !== null ? names[ui.holder] : undefined}
      onReady={() => ui.ready(view?.toMove)}
      onUndo={undo}
    />
  </div>
{/if}

<LiveRegion message={lastLines.join(' ')} />

<style>
  .game-screen {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    /* The structure keeps at least 15.5rem, enough for 44 px Age III cards on
       a 360x740 phone; when both strips grow tall (tokens, a danger line)
       they give up height and scroll inside instead. */
    --strip-row: minmax(0, max-content);
    grid-template-rows: auto var(--strip-row) auto minmax(15.5rem, 1fr) var(--strip-row) auto;
    grid-template-areas: 'top' 'away' 'board' 'structure' 'home' 'turn';
    row-gap: 0.25rem;
  }
  .game-screen[data-layout='tablet'] {
    row-gap: 0.375rem;
  }
  .area-top {
    grid-area: top;
  }
  .area-away {
    grid-area: away;
    padding-inline: max(0.5rem, var(--safe-l)) max(0.5rem, var(--safe-r));
  }
  .area-board {
    grid-area: board;
  }
  .area-structure {
    grid-area: structure;
    padding-inline: max(0.5rem, var(--safe-l)) max(0.5rem, var(--safe-r));
    padding-block: 0.125rem;
  }
  .area-home {
    grid-area: home;
    padding-inline: max(0.5rem, var(--safe-l)) max(0.5rem, var(--safe-r));
  }
  .area-turn {
    grid-area: turn;
  }
  .area-main {
    grid-area: main;
  }

  /* The pre-roll: the card (or draft wonder) the bot is about to take, ringed
     in its crest colour while its thinking frame is still up. A static ring,
     so it reads the same with reduced motion. */
  .game-screen :global([data-bot-pick]) {
    outline: 3px solid var(--bot-ink, var(--accent));
    outline-offset: 3px;
    box-shadow:
      0 0 0 6px color-mix(in oklch, var(--bot-ink, var(--accent)) 28%, transparent),
      var(--shadow);
  }
  .game-screen :global(li[data-bot-pick]) {
    border-radius: var(--radius-panel, 0.75rem);
  }
  .game-screen[data-bot='0'] {
    --bot-ink: var(--sun);
  }
  .game-screen[data-bot='1'] {
    --bot-ink: var(--moon);
  }

  /* The draft and the loading state: the top bar, one main area, the turn bar. */
  .game-screen[data-draft],
  .game-screen:has(> .area-main) {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: auto minmax(0, 1fr) auto;
    grid-template-areas: 'top' 'main' 'turn';
  }

  /* Phone landscape: my strip and the turn bar / structure / opponent strip,
     with the track and tokens in the top bar. */
  .game-screen[data-layout='phone-landscape']:not([data-draft]):not(:has(> .area-main)) {
    grid-template-columns: minmax(0, 11.5rem) minmax(0, 1fr) minmax(0, 11.5rem);
    grid-template-rows: auto minmax(0, 1fr) auto;
    grid-template-areas:
      'top top top'
      'home structure away'
      'turn structure away';
    row-gap: 0.25rem;
  }
  .game-screen[data-layout='phone-landscape'] .area-home,
  .game-screen[data-layout='phone-landscape'] .area-away {
    padding-inline: max(0.25rem, var(--safe-l)) 0.25rem;
    padding-block-end: 0.25rem;
  }
  .game-screen[data-layout='phone-landscape'] .area-away {
    padding-inline: 0.25rem max(0.25rem, var(--safe-r));
  }
  .game-screen[data-layout='phone-landscape'] .area-structure {
    padding-inline: 0.25rem;
  }

  /* Desktop: my city panel / board / opponent's city panel. */
  .game-screen[data-layout='desktop']:not([data-draft]):not(:has(> .area-main)) {
    grid-template-columns: 300px minmax(0, 1fr) 300px;
    grid-template-rows: auto auto minmax(0, 1fr) auto;
    grid-template-areas:
      'top top top'
      'home board away'
      'home structure away'
      'home turn away';
    column-gap: 0.75rem;
    row-gap: 0.5rem;
  }
  .game-screen[data-layout='desktop'] .area-home,
  .game-screen[data-layout='desktop'] .area-away {
    padding-block: 0.25rem 0.75rem;
  }
  .game-screen[data-layout='desktop'] .area-board {
    padding-block-start: 0.25rem;
  }
  @media (min-width: 100rem) {
    .game-screen[data-layout='desktop']:not([data-draft]):not(:has(> .area-main)) {
      grid-template-columns: 380px minmax(0, 1fr) 380px;
    }
  }
</style>
