<script lang="ts">
  // One of a seat's two running numbers, drawn the way the cards draw them
  // so it reads at a glance without a legend: coins as a gold coin with the
  // amount on it, victory points as the green laurel tablet with the number
  // (the counterparts of a card's coin disc and points tablet, which also
  // differ in greyscale: dark on gold, light on green). A leaf component:
  // the number comes from the view; `label` is its spoken form, given by
  // the caller through t(). Its tooltip is the spoken form, or `tip` (the
  // strip's and the city panel's, with the scoring hints as status lines).
  import { formatNumber } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { tooltip, type TooltipContent } from '$lib/ui/tooltip.svelte';

  interface Props {
    kind: 'coins' | 'points';
    value: number;
    /** The spoken form, such as "14 coins" or "If the game ended now: 7 points". */
    label: string;
    size?: 'sm' | 'md';
    /** The tooltip; by default the spoken form. */
    tip?: TooltipContent;
    class?: string;
  }

  let { kind, value, label, size = 'sm', tip, class: className = '' }: Props = $props();
</script>

<span
  class="tally inline-flex shrink-0 items-center gap-[0.2em] border leading-none font-bold tabular-nums
    {size === 'sm' ? 'h-5 px-1.5 text-[13px]' : 'h-6 px-2 text-sm'}
    {kind === 'coins'
    ? 'rounded-full border-(color:--gold-deep) bg-gold text-card-ink'
    : 'rounded-[0.45em] border-vp-edge bg-vp text-(--paper-bright)'} {className}"
  data-tally={kind}
  role="img"
  aria-label={label}
  {@attach tooltip(tip ?? { text: label })}
>
  {#if kind === 'points'}
    <Icon name="vp-laurel" class="text-[0.95em]" />
  {:else}
    <Icon name="coin" class="text-[0.95em] text-(--gold-deep)" />
  {/if}
  <span data-must-fit data-fx-count>{formatNumber(value)}</span>
</span>
