<script lang="ts">
  // /dev/gallery, ornaments, track and logo (M9.V5): every file of the
  // ornaments-track-logo vector set in the way a later component is meant to
  // use it, so the set can be checked in the app in both themes (flip the
  // gallery's theme switch). The girih, the track cells, the capitals and
  // the wordmark are wired into the app (M11.ornament-integration); the
  // rosette, cartouche, divider and border are shown only here.
  //
  // How each piece is meant to be used (see art/vector/log/ornaments-track-logo.md):
  // - rosette, track cells, wordmark: inline SVG, two-tone through color and --i2;
  // - cartouche and divider: three slices of the same file (fixed ends or
  //   centre, the rest stretched with preserveAspectRatio="none");
  // - border: a CSS border mask (mask-border) over a gild background;
  // - girih: a repeating CSS mask over a --rule background at low opacity.
  import { asset } from '$app/paths';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import wordmarkRaw from '$lib/brand/logo-wordmark.svg?raw';
  import borderUrl from '$lib/ornaments/svg/ornament-border.svg?url';
  import girihUrl from '$lib/ornaments/svg/ornament-girih.svg?url';

  const sources = import.meta.glob('../../ornaments/svg/*.svg', {
    query: '?raw',
    import: 'default',
    eager: true
  }) as Record<string, string>;

  type Art = { viewBox: string; width: number; height: number; inner: string };

  /** The viewBox, its size and the markup inside the root <svg> of a file. */
  function parse(raw: string): Art {
    const open = /<svg\b[^>]*>/.exec(raw);
    const viewBox = /viewBox="([^"]+)"/.exec(open?.[0] ?? '')?.[1] ?? '0 0 24 24';
    const [, , width = 24, height = 24] = viewBox.split(/\s+/).map(Number);
    const start = (open?.index ?? 0) + (open?.[0].length ?? 0);
    return { viewBox, width, height, inner: raw.slice(start, raw.lastIndexOf('</svg>')) };
  }

  const art: Record<string, Art> = {};
  for (const [path, raw] of Object.entries(sources)) {
    const id = /([^/]+)\.svg$/.exec(path)?.[1];
    if (id) art[id] = parse(raw);
  }
  const wordmark = parse(wordmarkRaw);

  // The military track: 17 spaces between the capitals; the zone of each
  // space by its distance from the centre (docs/rules/military-science.md).
  const ZONE_CELL = [
    'track-cell',
    'track-cell-2',
    'track-cell-2',
    'track-cell-5',
    'track-cell-5',
    'track-cell-5',
    'track-cell-10',
    'track-cell-10',
    'track-cell-10'
  ] as const;
  const spaces = Array.from({ length: 17 }, (_, k) => k - 8);

  const cartouches = [
    { width: 54, height: 14 },
    { width: 90, height: 18 },
    { width: 140, height: 22 },
    { width: 220, height: 28 }
  ];
</script>

