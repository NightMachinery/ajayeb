<script lang="ts">
  // /dev/gallery, cards section (M5.S02): every card at every render tier,
  // every wonder at both tile sizes, the four backs, the five affordability
  // badges as recorded views give them (normal and dimmed), and the three
  // age structures from recorded views in a phone-sized and a desktop-sized
  // box. The recorded views are a large module, so they load on demand.
  // It opens with the art frames (ArtFrames.svelte): bitmap art in its real
  // frames next to the placeholders.
  import { cardSlugs, wonderSlugs } from '$lib/catalog';
  import CardBack from '$lib/components/card/CardBack.svelte';
  import CardFace from '$lib/components/card/CardFace.svelte';
  import AffordBadge from '$lib/components/card/AffordBadge.svelte';
  import CardStructure from '$lib/components/board/CardStructure.svelte';
  import WonderTile from '$lib/components/wonder/WonderTile.svelte';
  import { t } from '$lib/i18n';
  import { AFFORD_STATES, affordOf, badgeSeat, type Afford } from '$lib/view/afford';
  import { describeCard, describeWonder } from '$lib/view/describe';
  import type { Back, View, WonderView } from '$lib/wire';
  import ArtFrames from './ArtFrames.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  const BACKS: Back[] = ['age1', 'age2', 'age3', 'guild'];

  const TIERS = [
    { tier: 'mini', w: 48, h: 74 },
    { tier: 'mini', w: 54, h: 83 },
    { tier: 'medium', w: 101, h: 156 },
    { tier: 'full', w: 180, h: 278 }
  ] as const;

  // The structure boxes of the design's results table: a 360×740 phone and
  // a 1280×800 desktop.
  const BOXES = [
    { name: 'phone', width: 344, height: 448, gap: 4 },
    { name: 'desktop', width: 648, height: 616, gap: 8 }
  ] as const;

  const STRUCTURES = ['age-1-start-seat0', 'starter-chosen-seat0', 'looting-seat1'] as const;

  interface Recorded {
    badges: { card: string; afford: Afford }[];
    wonders: Record<string, { state: WonderView; afford?: Afford }>;
    structures: { name: string; view: View }[];
  }

  async function loadRecorded(): Promise<Recorded> {
    const { protocolFixtures, protocolFixtureNames } = await import('$lib/wire/fixtures.gen');
    const views = protocolFixtures as View[];
    const badges: Partial<Record<string, { card: string; afford: Afford }>> = {};
    const wonders: Record<string, { state: WonderView; afford?: Afford }> = {};
    const rank = (s: WonderView) => (s.built ? 2 : s.removed ? 1 : 0);
    for (const view of views) {
      const seat = badgeSeat(view);
      for (const slot of view.structure) {
        const cost = slot.costs?.[seat];
        if (!cost || !slot.card) continue;
        const afford = affordOf(cost);
        badges[afford.state] ??= { card: slot.card, afford };
      }
      for (const player of view.players) {
        for (const w of player.wonders) {
          // Keep a built or removed state over an open one, so the gallery
          // shows every state the recordings have.
          const old = wonders[w.wonder];
          if (!old || rank(w) > rank(old.state)) {
            wonders[w.wonder] = { state: w, afford: w.cost ? affordOf(w.cost) : undefined };
          }
        }
      }
    }
    const structures = STRUCTURES.map((name) => ({
      name,
      view: views[protocolFixtureNames.indexOf(name)] as View
    }));
    const ordered = AFFORD_STATES.flatMap((s) => {
      const b = badges[s];
      return b ? [b] : [];
    });
    return { badges: ordered, wonders, structures };
  }

  const recorded = loadRecorded();
</script>

