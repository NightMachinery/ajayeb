<script lang="ts">
  // The compact, icon-only button beside a player's name that opens the
  // seat's device link (DeviceLinkSheet): a link that plays the seat on
  // another device. Its label is the accessible name, and the tooltip
  // helper shows the same text (never a native title attribute). A leaf:
  // plain props.
  import MonitorSmartphone from '@lucide/svelte/icons/monitor-smartphone';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    seat: 0 | 1;
    /** The accessible name, already translated. */
    label: string;
    onclick: () => void;
    class?: string;
  }

  let { seat, label, onclick, class: className = '' }: Props = $props();
</script>

<button
  type="button"
  class="grid size-8 shrink-0 place-items-center rounded-control text-fg-muted hover:bg-panel-sunken hover:text-fg {className}"
  aria-label={label}
  aria-haspopup="dialog"
  data-control="device-link"
  data-seat={seat}
  {onclick}
  {@attach tooltip({ text: label })}
>
  <MonitorSmartphone class="size-4" aria-hidden="true" />
</button>
