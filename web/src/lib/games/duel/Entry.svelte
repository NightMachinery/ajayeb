<script lang="ts">
  import Room from '../../../routes/play/[code]/Room.svelte';
  import type { EntryProps } from '$lib/entry/types';
  let { onwronggame, onstate }: EntryProps = $props();
</script>

<Room {onwronggame} {onstate} />
