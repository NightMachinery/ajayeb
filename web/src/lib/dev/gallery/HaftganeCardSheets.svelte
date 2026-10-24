<script lang="ts">
  // Real, seedless ViewFor captures from legal seeded games, not invented
  // payments. See docs/haftgane/card-sheet.md for the capture provenance.
  import { onMount } from 'svelte';
  import '$lib/games/haftgane/theme.css';
  import type { Choice, Disabled, Neighbour, Seat } from '$lib/wire/haftgane.gen';
  import CardSheet from '$lib/games/haftgane/table/CardSheet.svelte';
  import PaymentList from '$lib/games/haftgane/table/PaymentList.svelte';
  import { ht } from '$lib/games/haftgane/i18n/messages.svelte';
  import { cardName } from '$lib/games/haftgane/table/labels';
  import cases from './haftgane-card-cases.json';

  interface Sample {
    kind: string;
    seats: number;
    card: string;
    seat: Seat;
    choices: Choice[];
    disabled: Disabled[];
    neighbours: Neighbour[];
  }
  const samples: readonly Sample[] = cases;
  const names = Object.fromEntries(
    Array.from({ length: 7 }, (_, i) => [i, ht('hg.screen.seat', { seat: i + 1 })])
  );
  let sample = $state.raw<Sample | null>(null);
  let draft = $state<string | null>(null);
  let confirm = $state(true);
  let confirmed = $state(false);
  onMount(() => {
    const params = new URLSearchParams(location.search);
    const kind = params.get('hgSheet');
    const seats = Number(params.get('seats') ?? 5);
    sample =
      samples.find(
        (x) =>
          x.seats === seats && x.kind === (kind === 'stage' || kind === 'sale' ? 'trade' : kind)
      ) ?? null;
    if (kind === 'stage') draft = sample?.choices.find((c) => c.kind === 'stage')?.key ?? null;
    if (kind === 'sale') draft = sample?.choices.find((c) => c.kind === 'sell')?.key ?? null;
    confirm = params.get('confirm') !== 'off';
  });
</script>

<section class="flex flex-col gap-3" data-gallery-card-sheets>
  <h3 class="font-display text-xl">Card sheets and payments <!-- i18n-ok --></h3>
  <div class="flex flex-wrap gap-2">
    {#each samples as s (`${s.seats}:${s.kind}`)}
      <button
        type="button"
        class="min-h-11 min-w-11 rounded-control border border-rule bg-panel px-3 text-fg"
        data-gallery-sheet={`${s.seats}:${s.kind}`}
        onclick={() => {
          sample = s;
          draft = null;
          confirmed = false;
        }}
      >
        {s.seats} · {s.kind} · {cardName(s.card)}
      </button>
    {/each}
  </div>
  <div class="max-w-xl">
    <PaymentList
      choices={samples.find((x) => x.kind === 'trade')!.choices.filter((c) => c.kind === 'build')}
      selected={draft}
      label={ht('hg.sheet.build')}
      neighbours={samples.find((x) => x.kind === 'trade')!.neighbours}
      {names}
      onselect={(key) => (draft = key)}
    />
  </div>
  {#if confirmed}<p role="status" data-gallery-confirmed>{ht('hg.sheet.confirmed')}</p>{/if}
</section>
{#if sample}
  {#key `${sample.seats}:${sample.kind}`}
    <CardSheet
      card={sample.card}
      choices={sample.choices}
      disabled={sample.disabled}
      seat={sample.seat}
      neighbours={sample.neighbours}
      {names}
      pendingKey={draft}
      locked={false}
      confirmMoves={confirm}
      ondraft={(key) => (draft = key)}
      onconfirm={(key) => {
        draft = key;
        confirmed = true;
        sample = null;
      }}
      onclose={() => (sample = null)}
    />
  {/key}
{/if}
