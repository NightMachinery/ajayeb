<script lang="ts">
  // Haftgane's table screen (docs/haftgane/phone-layout.md, "The main
  // screen"): the seat ring around the anchor's strip and hand, between
  // the top bar and the turn bar. It renders the Presenter's frame and
  // sends the Session the server's keys: a pick stays changeable until it
  // is confirmed, and a confirmation can be withdrawn until the last seat
  // confirms. Nothing here decides what is legal or adds up a cost.
  //
  // The layout is CSS only. A phone in portrait stacks the far chips, the
  // two neighbour panels, the strip, the hand and the turn bar. A short
  // landscape screen puts the neighbours in side columns with the far
  // chips above them (the start column climbs away from the left
  // neighbour, the end column comes down to the right one), and a desktop
  // gives the neighbours full side panels and the far chips a row in the
  // middle. The grid and the seat ring are left to right in every locale,
  // because they are the table seen from the anchor's chair; the top bar,
  // the middle column, the notice and the menu follow the locale's
  // direction, as Duel's areas do.
  //
  // A spectator's ring centres on the seat they follow, and the spectator
  // panel takes the hand's place. An admin's seat menu opens from the game
  // menu or a city sheet; what each viewer may do comes from the frame.
  import '../theme.css';
  import type { Snippet } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { i18n } from '$lib/i18n';
  import type { TableState } from '$lib/wire/platform.gen';
  import { ht } from '../i18n/messages.svelte';
  import type { TablePresenter } from '../session/presenter.svelte';
  import type { TableSession } from '../session/session.svelte';
  import FarSeatChip from './FarSeatChip.svelte';
  import SeatMenuLayer from './SeatMenuLayer.svelte';
  import SpectatorPanel from './SpectatorPanel.svelte';
  import HandGrid from './HandGrid.svelte';
  import NeighbourPanel from './NeighbourPanel.svelte';
  import PassFlow from './PassFlow.svelte';
  import SidePicker from './SidePicker.svelte';
  import DiscardPicker from './DiscardPicker.svelte';
  import KeptCard from './KeptCard.svelte';
  import CardSheet from './CardSheet.svelte';
  import CitySheet from './CitySheet.svelte';
  import SeenHandsButton from './SeenHandsButton.svelte';
  import SeenHandsSheet from './SeenHandsSheet.svelte';
  import AgeMilitary from './AgeMilitary.svelte';
  import EndActions from './EndActions.svelte';
  import FinalReveal from './FinalReveal.svelte';
  import ScoreSheet from './ScoreSheet.svelte';
  import { downloadTableLog } from './download';
  import { conflictsByAge } from './end';
  import { settings } from '$lib/ui/settings.svelte';
  import TableMenu from './TableMenu.svelte';
  import TableTopBar from './TableTopBar.svelte';
  import TurnBar from './TurnBar.svelte';
  import YourStrip from './YourStrip.svelte';
  import { choiceLabel } from './labels';
  import {
    chosenCount,
    farColumns,
    humanAt,
    ring,
    roleOf,
    seatStatus,
    stageOpen,
    afford,
    cardChoices,
    TURNS_PER_AGE
  } from './model';
  import { formerOf, isAdmin, roleMap, watchers } from './admin';
  import { isBot, seatName, where } from './seat-names';

  interface Props {
    session: TableSession;
    presenter: TablePresenter;
    /** The frame on screen: the Presenter's, never null here. */
    frame: TableState;
    /** What follows the turn bar once the game is over, such as a rematch. */
    over?: Snippet;
  }

  let { session, presenter, frame, over }: Props = $props();

  const view = $derived(frame.view);
  const n = $derived(view.seats.length);
  // The seat a spectator follows; null centres on seat 0.
  let follow = $state<number | null>(null);
  const around = $derived(ring(view.seats, follow));
  const columns = $derived(farColumns(around?.far ?? []));
  const mine = $derived(view.you);
  const mySeat = $derived(frame.you.seat ?? null);
  const finished = $derived(frame.status === 'finished');
  // Input waits for the Presenter to catch up with the Session's frame.
  const ready = $derived(presenter.idle && frame === session.table);
  const deciding = $derived(
    !finished && !!frame.point && mySeat !== null && frame.point.required.includes(mySeat)
  );
  const canAct = $derived(ready && session.canPick);
  const hand = $derived(
    view.phase === 'last-card' && mine?.kept ? [mine.kept] : (mine?.hand ?? [])
  );
  const choices = $derived(mine?.choices ?? []);
  const pending = $derived(frame.pending ?? null);
  const pendingChoice = $derived(pending ? choices.find((c) => c.key === pending.key) : undefined);
  const picked = $derived(
    pending && pendingChoice && hand.includes(pendingChoice.design)
      ? { design: pendingChoice.design, confirmed: pending.confirm }
      : null
  );
  // Bind local selection and any armed confirmation to this exact point
  // incarnation and seat, including a control change that keeps its id.
  const decisionKey = $derived(
    frame.point && mySeat !== null
      ? `${frame.point.ref.id}@${frame.point.ref.revision}:${mySeat}`
      : null
  );
  // The selected hand card, for the decision it was chosen in only.
  let selection = $state<{ point: string; card: string } | null>(null);
  const selected = $derived(
    deciding && selection && selection.point === decisionKey && hand.includes(selection.card)
      ? selection.card
      : null
  );
  let dismissedSpecial = $state<string | null>(null);
  const special = $derived(view.phase === 'sides' || view.phase === 'discard-build');
  const showSpecial = $derived(
    deciding && special && !pending?.confirm && dismissedSpecial !== decisionKey
  );
  const publicDecider = $derived(frame.point?.required[0]);
  const specialWaiting = $derived(
    !deciding &&
      (view.phase === 'discard-build' || view.phase === 'last-card') &&
      publicDecider !== undefined
      ? ht(view.phase === 'discard-build' ? 'hg.special.wait.discard' : 'hg.special.wait.last', {
          name: seatName(frame, publicDecider)
        })
      : view.phase === 'sides'
        ? ht('hg.special.wait.sides')
        : ''
  );
  const names = $derived(
    Object.fromEntries(view.seats.map((s) => [s.seat, seatName(frame, s.seat)]))
  );
  const confirmMoves = $derived(settings.value.confirmMoves !== 'never');

  function select(card: string): void {
    const point = decisionKey;
    if (!point) return;
    if (pending?.confirm) session.withdraw();
    selection = selected === card ? null : { point, card };
  }

  function pick(key: string): void {
    if (pending?.key === key) return;
    session.pick(key);
  }

  function confirmCard(key: string): void {
    if (session.pick(key, true)) selection = null;
  }

  function change(): void {
    if (special) dismissedSpecial = null;
    if (session.withdraw() && pendingChoice && frame.point) {
      selection = { point: decisionKey!, card: pendingChoice.design };
    }
  }

  function confirm(): void {
    if (session.confirm()) selection = null;
  }

  let menuOpen = $state(false);
  // The city sheet's seat, and whether the Seen hands sheet is open.
  let citySeat = $state<number | null>(null);
  let seenOpen = $state(false);
  // The seat whose admin menu is open.
  let seatMenu = $state<number | null>(null);
  const admin = $derived(!finished && isAdmin(frame));
  const roleAt = (seat: number) => roleOf(humanAt(frame, seat));

  // The end of an Age and of the game (phone-layout.md, section 6). The
  // military dialog opens by itself while the newest conflicts are fresh,
  // that is while the batch on screen is the one that reported them, also
  // after a reload; closing it is local. The score sheet follows once the
  // game is over, and a row opens that seat's city over it. Both reopen on
  // request.
  const ages = $derived(conflictsByAge(session.history, frame.batch));
  const newest = $derived(ages.at(-1) ?? null);
  const fresh = $derived(newest !== null && newest.batch === frame.batch);
  let militaryClosed = $state<number | null>(null);
  let militaryAsked = $state(false);
  let militaryAge = $state<number | null>(null);
  const militaryOpen = $derived(
    newest !== null && (militaryAsked || (fresh && militaryClosed !== newest.batch))
  );
  const military = $derived(ages.find((a) => a.age === militaryAge) ?? newest);
  let scoresClosed = $state(false);
  const scoresOpen = $derived(finished && !!view.result && !militaryOpen && !scoresClosed);
  let revealOpen = $state(false);

  function openMilitary(): void {
    menuOpen = false;
    militaryAge = null;
    militaryAsked = true;
  }
  function closeMilitary(): void {
    militaryAsked = false;
    militaryAge = null;
    militaryClosed = newest?.batch ?? null;
  }

  const nameOf = (seat: number) => seatName(frame, seat);
  const passTo = $derived(
    !around
      ? null
      : view.pass === 'right'
        ? around.right
        : view.pass === 'left'
          ? around.left
          : null
  );
  const passFrom = $derived(
    !around
      ? null
      : view.pass === 'right'
        ? around.left
        : view.pass === 'left'
          ? around.right
          : null
  );
  const neighbourTrade = (direction: 'left' | 'right') =>
    mine?.neighbours.find((nb) => nb.direction === direction);

  const statusText = $derived.by(() => {
    if (finished) return ht('hg.screen.bar.over');
    if (mySeat === null) return ht('hg.screen.watching');
    if (deciding) {
      if (pending?.confirm) return ht('hg.screen.bar.confirmed');
      if (pending) return ht('hg.screen.bar.confirm');
      switch (view.phase) {
        case 'sides':
          return ht('hg.screen.bar.side');
        case 'discard-build':
          return ht('hg.screen.bar.discard');
        case 'last-card':
          return ht('hg.screen.bar.lastCard');
      }
      return ht('hg.screen.bar.choose');
    }
    const open = (frame.point?.required ?? []).filter((s) => !frame.point?.confirmed.includes(s));
    return open.length === 1
      ? ht('hg.screen.bar.waitingFor', { name: nameOf(open[0]!) })
      : ht('hg.screen.bar.waiting');
  });
  const pendingLabel = $derived(
    pendingChoice ? ht('hg.screen.pick.picked', { what: choiceLabel(pendingChoice) }) : ''
  );
