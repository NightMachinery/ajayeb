<script lang="ts">
  interface Props {
    heading: string;
    body: string;
    label: string;
    ondismiss: () => void;
  }

  let { heading, body, label, ondismiss }: Props = $props();
</script>

<!-- The whole notice is one button, so its text and background also dismiss it. -->
<button
  type="button"
  class="grid h-full w-full place-items-center p-6 text-center text-fg"
  data-act="small-window-dismiss"
  onclick={ondismiss}
>
  <span class="flex max-w-xs flex-col items-center gap-2">
    <span class="font-semibold">{heading}</span>
    <span class="text-sm text-fg-muted">{body}</span>
    <span class="mt-2 rounded-lg bg-accent px-4 py-3 font-semibold text-on-accent">
      {label}
    </span>
  </span>
</button>
