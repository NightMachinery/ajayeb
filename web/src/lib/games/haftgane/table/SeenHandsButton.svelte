<script lang="ts">
  // The hand header's button that opens the Seen hands notebook.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    onopen: () => void;
  }

  let { onopen }: Props = $props();
</script>

<button
  type="button"
  class="seen"
  data-act="seen-hands"
  onclick={onopen}
  {@attach tooltip({ text: ht('hg.seen.open.tip'), actionable: true })}
>
  <HgGlyph name="pass-hand" />
  <span>{ht('hg.seen.open')}</span>
</button>

<style>
  .seen {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    min-width: 44px;
    min-height: 44px;
    margin-block: -0.35rem;
    padding-inline: 0.5rem;
    border: 0;
    border-radius: var(--shape-control);
    background: none;
    color: var(--lapis);
    font: inherit;
    font-size: 0.875rem;
    font-weight: 600;
    white-space: nowrap;
    cursor: pointer;
  }
  .seen:hover {
    background: var(--panel-sunken);
  }
</style>
