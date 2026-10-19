<script lang="ts">
  // Render the original shapes directly. A sprite and SVG use shadow trees
  // made every native modal restyle hundreds of unnecessary elements.
  // Decorative by default; label, when supplied, is already translated.
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
  class="inline-block size-[1em] shrink-0 align-[-0.125em] {className}"
  aria-hidden={label ? undefined : 'true'}
  role={label ? 'img' : undefined}
  aria-label={label}
>
  <!-- Only our build-time vector sources enter markup (sources.ts). -->
  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html markup}
</svg>
