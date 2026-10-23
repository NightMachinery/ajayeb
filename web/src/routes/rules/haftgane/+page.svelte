<script lang="ts">
  // /rules/haftgane: Haftgane's rules reference as a full page, the way
  // /rules is Duel's. Each section is a deep link (/rules/haftgane#military).
  // On a phone the contents come first, then the sections; with room for
  // two columns (split) the contents stay beside the text. The same content
  // shows inside a game through openHaftganeRules()
  // ($lib/games/haftgane/rules/rules-overlay.svelte.ts).
  //
  // SvelteKit gives the page a chunk of its own, and everything it imports
  // from $lib/games/haftgane loads only here: Home and Duel carry none of
  // it (src/platform-imports.test.ts).
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import { ht } from '$lib/games/haftgane/i18n/messages.svelte';
  import RulesContent from '$lib/games/haftgane/rules/RulesContent.svelte';
  import RulesToc from '$lib/games/haftgane/rules/RulesToc.svelte';
  import { hgSectionFromHash } from '$lib/games/haftgane/rules/sections';

  // A deep link opened from outside lands on its section. The router scrolls
  // the window, but this page scrolls inside its own main element, so it
  // places the section itself once it is rendered. DOM only.
  onMount(() => {
    const id = hgSectionFromHash(location.hash);
    if (id) document.getElementById(id)?.scrollIntoView({ block: 'start', behavior: 'instant' });
  });
</script>

<svelte:head>
  <title>{ht('hg.rules.title')} · {t('app.name')}</title>
</svelte:head>

<main class="rules-page h-full overflow-y-auto" data-rules-page="haftgane">
  <div
    class="mx-auto flex w-full max-w-xl flex-col gap-6 px-4 py-6 split:grid split:max-w-5xl split:grid-cols-[15rem_minmax(0,1fr)] split:items-start split:gap-x-10 split:px-6"
  >
    <header class="flex flex-col gap-2 split:col-span-2">
      <a
        href={resolve('/')}
        class="inline-flex min-h-11 w-fit items-center text-sm font-semibold text-fg-muted underline decoration-gild underline-offset-2 hover:text-fg"
        data-control="rules-home"
      >
        {ht('hg.rules.backHome')}
      </a>
      <h1 class="font-display text-3xl text-fg">{ht('hg.rules.pageTitle')}</h1>
      <p class="max-w-prose text-sm leading-relaxed text-fg-muted">{ht('hg.rules.intro')}</p>
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
