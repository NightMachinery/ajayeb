<script lang="ts">
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { t } from '$lib/i18n';
</script>

<main class="flex h-full flex-col items-center justify-center gap-3 px-4 text-center">
  <p class="text-sm font-semibold text-fg-muted">{t('error.status', { status: page.status })}</p>
  <h1 class="font-display text-3xl text-fg">
    {page.status === 404 ? t('error.notFound') : t('error.generic')}
  </h1>
  <a
    class="mt-2 inline-flex min-h-11 items-center rounded-control bg-accent px-4 font-semibold text-on-accent"
    href={resolve('/')}
  >
    {t('app.home')}
  </a>
</main>
