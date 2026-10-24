<script lang="ts">
  // The lobby renders public roster and role metadata. The actor owns every
  // permission, option and seat change, including concurrent edits and Start.
  import type { Attachment } from 'svelte/attachments';
  import { resolve } from '$app/paths';
  import { t as uiT, formatNumber } from '$lib/i18n';
  import { t } from './i18n/messages';
  import Button from '$lib/components/shell/Button.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import HintsSummary from '$lib/components/hints/HintsSummary.svelte';
  import { defaultHints } from '$lib/hints/families';
  import { qrPath } from '$lib/ui/online';
  import type {
    PlatformHuman,
    PlatformSeat,
    LobbyEdit,
    LobbyOptions
  } from '$lib/wire/platform.gen';
  import type { TableSession } from '$lib/games/haftgane/session/session.svelte';

  let { session }: { session: TableSession } = $props();
  const lobby = $derived(session.lobby!);
  const ready = $derived(session.conn === 'open' && !session.pending);
  const spectators = $derived(
    lobby.humans.filter(
      (h) =>
        !lobby.seats.some(
          (s) => s.userId === h.userId && (s.kind === 'human' || s.kind === 'reserved')
        )
    )
  );
  const opens = $derived(lobby.seats.some((s) => s.kind === 'open'));
  // The server allows two seats only while Haftgane tables are off; a lobby
  // made larger before that keeps its own count among the choices.
  const seatChoices = $derived(
    [2, 3, 4, 5, 6, 7].filter((n) => n <= lobby.maxSeats || n === lobby.seatCount)
  );
  const reserved = $derived(lobby.seats.some((s) => s.kind === 'reserved'));
  const undoEligible = $derived(
    lobby.seats.filter((s) => s.kind === 'human').length === 1 && !!session.me?.creator
  );
  const url = $derived(
    typeof location === 'undefined'
      ? ''
      : new URL(resolve('/play/[code]', { code: lobby.code }), location.origin).href
  );
  let copied = $state<'idle' | 'done' | 'failed'>('idle');
  let showQr = $state(false);
  let botLevels = $state<Record<number, string>>({});
  let confirmation = $state<(() => void) | null>(null);

  // These selects have fixed options. Keep value updates in this lazy chunk:
  // select value attributes export extra shared-runtime helpers and rename
  // imports throughout Duel, increasing its gzip size without adding modules.
  function selectValue(value: string | number): Attachment<HTMLSelectElement> {
    return (node) => {
      node.value = String(value);
    };
  }

  function name(h: PlatformHuman | undefined): string {
    return h ? h.name + (h.nameNo > 1 ? ` ${formatNumber(h.nameNo)}` : '') : '';
  }
  function botName(seat: number, level: string | undefined): string {
    const persona = uiT(
      `game.persona.${level === 'hard' ? 'hard' : level === 'normal' ? 'normal' : 'easy'}`
    );
    const peers = lobby.seats.filter((s) => s.kind === 'bot' && s.bot === level);
    return peers.length > 1
      ? `${persona} ${formatNumber(peers.findIndex((s) => s.seat === seat) + 1)}`
      : persona;
  }
  function edit(payload: Omit<LobbyEdit, 't' | 'code' | 'seq'>): void {
    session.editLobby(payload);
  }
  function option(patch: Partial<LobbyOptions>): void {
    edit({ op: 'options', options: { ...lobby.options, ...patch } });
  }
  function count(n: number): void {
    const change = () => edit({ op: 'seats', seatCount: n });
    if (lobby.seats.some((s) => s.seat >= n && s.kind === 'human')) confirmation = change;
    else change();
  }
  async function copy(): Promise<void> {
    try {
      await navigator.clipboard.writeText(url);
      copied = 'done';
    } catch {
      copied = 'failed';
    }
  }
  async function share(): Promise<void> {
    try {
      await navigator.share({ url });
    } catch {
      /* Cancel leaves the link available. */
    }
  }
  async function qr(text: string) {
    const { encode } = await import('uqr');
    const data = encode(text, { ecc: 'M', border: 0 });
    return { size: data.size, path: qrPath(data.data) };
  }
  const qrCode = $derived(showQr ? qr(url) : null);
</script>

