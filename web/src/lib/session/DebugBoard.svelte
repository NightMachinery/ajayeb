<script lang="ts">
  // A deliberately plain view of one room, for development until the real
  // game screen exists (M5.S04 replaces it). It shows the state as data and
  // every legal choice as a button carrying data-act="<key>", which is
  // enough to play a whole game by clicking. It decides nothing itself:
  // the buttons are the view's choices, the prices are their cost
  // breakdowns, and whether input is open comes from the Session.
  import { formatNumber, t } from '$lib/i18n';
  import { connMessage } from './messages';
  import type { Presenter } from './presenter.svelte';
  import type { Session } from './session.svelte';
  import type { Choice } from '$lib/wire';

  interface Props {
    session: Session;
    presenter: Presenter;
    /** Called with a rematch room's code (the page navigates there). */
    onrematch?: (code: string) => void;
  }

  let { session, presenter, onrematch }: Props = $props();

  const frame = $derived(presenter.frame);
  const view = $derived(presenter.view);
  const inputOpen = $derived(presenter.idle && session.canAct);

  function rawName(seat: number): string | undefined {
    return frame?.room.seats.find((s) => s.seat === seat)?.name;
  }

  function choiceTarget(c: Choice): string {
    if (c.kind === 'wonder') return `${c.wonder} ← ${c.card}`;
    if (c.kind === 'starter') return t('debug.seat', { seat: c.starter ?? 0 });
    return c.card ?? c.wonder ?? c.token ?? '';
  }
</script>

