<script lang="ts">
  // Render the original shapes directly. A sprite and SVG use shadow trees
  // made every native modal restyle hundreds of unnecessary elements.
  // Decorative by default; label, when supplied, is already translated.
  //
  // Sized in em, so the surrounding font size sizes it. The defaults (1em
  // square, inline-block, unshrinking, a little below the baseline) are the
  // `svg.icon[data-icon]` rule in app.css's `components` layer: above
  // Tailwind's preflight, under its utilities, so any class a caller passes
  // (size-3, size-[1.1em], size-full, block) wins. As utilities in this
  // class list, size-[1em] sorted after size-3 and silently beat it. The
  // rule is not in a <style> block here, because a component's CSS chunk
  // can be linked before app.css's, and the first stylesheet to name a
  // cascade layer fixes the layer order (lint-ui.test.ts guards this).
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
