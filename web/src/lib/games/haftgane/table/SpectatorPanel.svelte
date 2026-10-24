<script lang="ts">
  // What a spectator sees in place of a hand (phone-layout.md, "Known
  // gaps": the spectator view). A row of seat buttons centres the table on
  // the seat the spectator follows; the newest turn recap says what each
  // seat did, in ring order from that seat, with the seat and role changes
  // of the turn; and the spectators are listed with their roles. A former
  // player first reads who holds their seat now. Nothing private reaches a
  // spectator, so nothing private is shown.
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import SpectatorList from './SpectatorList.svelte';
  import TurnRecap from './TurnRecap.svelte';
  import type { SeatAction, SeatPerson } from './admin';
  import type { Recap } from './recap';

  interface Props {
    /** Every seat, in seat order. */
    seats: readonly Seat[];
    names: Readonly<Record<number, string>>;
    bots: readonly number[];
    /** The seat the table is centred on. */
    follow: number;
    onfollow: (seat: number) => void;
    people: readonly SeatPerson[];
    you: string;
    roles?: Readonly<Record<string, { promote: boolean; demote: boolean }>>;
    disabled?: boolean;
    onrole?: (action: Extract<SeatAction, { kind: 'promote' | 'demote' }>) => void;
    /** This viewer held a seat earlier: who plays it now. */
    former?: { seat: number; by: 'bot' | 'human'; name: string } | null;
    /** The newest turn, recapped from the followed seat; null before the first. */
    recap?: Recap | null;
    /** A user's name, for the recap's seat and role changes. */
    userName?: (userId: string | undefined) => string;
  }

  let {
    seats,
    names,
    bots,
    follow,
    onfollow,
    people,
    you,
    roles = {},
    disabled = false,
    onrole,
    former = null,
    recap = null,
    userName = () => ''
  }: Props = $props();
</script>

<section class="watch" data-spectator-panel aria-labelledby="hg-watch-title">
  <h2 id="hg-watch-title">{ht('hg.watch.title')}</h2>
  {#if former}
    <p class="former" data-former-seat={former.seat} dir="auto">
      {former.by === 'bot'
        ? ht('hg.watch.former.bot')
        : ht('hg.watch.former.human', { name: former.name })}
    </p>
  {/if}

  <div class="follow" role="group" aria-label={ht('hg.watch.follow')}>
    {#each seats as s (s.seat)}
      {@const name = names[s.seat] ?? ''}
      {@const tip = ht('hg.watch.follow.tip', { name })}
      <button
        type="button"
        class="seat"
        aria-pressed={s.seat === follow}
        aria-label={tip}
        data-act="follow:{s.seat}"
        onclick={() => onfollow(s.seat)}
        {@attach tooltip({ text: tip, actionable: true })}
      >
        <SeatCrest
          seat={s.seat}
          {tip}
          bot={bots.includes(s.seat)}
          botTip={ht('hg.screen.botBadge')}
        />
        <span class="name" dir="auto">{name}</span>
      </button>
    {/each}
  </div>

  {#if recap}
    <TurnRecap {recap} {names} {bots} you={null} {userName} level={3} />
  {:else}
    <h3>{ht('hg.watch.last')}</h3>
    <p class="note">{ht('hg.recap.none')}</p>
  {/if}

  <h3>{ht('hg.admin.spectators.title')}</h3>
  <SpectatorList {people} {you} {roles} {disabled} {onrole} />
</section>

<style>
  .watch {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    gap: 0.5rem;
    min-width: 0;
    min-height: 0;
    padding: 0.6rem 0.75rem;
    border: 1px solid var(--rule);
    border-radius: var(--shape-panel);
    background: var(--panel);
    overflow-y: auto;
  }
  h2 {
    margin: 0;
    font: 600 1.15rem var(--type-display);
  }
  h3 {
    margin: 0.2rem 0 0;
    color: var(--fg-muted);
    font: 600 0.9rem var(--type-display);
  }
  .former {
    margin: 0;
    padding: 0.45rem 0.6rem;
    border-radius: var(--shape-control);
    background: var(--panel-sunken);
    font-size: 0.9rem;
    line-height: 1.4;
  }
  /* Seven seats need more than a narrow phone's row: they wrap. */
  .follow {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    min-width: 0;
  }
  .seat {
    display: flex;
    flex: 1 1 0;
    flex-direction: column;
    align-items: center;
    gap: 0.1rem;
    min-width: 44px;
    max-width: 6.5rem;
    min-height: 44px;
    padding: 0.2rem 0.15rem;
    border: 1px solid transparent;
    border-radius: var(--shape-control);
    background: none;
    color: var(--fg-muted);
    font: inherit;
    cursor: pointer;
  }
  .seat[aria-pressed='true'] {
    border-color: var(--lapis);
    background: var(--panel-sunken);
    color: var(--fg);
  }
  .seat .name {
    max-width: 100%;
    overflow: hidden;
    font-size: 0.7rem;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .note {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
  }
</style>
