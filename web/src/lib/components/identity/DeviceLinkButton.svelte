<script lang="ts">
  // The compact, icon-only button beside a player's name that opens the
  // seat's device link (DeviceLinkSheet): a link that plays the seat on
  // another device. Its label is the accessible name, and the tooltip
  // helper shows the same text (never a native title attribute). Its real
  // 44 px box keeps the hit area clear of neighbouring controls. A leaf:
  // plain props.
  import MonitorSmartphone from '@lucide/svelte/icons/monitor-smartphone';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    seat: number | string;
    userId?: string;
    /** The accessible name, already translated. */
    label: string;
    onclick: () => void;
    class?: string;
  }

  let { seat, userId, label, onclick, class: className = '' }: Props = $props();
</script>

<button
  type="button"
  class="grid size-11 shrink-0 place-items-center rounded-control text-fg-muted
    hover:bg-panel-sunken hover:text-fg {className}"
  aria-label={label}
  aria-haspopup="dialog"
  data-control="device-link"
  data-seat={userId === undefined ? seat : undefined}
  data-device-link-user={userId}
  data-act={userId === undefined ? undefined : `device-link:${userId}`}
  {onclick}
  {@attach tooltip({ text: label })}
>
  <MonitorSmartphone class="size-4" aria-hidden="true" />
</button>
