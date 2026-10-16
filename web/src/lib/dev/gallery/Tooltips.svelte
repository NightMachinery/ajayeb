<script lang="ts">
  // /dev/gallery/tooltips: every kind of tooltip the layer shows, for a
  // visual check in both themes and for e2e/specs/tooltips.spec.ts. Plain
  // text, text with parameters, a diagram, the stacked chain tooltips, a
  // chip inside an actionable card (innermost wins; the card keeps its tap
  // and shows its own tooltip on a long-press), a button, the bubble and
  // stack drawn standalone, and the dismiss layer. The micro-elements
  // section (M10.tooltip-coverage) draws the board's own components with
  // fixed data, each with the tooltip it has in a game: a cost pip with its
  // pay lines, an effect glyph, the trade badge with the trade diagram, a
  // face-down back with its candidates, the coin tally with the next point,
  // a chain chip with its status lines, the compact track's zones, and a
  // bubble with its "More in Rules" link.
  //
  // Two query parameters help screenshots: `show=<id>` opens the tooltip of
  // the element marked data-tip-demo="<id>" once the page has settled, and
  // `edges` adds stacked chain chips at the four corners of the viewport.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import CardFace from '$lib/components/card/CardFace.svelte';
  import ChainDiagram from '$lib/components/rules/ChainDiagram.svelte';
  import DiscardDiagram from '$lib/components/rules/DiscardDiagram.svelte';
  import GuildDiagram from '$lib/components/rules/GuildDiagram.svelte';
  import SeventhWonderDiagram from '$lib/components/rules/SeventhWonderDiagram.svelte';
  import StarterDiagram from '$lib/components/rules/StarterDiagram.svelte';
  import ChainChip from '$lib/components/card/ChainChip.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import DismissLayer from '$lib/components/shell/DismissLayer.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import TooltipBubble from '$lib/components/shell/TooltipBubble.svelte';
  import TooltipStack from '$lib/components/shell/TooltipStack.svelte';
  import { tooltip, tooltips } from '$lib/ui/tooltip.svelte';
  import { describeCard } from '$lib/view/describe';
  import AffordBadge from '$lib/components/card/AffordBadge.svelte';
  import CardBack from '$lib/components/card/CardBack.svelte';
  import CostPips from '$lib/components/card/CostPips.svelte';
  import EffectGlyph from '$lib/components/card/EffectGlyph.svelte';
  import MilitaryTrack from '$lib/components/board/MilitaryTrack.svelte';
  import Tally from '$lib/components/city/Tally.svelte';
  import ScienceTracker from '$lib/components/city/ScienceTracker.svelte';
  import { cardBySlug } from '$lib/catalog';
  import type { HintScreen } from '$lib/view/hints';
  import { backTip, chainLines, coinsLines } from '$lib/view/tips';
  import type { CostView, HintFamily, HintsView, PlayerHintsView } from '$lib/wire';
  import { HINT_FAMILIES } from '$lib/hints/families';

  interface Props {
    /** Open this demo's tooltip once the page has settled. */
    show?: string;
    /** Add chain chips at the viewport's corners. */
    edges?: boolean;
  }

  let { show, edges = false }: Props = $props();

  let sheetOpen = $state(false);
  let opened = $state(0);
  let pressed = $state(0);
  let dismissOpen = $state(false);

  // --- Fixed data for the micro-elements -----------------------------------
  const SCREEN: HintScreen = {
    perspective: { kind: 'seat', seat: 0 },
    families: new Set<HintFamily>(HINT_FAMILIES),
    boardMarks: true,
    names: { names: ['Sun', 'Moon'], you: 0 }
  };
  const ZERO = { wood: 0, clay: 0, stone: 0, glass: 0, papyrus: 0 };
  /** Aqueduct's 3 stone: one made in the city, two bought at 3 each. */
  const PAY: CostView = {
    affordable: true,
    shortfall: 0,
    chainFree: false,
    printedCoins: 0,
    own: { ...ZERO, stone: 1 },
    producers: [],
    trade: [{ resource: 'stone', units: 2, unitPrice: 3, fixed: false, coins: 6 }],
    tradeTotal: 6,
    total: 6,
    toOpponent: 0,
    urbanismBonus: 0
  };
  const PLAYER_HINTS: PlayerHintsView = {
    coinsToNextVp: 2,
    symbols: [],
    need: 6,
    reachable: true,
    lawWins: false,
    chainTargets: [{ chain: 'drop', target: 'aqueduct', place: 'unseenLaterAge', worth: 6 }],
    guildPoints: [],
    toCapital: 9,
    capitalSafe: false,
    threats: []
  };
  const BOARD_HINTS: HintsView = {
    discard: [],
    wondersLeft: 7,
    pairs: [],
    looting: [],
    tokens: [],
    boxTokens: [],
    guildPreview: [
      { guild: 'builders-guild', coins: [0, 0], vp: [4, 2] },
      { guild: 'moneylenders-guild', coins: [0, 0], vp: [3, 5] }
    ],
    unseen: [
      {
        back: 'guild',
        cards: ['builders-guild', 'moneylenders-guild', 'scientists-guild'],
        faceDown: 2,
        setAside: 1
      }
    ]
  };
  const aqueduct = cardBySlug('aqueduct');
  const glyphEffect = cardBySlug('library')?.effects[0];

  const EDGES = [
    { id: 'edge-tl', class: 'top-2 start-2', chain: 'drop', direction: 'out' },
    { id: 'edge-tr', class: 'top-2 end-2', chain: 'mask', direction: 'out' },
    { id: 'edge-bl', class: 'bottom-2 start-2', chain: 'moon', direction: 'in' },
    { id: 'edge-br', class: 'bottom-2 end-2', chain: 'sword', direction: 'in' }
  ] as const;

  // Screenshots only: scroll the demo into view, let the scroll settle,
  // then open its tooltip. The store is written from the timer, never from
  // the effect itself.
  $effect(() => {
    if (!show) return;
    const target = document.querySelector(`[data-tip-demo="${CSS.escape(show)}"]`);
    const anchor = target?.matches('[data-tooltip]')
      ? target
      : target?.querySelector('[data-tooltip]');
    if (!anchor) return;
    if (!edges) anchor.scrollIntoView({ block: 'center' });
    const timer = setTimeout(() => tooltips.show(anchor, 'api'), 500);
    return () => clearTimeout(timer);
  });
