<script lang="ts">
  // The board's top bar (UX design sec. 3): the game menu, the age and how
  // many cards are left in it, then the undo, log and sound controls. A
  // plain leaf component -- it reads nothing from the session itself, the
  // caller passes the age, the count and whether undo is available right
  // now (view.canUndo) and gets a click back for each button.
  //
  // The game menu (every mode) holds Home, Settings and the log, "Undo last
  // move" in a game that allows undo (greyed out, with the caller's reason,
  // while undo is not possible now), Forfeit while the caller says a
  // forfeit is possible (it opens a confirm step; nothing is sent from
  // here), and the replay download (disabled until the game has ended). It is a native
  // popover: light dismiss, Escape and the top layer come with it. The
  // undo arrow in the bar itself does the same as the menu item.
  //
  // In a tutorial room the menu also pauses or resumes the tutorial
  // (data-control="menu-tutorial"); the caller passes tutorialPaused there.
  //
  // Next to the log, the eye opens the opponent move summary (M10.S03b);
  // the caller passes onOpenSummary only while there is a summary to show.
  // A direct sound switch replaces the settings gear, keeping mute one tap
  // away on phones too. Settings stays in the game menu.
  import StarterDiagram from '../rules/StarterDiagram.svelte';
  import type { Snippet } from 'svelte';
  import { ageLabel, t } from '$lib/i18n';
  import HintsMenuItem from '../hints/HintsMenuItem.svelte';
  import House from '@lucide/svelte/icons/house';
  import Menu from '@lucide/svelte/icons/menu';
  import Undo2 from '@lucide/svelte/icons/undo-2';
  import Settings from '@lucide/svelte/icons/settings';
  import Volume2 from '@lucide/svelte/icons/volume-2';
  import VolumeX from '@lucide/svelte/icons/volume-x';
  import ScrollText from '@lucide/svelte/icons/scroll-text';
  import Download from '@lucide/svelte/icons/download';
  import { downloadReplay } from '$lib/session/replay';
  import Eye from '@lucide/svelte/icons/eye';
  import Flag from '@lucide/svelte/icons/flag';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { restoreFocus } from '$lib/ui/dismiss';
  import BookOpen from '@lucide/svelte/icons/book-open';
  import GraduationCap from '@lucide/svelte/icons/graduation-cap';
  import { openRules } from '../rules/rules-overlay.svelte';

  interface Props {
    /** 1 to 3, or 0 during the wonder draft. */
    age: number;
    cardsLeft: number;
    canUndo?: boolean;
    /** Undo, in the bar and in the menu; absent in a game without undo. */
    onUndo?: () => void;
    /** Why undo is greyed out now (shown under the menu item while !canUndo). */
    undoReason?: string;
    onOpenLog?: () => void;
    /** Opens the opponent move summary; absent while there is none. */
    onOpenSummary?: () => void;
    onOpenSettings?: () => void;
    /** A direct board switch, including narrow phones. */
    sound?: boolean;
    onToggleSound?: () => void;
    /** Leaves the room for Home; when given, the game menu shows. */
    onHome?: () => void;
    /** Under Home, say the game can be continued from there (not once it is over). */
    homeHint?: boolean;
    /** The room code for the server's unchanged replay download. */
    replayCode?: string;
    /** When set, downloading is disabled and this explains why. */
    replayReason?: string;
    /** When given, the menu offers Forfeit, which reports this (the caller
     *  shows the confirm step). */
    onForfeit?: () => void;
    /** In a tutorial room, whether the tutorial is paused here: the menu
     *  then offers Resume tutorial, else Pause tutorial. Absent elsewhere. */
    tutorialPaused?: boolean;
    /** The tutorial menu item's action. */
    onTutorial?: () => void;
    /** The age chip's tooltip (age-countdown and next-starter, M10.HS4). */
    ageTip?: string;
    /** The presented view has an actual centred-pawn choice (the default ruling). */
    starterChoice?: boolean;
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
    undoReason,
    onOpenLog,
    onOpenSummary,
    onOpenSettings,
    sound = true,
    onToggleSound,
    onHome,
    homeHint = true,
    replayCode,
    replayReason,
    onForfeit,
    tutorialPaused,
    onTutorial,
    ageTip,
    starterChoice = false,
    children,
    class: className = ''
  }: Props = $props();

  const menuId = $props.id();
  let menu = $state<HTMLElement | undefined>();
  let menuOpener = $state<HTMLButtonElement | undefined>();

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

