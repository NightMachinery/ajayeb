<script lang="ts">
  // The final score sheet as a table: one row per category, each with its
  // glyph and its name (never colour alone), a column per player, and the
  // total. A plain leaf: the rows come from lib/view/score.ts, which only
  // re-shapes the server's numbers.
  import Trophy from '@lucide/svelte/icons/trophy';
  import { formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import type { ScoreRow } from '$lib/view/score';

  interface Props {
    rows: readonly ScoreRow[];
    /** Display names by seat (user text). */
    names: readonly [string, string];
    /** The winner's seat, marked on the total line; null for none. */
    winner: 0 | 1 | null;
    /** Seat that reads "(you)" next to its name, if any. */
    you?: 0 | 1;
    class?: string;
  }

  let { rows, names, winner, you, class: className = '' }: Props = $props();

  const seats = [0, 1] as const;

  function icon(category: ScoreRow['category']): IconName {
    switch (category) {
      case 'blue':
      case 'green':
      case 'yellow':
      case 'purple':
        return `category-${category}`;
      case 'wonders':
        return 'wonder';
      case 'tokens':
        return 'token-frame';
      case 'coins':
        return 'coin';
      case 'military':
        return 'shield';
      case 'total':
        return 'vp-laurel';
    }
  }

  function label(category: ScoreRow['category']): string {
    return t(`choices.score.${category}`);
  }

  // The glyph tint per row: the card-colour inks for the four card rows (the
  // band colours fall under 3:1 on the sheet: yellow on paper, purple on the
  // lapis panels), the game colours for the rest. The name always sits next
  // to it.
  function tint(category: ScoreRow['category']): string {
    switch (category) {
      case 'blue':
        return 'text-card-blue-ink';
      case 'green':
        return 'text-card-green-ink';
      case 'yellow':
        return 'text-card-yellow-ink';
      case 'purple':
        return 'text-card-purple-ink';
      case 'wonders':
        return 'text-gild';
      case 'tokens':
        return 'text-gild';
      case 'coins':
        return 'text-coin-ink';
      case 'military':
        return 'text-card-red-ink';
      case 'total':
        return 'text-gild';
    }
  }
</script>

<table
  class="score-table w-full border-separate border-spacing-0 text-sm text-fg {className}"
  data-score-table
>
  <caption class="sr-only">{t('choices.score.caption')}</caption>
  <thead>
    <tr>
      <th scope="col" class="pb-1.5 text-start text-xs font-semibold text-fg-muted">
        <span class="sr-only">{t('choices.score.category')}</span>
      </th>
      {#each seats as seat (seat)}
        <th
          scope="col"
          class="w-[5.5rem] max-w-[5.5rem] px-1 pb-1.5 text-end text-xs font-semibold
            {seat === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
        >
          <!-- A long name (a bot's persona, "Apprentice") wraps under its
               crest rather than being cut off. -->
          <span class="flex flex-wrap items-center justify-end gap-x-1">
            <Icon name={seat === 0 ? 'crest-sun' : 'crest-moon'} class="shrink-0 text-[15px]" />
            <span class="min-w-0 text-end leading-tight [overflow-wrap:anywhere]" dir="auto"
              >{names[seat]}</span
            >
          </span>
          {#if you === seat}
            <span class="block text-[12px] font-normal text-fg-muted">{t('choices.age.you')}</span>
          {/if}
        </th>
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each rows as row (row.category)}
      {@const total = row.category === 'total'}
      <tr data-score-row={row.category}>
        <th
          scope="row"
          class="py-1 pe-2 text-start font-normal whitespace-nowrap {total
            ? 'border-t-2 border-rule-strong pt-2 font-semibold'
            : 'border-t border-rule'}"
        >
          <span class="flex items-center gap-2">
            <Icon name={icon(row.category)} class="shrink-0 text-[17px] {tint(row.category)}" />
            <span>{label(row.category)}</span>
          </span>
        </th>
        {#each seats as seat (seat)}
          <td
            class="px-1 py-1 text-end tabular-nums {total
              ? 'border-t-2 border-rule-strong pt-2 text-base font-bold'
              : 'border-t border-rule'} {row.ahead === seat && !total ? 'font-semibold' : ''}"
            data-must-fit
          >
            <span class="inline-flex items-center justify-end gap-1">
              <!-- A trophy, not a crown: the crown is the guilds' category
                   glyph, one row up. -->
              {#if total && winner === seat}
                <Trophy
                  class="size-4 text-gild"
                  role="img"
                  aria-label={t('choices.score.winner')}
                />
              {/if}
              {formatNumber(row.values[seat])}
            </span>
          </td>
        {/each}
      </tr>
    {/each}
  </tbody>
</table>
