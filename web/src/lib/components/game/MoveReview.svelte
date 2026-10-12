<script lang="ts">
  // The move review (the advisor hint family, M10.HS5): the last block of
  // the opponent move summary (MoveSummary's review slot), telling the
  // player, and only them, what they played and whether a clearly better
  // move was there, and why. The server decides both (View.review); the
  // words come from lib/view/review.ts. A lesson stands out a little; the
  // quiet "no better move" line stays muted.
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

<section
  class="flex flex-col gap-1 rounded-control border px-2.5 py-2
    {lines.better ? 'border-accent bg-accent-soft' : 'border-rule bg-panel-sunken'}"
  aria-label={t('hints.review.title')}
  data-move-review={lines.better ? 'better' : 'none'}
>
  <p
    class="flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase
      {lines.better ? 'text-accent-soft-fg' : 'text-fg-muted'}"
  >
    <span class="inline-flex" {@attach tooltip({ text: t('hints.review.tip') })}>
      <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
    </span>
    {t('hints.review.title')}
  </p>
  <p class="text-sm leading-snug text-fg" data-review-played>{lines.played}</p>
  <p
    class="text-sm leading-snug {lines.better ? 'font-semibold text-fg' : 'text-fg-muted'}"
    data-review-verdict
  >
    {lines.verdict}
  </p>
</section>
