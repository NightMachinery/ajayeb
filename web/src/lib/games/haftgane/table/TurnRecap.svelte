<script lang="ts">
  // The turn recap (phone-layout.md section 4): one line per seat in ring
  // order from the viewer (or the seat a spectator follows), each with its
  // crest, a sentence of what it did, its reveal tag and its coin changes;
  // then the seat switches and role changes made during the turn, and where
  // hands went. The recap overlay and the spectator panel both show it. A
  // leaf: it words the recap it is given and decides nothing.
  import { ht } from '../i18n/messages.svelte';
  import CoinDelta from './CoinDelta.svelte';
  import RevealTag from './RevealTag.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import type { Recap } from './recap';
  import {
    actText,
    changeText,
    closingText,
    coinText,
    lastCardText,
    recapTitle
  } from './recap-text';

  interface Props {
    recap: Recap;
    names: Readonly<Record<number, string>>;
    bots: readonly number[];
    /** The viewer's seat, or null for a spectator. */
    you: number | null;
    /** A user's name, for the seat and role changes. */
    userName: (userId: string | undefined) => string;
    /** Whose hand the viewer holds now, after a pass. */
    held?: { name: string; count: number } | null;
    /** The heading's id, for a dialog's aria-labelledby. */
    titleId?: string;
    /** The heading level: 2 in the overlay, 3 inside the spectator panel. */
    level?: 2 | 3;
  }

  let { recap, names, bots, you, userName, held = null, titleId, level = 2 }: Props = $props();

  const nameOf = (seat: number) => names[seat] ?? ht('hg.screen.seat', { seat: seat + 1 });
  const closing = $derived(closingText(recap, held));
</script>

<section class="recap" data-turn-recap={recap.batch}>
  {#if level === 2}
    <h2 id={titleId} class="title">{recapTitle(recap)}</h2>
  {:else}
    <h3 id={titleId} class="title">{recapTitle(recap)}</h3>
  {/if}
  <ol class="lines">
    {#each recap.lines as line (line.seat)}
      {@const name = nameOf(line.seat)}
      {@const mine = line.seat === you}
      {@const what = actText(line, name, mine)}
      {@const coins = line.coins.map((c) => coinText(c, nameOf))}
      {@const card =
        line.act && line.act.kind !== 'discard-nothing' && line.act.kind !== 'side'
          ? line.act.design
          : ''}
      <li
        data-recap-line={line.seat}
        data-recap-act={line.act?.kind ?? 'none'}
        data-recap-card={card || undefined}
      >
        <SeatCrest
          seat={line.seat}
          tip={name}
          bot={bots.includes(line.seat)}
          botTip={ht('hg.screen.botBadge')}
        />
        <span class="text">
          <span class="what">{what}</span>
          {#each lastCardText(line, name, mine) as extra, i (i)}
            <span class="extra" data-recap-last>{extra}</span>
          {/each}
        </span>
        <span class="side">
          <RevealTag {line} tip={what} lines={coins} />
          {#if line.coins.length > 0}
            <span class="coins">
              {#each line.coins as c, i (i)}
                <CoinDelta change={c} tip={coins[i] ?? ''} />
              {/each}
            </span>
          {/if}
        </span>
      </li>
    {/each}
  </ol>
  {#if recap.changes.length > 0}
    <ul class="changes">
      {#each recap.changes as e, i (i)}
        <li data-recap-change={e.kind}>{changeText(e, userName)}</li>
      {/each}
    </ul>
  {/if}
  {#if closing.length > 0}
    <p class="closing" data-recap-closing={recap.pass ?? ''}>{closing.join(' ')}</p>
  {/if}
</section>

<style>
  .recap {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
  }
  .title {
    margin: 0;
    font: 600 1.05rem var(--type-display);
  }
  h3.title {
    color: var(--fg-muted);
    font-size: 0.9rem;
  }
  .lines,
  .changes {
    display: flex;
    flex-direction: column;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .lines li {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.45rem;
    padding-block: 0.35rem;
    border-block-start: 1px solid var(--rule);
  }
  .text {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    min-width: 0;
    font-size: 0.875rem;
    line-height: 1.3;
  }
  .extra {
    color: var(--fg-muted);
    font-size: 0.8rem;
  }
  .side {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.2rem;
    max-width: 8.5rem;
    min-width: 0;
  }
  .coins {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.2rem;
  }
  .changes li {
    padding-block: 0.3rem;
    border-block-start: 1px solid var(--rule);
    font-size: 0.85rem;
  }
  .closing {
    margin: 0;
    padding-block-start: 0.35rem;
    border-block-start: 1px solid var(--rule);
    color: var(--fg-muted);
    font-size: 0.85rem;
  }
</style>
