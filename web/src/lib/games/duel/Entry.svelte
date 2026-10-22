<script lang="ts">
  import Room from '../../../routes/play/[code]/Room.svelte';
  import type { RoomBootstrap } from '$lib/entry/bootstrap';
  interface Props {
    connection: RoomBootstrap | null;
    snapshot: unknown;
  }
  let { connection, snapshot }: Props = $props();
</script>

<!-- Discovery data must be discarded before the unchanged Duel entry mounts. -->
{#if connection === null && snapshot === undefined}
  <Room />
{/if}
