<script lang="ts">
  // Render the original shapes directly. A sprite and SVG use shadow trees
  // made every native modal restyle hundreds of unnecessary elements.
  // Decorative by default; label, when supplied, is already translated.
  //
  // Sized in em, so the surrounding font size sizes it. The defaults (1em
  // square, inline, unshrinking, a little below the baseline) live in the
  // style block's `components` layer, under Tailwind's utilities, so any
  // class a caller passes (size-3, size-[1.1em], size-full, block) wins.
  // As a utility in the class list, size-[1em] sorted after size-3 and
  // silently beat it.
  import { iconMarkup, iconSources } from './sources';
  import type { IconName } from './names';

  interface Props {
    name: IconName;
    label?: string;
    class?: string;
  }

  let { name, label, class: className = '' }: Props = $props();
  const uid = $props.id();
  const source = $derived(iconSources[name]);
  const markup = $derived(iconMarkup(name, uid));
</script>

<svg
  viewBox={source.viewBox}
  data-icon={name}
  class="icon {className}"
  aria-hidden={label ? undefined : 'true'}
  role={label ? 'img' : undefined}
  aria-label={label}
>
  <!-- Only our build-time vector sources enter markup (sources.ts). -->
  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html markup}
</svg>

<style>
  @layer components {
    .icon {
      display: inline-block;
      width: 1em;
      height: 1em;
      flex-shrink: 0;
      vertical-align: -0.125em;
    }
  }
</style>
