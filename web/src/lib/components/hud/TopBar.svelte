<script lang="ts">
  // The board's top bar (UX design sec. 3): the age and how many cards are
  // left in it, then the log, settings and undo controls. A plain leaf
  // component -- it reads nothing from the session itself, the caller
  // passes the age, the count and whether undo is available right now
  // (view.canUndo) and gets a click back for each button.
  import { t } from '$lib/i18n';
  import Undo2 from '@lucide/svelte/icons/undo-2';
  import Settings from '@lucide/svelte/icons/settings';
  import ScrollText from '@lucide/svelte/icons/scroll-text';

  interface Props {
    /** 1 to 3, or 0 during the wonder draft. */
    age: number;
    cardsLeft: number;
    canUndo?: boolean;
    onUndo?: () => void;
    onOpenLog?: () => void;
    onOpenSettings?: () => void;
    class?: string;
  }

  let {
    age,
    cardsLeft,
    canUndo = false,
    onUndo,
    onOpenLog,
    onOpenSettings,
    class: className = ''
  }: Props = $props();
</script>

<header
  class="top-bar flex min-h-11 items-center justify-between gap-2 border-b border-rule bg-panel px-2 {className}"
>
  <div class="flex min-w-0 items-center gap-2 text-sm text-fg">
    <span class="font-display font-semibold" data-must-fit>
      {age > 0 ? t('board.age', { age }) : t('board.age.draft')}
    </span>
    {#if age > 0}
      <span class="text-fg-muted" data-must-fit>{t('cards.left', { count: cardsLeft })}</span>
    {/if}
  </div>

  <div class="flex items-center gap-1">
    {#if onUndo}
      <button
        type="button"
        onclick={onUndo}
        disabled={!canUndo}
        aria-label={t('board.topbar.undo')}
        title={t('board.topbar.undo')}
        class="grid size-11 place-items-center rounded-control text-fg transition-colors hover:bg-panel-sunken disabled:cursor-not-allowed disabled:opacity-35 disabled:hover:bg-transparent"
      >
        <Undo2 class="size-5" />
      </button>
    {/if}
    {#if onOpenLog}
      <button
        type="button"
        onclick={onOpenLog}
        aria-label={t('board.topbar.log')}
        title={t('board.topbar.log')}
        class="grid size-11 place-items-center rounded-control text-fg transition-colors hover:bg-panel-sunken"
      >
        <ScrollText class="size-5" />
      </button>
    {/if}
    {#if onOpenSettings}
      <button
        type="button"
        onclick={onOpenSettings}
        aria-label={t('board.topbar.settings')}
        title={t('board.topbar.settings')}
        class="grid size-11 place-items-center rounded-control text-fg transition-colors hover:bg-panel-sunken"
      >
        <Settings class="size-5" />
      </button>
    {/if}
  </div>
</header>
