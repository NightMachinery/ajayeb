<script lang="ts">
  // Empty entry only. No game UI, rules, offers, actions or event playback.
  import { onMount } from 'svelte';
  import type { EntryProps } from '$lib/entry/types';
  let { connection }: EntryProps = $props();
  onMount(() => connection?.resync());
</script>
