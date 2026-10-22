<script lang="ts">
  // Empty entry only. No game UI, rules, offers, actions or event playback.
  import { onMount } from 'svelte';
  import type { RoomBootstrap } from '$lib/entry/bootstrap';
  interface Props {
    connection: RoomBootstrap | null;
    snapshot: unknown;
  }
  let { connection }: Props = $props();
  onMount(() => connection?.resync());
</script>
