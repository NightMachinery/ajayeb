<script lang="ts">
  // The seventh-wonder rule: eight wonders are drafted, four a player, but
  // only seven can be built. Seven are built here (four and three); the
  // last unbuilt one leaves the game. Sprite icons and translatable labels.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import DiagramFrame from './DiagramFrame.svelte';

  interface Props {
    /** The tooltip size (about 220 by 120 px), without the caption. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const ROWS = [
    { crest: 'crest-sun-sm', ink: 'text-sun-ink', built: 4 },
    { crest: 'crest-moon-sm', ink: 'text-moon-ink', built: 3 }
  ] as const;
</script>

<DiagramFrame name="seventh" {compact} caption={t('rules.diagram.seventh.label')}>
  <div class="flex flex-col gap-1 {compact ? 'text-lg' : 'text-2xl'}">
    <span class="text-xs font-semibold">{t('rules.diagram.seventh.built', { count: 7 })}</span>
    {#each ROWS as row (row.crest)}
      <div dir="ltr" class="flex items-center gap-1.5" aria-hidden="true">
        <Icon name={row.crest} class="me-1 text-[0.7em] {row.ink}" />
        {#each { length: 4 }, i (i)}
          {#if i < row.built}
            <Icon name="wonder" class="text-fg [--i2:var(--gold)]" />
          {:else}
            <span class="relative inline-grid place-items-center" data-wonder-out>
              <Icon name="wonder" class="text-fg opacity-30" />
              <span class="absolute inset-x-0 top-1/2 h-0.5 -rotate-45 bg-cinnabar"></span>
            </span>
          {/if}
        {/each}
      </div>
    {/each}
    <span class="self-end text-xs text-cinnabar-ink">{t('rules.diagram.seventh.out')}</span>
  </div>
</DiagramFrame>
