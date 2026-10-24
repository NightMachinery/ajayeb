<script lang="ts">
  // The game menu: the rules, the connection, and until the admin slice
  // (M18.admin-controls-resign-and-spectator) draws its seat menu, the
  // development screen's controls: an admin gives a seat to a bot or seats
  // a spectator, and a player undoes or resigns. Every button sends what
  // the Session offers; none decides what is allowed.
  import { formatNumber, t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { TableState } from '$lib/wire/platform.gen';
  import { ht } from '../i18n/messages.svelte';
  import { openHaftganeRules } from '../rules/rules-overlay.svelte';
  import type { TableSession } from '../session/session.svelte';
  import { seatName } from './seat-names';

  interface Props {
    session: TableSession;
    frame: TableState;
    ready: boolean;
    onclose: () => void;
  }

  let { session, frame, ready, onclose }: Props = $props();

  const spectators = $derived(
    frame.humans.filter(
      (h) => h.connected && !frame.seats.some((s) => s.kind === 'human' && s.userId === h.userId)
    )
  );
  const nameOf = (userId: string) => {
    const h = frame.humans.find((x) => x.userId === userId);
    return h ? (h.nameNo > 1 ? `${h.name} ${formatNumber(h.nameNo)}` : h.name) : '';
  };
</script>

<div class="scrim" data-table-menu>
  <button
    type="button"
    class="backdrop"
    aria-label={ht('hg.screen.menu.close')}
    tabindex="-1"
    onclick={onclose}
  ></button>
  <div class="menu" role="dialog" aria-modal="true" aria-labelledby="hg-menu-title">
    <div class="head">
      <h2 id="hg-menu-title">{ht('hg.screen.menu.title', { code: frame.code })}</h2>
      <button
        type="button"
        class="close"
        aria-label={ht('hg.screen.menu.close')}
        data-act="menu-close"
        onclick={onclose}
        {@attach tooltip({ text: ht('hg.screen.menu.close') })}
      >
        <span aria-hidden="true">×</span>
      </button>
    </div>
    <p class="conn" data-conn={session.conn}>{t(`session.conn.${session.conn}`)}</p>
    <Button
      variant="secondary"
      onclick={() => {
        onclose();
        void openHaftganeRules();
      }}
      data-act="rules"
    >
      {ht('hg.screen.menu.rules')}
    </Button>
    {#if session.canControl && ready}
      <ul class="seats">
        {#each frame.seats as s (s.seat)}
          <li data-menu-seat={s.seat}>
            <span dir="auto">{seatName(frame, s.seat)}</span>
            {#if s.kind === 'human'}
              <Button
                variant="secondary"
                onclick={() => session.switchSeat(s.seat, { bot: 'easy' })}
                data-act={`seat-bot:${s.seat}`}
              >
                {ht('hg.screen.menu.toBot', { seat: s.seat + 1 })}
              </Button>
            {/if}
            {#each spectators as sp (sp.userId)}
              <Button
                variant="secondary"
                onclick={() => session.switchSeat(s.seat, { userId: sp.userId })}
                data-act={`seat-user:${s.seat}:${sp.userId}`}
              >
                {ht('hg.screen.menu.seatUser', { name: nameOf(sp.userId), seat: s.seat + 1 })}
              </Button>
            {/each}
          </li>
        {/each}
      </ul>
    {/if}
    {#if session.canUndo && ready}
      <Button variant="secondary" onclick={() => session.undo()} data-act="undo">
        {ht('hg.screen.menu.undo')}
      </Button>
    {/if}
    {#if session.canResign && ready}
      <Button variant="secondary" onclick={() => session.resign()} data-act="resign">
        {ht('hg.screen.menu.resign')}
      </Button>
    {/if}
  </div>
</div>

<style>
  .scrim {
    position: fixed;
    inset: 0;
    z-index: 40;
    display: flex;
    justify-content: flex-start;
  }
  .backdrop {
    position: absolute;
    inset: 0;
    border: 0;
    background: color-mix(in oklch, var(--ink) 35%, transparent);
  }
  .menu {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    width: min(22rem, 88vw);
    height: 100%;
    overflow-y: auto;
    padding: 0.75rem;
    padding-block-start: calc(0.75rem + var(--safe-t));
    background: var(--panel);
    box-shadow: var(--shadow);
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  h2 {
    margin: 0;
    font-family: var(--type-display);
    font-size: 1.25rem;
  }
  .close {
    display: grid;
    place-items: center;
    width: 2.75rem;
    height: 2.75rem;
    border: 0;
    border-radius: var(--shape-control);
    background: transparent;
    color: var(--fg);
    font-size: 1.4rem;
    cursor: pointer;
  }
  .conn {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.875rem;
  }
  .seats {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .seats li {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }
</style>
