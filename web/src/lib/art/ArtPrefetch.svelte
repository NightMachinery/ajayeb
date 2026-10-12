<script lang="ts">
  // Fetches the whole game's approved art while the browser is idle,
  // soonest needed first (prefetch.ts has the plan, and the small plan for
  // Save-Data and 2G). It renders nothing. Mount it once on the game screen
  // with the shown view's age, and nothing else: the plan depends on the
  // age alone, never on hidden state. Unmounting it (leaving the game
  // screen) stops it; with no art approved it does nothing at all.
  import { artPrefetchPlan, browserConnection, prefetchImages } from './prefetch';

  interface Props {
    /** The shown view's age (0 during the wonder draft), or null with no view. */
    age: number | null;
  }

  let { age }: Props = $props();

  const plan = $derived(
    age === null ? [] : artPrefetchPlan(age, { connection: browserConnection() })
  );

  // A side effect only (network requests); it writes no state.
  $effect(() => {
    if (plan.length === 0) return;
    return prefetchImages(plan);
  });
</script>
