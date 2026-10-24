<script lang="ts">
  // The cards the end of the game turns face up, part of the final board:
  // the card under each wonder stage, the last card each seat discarded or
  // kept at the end of each Age, the discard pile and the guilds left in
  // the box. It shows the view's `reveal` section only, which the server
  // sends when the table chose to reveal the end; with the end kept hidden
  // there is nothing to open.
  import { ageLabel } from '$lib/i18n';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import type { Reveal, Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import CardChip from './CardChip.svelte';
  import SeatCrest from './SeatCrest.svelte';

  interface Props {
    reveal: Reveal;
    seats: readonly Seat[];
    names: Readonly<Record<number, string>>;
    onclose: () => void;
  }

  let { reveal, seats, names, onclose }: Props = $props();

  let open = $state(true);
  const tucked = $derived(
    seats
      .map((s) => ({ seat: s.seat, cards: reveal.tucked[s.seat] ?? [] }))
      .filter((t) => t.cards.length > 0)
  );
  const ages = $derived(
    [1, 2, 3]
      .map((age) => ({ age, cards: reveal.leftovers.filter((l) => l.age === age) }))
      .filter((a) => a.cards.length > 0)
  );
  // The pile in catalog order, as the server lists it, one chip per card.
  const pile = $derived.by(() => {
    const out: { slug: string; count: number }[] = [];
    for (const slug of reveal.pile) {
      const last = out.at(-1);
      if (last?.slug === slug) last.count++;
      else out.push({ slug, count: 1 });
    }
    return out;
  });
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-reveal-title"
  class="hg-reveal"
  panelData={{ 'data-final-reveal': '' }}
>
  {#snippet footer()}
    <Button variant="primary" full onclick={() => (open = false)} data-act="reveal-close">
      {ht('hg.end.reveal.close')}
    </Button>
  {/snippet}
  <div class="body">
    <h2 id="hg-reveal-title">{ht('hg.end.reveal.title')}</h2>

    <section aria-labelledby="hg-reveal-tucked" data-reveal-part="tucked">
      <h3 id="hg-reveal-tucked">{ht('hg.end.reveal.tucked')}</h3>
      {#if tucked.length === 0}
        <p class="none">{ht('hg.end.reveal.none')}</p>
      {/if}
      {#each tucked as t (t.seat)}
        <div class="seat" data-reveal-seat={t.seat}>
          <SeatCrest seat={t.seat} tip={names[t.seat] ?? ''} />
          <span class="name" dir="auto">{names[t.seat] ?? ''}</span>
          <span class="chips">
            {#each t.cards as slug, i (`${slug}:${i}`)}<CardChip {slug} />{/each}
          </span>
        </div>
      {/each}
    </section>

    {#each ages as a (a.age)}
      <section aria-labelledby={`hg-reveal-age-${a.age}`} data-reveal-part={`age-${a.age}`}>
        <h3 id={`hg-reveal-age-${a.age}`}>
          {ht('hg.end.reveal.leftovers', { age: ageLabel(a.age) })}
        </h3>
        {#each a.cards as l (`${l.seat}:${l.design}:${l.kept}`)}
          <div class="seat" data-reveal-seat={l.seat}>
            <SeatCrest seat={l.seat} tip={names[l.seat] ?? ''} />
            <span class="name" dir="auto">{names[l.seat] ?? ''}</span>
            <span class="chips">
              <CardChip slug={l.design} />
              {#if l.kept}<span class="kept">{ht('hg.end.reveal.kept')}</span>{/if}
            </span>
          </div>
        {/each}
      </section>
    {/each}

    <section aria-labelledby="hg-reveal-pile" data-reveal-part="pile">
      <h3 id="hg-reveal-pile">{ht('hg.end.reveal.pile', { count: reveal.pile.length })}</h3>
      {#if pile.length === 0}
        <p class="none">{ht('hg.end.reveal.none')}</p>
      {/if}
      <div class="chips wrap">
        {#each pile as p (p.slug)}<CardChip slug={p.slug} count={p.count} />{/each}
      </div>
    </section>

    <section aria-labelledby="hg-reveal-box" data-reveal-part="box">
      <h3 id="hg-reveal-box">{ht('hg.end.reveal.box')}</h3>
      {#if reveal.box.length === 0}
        <p class="none">{ht('hg.end.reveal.none')}</p>
      {/if}
      <div class="chips wrap">
        {#each reveal.box as slug (slug)}<CardChip {slug} />{/each}
      </div>
    </section>
  </div>
</Sheet>

<style>
  :global(dialog.sheet.hg-reveal) {
    width: min(100%, 40rem);
  }
  .body {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    padding: 0.4rem 1rem 0.6rem;
    color: var(--fg);
  }
  h2 {
    margin: 0;
    font: 600 1.4rem/1.2 var(--type-display);
  }
  h3 {
    margin: 0 0 0.35rem;
    color: var(--fg-muted);
    font: 600 0.95rem var(--type-display);
  }
  section {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .seat {
    display: grid;
    grid-template-columns: auto minmax(0, 7rem) minmax(0, 1fr);
    align-items: center;
    gap: 0.35rem;
  }
  .name {
    overflow: hidden;
    font-size: 0.875rem;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.3rem;
    min-width: 0;
  }
  .kept {
    color: var(--fg-muted);
    font-size: 0.75rem;
  }
  .none {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.875rem;
  }
</style>
