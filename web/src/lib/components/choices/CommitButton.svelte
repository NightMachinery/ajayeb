<script lang="ts">
  // The one button that sends a choice panel's move, pinned to the bottom of
  // the panel. It carries data-act with the selected choice's key from the
  // view (the end-to-end contract), and no key at all until something is
  // selected. The danger style is for destroying a card.
  interface Props {
    /** Already translated. */
    label: string;
    /** The selected choice's key, or null before a selection. */
    actKey: string | null;
    danger?: boolean;
    disabled?: boolean;
    onact: (key: string) => void;
  }

  let { label, actKey, danger = false, disabled = false, onact }: Props = $props();
</script>

<div class="sticky bottom-0 -mx-4 mt-1 border-t border-rule bg-panel px-4 pt-3">
  <button
    type="button"
    class="commit inline-flex h-[52px] w-full items-center justify-center gap-2 rounded-control px-5 text-[15px] font-semibold
      transition-[filter,opacity] duration-(--dur-fast) hover:brightness-105 active:brightness-95
      disabled:cursor-not-allowed disabled:opacity-50
      {danger ? 'danger' : 'bg-accent text-on-accent'}"
    data-act={actKey ?? undefined}
    disabled={disabled || actKey === null}
    onclick={() => {
      if (actKey !== null) onact(actKey);
    }}
  >
    <span class="min-w-0 truncate" dir="auto">{label}</span>
  </button>
</div>

<style>
  /* A deep cinnabar, dark enough for white text in both themes. */
  .danger {
    background: color-mix(in oklch, var(--cinnabar) 78%, black);
    color: white;
  }
</style>
