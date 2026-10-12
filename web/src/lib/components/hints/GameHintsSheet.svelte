<script lang="ts">
  // The game's hint settings, mid-game (docs/dev/decisions.md D23), from
  // the game menu's Hints item: the admin edits them, which sends a hints
  // frame and saves them as this device's local default; everyone else
  // sees them read-only. Under them, "On my screen" (the local hide), for
  // every player. While a change is on its way the editor shows it and
  // waits; the server's next frame is the truth. In a hot-seat or bot game
  // the game's undo setting follows, read-only for everyone: the server
  // fixed it when the game was created.
  import { t } from '$lib/i18n';
  import { hintDefaults, hintHide } from '$lib/hints/local.svelte';
  import type { HintSettings } from '$lib/wire';
  import { tryGameContext } from '../choices/room';
  import Sheet from '../shell/Sheet.svelte';
  import HintLocalHide from './HintLocalHide.svelte';
  import HintSettingsEditor from './HintSettingsEditor.svelte';

  interface Props {
    open?: boolean;
  }

  let { open = $bindable(false) }: Props = $props();

  const ctx = tryGameContext();
  const session = ctx?.session ?? null;
  const titleId = 'game-hints-title';

  const sent = $derived(session?.pending?.frame.t === 'hints' ? session.pending.frame.hints : null);
  const shown = $derived(sent ?? session?.hints ?? null);
  const editable = $derived(!!session?.admin && (!!sent || !!session?.canSetHints));
  // Online games have no undo, so there is nothing to show for them.
  const undoShown = $derived(!!session?.room && session.room.mode !== 'online');

  function change(next: HintSettings): void {
    if (!session) return;
    const mode = session.room?.mode;
    if (session.setHints(next) && mode) hintDefaults.save(mode, next);
  }
</script>

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-5 px-5 pt-1 pb-5 split:pt-5" data-game-hints>
    <h2 id={titleId} class="font-display text-xl text-fg">{t('hints.sheet.title')}</h2>
    {#if shown}
      <HintSettingsEditor
        hints={shown}
        readonly={!editable}
        readonlyNote={!session?.admin}
        busy={!!sent}
        onchange={change}
      />
    {/if}
    {#if undoShown}
      <section
        class="flex flex-col gap-1 rounded-control border border-rule bg-panel px-3 py-2.5"
        aria-labelledby="{titleId}-undo"
        data-undo-setting={session?.allowUndo ? 'on' : 'off'}
      >
        <h3 id="{titleId}-undo" class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
          {t('hints.sheet.undoTitle')}
        </h3>
        <p class="text-sm font-semibold text-fg">
          {session?.allowUndo ? t('hints.sheet.undoOn') : t('hints.sheet.undoOff')}
        </p>
        <p class="text-xs text-fg-muted">{t('hints.sheet.undoFixed')}</p>
      </section>
    {/if}
    <HintLocalHide
      hide={hintHide.value}
      onfamily={(family, hidden) => hintHide.setFamily(family, hidden)}
      onboardmarks={(hidden) => hintHide.setBoardMarks(hidden)}
    />
  </div>
</Sheet>
