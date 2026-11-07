<script lang="ts">
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { BotName } from './seat-names';
  interface Props {
    name: string;
    generated?: BotName | null;
    actionable?: boolean;
  }
  let { name, generated = null, actionable = true }: Props = $props();
</script>

<span class="identity" dir="auto" {@attach tooltip({ text: name, actionable })}>
  <span class="label">{generated?.label ?? name}</span>
  {#if generated?.suffix}<span class="suffix" data-bot-suffix>{generated.suffix}</span>{/if}
</span>

<style>
  .identity {
    display: inline-flex;
    align-items: baseline;
    gap: 0.25em;
    max-width: 100%;
    min-width: 0;
    vertical-align: bottom;
  }
  .label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .suffix {
    flex: 0 0 auto;
  }
</style>
