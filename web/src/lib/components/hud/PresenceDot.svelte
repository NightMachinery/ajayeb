<script lang="ts">
  // A player's presence in an online room (UX design "Connection states"):
  // a filled green dot while they are connected, and a hollow dashed ring
  // with the word "away" once they drop. The shape and the word tell the two
  // apart without colour, so it reads in greyscale too. A leaf: the caller
  // passes the presence from the room's seats.
  import { t } from '$lib/i18n';
  import type { Presence } from '$lib/ui/online';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    presence: Presence;
    /** The player's display name, for the accessible label. */
    name: string;
    /** Show the word next to the dot even while connected. */
    labelled?: boolean;
    class?: string;
  }

  let { presence, name, labelled = false, class: className = '' }: Props = $props();

  const away = $derived(presence === 'away');
  const label = $derived(
    away ? t('online.presence.awayLabel', { name }) : t('online.presence.connectedLabel', { name })
  );
</script>

<span
  class="presence inline-flex shrink-0 items-center gap-1 text-xs font-semibold {away
    ? 'rounded-full bg-saffron/20 px-1.5 py-0.5 text-fg'
    : 'text-fg-muted'} {className}"
  data-presence={presence}
  {@attach tooltip({ text: label })}
>
  <svg class="size-2.5 shrink-0" viewBox="0 0 10 10" aria-hidden="true">
    {#if away}
      <circle
        cx="5"
        cy="5"
        r="3.6"
        fill="none"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-dasharray="2.2 1.6"
      />
    {:else}
      <circle cx="5" cy="5" r="4.2" class="fill-malachite" />
    {/if}
  </svg>
  {#if away || labelled}
    <span aria-hidden="true">
      {away ? t('online.presence.away') : t('online.presence.connected')}
    </span>
  {/if}
  <span class="sr-only">{label}</span>
</span>
