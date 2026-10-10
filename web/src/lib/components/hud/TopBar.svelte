<script lang="ts">
  // The board's top bar (UX design sec. 3): the game menu, the age and how
  // many cards are left in it, then the undo, log and settings controls. A
  // plain leaf component -- it reads nothing from the session itself, the
  // caller passes the age, the count and whether undo is available right
  // now (view.canUndo) and gets a click back for each button.
  //
  // The game menu (every mode) holds Home, Settings and the log. It is a
  // native popover: light dismiss, Escape and the top layer come with it.
  import type { Snippet } from 'svelte';
  import { t } from '$lib/i18n';
  import House from '@lucide/svelte/icons/house';
  import Menu from '@lucide/svelte/icons/menu';
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
    /** Leaves the room for Home; when given, the game menu shows. */
    onHome?: () => void;
    /** Extra content between the age and the buttons (the compact military
     *  track and tokens of the phone landscape layout). */
    children?: Snippet;
    class?: string;
  }

  let {
    age,
    cardsLeft,
    canUndo = false,
    onUndo,
    onOpenLog,
    onOpenSettings,
    onHome,
    children,
    class: className = ''
  }: Props = $props();

  const menuId = $props.id();
  let menu = $state<HTMLElement | undefined>();

  /** Closes the menu, then runs the item's action. */
  function choose(action: (() => void) | undefined): void {
    try {
      menu?.hidePopover();
    } catch {
      // Already closed.
    }
    action?.();
  }

  const iconButton =
    'grid size-11 shrink-0 place-items-center rounded-control text-fg transition-colors hover:bg-panel-sunken';
  const menuItem =
    'flex min-h-11 w-full items-center gap-3 rounded-control px-3 text-start text-sm font-semibold text-fg hover:bg-panel-sunken focus-visible:bg-panel-sunken';
</script>

<header
  class="top-bar flex min-h-11 items-center justify-between gap-2 border-b border-rule bg-panel px-2 {className}"
>
  <div class="flex min-w-0 items-center gap-1.5 text-sm text-fg">
    {#if onHome}
      <button
        type="button"
        popovertarget="game-menu-{menuId}"
        data-control="menu"
        aria-label={t('game.menu.open')}
        title={t('game.menu.open')}
        class="-ms-1 {iconButton}"
      >
        <Menu class="size-5" />
      </button>
    {/if}
    <span class="font-display font-semibold whitespace-nowrap" data-must-fit>
      {age > 0 ? t('board.age', { age }) : t('board.age.draft')}
    </span>
    {#if age > 0}
      <span class="whitespace-nowrap text-fg-muted" data-must-fit>
        {t('cards.left', { count: cardsLeft })}
      </span>
    {/if}
  </div>

  {#if children}
    <div class="flex min-w-0 flex-1 items-center justify-center gap-2">{@render children()}</div>
  {/if}

  <div class="flex items-center gap-1">
    {#if onUndo}
      <button
        type="button"
        onclick={onUndo}
        disabled={!canUndo}
        data-control="undo"
        aria-label={t('board.topbar.undo')}
        title={t('board.topbar.undo')}
        class="{iconButton} disabled:cursor-not-allowed disabled:opacity-35 disabled:hover:bg-transparent"
      >
        <Undo2 class="size-5" />
      </button>
    {/if}
    {#if onOpenLog}
      <button
        type="button"
        onclick={onOpenLog}
        data-control="log"
        aria-label={t('board.topbar.log')}
        title={t('board.topbar.log')}
        class={iconButton}
      >
        <ScrollText class="size-5" />
      </button>
    {/if}
    {#if onOpenSettings}
      <button
        type="button"
        onclick={onOpenSettings}
        data-control="settings"
        aria-label={t('board.topbar.settings')}
        title={t('board.topbar.settings')}
        class={iconButton}
      >
        <Settings class="size-5" />
      </button>
    {/if}
  </div>
</header>

{#if onHome}
  <div
    id="game-menu-{menuId}"
    popover="auto"
    bind:this={menu}
    class="game-menu w-64 max-w-[calc(100vw-1rem)] rounded-panel border border-rule bg-panel p-1.5 text-fg shadow-lg"
    data-game-menu
    role="group"
    aria-label={t('game.menu.open')}
  >
    <button type="button" class={menuItem} data-control="menu-home" onclick={() => choose(onHome)}>
      <House class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
      <span class="flex min-w-0 flex-col py-1.5">
        <span>{t('game.menu.home')}</span>
        <span class="text-xs font-normal text-fg-muted">{t('game.menu.homeHint')}</span>
      </span>
    </button>
    {#if onOpenSettings}
      <button
        type="button"
        class={menuItem}
        data-control="menu-settings"
        onclick={() => choose(onOpenSettings)}
      >
        <Settings class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
        {t('game.menu.settings')}
      </button>
    {/if}
    {#if onOpenLog}
      <button
        type="button"
        class={menuItem}
        data-control="menu-log"
        onclick={() => choose(onOpenLog)}
      >
        <ScrollText class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
        {t('game.menu.log')}
      </button>
    {/if}
  </div>
{/if}

<style>
  /* The menu drops from the top bar at its start edge, under the menu button. */
  .game-menu {
    position: fixed;
    inset: auto;
    margin: 0;
    top: calc(var(--safe-t, 0px) + 2.875rem);
    inset-inline-start: 0.375rem;
  }
  .game-menu:not(:popover-open) {
    display: none;
  }
</style>
