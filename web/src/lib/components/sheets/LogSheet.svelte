<script lang="ts">
  // The structured game log (UX design sec. 5 "Log"): every event so far,
  // grouped by age, each line translated by lib/view/log-format.ts. The
  // caller owns accumulating `events` across updates (the view itself only
  // ever carries the latest action's events, not history); this component
  // just renders whatever list it is given.
  import { t } from '$lib/i18n';
  import type { GameEvent } from '$lib/wire';
  import { groupLogByAge } from '$lib/view/log-format';
  import Sheet from '../shell/Sheet.svelte';

  interface Props {
    open?: boolean;
    events: readonly GameEvent[];
    names: readonly [string, string];
    titleId?: string;
  }

  let { open = $bindable(false), events, names, titleId = 'log-sheet-title' }: Props = $props();

  const groups = $derived(groupLogByAge(events, names));
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
