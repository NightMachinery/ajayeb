<script lang="ts">
  // /dev/fixture: every recorded game state a /dev/fixture/<name> page can show.
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import { fixtureList } from '$lib/session/fixtures';

  const names = fixtureList();
</script>

<main class="mx-auto flex h-full max-w-md flex-col gap-3 overflow-y-auto px-4 py-6">
  <h1 class="font-display text-3xl text-fg">{t('dev.fixture.title')}</h1>
  <ul class="flex flex-col">
    {#each names as name (name)}
      <li>
        <a
          class="inline-flex min-h-11 items-center text-fg underline decoration-gild underline-offset-2"
          href={resolve('/dev/fixture/[name]', { name })}
        >
          <code>{name}</code>
        </a>
      </li>
    {/each}
  </ul>
</main>
