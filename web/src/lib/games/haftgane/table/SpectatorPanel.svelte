<script lang="ts">
  // What a spectator sees in place of a hand (phone-layout.md, "Known
  // gaps": the spectator view). A row of seat buttons centres the table on
  // the seat the spectator follows; the last turn lists what each seat did,
  // from the public view; and the spectators are listed with their roles.
  // A former player first reads who holds their seat now. Nothing private
  // reaches a spectator, so nothing private is shown.
  import type { Seat } from '$lib/wire/haftgane.gen';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import SpectatorList from './SpectatorList.svelte';
  import type { SeatAction, SeatPerson } from './admin';
  import { playText } from './labels';

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
    former = null
  }: Props = $props();

  const plays = $derived(seats.map((s) => ({ seat: s.seat, text: playText(s.last) })));
  const anyPlay = $derived(plays.some((p) => p.text));
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

  <h3>{ht('hg.watch.last')}</h3>
  {#if anyPlay}
    <ul class="plays" data-last-plays>
      {#each plays as p (p.seat)}
        <li data-last-play={p.seat}>
          <span class="who" dir="auto">{names[p.seat] ?? ''}</span>
          <span class="what">{p.text || ht('hg.watch.last.waiting')}</span>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="note">{ht('hg.watch.last.none')}</p>
  {/if}

  <h3>{ht('hg.admin.spectators.title')}</h3>
  <SpectatorList {people} {you} {roles} {disabled} {onrole} />
</section>

<style>
  .watch {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    min-width: 0;
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
  .follow {
    display: flex;
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
  .plays {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    margin: 0;
    padding: 0;
    list-style: none;
    font-size: 0.9rem;
  }
  .plays li {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .who {
    font-weight: 600;
  }
  .what {
    color: var(--fg-muted);
  }
  .note {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
  }
</style>
