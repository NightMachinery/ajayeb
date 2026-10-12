<script lang="ts">
  // The Settings controls (UX design §2's Settings table), shared by the
  // sheet (opened from Home) and the /settings route: both just place this
  // component in their own shell. It reads the app's global UI state
  // directly (theme, settings), the same way Home already reads
  // theme.svelte.ts, rather than taking it all as props: there is exactly
  // one of each, for the life of the app, and every caller wants the same
  // live one.
  import { t, type MessageKey } from '$lib/i18n';
  import Segmented from '$lib/components/shell/Segmented.svelte';
  import {
    ANIMATION_SPEEDS,
    CARD_TEXT_SIZES,
    CONFIRM_MOVES_OPTIONS,
    SOUND_VOLUMES,
    settings,
    type AnimationSpeed,
    type CardTextSize,
    type ConfirmMoves,
    type SoundVolume
  } from '$lib/ui/settings.svelte';
  import { installFx, play, unlockAudio } from '$lib/fx';
  import { vibrationShown } from './vibration';
  import { theme, THEME_PREFS, type ThemePref } from '$lib/ui/theme.svelte';
  import { tips } from '$lib/ui/tips.svelte';
  import { hintHide } from '$lib/hints/local.svelte';
  import HintLocalHide from '$lib/components/hints/HintLocalHide.svelte';

  // as const satisfies (not a type annotation) so each entry keeps its own
  // literal key type; t() needs that to know no params are required for it
  // (see $lib/i18n's MessageParams: it cannot tell for the widened MessageKey).
  const SPEED_LABEL = {
    normal: 'home.settings.speedNormal',
    fast: 'home.settings.speedFast',
    relaxed: 'home.settings.speedRelaxed',
    off: 'home.settings.speedOff'
  } as const satisfies Record<AnimationSpeed, MessageKey>;
  const CONFIRM_LABEL = {
    onlineOnly: 'home.settings.confirmOnlineOnly',
    always: 'home.settings.confirmAlways',
    never: 'home.settings.confirmNever'
  } as const satisfies Record<ConfirmMoves, MessageKey>;
  const CARD_TEXT_LABEL = {
    normal: 'home.settings.cardTextNormal',
    large: 'home.settings.cardTextLarge'
  } as const satisfies Record<CardTextSize, MessageKey>;

  const VOLUME_LABEL = {
    quiet: 'home.settings.volumeQuiet',
    medium: 'home.settings.volumeMedium',
    loud: 'home.settings.volumeLoud'
  } as const satisfies Record<SoundVolume, MessageKey>;

  // Only on a touch device that can vibrate (not iOS, not a desktop, even
  // one whose browser has navigator.vibrate); read once, it never changes
  // during a page's life.
  const canVibrate = vibrationShown(typeof navigator === 'undefined' ? undefined : navigator);

  // Turning sound on is itself the user gesture that may start audio, so
  // the switch unlocks it and plays a sample at once. The listener that
  // unlocks audio on later gestures (after a reload) is installed too.
  function setSound(on: boolean): void {
    settings.set('sound', on);
    if (!on) return;
    installFx();
    unlockAudio();
    play('chime', { volume: settings.value.soundVolume });
  }

  // "Show the tips again" says it worked until the next change.
  let tipsReset = $state(false);
  const uid = $props.id();
  const hintsTitleId = `${uid}-hints`;
  function resetTips(): void {
    tips.resetSeen();
    tipsReset = true;
  }

  function setVolume(v: SoundVolume): void {
    settings.set('soundVolume', v);
    unlockAudio();
    play('coin', { volume: v });
  }

  const themeOptions = $derived(THEME_PREFS.map((p) => ({ value: p, label: t(`theme.${p}`) })));
  const speedOptions = $derived(
    ANIMATION_SPEEDS.map((s) => ({ value: s, label: t(SPEED_LABEL[s]) }))
  );
  const confirmOptions = $derived(
    CONFIRM_MOVES_OPTIONS.map((c) => ({ value: c, label: t(CONFIRM_LABEL[c]) }))
  );
  const volumeOptions = $derived(
    SOUND_VOLUMES.map((v) => ({ value: v, label: t(VOLUME_LABEL[v]) }))
  );
  const cardTextOptions = $derived(
    CARD_TEXT_SIZES.map((c) => ({ value: c, label: t(CARD_TEXT_LABEL[c]) }))
  );
</script>

