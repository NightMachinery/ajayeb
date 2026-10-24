<script lang="ts">
  import { ageLabel, i18n, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import type { Choice, Disabled, Neighbour, Seat } from '$lib/wire/haftgane.gen';
  import { hgDesignSlugs } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import { cardName, choiceAction, choiceLabel } from './labels';
  import { afford, cardChoices, cheapest, design, stagesOf } from './model';
  import { effectText } from './effect-text';
  import HandCard from './HandCard.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import ChainIcon from './ChainIcon.svelte';
  import CostPips from './CostPips.svelte';
  import ResourceTile from './ResourceTile.svelte';
  import PaymentList from './PaymentList.svelte';

  interface Props {
    card: string;
    choices: readonly Choice[];
    disabled: readonly Disabled[];
    seat: Seat;
    neighbours: readonly Neighbour[];
    names: Readonly<Record<number, string>>;
    pendingKey: string | null;
    locked: boolean;
    confirmMoves: boolean;
    ondraft: (key: string) => void;
    onconfirm: (key: string) => void;
    onclose: () => void;
  }
  let {
    card,
    choices,
    disabled,
    seat,
    neighbours,
    names,
    pendingKey,
    locked,
    confirmMoves,
    ondraft,
    onconfirm,
    onclose
  }: Props = $props();
  let open = $state(true);
  const d = $derived(design(card));
  const cc = $derived(cardChoices(card, choices, disabled));
  let buildKey = $state<string | null>(null);
  let stageKey = $state<string | null>(null);
  const build = $derived(
    cc.builds.find((c) => c.key === buildKey) ??
      cc.builds.find((c) => c.key === pendingKey) ??
      cheapest(cc.builds)
  );
  const stage = $derived(
    cc.stages.find((c) => c.key === stageKey) ??
      cc.stages.find((c) => c.key === pendingKey) ??
      cheapest(cc.stages)
  );
  const sale = $derived(cc.sells[0]);
  // The server's sixth-turn preview: the card each action keeps for an
  // extra turn. Said once when every action here keeps the same card.
  const offered = $derived([build, stage, sale].filter((c) => c !== undefined));
  const keptByAll = $derived(
    offered.length > 0 && offered.every((c) => c.kept && c.kept === offered[0]!.kept)
      ? offered[0]!.kept!
      : null
  );
  // The footer leads with the outcome the seat has drafted here, or the
  // build when it has drafted nothing here, and names it with the card;
  // the other outcomes follow in the sheet's order.
  const chosen = $derived(offered.find((c) => c.key === pendingKey) ?? build);
  const actions = $derived(chosen ? [chosen, ...offered.filter((c) => c !== chosen)] : offered);
  const why = $derived(cc.disabled.find((x) => x.kind === 'build'));
  const stageCost = $derived(stage ? stagesOf(seat)[stage.stage - 1]?.cost : undefined);
  const chainSources = $derived(
    d?.chainIn ? hgDesignSlugs.filter((slug) => design(slug)?.chainOut?.includes(d.chainIn!)) : []
  );
  const buildsFree = $derived(build?.free ?? '');

  const discounts = $derived(
    Array.from(
      new Set(
        (build?.witness ?? []).flatMap((u) => {
          const d = neighbours
            .find((n) => u.source.startsWith(n.direction))
            ?.units.find((p) => p.producer === u.producer)
            ?.options.find((o) => o.resource === u.resource && o.price === u.price)?.discount;
          return d
            ? [
                d.design
                  ? ht('hg.screen.trade.discountCard', { card: cardName(d.design) })
                  : ht('hg.screen.trade.discountStage', { stage: d.stage })
              ]
            : [];
        })
      )
    )
  );

  // Keep the sheet's arming state in this lazy entry. Importing the
  // shared ConfirmButton extracts it from Duel's existing PlayerStrip
  // chunk and makes hot-seat Duel fetch it even though it never renders.
  // Match that control's 3 s window and 350 ms accidental-double-tap guard.
  let armedKey = $state<string | null>(null);
  let armedAt = 0;
  let resetTimer: ReturnType<typeof setTimeout> | undefined;
  function disarm(): void {
    clearTimeout(resetTimer);
    resetTimer = undefined;
    armedKey = null;
  }
  function act(c: Choice): void {
    if (!confirmMoves) {
      onconfirm(c.key);
      return;
    }
    const now = Date.now();
    if (armedKey !== c.key) {
      disarm();
      armedKey = c.key;
      armedAt = now;
      resetTimer = setTimeout(disarm, 3000);
      return;
    }
    if (now - armedAt < 350) return;
    disarm();
    onconfirm(c.key);
  }
  $effect(() => () => clearTimeout(resetTimer));

  function draft(key: string): void {
    disarm();
    const c = choices.find((x) => x.key === key);
    if (c?.kind === 'build') buildKey = key;
    if (c?.kind === 'stage') stageKey = key;
    ondraft(key);
  }
  function blocked(): string {
    if (why?.reason === 'insufficient-coins')
      return ht('hg.sheet.blocked.insufficient-coins', {
        count: why.shortBy,
        cheapest: why.cheapest
      });
    if (why?.reason === 'already-built') return ht('hg.sheet.blocked.already-built');
    if (why?.reason === 'cannot-cover') return ht('hg.sheet.blocked.cannot-cover');
    return ht('hg.sheet.blocked.other');
  }
</script>

{#snippet keptNote(card: string, all: boolean)}
  <p class="kept-preview" data-kept-preview={card}>
    <span
      role="img"
      aria-label={ht('hg.special.kept.preview.tip')}
      {@attach tooltip({ text: ht('hg.special.kept.preview.tip') })}><HgGlyph name="wonder" /></span
    >{ht(all ? 'hg.special.kept.preview.all' : 'hg.special.kept.preview.this', {
      card: cardName(card)
    })}
  </p>
{/snippet}

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-card-sheet-title"
  class="hg-card-sheet"
  panelData={{ 'data-card-sheet': card }}
>
  <div class="content" dir={i18n.dir}>
    <header class="head">
      <div class="face"><HandCard slug={card} afford={afford(cc)} preview /></div>
      <div class="summary">
        <div class="title">
          <h2 id="hg-card-sheet-title">{cardName(card)}</h2>
          <button
            type="button"
            class="close"
            data-act="pick-close"
            aria-label={ht('hg.screen.pick.close')}
            onclick={onclose}
            {@attach tooltip({ text: ht('hg.screen.pick.close'), actionable: true })}>×</button
          >
        </div>
        {#if d}
          <p class="category">
            {ht(`hg.screen.colourName.${d.colour}`)} · {ht('hg.screen.age', {
              age: ageLabel(d.age)
            })}
          </p>
          <p class="effects">{d.effects.map(effectText).join(' ')}</p>
          <div class="chains">
            {#if d.chainIn}
              {#each chainSources as source (source)}
                <span class="chain"
                  ><ChainIcon chain={d.chainIn} direction="in" />
                  {ht(seat.built.includes(source) ? 'hg.sheet.chainOwned' : 'hg.sheet.chain', {
                    card: cardName(source)
                  })}</span
                >
              {/each}
            {/if}
            {#each d.chainOut ?? [] as chain (chain)}
              <span class="chain"
                ><ChainIcon {chain} direction="out" />{ht('hg.sheet.chainOut', {
                  card: cardName(chain)
                })}</span
              >
            {/each}
          </div>
        {/if}
      </div>
    </header>
    <div class="options">
      {#if keptByAll}{@render keptNote(keptByAll, true)}{/if}
      <section aria-labelledby="hg-build-title">
        <h3 id="hg-build-title">{ht('hg.sheet.build')}</h3>
        <div class="cost">
          <span>{ht('hg.sheet.cost')}</span>{#if d}<CostPips cost={d.cost} />{/if}
        </div>
        {#if build}
          {#if buildsFree}<p class="free">
              {ht(`hg.screen.afford.free.${buildsFree}` as 'hg.screen.afford.free.chain')}
            </p>{/if}
          {#if buildsFree === 'first-colour'}<p>{ht('hg.sheet.firstColour')}</p>{/if}
          {#if buildsFree}
            <button
              type="button"
              class="free-pick"
              data-act={build.key}
              data-pick={build.key}
              disabled={locked}
              onclick={() => draft(build.key)}>{choiceAction(build)}</button
            >
          {:else}
            <div class="witness-summary">
              {#each ['own', 'buy'] as source (source)}
                {@const units = build.witness.filter((u) =>
                  source === 'own' ? u.source.startsWith('own') : !u.source.startsWith('own')
                )}
                {#if units.length}
                  <span
                    >{ht(source === 'own' ? 'hg.sheet.own' : 'hg.sheet.buy')}
                    {#each units as u, i (i)}<span
                        role="img"
                        aria-label={ht(`hg.resource.${u.resource}` as 'hg.resource.wood')}
                        {@attach tooltip({
                          text: ht(`hg.resource.${u.resource}` as 'hg.resource.wood')
                        })}><ResourceTile res={u.resource} /></span
                      >{/each}</span
                  >
                {/if}
              {/each}
            </div>
            <PaymentList
              choices={cc.builds}
              selected={build.key}
              label={ht('hg.sheet.build')}
              {neighbours}
              {names}
              {locked}
              onselect={draft}
            />
          {/if}
        {:else}<p class="blocked" data-cannot-build>{blocked()}</p>{/if}
        {#if build?.kept && !keptByAll}{@render keptNote(build.kept, false)}{/if}
        <p class="note">{ht('hg.sheet.coins', { count: seat.coins })} {discounts.join(' ')}</p>
      </section>
      <section aria-labelledby="hg-stage-title">
        <h3 id="hg-stage-title">
          {stage ? ht('hg.sheet.stage', { stage: stage.stage }) : ht('hg.sheet.cannotStage')}
        </h3>
        {#if stage}
          {#if stageCost}<div class="cost">
              <span>{ht('hg.sheet.cost')}</span><CostPips cost={stageCost} />
            </div>{/if}
          <PaymentList
            choices={cc.stages}
            selected={stage.key}
            label={ht('hg.sheet.stage', { stage: stage.stage })}
            {neighbours}
            {names}
            {locked}
            onselect={draft}
          />
          <p class="note">{ht('hg.sheet.hiddenStage')}</p>
          {#if stage.kept && !keptByAll}{@render keptNote(stage.kept, false)}{/if}
        {/if}
      </section>
      {#if sale}
        <section aria-labelledby="hg-sale-title">
          <h3 id="hg-sale-title">{ht('hg.sheet.sell')}</h3>
          <PaymentList
            choices={cc.sells}
            selected={pendingKey === sale.key ? sale.key : null}
            label={ht('hg.sheet.sell')}
            {neighbours}
            {names}
            {locked}
            onselect={draft}
          />
          <p class="note">{ht('hg.sheet.hiddenSale')}</p>
          {#if sale.kept && !keptByAll}{@render keptNote(sale.kept, false)}{/if}
        </section>
      {/if}
      {#if choices.some((c) => c.key === pendingKey)}<p class="note" role="status">
          {ht('hg.sheet.draft', {
            action: choiceLabel(choices.find((c) => c.key === pendingKey)!)
          })}
        </p>{/if}
    </div>
  </div>
  {#snippet footer()}
    <div class="actions" dir={i18n.dir}>
      {#if !chosen}<Button disabled class="sheet-action">{ht('hg.sheet.cannotBuild')}</Button>{/if}
      {#each actions as c (c.key)}
        {@const label = c === chosen ? choiceLabel(c) : choiceAction(c)}
        <Button
          variant={c === chosen ? 'primary' : 'secondary'}
          class="sheet-action relative isolate overflow-hidden"
          data-act={c.key}
          data-kind={c.kind}
          data-armed={armedKey === c.key ? '' : undefined}
          disabled={locked}
          onclick={() => act(c)}
        >
          {#if armedKey === c.key}
            <span class="confirm-ring" aria-hidden="true"></span>
          {/if}
          {armedKey === c.key ? t('shell.confirmArmed', { label }) : label}
        </Button>
      {/each}
    </div>
  {/snippet}
</Sheet>

<style>
  .kept-preview {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    margin: 0.4rem 0 0;
    font-size: 0.85rem;
    font-weight: 600;
    line-height: 1.35;
  }
  :global(dialog.sheet.hg-card-sheet) {
    width: min(100%, 48rem);
    --sheet-max-height: calc(100dvh - 0.5rem);
  }
  .content {
    display: grid;
    gap: 1rem;
    padding: 0.6rem 1rem 1rem;
    color: var(--fg);
  }
  .head {
    display: flex;
    align-items: flex-start;
    gap: 0.8rem;
    min-width: 0;
  }
  .face {
    flex: 0 0 6.5rem;
    height: 10rem;
  }
  .summary {
    flex: 1;
    min-width: 0;
  }
  .title {
    display: flex;
    align-items: flex-start;
    gap: 0.3rem;
  }
  h2 {
    flex: 1;
    margin: 0;
    font: 600 1.55rem/1.15 var(--type-display);
    overflow-wrap: anywhere;
  }
  .close {
    display: grid;
    flex: 0 0 44px;
    place-items: center;
    width: 44px;
    height: 44px;
    margin-block-start: -0.4rem;
    border: 0;
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 1.5rem;
    cursor: pointer;
  }
  p {
    margin: 0.4rem 0;
  }
  .category,
  .note {
    color: var(--fg-muted);
    font-size: 0.85rem;
    line-height: 1.4;
  }
  .effects {
    font-size: 0.9rem;
    line-height: 1.4;
  }
  .chains {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }
  .chain {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    padding: 0.2rem 0.45rem;
    border: 1px solid var(--rule-strong);
    border-radius: 1rem;
    background: var(--panel-sunken);
    font-size: 0.75rem;
  }
  .options,
  section {
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    min-width: 0;
  }
  .options {
    gap: 0.9rem;
  }
  h3 {
    margin: 0;
    color: var(--fg-muted);
    font: 600 1rem var(--type-display);
  }
  .cost {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    color: var(--fg-muted);
    font-size: 0.85rem;
  }
  .free-pick {
    min-height: 44px;
    min-width: 44px;
    padding: 0.5rem;
    border: 1px solid var(--afford-ok-edge);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .witness-summary,
  .witness-summary > span {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.25rem;
  }
  .witness-summary {
    gap: 0.75rem;
    font-size: 0.85rem;
  }
  .free {
    color: var(--fg);
    font-weight: 600;
  }
  .blocked {
    padding: 0.6rem;
    border-inline-start: 3px solid var(--cinnabar);
    background: var(--afford-short-bg);
    color: var(--cinnabar-paper-ink);
  }
  .confirm-ring {
    position: absolute;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    background: color-mix(in oklch, var(--on-accent) 20%, transparent);
    transform-origin: left; /* physical-ok */
    animation: sheet-confirm-shrink 3000ms linear forwards;
  }
  .confirm-ring:dir(rtl) {
    transform-origin: right; /* physical-ok */
  }
  @keyframes sheet-confirm-shrink {
    from {
      scale: 1 1;
    }
    to {
      scale: 0 1;
    }
  }
  .actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
  }
  .actions > :global(:first-child) {
    grid-column: 1 / -1;
  }
  .actions :global(.sheet-action) {
    width: 100%;
    height: auto;
    min-width: 44px;
    min-height: 44px;
    padding: 0.65rem 0.5rem;
    line-height: 1.25;
    overflow-wrap: anywhere;
  }
  @media (min-width: 64rem), (orientation: landscape) and (max-height: 32rem) {
    :global(dialog.sheet.hg-card-sheet) {
      width: min(66rem, calc(100% - 1rem));
    }
    .content {
      grid-template-columns: minmax(0, 0.85fr) minmax(0, 1.15fr);
    }
    .face {
      flex-basis: 6rem;
      height: 9rem;
    }
    .confirm-ring {
      position: absolute;
      inset: 0;
      z-index: -1;
      pointer-events: none;
      background: color-mix(in oklch, var(--on-accent) 20%, transparent);
      transform-origin: left; /* physical-ok */
      animation: sheet-confirm-shrink 3000ms linear forwards;
    }
    .confirm-ring:dir(rtl) {
      transform-origin: right; /* physical-ok */
    }
    @keyframes sheet-confirm-shrink {
      from {
        scale: 1 1;
      }
      to {
        scale: 0 1;
      }
    }
    .actions {
      grid-template-columns: 2fr 1fr 1fr;
    }
    .actions > :global(:first-child) {
      grid-column: auto;
    }
  }
</style>
