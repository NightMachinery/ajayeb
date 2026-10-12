<script lang="ts">
  // /dev/gallery/tooltips: the tooltip layer's own gallery page
  // (lib/dev/gallery/Tooltips.svelte), apart from the long main gallery so
  // the e2e spec and the screenshots get a short, quiet page. Dev-only, like
  // every /dev route.
  import { page } from '$app/state';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import Tooltips from '$lib/dev/gallery/Tooltips.svelte';

  const show = $derived(page.url.searchParams.get('show') ?? undefined);
  const edges = $derived(page.url.searchParams.has('edges'));
</script>

<main class="mx-auto flex h-full max-w-3xl flex-col gap-6 overflow-y-auto px-4 py-6 pb-16">
  <header class="flex flex-wrap items-baseline justify-between gap-2">
    <h1 class="font-display text-3xl text-fg">{t('dev.tooltips.title')}</h1>
    <a class="text-sm text-fg underline decoration-gild" href={resolve('/dev/gallery')}>
      {t('dev.tooltips.back')}
    </a>
  </header>
  <Tooltips {show} {edges} />
</main>