{#snippet starterDiagram()}<StarterDiagram compact />{/snippet}

<header
  class="top-bar flex min-h-11 items-center justify-between gap-2 border-b border-rule bg-panel px-2 {className}"
>
  <div class="flex min-w-0 items-center gap-1.5 text-sm text-fg">
    {#if onHome}
      <button
        type="button"
        popovertarget="game-menu-{menuId}"
        bind:this={menuOpener}
        data-control="menu"
        aria-label={t('game.menu.open')}
        {@attach tooltip({ text: t('game.menu.open') })}
        class="-ms-1 {iconButton}"
      >
        <Menu class="size-5" />
      </button>
    {/if}
    <span class="font-display font-semibold whitespace-nowrap" data-must-fit data-tut="age">
      {age > 0 ? t('board.age', { age: ageLabel(age) }) : t('board.age.draft')}
    </span>
    {#if age > 0}
      <span
        class="whitespace-nowrap text-fg-muted"
        data-must-fit
        data-age-tip={ageTip ? '' : undefined}
        {@attach tooltip(
          ageTip || starterChoice
            ? {
                text: ageTip ?? t('choices.starter.hint'),
                rulesId: 'age-end',
                diagram: starterChoice ? starterDiagram : undefined
              }
            : undefined
        )}
      >
        <span class="min-[26rem]:hidden">{t('cards.left.compact', { count: cardsLeft })}</span>
        <span class="max-[25.99rem]:hidden">{t('cards.left', { count: cardsLeft })}</span>
      </span>
    {/if}
  </div>

  {#if children}
    <div class="flex min-w-0 flex-1 items-center justify-center gap-2">{@render children()}</div>
  {/if}

  <div class="flex items-center gap-1">
    {#if onUndo}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex (This named tooltip group explains its native disabled button to keyboard users.) -->
      <span
        role="group"
        aria-label={t('board.topbar.undo')}
        class="inline-flex"
        tabindex={!canUndo ? 0 : undefined}
        data-undo-tooltip
        {@attach tooltip(
          !canUndo ? { text: undoReason ?? t('board.topbar.undo'), actionable: false } : undefined
        )}
      >
        <button
          type="button"
          onclick={onUndo}
          disabled={!canUndo}
          data-control="undo"
          aria-label={t('board.topbar.undo')}
          {@attach tooltip(canUndo ? { text: t('board.topbar.undo') } : undefined)}
          class="{iconButton} disabled:cursor-not-allowed disabled:opacity-35 disabled:hover:bg-transparent"
        >
          <Undo2 class="size-5 rtl:-scale-x-100" />
        </button>
      </span>
    {/if}
    {#if onOpenSummary}
      <button
        type="button"
        onclick={onOpenSummary}
        data-control="move-summary"
        aria-label={t('game.summary.open')}
        {@attach tooltip({ text: t('game.summary.open') })}
        class={iconButton}
      >
        <Eye class="size-5" />
      </button>
    {/if}
    {#if onOpenLog}
      <button
        type="button"
        onclick={onOpenLog}
        data-control="log"
        aria-label={t('board.topbar.log')}
        {@attach tooltip({ text: t('board.topbar.log') })}
        class={iconButton}
      >
        <ScrollText class="size-5" />
      </button>
    {/if}
    {#if onToggleSound}
      <button
        type="button"
        onclick={onToggleSound}
        data-control="sound"
        aria-pressed={sound}
        aria-label={t(sound ? 'game.sound.turnOff' : 'game.sound.turnOn')}
        {@attach tooltip({ text: t(sound ? 'game.sound.turnOff' : 'game.sound.turnOn') })}
        class={iconButton}
      >
        {#if sound}<Volume2 class="size-5" />{:else}<VolumeX class="size-5" />{/if}
      </button>
    {:else if onOpenSettings}
      <button
        type="button"
        onclick={onOpenSettings}
        data-control="settings"
        aria-label={t('board.topbar.settings')}
        {@attach tooltip({ text: t('board.topbar.settings') })}
        class="{iconButton} max-[25.99rem]:hidden"
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
    onbeforetoggle={(event) => {
      // Native closing restores focus synchronously. Quietly return it
      // before that operation so a focus tooltip cannot reenter popovers.
      if (event.newState === 'closed' && menuOpener && menu?.contains(document.activeElement)) {
        restoreFocus(menuOpener);
      }
    }}
    class="game-menu w-64 max-w-[calc(100vw-1rem)] rounded-panel border border-rule bg-panel p-1.5 text-fg shadow-lg"
    data-game-menu
    role="group"
    aria-label={t('game.menu.open')}
  >
    <button type="button" class={menuItem} data-control="menu-home" onclick={() => choose(onHome)}>
      <House class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
      <span class="flex min-w-0 flex-col py-1.5">
        <span>{t('game.menu.home')}</span>
        {#if homeHint}
          <span class="text-xs font-normal text-fg-muted" data-home-hint>
            {t('game.menu.homeHint')}
          </span>
        {/if}
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
    <HintsMenuItem class={menuItem} onclose={() => choose(undefined)} />
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
    {#if onUndo}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex (This named tooltip group explains its native disabled button to keyboard users.) -->
      <span
        role="group"
        aria-label={t('game.menu.undo')}
        class="block"
        tabindex={!canUndo ? 0 : undefined}
        data-undo-tooltip
        {@attach tooltip(
          !canUndo && undoReason ? { text: undoReason, actionable: false } : undefined
        )}
      >
        <button
          type="button"
          class="{menuItem} disabled:cursor-not-allowed disabled:text-fg-muted disabled:hover:bg-transparent"
          data-control="menu-undo"
          disabled={!canUndo}
          onclick={() => choose(onUndo)}
        >
          <Undo2 class="size-5 shrink-0 text-fg-muted rtl:-scale-x-100" aria-hidden="true" />
          <span class="flex min-w-0 flex-col py-1.5">
            <span>{t('game.menu.undo')}</span>
            {#if !canUndo && undoReason}
              <span class="text-xs font-normal text-fg-muted" data-undo-reason>{undoReason}</span>
            {/if}
          </span>
        </button>
      </span>
    {/if}
    {#if tutorialPaused !== undefined && onTutorial}
      <button
        type="button"
        class={menuItem}
        data-control="menu-tutorial"
        data-tutorial-paused={tutorialPaused ? '' : undefined}
        onclick={() => choose(onTutorial)}
      >
        <GraduationCap class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
        {tutorialPaused ? t('tutorial.menu.resume') : t('tutorial.menu.pause')}
      </button>
    {/if}
    <button
      type="button"
      class={menuItem}
      data-control="menu-rules"
      onclick={() => choose(() => void openRules())}
    >
      <BookOpen class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
      {t('rules.menu')}
    </button>
    {#if onForfeit}
      <button
        type="button"
        class="{menuItem} mt-1 border-t border-rule pt-1 text-cinnabar-ink"
        data-control="menu-forfeit"
        onclick={() => choose(onForfeit)}
      >
        <Flag class="size-5 shrink-0" aria-hidden="true" />
        {t('game.menu.forfeit')}
      </button>
    {/if}
    {#if replayCode}
      {@const code = replayCode}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex (The wrapper explains the disabled download to keyboard users.) -->
      <span
        role="group"
        aria-label={t('game.menu.replay')}
        class="block"
        tabindex={replayReason ? 0 : undefined}
        data-replay-tooltip
        {@attach tooltip(replayReason ? { text: replayReason, actionable: false } : undefined)}
      >
        <button
          type="button"
          class="{menuItem} disabled:cursor-not-allowed disabled:text-fg-muted disabled:hover:bg-transparent"
          data-control="menu-replay"
          disabled={!!replayReason}
          onclick={() => choose(() => downloadReplay(code))}
        >
          <Download class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
          {t('game.menu.replay')}
        </button>
      </span>
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
    max-height: calc(100dvh - var(--safe-t, 0px) - 3.25rem);
    overflow-y: auto;
  }
  .game-menu:not(:popover-open) {
    display: none;
  }
</style>
