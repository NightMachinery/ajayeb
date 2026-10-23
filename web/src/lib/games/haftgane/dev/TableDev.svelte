<script lang="ts">
  // The minimal development screen of a Haftgane lobby or table, which
  // drives the Session until the table screen (M18.table-screen) and the
  // lobby screens (M18.home-picker-and-lobby) replace it. It renders what
  // the frames say and sends keys: every choice button is one of the
  // server's choices, with its key in data-act. It holds no state of its
  // own beyond what it shows.
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import type { PlatformHuman, PlatformSeat } from '$lib/wire/platform.gen';
  import { ht } from '../i18n/messages.svelte';
  import type { TablePresenter } from '../session/presenter.svelte';
  import type { TableSession } from '../session/session.svelte';

  interface Props {
    session: TableSession;
    presenter: TablePresenter;
  }
  let { session, presenter }: Props = $props();

  const frame = $derived(presenter.frame);
  const lobby = $derived(session.lobby);
  const humans = $derived(frame?.humans ?? lobby?.humans ?? []);
  const me = $derived(frame?.you.userId ?? lobby?.you.userId ?? '');
  const ready = $derived(presenter.idle && frame === session.table);
  // What the frame on screen asks of this seat.
  const deciding = $derived(
    !!frame?.point && frame.you.seat !== undefined && frame.point.required.includes(frame.you.seat)
  );
  // Connected humans without a seat: whom an admin may seat.
  const spectators = $derived(
    humans.filter(
      (h) =>
        h.connected &&
        !(frame?.seats ?? []).some((s) => s.kind === 'human' && s.userId === h.userId)
    )
  );

  function nameOf(userId: string | undefined): string {
    const h = humans.find((x) => x.userId === userId);
    return h ? (h.nameNo > 1 ? `${h.name} ${h.nameNo}` : h.name) : '';
  }

  function seatLabel(s: PlatformSeat): string {
    if (s.kind === 'open') return ht('hg.table.dev.open');
    if (s.kind === 'reserved') return ht('hg.table.dev.reserved');
    if (s.kind === 'bot') return ht('hg.table.dev.bot', { level: s.bot ?? '' });
    return nameOf(s.userId);
  }

  function tags(h: PlatformHuman | undefined): string[] {
    if (!h) return [];
    const out: string[] = [];
    if (h.userId === me) out.push(ht('hg.table.dev.you'));
    if (h.admin) out.push(ht('hg.table.dev.admin'));
    if (h.temporary) out.push(ht('hg.table.dev.temporary'));
    if (!h.connected) out.push(ht('hg.table.dev.away'));
    return out;
  }

  const conn = $derived(session.conn);
</script>