{#snippet botLevel(seat: PlatformSeat)}
  <label class="flex flex-wrap items-center justify-between gap-2">
    <span>{t('lobby.botLevel')}</span>
    <select
      class="min-h-11 rounded-control border border-rule bg-panel px-3 text-fg"
      {@attach selectValue(
        seat.kind === 'bot' ? (seat.bot ?? 'easy') : (botLevels[seat.seat] ?? 'easy')
      )}
      disabled={!ready}
      data-act={`lobby-bot-level:${seat.seat}`}
      onchange={(e) => {
        const bot = e.currentTarget.value;
        if (seat.kind === 'bot') edit({ op: 'add-bot', seat: seat.seat, bot });
        else botLevels[seat.seat] = bot;
      }}
    >
      <option value="easy">{uiT('game.persona.easy')}</option>
      <option value="normal">{uiT('game.persona.normal')}</option>
      {#if lobby.seatCount === 2}<option value="hard">{uiT('game.persona.hard')}</option>{/if}
    </select>
  </label>
{/snippet}
{#snippet person(h: PlatformHuman)}
  <span class="min-w-0 font-semibold break-words" dir="auto">{name(h)}</span>
  <span class="flex flex-wrap gap-1 text-xs text-fg-muted" data-role-tags={h.userId}>
    {#if h.userId === lobby.you.userId}<span>{t('lobby.you')}</span>{/if}
    {#if h.creator}<span>{t('lobby.creator')}</span>{/if}
    {#if h.admin}<span data-admin>{t('lobby.admin')}</span>{/if}
    {#if h.temporary}<span>{t('lobby.temporary')}</span>{/if}
    <span>{t(h.connected ? 'lobby.connected' : 'lobby.away')}</span>
  </span>
{/snippet}

{#snippet roles(h: PlatformHuman)}
  {#if session.me?.admin && h.userId !== lobby.you.userId}
    {#if !h.admin}
      <Button
        variant="secondary"
        disabled={!ready}
        data-act={`lobby-promote:${h.userId}`}
        onclick={() => session.lobbyRole('promote', h.userId)}>{t('lobby.promote')}</Button
      >
    {:else if !h.creator && (session.me.creator || h.promotedBy === lobby.you.userId)}
      <Button
        variant="secondary"
        disabled={!ready}
        data-act={`lobby-demote:${h.userId}`}
        onclick={() => session.lobbyRole('demote', h.userId)}>{t('lobby.demote')}</Button
      >
    {/if}
  {/if}
{/snippet}

<main class="h-full overflow-y-auto bg-table text-fg" data-lobby={lobby.code}>
  <div class="mx-auto flex max-w-5xl flex-col gap-5 p-4 sm:p-6 short:gap-3">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="font-display text-2xl">
          {t(lobby.seatCount === 2 ? 'lobby.duelLobby' : 'lobby.title')}
        </h1>
        <p class="text-sm text-fg-muted" data-room-code={lobby.code}>
          {t('lobby.code', { code: lobby.code })}
        </p>
      </div>
      <a
        href={resolve('/')}
        class="inline-flex min-h-11 items-center rounded-control border border-rule bg-panel px-4 font-semibold"
        data-act="lobby-home">{t('lobby.home')}</a
      >
    </header>
    <p role="status" class="text-sm text-fg-muted" data-conn={session.conn}>
      {session.pending ? t('lobby.pending') : uiT(`session.conn.${session.conn}`)}
    </p>
    {#if session.notice?.kind === 'reject'}
      <div
        role="alert"
        class="rounded-panel border border-rule bg-panel p-3"
        data-notice={session.notice.code}
      >
        <p>{t('lobby.failed')}</p>
        <Button variant="secondary" data-act="lobby-dismiss" onclick={() => session.dismissNotice()}
          >{t('lobby.cancel')}</Button
        >
      </div>
    {/if}
    <div class="grid gap-5 split:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
      <section
        class="min-w-0 rounded-panel border border-rule bg-panel p-4"
        aria-labelledby="lobby-seats"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 class="font-display text-xl" id="lobby-seats">{t('lobby.seats')}</h2>
          {#if seatChoices.length > 1}
            <label class="flex items-center gap-2 text-sm font-semibold">
              <span class="sr-only">{t('lobby.seats')}</span>
              <select
                class="min-h-11 min-w-16 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
                {@attach selectValue(lobby.seatCount)}
                disabled={!ready || !session.admin}
                onchange={(e) => {
                  const next = Number(e.currentTarget.value);
                  e.currentTarget.value = String(lobby.seatCount);
                  count(next);
                }}
                data-act="lobby-seats"
              >
                {#each seatChoices as n (n)}<option value={n}>{formatNumber(n)}</option>{/each}
              </select>
            </label>
          {/if}
        </div>
        {#if lobby.seatCount > 2}<p class="mt-2 text-xs text-fg-muted">{t('lobby.order')}</p>{/if}
        <ol class="mt-4 flex flex-col gap-2">
          {#each lobby.seats as seat (seat.seat)}
            {@const h = lobby.humans.find((h) => h.userId === seat.userId)}
            <li
              class="min-w-0 rounded-control border border-rule bg-panel-sunken p-3"
              data-seat={seat.seat}
              data-kind={seat.kind}
            >
              <div class="flex min-h-11 items-center gap-3">
                <span
                  class="flex size-11 shrink-0 items-center justify-center rounded-full border border-rule-strong font-display text-xl"
                  {@attach tooltip({ text: t('lobby.seat', { seat: seat.seat + 1 }) })}
                  >{formatNumber(seat.seat + 1)}</span
                >
                <div class="min-w-0 flex-1">
                  {#if seat.kind === 'human' && h}{@render person(h)}
                  {:else if seat.kind === 'bot'}<p class="font-semibold">
                      {t('lobby.bot', {
                        name: botName(seat.seat, seat.bot)
                      })}
                    </p>
                  {:else}<p class="text-sm text-fg-muted">
                      {t(seat.kind === 'reserved' ? 'lobby.reserved' : 'lobby.open')}
                    </p>{/if}
                </div>
              </div>
              {#if session.admin || h?.userId === lobby.you.userId}
                <details class="mt-1">
                  <summary
                    class="flex min-h-11 cursor-pointer items-center rounded-control px-2 text-sm font-semibold text-accent"
                    data-act={`lobby-seat-menu:${seat.seat}`}
                    >{h ? t('lobby.manage', { name: name(h) }) : t('lobby.controls')}</summary
                  >
                  <div class="flex flex-col gap-2 pt-2">
                    {#if session.admin && seat.kind === 'open'}
                      {@render botLevel(seat)}
                      <Button
                        variant="secondary"
                        disabled={!ready}
                        data-act={`lobby-add-bot:${seat.seat}`}
                        onclick={() =>
                          edit({
                            op: 'add-bot',
                            seat: seat.seat,
                            bot: botLevels[seat.seat] ?? 'easy'
                          })}>{t('lobby.addBot')}</Button
                      >
                      <Button
                        variant="secondary"
                        data-act={`lobby-invite:${seat.seat}`}
                        onclick={copy}>{t('lobby.copy')}</Button
                      >
                      {#each spectators.filter((h) => h.connected) as sp (sp.userId)}<Button
                          variant="secondary"
                          disabled={!ready}
                          data-act={`lobby-seat-user:${seat.seat}:${sp.userId}`}
                          onclick={() => edit({ op: 'seat', seat: seat.seat, userId: sp.userId })}
                          >{t('lobby.seatHuman', { name: name(sp) })}</Button
                        >{/each}
                    {:else if session.admin && seat.kind === 'bot'}
                      {#if lobby.seatCount === 2}{@render botLevel(seat)}{/if}
                      <Button
                        variant="secondary"
                        disabled={!ready}
                        data-act={`lobby-remove-bot:${seat.seat}`}
                        onclick={() => edit({ op: 'remove-bot', seat: seat.seat })}
                        >{t('lobby.removeBot')}</Button
                      >
                    {:else if session.admin && seat.kind === 'reserved'}
                      <Button
                        variant="secondary"
                        disabled={!ready}
                        data-act={`lobby-open:${seat.seat}`}
                        onclick={() => edit({ op: 'open-reservation', seat: seat.seat })}
                        >{t('lobby.openReservation')}</Button
                      >
                    {:else if h}
                      {@render roles(h)}
                      {#if h.userId === lobby.you.userId}<Button
                          variant="secondary"
                          disabled={!ready}
                          data-act="lobby-release"
                          onclick={() => edit({ op: 'release' })}>{t('lobby.watch')}</Button
                        >
                      {:else if session.admin}<Button
                          variant="secondary"
                          disabled={!ready}
                          data-act={`lobby-remove-human:${h.userId}`}
                          onclick={() => edit({ op: 'remove-human', userId: h.userId })}
                          >{t('lobby.removeHuman')}</Button
                        >{/if}
                    {/if}
                  </div>
                </details>
              {/if}
            </li>
          {/each}
        </ol>
      </section>
      <div class="flex min-w-0 flex-col gap-4">
        <section
          class="rounded-panel border border-rule bg-panel p-4"
          aria-labelledby="lobby-invite"
        >
          <h2 id="lobby-invite" class="font-display text-xl">{t('lobby.invite')}</h2>
          <div class="mt-3 flex flex-wrap gap-2">
            <Button variant="secondary" data-act="lobby-copy" onclick={copy}
              >{t('lobby.copy')}</Button
            >
            {#if typeof navigator !== 'undefined' && typeof navigator.share === 'function'}<Button
                variant="secondary"
                data-act="lobby-share"
                onclick={share}>{t('lobby.share')}</Button
              >{/if}
            <Button variant="secondary" data-act="lobby-qr" onclick={() => (showQr = !showQr)}
              >{t(showQr ? 'lobby.closeQr' : 'lobby.qr')}</Button
            >
          </div>
          {#if copied !== 'idle'}<p role="status" class="mt-2 text-sm">
              {t(copied === 'done' ? 'lobby.copied' : 'lobby.copyFailed')}
            </p>{/if}
          <input
            class="mt-3 min-h-11 w-full min-w-0 rounded-control border border-rule bg-panel-sunken px-3 text-sm"
            readonly
            value={url}
            aria-label={t('lobby.invite')}
            dir="ltr"
            data-invite-url
          />
          {#if qrCode}{#await qrCode then code}
              <div class="mx-auto mt-3 w-44 max-w-full p-3" style="background: #fff">
                <svg
                  viewBox={`0 0 ${code.size} ${code.size}`}
                  role="img"
                  aria-label={t('lobby.qr')}
                  class="w-full"
                  style="color: #000; background: #fff"
                  {@attach tooltip({ text: t('lobby.qr') })}
                  ><path d={code.path} fill="currentColor" /></svg
                >
              </div>
            {/await}{/if}
        </section>
        <section
          class="rounded-panel border border-rule bg-panel p-4"
          aria-labelledby="lobby-settings"
        >
          <h2 id="lobby-settings" class="font-display text-xl">{t('lobby.settings')}</h2>
          <div class="mt-3 flex flex-col gap-3">
            {#if lobby.seatCount === 2}
              <div class="flex flex-col gap-3" data-duel-settings>
                <label class="flex flex-wrap items-center justify-between gap-2 font-semibold">
                  <span>{t('lobby.firstPlayer')}</span>
                  <select
                    class="min-h-11 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
                    disabled={!ready || !session.admin}
                    data-act="lobby-first-player"
                    {@attach selectValue(lobby.options.firstPlayer ?? 'random')}
                    onchange={(e) => {
                      const first = e.currentTarget.value;
                      const options = { ...lobby.options };
                      if (first === 'random') delete options.firstPlayer;
                      else options.firstPlayer = first === '0' ? 0 : 1;
                      edit({ op: 'options', options });
                    }}
                  >
                    <option value="random">{t('lobby.random')}</option>
                    <option value="0">{uiT('online.crest.sun')}</option>
                    <option value="1">{uiT('online.crest.moon')}</option>
                  </select>
                </label>
                <HintsSummary hints={lobby.options.hints ?? defaultHints('online')} />
              </div>
            {:else}
              <div class="flex flex-col gap-3" data-haftgane-settings>
                <label class="flex flex-wrap items-center justify-between gap-2 font-semibold"
                  ><span>{t('lobby.seating')}</span>
                  <select
                    class="min-h-11 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
                    data-act="lobby-seating"
                    disabled={!ready || !session.admin}
                    {@attach selectValue(lobby.options.seating ?? 'random')}
                    onchange={(e) =>
                      option({ seating: e.currentTarget.value as 'random' | 'lobby' })}
                    ><option value="random">{t('lobby.random')}</option><option value="lobby"
                      >{t('lobby.listed')}</option
                    ></select
                  >
                </label>
                <label class="flex flex-wrap items-center justify-between gap-2 font-semibold">
                  <span>{t('lobby.sides')}</span>
                  <select
                    class="min-h-11 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
                    data-act="lobby-sides"
                    disabled={!ready || !session.admin}
                    {@attach selectValue(lobby.options.sides ?? 'day')}
                    onchange={(e) =>
                      option({ sides: e.currentTarget.value as 'choose' | 'day' | 'random' })}
                  >
                    <option value="choose">{t('lobby.sidesChoose')}</option>
                    <option value="day">{t('lobby.sidesDay')}</option>
                    <option value="random">{t('lobby.random')}</option>
                  </select>
                </label>
                <label class="flex flex-wrap items-center justify-between gap-2 font-semibold"
                  ><span>{t('lobby.end')}</span>
                  <select
                    class="min-h-11 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
                    data-act="lobby-reveal"
                    disabled={!ready || !session.admin}
                    {@attach selectValue(String(lobby.options.finalReveal ?? true))}
                    onchange={(e) => option({ finalReveal: e.currentTarget.value === 'true' })}
                    ><option value="true">{t('lobby.reveal')}</option><option value="false"
                      >{t('lobby.hidden')}</option
                    ></select
                  >
                </label>
                <label class="flex flex-wrap items-center justify-between gap-2 font-semibold"
                  ><span>{t('lobby.undo')}</span>
                  <select
                    class="min-h-11 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
                    data-act="lobby-undo"
                    disabled={!ready || !session.admin || !undoEligible}
                    {@attach selectValue(String(undoEligible && (lobby.options.allowUndo ?? true)))}
                    onchange={(e) => option({ allowUndo: e.currentTarget.value === 'true' })}
                    ><option value="true">{t('lobby.on')}</option><option value="false"
                      >{t('lobby.off')}</option
                    ></select
                  >
                </label>
                <p class="text-xs text-fg-muted">{t('lobby.undoNote')}</p>
              </div>
            {/if}
          </div>
        </section>
        <details class="rounded-panel border border-rule bg-panel p-4" data-spectators>
          <summary
            class="flex min-h-11 cursor-pointer items-center font-semibold"
            data-act="lobby-spectators"
            >{t('lobby.spectators', { count: spectators.length })}</summary
          >
          <ul class="flex flex-col gap-3 pt-2">
            {#each spectators as h (h.userId)}<li class="flex flex-col gap-2">
                {@render person(h)}{@render roles(h)}
              </li>{:else}<li class="text-sm text-fg-muted">{t('lobby.noSpectators')}</li>{/each}
          </ul>
        </details>
        {#if session.me?.admin}
          <details class="rounded-panel border border-rule bg-panel p-4">
            <summary
              class="flex min-h-11 cursor-pointer items-center font-semibold"
              data-act="lobby-roles">{t('lobby.roles')}</summary
            ><Button
              variant="secondary"
              disabled={!ready}
              data-act="lobby-step-down"
              onclick={() => session.lobbyRole('step-down')}>{t('lobby.stepDown')}</Button
            >
          </details>
        {/if}
        <div class="flex flex-col gap-2">
          {#if session.admin}<Button
              full
              disabled={!session.canStart}
              data-act="lobby-start"
              onclick={() => session.start(opens)}>{t(opens ? 'lobby.fill' : 'lobby.start')}</Button
            >
            <p class="text-xs text-fg-muted">
              {t(reserved ? 'lobby.waitReservation' : 'lobby.startNote')}
            </p>
          {:else}<p class="rounded-panel border border-rule bg-panel p-4 text-sm text-fg-muted">
              {t('lobby.guest')}
            </p>{/if}
        </div>
      </div>
    </div>
  </div>
</main>

<!-- Each confirmation owns its dialog, including any queued native close event. -->
{#key confirmation}
  <Sheet
    bind:open={
      () => confirmation !== null,
      (open) => {
        if (!open) confirmation = null;
      }
    }
    titleId="lobby-confirm-title"
  >
    <div class="flex flex-col gap-4 p-5">
      <h2 id="lobby-confirm-title" class="font-display text-xl">{t('lobby.shrink')}</h2>
      <Button
        data-act="lobby-confirm"
        onclick={() => {
          confirmation?.();
          confirmation = null;
        }}>{t('lobby.confirm')}</Button
      ><Button variant="secondary" data-act="lobby-cancel" onclick={() => (confirmation = null)}
        >{t('lobby.cancel')}</Button
      >
    </div>
  </Sheet>
{/key}
