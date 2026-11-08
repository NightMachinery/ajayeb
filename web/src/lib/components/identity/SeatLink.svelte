<script lang="ts">
  // The device link of one seat, where the room page shows it (the strips,
  // the city panels, the lobby): the compact button beside the name, and
  // its sheet. It shows for the player's own seat, and for every player's
  // seat to the game's admin, never to a spectator, a bot's seat or an open
  // one; in a hot-seat room only once, by the first seat (both seats are
  // the same player). Without a game context (the gallery) it renders
  // nothing.
  //
  // Opening it asks the server for the seat's claim (Session.requestLink),
  // builds the link (/play/<code>#claim=<claim>) and copies it at once,
  // with the shared clipboard helper; when that fails the link is selected in the
  // sheet to copy by hand. Reset asks for a fresh claim, which stops every
  // earlier link.
  import { t } from '$lib/i18n';
  import { tryGameContext } from '../choices/room';
  import DeviceLinkButton from './DeviceLinkButton.svelte';

  interface Props {
    seat: 0 | 1;
    /** The seat's display name. */
    name: string;
    class?: string;
  }

  let { seat, name, class: className = '' }: Props = $props();

  const ctx = tryGameContext();
  const frame = $derived(ctx?.session.frame ?? null);
  const info = $derived(frame?.room.seats.find((s) => s.seat === seat));
  const own = $derived(!!info?.you);
  const shown = $derived.by(() => {
    if (!frame || info?.kind !== 'human') return false;
    if (frame.you.seats.length === 0) return false;
    if (frame.room.mode === 'hotseat' && seat !== 0) return false;
    return own || (frame.you.linkAdmin ?? frame.you.admin);
  });

  // Loading the interaction only after a tap keeps its shared chunk out of
  // Duel's cold entry. The leaf button and permissions remain unchanged.
  let open = $state(false);
</script>

{#if shown}
  <DeviceLinkButton
    {seat}
    class={className}
    label={own ? t('online.link.buttonOwn') : t('online.link.button', { name })}
    onclick={() => (open = true)}
  />
  {#if open && ctx?.session.code}
    {#await import('./DeviceLinkControl.svelte') then { default: DeviceLinkControl }}
      <DeviceLinkControl
        bind:open
        code={ctx.session.code}
        {seat}
        {name}
        {own}
        trigger={false}
        autoOpen
        request={(reset) => (reset ? ctx!.session.resetLink(seat) : ctx!.session.requestLink(seat))}
      />
    {:catch}
      <p role="alert" class="text-sm text-cinnabar-ink">{t('online.link.failed')}</p>
    {/await}
  {/if}
{/if}