<main class="flex h-full flex-col gap-3 overflow-y-auto p-4 text-fg" data-table-dev>
  <p role="status" class="text-sm text-fg-muted" data-conn={conn}>{t(`session.conn.${conn}`)}</p>

  {#if session.notice}
    <div
      role="alert"
      class="flex items-center gap-3 rounded-panel border border-rule bg-panel p-3"
      data-notice={session.notice.code}
    >
      <p class="flex-1">
        {#if session.notice.kind === 'seated'}
          {ht('hg.table.seat.seated', { seat: session.notice.seat })}
        {:else if session.notice.kind === 'watching'}
          {ht('hg.table.seat.watching')}
        {:else}
          {ht(session.notice.message)}
        {/if}
      </p>
      <Button variant="secondary" onclick={() => session.dismissNotice()} data-act="dismiss">
        {ht('hg.table.dev.dismiss')}
      </Button>
    </div>
  {/if}

  {#if frame}
    <h1 class="font-display text-xl">{ht('hg.table.dev.table', { code: frame.code })}</h1>
    <section aria-labelledby="hg-dev-seats" class="flex flex-col gap-1">
      <h2 id="hg-dev-seats" class="font-semibold">{ht('hg.table.dev.seats')}</h2>
      <ul class="flex flex-col gap-1">
        {#each frame.seats as s (s.seat)}
          {@const h = humans.find((x) => x.userId === s.userId)}
          <li
            class="flex flex-wrap items-center gap-2"
            data-seat={s.seat}
            data-kind={s.kind}
            data-confirmed={frame.point?.confirmed.includes(s.seat) ? 'true' : 'false'}
          >
            <span>{ht('hg.table.dev.seat', { seat: s.seat })}</span>
            <span dir="auto">{seatLabel(s)}</span>
            {#each s.kind === 'human' ? tags(h) : [] as tag (tag)}
              <span class="text-xs text-fg-muted">{tag}</span>
            {/each}
            {#if session.canControl && ready}
              {#if s.kind === 'human'}
                <Button
                  variant="secondary"
                  onclick={() => session.switchSeat(s.seat, { bot: 'easy' })}
                  data-act={`seat-bot:${s.seat}`}
                >
                  {ht('hg.table.dev.toBot', { seat: s.seat })}
                </Button>
              {/if}
              {#each spectators as sp (sp.userId)}
                <Button
                  variant="secondary"
                  onclick={() => session.switchSeat(s.seat, { userId: sp.userId })}
                  data-act={`seat-user:${s.seat}:${sp.userId}`}
                >
                  {ht('hg.table.dev.seatUser', { name: nameOf(sp.userId), seat: s.seat })}
                </Button>
              {/each}
            {/if}
          </li>
        {/each}
      </ul>
    </section>

    {#if frame.status === 'finished'}
      <p data-finished>{ht('hg.table.dev.finished')}</p>
    {:else if frame.point}
      <section class="flex flex-col gap-2" data-point={frame.point.ref.id}>
        <h2 class="font-semibold">{ht('hg.table.dev.point', { id: frame.point.ref.id })}</h2>
        {#if frame.you.seat === undefined}
          <p data-watching>{ht('hg.table.dev.watching')}</p>
        {:else if deciding}
          {#if frame.pending}
            <p
              data-pending={frame.pending.key}
              data-pending-confirm={String(frame.pending.confirm)}
            >
              {ht('hg.table.dev.picked', { key: frame.pending.key })}
              {#if frame.pending.confirm}· {ht('hg.table.dev.confirmed')}{/if}
            </p>
          {/if}
          <div class="flex flex-wrap gap-2" role="group" aria-label={ht('hg.table.dev.decide')}>
            {#each frame.view.you?.choices ?? [] as c (c.key)}
              <Button
                variant="secondary"
                disabled={!ready || !session.canPick || !!frame.pending?.confirm}
                aria-pressed={frame.pending?.key === c.key}
                onclick={() => session.pick(c.key)}
                data-act={c.key}
              >
                <span dir="ltr">{c.kind} {c.design}</span>
              </Button>
            {/each}
          </div>
          {#if frame.pending && !frame.pending.confirm}
            <Button
              disabled={!ready || !session.canPick}
              onclick={() => session.confirm()}
              data-act="confirm"
            >
              {ht('hg.table.dev.confirm')}
            </Button>
          {:else if frame.pending?.confirm}
            <Button
              variant="secondary"
              disabled={!ready || !session.canPick}
              onclick={() => session.withdraw()}
              data-act="withdraw"
            >
              {ht('hg.table.dev.change')}
            </Button>
          {/if}
        {:else}
          <p data-waiting>{ht('hg.table.dev.waiting')}</p>
        {/if}
      </section>
    {/if}

    <div class="flex flex-wrap gap-2">
      {#if session.canUndo && ready}
        <Button variant="secondary" onclick={() => session.undo()} data-act="undo">
          {ht('hg.table.dev.undo')}
        </Button>
      {/if}
      {#if session.canResign && ready}
        <Button variant="secondary" onclick={() => session.resign()} data-act="resign">
          {ht('hg.table.dev.resign')}
        </Button>
      {/if}
    </div>
  {:else if lobby}
    <h1 class="font-display text-xl">{ht('hg.table.dev.lobby', { code: lobby.code })}</h1>
    <ul class="flex flex-col gap-1">
      {#each lobby.seats as s (s.seat)}
        <li class="flex flex-wrap items-center gap-2" data-seat={s.seat} data-kind={s.kind}>
          <span>{ht('hg.table.dev.seat', { seat: s.seat })}</span>
          <span dir="auto">{seatLabel(s)}</span>
          {#each s.kind === 'human' ? tags(humans.find((x) => x.userId === s.userId)) : [] as tag (tag)}
            <span class="text-xs text-fg-muted">{tag}</span>
          {/each}
        </li>
      {/each}
    </ul>
    <Button disabled={!session.canStart} onclick={() => session.start(true)} data-act="lobby-start">
      {ht('hg.table.dev.start')}
    </Button>
  {/if}
</main>
