<script lang="ts">
  import { i18n } from '$lib/i18n';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import type { Choice, Disabled, OfferEntry } from '$lib/wire/haftgane.gen';
  import type { HgColour } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import { cardName } from './labels';
  import { COLOURS, design } from './model';
  import HandCard from './HandCard.svelte';
  interface Props {
    offer: readonly OfferEntry[];
    choices: readonly Choice[];
    disabled: readonly Disabled[];
    pendingKey: string | null;
    locked: boolean;
    ondraft: (key: string) => void;
    onconfirm: (key: string) => void;
    onclose: () => void;
  }
  let { offer, choices, disabled, pendingKey, locked, ondraft, onconfirm, onclose }: Props =
    $props();
  let open = $state(true);
  let filter = $state<HgColour | null>(null);
  const cards = $derived(
    offer.filter((o) => filter === null || design(o.design)?.colour === filter)
  );
  const selected = $derived(
    choices.find((c) => c.kind === 'build-discarded' && c.key === pendingKey)
  );
  /** The server's reason a pile card is not offered. */
  function why(slug: string): Disabled | undefined {
    return disabled.find((d) => d.kind === 'build-discarded' && d.design === slug);
  }
  function reason(d: Disabled | undefined): string {
    return d?.reason === 'already-built'
      ? ht('hg.sheet.blocked.already-built')
      : ht('hg.sheet.blocked.other');
  }
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-discard-title"
  class="hg-discard-picker"
  panelData={{ 'data-discard-picker': '' }}
>
  <div class="content" dir={i18n.dir}>
    <h2 id="hg-discard-title">{ht('hg.special.discard.title')}</h2>
    <p>{ht('hg.special.discard.private')}</p>
    <div class="filters" role="group" aria-label={ht('hg.special.discard.filter')}>
      {#each [null, ...COLOURS] as colour (colour)}<button
          type="button"
          data-act="discard-filter:{colour ?? 'all'}"
          aria-pressed={filter === colour}
          onclick={() => (filter = colour)}
          >{colour ? ht(`hg.screen.colourName.${colour}`) : ht('hg.special.discard.all')}</button
        >{/each}
    </div>
    <div class="cards">
      {#each cards as o (o.design)}
        {@const c = choices.find((c) => c.kind === 'build-discarded' && c.design === o.design)}
        <div class="entry" class:blocked={!o.legal || !c}>
          <div class="face">
            <HandCard
              slug={o.design}
              afford={c ? { kind: 'free', via: c.free } : null}
              selected={selected?.design === o.design}
              disabled={locked || !o.legal || !c}
              actKey={c?.key}
              onselect={() => {
                if (o.legal && c) ondraft(c.key);
              }}
            />
          </div>
          {#if !o.legal || !c}
            {@const d = why(o.design)}
            <p data-blocked-reason={d?.reason ?? ''}>{reason(d)}</p>
          {/if}
          {#if o.count > 1}<p>{ht('hg.special.discard.copies', { count: o.count })}</p>{/if}
        </div>
      {/each}
    </div>
    {#if cards.length === 0}<p>{ht('hg.special.discard.empty')}</p>{/if}
    <Button variant="secondary" data-act="discard-close" onclick={onclose}
      >{ht('hg.screen.pick.close')}</Button
    >
  </div>
  {#snippet footer()}<div dir={i18n.dir}>
      <Button
        class="confirm"
        data-act={selected?.key ?? 'discard-confirm'}
        data-special-confirm
        disabled={locked || !selected}
        onclick={() => {
          if (selected) onconfirm(selected.key);
        }}
        >{selected
          ? ht('hg.special.discard.confirm', { card: cardName(selected.design) })
          : ht('hg.screen.bar.discard')}</Button
      >
    </div>{/snippet}
</Sheet>

<style>
  :global(dialog.sheet.hg-discard-picker) {
    width: min(100%, 52rem);
    --sheet-max-height: calc(100dvh - 0.5rem);
  }
  .content {
    padding: 0.6rem 0.75rem 1rem;
    color: var(--fg);
  }
  h2 {
    margin: 0;
    font: 600 1.5rem/1.15 var(--type-display);
  }
  p {
    font-size: 0.85rem;
    line-height: 1.4;
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin-block: 0.75rem;
  }
  .filters button {
    min-width: 44px;
    min-height: 44px;
    padding: 0.4rem 0.6rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 0.8rem;
    cursor: pointer;
  }
  .filters button[aria-pressed='true'] {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
  }
  .cards {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.6rem;
    margin-block: 1rem;
  }
  .entry {
    min-width: 0;
  }
  .face {
    height: 9.5rem;
  }
  .blocked .face {
    opacity: 0.55;
  }
  .entry p {
    margin: 0.3rem 0;
    overflow-wrap: anywhere;
  }
  :global(.hg-discard-picker .confirm) {
    width: 100%;
    height: auto;
    min-height: 44px;
    white-space: normal;
  }
  @media (min-width: 40rem) {
    .cards {
      grid-template-columns: repeat(6, minmax(0, 1fr));
    }
  }
</style>
