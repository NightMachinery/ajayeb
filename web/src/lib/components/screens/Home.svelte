<script lang="ts">
  // The Home screen (UX design §2). A screen, not a tiny leaf: it reads the
  // live theme directly (there is exactly one, like every other screen that
  // already does this), but every action it cannot decide for itself
  // (continuing a room, opening a sheet) comes back through callback props,
  // so the gallery can render it with a few fake saved rooms and no live
  // connection. The hero keeps its text logotype (live text, painted first);
  // the drawn Nastaliq wordmark heads the footer, the app's about block
  // (M11.ornament-integration). The theme switcher and the publisher footer
  // link carry over unchanged from the M1 placeholder this replaces.
  //
  // The hero painting (hero/hero) sits above the logotype in a 16:9 frame
  // whose size never depends on the art, so the page does not move when it
  // is approved: until then the frame shows the eight-pointed star. The
  // frame is capped in height so the mode buttons stay above the fold on a
  // 360 x 740 phone, and on a short landscape screen (where split applies
  // too, so short wins with !) it is no taller than the star it replaces,
  // so the tagline stays on screen.
  //
  // The tutorial (the tutorial design, section 2.5): with `tutorialOffer`,
  // the first-launch offer panel sits above the mode buttons (it is inline,
  // so it swallows nothing), and with `onTutorial` a quiet Learn to play row
  // sits under them ("play it again" once `tutorialDone`). A saved tutorial
  // room continues as "Tutorial".
  import { resolve } from '$app/paths';
  import ArtImage from '$lib/art/ArtImage.svelte';
  import Wordmark from '$lib/brand/Wordmark.svelte';
  import { artId, artSource } from '$lib/art/lookup';
  import Bot from '@lucide/svelte/icons/bot';
  import Globe from '@lucide/svelte/icons/globe';
  import GraduationCap from '@lucide/svelte/icons/graduation-cap';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import Monitor from '@lucide/svelte/icons/monitor';
  import Moon from '@lucide/svelte/icons/moon';
  import SettingsIcon from '@lucide/svelte/icons/settings';
  import Sun from '@lucide/svelte/icons/sun';
  import Users from '@lucide/svelte/icons/users';
  import { ageLabel, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { SavedRoom } from '$lib/session/rooms.svelte';
  import { theme, THEME_PREFS } from '$lib/ui/theme.svelte';
  import TutorialOffer from '../tutorial/TutorialOffer.svelte';

  const heroSizes = '(min-width: 64rem) 28rem, 20rem';
  const hero = artSource(artId('hero', 'hero'), 'default', heroSizes);

  const PUBLISHER_URL = 'https://www.rprod.com/en/games/7-wonders-duel';
  const SOURCE_URL = 'https://github.com/NightMachinery/ajayeb';

  interface Props {
    /** Native controls stay unavailable until the route has mounted. */
    ready?: boolean;
    /** The saved rooms this browser has played, newest first. */
    rooms: readonly SavedRoom[];
    onContinue: (code: string) => void;
    onPlayAi: () => void;
    onPassAndPlay: () => void;
    onPlayOnline: () => void;
    /** Opens the join-by-code sheet; the link shows only when given. */
    onJoinCode?: () => void;
    onSettings: () => void;
    /** Shows the first-launch tutorial offer (needs onTutorial). */
    tutorialOffer?: boolean;
    /** This device finished the tutorial: the row offers it again. */
    tutorialDone?: boolean;
    /** Starts the tutorial; the offer and the row show only when given. */
    onTutorial?: () => void;
    /** The offer's No thanks. */
    onDeclineTutorial?: () => void;
  }

  let {
    ready = true,
    rooms,
    onContinue,
    onPlayAi,
    onPassAndPlay,
    onPlayOnline,
    onJoinCode,
    onSettings,
    tutorialOffer = false,
    tutorialDone = false,
    onTutorial,
    onDeclineTutorial
  }: Props = $props();
  // Unique per instance: the gallery shows Home more than once on a page.
  const uid = $props.id();

  /** Home shows at most this many; the full list lives in localStorage. */
  const MAX_CONTINUE = 3;
  const continueRooms = $derived(rooms.slice(0, MAX_CONTINUE));

  const MODE_LABEL = {
    online: 'home.mode.online',
    hotseat: 'home.mode.hotseat',
    ai: 'home.mode.ai'
  } as const;

  // $derived, not a plain const: onPlayAi and friends are props, so a plain
  // array literal would only capture their initial value (Svelte's
  // state_referenced_locally warning).
  const modes = $derived([
    {
      id: 'ai',
      icon: Bot,
      label: 'home.playAi',
      hint: 'home.playAiHint',
      primary: true,
      onclick: onPlayAi
    },
    {
      id: 'hotseat',
      icon: Users,
      label: 'home.passAndPlay',
      hint: 'home.passAndPlayHint',
      onclick: onPassAndPlay
    },
    {
      id: 'online',
      icon: Globe,
      label: 'home.playOnline',
      hint: 'home.playOnlineHint',
      onclick: onPlayOnline
    }
  ] as const);

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

<svelte:head>
  {#if hero}
    <link
      rel="preload"
      as="image"
      href={hero.src}
      imagesrcset={hero.srcset}
      imagesizes={hero.sizes}
      fetchpriority="high"
    />
  {/if}
</svelte:head>

<!-- Shown only while a stored Persian locale keeps main hidden (see the
     style below), so the page is never empty to a screen reader. -->
<p role="status" class="home-pending-status text-sm font-semibold text-fg">
  {t('home.loading')}
</p>
<main class="h-full overflow-y-auto" data-home-loading={!ready ? '' : undefined}>
  <div
    class="mx-auto flex min-h-full w-full max-w-md flex-col px-4 pt-3 pb-6 split:max-w-5xl split:px-6"
  >
    <div class="flex items-center justify-end gap-2">
      <fieldset
        disabled={!ready}
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
      <button
        type="button"
        disabled={!ready}
        class="flex size-11 items-center justify-center rounded-full border border-rule bg-panel text-fg-muted transition-colors duration-150 hover:text-fg"
        onclick={onSettings}
        aria-label={t('home.settings')}
        {@attach tooltip({ text: t('home.settings') })}
      >
        <SettingsIcon size={18} aria-hidden="true" />
      </button>
    </div>

    <div
      class="flex flex-col short:pt-3 split:my-auto split:grid split:grid-cols-2 split:items-center split:gap-10"
    >
      <header class="relative mt-3 flex flex-col items-center text-center short:mt-0 split:mt-0">
        <div
          class="hero relative grid aspect-video h-auto max-h-44 w-full max-w-xs place-items-center short:max-h-16! short:max-w-28! split:max-h-64 split:max-w-md"
          data-hero
        >
          <ArtImage
            id={artId('hero', 'hero')}
            sizes={heroSizes}
            priority
            feather="soft"
            class="absolute inset-0 h-full w-full"
          >
            {#snippet fallback()}
              <svg
                class="aspect-square h-1/2 w-auto text-gild short:h-full"
                viewBox="0 0 100 100"
                aria-hidden="true"
              >
                <path
                  d={STAR_OUTER}
                  fill="var(--accent-soft)"
                  stroke="currentColor"
                  stroke-width="2.5"
                />
                <circle
                  cx="50"
                  cy="50"
                  r="24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                />
                <path d={STAR_INNER} fill="currentColor" />
              </svg>
            {/snippet}
          </ArtImage>
        </div>
        <p
          class="mt-3 text-2xl leading-none text-gild short:mt-1"
          lang="fa"
          dir="rtl"
          aria-hidden="true"
        >
          {t('app.nameNative')}
        </p>
        <h1 class="mt-2 font-display text-5xl tracking-tight text-fg">{t('app.name')}</h1>
        <p class="mt-3 text-lg text-fg-muted">{t('app.tagline')}</p>
        {#if !ready}
          <p role="status" class="home-loading-status mx-auto w-fit text-sm font-semibold text-fg">
            {t('home.loading')}
          </p>
        {/if}
      </header>

      <div class="mt-8 short:mt-4 split:mt-0 split:w-full split:max-w-md split:justify-self-end">
        {#if tutorialOffer && onTutorial}
          <div class="home-offer mb-3" inert={!ready}>
            <TutorialOffer
              botName={t('game.persona.easy')}
              onstart={onTutorial}
              ondecline={() => onDeclineTutorial?.()}
            />
          </div>
        {/if}
        {#if continueRooms.length > 0}
          <section class="mb-3" aria-labelledby="{uid}-continue">
            <h2 id="{uid}-continue" class="sr-only">{t('home.continueLabel')}</h2>
            <ul class="flex flex-col gap-2">
              {#each continueRooms as room (room.code)}
                <li>
                  <button
                    type="button"
                    disabled={!ready}
                    class="flex min-h-14 w-full items-center justify-between gap-3 rounded-control border border-rule-strong bg-accent-soft px-4 py-2 text-start text-accent-soft-fg"
                    onclick={() => onContinue(room.code)}
                  >
                    <span class="flex min-w-0 flex-col gap-0.5">
                      <span class="font-semibold"
                        >{t('home.continueEntry', {
                          mode: t(room.tutorial ? 'home.mode.tutorial' : MODE_LABEL[room.mode]),
                          code: room.code
                        })}</span
                      >
                      {#if room.age !== undefined || room.bot}
                        {@const stage =
                          room.age === undefined
                            ? ''
                            : room.age === 0
                              ? t('home.continueDraft')
                              : t('home.continueAge', { age: ageLabel(room.age) })}
                        <span class="text-sm" data-continue-details>
                          {room.bot && stage
                            ? t('home.continueDetails', {
                                level: t(`game.persona.${room.bot}`),
                                stage
                              })
                            : room.bot
                              ? t(`game.persona.${room.bot}`)
                              : stage}
                        </span>
                      {/if}
                    </span>
                  </button>
                </li>
              {/each}
            </ul>
          </section>
        {/if}

        <nav aria-labelledby="{uid}-modes">
          <h2 id="{uid}-modes" class="sr-only">{t('home.modes')}</h2>
          <ul class="flex flex-col gap-3 short:gap-2">
            {#each modes as mode (mode.id)}
              <li>
                <button
                  type="button"
                  disabled={!ready}
                  onclick={mode.onclick}
                  class={[
                    'flex min-h-16 w-full items-center gap-3 rounded-control border px-4 py-3 text-start shadow-panel short:min-h-14 short:py-2',
                    'primary' in mode
                      ? 'border-transparent bg-accent text-on-accent hover:brightness-105'
                      : 'border-rule-strong bg-panel text-fg hover:bg-panel-sunken'
                  ]}
                >
                  <mode.icon class="shrink-0" size={24} aria-hidden="true" />
                  <span class="flex min-w-0 flex-1 flex-col">
                    <span class="text-base font-semibold">{t(mode.label)}</span>
                    <span class={['text-sm', 'primary' in mode ? 'opacity-90' : 'text-fg-muted']}>
                      {t(mode.hint)}
                    </span>
                  </span>
                </button>
              </li>
            {/each}
          </ul>
          {#if onTutorial}
            <button
              type="button"
              disabled={!ready}
              class="mt-3 flex min-h-12 w-full items-center gap-3 rounded-control border border-dashed border-rule-strong px-4 py-2 text-start text-fg hover:bg-panel-sunken short:mt-2"
              data-control="home-tutorial"
              data-tutorial-replay={tutorialDone ? '' : undefined}
              onclick={onTutorial}
            >
              <GraduationCap class="shrink-0 text-accent" size={20} aria-hidden="true" />
              <span class="flex min-w-0 flex-col">
                <span class="text-sm font-semibold"
                  >{tutorialDone ? t('home.tutorialDone') : t('home.tutorial')}</span
                >
                {#if !tutorialDone}
                  <span class="text-xs text-fg-muted"
                    >{t('home.tutorialHint', { name: t('game.persona.easy') })}</span
                  >
                {/if}
              </span>
            </button>
          {/if}
          {#if onJoinCode}
            <button
              type="button"
              disabled={!ready}
              class="mx-auto mt-2 flex min-h-11 items-center gap-2 rounded-control px-3 text-sm font-semibold text-fg underline decoration-gild underline-offset-2"
              data-control="home-join"
              onclick={onJoinCode}
            >
              <KeyRound size={16} aria-hidden="true" />
              {t('home.joinCode')}
            </button>
          {/if}
        </nav>
      </div>
    </div>

    <footer class="mt-auto pt-10 text-center text-xs leading-relaxed text-fg-muted">
      <!-- The about block: the wordmark over what Ajayeb is and where it comes from. -->
      <Wordmark class="mx-auto mb-3 block w-[120px] text-fg" />
      <p>{t('app.legal')}</p>
      <p class="mt-1">{t('app.foss')}</p>
      <p class="mt-1 flex flex-wrap items-center justify-center gap-x-5">
        <a
          class="inline-flex min-h-11 items-center font-semibold text-fg underline decoration-gild underline-offset-2"
          href={resolve('/rules')}
          data-control="home-rules"
        >
          {t('rules.homeLink')}
        </a>
        <a
          class="inline-flex min-h-11 items-center font-semibold text-fg underline decoration-gild underline-offset-2"
          href={PUBLISHER_URL}
          rel="external noopener noreferrer"
          target="_blank"
        >
          {t('app.getGame')}
        </a>
        <a
          class="inline-flex min-h-11 items-center font-semibold text-fg underline decoration-gild underline-offset-2"
          href={SOURCE_URL}
          rel="external noopener noreferrer"
          target="_blank"
          data-control="home-source"
        >
          {t('app.source')}
        </a>
      </p>
    </footer>
  </div>
</main>

<style>
  .home-loading-status {
    position: absolute;
    inset-inline: 0;
    inset-block-start: 100%;
    margin-block-start: 0.25rem;
  }
  /* Native disabled semantics cover touch, click, keyboard and radio arrows. */
  [data-home-loading] button,
  [data-home-loading] fieldset {
    opacity: 0.55;
  }
  [data-home-loading] button,
  [data-home-loading] label {
    cursor: wait;
  }

  /* Cold loads must not move Home (M11.home-cold-cls). The build-time Home
     carries the first-launch offer, and the bootstrap in app.html says
     before the first paint whether this device will keep it, so the offer
     is never inserted above the modes after the app starts. */
  :global(:root:not([data-boot-offer])) [data-home-loading] .home-offer {
    display: none;
  }
  /* A stored Persian locale is pending until its messages load. Home's
     text waits for them hidden, already laid out right to left, so the
     switch neither mirrors the columns nor moves the hero, which stays
     visible as the page's largest paint. */
  :global(:root[data-locale-pending='fa']) main {
    direction: rtl;
    visibility: hidden;
  }
  :global(:root[data-locale-pending]) .hero {
    visibility: visible;
  }
  /* Meanwhile this status stands in for main's, outside the hidden content
     and out of the flow, so it adds no layout space and its removal moves
     nothing. The pending mark lasts until the app has started and the
     initial locale request has settled (or the bootstrap's timer ends). */
  .home-pending-status {
    display: none;
  }
  :global(:root[data-locale-pending]) .home-pending-status {
    display: block;
    position: fixed;
    inset-inline: 0;
    inset-block-end: calc(var(--safe-b) + 2rem);
    text-align: center;
  }

  /* Night uses a deliberate painting frame rather than fading cream into navy. */
  :global([data-theme='dark']) .hero {
    overflow: hidden;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
    box-shadow: var(--shadow-panel);
  }
  :global([data-theme='dark']) .hero :global(.art-image) {
    mask-image: none;
    -webkit-mask-image: none;
  }
</style>