<div class="flex flex-col gap-6">
  <p class="max-w-prose text-sm text-fg-muted">{t('dev.cards.intro')}</p>

  <ArtFrames />

  {#await recorded}
    <p class="text-sm text-fg-muted">{t('dev.cards.loading')}</p>
  {:then data}
    <!-- Affordability badges, from recorded views. -->
    <section id="cards-badges" class="flex flex-col gap-3">
      <h3 class="font-display text-lg text-fg">{t('dev.cards.badges')}</h3>
      {#each [false, true] as dim (dim)}
        <div class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
          {#if dim}<p class="text-xs text-fg-muted">{t('dev.cards.badgesDim')}</p>{/if}
          <div class="flex flex-wrap items-end gap-3">
            {#each data.badges as b (b.afford.state)}
              <div class="flex flex-col items-center gap-1" data-badge-state={b.afford.state}>
                <div class="h-[83px] w-[54px]" {@attach tooltip({ text: describeCard(b.card) })}>
                  <CardFace slug={b.card} tier="mini" width={54} afford={b.afford} {dim} />
                </div>
                <div class="h-[156px] w-[101px]">
                  <CardFace slug={b.card} tier="medium" width={101} afford={b.afford} {dim} />
                </div>
                <AffordBadge afford={b.afford} {dim} class="text-[12px]" />
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </section>

    <!-- Backs. -->
    <section id="cards-backs" class="flex flex-col gap-3">
      <h3 class="font-display text-lg text-fg">{t('dev.cards.backs')}</h3>
      <div class="flex flex-wrap items-end gap-3 rounded-panel border border-rule bg-panel p-3">
        {#each BACKS as back (back)}
          <div class="h-[83px] w-[54px]" {@attach tooltip({ text: t(`card.back.${back}`) })}>
            <CardBack {back} class="h-full w-full" />
          </div>
        {/each}
        {#each BACKS as back (back)}
          <div class="h-[156px] w-[101px]">
            <CardBack {back} class="h-full w-full" />
          </div>
        {/each}
      </div>
    </section>

    <!-- Every card at every tier. -->
    <section id="cards-band" class="flex flex-col gap-3">
      <h3 class="font-display text-lg text-fg">{t('dev.cards.tier', { tier: 'band' })}</h3>
      <div
        class="grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-2 rounded-panel border border-rule bg-panel p-3"
      >
        {#each cardSlugs as slug (slug)}
          <div {@attach tooltip({ text: describeCard(slug) })}><CardFace {slug} tier="band" /></div>
        {/each}
      </div>
    </section>

    {#each TIERS as size (size.w)}
      <section
        id={size.w === 48 ? 'cards-mini-48' : `cards-${size.tier}`}
        data-card-grid={size.w}
        class="flex flex-col gap-3"
      >
        <h3 class="font-display text-lg text-fg">{t('dev.cards.tier', { tier: size.tier })}</h3>
        <div
          class="grid justify-center gap-2 rounded-panel border border-rule bg-panel p-3"
          style:grid-template-columns="repeat(auto-fill, {size.w}px)"
        >
          {#each cardSlugs as slug (slug)}
            <div
              data-mini-card={slug}
              style:width="{size.w}px"
              style:height="{size.h}px"
              {@attach tooltip({ text: describeCard(slug) })}
            >
              <CardFace {slug} tier={size.tier} width={size.w} />
            </div>
          {/each}
        </div>
      </section>
    {/each}

    <!-- Wonders, at both sizes, in the states the recordings give. -->
    <section id="cards-wonders" class="flex flex-col gap-3">
      <h3 class="font-display text-lg text-fg">{t('dev.cards.wonders')}</h3>
      <div
        class="grid grid-cols-2 gap-3 rounded-panel border border-rule bg-panel p-3 sm:grid-cols-4"
      >
        {#each wonderSlugs as slug (slug)}
          {@const rec = data.wonders[slug]}
          <div {@attach tooltip({ text: describeWonder(slug, rec?.state) })}>
            <WonderTile {slug} size="draft" afford={rec?.afford} />
          </div>
        {/each}
      </div>
      <div
        class="grid grid-cols-1 gap-2 rounded-panel border border-rule bg-panel p-3 sm:grid-cols-2"
      >
        {#each wonderSlugs as slug (slug)}
          {@const rec = data.wonders[slug]}
          <WonderTile {slug} size="city" state={rec?.state} afford={rec?.afford} />
        {/each}
      </div>
    </section>

    <!-- The three ages, from recorded views, in a phone and a desktop box. -->
    <section id="cards-structures" class="flex flex-col gap-3">
      <h3 class="font-display text-lg text-fg">{t('dev.cards.structures')}</h3>
      {#each data.structures as s (s.name)}
        {#each BOXES as box (box.name)}
          <!-- The phone box is the full 344 px of a 360 px screen, so it
               reaches into the gallery's 16 px side padding there. -->
          <figure
            class="flex flex-col gap-1 {box.name === 'phone' ? '-mx-2 sm:mx-0' : ''}"
            data-structure="{s.name}-{box.name}"
          >
            <figcaption class="text-xs text-fg-muted">
              {t('dev.cards.structureBox', {
                fixture: s.name,
                width: box.width,
                height: box.height
              })}
            </figcaption>
            <div
              class="max-w-full rounded-panel bg-panel-sunken outline outline-rule"
              style:width="{box.width}px"
              style:height="{box.height}px"
            >
              <CardStructure
                slots={s.view.structure}
                age={s.view.age}
                seat={badgeSeat(s.view)}
                toMove={s.view.toMove}
                gap={box.gap}
              />
            </div>
          </figure>
        {/each}
      {/each}
    </section>
  {/await}
</div>
