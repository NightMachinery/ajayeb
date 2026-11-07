<script lang="ts">
  // Decorative approved art. Failed files and unapproved ids keep the
  // caller's procedural fallback, with no broken-image symbol.
  import type { Snippet } from 'svelte';
  import type { Attachment } from 'svelte/attachments';
  import { art } from '$lib/art/available-haftgane.gen';
  import { listenForImage } from '$lib/art/image-events';
  let { id, sizes, fallback }: { id: string; sizes: string; fallback?: Snippet } = $props();
  const entry = $derived(art[id]);
  const variants = $derived(Object.values(entry?.variants ?? {}).sort((a, b) => a.width - b.width));
  const pick = $derived(entry?.variants.default ?? variants.at(-1));
  const srcset = $derived(variants.map((v) => `${v.path} ${v.width}w`).join(', '));
  let failed = $state<string | null>(null);
  const observe: Attachment<HTMLImageElement> = (image) => {
    const src = pick?.path ?? '';
    return listenForImage(
      image,
      () => {},
      () => {
        failed = src;
      }
    );
  };
</script>

{#if pick && failed !== pick.path}
  <img
    src={pick.path}
    {srcset}
    {sizes}
    width={pick.width}
    height={pick.height}
    alt=""
    loading="lazy"
    decoding="async"
    {@attach observe}
  />
{:else}
  {@render fallback?.()}
{/if}

<style>
  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
</style>
