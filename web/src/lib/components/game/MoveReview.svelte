<script lang="ts">
  // The move review (the advisor hint family, M10.HS5): the last block of
  // the opponent move summary (MoveSummary's review slot), telling the
  // player, and only them, what they played and whether a clearly better
  // move was there, and why. The server decides both (View.review); the
  // words come from lib/view/review.ts. A lesson stands out as an advice
  // card; when no better move was found, the block collapses to a quiet,
  // compact line showing only what the player played.
  //
  // A leaf with plain props, so the gallery can render it standalone.
  import Lightbulb from '@lucide/svelte/icons/lightbulb';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { ReviewLines } from '$lib/view/review';

  interface Props {
    lines: ReviewLines;
  }

  let { lines }: Props = $props();
</script>

{#if lines.better}
  <section
    class="flex flex-col gap-1 rounded-control border border-accent bg-accent-soft px-2.5 py-2"
    aria-label={t('hints.review.title')}
    data-move-review="better"
  >
    <p
      class="flex items-center gap-1.5 text-xs font-semibold tracking-wide text-accent-soft-fg uppercase"
    >
      <span class="inline-flex" {@attach tooltip({ text: t('hints.review.tip') })}>
        <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
      </span>
      {t('hints.review.title')}
    </p>
    {#each lines.entries as entry (entry.ply)}
      <div class="flex flex-col gap-0.5" data-review-entry={entry.ply}>
        <p dir="auto" class="text-sm leading-snug text-fg" data-review-played>{entry.played}</p>
        {#if entry.verdict}
          <p dir="auto" class="text-sm leading-snug font-semibold text-fg" data-review-verdict>
            {entry.verdict}
          </p>
        {/if}
      </div>
    {/each}
  </section>
{:else}
  <section
    class="flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5 rounded-control border border-rule bg-panel-sunken px-2.5 py-1.5 text-xs leading-snug text-fg-muted"
    aria-label={t('hints.review.title')}
    data-move-review="none"
  >
    <span
      class="font-semibold tracking-wide uppercase"
      {@attach tooltip({ text: t('hints.review.tip') })}
    >
      {t('hints.review.title')}
    </span>
    {#each lines.entries as entry (entry.ply)}
      <span dir="auto" class="text-fg" data-review-entry={entry.ply} data-review-played
        >{entry.played}</span
      >
    {/each}
  </section>
{/if}
