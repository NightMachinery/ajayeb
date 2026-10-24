<script lang="ts">
  // The people watching a table (phone-layout.md section 3, "Roles"): each
  // name with its role tag, "away" when not connected, and for an admin the
  // previous-seat hint beside a former player (the frame names it only to
  // admins). A real admin gets "Make admin" or "Remove admin" where the
  // roles allow, behind a confirmation. What each row offers comes from the
  // caller, read from the frame; the server decides.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import { ht } from '../i18n/messages.svelte';
  import AdminConfirm from './AdminConfirm.svelte';
  import RoleTag from './RoleTag.svelte';
  import type { SeatAction, SeatPerson } from './admin';

  interface Props {
    people: readonly SeatPerson[];
    /** The viewer's user id, tagged "you". */
    you: string;
    /** The role buttons each person gets, by user id. */
    roles?: Readonly<Record<string, { promote: boolean; demote: boolean }>>;
    disabled?: boolean;
    onrole?: (action: Extract<SeatAction, { kind: 'promote' | 'demote' }>) => void;
  }

  let { people, you, roles = {}, disabled = false, onrole }: Props = $props();

  let asking = $state<{ kind: 'promote' | 'demote'; userId: string; text: string } | null>(null);

  function ask(kind: 'promote' | 'demote', p: SeatPerson): void {
    asking = {
      kind,
      userId: p.userId,
      text: ht(kind === 'promote' ? 'hg.admin.confirm.promote' : 'hg.admin.confirm.demote', {
        name: p.name
      })
    };
  }
</script>

{#if people.length === 0}
  <p class="none" data-spectators-none>{ht('hg.admin.spectators.none')}</p>
{:else}
  <ul class="list" data-spectator-list>
    {#each people as p (p.userId)}
      {@const r = roles[p.userId]}
      <li data-spectator={p.userId}>
        <div class="row">
          <span class="name" dir="auto">{p.name}</span>
          {#if p.userId === you}<span class="you">{ht('hg.admin.you')}</span>{/if}
          {#if p.role}<RoleTag role={p.role} name={p.name} />{/if}
          {#if !p.connected}<span
              class="muted"
              {@attach tooltip({ text: ht('hg.admin.away.tip', { name: p.name }) })}
              >{ht('hg.admin.away')}</span
            >{/if}
          {#if p.previous !== undefined && p.previous !== null}
            <span
              class="muted"
              data-previous-seat={p.previous}
              {@attach tooltip({ text: ht('hg.admin.seat.hint.tip') })}
              >{ht('hg.admin.seat.hint', { seat: p.previous + 1 })}</span
            >
          {/if}
        </div>
        {#if asking?.userId === p.userId}
          <AdminConfirm
            text={asking.text}
            {disabled}
            onconfirm={() => {
              if (asking) onrole?.({ kind: asking.kind, userId: asking.userId });
              asking = null;
            }}
            onback={() => (asking = null)}
          />
        {:else if r && (r.promote || r.demote)}
          <div class="actions">
            {#if r.promote}
              <Button
                variant="secondary"
                {disabled}
                data-act="role-promote:{p.userId}"
                onclick={() => ask('promote', p)}>{ht('hg.admin.promote')}</Button
              >
            {/if}
            {#if r.demote}
              <Button
                variant="secondary"
                {disabled}
                data-act="role-demote:{p.userId}"
                onclick={() => ask('demote', p)}>{ht('hg.admin.demote')}</Button
              >
            {/if}
          </div>
        {/if}
      </li>
    {/each}
  </ul>
{/if}

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    min-width: 0;
  }
  .name {
    min-width: 0;
    font-weight: 600;
    overflow-wrap: anywhere;
  }
  .you,
  .muted {
    color: var(--fg-muted);
    font-size: 0.85rem;
  }
  .muted {
    font-style: italic;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .actions :global(button) {
    min-height: 44px;
  }
  .none {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
  }
</style>
