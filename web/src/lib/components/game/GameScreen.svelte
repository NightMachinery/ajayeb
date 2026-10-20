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
  // (or the wonder it drafts) is handed to CardStructure (or Draft) as its
  // botPick prop meanwhile, which marks it with data-bot-pick. The military
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
  // says so and the move buttons (the pending choices' included) read
  // "Waiting for connection"; the Session sends nothing until it is back. A
  // spectator gets the same board, read only. Online rooms have no undo, and
  // every move takes the confirm step the settings ask for.
  //
  // A wonder's extra turn reads "Play again" in the turn bar for the whole
  // turn (playAgainFor over the shown view and its history,
  // so it survives a reload). A player's own move gets no
  // toast (they just made it); the one exception is a choice the server
  // skipped (nothing to destroy, an empty discard pile), which the player
  // could not foresee, so a toast explains it.
  //
  // Undo (hot-seat and bot games created with undo allowed, a setting the
  // server fixes for the game's life: room.allowUndo) is the game menu's
  // "Undo last move", the top bar's arrow, U or Ctrl+Z, and the pass-device
  // screen's "Undo X's move". A game without undo shows none of them.
  //
  // The opponent move summary opens when unseen opponent moves hand back
  // the turn, including on reconnect, reload and waking a hidden page.
  // The per-room/seat marker advances only after a visible presented frame.
  // The pass-device screen already summarizes hot-seat moves, so it replaces
  // the automatic popup there. The eye button always reopens the latest run.
  // Any tap closes a summary and is swallowed, like a tooltip.
  //
  // The tutorial (M10.tutorial-client, the tutorial design): in a tutorial
  // room the presented frame carries the server's tutorial object. The step
  // machine (lib/tutorial/steps.ts) turns it, the Next acknowledgements and
  // this screen's sheets into the coach step; CoachLayer shows it as a
  // bubble on the board and rings its target, and the card sheet, its
  // wonder step and the draft's wonder sheet show it as a note. Play on my
  // own pauses the tutorial in this room (the tips store), and the game
  // menu resumes it. No first-time tip shows while a coach step is due or
  // the guided part runs.
  import { resolve } from '$app/paths';
  import { onDestroy, onMount } from 'svelte';
  import type { Attachment } from 'svelte/attachments';
  import { SeenMoves } from '$lib/ui/seen-moves';
  import ArtPrefetch from '$lib/art/ArtPrefetch.svelte';
  import { seatBotLevel } from '$lib/art/lookup';
  import { MediaQuery } from 'svelte/reactivity';
  import { wonderName } from '$lib/components/card/effect-text';
  import { i18n, t } from '$lib/i18n';
  import { breakpoints } from '$lib/layout/breakpoints.svelte';
  import { connMessage } from '$lib/session/messages';
  import type { Presenter } from '$lib/session/presenter.svelte';
  import type { Notice, Session } from '$lib/session/session.svelte';
  import { inviteUrl, presenceChanges, seatPresence } from '$lib/ui/online';
  import { settings } from '$lib/ui/settings.svelte';
  import { unlockAudio } from '$lib/fx';
  import { GameUi, type Acked, type Seat } from '$lib/ui/ui.svelte';
  import { badgeSeat } from '$lib/view/afford';
  import {
    lastPly,
    latestSummary,
    summarizeSince,
    summaryTrigger,
    summaryViewer
  } from '$lib/view/move-summary';
  import { hintHide } from '$lib/hints/local.svelte';
  import { visibleFamilies } from '$lib/hints/families';
  import {
    EMPTY_RECAP,
    isCurrentRun,
    nextRecap,
    priceTicks,
    recapBoard,
    setRecapBoard
  } from '$lib/view/recap';
  import { reviewForSummary, reviewLines } from '$lib/view/review';
  import { ageTip } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { seatGhost, selectionGhost } from '$lib/view/outcome';
  import { gameEnded } from '$lib/view/threats';
  import { playAgainFor, turnState } from '$lib/view/turn';
  import { buildableWonders } from '$lib/view/wonders';
  import type { SlotView, State, WonderView } from '$lib/wire';
  import MilitaryTrack from '../board/MilitaryTrack.svelte';
  import CardStructure from '../board/CardStructure.svelte';
  import ChoiceHost from '../choices/ChoiceHost.svelte';
  import { AgeSummary } from '$lib/ui/age-summary.svelte';
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
  import WonderListSheet from '../sheets/WonderListSheet.svelte';
  import WonderSheet from '../sheets/WonderSheet.svelte';
  import FxLayer from '../shell/FxLayer.svelte';
  import { replay as moveReplay } from '$lib/fx/anim/replay.svelte';
  import { motion } from '$lib/ui/motion.svelte';
  import LiveRegion from '../shell/LiveRegion.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import ToastList from '../shell/Toasts.svelte';
  import { toastLane } from '$lib/ui/toast-lane';
  import { Toasts } from '../shell/toast-queue.svelte';
  import ProgressTokenRow from '../token/ProgressTokenRow.svelte';
  import TokenSheet from '../sheets/TokenSheet.svelte';
  import CardSheet from './CardSheet.svelte';
  import ForfeitConfirm from './ForfeitConfirm.svelte';
  import MoveSummary from './MoveSummary.svelte';
  import MoveReview from './MoveReview.svelte';
  import FirstTips from '../hints/FirstTips.svelte';
  import CoachLayer from '../tutorial/CoachLayer.svelte';
  import TutorialDone from '../tutorial/TutorialDone.svelte';
  import { goto } from '$app/navigation';
  import { tips } from '$lib/ui/tips.svelte';
  import { COACH_TEXT_ID, coachAcks, type CoachNoteView } from '$lib/tutorial/coach.svelte';
  import { coachStep, type CoachStep, type CoachSurface } from '$lib/tutorial/steps';
  import { coachText, lockedReason } from '$lib/tutorial/text';
  import { tutorialAllow } from '$lib/tutorial/gate';
  import {
    botPick,
    cardActions,
    eventLines,
    homeSeat,
    needsConfirm,
    passPending,
    pendingSlot,
    pickableSlots,
    wonderStatus
  } from './card-actions';
  import { botSeat, seatNames } from './names';
  import { boardUndoKey, modalOpen } from './undo-keys';
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
  const seenMoves = new SeenMoves();
  const ageSummary = new AgeSummary();
  let pageVisible = $state(
    typeof document === 'undefined' || document.visibilityState !== 'hidden'
  );
  const coarsePointer = new MediaQuery('(pointer: coarse)', false);
  // The draft's split layout (Draft.svelte): every tile shows its Take button.
  const draftSplit = new MediaQuery(
    '(min-width: 64rem), (orientation: landscape) and (max-height: 32rem)',
    false
  );

  // The request this screen sent last, to tell an act's ack from an undo's.
  let lastRequest: { seq: number; kind: 'act' | 'undo' } | null = null;
  let roomCode: string | null = null;
  // The previous frame of this room, for the presence toasts.
  let lastFrame: State | null = null;
  // The notice the player dismissed; a newer one shows again.
  let dismissedNotice = $state<Notice | null>(null);
  // The seat whose wonder list is open (the phone strip's wonder button),
  // and the seat whose list the open wonder sheet was reached from, for its
  // Back button. Kept here rather than in GameUi: only this screen opens it.
  let wonderListSeat = $state<Seat | null>(null);
  let wonderFromList = $state<Seat | null>(null);
  // The list's seat while it closes, so its content stays put.
  let wonderListShown = $state<Seat>(0);
  // The forfeit's confirm step (from the game menu).
  let forfeitOpen = $state(false);
  // The draft's wonder sheet on a phone: the wonder it shows, or null.
  let draftDetail = $state<string | null>(null);
  // Where the player resumed the tutorial after leaving its path (room and
  // ply): leftPath shows there, once.
  let resumedLeft = $state<{ code: string; ply: number } | null>(null);
  // The room whose tutorial completion card is open, or null.
  let doneRoom = $state<string | null>(null);
  // The board token whose sheet is open (a tap on a board token while no
  // token pick is pending), kept while the sheet closes.
  let tokenOpen = $state(false);
  let tokenSlug = $state<string | null>(null);
  // The opponent move summary: the ply of the entry a live frame opened it
  // for (shown once the Presenter has that entry on screen), and whether
  // the top bar's button opened it.
  let summaryAuto = $state<number | null>(null);
  let summaryManual = $state(false);
  let summaryAway = $state(false);
  // What changed since each seat's last move (lib/view/recap.ts), from the
  // frames in order: the summary's price lines.
  let recap = $state.raw(EMPTY_RECAP);
  // The boxes the summary is placed over.
  let structureEl = $state<HTMLElement | undefined>();
  let homeEl = $state<HTMLElement | undefined>();
  let awayEl = $state<HTMLElement | undefined>();
  let turnEl = $state<HTMLElement | undefined>();

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
  // Whether the game has undo at all (fixed when it was created), and
  // whether it can be sent now. Undo stays available while a bot's answer
  // is held on screen: undo() shows it at once and then takes it back with
  // the player's move.
  const undoAllowed = $derived(mode !== 'online' && !!frame?.room.allowUndo);
  const canUndo = $derived(undoAllowed && session.canUndo);
  // Why the menu's undo is greyed out, read from the frame and the
  // connection only (the server's canUndo is the rule).
  const undoReason = $derived(
    canUndo
      ? undefined
      : session.conn !== 'open'
        ? t('game.menu.undoOffline')
        : session.pending
          ? t('game.menu.undoWaiting')
          : frame?.undoUnavailable === 'online'
            ? t('game.menu.undoOnline')
            : frame?.undoUnavailable === 'disabled'
              ? t('game.menu.undoDisabled')
              : frame?.undoUnavailable === 'spectator'
                ? t('game.menu.undoSpectator')
                : frame?.undoUnavailable === 'game-over'
                  ? t('game.menu.undoOver')
                  : t('game.menu.undoNothing')
  );
  // The seat a forfeit would give up now (the Session says when it can be
  // sent: a seated player while the game runs, the player to move in a
  // hot-seat game), and the seat that would win.
  const resignSeat = $derived(session.resignSeat);
  const resignWinner = $derived<Seat>(resignSeat === 1 ? 0 : 1);

  const layout = $derived(breakpoints.viewport);
  // Where the toasts sit in play. Above the turn bar by default (on a phone
  // that covers only this player's own strip). On a desktop they sit at the
  // foot of this player's own column, clear of the structure's takeable
  // bottom row, the track and the opponent. In phone landscape that column
  // is a narrow strip with little room under it, so a long sentence would
  // grow up over the strip, and the structure fills the height, its foot
  // holding the cards to take for most of an age and its head near the end;
  // there they take the first lane clear of every card that can be taken
  // and both strips (ui/toast-lane.ts).
  const toastSpot = $derived(
    !view || view.phase === 'wonderDraft'
      ? ('turn' as const)
      : layout === 'desktop'
        ? ('home' as const)
        : layout === 'phone-landscape'
          ? ('lane' as const)
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
  const ended = $derived(gameEnded(view?.phase));
  // The wonders each seat can build now, from the view's choices only.
  const buildable = $derived(
    view
      ? ([buildableWonders(view, 0), buildableWonders(view, 1)] as const)
      : ([new Set<string>(), new Set<string>()] as const)
  );
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
  // A board token opens its sheet unless a token pick is pending, which
  // keeps its own pick behaviour (the choice panel).
  const tokenPick = $derived(
    view?.pending?.kind === 'pickBoardToken' || view?.pending?.kind === 'pickLibraryToken'
  );
  function openToken(slug: string): void {
    tokenSlug = slug;
    tokenOpen = true;
  }
  const pendingKey = $derived(
    session.pending?.frame.t === 'act' ? session.pending.frame.key : undefined
  );
  const pendingIndex = $derived(pendingSlot(view, pendingKey));
  const marked = $derived(view && ui.pickFor ? pickableSlots(view, ui.pickFor) : undefined);
  const selectedSlot = $derived<SlotView | undefined>(
    view && ui.selected !== null ? view.structure[ui.selected] : undefined
  );

  // The tutorial: the presented frame's object, whether it is paused here,
  // and the coach step due now.
  const tutorial = $derived(frame?.tutorial);
  const tutorialPaused = $derived(!!frame && tips.isPaused(frame.code));
  // The rails: during a guided turn only the guided key's buttons are open.
  const allow = $derived(tutorialAllow(tutorial, tutorialPaused));
  const locked = $derived(lockedReason(tutorial));
  const cardSheetOpen = $derived(
    ui.isOpen('card') && !!selectedSlot && selectedSlot.state !== 'empty'
  );
  const coach = $derived.by((): CoachStep | null => {
    if (!frame || !view || !tutorial) return null;
    const extra =
      resumedLeft?.code === frame.code &&
      resumedLeft.ply === view.ply &&
      tutorial.left &&
      !tutorial.lessons.some((l) => l.id === 'leftPath')
        ? [{ id: 'leftPath' }]
        : [];
    return coachStep({
      tutorial,
      code: frame.code,
      ply: view.ply,
      paused: tutorialPaused,
      acked: (key) => coachAcks.has(key),
      extra,
      ui: {
        sheet:
          cardSheetOpen && selectedSlot
            ? {
                card: selectedSlot.state === 'faceUp' ? (selectedSlot.card ?? null) : null,
                step: ui.step
              }
            : null,
        draftSheet: view.phase === 'wonderDraft' ? draftDetail : null,
        split: draftSplit.current,
        pickBar: !!ui.pickFor
      }
    });
  });

  // The history up to the frame on screen: a bot's answer still being
  // paced by the Presenter is not in it yet.
  const shownHistory = $derived.by(() => {
    const history = session.history;
    const ply = view?.ply;
    const last = history[history.length - 1];
    if (ply === undefined || !last || last.ply < ply) return history;
    return history.filter((e) => e.ply < ply);
  });
  // Its events: the whole game even after a reload. Without a history (a
  // fixture room), the events this screen has seen.
  const shownLog = $derived(
    shownHistory.length > 0 ? shownHistory.flatMap((e) => e.events) : ui.log
  );

  // The board hints of this screen (M10.HS4): the selection ghosts of the
  // selected card's build (the pawn on the board's track, coins and score
  // on both seats' panels), and the age chip's tooltip.
  const boardHints = $derived(sessionHintScreen(session, hintHide.value));
  const ghost = $derived.by(() => {
    // Not in the wonder step, whose moves are wonders, not this build.
    if (!view || !boardHints?.boardMarks || !selectedSlot?.card || ui.step !== 'card') {
      return undefined;
    }
    const build = cardActions(view, selectedSlot.card).build;
    return selectionGhost(
      build?.outcome,
      [view.scorePreview[0].total, view.scorePreview[1].total],
      boardHints.families
    );
  });
  const ageChipTip = $derived(
    view && boardHints ? ageTip(view, cardsLeft, boardHints.families, boardHints.names) : undefined
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
      playAgain: playAgainFor(view, shownLog)
    })
  );
  const thinkingSeat = $derived(turn.kind === 'aiThinking' ? turn.seat : undefined);
  // During the Presenter's pre-roll: what the bot's answer takes.
  const pick = $derived(botPick(view, presenter.preview, frame?.thinking));

  const passRequested = $derived(
    !!view &&
      passPending({
        mode,
        enabled: passDevice,
        phase: view.phase,
        toMove: view.toMove,
        holder: ui.holder
      })
  );
  const lastLines = $derived(frame ? eventLines(presenter.shownEvents, names) : []);

  const ageViewer = $derived(mode === 'hotseat' && passDevice ? ui.holder : null);
  const ageBoundary = $derived(ageSummary.viewFor(ageViewer));
  const ageUnread = $derived(
    !!ageBoundary && !!view && view.age >= ageBoundary.age && !ageSummary.isDismissed(ageViewer)
  );
  // Ages II and III open on the starter choice, and the Presenter holds that
  // frame while its deal plays (fx/anim/plan.ts dealHold). The age dialog
  // waits for the Presenter anyway; the pass-device screen and the move
  // summary wait too, so nothing covers the deal (M11.age-start-order).
  const dealing = $derived(!presenter.idle && view?.phase === 'chooseStarter');
  // Choosing the other starter must not unmount this holder's open age
  // summary. Their explicit close releases the requested handoff.
  const showPass = $derived(
    passRequested && !dealing && !(ageSummary.isShown(ageViewer) && ageUnread)
  );
  // A chooser may take the final card and choose a starter before handing
  // the device on. Give the next holder that whole existing opponent run
  // before their unread age dialog, rather than only starterChosen.
  const passLines = $derived.by(() => {
    const next = view?.toMove;
    const boundary = next === undefined ? null : ageSummary.viewFor(next);
    if (
      !showPass ||
      next === undefined ||
      !boundary ||
      !view ||
      view.age < boundary.age ||
      ageSummary.isDismissed(next)
    )
      return lastLines;
    const run = summarizeSince(shownHistory, next === 1 ? 1 : 0);
    return run
      ? eventLines(
          shownHistory
            .filter((entry) => run.plies.includes(entry.ply))
            .flatMap((entry) => entry.events),
          names
        )
      : lastLines;
  });
  const summarySeat = $derived(view ? summaryViewer(mode, mySeats, view.toMove) : null);
  const moveSummary = $derived(
    summarySeat === null ? null : latestSummary(shownHistory, summarySeat)
  );
  const summaryOpen = $derived(
    pageVisible &&
      !dealing &&
      !(ageSummary.isShown(ageViewer) && ageUnread) &&
      !!moveSummary &&
      (summaryManual ||
        (summaryAuto !== null &&
          settings.value.showOpponentMove &&
          !showPass &&
          lastPly(moveSummary) === summaryAuto))
  );
  const coachWords = $derived(coach ? coachText(coach, names[bot === 0 ? 0 : 1]) : null);
  // A coach step shows on the player's own turn of the frame on screen,
  // with the Presenter idle and the move summary closed.
  const coachActive = $derived(
    !!coach && myTurn && presenter.idle && !summaryOpen && !showPass && session.pending === null
  );
  /** The coach step as a sheet's note, while it belongs to that sheet. */
  function noteFor(surface: CoachSurface): CoachNoteView | null {
    if (!coach || !coachWords || !coachActive || coach.surface !== surface) return null;
    return {
      title: coachWords.title,
      progress: coachWords.progress,
      text: coachWords.text,
      step: coach.id,
      kind: coach.kind,
      textId: COACH_TEXT_ID,
      onpause: tutorial?.status === 'guided' ? pauseTutorial : undefined
    };
  }
  const cardNote = $derived(noteFor('card-sheet'));
  const draftNote = $derived(noteFor('draft-sheet'));
  // The completion card is due once per room when the tutorial completes
  // in play (Age II), with nothing else on screen; it waits for the move
  // summary, the presenter and any modal (checked in the DOM, below). A
  // game over (a fast finish, or a forfeit) shows no card.
  const doneDue = $derived(
    !!frame &&
      !!view &&
      tutorial?.status === 'complete' &&
      view.phase !== 'gameOver' &&
      !tips.isCompleted(frame.code)
  );
  const doneReady = $derived(doneDue && presenter.idle && !summaryOpen && !showPass);
  // First-time tips wait while a coach step is due, through the guided part,
  // until the completion card has shown (it marks newAge seen) and while it
  // is open (a tip beside it would swallow the tap on its buttons).
  const tipsQuiet = $derived(
    !!coach ||
      (tutorial?.status === 'guided' && !tutorialPaused) ||
      doneDue ||
      (!!frame && doneRoom === frame.code)
  );

  // On a desktop, the viewer's own column (in a fixed hot-seat board the
  // player to move may sit on the far side), so the mover's city stays in
  // view. On a phone, over the structure: in landscape the side columns are
  // only as wide as a strip, so the summary would cover a whole strip.
  const summaryAnchor = $derived(
    layout === 'desktop' ? (summarySeat === away ? awayEl : homeEl) : structureEl
  );

  // The summary's move review (the advisor) and the viewer's trade prices
  // the other side changed (the recap), each while its family shows here.
  const shownFamilies = $derived(visibleFamilies(session.hints, hintHide.value));
  const summaryReview = $derived.by(() => {
    if (summarySeat === null || !shownFamilies.has('advisor')) return null;
    const review = reviewForSummary(view?.review, moveSummary, shownHistory);
    return review ? reviewLines(review, { names, seat: summarySeat }) : null;
  });
  const summaryPrices = $derived(
    view &&
      summarySeat !== null &&
      shownFamilies.has('recap') &&
      isCurrentRun(moveSummary, shownHistory)
      ? priceTicks(recap, view, summarySeat)
      : []
  );

  // The recap's board marks (M10.F3): the trade-price carets, the badge
  // lines, the emptied slots' traces and the fresh cards' dots, for the
  // viewer's seat until its move ends, while the recap family shows here.
  // The strips and the structure read them from this context.
  const recapMarks = $derived(
    view && !ended && shownFamilies.has('recap')
      ? recapBoard(recap, view, summarySeat, names, !hintHide.value.boardMarks)
      : null
  );
  setRecapBoard(() => recapMarks);

  function closeSummary(): void {
    summaryAuto = null;
    summaryManual = false;
    summaryAway = false;
  }

  function dismissSummary(): void {
    // An explicit dismissal proves this summary was seen, even if the page
    // hides again before the next animation frame can record the board.
    const key = frame ? seenKey(frame) : null;
    const ply = moveSummary ? lastPly(moveSummary) : undefined;
    if (key && pageVisible && ply !== undefined) {
      seenMoves.remember(key, Math.max(seenMoves.get(key), ply));
    }
    closeSummary();
  }

  function seenKey(f: State): string | null {
    const seat = summaryViewer(f.room.mode, f.you.seats, f.view.toMove);
    return seat === null ? null : `${f.code}:${f.room.mode}:${seat}`;
  }

  // This attachment reruns with the presented frame, after its DOM update.
  // A hidden incoming frame is never counted as seen. Effects/attachments
  // only schedule DOM work; the marker itself has no reactive state.
  function rememberShown(
    f: State | null,
    idle: boolean,
    visible: boolean,
    ply: number
  ): Attachment {
    return () => {
      if (!f || !idle || !visible) return;
      const key = seenKey(f);
      if (!key) return;
      const raf = requestAnimationFrame(() => {
        if (document.visibilityState !== 'hidden') seenMoves.remember(key, ply);
      });
      return () => cancelAnimationFrame(raf);
    };
  }

  function openUnseen(f: State, prev: State | null, away: boolean): void {
    const key = seenKey(f);
    const passOn = settings.value.hotseatPassDevice ?? coarsePointer.current;
    const passing = passPending({
      mode: f.room.mode,
      enabled: passOn,
      phase: f.view.phase,
      toMove: f.view.toMove,
      holder: ui.holder
    });
    const trigger =
      key && settings.value.showOpponentMove && !passing
        ? summaryTrigger(prev, f, session.history, seenMoves.get(key))
        : null;
    if (trigger !== null) {
      // Presence can arrive before the visible-ply write. Keep the reason
      // for this same summary instead of erasing its away label.
      summaryAway = away || (summaryAuto === trigger && summaryAway);
      summaryAuto = trigger;
      summaryManual = false;
    } else if (f.events.length > 0 || f.history) closeSummary();
  }

  onMount(() => {
    const wake = (): void => {
      pageVisible = document.visibilityState !== 'hidden';
      if (pageVisible && lastFrame) openUnseen(lastFrame, lastFrame, true);
    };
    document.addEventListener('visibilitychange', wake);
    return () => document.removeEventListener('visibilitychange', wake);
  });

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
  // device holder and the skipped-choice toast. Runs outside any effect.
  function onFrame(f: State): void {
    ageSummary.record(f);
    if (f.code !== roomCode) {
      roomCode = f.code;
      lastFrame = null;
      closeSummary();
      ui.reset();
      tokenOpen = false;
      tokenSlug = null;
      toasts.clear();
      wonderListSeat = null;
      wonderFromList = null;
      draftDetail = null;
      resumedLeft = null;
    }
    announcePresence(lastFrame, f);
    const prevFrame = lastFrame;
    lastFrame = f;
    recap = nextRecap(recap, f);
    let acked: Acked = null;
    if (f.ack && f.ack.clientId === session.clientId && lastRequest?.seq === f.ack.seq) {
      acked = lastRequest.kind;
    }
    const key = seenKey(f);
    const rewound =
      acked === 'undo' ||
      (!!prevFrame && f.view.ply < prevFrame.view.ply) ||
      (key !== null && seenMoves.get(key) >= f.view.ply);
    if (rewound) {
      // History entries are zero based; view.ply counts accepted moves.
      seenMoves.rewind(f.code, f.view.ply - 1);
      closeSummary();
    } else if (pageVisible) {
      openUnseen(f, prevFrame, !prevFrame || !!f.history);
    }
    // A skipped-choice toast speaks of the latest move only: a newer move
    // or an undo replaces it. In a bot room the bot's reply keeps it.
    if (acked !== null || f.room.mode === 'hotseat') toasts.clear();
    if (acked === 'undo' && f.room.mode === 'ai') {
      // ToastList's polite live region announces the visible confirmation
      // once, rather than repeating it in the board's event live region.
      toasts.show(t('game.undo.done'), { kind: 'info', duration: 5000 });
    }
    // An undo goes back to a guided turn: its lessons show again.
    if (acked === 'undo' && f.tutorial) coachAcks.forgetFrom(f.code, f.view.ply);
    const passOn = settings.value.hotseatPassDevice ?? coarsePointer.current;
    const holderBefore = ui.holder;
    ui.record(f, acked, passOn);
    if (acked === 'act') {
      const passNext =
        f.room.mode === 'hotseat' &&
        passOn &&
        f.view.toMove !== undefined &&
        f.view.toMove !== holderBefore;
      // No toast echoes the player's own move. Only a choice the move
      // should have opened but the server skipped gets one, saying why
      // (UX design: "the server skips it and a toast explains why"). The
      // pass-device screen shows the same lines in its summary instead.
      const skippedLines = eventLines(
        f.events.filter((e) => e.kind === 'effectSkipped'),
        seatNames(f.room)
      );
      if (skippedLines.length > 0 && !passNext) {
        toasts.show(skippedLines.join(' '), { kind: 'info', duration: 8000 });
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
    if (!inputOpen || !allow(key)) return;
    wonderListSeat = null;
    // The science lesson teaches what the symbol tip would.
    const science = tutorial?.action?.id === 'science' && tutorial.action.key === key;
    if (session.act(key)) {
      lastRequest = { seq: session.seq, kind: 'act' };
      ui.sent();
      if (science) tips.markSeen('science');
    }
  }

  /** Next on a coach step's info lesson. */
  function coachNext(step: CoachStep): void {
    if (step.ack) coachAcks.add(step.ack);
    // The military lesson teaches what the pawn tip would.
    if (step.id === 'military') tips.markSeen('track');
  }

  /** Play on my own: no coach steps and no rails in this room. */
  function pauseTutorial(): void {
    if (!frame) return;
    tips.pause(frame.code);
    toasts.show(t('tutorial.paused.toast'), { kind: 'info', duration: 5000 });
  }

  /** Resume tutorial (the game menu). After leaving the path it says so. */
  function resumeTutorial(): void {
    if (!frame || !view) return;
    tips.resume(frame.code);
    resumedLeft = frame.tutorial?.left ? { code: frame.code, ply: view.ply } : null;
  }

  // The completion card opens when no modal is left (the age-start sheet,
  // a first-time tip). DOM only: the observer callback writes the state.
  $effect(() => {
    if (!doneReady || !frame) return;
    const code = frame.code;
    let raf = 0;
    const check = (): void => {
      raf = 0;
      if (modalOpen(document)) return;
      doneRoom = code;
      tips.markCompleted(code);
      tips.markSeen('newAge');
    };
    const schedule = (): void => {
      if (!raf) raf = requestAnimationFrame(check);
    };
    const observer = new MutationObserver(schedule);
    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['open']
    });
    schedule();
    return () => {
      observer.disconnect();
      if (raf) cancelAnimationFrame(raf);
    };
  });

  /** The completion card's New game: Home with the vs-AI sheet open. */
  function tutorialNewGame(): void {
    doneRoom = null;
    // The query rides on the resolved Home path.
    // eslint-disable-next-line svelte/no-navigation-without-resolve
    void goto(`${resolve('/')}?play=ai`);
  }

  /** Sends the forfeit the confirm step agreed to. */
  function forfeit(): void {
    forfeitOpen = false;
    wonderListSeat = null;
    if (session.resign()) {
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
    wonderFromList = null;
    ui.openWonder(w.wonder);
  }

  /** From a seat's wonder list to one wonder's own sheet (Back returns). */
  function selectListedWonder(seat: Seat, w: WonderView): void {
    wonderListSeat = null;
    wonderFromList = seat;
    ui.openWonder(w.wonder);
  }

  function openWonderList(seat: Seat): void {
    wonderListShown = seat;
    wonderListSeat = seat;
  }

  function backToWonderList(): void {
    const seat = wonderFromList;
    ui.close('wonder');
    wonderFromList = null;
    if (seat !== null) openWonderList(seat);
  }

  /**
   * Replay last move (the log sheet's control): closes the sheet, then lets
   * the animation runner play the last move again on the board once the
   * sheet is gone. Visual only (fx/anim/replay.svelte.ts).
   */
  function replayLastMove(): void {
    ui.close('log');
    requestAnimationFrame(() => requestAnimationFrame(() => moveReplay.play()));
  }

  /** Whether a key press belongs to a text field rather than the board. */
  function typing(target: EventTarget | null): boolean {
    return (
      target instanceof HTMLElement &&
      (target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName))
    );
  }

  function onKey(event: KeyboardEvent): void {
    if (event.key === 'Escape' && ui.pickFor && ui.sheet === null) ui.cancelWonderPick();
    // U or Ctrl+Z (Cmd+Z on a Mac): undo, in a game that has it, unless a
    // text field or an open sheet or dialog has the keyboard.
    const where = { typing: typing(event.target), modalOpen: modalOpen(document) };
    if (boardUndoKey(event, where) && undoAllowed) {
      event.preventDefault();
      if (canUndo) undo();
    }
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
      legend={layout === 'desktop'}
      ghost={ghost?.pawn}
      class={compact ? 'min-w-24 flex-1' : 'w-full'}
    />
    <ProgressTokenRow
      tokens={view.boardTokens}
      {compact}
      onopen={tokenPick ? undefined : openToken}
      class="flex-nowrap justify-center {compact ? 'shrink-0' : ''}"
    />
  {/if}
{/snippet}

{#snippet seatPanel(seat: Seat, cls: string)}
  {#if view}
    <SeatPanel
      {variant}
      name={names[seat]}
      {seat}
      player={view.players[seat]}
      cardTooltips={view.cardTooltips}
      {names}
      you={mode === 'hotseat' ? undefined : view.viewer}
      military={view.military}
      score={view.scorePreview[seat]}
      bot={seatBotLevel(frame?.room, seat)}
      {ended}
      buildable={buildable[seat]}
      onwonders={() => openWonderList(seat)}
      lit={lit === seat}
      thinking={thinkingSeat === seat}
      presence={mySeats.includes(seat) ? undefined : presence[seat]}
      onwonder={mySeats.includes(seat) ? selectWonder : undefined}
      onopen={() => ui.openCity(seat)}
      ghost={seatGhost(ghost, seat)}
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
    {@attach rememberShown(frame, presenter.idle, pageVisible, shownHistory.at(-1)?.ply ?? -1)}
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
    tabindex="-1"
  >
    <div class="area-top" dir={i18n.dir}>
      <TopBar
        age={view?.age ?? 0}
        {cardsLeft}
        ageTip={ageChipTip}
        starterChoice={view?.phase === 'chooseStarter' && view.military.position === 0}
        {canUndo}
        onUndo={undoAllowed ? undo : undefined}
        {undoReason}
        onOpenLog={() => ui.openLog()}
        onOpenSummary={moveSummary ? () => (summaryManual = true) : undefined}
        onOpenSettings={() => ui.openSettings()}
        sound={settings.value.sound}
        onToggleSound={() => {
          settings.set('sound', !settings.value.sound);
          if (settings.value.sound) unlockAudio();
        }}
        onHome={onhome}
        homeHint={view?.phase !== 'gameOver'}
        replayCode={frame?.code}
        replayReason={view?.phase !== 'gameOver'
          ? t('game.menu.replayRunning')
          : !view.revealed
            ? t('game.menu.replayHidden')
            : undefined}
        onForfeit={resignSeat !== null ? () => (forfeitOpen = true) : undefined}
        tutorialPaused={tutorial && tutorial.status !== 'complete' ? tutorialPaused : undefined}
        onTutorial={tutorialPaused ? resumeTutorial : pauseTutorial}
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
      <div class="area-main relative min-h-0 overflow-y-auto" dir={i18n.dir}>
        <Draft
          {view}
          canAct={inputOpen}
          onact={act}
          {names}
          {confirm}
          {connected}
          botPick={pick?.wonder}
          bind:detail={draftDetail}
          coach={draftNote}
          {allow}
          lockedReason={locked}
        />
      </div>
    {:else}
      <!-- The panels scroll inside a positioned box, so nothing absolutely
         positioned in them (sr-only text) can reach past it and stretch the page. -->
      <div
        class="area-away relative min-h-0 min-w-0"
        dir={i18n.dir}
        data-tut="away"
        bind:this={awayEl}
      >
        {@render seatPanel(away, 'relative max-h-full overflow-y-auto')}
      </div>

      {#if layout !== 'phone-landscape'}
        <div class="area-board flex min-w-0 flex-col items-center gap-2 px-1" dir={i18n.dir}>
          {@render trackAndTokens(false)}
        </div>
      {/if}

      <div class="area-structure relative min-h-0 min-w-0" bind:this={structureEl}>
        <CardStructure
          slots={view.structure}
          cardTooltips={view.cardTooltips}
          choices={view.choices}
          costSeat={view.toMove}
          {names}
          you={mode === 'hotseat' ? undefined : view.viewer}
          age={view.age}
          seat={badge}
          toMove={view.toMove}
          selected={ui.selected ?? undefined}
          pending={pendingIndex}
          {marked}
          botPick={pick?.slot}
          onselect={selectSlot}
        />
      </div>
      {#if toastSpot === 'lane'}
        <!-- Phone landscape: the first lane clear of the cards to take, both
             strips and the turn bar (see toastSpot). -->
        <div
          class="toast-lane pointer-events-none fixed z-50"
          dir={i18n.dir}
          {@attach toastLane(() => ({
            structure: structureEl ?? null,
            home: homeEl ?? null,
            away: awayEl ?? null,
            turn: turnEl ?? null,
            version: frame?.version ?? 0
          }))}
        >
          <ToastList {toasts} placement="flow" />
        </div>
      {/if}

      <div
        class="area-home relative min-h-0 min-w-0"
        dir={i18n.dir}
        data-tut="home"
        bind:this={homeEl}
      >
        {@render seatPanel(home, 'relative max-h-full overflow-y-auto')}
        {#if toastSpot === 'home'}
          <!-- Desktop: the foot of this player's own column (see toastSpot). -->
          <ToastList {toasts} placement="inside" />
        {/if}
      </div>
    {/if}

    <div class="area-turn relative flex min-w-0 flex-col" dir={i18n.dir} bind:this={turnEl}>
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
         shows the "who starts" bar of a new age. Its bars sit here in the flow,
         just above the turn bar, so they never cover the takeable cards or this
         player's own strip. It waits while the pass-device screen is up,
         because its panels are modal dialogs that would cover that screen and
         make it inert. -->
      {#if view?.phase === 'gameOver'}
        <!-- Its modal stays in the top layer; its peek bar floats over the turn bar.
             It opens once the final move has played out on the board (the
             presenter is idle) and shows that move itself: the summary popup
             never opens at game over (M11.gameover-last-move), so it waits
             only behind an unread age summary or a summary opened from the
             top bar. -->
        <div class="contents" dir={i18n.dir}>
          <GameOver
            {view}
            show={!summaryOpen && !ageUnread}
            ready={pageVisible && presenter.idle}
            replayCode={frame?.code}
            {names}
            onrematch={rematchOffer ? () => session.rematch() : undefined}
            rematchLabel={rematchOffer?.label}
            rematchNote={rematchOffer?.note}
            {onhome}
          />
        </div>
      {/if}
      {#if view && view.phase !== 'wonderDraft' && (view.phase !== 'gameOver' || ageUnread) && !showPass}
        <ChoiceHost
          {view}
          {ageSummary}
          {ageViewer}
          showAge={pageVisible && presenter.idle && !summaryOpen}
          canAct={inputOpen}
          onact={act}
          {names}
          {confirm}
          {connected}
        />
      {/if}
      {#if ui.pickFor}
        <div
          class="flex min-h-10 items-center justify-between gap-2 border-t border-rule bg-accent-soft ps-3 pe-1 pb-(--safe-b) text-accent-soft-fg"
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
          hints={session.hints}
          hintsAdmin={!!frame?.you.admin}
          onRetry={() => session.retry()}
          class="pb-[max(0.25rem,var(--safe-b))]"
        />
      {/if}
    </div>
  </div>
{/if}

{#if view}
  <CardSheet
    open={cardSheetOpen}
    {view}
    slot={selectedSlot}
    seat={badge}
    opponentName={names[badge === 1 ? 0 : 1]}
    ownName={mode === 'hotseat' || spectator ? names[badge === 1 ? 1 : 0] : undefined}
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
    coach={cardNote}
    tutorial={!!tutorial}
    {allow}
    lockedReason={locked}
  />

  <WonderListSheet
    open={wonderListSeat !== null}
    name={names[wonderListShown]}
    seat={wonderListShown}
    wonders={view.players[wonderListShown].wonders}
    buildable={buildable[wonderListShown]}
    onselect={mySeats.includes(wonderListShown)
      ? (w) => selectListedWonder(wonderListShown, w)
      : undefined}
    onclose={() => (wonderListSeat = null)}
  />

  <WonderSheet
    open={ui.isOpen('wonder') && !!wonderOwner}
    onback={wonderFromList !== null ? backToWonderList : undefined}
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
    cardTooltips={view.cardTooltips}
    you={mode === 'hotseat' ? undefined : view.viewer}
    military={view.military}
    scores={view.scorePreview}
    bots={[seatBotLevel(frame?.room, 0), seatBotLevel(frame?.room, 1)]}
    bind:activeSeat={ui.citySeat}
  />

  <TokenSheet bind:open={() => tokenOpen && !tokenPick, (v) => (tokenOpen = v)} slug={tokenSlug} />

  <LogSheet
    bind:open={() => ui.isOpen('log'), (v) => !v && ui.close('log')}
    events={ui.log}
    {names}
    replay={{
      available: moveReplay.available,
      off: motion.effective === 'off',
      onreplay: replayLastMove
    }}
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
      summary={passLines}
      {canUndo}
      previousName={ui.holder !== null ? names[ui.holder] : undefined}
      onReady={() => ui.ready(view?.toMove)}
      onUndo={undoAllowed ? undo : undefined}
    />
  </div>
{/if}

<ForfeitConfirm
  open={forfeitOpen && resignSeat !== null}
  name={mode === 'hotseat' && resignSeat !== null ? names[resignSeat] : undefined}
  winner={names[resignWinner]}
  onconfirm={forfeit}
  onclose={() => (forfeitOpen = false)}
/>

<MoveSummary
  open={summaryOpen}
  summary={moveSummary}
  away={!summaryManual && summaryAway}
  {names}
  prices={summaryPrices}
  anchor={summaryAnchor}
  onclose={dismissSummary}
>
  {#snippet review()}
    {#if summaryReview}<MoveReview lines={summaryReview} />{/if}
  {/snippet}
</MoveSummary>

<FirstTips
  {view}
  seat={summarySeat}
  families={boardHints?.families ?? shownFamilies}
  boardMarks={!!boardHints?.boardMarks}
  {names}
  canShow={(turn.kind === 'yourTurn' || turn.kind === 'hotseatTurn') &&
    !summaryOpen &&
    !showPass &&
    !selectedSlot &&
    !tipsQuiet &&
    presenter.idle}
/>

<CoachLayer
  step={coach}
  text={coachWords}
  active={coachActive}
  onnext={coachNext}
  onpause={tutorial?.status === 'guided' ? pauseTutorial : undefined}
/>

<TutorialDone
  open={!!frame && doneRoom === frame.code}
  onkeep={() => (doneRoom = null)}
  onnew={tutorialNewGame}
  onhome={() => {
    doneRoom = null;
    onhome();
  }}
/>

<LiveRegion message={lastLines.join(' ')} />
<!-- What the move animations draw over the board (lib/fx/anim). -->
<FxLayer />
<!-- The next age's approved card art, fetched while this one is played. -->
<ArtPrefetch age={view?.age ?? null} />

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

  /* The pre-roll ring's colour: the bot's crest (StructureSlot and Draft draw
     the ring on what their botPick prop names). */
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
  /* A toast in its lane: compact, so a long skipped-choice sentence fits a
     lane beside the cards, while the dismiss button keeps its 44 px hit
     target. */
  .game-screen[data-layout='phone-landscape'] .toast-lane :global([data-toast]) {
    gap: 0.5rem;
    padding-inline: 0.75rem 0.5rem;
    padding-block: 0.375rem;
  }
  .game-screen[data-layout='phone-landscape'] .toast-lane :global([data-toast] p) {
    font-size: 0.8125rem;
    line-height: 1.3;
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
