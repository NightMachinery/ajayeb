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
  // with the Clipboard API; when that fails the link is selected in the
  // sheet to copy by hand. Reset asks for a fresh claim, which stops every
  // earlier link.
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import { deviceLink } from '$lib/session/claim';
  import { inviteUrl } from '$lib/ui/online';
  import { tryGameContext } from '../choices/room';
  import DeviceLinkButton from './DeviceLinkButton.svelte';
  import DeviceLinkSheet from './DeviceLinkSheet.svelte';

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
    return own || frame.you.admin;
  });

  let open = $state(false);
  let link = $state<string | null>(null);
  let status = $state<'loading' | 'ready' | 'failed'>('loading');
  let copied = $state<'idle' | 'copied' | 'failed'>('idle');
  let wasReset = $state(false);

  function toLink(code: string, claim: string): string {
    return deviceLink(inviteUrl(location.origin, resolve('/play/[code]', { code })), claim);
  }

  async function fetchLink(reset: boolean): Promise<boolean> {
    const session = ctx?.session;
    const code = session?.code;
    if (!session || !code) return false;
    status = 'loading';
    copied = 'idle';
    wasReset = false;
    try {
      const claim = reset ? await session.resetLink(seat) : await session.requestLink(seat);
      link = toLink(code, claim);
      status = 'ready';
      wasReset = reset;
      return true;
    } catch {
      status = 'failed';
      return false;
    }
  }

  async function copy(): Promise<void> {
    if (!link) return;
    try {
      await navigator.clipboard.writeText(link);
      copied = 'copied';
    } catch {
      copied = 'failed';
    }
  }

  async function openSheet(): Promise<void> {
    open = true;
    link = null;
    if (await fetchLink(false)) await copy();
  }

  async function reset(): Promise<void> {
    if (await fetchLink(true)) await copy();
  }
</script>

{#if shown}
  <DeviceLinkButton
    {seat}
    class={className}
    label={own ? t('online.link.buttonOwn') : t('online.link.button', { name })}
    onclick={openSheet}
  />
  <DeviceLinkSheet
    bind:open
    {seat}
    {name}
    {own}
    {link}
    {status}
    copy={copied}
    reset={wasReset}
    oncopy={copy}
    onreset={reset}
    onretry={() => void fetchLink(false)}
  />
{/if}
