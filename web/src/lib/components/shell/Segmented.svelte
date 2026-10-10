<script lang="ts">
  // A segmented control, built on native radio inputs (role="radiogroup" for
  // free, and arrow-key roving between options for free) but styled as one
  // pill-shaped group. Used for small exclusive choices such as "First
  // player: Random / Me / Rival".
  interface Option {
    value: string;
    label: string;
  }

  interface Props {
    name: string;
    options: Option[];
    value: string;
    legend?: string;
    class?: string;
  }

  let { name, options, value = $bindable(), legend, class: className = '' }: Props = $props();
</script>

<div
  class="inline-flex gap-1 rounded-control border border-rule bg-panel p-1 {className}"
  role="radiogroup"
  aria-label={legend}
>
  {#each options as option (option.value)}
    <label
      class="relative flex-1 cursor-pointer rounded-[calc(var(--radius-control)-4px)] px-3 py-2 text-center text-sm font-medium whitespace-nowrap
        transition-colors duration-(--dur-fast)
        {value === option.value ? 'bg-accent text-on-accent' : 'text-fg-muted hover:text-fg'}"
    >
      <input
        type="radio"
        {name}
        value={option.value}
        checked={value === option.value}
        onchange={() => (value = option.value)}
        class="sr-only"
      />
      {option.label}
    </label>
  {/each}
</div>
