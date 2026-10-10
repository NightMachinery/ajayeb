<script lang="ts">
  // A bot's avatar: the persona's approved portrait (avatar/apprentice,
  // avatar/scholar or avatar/master in the generated art index) in a round
  // gold-ringed frame, or, until it is approved or if the file fails to
  // load, a placeholder medallion with the persona's initial. Decorative:
  // the persona's name is always shown beside it, so the image is hidden
  // from assistive technology. Size it with the class (it fills its box).
  import type { BotLevel } from '$lib/wire';
  import { personaName } from '$lib/components/game/names';
  import ArtImage from './ArtImage.svelte';
  import { avatarArtId } from './lookup';

  interface Props {
    level: BotLevel;
    class?: string;
  }

  let { level, class: className = 'size-10' }: Props = $props();

  const initial = $derived(personaName(level).charAt(0).toUpperCase());
</script>

<span
  class="bot-avatar [container-type:size] relative inline-grid shrink-0 place-items-center overflow-hidden rounded-full border-2 border-(color:--gold-deep) bg-card-paper text-card-ink {className}"
  data-avatar={level}
  aria-hidden="true"
>
  <ArtImage id={avatarArtId(level)} feather="none" class="absolute inset-0 h-full w-full">
    {#snippet fallback()}
      <span class="font-display text-[length:52cqh] leading-none font-semibold text-(--lapis)"
        >{initial}</span
      >
    {/snippet}
  </ArtImage>
</span>
