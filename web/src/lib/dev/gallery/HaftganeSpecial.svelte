<script lang="ts">
  // Captured ViewFor samples from legal, seeded games. This gallery never
  // constructs payments or infers private cards from the public city.
  import { onMount } from 'svelte';
  import { i18n } from '$lib/i18n';
  import '$lib/games/haftgane/theme.css';
  import type { TableView } from '$lib/wire/platform.gen';
  import SidePicker from '$lib/games/haftgane/table/SidePicker.svelte';
  import DiscardPicker from '$lib/games/haftgane/table/DiscardPicker.svelte';
  import KeptCard from '$lib/games/haftgane/table/KeptCard.svelte';
  import CardSheet from '$lib/games/haftgane/table/CardSheet.svelte';
  import HandGrid from '$lib/games/haftgane/table/HandGrid.svelte';
  import YourStrip from '$lib/games/haftgane/table/YourStrip.svelte';
  import TableTopBar from '$lib/games/haftgane/table/TableTopBar.svelte';
  import { ht } from '$lib/games/haftgane/i18n/messages.svelte';
  import { afford, cardChoices } from '$lib/games/haftgane/table/model';
  import cases from './haftgane-special-cases.json';
  interface Sample {
    kind: string;
    seats: number;
    view: TableView & { required: number[] };
    waiting: TableView & { required: number[] };
  }
  const samples: readonly Sample[] = cases;
  let sample = $state.raw<Sample | null>(null);
  let waiting = $state(false);
  let draft = $state<string | null>(null);
  let closed = $state(false);
  let confirmed = $state(false);
  let selected = $state<string | null>(null);
  onMount(() => {
    const p = new URLSearchParams(location.search);
    sample =
      samples.find(
        (s) => s.kind === p.get('hgSpecial') && s.seats === Number(p.get('seats') ?? 5)
      ) ?? null;
    waiting = p.get('waiting') === '1';
  });
  const view = $derived(sample ? (waiting ? sample.waiting : sample.view) : null);
  const mine = $derived(view?.you);
  const seat = $derived(view?.seats.find((s) => s.relation === 'self'));
  const free = $derived(mine?.choices.find((c) => c.free === sample?.kind));
  const card = $derived(selected ?? free?.design ?? null);
  const names = $derived(
    Object.fromEntries(
      view?.seats.map((s) => [s.seat, ht('hg.screen.seat', { seat: s.seat + 1 })]) ?? []
    )
  );
  function commit(key: string): void {
    draft = key;
    confirmed = true;
    closed = true;
  }
</script>

{#if view && mine && seat && sample}
  <main class="special" dir={i18n.dir} data-gallery-special={sample.kind} data-seats={sample.seats}>
    <TableTopBar
      phase={view.phase}
      age={view.age}
      turn={view.turn}
      turns={6}
      pass={view.pass}
      passTo={ht('hg.screen.seat', { seat: 2 })}
      finished={false}
      onmenu={() => {}}
    />
    <YourStrip
      {seat}
      name={ht('hg.screen.you')}
      where={ht('hg.screen.where.self')}
      own
      bot={false}
      status={waiting ? 'confirmed' : 'choosing'}
      stageOpen={false}
    />
    {#if waiting}
      <p role="status" data-special-wait>
        {view.phase === 'turn'
          ? ht('hg.screen.bar.confirmed')
          : sample.kind === 'side'
            ? ht('hg.special.wait.sides')
            : ht(
                view.phase === 'discard-build' ? 'hg.special.wait.discard' : 'hg.special.wait.last',
                { name: ht('hg.screen.seat', { seat: view.required[0]! + 1 }) }
              )}
      </p>
    {:else if sample.kind === 'sixth'}
      <p data-sixth-turn>{ht('hg.special.sixth')}</p>
      <div class="hand">
        <HandGrid
          hand={mine.hand}
          choices={mine.choices}
          disabled={mine.disabled}
          selected={card}
          picked={null}
          onselect={(c) => (selected = c)}
        />
      </div>
    {:else if mine.kept}
      <KeptCard
        card={mine.kept}
        locked={sample.kind === 'kept'}
        afford={afford(cardChoices(mine.kept, mine.choices, mine.disabled))}
        onselect={(c) => (selected = c)}
      />
    {/if}
    {#if confirmed}<p data-gallery-confirmed>{ht('hg.sheet.confirmed')}</p>{/if}
  </main>
  {#if !closed && !waiting}
    {#if sample.kind === 'side'}
      <SidePicker
        boardId={seat.board}
        choices={mine.choices}
        pendingKey={draft}
        locked={false}
        ondraft={(key) => (draft = key)}
        onconfirm={commit}
        onclose={() => (closed = true)}
      />
    {:else if sample.kind === 'discard'}
      <DiscardPicker
        offer={mine.offer}
        choices={mine.choices}
        disabled={mine.disabled}
        pendingKey={draft}
        locked={false}
        ondraft={(key) => (draft = key)}
        onconfirm={commit}
        onclose={() => (closed = true)}
      />
    {:else if card}
      <CardSheet
        {card}
        choices={mine.choices}
        disabled={mine.disabled}
        {seat}
        neighbours={mine.neighbours}
        {names}
        pendingKey={draft}
        locked={false}
        confirmMoves={false}
        ondraft={(key) => (draft = key)}
        onconfirm={commit}
        onclose={() => {
          closed = true;
          selected = null;
        }}
      />
    {/if}
  {/if}
{/if}

<style>
  .special {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    height: 100dvh;
    padding: 0.5rem;
    overflow: hidden;
    background: var(--table);
    color: var(--fg);
  }
  .hand {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
  }
  p {
    margin: 0;
    line-height: 1.4;
  }
</style>