<div class="flex flex-col gap-6">
  <div class="flex flex-col gap-1.5">
    <span class="text-sm font-medium text-fg-muted" aria-hidden="true">{t('theme.label')}</span>
    <Segmented
      name="settings-theme"
      legend={t('theme.label')}
      options={themeOptions}
      bind:value={() => theme.pref, (v) => theme.set(v as ThemePref)}
    />
  </div>

  <div class="flex flex-col gap-1.5">
    <span class="text-sm font-medium text-fg-muted" aria-hidden="true">
      {t('home.settings.animationSpeedLegend')}
    </span>
    <Segmented
      name="settings-speed"
      legend={t('home.settings.animationSpeedLegend')}
      options={speedOptions}
      bind:value={
        () => settings.value.animationSpeed,
        (v) => settings.set('animationSpeed', v as AnimationSpeed)
      }
    />
  </div>

  <div class="flex flex-col gap-1.5">
    <span class="text-sm font-medium text-fg-muted" aria-hidden="true">
      {t('home.settings.confirmMovesLegend')}
    </span>
    <Segmented
      name="settings-confirm"
      legend={t('home.settings.confirmMovesLegend')}
      options={confirmOptions}
      bind:value={
        () => settings.value.confirmMoves, (v) => settings.set('confirmMoves', v as ConfirmMoves)
      }
    />
  </div>

  <label class="flex min-h-11 items-center justify-between gap-3">
    <span class="flex flex-col">
      <span class="text-sm font-medium text-fg">{t('home.settings.colorPatterns')}</span>
      <span class="text-xs text-fg-muted">{t('home.settings.colorPatternsHint')}</span>
    </span>
    <input
      type="checkbox"
      class="size-5 shrink-0 accent-accent"
      checked={settings.value.colorPatterns}
      onchange={(e) => settings.set('colorPatterns', e.currentTarget.checked)}
    />
  </label>

  <div class="flex flex-col gap-1.5">
    <span class="text-sm font-medium text-fg-muted" aria-hidden="true">
      {t('home.settings.cardTextLegend')}
    </span>
    <Segmented
      name="settings-card-text"
      legend={t('home.settings.cardTextLegend')}
      options={cardTextOptions}
      bind:value={() => settings.value.cardText, (v) => settings.set('cardText', v as CardTextSize)}
    />
  </div>

  <label class="flex min-h-11 items-center justify-between gap-3">
    <span class="flex flex-col">
      <span class="text-sm font-medium text-fg">{t('home.settings.showOpponentMove')}</span>
      <span class="text-xs text-fg-muted">{t('home.settings.showOpponentMoveHint')}</span>
    </span>
    <input
      type="checkbox"
      class="size-5 shrink-0 accent-accent"
      data-setting="show-opponent-move"
      checked={settings.value.showOpponentMove}
      onchange={(e) => settings.set('showOpponentMove', e.currentTarget.checked)}
    />
  </label>

  <label class="flex min-h-11 items-center justify-between gap-3">
    <span class="flex flex-col">
      <span class="text-sm font-medium text-fg">{t('home.settings.sound')}</span>
      <span class="text-xs text-fg-muted">{t('home.settings.soundHint')}</span>
    </span>
    <input
      type="checkbox"
      class="size-5 shrink-0 accent-accent"
      data-setting="sound"
      checked={settings.value.sound}
      onchange={(e) => setSound(e.currentTarget.checked)}
    />
  </label>

  {#if settings.value.sound}
    <div class="flex flex-col gap-1.5" data-setting="sound-volume">
      <span class="text-sm font-medium text-fg-muted" aria-hidden="true">
        {t('home.settings.volumeLegend')}
      </span>
      <Segmented
        name="settings-volume"
        legend={t('home.settings.volumeLegend')}
        options={volumeOptions}
        bind:value={() => settings.value.soundVolume, (v) => setVolume(v as SoundVolume)}
      />
    </div>
  {/if}

  {#if canVibrate}
    <label class="flex min-h-11 items-center justify-between gap-3">
      <span class="flex flex-col">
        <span class="text-sm font-medium text-fg">{t('home.settings.haptics')}</span>
        <span class="text-xs text-fg-muted">{t('home.settings.hapticsHint')}</span>
      </span>
      <input
        type="checkbox"
        class="size-5 shrink-0 accent-accent"
        data-setting="haptics"
        checked={settings.value.haptics}
        onchange={(e) => settings.set('haptics', e.currentTarget.checked)}
      />
    </label>
  {/if}

  <section class="flex flex-col gap-3" aria-labelledby={hintsTitleId} data-settings-hints>
    <h3 id={hintsTitleId} class="text-sm font-medium text-fg-muted">
      {t('hints.settings.title')}
    </h3>
    <label class="flex min-h-11 items-center justify-between gap-3">
      <span class="flex flex-col">
        <span class="text-sm font-medium text-fg">{t('hints.settings.tips')}</span>
        <span class="text-xs text-fg-muted">{t('hints.settings.tipsHint')}</span>
      </span>
      <input
        type="checkbox"
        role="switch"
        class="size-5 shrink-0 accent-accent"
        data-setting="tips"
        checked={!tips.value.off}
        onchange={(e) => {
          tips.setOff(!e.currentTarget.checked);
          tipsReset = false;
        }}
      />
    </label>
    {#if !tips.value.off}
      <div class="flex flex-wrap items-center gap-3">
        <button
          type="button"
          class="min-h-9 rounded-control border border-rule bg-panel px-3 text-sm text-fg hover:bg-panel-sunken"
          data-setting="tips-reset"
          onclick={resetTips}
        >
          {t('hints.settings.tipsReset')}
        </button>
        <span class="text-xs text-fg-muted" role="status">
          {tipsReset ? t('hints.settings.tipsResetDone') : ''}
        </span>
      </div>
    {/if}
    <HintLocalHide
      hide={hintHide.value}
      onfamily={(family, hidden) => hintHide.setFamily(family, hidden)}
      onboardmarks={(hidden) => hintHide.setBoardMarks(hidden)}
    />
  </section>
</div>
