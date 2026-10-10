<script lang="ts">
  // /dev/gallery: a visual check for every hand-authored icon and shell
  // primitive, plus one stub section per upcoming component slice (M5.S02
  // cards, M5.S03 board, M5.S05 screens) and the M5.S04a game screen. Dev-only; see
  // src/lib/dev/flags.ts and src/routes/dev/+layout.ts.
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import { ICON_NAMES } from '$lib/icons/names';
  import Icon from '$lib/icons/Icon.svelte';
  import Board from '$lib/dev/gallery/Board.svelte';
  import Cards from '$lib/dev/gallery/Cards.svelte';
  import Choices from '$lib/dev/gallery/Choices.svelte';
  import Game from '$lib/dev/gallery/Game.svelte';
  import Ornaments from '$lib/dev/gallery/Ornaments.svelte';
  import Screens from '$lib/dev/gallery/Screens.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import ConfirmButton from '$lib/components/shell/ConfirmButton.svelte';
  import LiveRegion from '$lib/components/shell/LiveRegion.svelte';
  import Segmented from '$lib/components/shell/Segmented.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import ToastList from '$lib/components/shell/Toasts.svelte';
  import { Toasts } from '$lib/components/shell/toast-queue.svelte';
  import { theme, THEME_PREFS, type ThemePref } from '$lib/ui/theme.svelte';
  import { innerHeight, innerWidth } from 'svelte/reactivity/window';

  const themeOptions = THEME_PREFS.map((pref) => ({ value: pref, label: t(`theme.${pref}`) }));

  const toasts = new Toasts();

  let sheetOpen = $state(false);
  let firstPlayer = $state('random');
  let liveMessage = $state('');
  let announceCount = 0;
</script>

<main class="mx-auto flex h-full max-w-3xl flex-col gap-10 overflow-y-auto px-4 py-6 pb-16">
  <header class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="font-display text-3xl text-fg">{t('dev.gallery.title')}</h1>
      <p class="mt-1 max-w-prose text-sm text-fg-muted">{t('dev.gallery.intro')}</p>
      <p class="mt-1 text-xs text-fg-muted" data-testid="gallery-viewport">
        {t('dev.gallery.viewport', {
          width: String(innerWidth.current ?? 0),
          height: String(innerHeight.current ?? 0)
        })}
      </p>
    </div>
    <Segmented
      name="gallery-theme"
      legend={t('theme.label')}
      options={themeOptions}
      bind:value={() => theme.pref, (pref) => theme.set(pref as ThemePref)}
    />
  </header>

  <section class="flex flex-col gap-3">
    <h2 class="font-display text-xl text-fg">
      {t('dev.gallery.icons', { count: ICON_NAMES.length })}
    </h2>
    <div
      class="grid grid-cols-[repeat(auto-fill,minmax(5.5rem,1fr))] gap-3 rounded-panel border border-rule bg-panel p-4"
    >
      {#each ICON_NAMES as name (name)}
        <div class="flex flex-col items-center gap-1 rounded-control p-2 text-center">
          <Icon {name} class="text-[28px] text-fg" />
          <code class="text-[10px] leading-tight break-all text-fg-muted" data-min-font="10"
            >{name}</code
          >
        </div>
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2" data-testid="gallery-ornaments-section">
    <h2 class="font-display text-xl text-fg">{t('dev.gallery.ornaments')}</h2>
    <Ornaments />
  </section>

  <section class="flex flex-col gap-4">
    <h2 class="font-display text-xl text-fg">{t('dev.gallery.shell')}</h2>

    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-4">
      <Button variant="primary" onclick={() => (sheetOpen = true)}>
        {t('shell.demo.openSheet')}
      </Button>
      <ConfirmButton
        label={t('shell.demo.confirmLabel')}
        onConfirm={() =>
          toasts.show(t('shell.demo.toastMessage'), {
            kind: 'success',
            actionLabel: t('shell.demo.toastUndo')
          })}
      />
      <Button
        variant="secondary"
        onclick={() =>
          toasts.show(t('shell.demo.toastMessage'), {
            actionLabel: t('shell.demo.toastUndo')
          })}
      >
        {t('shell.demo.showToast')}
      </Button>
      <Button
        variant="secondary"
        onclick={() => {
          announceCount += 1;
          liveMessage = `${t('shell.demo.announced')} (${announceCount})`;
        }}
      >
        {t('shell.demo.announce')}
      </Button>
    </div>

    <Segmented
      name="gallery-first-player"
      legend={t('shell.demo.segmentedLegend')}
      options={[
        { value: 'random', label: t('shell.demo.optionRandom') },
        { value: 'me', label: t('shell.demo.optionMe') },
        { value: 'rival', label: t('shell.demo.optionRival') }
      ]}
      bind:value={firstPlayer}
    />

    <p class="text-sm text-fg-muted" aria-hidden="true">{liveMessage || ' '}</p>
  </section>

  <section class="flex flex-col gap-2">
    <h2 class="font-display text-xl text-fg">{t('dev.gallery.cards')}</h2>
    <Cards />
  </section>

  <section class="flex flex-col gap-2">
    <h2 class="font-display text-xl text-fg">{t('dev.gallery.board')}</h2>
    <Board />
  </section>

  <section class="flex flex-col gap-2">
    <h2 class="font-display text-xl text-fg">{t('dev.gallery.screens')}</h2>
    <Screens />
  </section>

  <section class="flex flex-col gap-2">
    <h2 class="font-display text-xl text-fg">{t('dev.gallery.choices')}</h2>
    <Choices />
  </section>

  <section class="flex flex-col gap-2">
    <h2 class="font-display text-xl text-fg">{t('game.gallery.title')}</h2>
    <Game />
  </section>

  <p>
    <a
      class="inline-flex min-h-11 items-center font-semibold text-fg underline decoration-gild underline-offset-2"
      href={resolve('/dev')}
    >
      {t('app.home')}
    </a>
  </p>
</main>

<Sheet bind:open={sheetOpen} titleId="gallery-sheet-title">
  <div class="flex flex-col gap-4 px-5 pt-1 pb-5">
    <h2 id="gallery-sheet-title" class="font-display text-xl text-fg">
      {t('shell.demo.sheetTitle')}
    </h2>
    <p class="text-sm text-fg-muted">{t('shell.demo.sheetBody')}</p>
    <Button onclick={() => (sheetOpen = false)}>{t('shell.close')}</Button>
  </div>
</Sheet>

<ToastList {toasts} />
<LiveRegion message={liveMessage} />
