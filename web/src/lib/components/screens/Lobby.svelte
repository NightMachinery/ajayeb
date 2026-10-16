<script lang="ts">
  // The online lobby (UX design section 2, "Online lobby / waiting room"):
  // the room before the second player arrives. The room code in large type,
  // the invite link with Copy and Share, a QR code of the link, and the two
  // seats: this player's and the open one the friend will take. The game
  // screen swaps it for the wonder draft as soon as the room's status turns
  // to playing, so it needs no button to start.
  //
  // A leaf: the caller passes the code, the link, the room's seats and the
  // connection, and handles Copy, Share and Home (the clipboard, the share
  // sheet and the toasts belong to the game screen). The QR encoder (uqr)
  // is imported here, lazily, so it loads with the lobby and nowhere else.
  //
  // Under the players, a read-only summary of the game's hint settings
  // (docs/dev/decisions.md D23), so they are known before the game starts:
  // the `hints` prop, or else the room's own from the page's game context.
  import ArrowLeft from '@lucide/svelte/icons/arrow-left';
  import Copy from '@lucide/svelte/icons/copy';
  import Share2 from '@lucide/svelte/icons/share-2';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { qrPath, type Presence } from '$lib/ui/online';
  import { seatDisplayName } from '$lib/view/display-name';
  import type { HintSettings, SeatInfo } from '$lib/wire';
  import { tryGameContext } from '../choices/room';
  import HintsSummary from '../hints/HintsSummary.svelte';
  import SeatLink from '../identity/SeatLink.svelte';
  import SelfCrest from '../identity/SelfCrest.svelte';
  import SelfMark from '../identity/SelfMark.svelte';
  import ConnectionPill from '../hud/ConnectionPill.svelte';
  import PresenceDot from '../hud/PresenceDot.svelte';
  import ThinkingDots from '../hud/ThinkingDots.svelte';
  import Button from '../shell/Button.svelte';

  interface Props {
    code: string;
    /** The invite link: the page's origin and /play/<code>. */
    url: string;
    /** The room's seats (RoomInfo.seats). */
    seats: readonly SeatInfo[];
    conn?: 'connecting' | 'open' | 'reconnecting' | 'offline';
    /** The browser has a share sheet (navigator.share). */
    canShare?: boolean;
    oncopy: () => void;
    onshare?: () => void;
    onhome: () => void;
    /** The game's hint settings; by default the room's (the game context). */
    hints?: HintSettings | null;
  }

  let {
    code,
    url,
    seats,
    conn = 'open',
    canShare = false,
    oncopy,
    onshare,
    onhome,
    hints
  }: Props = $props();

  const ctx = tryGameContext();
  const shownHints = $derived(hints ?? ctx?.session.hints ?? null);

  interface Qr {
    size: number;
    path: string;
  }

  async function encodeQr(text: string): Promise<Qr> {
    const { encode } = await import('uqr');
    const qr = encode(text, { ecc: 'M', border: 0 });
    return { size: qr.size, path: qrPath(qr.data) };
  }

  // A promise, rendered with an await block: no state is written.
  const qr = $derived(encodeQr(url));

  const SEATS = [0, 1] as const;

  /** A div and label[for], not a wrapping label: the app's ring for an
   *  input inside a label is meant for hidden radio inputs. */
  const uid = $props.id();
  const linkId = `${uid}-link`;
  const codeLabelId = `${uid}-code-label`;
  const playersId = `${uid}-players`;

  function seatName(seat: 0 | 1): string {
    return seatDisplayName(
      seats.find((s) => s.seat === seat),
      t(seat === 0 ? 'board.crest.sun' : 'board.crest.moon')
    );
  }

  function presenceOf(info: SeatInfo | undefined): Presence | undefined {
    if (!info || info.kind !== 'human') return undefined;
    return info.connected ? 'connected' : 'away';
  }

  function selectAll(event: Event): void {
    (event.currentTarget as HTMLInputElement).select();
  }
</script>

