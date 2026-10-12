<script lang="ts">
  // /rules: the rules reference as a full page, linked from Home. Each
  // section is a deep link (/rules#military). On a phone the contents come
  // first, then the sections; with room for two columns (split) the contents
  // stay beside the text. The same content shows inside a game through
  // openRules() (components/rules/rules-overlay.svelte.ts).
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import RulesContent from '$lib/components/rules/RulesContent.svelte';
  import RulesToc from '$lib/components/rules/RulesToc.svelte';
  import { sectionFromHash } from '$lib/components/rules/sections';

  // A deep link opened from outside lands on its section. The router scrolls
  // the window, but this page scrolls inside its own main element, so it
  // places the section itself once it is rendered. DOM only.
  onMount(() => {
    const id = sectionFromHash(location.hash);
    if (id) document.getElementById(id)?.scrollIntoView({ block: 'start', behavior: 'instant' });
  });
</script>

<svelte:head>
  <title>{t('rules.title')} · {t('app.name')}</title>
</svelte:head>

<main class="rules-page h-full overflow-y-auto" data-rules-page>
  <div
    class="mx-auto flex w-full max-w-xl flex-col gap-6 px-4 py-6 split:grid split:max-w-5xl split:grid-cols-[15rem_minmax(0,1fr)] split:items-start split:gap-x-10 split:px-6"
  >
    <header class="flex flex-col gap-2 split:col-span-2">
      <a
        href={resolve('/')}
        class="inline-flex min-h-11 w-fit items-center text-sm font-semibold text-fg-muted underline decoration-gild underline-offset-2 hover:text-fg"
        data-control="rules-home"
      >
        {t('rules.backHome')}
      </a>
      <h1 class="font-display text-3xl text-fg">{t('rules.pageTitle')}</h1>
      <p class="max-w-prose text-sm leading-relaxed text-fg-muted">{t('rules.intro')}</p>
    </header>

    <RulesToc
      class="rounded-panel border border-rule bg-panel p-3 split:sticky split:top-0 split:max-h-[calc(100dvh-3rem)] split:overflow-y-auto"
    />

    <RulesContent anchors class="max-w-prose" />
  </div>
</main>

<style>
  .rules-page {
    scroll-behavior: smooth;
  }
</style>
