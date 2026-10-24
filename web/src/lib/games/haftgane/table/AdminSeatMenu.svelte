<script lang="ts">
  // An admin's menu on one seat in a game (phone-layout.md section 3,
  // "Admin controls in a game"). A human's seat offers "Give the seat to a
  // bot", at any of the table's bot levels; every seat offers "Seat a
  // spectator here", listing the connected spectators with the
  // previous-seat hint beside a former player (the frame names it only to
  // admins). A real admin also gets the occupant's role buttons. Every
  // action first shows what will change and waits for Confirm. The menu
  // offers what the frame says this viewer may do; the server decides.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import { ht } from '../i18n/messages.svelte';
  import AdminConfirm from './AdminConfirm.svelte';
  import RoleTag from './RoleTag.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import type { SeatAction, SeatPerson } from './admin';

  interface Props {
    seat: number;
    /** The seat's display name. */
    name: string;
    crestTip: string;
    /** The human in the seat, or null for a bot. */
    occupant: SeatPerson | null;
    /** The bot levels to offer a human's seat, with their persona names. */
    levels: readonly { level: string; persona: string }[];
    /** Connected spectators who may take the seat. */
    spectators: readonly SeatPerson[];
    /** The role buttons for the occupant. */
    roles: { promote: boolean; demote: boolean };
    /** The viewer is the temporary admin: seat powers only. */
    temporary?: boolean;
    /** No request can go out now. */
    disabled?: boolean;
    onaction: (action: SeatAction) => void;
    onclose: () => void;
  }

  let {
    seat,
    name,
    crestTip,
    occupant,
    levels,
    spectators,
    roles,
    temporary = false,
    disabled = false,
    onaction,
    onclose
  }: Props = $props();

  let open = $state(true);
  let asking = $state<{ action: SeatAction; text: string } | null>(null);
  const number = $derived(seat + 1);
  const titleId = $derived(`hg-seat-menu-${seat}`);

  function ask(action: SeatAction): void {
    asking = { action, text: describe(action) };
  }

  function describe(action: SeatAction): string {
    switch (action.kind) {
      case 'bot': {
        const bot = levels.find((l) => l.level === action.level)?.persona ?? '';
        return occupant
          ? ht('hg.admin.confirm.toBot', { bot, seat: number, name: occupant.name })
          : ht('hg.admin.confirm.botToBot', { bot, seat: number });
      }
      case 'human': {
        const who = spectators.find((s) => s.userId === action.userId)?.name ?? '';
        return occupant
          ? ht('hg.admin.confirm.seatReplace', { name: who, seat: number, old: occupant.name })
          : ht('hg.admin.confirm.seat', { name: who, seat: number });
      }
      case 'promote':
        return ht('hg.admin.confirm.promote', { name: occupant?.name ?? '' });
      case 'demote':
        return ht('hg.admin.confirm.demote', { name: occupant?.name ?? '' });
    }
  }

  function confirm(): void {
    if (!asking) return;
    onaction(asking.action);
    asking = null;
    onclose();
  }

  const hint = (p: number) => ht('hg.admin.seat.hint', { seat: p + 1 });
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  {titleId}
  class="hg-seat-menu"
  panelData={{ 'data-seat-menu': String(seat) }}
