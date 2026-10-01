<script lang="ts">
  // Home: a placeholder until the game modes exist. The three mode buttons
  // are real but disabled, so the layout, theming and strings are in place.
  import Bot from '@lucide/svelte/icons/bot';
  import Globe from '@lucide/svelte/icons/globe';
  import Monitor from '@lucide/svelte/icons/monitor';
  import Moon from '@lucide/svelte/icons/moon';
  import Sun from '@lucide/svelte/icons/sun';
  import Users from '@lucide/svelte/icons/users';
  import { t } from '$lib/i18n';
  import { theme, THEME_PREFS } from '$lib/ui/theme.svelte';

  const PUBLISHER_URL = 'https://www.rprod.com/en/games/7-wonders-duel';

  const modes = [
    { id: 'ai', icon: Bot, label: 'home.playAi', hint: 'home.playAiHint', primary: true },
    { id: 'hotseat', icon: Users, label: 'home.passAndPlay', hint: 'home.passAndPlayHint' },
    { id: 'online', icon: Globe, label: 'home.playOnline', hint: 'home.playOnlineHint' }
  ] as const;

  const themeOptions = {
    system: { icon: Monitor, label: 'theme.system' },
    light: { icon: Sun, label: 'theme.light' },
    dark: { icon: Moon, label: 'theme.dark' }
  } as const;

  // Two overlapping squares make the eight-pointed star (khatam).
  const STAR_OUTER =
    'M50 4 L63.47 17.47 L82.53 17.47 L82.53 36.53 L96 50 L82.53 63.47 L82.53 82.53 L63.47 82.53 L50 96 L36.53 82.53 L17.47 82.53 L17.47 63.47 L4 50 L17.47 36.53 L17.47 17.47 L36.53 17.47 Z';
  const STAR_INNER =
    'M50 36 L54.1 40.1 L59.9 40.1 L59.9 45.9 L64 50 L59.9 54.1 L59.9 59.9 L54.1 59.9 L50 64 L45.9 59.9 L40.1 59.9 L40.1 54.1 L36 50 L40.1 45.9 L40.1 40.1 L45.9 40.1 Z';
</script>

<main class="h-full overflow-y-auto">
  <div
    class="mx-auto flex min-h-full w-full max-w-md flex-col px-4 pt-3 pb-6 split:max-w-5xl split:px-6"
  >
    <div class="flex justify-end">
      <fieldset
        class="inline-flex gap-0.5 rounded-full border border-rule bg-panel p-1 shadow-panel"
      >
        <legend class="sr-only">{t('theme.label')}</legend>
        {#each THEME_PREFS as pref (pref)}
          {@const option = themeOptions[pref]}
          <label
            class="flex min-h-11 cursor-pointer items-center gap-1.5 rounded-full px-3 text-sm font-semibold text-fg-muted transition-colors duration-150 hover:text-fg has-checked:bg-accent has-checked:text-on-accent"
          >
            <input
              class="sr-only"
              type="radio"
              name="theme"
              value={pref}
              checked={theme.pref === pref}
              onchange={() => theme.set(pref)}
            />
            <option.icon size={16} aria-hidden="true" />
            <span>{t(option.label)}</span>
          </label>
        {/each}
      </fieldset>
    </div>

    <div
      class="flex flex-col short:pt-3 split:my-auto split:grid split:grid-cols-2 split:items-center split:gap-10"
    >
      <header class="mt-8 flex flex-col items-center text-center short:mt-2 split:mt-0">
        <svg class="size-22 text-gild short:size-16" viewBox="0 0 100 100" aria-hidden="true">
          <path d={STAR_OUTER} fill="var(--accent-soft)" stroke="currentColor" stroke-width="2.5" />
          <circle cx="50" cy="50" r="24" fill="none" stroke="currentColor" stroke-width="1.5" />
          <path d={STAR_INNER} fill="currentColor" />
        </svg>
        <p
          class="mt-4 text-2xl leading-none text-gild short:mt-2"
          lang="fa"
          dir="rtl"
          aria-hidden="true"
        >
          {t('app.nameNative')}
        </p>
        <h1 class="mt-2 font-display text-5xl tracking-tight text-fg">{t('app.name')}</h1>
        <p class="mt-3 text-lg text-fg-muted">{t('app.tagline')}</p>
      </header>

      <nav
        class="mt-10 short:mt-4 split:mt-0 split:w-full split:max-w-md split:justify-self-end"
        aria-labelledby="home-modes"
      >
        <h2 id="home-modes" class="sr-only">{t('home.modes')}</h2>
        <ul class="flex flex-col gap-3 short:gap-2">
          {#each modes as mode (mode.id)}
            <li>
              <button
                type="button"
                disabled
                aria-describedby="home-coming-soon"
                class={[
                  'flex min-h-16 w-full cursor-not-allowed items-center gap-3 rounded-control border px-4 py-3 text-start shadow-panel short:min-h-14 short:py-2',
                  'primary' in mode
                    ? 'border-transparent bg-accent text-on-accent'
                    : 'border-rule-strong bg-panel text-fg'
                ]}
              >
                <mode.icon class="shrink-0" size={24} aria-hidden="true" />
                <span class="flex min-w-0 flex-1 flex-col">
                  <span class="text-base font-semibold">{t(mode.label)}</span>
                  <span class={['text-sm', 'primary' in mode ? 'opacity-90' : 'text-fg-muted']}>
                    {t(mode.hint)}
                  </span>
                </span>
                <span
                  class={[
                    'shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold',
                    'primary' in mode
                      ? 'bg-on-accent text-accent'
                      : 'bg-accent-soft text-accent-soft-fg'
                  ]}
                >
                  {t('home.soon')}
                </span>
              </button>
            </li>
          {/each}
        </ul>
        <p id="home-coming-soon" class="mt-4 text-center text-sm text-fg-muted short:mt-3">
          {t('home.comingSoon')}
        </p>
      </nav>
    </div>

    <footer class="mt-auto pt-10 text-center text-xs leading-relaxed text-fg-muted">
      <p>{t('app.legal')}</p>
      <p class="mt-1">
        <a
          class="inline-flex min-h-11 items-center font-semibold text-fg underline decoration-gild underline-offset-2"
          href={PUBLISHER_URL}
          rel="external noopener noreferrer"
          target="_blank"
        >
          {t('app.getGame')}
        </a>
      </p>
    </footer>
  </div>
</main>
