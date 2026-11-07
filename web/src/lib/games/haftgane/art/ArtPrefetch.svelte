<script lang="ts">
  import { browserConnection, prefetchImages } from '$lib/art/prefetch';
  import { haftganePrefetchPlan } from './prefetch';
  let { age }: { age: number } = $props();
  const plan = $derived(haftganePrefetchPlan(age, browserConnection()));
  $effect(() => prefetchImages(plan));
</script>
