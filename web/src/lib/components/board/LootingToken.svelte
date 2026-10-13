<script lang="ts">
  // One looting token on the military track, drawn as what it does: a gold
  // coin chip reading −2 or −5 (formatSigned: U+2212), the coins its side
  // loses as soon as the pawn enters its space (3 or 6 from the centre).
  // Once taken it leaves a faint dashed ring where it stood. A leaf
  // component, plain props, so MilitaryTrack just maps view.military.looting
  // over it.
  import { formatSigned, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    /** Coins lost when this token triggers (printed on the token itself). */
    value: number;
    /** Its space on the track, for the label only (sign already tells the
     *  caller which side; this component shows the magnitude). */
    magnitude: number;
    /** Still on the track (true) or already claimed (false, shown ghosted). */
    present: boolean;
    size?: 'sm' | 'md';
    /** A fuller tooltip (the track's looting hint, M10.HS4); else the label. */
    tip?: string;
    class?: string;
  }

  let { value, magnitude, present, size = 'md', tip, class: className = '' }: Props = $props();

  const label = $derived(
    t(present ? 'board.military.lootCell' : 'board.military.lootTaken', { value, magnitude })
  );
</script>

<span
  class="loot-token inline-grid shrink-0 place-items-center rounded-full leading-none font-bold tabular-nums
    {size === 'sm' ? 'h-3.5 min-w-3.5 px-px text-[10px]' : 'h-4 min-w-4 px-0.5 text-[10px]'}
    {present
    ? 'border border-(color:--gold-deep) bg-gold text-card-ink'
    : 'border border-dashed border-fg-muted/60 bg-transparent'} {className}"
  role="img"
  {@attach tooltip({ text: tip ?? label })}
  aria-label={tip ? `${label}. ${tip}` : label}
  data-present={present ? '' : undefined}
  data-min-font="10"
>
  {#if present}{formatSigned(-value)}{/if}
</span>
