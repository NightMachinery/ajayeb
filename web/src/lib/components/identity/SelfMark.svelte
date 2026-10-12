<script lang="ts">
  // The small "you" mark beside the player's own name (in the strips, the
  // city panel, the lobby and the log), so a player always knows which seat
  // is theirs. Text, not colour alone: the accent fill only reinforces the
  // word. A leaf with no props but a class.
  import { t } from '$lib/i18n';

  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();
</script>

<span
  class="inline-flex shrink-0 items-center rounded-full bg-accent px-1.5 text-[0.625rem] leading-4 font-semibold tracking-wide text-on-accent uppercase {className}"
  data-self-mark
>
  <span aria-hidden="true">{t('online.identity.you')}</span>
  <span class="sr-only">{t('online.identity.youLabel')}</span>
</span>
