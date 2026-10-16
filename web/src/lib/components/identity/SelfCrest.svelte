<script lang="ts">
  // A seat's crest (the Sun or the Moon), with an accent ring when the seat
  // is the player's own. The ring goes with the "you" mark (SelfMark), never
  // alone. A leaf: plain props. Its tooltip names the crest (and the seat as
  // the player's own).
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    seat: 0 | 1;
    /** The seat is the player's own: the ring shows. */
    self?: boolean;
    /** The icon's classes (its size); the crest colour is added here. */
    class?: string;
  }

  let { seat, self = false, class: className = '' }: Props = $props();

  const crest = $derived(t(seat === 0 ? 'board.crest.sun' : 'board.crest.moon'));
</script>

<span
  class="grid shrink-0 place-items-center rounded-full {self
    ? 'ring-2 ring-accent ring-offset-1 ring-offset-panel'
    : ''}"
  data-self-crest={self ? '' : undefined}
  {@attach tooltip({
    text: self ? t('board.crest.tipSelf', { crest }) : t('board.crest.tip', { crest })
  })}
>
  <Icon
    name={seat === 0 ? 'crest-sun' : 'crest-moon'}
    class="{className} {seat === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
  />
</span>
