<script lang="ts">
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
</script>

<main class="mx-auto flex h-full max-w-md flex-col gap-3 overflow-y-auto px-4 py-6">
  <h1 class="font-display text-3xl text-fg">{t('dev.title')}</h1>
  <p class="text-fg-muted">{t('dev.intro')}</p>
  <p>
    <a
      class="font-semibold text-fg underline decoration-gild underline-offset-2"
      href={resolve('/')}
    >
      {t('app.home')}
    </a>
  </p>
</main>
