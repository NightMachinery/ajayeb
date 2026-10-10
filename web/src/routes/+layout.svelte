<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import IconSprite from '$lib/icons/IconSprite.svelte';
  import { i18n, t } from '$lib/i18n';
  import { motion } from '$lib/ui/motion.svelte';
  import { applySafeInsets, parseSafeParam } from '$lib/ui/safe-area';
  import { theme } from '$lib/ui/theme.svelte';
  import type { LayoutProps } from './$types';

  let { children }: LayoutProps = $props();

  // These effects only write to the DOM; they never write $state.
  $effect(() => {
    theme.apply();
  });

  // data-motion on <html> follows the animation speed setting (and the
  // system's reduced-motion preference, which forces it off).
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
</svelte:head>

<IconSprite />

<div class="app-shell">
  {@render children()}
</div>