{#snippet seatLabel(seat: number)}
  {@const name = rawName(seat)}
  {#if name}<span dir="auto">{name}</span>{:else}{t('debug.seat', { seat })}{/if}
{/snippet}

<main class="h-full overflow-y-auto" data-debug-board>
  <div class="mx-auto flex w-full max-w-3xl flex-col gap-4 px-4 py-4">
    <header class="flex flex-col gap-1">
      <h1 class="font-display text-2xl text-fg">
        {t('debug.title', { code: frame?.code ?? session.code ?? '' })}
      </h1>
      <p class="text-sm text-fg-muted">{t('debug.note')}</p>
    </header>

    {#if session.catalogStale}
      <p
        class="rounded-control border border-rule-strong bg-panel px-3 py-2 text-sm text-fg"
        role="alert"
      >
        {t('session.catalogStale')}
      </p>
    {/if}
    {#if session.notice}
      <p
        class="rounded-control border border-rule-strong bg-accent-soft px-3 py-2 text-sm text-accent-soft-fg"
        role="status"
        data-notice={session.notice.code}
      >
        {t(session.notice.message)}
      </p>
    {/if}

    <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
      <dt class="text-fg-muted">{t('debug.connection')}</dt>
      <dd data-conn={session.conn}>{t(connMessage(session.conn))}</dd>
      {#if frame}
        <dt class="text-fg-muted">{t('debug.mode')}</dt>
        <dd><code>{frame.room.mode}</code></dd>
        <dt class="text-fg-muted">{t('debug.status')}</dt>
        <dd><code>{frame.room.status}</code></dd>
        <dt class="text-fg-muted">{t('debug.version')}</dt>
        <dd>{formatNumber(frame.version)}</dd>
      {/if}
      {#if view}
        <dt class="text-fg-muted">{t('debug.phase')}</dt>
        <dd><code data-phase={view.phase}>{view.phase}</code></dd>
        <dt class="text-fg-muted">{t('debug.age')}</dt>
        <dd>{formatNumber(view.age)}</dd>
        <dt class="text-fg-muted">{t('debug.toMove')}</dt>
        <dd>
          {#if view.toMove === undefined}{t('debug.none')}{:else}{@render seatLabel(
              view.toMove
            )}{/if}
        </dd>
        <dt class="text-fg-muted">{t('debug.active')}</dt>
        <dd>{@render seatLabel(view.active)}</dd>
        <dt class="text-fg-muted">{t('debug.extraTurn')}</dt>
        <dd>{view.extraTurn ? t('debug.yes') : t('debug.no')}</dd>
        <dt class="text-fg-muted">{t('debug.military')}</dt>
        <dd>{formatNumber(view.military.position)}</dd>
      {/if}
    </dl>

    {#if !view}
      <p class="text-fg-muted">{t('debug.loading')}</p>
    {:else}
      {#if frame?.thinking !== undefined}
        <p class="text-sm text-fg-muted" role="status">
          {t('debug.thinking', { seat: frame?.thinking ?? 0 })}
        </p>
      {/if}

      <section class="flex flex-col gap-2" aria-labelledby="debug-choices">
        <h2 id="debug-choices" class="font-semibold text-fg">{t('debug.choices')}</h2>
        {#if session.slow}
          <p class="text-sm text-fg-muted" role="status">{t('session.pendingSlow')}</p>
        {/if}
        {#if view.choices.length === 0}
          <p class="text-sm text-fg-muted">{t('debug.noChoices')}</p>
        {:else}
          <ul class="flex flex-wrap gap-2">
            {#each view.choices as choice (choice.key)}
              <li>
                <button
                  type="button"
                  class="flex min-h-11 flex-col items-start rounded-control border border-rule-strong bg-panel px-3 py-1 text-start text-sm text-fg shadow-panel enabled:hover:border-accent disabled:opacity-60"
                  data-act={choice.key}
                  disabled={!inputOpen}
                  onclick={() => session.act(choice.key)}
                >
                  <span class="font-semibold">{choice.kind}: {choiceTarget(choice)}</span>
                  {#if choice.cost}
                    <span class="text-fg-muted"
                      >{t('debug.cost', { count: choice.cost.total })}</span
                    >
                  {:else if choice.gain !== undefined}
                    <span class="text-fg-muted">{t('debug.gain', { count: choice.gain })}</span>
                  {/if}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="min-h-11 rounded-control border border-rule px-3 text-sm text-fg disabled:opacity-60"
            data-control="undo"
            disabled={!presenter.idle || !session.canUndo}
            onclick={() => session.undo()}
          >
            {t('debug.undo')}
          </button>
          <button
            type="button"
            class="min-h-11 rounded-control border border-rule px-3 text-sm text-fg"
            data-control="resync"
            onclick={() => session.resync()}
          >
            {t('debug.resync')}
          </button>
        </div>
      </section>

      {#if view.pending}
        <section class="flex flex-col gap-1 text-sm" aria-labelledby="debug-pending">
          <h2 id="debug-pending" class="font-semibold text-fg">{t('debug.pending')}</h2>
          <p>
            <code>{view.pending.kind}</code>
            <span>({@render seatLabel(view.pending.seat)})</span>
            {#if view.pending.color}<code>{view.pending.color}</code>{/if}
          </p>
          {#if view.pending.cards.length > 0}<p>
              <code>{view.pending.cards.join(', ')}</code>
            </p>{/if}
          {#if view.pending.tokens.length > 0}<p>
              <code>{view.pending.tokens.join(', ')}</code>
            </p>{/if}
          {#if view.pending.hiddenTokens > 0}
            <p>{t('debug.hiddenTokens', { count: view.pending.hiddenTokens })}</p>
          {/if}
        </section>
      {/if}

      {#if view.result}
        <section
          class="flex flex-col gap-1 text-sm"
          aria-labelledby="debug-result"
          data-result={view.result.victory}
        >
          <h2 id="debug-result" class="font-semibold text-fg">{t('debug.result')}</h2>
          <p>
            <code>{view.result.victory}</code>
            {#if view.result.winner !== undefined}
              · {t('debug.winner')}: <span>{@render seatLabel(view.result.winner)}</span>
            {:else if view.result.shared}
              · {t('debug.shared')}
            {/if}
          </p>
          <p>
            {#each view.result.scores as score, seat (seat)}
              <span class="me-3"
                ><span>{@render seatLabel(seat)}</span>: {formatNumber(score.total)}</span
              >
            {/each}
          </p>
          {#if session.rematchCode}
            <button
              type="button"
              class="min-h-11 self-start rounded-control bg-accent px-3 text-sm font-semibold text-on-accent"
              data-control="to-rematch"
              onclick={() => session.rematchCode && onrematch?.(session.rematchCode)}
            >
              {t('debug.toRematch')}
            </button>
          {:else if frame?.room.status === 'finished' && (frame?.you.seats.length ?? 0) > 0}
            <button
              type="button"
              class="min-h-11 self-start rounded-control border border-rule-strong px-3 text-sm text-fg"
              data-control="rematch"
              onclick={() => session.rematch()}
            >
              {t('debug.rematch')}
            </button>
          {/if}
        </section>
      {/if}

      <section class="grid gap-3 split:grid-cols-2" aria-label={t('debug.city')}>
        {#each view.players as player (player.seat)}
          <div
            class="flex flex-col gap-1 rounded-panel border border-rule bg-panel p-3 text-sm"
            data-seat={player.seat}
          >
            <h2 class="font-semibold text-fg">
              <span>{@render seatLabel(player.seat)}</span>
              {#if frame?.you.seats.includes(player.seat)}<span class="text-fg-muted"
                  >({t('debug.you')})</span
                >{/if}
            </h2>
            <p>
              {t('debug.coins')}: <span data-coins>{formatNumber(player.coins)}</span> ·
              {t('debug.points')}: {formatNumber(view.scorePreview[player.seat]?.total ?? 0)}
            </p>
            <p>{t('debug.city')}: <code>{player.city.join(', ') || '-'}</code></p>
            <p>
              {t('debug.wonders')}:
              {#each player.wonders as w (w.wonder)}
                <code
                  class={[
                    'me-2',
                    w.built ? 'font-semibold' : w.removed ? 'line-through' : 'text-fg-muted'
                  ]}
                >
                  {w.wonder}{w.built ? ` (${t('debug.built')})` : ''}
                </code>
              {/each}
            </p>
            {#if player.tokens.length > 0}
              <p>{t('debug.tokens')}: <code>{player.tokens.join(', ')}</code></p>
            {/if}
          </div>
        {/each}
      </section>

      <section class="flex flex-col gap-1 text-sm">
        <p>{t('debug.boardTokens')}: <code>{view.boardTokens.join(', ') || '-'}</code></p>
        <p>{t('debug.discard')}: <code>{view.discard.join(', ') || '-'}</code></p>
      </section>

      {#if view.structure.length > 0}
        <section class="flex flex-col gap-1 text-sm" aria-labelledby="debug-structure">
          <h2 id="debug-structure" class="font-semibold text-fg">{t('debug.structure')}</h2>
          <ol class="grid grid-cols-2 gap-x-3 sm:grid-cols-4">
            {#each view.structure as slot (slot.index)}
              <li
                class={slot.state === 'empty' ? 'text-fg-muted' : 'text-fg'}
                data-slot={slot.index}
              >
                <code>{slot.index}·{slot.row}·{slot.col}</code>
                {#if slot.card}
                  <code data-card={slot.card} class={slot.accessible ? 'font-semibold' : ''}
                    >{slot.card}</code
                  >
                {:else if slot.back}
                  <code>[{slot.back}]</code>
                {/if}
                {#if slot.accessible}<span class="text-fg-muted">({t('debug.accessible')})</span
                  >{/if}
              </li>
            {/each}
          </ol>
        </section>
      {/if}
    {/if}
  </div>
</main>
