<script lang="ts">
  import '../app.css';
  import interRegular from '$lib/fonts/inter-400.woff2?url';
  import frauncesHeading from '$lib/fonts/fraunces-600.woff2?url';
  import { page } from '$app/state';
  import { online } from 'svelte/reactivity/window';
  import OfflineNotice from '$lib/components/shell/OfflineNotice.svelte';
  import { holdGameLock } from '$lib/sw/game-lock';
  import { isGamePath } from '$lib/sw/policy';
  import TooltipLayer from '$lib/components/shell/TooltipLayer.svelte';
  import IconSprite from '$lib/icons/IconSprite.svelte';
  import { i18n, t } from '$lib/i18n';
  import { motion } from '$lib/ui/motion.svelte';
  import { applySafeInsets, parseSafeParam } from '$lib/ui/safe-area';
  import { theme } from '$lib/ui/theme.svelte';
  import type { LayoutProps } from './$types';

  let { children }: LayoutProps = $props();

  const inGame = $derived(isGamePath(page.url.pathname));

  // The app opens offline from the service worker's cache. A game page
  // shows the session's own offline state; every other page gets the
  // shell's one-line notice. `online.current` is undefined on the server.
  const offline = $derived(online.current === false && !inGame);

  // A game page holds the game lock, so a new service worker waits for the
  // game (lib/sw/game-lock.ts). It takes and releases a lock, never state.
  $effect(() => {
    if (inGame) return holdGameLock();
  });

  // These effects only write to the DOM; they never write $state.
  $effect(() => {
    theme.apply();
  });

  // data-motion on <html> follows the game's own Animation setting (D21:
  // the OS's reduced-motion preference is deliberately not read).
  $effect(() => {
    motion.apply();
  });

  $effect(() => {
    const root = document.documentElement;
    root.lang = i18n.locale;
    root.dir = i18n.dir;
  });

  $effect(() => {
    applySafeInsets(document.documentElement, parseSafeParam(page.url.searchParams.get('safe')));
  });
</script>

<svelte:head>
  <title>{t('app.name')}</title>
  <link rel="preload" href={interRegular} as="font" type="font/woff2" crossorigin="anonymous" />
  <link rel="preload" href={frauncesHeading} as="font" type="font/woff2" crossorigin="anonymous" />
</svelte:head>

<IconSprite />

<div
  class="app-shell"
  style:padding-block-end={offline ? 'calc(var(--safe-b) + 4.5rem)' : undefined}
>
  {@render children()}
</div>

<OfflineNotice {offline} />

<!-- The one tooltip layer (ui/tooltip.svelte.ts): every anchor given a
     tooltip with the tooltip attachment shows it here. -->
<TooltipLayer />