>
  <div class="content">
    <header class="head">
      <SeatCrest {seat} tip={crestTip} bot={!occupant} botTip={ht('hg.screen.botBadge')} />
      <div class="who">
        <h2 id={titleId} dir="auto">{ht('hg.admin.seat.title', { seat: number, name })}</h2>
        <p class="sub">
          {#if occupant}
            <span dir="auto">{ht('hg.admin.seat.player', { name: occupant.name })}</span>
            {#if occupant.role}<RoleTag role={occupant.role} name={occupant.name} />{/if}
            {#if !occupant.connected}<span
                class="away"
                {@attach tooltip({ text: ht('hg.admin.away.tip', { name: occupant.name }) })}
                >{ht('hg.admin.away')}</span
              >{/if}
          {:else}
            {ht('hg.admin.seat.bot')}
          {/if}
        </p>
      </div>
      <button
        type="button"
        class="close"
        data-act="seat-menu-close"
        aria-label={ht('hg.admin.seat.close')}
        onclick={onclose}
        {@attach tooltip({ text: ht('hg.admin.seat.close'), actionable: true })}>×</button
      >
    </header>

    {#if asking}
      <AdminConfirm
        text={asking.text}
        {disabled}
        onconfirm={confirm}
        onback={() => (asking = null)}
      />
    {:else}
      {#if occupant}
        <section class="part" aria-labelledby="{titleId}-bot">
          <h3 id="{titleId}-bot">{ht('hg.admin.seat.toBot')}</h3>
          <div class="row">
            {#each levels as l (l.level)}
              <Button
                variant="secondary"
                {disabled}
                data-act="seat-bot:{seat}:{l.level}"
                onclick={() => ask({ kind: 'bot', level: l.level })}>{l.persona}</Button
              >
            {/each}
          </div>
        </section>
      {/if}

      <section class="part" aria-labelledby="{titleId}-seat">
        <h3 id="{titleId}-seat">{ht('hg.admin.seat.seatHere')}</h3>
        {#if spectators.length === 0}
          <p class="note">{ht('hg.admin.seat.noSpectators')}</p>
        {:else}
          <ul class="people">
            {#each spectators as sp (sp.userId)}
              <li data-spectator={sp.userId}>
                <Button
                  variant="secondary"
                  {disabled}
                  class="person"
                  data-act="seat-user:{seat}:{sp.userId}"
                  onclick={() => ask({ kind: 'human', userId: sp.userId })}
                >
                  <span class="name" dir="auto">{sp.name}</span>
                </Button>
                {#if sp.role}<RoleTag role={sp.role} name={sp.name} />{/if}
                {#if sp.previous !== undefined && sp.previous !== null}
                  <span
                    class="hint"
                    data-previous-seat={sp.previous}
                    {@attach tooltip({ text: ht('hg.admin.seat.hint.tip') })}
                    >{hint(sp.previous)}</span
                  >
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      {#if occupant && (roles.promote || roles.demote)}
        <section class="part" aria-labelledby="{titleId}-roles">
          <h3 id="{titleId}-roles">{ht('hg.admin.seat.roles')}</h3>
          <div class="row">
            {#if roles.promote}
              <Button
                variant="secondary"
                {disabled}
                data-act="role-promote:{occupant.userId}"
                onclick={() => ask({ kind: 'promote', userId: occupant.userId })}
                >{ht('hg.admin.promote')}</Button
              >
            {/if}
            {#if roles.demote}
              <Button
                variant="secondary"
                {disabled}
                data-act="role-demote:{occupant.userId}"
                onclick={() => ask({ kind: 'demote', userId: occupant.userId })}
                >{ht('hg.admin.demote')}</Button
              >
            {/if}
          </div>
        </section>
      {:else if occupant && temporary}
        <p class="note">{ht('hg.admin.seat.temporary')}</p>
      {/if}
    {/if}
  </div>
</Sheet>

<style>
  :global(dialog.sheet.hg-seat-menu) {
    width: min(100%, 32rem);
  }
  .content {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    padding: 0.4rem 0.75rem 1.2rem;
    color: var(--fg);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }
  .who {
    flex: 1;
    min-width: 0;
  }
  h2 {
    margin: 0;
    font: 600 1.3rem/1.2 var(--type-display);
    overflow-wrap: anywhere;
  }
  .sub {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    margin: 0.15rem 0 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
  }
  .close {
    display: grid;
    flex: 0 0 44px;
    place-items: center;
    width: 44px;
    height: 44px;
    border: 0;
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 1.5rem;
    cursor: pointer;
  }
  .part {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  h3 {
    margin: 0;
    color: var(--fg-muted);
    font: 600 0.95rem var(--type-display);
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .row :global(button),
  .people :global(button) {
    min-height: 44px;
  }
  .people {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .people li {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }
  .people :global(.person) {
    max-width: 100%;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .hint,
  .away {
    color: var(--fg-muted);
    font-size: 0.85rem;
    font-style: italic;
  }
  .note {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
    line-height: 1.4;
  }
</style>
