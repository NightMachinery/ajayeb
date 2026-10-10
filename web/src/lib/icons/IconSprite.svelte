<script lang="ts">
  // Inlines the whole icon sprite once. Mounted in the root layout, before
  // any <Icon> is rendered. The markup is our own build-time output (see
  // sprite.ts), never user data, so {@html} carries no injection risk.
  import { iconSpriteMarkup } from './sprite';
</script>

<!-- iconSpriteMarkup is our own build-time output (see sprite.ts), assembled
     only from the files in ./svg/*.svg; it never carries user or server
     data, so there is no injection surface here. -->
<!-- eslint-disable-next-line svelte/no-at-html-tags -->
{@html iconSpriteMarkup}
