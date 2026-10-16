<script lang="ts">
  // The guild rule as a worked example: the Merchants Guild counts the
  // yellow cards of whichever city has more of them, here the Moon's three,
  // whoever builds it. Card swatches and crests; labels are translatable.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import CardSwatch from './CardSwatch.svelte';
  import DiagramFrame from './DiagramFrame.svelte';

  interface Props {
    /** The tooltip size (about 220 by 120 px), without the caption. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const ROWS = [
    { crest: 'crest-sun-sm', ink: 'text-sun-ink', cards: 2, counted: false },
    { crest: 'crest-moon-sm', ink: 'text-moon-ink', cards: 3, counted: true }
  ] as const;
</script>

<DiagramFrame
  name="guild"
  {compact}
  caption={t('rules.diagram.guild.label', {
    guild: t('card.name.merchants-guild'),
    moon: t('board.crest.moon'),
    sun: t('board.crest.sun')
  })}
>
  <div class="flex flex-col gap-1.5 {compact ? 'text-sm' : 'text-base'}">
    <div class="flex items-center gap-1.5">
      <CardSwatch color="purple" />
      <span class="text-xs font-semibold">{t('card.name.merchants-guild')}</span>
    </div>
    {#each ROWS as row (row.crest)}
      <div class="flex items-center gap-2">
        <span
          dir="ltr"
          class="flex items-center gap-1 rounded-[0.35rem] px-1 py-0.5 {row.counted
            ? 'outline-2 outline-(--gold-deep)'
            : ''}"
          aria-hidden="true"
          data-guild-counted={row.counted ? '' : undefined}
        >
          <Icon name={row.crest} class="me-1 text-[0.95em] {row.ink}" />
          {#each { length: row.cards }, i (i)}
            <CardSwatch color="yellow" />
          {/each}
        </span>
        {#if row.counted}
          <span class="text-xs font-semibold text-(--gold-deep)"
            >{t('rules.diagram.guild.counted')}</span
          >
        {/if}
      </div>
    {/each}
  </div>
</DiagramFrame>
