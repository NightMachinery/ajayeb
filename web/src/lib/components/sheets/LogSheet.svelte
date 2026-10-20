<script lang="ts">
  // The structured game log (UX design sec. 5 "Log"): every event so far,
  // grouped by age, each line translated by lib/view/log-format.ts. On a
  // room page the log comes from the Session's history, which the server
  // re-sends whole on a join, a resync and an undo, so a reload still shows
  // the whole game in every mode. It stops at the frame the Presenter shows,
  // so a bot's answer that is still being paced is not in the log before it
  // is on the board. The changes of the game's hint settings (the room log)
  // are lines among the events. Without a game context (the gallery), or
  // before the Session has any history, it shows the `events` it is given.
  // The player's own lines (their seat's, outside hot-seat rooms) carry a
  // "you" mark and an accent edge: `selfSeat`, else the room's. The mark is
  // generated content (::before, from data-you, translated) rather than a
  // SelfMark element, so a line's text stays the log entry alone, the same
  // for every player; it still has text, so it is not colour alone.
  //
  // Each card, wonder and progress token a line names (LogLine.parts) has
  // the detail tooltip the move summary's pictures have (view/tips.ts
  // namedTip): a dotted underline marks it, a fine pointer hovers or
  // focuses it, and a tap shows it on touch (a line has no action). It
  // stays a span inside the sentence, never a button; a line that names
  // nothing is plain text.
  //
  // Replay last move (UX design section 5, "Log"; `replay`, from the game
  // screen): a control that plays the last move's animations again on the
  // board as it is now (fx/anim/replay.svelte.ts), closing the sheet first.
  // It changes nothing. It is disabled, with a tooltip saying why, while
  // Animation is off or before any move has animated.
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
  import type { Attachment } from 'svelte/attachments';
  import { ageLabel, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { GameEvent } from '$lib/wire';
  import { groupLogByAge, type LogPart } from '$lib/view/log-format';
  import { namedTip } from '$lib/view/tips';
  import { groupGameLog, type GameLogGroup } from '$lib/hints/log';
  import { tryGameContext } from '../choices/room';
  import Sheet from '../shell/Sheet.svelte';

  interface Props {
    open?: boolean;
    events: readonly GameEvent[];
    names: readonly [string, string];
    titleId?: string;
    /** The player's own seat, whose lines are marked; by default the room's. */
    selfSeat?: 0 | 1 | null;
    /** Current age when a fixture has no ageStart event. */
    age?: number;
    /** Replay last move: whether a move is on offer, whether Animation is
     *  off, and what replays it. Absent outside a game (the gallery). */
    replay?: { available: boolean; off: boolean; onreplay: () => void };
  }

  let {
    open = $bindable(false),
    events,
    names,
    titleId = 'log-sheet-title',
    selfSeat,
    age = 0,
    replay
  }: Props = $props();

  // Why the replay control is disabled, for its tooltip; none while it works.
  const replayReason = $derived(
    !replay
      ? undefined
      : replay.off
        ? t('board.logSheet.replayOff')
        : !replay.available
          ? t('board.logSheet.replayNone')
          : undefined
  );

  const ctx = tryGameContext();
  const mine = $derived.by(() => {
    if (selfSeat !== undefined) return selfSeat;
    const f = ctx?.session.frame;
    if (!f || f.room.mode === 'hotseat' || f.you.seats.length !== 1) return null;
    const seat = f.you.seats[0];
    return seat === 0 || seat === 1 ? seat : null;
  });

  // The room log (changes of the hint settings) goes among the events, at
  // the ply each change happened (lib/hints/log.ts).
  const groups: GameLogGroup[] = $derived.by(() => {
    const history = ctx?.session.history ?? [];
    const roomLog = ctx?.session.roomLog ?? [];
    const currentAge = ctx?.presenter.view?.age ?? age;
    if (history.length === 0 && roomLog.length === 0)
      return groupLogByAge(events, names, currentAge);
    return groupGameLog(history, roomLog, names, ctx?.presenter.view?.ply, currentAge);
  });

  // The lines are in the DOM only while the sheet shows: open, or sliding
  // away. A long game's log would otherwise weigh on every board frame, and
  // dropping them at the close itself would shrink the sheet to its header
  // while its exit still plays. `showLines` turns on when the log opens and
  // stays on through the close; the dialog's close turns it off once the
  // exit transitions are done (at once with Animation off, which has none).
  // Reopening mid-exit keeps the lines. `opened` is the plain memory the
  // derived reads every time `open` changes, so it needs no $state write.
  let opened = false;
  let showLines = $derived.by(() => {
    if (open) opened = true;
    return opened;
  });

  const settleAfterExit: Attachment<HTMLElement> = (node) => {
    const dialog = node.closest('dialog');
    if (!dialog) return;
    const onclose = (): void => {
      const exits = dialog
        .getAnimations({ subtree: true })
        .filter((a) => a instanceof CSSTransition)
        .map((a) => a.finished);
      void Promise.allSettled(exits).then(() => {
        if (open || dialog.open) return;
        opened = false;
        showLines = false;
      });
    };
    dialog.addEventListener('close', onclose);
    return () => dialog.removeEventListener('close', onclose);
  };
</script>

{#snippet piece(part: LogPart)}
  {#if typeof part === 'string'}
    {part}
  {:else}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex (Keyboard users reach a named item's tooltip by focus; the line has no action, so it is not a button.) -->
    <span
      class="cursor-help rounded-sm underline decoration-fg-muted decoration-dotted decoration-1 underline-offset-[0.25em] hover:decoration-fg hover:decoration-solid"
      dir="auto"
      tabindex="0"
      data-log-item={part.kind}
      data-log-slug={part.slug}
      {@attach tooltip(namedTip(part.kind, part.slug, names))}>{part.text}</span
    >
  {/if}
{/snippet}

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-5" {@attach settleAfterExit}>
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 id={titleId} class="font-display text-xl text-fg">{t('board.logSheet.title')}</h2>
      {#if replay}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex (This named tooltip group explains its native disabled button to keyboard users.) -->
        <span
          role="group"
          aria-label={t('board.logSheet.replay')}
          class="inline-flex"
          tabindex={replayReason ? 0 : undefined}
          data-replay-tooltip
          {@attach tooltip(replayReason ? { text: replayReason, actionable: false } : undefined)}
        >
          <button
            type="button"
            class="inline-flex min-h-11 items-center gap-1.5 rounded-control border border-rule-strong bg-panel px-3 text-sm font-semibold text-fg
              hover:bg-panel-sunken disabled:cursor-not-allowed disabled:opacity-45 disabled:hover:bg-panel"
            data-control="replay-last-move"
            disabled={!!replayReason}
            onclick={() => replay?.onreplay()}
          >
            <RotateCcw class="size-4 rtl:-scale-x-100" aria-hidden="true" />
            {t('board.logSheet.replay')}
          </button>
        </span>
      {/if}
    </div>
    {#if !showLines}
      <!-- Fully closed: no lines (see settleAfterExit). -->
    {:else if groups.length === 0}
      <p class="text-sm text-fg-muted">{t('board.logSheet.empty')}</p>
    {:else}
      {#each groups as group (group.age)}
        <section class="flex flex-col gap-1.5">
          <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
            {group.age > 0
              ? t('board.logSheet.age', { age: ageLabel(group.age) })
              : t('board.logSheet.draftAge')}
          </h3>
          <ol class="flex flex-col gap-1">
            {#each group.lines as line, i (i)}
              {@const self = mine !== null && line.seat === mine}
              {@const parts = 'parts' in line ? line.parts : undefined}
              <li
                class="rounded-control border border-rule bg-panel px-2.5 py-1.5 text-sm text-fg
                  {self
                  ? 'border-s-4 border-s-accent before:me-1.5 before:inline-block before:rounded-full before:bg-accent before:px-1.5 before:align-[0.1em] before:text-[0.625rem] before:leading-4 before:font-semibold before:tracking-wide before:text-on-accent before:uppercase before:content-[attr(data-you)]'
                  : ''}"
                data-self-line={self ? '' : undefined}
                data-you={self ? t('online.identity.you') : undefined}
              >
                {#if parts}
                  {#each parts as part, j (j)}{@render piece(part)}{/each}
                {:else}
                  {line.text}
                {/if}
              </li>
            {/each}
          </ol>
        </section>
      {/each}
    {/if}
  </div>
</Sheet>
