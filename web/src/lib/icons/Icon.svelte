<script lang="ts">
  // One icon from the inline sprite (see sprite.ts, inlined once by
  // IconSprite.svelte in +layout.svelte). Decorative by default
  // (aria-hidden); pass `label` (already translated, through t()) to make it
  // a named image instead. Sized in em, so the surrounding font-size controls
  // it; override with a `class` that sets width/height or font-size.
  //
  //   <Icon name="resource-wood" />
  //   <Icon name="shield" label={t('icons.shield')} class="text-cinnabar" />

  import type { IconName } from './names';

  interface Props {
    name: IconName;
    label?: string;
    class?: string;
  }

  let { name, label, class: className = '' }: Props = $props();
</script>

<svg
  class="inline-block size-[1em] shrink-0 align-[-0.125em] {className}"
  aria-hidden={label ? undefined : 'true'}
  role={label ? 'img' : undefined}
  aria-label={label}
>
  <use href="#i-{name}" />
</svg>
