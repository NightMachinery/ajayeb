<script lang="ts">
  // One radio per server outcome. Resource groups are projections of the
  // witness, never a new quote. The server's producer index links a bought
  // unit to its discount explanation in the neighbour's trade view.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Choice, Neighbour } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import { choiceAction } from './labels';
  import ResourceTile from './ResourceTile.svelte';
  import SeatCrest from './SeatCrest.svelte';

  interface Props {
    choices: readonly Choice[];
    selected: string | null;
    label: string;
    neighbours: readonly Neighbour[];
    names: Readonly<Record<number, string>>;
    locked?: boolean;
    onselect: (key: string) => void;
  }
  let { choices, selected, label, neighbours, names, locked = false, onselect }: Props = $props();

  function unitTip(c: Choice, i: number): string {
    const u = c.witness[i]!;
    const text = ht('hg.sheet.unit', {
      resource: ht(`hg.resource.${u.resource}` as 'hg.resource.wood'),
      price: u.price
    });
    const d = neighbours
      .find((n) => u.source.startsWith(n.direction))
      ?.units.find((p) => p.producer === u.producer)
      ?.options.find((o) => o.resource === u.resource && o.price === u.price)?.discount;
    return d
      ? text +
          ' ' +
          (d.design
            ? ht('hg.screen.trade.discountCard', {
                card: ht(`hg.card.${d.design}` as 'hg.card.marketplace')
              })
            : ht('hg.screen.trade.discountStage', { stage: d.stage }))
      : text;
  }

  function move(event: KeyboardEvent, index: number): void {
    if (!['ArrowDown', 'ArrowUp', 'ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key))
      return;
    event.preventDefault();
    const next =
      event.key === 'Home'
        ? 0
        : event.key === 'End'
          ? choices.length - 1
          : (index +
              (event.key === 'ArrowDown' || event.key === 'ArrowRight' ? 1 : -1) +
              choices.length) %
            choices.length;
    const button = (
      event.currentTarget as HTMLElement
    ).parentElement?.querySelectorAll<HTMLButtonElement>('[role="radio"]')[next];
    button?.focus();
    if (!locked && choices[next]) onselect(choices[next].key);
  }
</script>

<div class="payments" role="radiogroup" aria-label={label}>
  {#each choices as c, index (c.key)}
    <button
      type="button"
      class="payment"
      class:chosen={selected === c.key}
      role="radio"
      aria-checked={selected === c.key}
      disabled={locked}
      aria-label={ht('hg.sheet.payment', { number: index + 1, action: choiceAction(c) })}
      data-pick={c.key}
      data-kind={c.kind}
      onclick={() => onselect(c.key)}
      onkeydown={(event) => move(event, index)}
    >
      <span class="radio" aria-hidden="true"></span>
      <span class="details">
        {#if c.free}
          <span>{ht(`hg.screen.afford.free.${c.free}` as 'hg.screen.afford.free.chain')}</span>
        {/if}
        {#if c.kind === 'sell'}<span>{choiceAction(c)}</span>{/if}
        {#each ['own', 'left', 'right'] as source (source)}
          {@const units = c.witness
            .map((u, i) => ({ u, i }))
            .filter(({ u }) => u.source.startsWith(source))}
          {#if units.length > 0}
            {@const nb = neighbours.find((n) => n.direction === source)}
            <span class="source" data-payment-source={source}>
              {#if nb}
                <SeatCrest seat={nb.seat} tip={names[nb.seat] ?? ''} />
                <span class="name" dir="auto">{names[nb.seat]}</span>
              {:else}<span>{ht('hg.sheet.own')}</span>{/if}
              {#each units as { u, i } (i)}
                <span
                  class="unit"
                  role="img"
                  aria-label={unitTip(c, i)}
                  {@attach tooltip({ text: unitTip(c, i), actionable: true })}
                >
                  <ResourceTile res={u.resource} />
                  {#if source !== 'own'}<span class="unit-price">{formatNumber(u.price)}</span>{/if}
                </span>
              {/each}
              {#if source !== 'own'}
                <span class="subtotal"
                  >{ht('hg.sheet.total', { count: source === 'left' ? c.left : c.right })}</span
                >
              {/if}
            </span>
          {/if}
        {/each}
        {#if c.printed > 0}<span class="bank">{ht('hg.sheet.bank', { count: c.printed })}</span
          >{/if}
        {#if c.total === 0 && !c.free && c.kind !== 'sell'}<span>{choiceAction(c)}</span>{/if}
      </span>
      {#if c.kind !== 'sell'}<strong class="total"
          >{ht('hg.sheet.total', { count: c.total })}</strong
        >{/if}
    </button>
  {/each}
</div>

<style>
  .payments {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .payment {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    width: 100%;
    min-width: 44px;
    min-height: 44px;
    padding: 0.6rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .chosen {
    border-color: var(--accent);
    box-shadow: inset 0 0 0 1.5px var(--accent);
    background: color-mix(in oklch, var(--accent) 12%, var(--panel));
  }
  .radio {
    flex: 0 0 auto;
    width: 1.1rem;
    height: 1.1rem;
    border: 2px solid var(--rule-strong);
    border-radius: 50%;
  }
  .chosen .radio {
    border-color: var(--accent);
    background: var(--accent);
    box-shadow: inset 0 0 0 3px var(--panel);
  }
  .details {
    display: flex;
    flex: 1;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem 0.6rem;
    min-width: 0;
  }
  .source {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.2rem;
  }
  .name {
    max-width: 9rem;
    overflow-wrap: anywhere;
    font-weight: 600;
  }
  .unit {
    display: inline-flex;
    align-items: center;
    gap: 0.15rem;
  }
  .unit-price {
    color: var(--fg-muted);
    font-size: 0.8rem;
  }
  .subtotal,
  .total {
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .subtotal,
  .bank {
    font-size: 0.85rem;
    color: var(--fg-muted);
  }
  .total {
    margin-inline-start: auto;
  }
  .payment:focus-visible {
    outline: 3px solid var(--focus);
    outline-offset: 2px;
  }
  .payment:disabled {
    cursor: default;
    opacity: 0.65;
  }
</style>