</script>

{#snippet trackDiagram()}
  <!-- A placeholder concept diagram: the conflict track with the pawn
       moving one space. Its labels are translatable text beside the SVG,
       never baked into it. -->
  <span class="flex w-40 flex-col items-stretch gap-0.5">
    <svg viewBox="0 0 160 28" class="h-7 w-40" aria-hidden="true">
      <rect x="2" y="10" width="156" height="8" rx="2" fill="var(--panel-sunken)" />
      <rect x="2" y="10" width="156" height="8" rx="2" fill="none" stroke="var(--rule-strong)" />
      {#each [1, 2, 3, 4, 5, 6, 7, 8] as i (i)}
        <line x1={2 + i * 17.3} y1="10" x2={2 + i * 17.3} y2="18" stroke="var(--rule)" />
      {/each}
      <circle cx="80" cy="14" r="5" fill="var(--cinnabar)" opacity="0.35" />
      <circle cx="97" cy="14" r="5" fill="var(--cinnabar)" />
      <path d="M84 5 L94 5" stroke="currentColor" stroke-width="1.5" />
      <path d="M91 2 L95 5 L91 8" fill="none" stroke="currentColor" stroke-width="1.5" />
    </svg>
    <span class="flex justify-between text-[11px] leading-none">
      <span>{t('dev.tooltips.diagramYou')}</span>
      <span>{t('dev.tooltips.diagramRival')}</span>
    </span>
  </span>
{/snippet}

<!-- The concept diagrams in their compact form, as tooltip diagrams. -->
{#snippet chainDiagram()}<ChainDiagram compact />{/snippet}
{#snippet discardDiagram()}<DiscardDiagram compact />{/snippet}
{#snippet seventhDiagram()}<SeventhWonderDiagram compact />{/snippet}
{#snippet guildDiagram()}<GuildDiagram compact />{/snippet}
{#snippet starterDiagram()}<StarterDiagram compact />{/snippet}

<div class="flex flex-col gap-6 text-fg" data-tooltips-gallery>
  <p class="max-w-prose text-sm text-fg-muted">{t('dev.tooltips.intro')}</p>

  <div class="grid grid-cols-2 gap-4 sm:grid-cols-3">
    <section class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.text')}</h3>
      <span
        class="inline-grid size-11 place-items-center self-start rounded-control text-2xl text-fg"
        data-tip-demo="text"
        data-testid="tip-text"
        {@attach tooltip({ text: t('dev.tooltips.laurelTip') })}
      >
        <Icon name="vp-laurel" />
      </span>
    </section>

    <section class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.params')}</h3>
      <span
        class="flex items-center gap-1 self-start rounded-control border border-rule bg-panel px-1.5 py-0.5 text-xs"
        data-tip-demo="params"
        data-testid="tip-params"
        {@attach tooltip({
          text: t('board.resource.chip', {
            resource: t('card.resource.wood'),
            count: 2,
            price: 3
          })
        })}
      >
        <Icon name="resource-wood" class="text-res-wood-ink" />
        <span class="font-semibold tabular-nums">2</span>
      </span>
    </section>

    <section class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.diagram')}</h3>
      <span
        class="inline-grid size-11 place-items-center self-start rounded-control text-2xl text-cinnabar"
        data-tip-demo="diagram"
        data-testid="tip-diagram"
        {@attach tooltip({
          text: t('dev.tooltips.diagramTip'),
          diagram: trackDiagram,
          rulesId: 'military'
        })}
      >
        <Icon name="shield" />
      </span>
    </section>

    <section class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.stack')}</h3>
      <div class="flex items-center gap-3 text-2xl">
        <span data-tip-demo="stack-out" data-testid="tip-chain-out">
          <ChainChip chain="drop" direction="out" />
        </span>
        <span data-tip-demo="stack-in" data-testid="tip-chain-in">
          <ChainChip chain="drop" direction="in" />
        </span>
        <span data-tip-demo="stack-out-2">
          <ChainChip chain="mask" direction="out" />
        </span>
      </div>
    </section>

    <section class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.inCard')}</h3>
      <button
        type="button"
        class="h-[110px] w-[72px] self-start rounded-[7%/4.5%]"
        aria-label={describeCard('baths')}
        data-card="baths"
        data-tip-demo="card"
        data-testid="tip-card"
        {@attach tooltip({ text: describeCard('baths') })}
        onclick={() => {
          opened++;
          sheetOpen = true;
        }}
      >
        <CardFace slug="baths" tier="medium" width={72} />
      </button>
      <p class="text-xs text-fg-muted" data-testid="tip-card-opened">
        {t('dev.tooltips.cardOpened', { count: opened })}
      </p>
    </section>

    <section class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.button')}</h3>
      <Button
        class="self-start"
        data-tip-demo="button"
        data-testid="tip-button"
        {@attach tooltip({ text: t('dev.tooltips.pressTip') })}
        onclick={() => pressed++}
      >
        {t('dev.tooltips.pressMe')}
      </Button>
      <p class="text-xs text-fg-muted" data-testid="tip-button-pressed">
        {t('dev.tooltips.pressed', { count: pressed })}
      </p>
    </section>
  </div>

  <section class="flex flex-col gap-2" data-tip-micro>
    <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.micro')}</h3>
    <div
      class="flex flex-wrap items-center gap-4 rounded-panel border border-rule bg-panel p-3 text-lg text-card-ink"
    >
      {#if aqueduct}
        <span class="rounded-control bg-card-paper px-1.5 py-1" data-tip-demo="pip">
          <CostPips cost={aqueduct.cost} pay={PAY} />
        </span>
      {/if}
      {#if glyphEffect}
        <span class="rounded-control bg-card-paper px-1.5 py-1" data-tip-demo="glyph">
          <EffectGlyph effect={glyphEffect} />
        </span>
      {/if}
      <span class="text-sm" data-tip-demo="badge-trade">
        <AffordBadge afford={{ state: 'trade', coins: 6, shortfall: 0 }} cost={PAY} />
      </span>
      <span class="block h-[74px] w-12" data-tip-demo="back">
        <CardBack back="guild" tip={backTip('guild', BOARD_HINTS, SCREEN)} class="h-full w-full" />
      </span>
      <span data-tip-demo="tally">
        <Tally
          kind="coins"
          value={7}
          label={t('coins.count', { count: 7 })}
          tip={{
            text: t('coins.count', { count: 7 }),
            lines: coinsLines(PLAYER_HINTS, SCREEN)
          }}
        />
      </span>
      <span data-tip-demo="chain-status">
        <ChainChip
          chain="drop"
          direction="out"
          lines={chainLines('drop', [{ hints: PLAYER_HINTS }, {}], SCREEN)}
        />
      </span>
      <span data-tip-demo="science">
        <ScienceTracker
          science={{
            distinct: 3,
            symbols: [
              { symbol: 'globe', count: 1 },
              { symbol: 'quill', count: 1 },
              { symbol: 'wheel', count: 1 }
            ]
          }}
          pill
        />
      </span>
      <span class="w-56" data-tip-demo="track-zone">
        <MilitaryTrack
          military={{
            position: 2,
            looting: [
              { at: -6, seat: 0, value: 5, present: true },
              { at: -3, seat: 0, value: 2, present: true },
              { at: 3, seat: 1, value: 2, present: true },
              { at: 6, seat: 1, value: 5, present: true }
            ]
          }}
          viewerSeat={0}
          compact
        />
      </span>
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.specimens')}</h3>
    <div
      class="flex flex-wrap items-start gap-4 rounded-panel border border-rule bg-table p-4 pb-6"
    >
      <TooltipBubble text={t('dev.tooltips.laurelTip')} arrowX={40} />
      <TooltipBubble text={t('dev.tooltips.diagramTip')} diagram={trackDiagram} arrowX={120} />
      <TooltipBubble text={t('rules.costs.chainTitle')} diagram={chainDiagram} arrowX={40} />
      <TooltipBubble
        text={t('rules.diagram.discard.action')}
        diagram={discardDiagram}
        arrowX={40}
      />
      <TooltipBubble text={t('rules.wonders.seventhTitle')} diagram={seventhDiagram} arrowX={40} />
      <TooltipBubble text={t('rules.guilds.title')} diagram={guildDiagram} arrowX={40} />
      <TooltipBubble text={t('rules.ageEnd.title')} diagram={starterDiagram} arrowX={40} />
      <TooltipBubble
        text={t('dev.tooltips.diagramTip')}
        lines={[t('dev.tooltips.statusLine')]}
        rulesId="military"
        arrowX={40}
      />
      <TooltipStack
        items={[{ kind: 'card', slug: 'aqueduct', caption: t('card.tip.chainOutStack') }]}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">{t('dev.tooltips.dismiss')}</h3>
    <Button
      variant="secondary"
      class="self-start"
      data-testid="tip-dismiss-open"
      onclick={() => (dismissOpen = true)}
    >
      {t('dev.tooltips.openDismiss')}
    </Button>
  </section>
</div>

{#if edges}
  {#each EDGES as edge (edge.id)}
    <span class="fixed z-10 text-2xl {edge.class}" data-tip-demo={edge.id}>
      <ChainChip chain={edge.chain} direction={edge.direction} />
    </span>
  {/each}
{/if}

<Sheet bind:open={sheetOpen} titleId="tip-sheet-title">
  <div class="flex flex-col gap-3 p-4" data-testid="tip-card-sheet">
    <h2 id="tip-sheet-title" class="font-display text-xl text-fg">
      {t('dev.tooltips.sheetTitle')}
    </h2>
    <p class="text-sm text-fg-muted">{t('dev.tooltips.sheetBody')}</p>
  </div>
</Sheet>

<DismissLayer bind:open={dismissOpen} label={t('dev.tooltips.dismiss')} class="w-72">
  <p class="text-sm" data-testid="tip-dismiss-panel">{t('dev.tooltips.dismissBody')}</p>
  <!-- Something to Tab to: the focus stays inside the open panel. -->
  <Button
    variant="secondary"
    class="mt-3"
    data-testid="tip-dismiss-close"
    onclick={() => (dismissOpen = false)}
  >
    {t('shell.close')}
  </Button>
</DismissLayer>
