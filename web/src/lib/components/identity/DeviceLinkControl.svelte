<script lang="ts">
  // Shared device-link interaction. Callers supply permissions and the request;
  // this component owns the sheet, clipboard fallback and reset feedback.
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import { copyText } from '$lib/ui/clipboard';
  import DeviceLinkButton from './DeviceLinkButton.svelte';
  import DeviceLinkSheet from './DeviceLinkSheet.svelte';
  interface Props {
    open?: boolean;
    trigger?: boolean;
    autoOpen?: boolean;
    code: string;
    seat: number | string;
    userId?: string;
    name: string;
    own: boolean;
    description?: string;
    resetHint?: string;
    request: (reset: boolean) => Promise<string>;
    class?: string;
  }
  let {
    open = $bindable(false),
    trigger = true,
    autoOpen = false,
    code,
    seat,
    userId,
    name,
    own,
    description,
    resetHint,
    request,
    class: className = ''
  }: Props = $props();
  let link = $state<string | null>(null);
  let status = $state<'loading' | 'ready' | 'failed'>('loading');
  let copied = $state<'idle' | 'copied' | 'failed'>('idle');
  let wasReset = $state(false);

  onMount(() => {
    if (autoOpen) void openSheet();
  });

  function toLink(code: string, claim: string): string {
    // Keep this lazy interaction from adding exports to the cold identity chunk.
    return new URL(`${resolve('/play/[code]', { code })}#claim=${claim}`, location.origin).href;
  }

  async function fetchLink(reset: boolean): Promise<boolean> {
    if (!code) return false;
    status = 'loading';
    copied = 'idle';
    wasReset = false;
    try {
      const claim = await request(reset);
      link = toLink(code, claim);
      status = 'ready';
      wasReset = reset;
      return true;
    } catch {
      status = 'failed';
      return false;
    }
  }

  async function copy(): Promise<void> {
    if (!link) return;
    copied = (await copyText(link)) ? 'copied' : 'failed';
  }

  async function openSheet(): Promise<void> {
    open = true;
    link = null;
    if (await fetchLink(false)) await copy();
  }

  async function reset(): Promise<void> {
    if (await fetchLink(true)) await copy();
  }
</script>

{#if trigger}
  <DeviceLinkButton
    {seat}
    {userId}
    class={className}
    label={own ? t('online.link.buttonOwn') : t('online.link.button', { name })}
    onclick={openSheet}
  />
{/if}
<DeviceLinkSheet
  bind:open
  {seat}
  {name}
  {own}
  {link}
  {status}
  copy={copied}
  reset={wasReset}
  {description}
  {resetHint}
  oncopy={copy}
  onreset={reset}
  onretry={() => void fetchLink(false)}
/>
