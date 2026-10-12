<script lang="ts">
  // One side of the military track, from the centre to a capital, for the
  // rules reference: the zones and their points, the looting tokens on
  // spaces 3 and 6, the pawn on the centre and the capital at the end. The
  // other side is the same. Fixed rule numbers only, no game state. Laid out
  // in HTML, so every label is translatable text; the board is spatial, so
  // the strip is dir="ltr" like the real track.
  import { formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import LootingToken from '../board/LootingToken.svelte';

  const SPACES = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9] as const;

  /** The zones from the centre out: the spaces they span and their points. */
  const ZONES = [
    { from: 1, to: 2, points: 2, fill: 'zone-2' },
    { from: 3, to: 5, points: 5, fill: 'zone-5' },
    { from: 6, to: 8, points: 10, fill: 'zone-10' }
  ] as const;

  function fill(space: number): string {
    return ZONES.find((zone) => space >= zone.from && space <= zone.to)?.fill ?? '';
  }

  /** Grid column of a space (1-based: the centre is column 1). */
  const col = (space: number) => space + 1;
</script>

<figure
  class="flex flex-col gap-2 rounded-panel border border-rule bg-panel-sunken p-3"
  data-rules-diagram="military"
>
  <div dir="ltr" class="track grid w-full max-w-sm gap-y-1 self-center" aria-hidden="true">
    <!-- Row 1: the looting tokens above spaces 3 and 6. -->
    <div class="row-start-1 grid place-items-center" style:grid-column={col(3)}>
      <LootingToken value={2} magnitude={3} present />
    </div>
    <div class="row-start-1 grid place-items-center" style:grid-column={col(6)}>
      <LootingToken value={5} magnitude={6} present />
    </div>

    <!-- Row 2: the spaces, shaded by zone, the pawn and the capital. -->
    {#each SPACES as space (space)}
      <div
        class="row-start-2 grid aspect-square place-items-center rounded-[0.3rem] border border-rule {fill(
          space
        )}"
        class:bg-panel={space === 0}
        style:grid-column={col(space)}
      >
        {#if space === 0}
          <Icon name="military-pawn" class="text-[0.8rem] text-fg" />
        {:else if space === 9}
          <Icon name="capital-moon" class="text-[0.95rem] text-moon-ink" />
        {/if}
      </div>
    {/each}

    <!-- Row 3: the space numbers. -->
    {#each SPACES as space (space)}
      <span
        class="row-start-3 text-center text-[10px] leading-none text-fg-muted tabular-nums"
        data-min-font="10"
        style:grid-column={col(space)}>{formatNumber(space)}</span
      >
    {/each}

    <!-- Row 4: what each zone is worth. -->
    {#each ZONES as zone (zone.points)}
      <span
        class="zone-label row-start-4 text-center text-[11px] leading-tight font-semibold text-fg"
        style:grid-column="{col(zone.from)} / {col(zone.to) + 1}"
      >
        {t('rules.diagram.points', { count: zone.points })}
      </span>
    {/each}
    <span
      class="row-start-4 text-center text-[11px] leading-tight font-semibold text-moon-ink"
      style:grid-column={col(9)}
    >
      {t('rules.diagram.win')}
    </span>
  </div>
  <figcaption class="text-xs leading-snug text-fg-muted">{t('rules.diagram.military')}</figcaption>
</figure>

<style>
  .track {
    grid-template-columns: repeat(10, minmax(0, 1fr));
  }
  .zone-2 {
    background: color-mix(in oklch, var(--cinnabar) 14%, var(--panel));
  }
  .zone-5 {
    background: color-mix(in oklch, var(--cinnabar) 30%, var(--panel));
  }
  .zone-10 {
    background: color-mix(in oklch, var(--cinnabar) 50%, var(--panel));
  }
  .zone-label {
    border-block-start: 2px solid var(--rule-strong);
    padding-block-start: 0.125rem;
    margin-inline: 0.125rem;
  }
</style>
