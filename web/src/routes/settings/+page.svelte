<script lang="ts">
  // /settings: a deep-linkable full page around the same Settings component
  // Home opens as a sheet, for a link or a bookmark (UX design §2).
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import Settings from '$lib/components/screens/Settings.svelte';
</script>

<svelte:head>
  <title>{t('home.settings.title')} · {t('app.name')}</title>
</svelte:head>

<main class="h-full overflow-y-auto">
  <div class="mx-auto flex w-full max-w-md flex-col gap-6 px-4 py-6 split:max-w-lg">
    <header class="flex flex-col gap-2">
      <a
        href={resolve('/')}
        class="inline-flex min-h-11 w-fit items-center text-sm font-semibold text-fg-muted underline decoration-gild underline-offset-2 hover:text-fg"
      >
        {t('app.home')}
      </a>
      <h1 class="font-display text-2xl text-fg">{t('home.settings.title')}</h1>
    </header>

    <Settings />
  </div>
</main>
