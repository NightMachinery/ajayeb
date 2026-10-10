<script lang="ts">
  // Fetches the next age's approved art while the current age is played
  // (prefetch.ts has the plan and the budget). It renders nothing. Mount it
  // once on the game screen with the shown view's age; with no art approved
  // the plan is empty and it does nothing at all.
  import { prefetchImages, prefetchPlan } from './prefetch';

  interface Props {
    /** The shown view's age (0 during the wonder draft), or null with no view. */
    age: number | null;
  }

  let { age }: Props = $props();

  const plan = $derived(age === null ? [] : prefetchPlan(age));

  // A side effect only (network requests); it writes no state.
  $effect(() => {
    if (plan.length === 0) return;
    return prefetchImages(plan);
  });
</script>
