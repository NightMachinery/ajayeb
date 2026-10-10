<script lang="ts">
  // A visually hidden aria-live region: mounted once, fed a new `message` to
  // have a screen reader announce it (the TurnBar's event summaries, per the
  // UX design §5). `assertive` is for something that must interrupt, such as
  // a disconnect; the default `polite` waits its turn.
  interface Props {
    message?: string;
    assertive?: boolean;
  }

  let { message = '', assertive = false }: Props = $props();
</script>

<div class="sr-only" aria-live={assertive ? 'assertive' : 'polite'} aria-atomic="true">
  {message}
</div>
