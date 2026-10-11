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
    settings,
    type AnimationSpeed,
    type CardTextSize,
    type ConfirmMoves
  } from '$lib/ui/settings.svelte';
  import { theme, THEME_PREFS, type ThemePref } from '$lib/ui/theme.svelte';

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

  const themeOptions = $derived(THEME_PREFS.map((p) => ({ value: p, label: t(`theme.${p}`) })));
  const speedOptions = $derived(
    ANIMATION_SPEEDS.map((s) => ({ value: s, label: t(SPEED_LABEL[s]) }))
  );
  const confirmOptions = $derived(
    CONFIRM_MOVES_OPTIONS.map((c) => ({ value: c, label: t(CONFIRM_LABEL[c]) }))
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
</div>
