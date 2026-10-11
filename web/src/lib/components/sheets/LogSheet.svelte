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
  import { t } from '$lib/i18n';
  import type { GameEvent } from '$lib/wire';
  import { groupLogByAge } from '$lib/view/log-format';
  import { groupGameLog, type GameLogGroup } from '$lib/hints/log';
  import { tryGameContext } from '../choices/room';
  import Sheet from '../shell/Sheet.svelte';

  interface Props {
    open?: boolean;
    events: readonly GameEvent[];
    names: readonly [string, string];
    titleId?: string;
  }

  let { open = $bindable(false), events, names, titleId = 'log-sheet-title' }: Props = $props();

  const ctx = tryGameContext();

  // The room log (changes of the hint settings) goes among the events, at
  // the ply each change happened (lib/hints/log.ts).
  const groups: GameLogGroup[] = $derived.by(() => {
    const history = ctx?.session.history ?? [];
    const roomLog = ctx?.session.roomLog ?? [];
    if (history.length === 0 && roomLog.length === 0) return groupLogByAge(events, names);
    return groupGameLog(history, roomLog, names, ctx?.presenter.view?.ply);
  });
</script>

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-5">
    <h2 id={titleId} class="font-display text-xl text-fg">{t('board.logSheet.title')}</h2>
    {#if groups.length === 0}
      <p class="text-sm text-fg-muted">{t('board.logSheet.empty')}</p>
    {:else}
      {#each groups as group (group.age)}
        <section class="flex flex-col gap-1.5">
          <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
            {group.age > 0
              ? t('board.logSheet.age', { age: group.age })
              : t('board.logSheet.draftAge')}
          </h3>
          <ol class="flex flex-col gap-1">
            {#each group.lines as line, i (i)}
              <li class="rounded-control border border-rule bg-panel px-2.5 py-1.5 text-sm text-fg">
                {line.text}
              </li>
            {/each}
          </ol>
        </section>
      {/each}
    {/if}
  </div>
</Sheet>
