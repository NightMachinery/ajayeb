<script lang="ts">
  // One looting token on the military track: a coin-loss marker at a fixed
  // space (3 or 6 from the centre, on either side) that pays out once the
  // pawn reaches it and is then gone. A leaf component, plain props, so
  // MilitaryTrack just maps view.military.looting over it.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';

  interface Props {
    /** Coins lost when this token triggers (printed on the token itself). */
    value: number;
    /** Its space on the track, for the label only (sign already tells the
     *  caller which side; this component shows the magnitude). */
    magnitude: number;
    /** Still on the track (true) or already claimed (false, shown ghosted). */
    present: boolean;
    size?: 'sm' | 'md';
    class?: string;
  }

  let { value, magnitude, present, size = 'md', class: className = '' }: Props = $props();

  const label = $derived(
    t(present ? 'board.military.lootCell' : 'board.military.lootTaken', { value, magnitude })
  );
</script>

<div
  class="loot-token grid shrink-0 place-items-center rounded-full border {size === 'sm'
    ? 'size-4'
    : 'size-6'} {present
    ? 'border-rule-strong bg-panel text-fg'
    : 'border-rule/50 bg-panel-sunken text-fg-muted opacity-50'} {className}"
  role="img"
  title={label}
  aria-label={label}
>
  <Icon
    name={value <= 2 ? 'looting-2' : 'looting-5'}
    class={size === 'sm' ? 'text-[10px]' : 'text-xs'}
  />
</div>
