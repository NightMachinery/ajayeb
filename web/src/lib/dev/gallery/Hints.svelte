<script lang="ts">
  // The hint settings (docs/dev/decisions.md D23) for a visual check in
  // both themes: the editor as the admin and read-only, the "On my screen"
  // local hide, the collapsible new-game section, and the lobby summary.
  // Demo state only; nothing is sent or saved.
  //
  // The per-card hints (M10.HS2): the pip on mini and medium cards for
  // each code and side (solid, outlined, a hot-seat tint, a spectator's
  // tints, the "more" dot, a covered card), the badge's Urbanism bonus and
  // Economy notch, the card sheet's hint sections, the Mausoleum pool and
  // the destroy choice's lines.
  import HintLocalHide from '$lib/components/hints/HintLocalHide.svelte';
  import FirstTip from '$lib/components/hints/FirstTip.svelte';
  import HintGlyphLegend from '$lib/components/hints/HintGlyphLegend.svelte';
  import MoveReview from '$lib/components/game/MoveReview.svelte';
  import MoveSummary from '$lib/components/game/MoveSummary.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import { summarize } from '$lib/view/move-summary';
  import { reviewLines } from '$lib/view/review';
  import type { PriceTick } from '$lib/view/recap';
  import HintSettingsEditor from '$lib/components/hints/HintSettingsEditor.svelte';
  import HintsSection from '$lib/components/hints/HintsSection.svelte';
  import HintsSummary from '$lib/components/hints/HintsSummary.svelte';
  import { defaultHints, withFamily, type LocalHide } from '$lib/hints/families';
  import { t } from '$lib/i18n';
  import type { CardHints, HintFamily, HintSettings, View } from '$lib/wire';
  import { HINT_FAMILIES } from '$lib/hints/families';
  import CardFace from '$lib/components/card/CardFace.svelte';
  import HintPip from '$lib/components/card/HintPip.svelte';
  import HintLines from '$lib/components/game/HintLines.svelte';
  import DestroyChoice from '$lib/components/choices/DestroyChoice.svelte';
  import DiscardPickChoice from '$lib/components/choices/DiscardPickChoice.svelte';
  import { cardName } from '$lib/components/card/effect-text';
  import {
    cardPip,
    cardSections,
    hintScreen,
    type CardPip,
    type HintNames,
    type Perspective
  } from '$lib/view/hints';

  let edited = $state<HintSettings>(withFamily(defaultHints('online'), 'prices', false));
  let draft = $state<HintSettings>(defaultHints('ai'));
  let hide = $state.raw<LocalHide>({ families: ['counting'], boardMarks: false });

  const ALL: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);
  const YOU: HintNames = { names: ['Sun', 'Moon'], you: 0 }; // i18n-ok
  const SHARED: HintNames = { names: ['Sun', 'Moon'], you: null }; // i18n-ok
  const SEAT0: Perspective = { kind: 'seat', seat: 0 };

  function ch(over: Partial<CardHints> = {}): CardHints {
    return {
      codes: [],
      effect: { paid: true, coins: 0, scoreDelta: 0, oppScoreDelta: 0 },
      priceShift: [],
      ...over
    };
  }
  const pair = (symbol: 'quill' | 'mortar' | 'wheel'): CardHints =>
    ch({
      codes: ['pair'],
      effect: {
        paid: true,
        coins: 0,
        scoreDelta: 0,
        oppScoreDelta: 0,
        science: { symbol, kind: 'pair', distinctAfter: 3, tokenOnBoard: true }
      }
    });
  const win = ch({
    codes: ['win'],
    effect: {
      paid: true,
      coins: 0,
      scoreDelta: 5,
      oppScoreDelta: 0,
      wins: { kind: 'military', steps: 1 },
      military: { from: 7, to: 9, loot: [] }
    }
  });
  const loot = ch({
    codes: ['loot'],
    effect: {
      paid: true,
      coins: 0,
      scoreDelta: 3,
      oppScoreDelta: -1,
      military: { from: 2, to: 3, loot: [{ at: 3, lost: 2 }] }
    }
  });
  const producer = ch({
    priceShift: [{ seat: 1, resource: 'clay', from: 3, to: 5 }],
    savesInView: 6,
    lastProducer: 'lastVisible'
  });

  interface Specimen {
    label: string;
    slug: string;
    pip: CardPip | undefined;
    faint?: boolean;
  }
  const specimens: Specimen[] = [
    {
      label: 'own pair (solid)',
      slug: 'library',
      pip: cardPip([pair('quill'), ch()], SEAT0, ALL, YOU)
    },
    {
      label: 'deny: their pair (outlined)',
      slug: 'dispensary',
      pip: cardPip([ch(), pair('mortar')], SEAT0, ALL, YOU)
    },
    { label: 'own win', slug: 'arsenal', pip: cardPip([win, ch()], SEAT0, ALL, YOU) },
    { label: 'their win', slug: 'walls', pip: cardPip([ch(), win], SEAT0, ALL, YOU) },
    { label: 'own loot', slug: 'garrison', pip: cardPip([loot, ch()], SEAT0, ALL, YOU) },
    { label: 'their loot', slug: 'stable', pip: cardPip([ch(), loot], SEAT0, ALL, YOU) },
    {
      label: 'pair + their win (more dot)',
      slug: 'school',
      pip: cardPip([pair('wheel'), win], SEAT0, ALL, YOU)
    },
    {
      label: 'hot-seat (Moon to move)',
      slug: 'library',
      pip: cardPip([ch(), pair('quill')], { kind: 'hotseat', seat: 1 }, ALL, SHARED)
    },
    {
      label: 'spectator, Sun',
      slug: 'library',
      pip: cardPip([pair('quill'), ch()], { kind: 'spectator', seat: 0 }, ALL, SHARED)
    },
    {
      label: 'spectator, Moon',
      slug: 'dispensary',
      pip: cardPip([ch(), pair('mortar')], { kind: 'spectator', seat: 0 }, ALL, SHARED)
    },
    {
      label: 'covered (half strength)',
      slug: 'library',
      pip: cardPip([pair('quill'), ch()], SEAT0, ALL, YOU),
      faint: true
    }
  ]; // i18n-ok

  const screen = hintScreen({ viewer: 0, active: 0 }, 'online', ALL, true, ['Sun', 'Moon']); // i18n-ok
  const sections = cardSections('brickyard', [producer, loot], SEAT0, ALL, YOU);
  const pairSections = cardSections('library', [pair('quill'), win], SEAT0, ALL, YOU);
  const poolView = {
    hints: {
      discard: [
        { card: 'library', seat: 0, hints: pair('quill'), total: 0, chainFree: true },
        { card: 'garrison', seat: 0, hints: loot, total: 2, chainFree: false },
        {
          card: 'baths',
          seat: 0,
          hints: ch({ effect: { paid: false, coins: 0, scoreDelta: 3, oppScoreDelta: 0 } }),
          total: 4,
          chainFree: false
        }
      ]
    }
  } as unknown as View;
  const destroyPending = {
    kind: 'destroyCard',
    seat: 0,
    color: 'brown',
    cards: ['clay-pool', 'quarry'],
    tokens: [],
    hiddenTokens: 0,
    cardHints: [
      {
        card: 'clay-pool',
        priceShift: [{ seat: 0, resource: 'clay', from: 4, to: 3 }],
        savesInView: 3,
        onlySource: false
      },
      {
        card: 'quarry',
        priceShift: [{ seat: 0, resource: 'stone', from: 3, to: 2 }],
        savesInView: 2,
        onlySource: true
      }
    ]
  } as unknown as View['pending'];

  // M10.HS5: the move review, the price ticks, a first-time tip and the
  // Rules panel's glyph legend.
  const REVIEW_NAMES = { names: ['Sun', 'Moon'] as const, seat: 0 as const }; // i18n-ok
  const reviewBetter = reviewLines(
    {
      entries: [],
      ply: 20,
      played: 'discard:walls',
      better: 'construct:walls',
      reason: 'wins',
      gap: 40,
      strength: 'twoStage'
    },
    REVIEW_NAMES
  );
  const reviewLoot = reviewLines(
    {
      entries: [],
      ply: 20,
      played: 'construct:clay-pool',
      better: 'construct:garrison',
      reason: 'loot',
      gap: 4,
      coins: 2,
      strength: 'twoStage'
    },
    REVIEW_NAMES
  );
  const reviewNone = reviewLines(
    {
      entries: [],
      ply: 20,
      played: 'wonder:pyramids:baths',
      reason: 'none',
      gap: 0,
      strength: 'twoStage'
    },
    REVIEW_NAMES
  );
  const reviewTurn = reviewLines(
    {
      entries: [
        { ply: 20, played: 'wonder:great-library:rostrum', reason: 'none', gap: 0 },
        { ply: 21, played: 'token:strategy', better: 'token:philosophy', reason: 'points', gap: 4 },
        { ply: 22, played: 'construct:statue', reason: 'none', gap: 0 }
      ],
      ply: 22,
      played: 'wonder:great-library:rostrum token:strategy construct:statue',
      better: 'wonder:great-library:rostrum token:philosophy',
      reason: 'points',
      gap: 4,
      strength: 'twoStage'
    },
    REVIEW_NAMES
  );
  const reviewSummary = summarize(
    [
      {
        ply: 21,
        seat: 1,
        events: [
          { kind: 'build', private: false, seat: 1, card: 'clay-pit' },
          { kind: 'pay', private: false, seat: 1, amount: 1, toOpponent: 0 }
        ]
      }
    ],
    1
  );
  const reviewPrices: PriceTick[] = [
    { resource: 'clay', from: 2, to: 3, cause: { seat: 1, card: 'clay-pit' } },
    { resource: 'stone', from: 3, to: 2 }
  ];
  let summaryShown = $state(false);
  let tipShown = $state(false);