</script>

<div
  class="screen"
  dir="ltr"
  data-table-screen
  data-seats={n}
  data-phase={view.phase}
  data-finished={finished ? '' : undefined}
>
  <div class="top" dir={i18n.dir}>
    <TableTopBar
      phase={view.phase}
      age={view.age}
      turn={view.turn}
      turns={TURNS_PER_AGE}
      pass={view.pass}
      passTo={passTo ? nameOf(passTo.seat) : ''}
      {finished}
      onmenu={() => (menuOpen = true)}
      onundo={ready && session.canUndo ? () => session.undo() : null}
      spectators={watchers(frame).length}
      onspectators={() => (menuOpen = true)}
    />
  </div>

  {#if session.notice}
    <div class="notice" role="alert" data-notice={session.notice.code} dir={i18n.dir}>
      <p dir="auto">
        {#if session.notice.kind === 'seated'}
          {ht('hg.table.seat.seated', { seat: session.notice.seat + 1 })}
        {:else if session.notice.kind === 'watching'}
          {ht('hg.table.seat.watching')}
        {:else if session.notice.kind === 'undo-off'}
          {ht('hg.admin.undoOff')}
        {:else}
          {ht(session.notice.message)}
        {/if}
      </p>
      <button
        type="button"
        class="dismiss"
        onclick={() => session.dismissNotice()}
        data-act="dismiss"
      >
        {ht('hg.screen.notice.dismiss')}
      </button>
    </div>
  {/if}

  {#if around}
    <div class="far" data-far-count={around.far.length}>
      {#each [columns.start, columns.end] as half, h (h)}
        <div class="half" class:start={h === 0} class:end={h === 1}>
          {#each half as s (s.seat)}
            <div class="slot">
              <FarSeatChip
                seat={s}
                name={nameOf(s.seat)}
                where={where(s, n)}
                bot={isBot(frame, s.seat)}
                status={seatStatus(frame, s.seat)}
                role={roleAt(s.seat)}
                onopen={() => (citySeat = s.seat)}
              />
            </div>
          {/each}
        </div>
      {/each}
    </div>

    {#each [around.left, around.right] as s, i (s.seat)}
      {@const side = i === 0 ? 'left' : 'right'}
      <div class="neighbour {side}">
        <NeighbourPanel
          seat={s}
          {side}
          name={nameOf(s.seat)}
          where={where(s, n)}
          bot={isBot(frame, s.seat)}
          status={seatStatus(frame, s.seat)}
          trade={neighbourTrade(side)}
          yourShields={around.own ? around.anchor.shields : 0}
          pass={view.pass}
          role={roleAt(s.seat)}
          onopen={() => (citySeat = s.seat)}
        />
      </div>
    {/each}

    <div class="mid" dir={i18n.dir}>
      <div class="strip">
        <YourStrip
          seat={around.anchor}
          name={nameOf(around.anchor.seat)}
          where={where(around.anchor, n)}
          own={around.own}
          bot={isBot(frame, around.anchor.seat)}
          status={seatStatus(frame, around.anchor.seat)}
          stageOpen={deciding && stageOpen(choices)}
          role={roleAt(around.anchor.seat)}
          onopen={() => (citySeat = around.anchor.seat)}
        />
      </div>
      {#if mine}
        <div class="hand-head">
          <h2>{ht(view.phase === 'last-card' ? 'hg.special.extra' : 'hg.screen.hand')}</h2>
          {#if passFrom && passTo && !finished}
            <PassFlow
              from={{ seat: passFrom.seat, name: nameOf(passFrom.seat) }}
              to={{ seat: passTo.seat, name: nameOf(passTo.seat) }}
              handSize={hand.length}
            />
          {/if}
          <SeenHandsButton onopen={() => (seenOpen = true)} />
        </div>
        <div class="hand">
          {#if view.phase === 'turn' && view.turn === TURNS_PER_AGE}
            <p class="special-note" data-sixth-turn>{ht('hg.special.sixth')}</p>
          {/if}
          {#if mine.kept}
            <KeptCard
              card={mine.kept}
              locked={!deciding || view.phase !== 'last-card'}
              afford={afford(cardChoices(mine.kept, choices, mine.disabled))}
              onselect={select}
            />
          {:else if hand.length > 0}
            <HandGrid
              {hand}
              {choices}
              disabled={mine.disabled}
              {selected}
              {picked}
              locked={!deciding}
              onselect={select}
            />
          {:else}
            <p class="empty">{ht('hg.screen.hand.empty')}</p>
          {/if}
        </div>
      {:else}
        <div class="hand">
          <SpectatorPanel
            seats={view.seats}
            {names}
            bots={view.seats.filter((s) => isBot(frame, s.seat)).map((s) => s.seat)}
            follow={around.anchor.seat}
            onfollow={(s) => (follow = s)}
            people={watchers(frame)}
            you={frame.you.userId}
            roles={roleMap(frame)}
            disabled={!ready || !session.canControl}
            onrole={(a) => session.role(a.kind, a.userId)}
            former={formerOf(frame, session.held)}
          />
        </div>
      {/if}
      {#if specialWaiting}<p class="special-note" role="status" data-special-wait>
          {specialWaiting}
        </p>{/if}
      {#if deciding && special && !showSpecial && !pending?.confirm}
        <button
          type="button"
          class="dismiss"
          data-act="special-open"
          onclick={() => (dismissedSpecial = null)}>{ht('hg.special.reopen')}</button
        >
      {/if}
      <div class="foot">
        <TurnBar
          text={statusText}
          chosen={chosenCount(frame)}
          pending={deciding ? pending : null}
          {pendingLabel}
          {canAct}
          watching={mySeat === null}
          onconfirm={confirm}
          onwithdraw={change}
        />
        {#if finished && over && !scoresOpen}
          {@render over()}
        {/if}
        {#if finished && view.result}
          <EndActions
            onscores={() => (scoresClosed = false)}
            onconflicts={newest ? openMilitary : undefined}
            onreveal={view.reveal ? () => (revealOpen = true) : undefined}
          />
        {/if}
      </div>
    </div>
  {/if}

  {#if showSpecial && mine && around}
    {#key decisionKey}
      {#if view.phase === 'sides'}
        <SidePicker
          boardId={around.anchor.board}
          {choices}
          pendingKey={pending?.key ?? null}
          locked={!canAct}
          ondraft={pick}
          onconfirm={confirmCard}
          onclose={() => (dismissedSpecial = decisionKey)}
        />
      {:else}
        <DiscardPicker
          offer={mine.offer}
          {choices}
          disabled={mine.disabled}
          pendingKey={pending?.key ?? null}
          locked={!canAct}
          ondraft={pick}
          onconfirm={confirmCard}
          onclose={() => (dismissedSpecial = decisionKey)}
        />
      {/if}
    {/key}
  {/if}
  {#if selected && mine && around}
    {#key `${decisionKey}:${selected}`}
      <CardSheet
        card={selected}
        {choices}
        disabled={mine.disabled}
        seat={around.anchor}
        neighbours={mine.neighbours}
        {names}
        pendingKey={pending?.key ?? null}
        locked={!canAct || !!pending?.confirm}
        {confirmMoves}
        ondraft={pick}
        onconfirm={confirmCard}
        onclose={() => (selection = null)}
      />
    {/key}
  {/if}

  {#if citySeat !== null}
    <CitySheet
      seats={view.seats}
      {names}
      bots={view.seats.filter((s) => isBot(frame, s.seat)).map((s) => s.seat)}
      statuses={Object.fromEntries(view.seats.map((s) => [s.seat, seatStatus(frame, s.seat)]))}
      own={mine ? { seat: mine.seat, tucked: mine.tucked, kept: mine.kept } : null}
      revealed={view.reveal?.tucked ?? null}
      active={citySeat}
      onselect={(s) => (citySeat = s)}
      onclose={() => (citySeat = null)}
      onmanage={admin ? (s) => (seatMenu = s) : undefined}
    />
  {/if}
  {#if seenOpen && mine}
    <SeenHandsSheet
      notebook={mine.notebook}
      age={view.age}
      {names}
      onclose={() => (seenOpen = false)}
    />
  {/if}

  {#if militaryOpen && military}
    <div dir={i18n.dir}>
      <AgeMilitary
        age={military.age}
        ages={ages.map((a) => a.age)}
        conflicts={military.conflicts}
        seats={view.seats}
        {names}
        you={mySeat}
        shieldsKnown={military.batch === frame.batch}
        latest={military.batch === newest?.batch}
        onage={(age) => (militaryAge = age)}
        onclose={closeMilitary}
      />
    </div>
  {/if}

  {#if scoresOpen && view.result}
    <div dir={i18n.dir}>
      <ScoreSheet
        result={view.result}
        seats={view.seats}
        {names}
        you={mySeat}
        canDownload={view.canDownload}
        ondownload={() => downloadTableLog(frame.code)}
        onhome={() => goto(resolve('/'))}
        onclose={() => (scoresClosed = true)}
        onseat={(s) => (citySeat = s)}
        rematch={over}
      />
    </div>
  {/if}

  {#if revealOpen && view.reveal}
    <div dir={i18n.dir}>
      <FinalReveal
        reveal={view.reveal}
        seats={view.seats}
        {names}
        onclose={() => (revealOpen = false)}
      />
    </div>
  {/if}

  {#if menuOpen}
    <div dir={i18n.dir}>
      <TableMenu
        {session}
        {frame}
        {ready}
        onmilitary={newest ? openMilitary : undefined}
        onclose={() => (menuOpen = false)}
        onmanage={(s) => (seatMenu = s)}
      />
    </div>
  {/if}
  {#if seatMenu !== null && admin}
    <div dir={i18n.dir}>
      <SeatMenuLayer {session} {frame} {ready} seat={seatMenu} onclose={() => (seatMenu = null)} />
    </div>
  {/if}
</div>

<style>
  .screen {
    position: relative;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    grid-template-rows: auto auto auto minmax(0, 1fr);
    grid-template-areas:
      'top top'
      'far far'
      'left right'
      'mid mid';
    gap: 0.4rem;
    height: 100%;
    min-height: 0;
    overflow: hidden;
    background: var(--table);
    color: var(--fg);
  }
  .top {
    grid-area: top;
  }
  .far {
    grid-area: far;
    display: flex;
    justify-content: center;
    gap: 0.4rem;
    padding-inline: 0.4rem;
  }
  .far:empty,
  .far[data-far-count='0'] {
    display: none;
  }
  .half {
    display: contents;
  }
  .slot {
    display: flex;
    flex: 1 1 0;
    min-width: 0;
    max-width: 11rem;
  }
  .neighbour {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .neighbour > :global(*) {
    flex: 1 1 auto;
  }
  .left {
    grid-area: left;
    padding-inline-start: 0.4rem;
  }
  .right {
    grid-area: right;
    padding-inline-end: 0.4rem;
  }
  .mid {
    grid-area: mid;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    min-width: 0;
    min-height: 0;
  }
  .strip {
    padding-inline: 0.4rem;
  }
  .hand-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding-inline: 0.6rem;
  }
  h2 {
    margin: 0;
    font-family: var(--type-display);
    font-size: 1rem;
    font-weight: 600;
  }
  .hand {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
    padding-inline: 0.4rem;
  }
  .special-note {
    margin: 0;
    padding: 0.4rem 0.6rem;
    color: var(--fg-muted);
    font-size: 0.8rem;
    line-height: 1.3;
  }
  .empty {
    margin: auto;
    color: var(--fg-muted);
  }
  /* The pick list takes its room from the hand, whose cards shrink to
     fit, so it covers nothing. */
  .pick {
    flex: 0 0 auto;
    padding-inline: 0.4rem;
  }
  .notice {
    position: absolute;
    inset-block-start: 3.2rem;
    inset-inline: 0.5rem;
    z-index: 30;
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.4rem 0.4rem 0.4rem 0.75rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow);
  }
  .notice p {
    flex: 1 1 auto;
    margin: 0;
  }
  .dismiss {
    min-width: 2.75rem;
    min-height: 2.75rem;
    padding-inline: 0.75rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel-sunken);
    color: var(--fg);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }

  /* A short landscape screen (Duel's short: variant): side columns. */
  @media (orientation: landscape) and (max-height: 32rem) {
    .screen {
      grid-template-columns: 10.5rem minmax(0, 1fr) 10.5rem;
      grid-template-rows: auto minmax(0, 1fr) auto;
      grid-template-areas:
        'top top top'
        'fars mid fare'
        'left mid right';
      gap: 0.3rem;
      padding-block-end: 0.3rem;
    }
    .far,
    .far[data-far-count='0'] {
      display: contents;
    }
    .half {
      display: flex;
      flex-direction: column;
      gap: 0.3rem;
      min-height: 0;
      overflow: hidden;
    }
    .half.start {
      grid-area: fars;
      flex-direction: column-reverse;
      padding-inline-start: 0.3rem;
    }
    .half.end {
      grid-area: fare;
      justify-content: flex-end;
      padding-inline-end: 0.3rem;
    }
    .slot {
      flex: 0 0 auto;
    }
    .left {
      padding-inline-start: 0.3rem;
    }
    .right {
      padding-inline-end: 0.3rem;
    }
    .strip,
    .hand,
    .pick {
      padding-inline: 0;
    }
  }

  /* A desktop (Duel's split: variant without the short case): full side
     panels, the far chips across the middle, and the hand above the strip. */
  @media (min-width: 64rem) and (min-height: 32.01rem) {
    .screen {
      grid-template-columns: 19rem minmax(0, 1fr) 19rem;
      grid-template-rows: auto auto minmax(0, 1fr);
      grid-template-areas:
        'top top top'
        'left far right'
        'left mid right';
      gap: 0.6rem;
    }
    .far {
      padding-inline: 0;
    }
    .slot {
      flex-basis: 11rem;
    }
    .neighbour {
      overflow-y: auto;
      padding-block-end: 0.6rem;
    }
    .neighbour > :global(*) {
      flex: 0 0 auto;
    }
    .mid {
      width: 100%;
      max-width: 44rem;
      justify-self: center;
    }
    .hand-head {
      order: 0;
    }
    .hand {
      order: 1;
    }
    .pick {
      order: 1;
    }
    .strip {
      order: 2;
    }
    .foot {
      order: 3;
    }
  }
</style>
