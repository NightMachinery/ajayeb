<script lang="ts">
  // Who starts the next age: the player on whose side the pawn stands (the
  // weaker one) chooses; with the pawn on the centre, the player who took
  // the last card chooses (docs/rules/decisions.md). Two small tracks with
  // the real pawn and crests; the board is spatial, so the tracks are
  // dir="ltr" like the real one. Labels are translatable.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import DiagramFrame from './DiagramFrame.svelte';

  interface Props {
    /** The tooltip size (about 220 by 120 px), without the caption. */
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const CELLS = [0, 1, 2, 3, 4, 5, 6] as const;
  const ROWS = [
    { pawn: 5, line: 'rules.diagram.starter.weaker' },
    { pawn: 3, line: 'rules.diagram.starter.centre' }
  ] as const;
</script>

{#snippet track(pawn: number)}
  <!-- Sized in em, so the track grows with the full variant. -->
  <span
    dir="ltr"
    class="flex shrink-0 items-center gap-px {compact ? 'text-[11px]' : 'text-base'}"
    aria-hidden="true"
  >
    <Icon name="crest-sun-sm" class="me-[0.2em] text-[1em] text-sun-ink" />
    {#each CELLS as cell (cell)}
      <span
        class="grid size-[1.15em] place-items-center rounded-[0.2em] border border-rule {cell === 3
          ? 'bg-panel'
          : 'bg-cinnabar/15'}"
      >
        {#if cell === pawn}
          <Icon name="military-pawn" class="text-[0.8em] text-fg" />
        {/if}
      </span>
    {/each}
    <Icon name="crest-moon-sm" class="ms-[0.2em] text-[1em] text-moon-ink" />
  </span>
{/snippet}

<DiagramFrame name="starter" {compact} caption={t('rules.diagram.starter.label')}>
  <div class="flex flex-col gap-1.5">
    {#each ROWS as row (row.pawn)}
      <div class="flex flex-col gap-0.5" data-starter-pawn={row.pawn}>
        {@render track(row.pawn)}
        <span class="text-xs leading-tight">
          {t(row.line, { moon: t('board.crest.moon') })}
        </span>
      </div>
    {/each}
  </div>
</DiagramFrame>