<main class="lobby h-full overflow-y-auto" data-lobby={code}>
  <div
    class="mx-auto flex min-h-full w-full max-w-md flex-col gap-4 ps-[max(1rem,var(--safe-l))] pe-[max(1rem,var(--safe-r))]
      pt-[max(0.75rem,var(--safe-t))] pb-[max(1.5rem,var(--safe-b))] split:max-w-3xl"
  >
    <header class="flex items-center justify-between gap-2">
      <button
        type="button"
        class="-ms-2 flex min-h-11 items-center gap-1.5 rounded-control px-2 text-sm font-semibold text-fg hover:bg-panel-sunken"
        data-control="lobby-home"
        onclick={onhome}
      >
        <ArrowLeft class="size-5 rtl:-scale-x-100" aria-hidden="true" />
        {t('online.lobby.home')}
      </button>
      <ConnectionPill {conn} />
    </header>

    <div>
      <h1 class="font-display text-3xl text-fg">{t('online.lobby.title')}</h1>
      <p class="mt-1 text-fg-muted">{t('online.lobby.intro')}</p>
    </div>

    <section
      class="grid gap-4 rounded-panel border border-rule bg-panel p-4 shadow-panel split:grid-cols-[minmax(0,1fr)_auto] split:items-center split:gap-6"
      aria-labelledby={codeLabelId}
    >
      <div class="flex min-w-0 flex-col gap-3">
        <div class="flex flex-col items-center gap-1 split:items-start">
          <span
            id={codeLabelId}
            class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
          >
            {t('online.lobby.code')}
          </span>
          <span
            class="font-display text-5xl leading-none tracking-[0.18em] text-fg tabular-nums"
            data-room-code={code}
            dir="ltr"
          >
            {code}
          </span>
        </div>

        <div class="flex flex-col gap-1">
          <label for={linkId} class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
            {t('online.lobby.link')}
          </label>
          <input
            class="min-h-11 w-full rounded-control border border-rule bg-panel-sunken px-3 text-sm text-fg"
            type="text"
            id={linkId}
            readonly
            value={url}
            dir="ltr"
            data-invite-link
            onfocus={selectAll}
          />
        </div>

        <div class="grid gap-2 {canShare && onshare ? 'grid-cols-2' : 'grid-cols-1'}">
          <Button variant="primary" full onclick={oncopy} data-control="copy-link">
            <Copy class="size-4" aria-hidden="true" />
            {t('online.lobby.copy')}
          </Button>
          {#if canShare && onshare}
            <Button variant="secondary" full onclick={onshare} data-control="share-link">
              <Share2 class="size-4" aria-hidden="true" />
              {t('online.lobby.share')}
            </Button>
          {/if}
        </div>
      </div>

      <figure class="flex flex-col items-center gap-2">
        <div
          class="grid size-44 place-items-center rounded-control bg-card-paper p-3 shadow-panel"
          data-qr
          {@attach tooltip({ text: t('online.lobby.qr') })}
        >
          {#await qr}
            <span class="size-full rounded-[4px] bg-panel-sunken/40" aria-hidden="true"></span>
          {:then code}
            <svg
              class="size-full text-card-ink"
              viewBox="0 0 {code.size} {code.size}"
              shape-rendering="crispEdges"
              role="img"
              aria-label={t('online.lobby.qr')}
            >
              <path d={code.path} fill="currentColor" />
            </svg>
          {/await}
        </div>
        <figcaption class="text-center text-sm text-fg-muted">{t('online.lobby.scan')}</figcaption>
      </figure>
    </section>

    <section class="flex flex-col gap-2" aria-labelledby={playersId}>
      <h2 id={playersId} class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
        {t('online.lobby.players')}
      </h2>
      <ul class="flex flex-col gap-2">
        {#each SEATS as seat (seat)}
          {@const info = seats.find((s) => s.seat === seat)}
          {@const presence = presenceOf(info)}
          <li
            class="flex min-h-12 items-center gap-2 rounded-control border px-3 py-2
              {info?.kind === 'open' ? 'border-dashed border-rule-strong' : 'border-rule bg-panel'}"
            data-lobby-seat={seat}
            data-open={info?.kind === 'open' ? '' : undefined}
          >
            <SelfCrest {seat} self={!!info?.you} class="text-xl" />
            {#if info?.kind === 'open'}
              <span class="flex min-w-0 flex-1 items-center gap-2 text-sm text-fg-muted">
                <ThinkingDots class={seat === 0 ? 'text-sun-ink' : 'text-moon-ink'} />
                {t('online.lobby.waiting')}
              </span>
            {:else}
              <span class="flex min-w-0 flex-1 items-center gap-1.5">
                <span class="min-w-0 truncate text-sm font-semibold text-fg" dir="auto">
                  {seatName(seat)}
                </span>
                {#if info?.you}<SelfMark />{/if}
              </span>
              <SeatLink {seat} name={seatName(seat)} />
              {#if presence}
                <PresenceDot {presence} name={seatName(seat)} labelled />
              {/if}
            {/if}
          </li>
        {/each}
      </ul>
    </section>

    {#if shownHints}
      <HintsSummary hints={shownHints} title={t('hints.lobby.title')} />
    {/if}

    <p class="mt-auto text-center text-xs text-fg-muted">{t('online.lobby.homeHint')}</p>
  </div>
</main>
