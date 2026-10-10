<script lang="ts">
  // How many spectators watch an online game: a quiet eye and a number in
  // the top bar, for the players. Nothing shows while nobody watches.
  import Eye from '@lucide/svelte/icons/eye';
  import { t } from '$lib/i18n';

  interface Props {
    count: number;
    class?: string;
  }

  let { count, class: className = '' }: Props = $props();
</script>

{#if count > 0}
  <span
    class="inline-flex min-h-7 shrink-0 items-center gap-1 rounded-full px-2 text-xs font-semibold text-fg-muted {className}"
    title={t('online.spectatorsLabel', { count })}
    data-spectators={count}
  >
    <Eye class="size-3.5" aria-hidden="true" />
    <span aria-hidden="true">{count}</span>
    <span class="sr-only">{t('online.spectatorsLabel', { count })}</span>
  </span>
{/if}
