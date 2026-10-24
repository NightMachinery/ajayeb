<script lang="ts">
  // The game menu (phone-layout.md section 3): the rules, the app's Sound
  // switch and the connection; for an admin, every seat with its
  // occupant's role and a Manage button that opens the seat's admin menu
  // over this one; the spectators, with a real admin's role buttons; "Stop
  // being admin" for a real admin; and "Resign to a bot" for a seated
  // player. Every action asks first, and each appears only when the frame
  // says this viewer may take it; the Session sends it and the server
  // decides.
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import { settings } from '$lib/ui/settings.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { TableState } from '$lib/wire/platform.gen';
  import { ht } from '../i18n/messages.svelte';
  import { openHaftganeRules } from '../rules/rules-overlay.svelte';
  import type { TableSession } from '../session/session.svelte';
  import AdminConfirm from './AdminConfirm.svelte';
  import RoleTag from './RoleTag.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import SpectatorList from './SpectatorList.svelte';
  import { isAdmin, occupant, roleMap, watchers } from './admin';
  import { seatName, where } from './seat-names';
  import { setTableSound } from './sound';

  interface Props {
    session: TableSession;
    frame: TableState;
    ready: boolean;
    /** Opens the newest end of Age's conflicts; absent before the first. */
    onmilitary?: () => void;
    onclose: () => void;
    /** Opens a seat's admin menu; without it the menu lists no seats. */
    onmanage?: (seat: number) => void;
  }

  let { session, frame, ready, onmilitary, onclose, onmanage }: Props = $props();

  const playing = $derived(frame.status === 'playing');
  const admin = $derived(playing && !!onmanage && isAdmin(frame));
  const realAdmin = $derived(
    playing && !!frame.humans.find((h) => h.userId === frame.you.userId)?.admin
  );
  const controls = $derived(ready && session.canControl);
  const people = $derived(watchers(frame));
  const roles = $derived(roleMap(frame));
  let asking = $state<'resign' | 'step-down' | null>(null);
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
    <Button
      variant="secondary"
      onclick={() => setTableSound(!settings.value.sound)}
      data-act="sound"
      data-sound={settings.value.sound ? 'on' : 'off'}
    >
      {t(settings.value.sound ? 'game.sound.turnOff' : 'game.sound.turnOn')}
    </Button>
    {#if onmilitary}
      <Button variant="secondary" onclick={onmilitary} data-act="menu-military">
        {ht('hg.end.open.military')}
      </Button>
    {/if}

    {#if admin}
      <section class="part" aria-labelledby="hg-menu-seats">
        <h3 id="hg-menu-seats">{ht('hg.admin.seats')}</h3>
        <ul class="seats">
          {#each frame.seats as s (s.seat)}
            {@const name = seatName(frame, s.seat)}
            {@const who = occupant(frame, s.seat)}
            {@const tip = ht('hg.admin.seat.open', { name })}
            {@const place = frame.view.seats.find((v) => v.seat === s.seat)}
            <li data-menu-seat={s.seat}>
              <SeatCrest
                seat={s.seat}
                tip={place
                  ? ht('hg.screen.crest.tip', { name, where: where(place, frame.seats.length) })
                  : name}
                bot={s.kind === 'bot'}
                botTip={ht('hg.screen.botBadge')}
              />
              <span class="name" dir="auto">{name}</span>
              {#if who?.role}<RoleTag role={who.role} name={who.name} compact />{/if}
              <button
                type="button"
                class="manage"
                data-act="seat-menu:{s.seat}"
                aria-label={tip}
                disabled={!controls}
                onclick={() => onmanage?.(s.seat)}
                {@attach tooltip({ text: tip, actionable: true })}
                >{ht('hg.admin.seats.manage')}</button
              >
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    <section class="part" aria-labelledby="hg-menu-spectators">
      <h3 id="hg-menu-spectators">{ht('hg.admin.spectators.title')}</h3>
      <SpectatorList
        {people}
        you={frame.you.userId}
        {roles}
        disabled={!controls}
        onrole={(a) => session.role(a.kind, a.userId)}
      />
    </section>

    {#if asking}
      <AdminConfirm
        text={ht(asking === 'resign' ? 'hg.admin.confirm.resign' : 'hg.admin.confirm.stepDown')}
        disabled={!ready || (asking === 'resign' ? !session.canResign : !controls)}
        onconfirm={() => {
          const sent = asking === 'resign' ? session.resign() : session.role('step-down');
          asking = null;
          if (sent) onclose();
        }}
        onback={() => (asking = null)}
      />
    {:else}
      {#if realAdmin}
        <Button
          variant="secondary"
          disabled={!controls}
          onclick={() => (asking = 'step-down')}
          data-act="step-down"
        >
          {ht('hg.admin.stepDown')}
        </Button>
      {/if}
      {#if session.canResign && ready}
        <Button variant="secondary" onclick={() => (asking = 'resign')} data-act="resign">
          {ht('hg.admin.resign')}
        </Button>
      {/if}
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
  .menu :global(button) {
    min-height: 44px;
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
  h3 {
    margin: 0;
    color: var(--fg-muted);
    font: 600 0.95rem var(--type-display);
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
  .part {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .seats {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .seats li {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    min-width: 0;
  }
  .name {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .manage {
    flex: 0 0 auto;
    padding-inline: 0.75rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
  }
  .manage:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