</script>

<div class="flex flex-col gap-10" data-testid="gallery-hints-section">
  <section class="flex flex-col gap-2" data-testid="gallery-hint-pips">
    <h3 class="text-sm font-semibold text-fg-muted">Hint pips <!-- i18n-ok --></h3>
    <div class="flex flex-wrap gap-x-4 gap-y-5 rounded-panel border border-rule bg-table p-4">
      {#each specimens as s, i (i)}
        <figure class="flex w-[88px] flex-col items-center gap-1">
          <div class="relative" style:width="56px" style:height="86px">
            <CardFace focusable slug={s.slug} tier="mini" width={56} />
            {#if s.pip}
              <HintPip pip={s.pip} card={cardName(s.slug)} size="sm" faint={s.faint} />
            {/if}
          </div>
          <figcaption class="text-center text-[11px] leading-tight text-fg-muted">
            {s.label}
          </figcaption>
        </figure>
      {/each}
      {#each specimens.slice(0, 3) as s, i (i)}
        <figure class="flex w-[110px] flex-col items-center gap-1">
          <div class="relative" style:width="96px" style:height="148px">
            <CardFace focusable slug={s.slug} tier="medium" width={96} />
            {#if s.pip}<HintPip pip={s.pip} card={cardName(s.slug)} />{/if}
          </div>
          <figcaption class="text-center text-[11px] text-fg-muted">
            {`medium: ${s.label}`}
          </figcaption>
        </figure>
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Badge: Urbanism bonus, Economy notch <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap gap-4 rounded-panel border border-rule bg-table p-4">
      <div style:width="96px" style:height="148px">
        <CardFace
          focusable
          slug="library"
          tier="medium"
          width={96}
          afford={{ state: 'chain', coins: 0, shortfall: 0 }}
          bonus={4}
        />
      </div>
      <div style:width="56px" style:height="86px">
        <CardFace
          focusable
          slug="library"
          tier="mini"
          width={56}
          afford={{ state: 'chain', coins: 0, shortfall: 0 }}
          bonus={4}
        />
      </div>
      <div style:width="96px" style:height="148px">
        <CardFace
          focusable
          slug="baths"
          tier="medium"
          width={96}
          afford={{ state: 'trade', coins: 3, shortfall: 0 }}
          flow={3}
          flowTo="Moon"
        />
      </div>
    </div>
  </section>

  <section class="flex flex-col gap-2" data-testid="gallery-hint-lines">
    <h3 class="text-sm font-semibold text-fg-muted">Card sheet hint lines <!-- i18n-ok --></h3>
    <div class="flex flex-col gap-4 rounded-panel border border-rule bg-panel p-4">
      <HintLines sections={pairSections} />
      <HintLines {sections} />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Mausoleum pool and destroy choice <!-- i18n-ok -->
    </h3>
    <div class="flex flex-col gap-6 rounded-panel border border-rule bg-panel p-4">
      <DiscardPickChoice
        cards={['library', 'garrison', 'baths']}
        keys={new Map([
          ['library', 'revive:library'],
          ['garrison', 'revive:garrison'],
          ['baths', 'revive:baths']
        ])}
        canAct
        hintView={poolView}
        hints={screen}
        onact={() => {}}
      />
      <DestroyChoice
        keys={new Map([
          ['clay-pool', 'destroy:clay-pool'],
          ['quarry', 'destroy:quarry']
        ])}
        color="brown"
        opponentName="Moon"
        canAct
        pending={destroyPending}
        hints={screen}
        onact={() => {}}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Hint settings — the admin <!-- i18n-ok --></h3>
    <div class="rounded-panel border border-rule bg-panel p-4">
      <HintSettingsEditor hints={edited} onchange={(next) => (edited = next)} />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Hint settings — read-only, with the local hide <!-- i18n-ok -->
    </h3>
    <div class="flex flex-col gap-5 rounded-panel border border-rule bg-panel p-4">
      <HintSettingsEditor hints={edited} readonly readonlyNote />
      <HintLocalHide
        {hide}
        onfamily={(family, hidden) =>
          (hide = {
            ...hide,
            families: hidden
              ? [...hide.families, family]
              : hide.families.filter((f) => f !== family)
          })}
        onboardmarks={(hidden) => (hide = { ...hide, boardMarks: hidden })}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      New-game Hints section (closed, then open) <!-- i18n-ok -->
    </h3>
    <div class="flex flex-col gap-3 rounded-panel border border-rule bg-panel p-4">
      <HintsSection bind:value={draft} />
      <HintsSection bind:value={draft} open />
    </div>
  </section>

  <section class="flex flex-col gap-2" data-testid="gallery-move-review">
    <h3 class="text-sm font-semibold text-fg-muted">
      Move review, price ticks and a first-time tip <!-- i18n-ok -->
    </h3>
    <div class="flex max-w-sm flex-col gap-3 rounded-panel border border-rule bg-panel p-4">
      <MoveReview lines={reviewBetter} />
      <MoveReview lines={reviewLoot} />
      <MoveReview lines={reviewNone} />
      <MoveReview lines={reviewTurn} />
      <div class="flex flex-wrap gap-2">
        <Button variant="secondary" onclick={() => (summaryShown = true)}>
          Open a summary with a review <!-- i18n-ok -->
        </Button>
        <Button variant="secondary" onclick={() => (tipShown = true)}>
          Open a first-time tip <!-- i18n-ok -->
        </Button>
      </div>
    </div>
  </section>

  <section class="flex flex-col gap-2" data-testid="gallery-hint-legend">
    <h3 class="text-sm font-semibold text-fg-muted">Rules: hint glyphs <!-- i18n-ok --></h3>
    <div class="max-w-md rounded-panel border border-rule bg-panel p-4">
      <HintGlyphLegend />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Lobby summary <!-- i18n-ok --></h3>
    <div class="flex flex-col gap-4 rounded-panel border border-rule bg-table p-4">
      <HintsSummary hints={edited} title={t('hints.lobby.title')} />
      <HintsSummary hints={defaultHints('ai')} title={t('hints.lobby.title')} />
    </div>
  </section>
</div>

<MoveSummary
  open={summaryShown}
  summary={reviewSummary}
  names={REVIEW_NAMES.names}
  prices={reviewPrices}
  onclose={() => (summaryShown = false)}
>
  {#snippet review()}<MoveReview lines={reviewBetter} />{/snippet}
</MoveSummary>

<FirstTip
  open={tipShown}
  id="pair"
  text={t('hints.tip.pair')}
  onclose={() => (tipShown = false)}
  onmore={() => {}}
/>
