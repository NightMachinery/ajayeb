<script lang="ts">
  // /dev/gallery, art frames (M9.B2): bitmap art inside the real frames, next
  // to the placeholder it replaces. For each kind it shows the ids that have
  // approved art in the generated index (or the ids named by ?art=, a comma
  // list such as ?art=card/theater,wonder/pyramids) and one id without art:
  // cards at the medium and full tiers (and the mini tier, which never shows
  // art), wonders as a draft tile and a city row, tokens at both sizes, the
  // four backs and the three bot avatars. This is where a reviewer checks
  // that art reads in its frame, feathers into the paper and holds up in
  // both themes.
  import { page } from '$app/state';
  import { artId, hasApprovedArt, type ArtKind } from '$lib/art/lookup';
  import BotAvatar from '$lib/art/BotAvatar.svelte';
  import { cardSlugs, tokenSlugs, wonderSlugs } from '$lib/catalog';
  import CardBack from '$lib/components/card/CardBack.svelte';
  import CardFace from '$lib/components/card/CardFace.svelte';
  import ProgressToken from '$lib/components/token/ProgressToken.svelte';
  import WonderTile from '$lib/components/wonder/WonderTile.svelte';
  import { t } from '$lib/i18n';
  import { describeCard } from '$lib/view/describe';
  import type { Back, BotLevel } from '$lib/wire';

  const BACKS: Back[] = ['age1', 'age2', 'age3', 'guild'];
  const LEVELS: BotLevel[] = ['easy', 'normal', 'hard'];
  const MAX_SHOWN = 4;

  const requested = $derived(
    (page.url.searchParams.get('art') ?? '')
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  );

  /** Up to MAX_SHOWN slugs of a kind to show with art (the requested ones
   *  first, then those the index has), then one without art, if any. */
  function pickSlugs(kind: ArtKind, slugs: readonly string[]) {
    const has = (slug: string) => hasApprovedArt(artId(kind, slug));
    const asked = requested
      .filter((id) => id.startsWith(`${kind}/`))
      .map((id) => id.slice(kind.length + 1))
      .filter((slug) => slugs.includes(slug));
    const shown = [...new Set([...asked, ...slugs.filter(has)])].slice(0, MAX_SHOWN);
    const without = slugs.find((slug) => !has(slug) && !shown.includes(slug));
    return without === undefined ? shown : [...shown, without];
  }

  const cards = $derived(pickSlugs('card', cardSlugs));
  const wonders = $derived(pickSlugs('wonder', wonderSlugs));
  const tokens = $derived(pickSlugs('token', tokenSlugs));
</script>

<section id="cards-art" class="flex flex-col gap-3" data-art-frames>
  <h3 class="font-display text-lg text-fg">Art frames <!-- i18n-ok: dev label --></h3>

  <div class="flex flex-wrap items-end gap-3 rounded-panel border border-rule bg-panel p-3">
    {#each cards as slug (slug)}
      <figure
        class="flex flex-wrap items-end gap-2"
        data-art-sample="card/{slug}"
        data-has-art={hasApprovedArt(artId('card', slug))}
      >
        <div class="h-[83px] w-[54px]" title={describeCard(slug)}>
          <CardFace {slug} tier="mini" width={54} />
        </div>
        <div class="h-[156px] w-[101px]">
          <CardFace {slug} tier="medium" width={101} />
        </div>
        <div class="h-[278px] w-[180px]">
          <CardFace {slug} tier="full" width={180} />
        </div>
      </figure>
    {/each}
  </div>

  <!-- One wonder per row on a phone: two columns would squeeze the city
       row's name to a few letters beside its art and cost. -->
  <div
    class="grid grid-cols-1 items-start gap-3 rounded-panel border border-rule bg-panel p-3 sm:grid-cols-2 lg:grid-cols-3"
  >
    {#each wonders as slug (slug)}
      <div
        class="flex flex-col gap-2"
        data-art-sample="wonder/{slug}"
        data-has-art={hasApprovedArt(artId('wonder', slug))}
      >
        <WonderTile {slug} size="draft" />
        <WonderTile {slug} size="city" />
      </div>
    {/each}
  </div>

  <div class="flex flex-wrap items-center gap-4 rounded-panel border border-rule bg-panel p-3">
    {#each tokens as slug (slug)}
      <div
        class="flex items-center gap-2"
        data-art-sample="token/{slug}"
        data-has-art={hasApprovedArt(artId('token', slug))}
      >
        <ProgressToken {slug} />
        <ProgressToken {slug} size="sm" />
        <ProgressToken {slug} class="scale-125" />
      </div>
    {/each}
  </div>

  <div class="flex flex-wrap items-end gap-3 rounded-panel border border-rule bg-panel p-3">
    {#each BACKS as back (back)}
      <div class="h-[156px] w-[101px]" title={t(`card.back.${back}`)}>
        <CardBack {back} class="h-full w-full" />
      </div>
    {/each}
    {#each LEVELS as level (level)}
      <BotAvatar {level} class="size-12" />
      <BotAvatar {level} class="size-8" />
    {/each}
  </div>
</section>