<!-- One window of an ornament file. The markup comes from our own SVG files
     under src/lib/ornaments and src/lib/brand, imported at build time; it
     never carries user or server data. -->
{#snippet win(a: Art, viewBox: string, w: number | undefined, h: number, stretch = false)}
  <svg
    {viewBox}
    width={w}
    height={h}
    preserveAspectRatio={stretch ? 'none' : undefined}
    class="block shrink-0 overflow-hidden {stretch ? 'min-w-0 flex-1' : ''}"
    aria-hidden="true"
  >
    <!-- eslint-disable-next-line svelte/no-at-html-tags -->
    {@html a.inner}
  </svg>
{/snippet}

{#snippet whole(id: string, h: number)}
  {@const a = art[id]}
  {#if a}{@render win(a, a.viewBox, (a.width * h) / a.height, h)}{/if}
{/snippet}

<!-- Fixed ends, stretched middle (the cartouche): slices at x = 20 and 76. -->
{#snippet cartouche(width: number, h: number)}
  {@const a = art['ornament-cartouche']}
  {#if a}
    {@const s = h / a.height}
    <div class="flex" style:width="{width}px" style:height="{h}px">
      {@render win(a, `0 0 20 ${a.height}`, 20 * s, h)}
      {@render win(a, `20 0 56 ${a.height}`, undefined, h, true)}
      {@render win(a, `76 0 20 ${a.height}`, 20 * s, h)}
    </div>
  {/if}
{/snippet}

<!-- Stretched arms, fixed centre (the divider): slices at x = 80 and 112. -->
{#snippet divider(h: number)}
  {@const a = art['ornament-divider']}
  {#if a}
    <div class="flex w-full" style:height="{h}px">
      {@render win(a, `0 0 80 ${a.height}`, undefined, h, true)}
      {@render win(a, `80 0 32 ${a.height}`, (32 * h) / a.height, h)}
      {@render win(a, `112 0 80 ${a.height}`, undefined, h, true)}
    </div>
  {/if}
{/snippet}

<div
  class="flex flex-col gap-5"
  data-testid="gallery-ornaments"
  style:--border-url={`url("${borderUrl}")`}
  style:--girih-url={`url("${girihUrl}")`}
>
  <div class="flex flex-col gap-2">
    <h3 class="font-display text-base text-fg">{t('dev.gallery.ornaments.track')}</h3>
    <div
      dir="ltr"
      class="flex w-full max-w-[340px] items-end gap-px rounded-control bg-panel py-1.5"
      data-testid="gallery-track-row"
    >
      <span class="grid flex-1 place-items-center text-sun-ink [--i-halo:var(--panel)]">
        <Icon name="capital-sun" class="text-[16px]" />
      </span>
      {#each spaces as pos (pos)}
        {@const a = art[ZONE_CELL[Math.abs(pos)] ?? 'track-cell']}
        <span class="min-w-0 flex-1 text-rule-strong [--i2:var(--panel-sunken)]" data-space={pos}>
          {#if a}
            <svg viewBox={a.viewBox} class="block h-auto w-full" aria-hidden="true">
              <!-- eslint-disable-next-line svelte/no-at-html-tags -->
              {@html a.inner}
            </svg>
          {/if}
        </span>
      {/each}
      <span class="grid flex-1 place-items-center text-moon-ink [--i-halo:var(--panel)]">
        <Icon name="capital-moon" class="text-[16px]" />
      </span>
    </div>
  </div>

  <div class="flex flex-col gap-2">
    <h3 class="font-display text-base text-fg">{t('dev.gallery.ornaments.capitals')}</h3>
    <div class="flex flex-wrap items-end gap-4 rounded-panel border border-rule bg-panel p-3">
      {#each [16, 20, 24, 48] as size (size)}
        <span class="flex items-end gap-1 [--i2:var(--panel-sunken)]" style:font-size="{size}px">
          <Icon name="capital-sun" class="text-sun-ink" />
          <Icon name="capital-moon" class="text-moon-ink" />
        </span>
      {/each}
    </div>
  </div>

  <div class="flex flex-col gap-2">
    <h3 class="font-display text-base text-fg">{t('dev.gallery.ornaments.card')}</h3>
    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-3">
      <span
        class="flex items-center gap-2 rounded-control bg-card-paper p-2 text-gild [--i2:var(--card-paper)]"
      >
        {#each [10, 12, 16, 24] as size (size)}{@render whole('ornament-rosette', size)}{/each}
      </span>
      {#each cartouches as c (c.width)}
        <span class="rounded-control bg-card-blue p-1.5 text-gild [--i2:var(--paper)]">
          {@render cartouche(c.width, c.height)}
        </span>
      {/each}
    </div>
  </div>

  <div class="flex flex-col gap-2">
    <h3 class="font-display text-base text-fg">{t('dev.gallery.ornaments.sheet')}</h3>
    <div class="flex flex-col gap-3 rounded-panel border border-rule bg-panel p-3 text-gild">
      {@render whole('ornament-divider', 12)}
      {@render divider(16)}
      <div class="ornament-border grid h-28 place-items-center">
        <span class="font-display text-lg text-fg">{t('app.name')}</span>
      </div>
    </div>
  </div>

  <div class="flex flex-col gap-2">
    <h3 class="font-display text-base text-fg">{t('dev.gallery.ornaments.table')}</h3>
    <div class="flex flex-wrap gap-3">
      <div
        class="girih size-[288px] max-w-full rounded-control border border-dashed border-rule-strong [--girih-alpha:1]"
      ></div>
      <div
        class="girih h-[160px] w-[200px] rounded-control border border-rule bg-table [--girih-alpha:0.08]"
      ></div>
    </div>
  </div>

  <div class="flex flex-col gap-2">
    <h3 class="font-display text-base text-fg">{t('dev.gallery.ornaments.logo')}</h3>
    <div class="flex flex-wrap items-center gap-6 rounded-panel border border-rule bg-panel p-4">
      <svg
        viewBox={wordmark.viewBox}
        width="200"
        class="block text-fg"
        role="img"
        aria-label={t('app.nameNative')}
      >
        <!-- eslint-disable-next-line svelte/no-at-html-tags -->
        {@html wordmark.inner}
      </svg>
      <span class="flex items-end gap-3">
        {#each [16, 32, 48] as size (size)}
          <img src={asset('/favicon.svg')} width={size} height={size} alt="" />
        {/each}
      </span>
    </div>
  </div>
</div>

<style>
  /* The border tile is a mask: the gild background shows through its ink, in
     both themes, with no colour baked into the file. */
  .ornament-border {
    position: relative;
  }
  .ornament-border::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--gild);
    -webkit-mask-box-image: var(--border-url) 16 round;
    mask-border: var(--border-url) 16 round;
    border: 12px solid transparent;
  }
  /* The girih tile repeats as a mask over the rule colour. */
  .girih {
    position: relative;
    overflow: hidden;
  }
  .girih::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--rule);
    opacity: var(--girih-alpha);
    mask: var(--girih-url) 0 0 / 96px 96px repeat;
  }
</style>
