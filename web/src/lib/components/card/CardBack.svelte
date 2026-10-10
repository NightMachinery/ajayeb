<script lang="ts">
  // The back of a face-down card: Age I, II or III, or the guild back. A
  // procedural lattice in the back's own palette inside a gold frame, with a
  // cartouche that carries the age numeral as SVG text (never baked into
  // art, so it stays crisp and translatable). The guild back carries a crown
  // over the numeral and a checker field, so it differs from Age III by
  // shape as well as colour. Fills its container; decorative unless the
  // parent labels it.
  import { t } from '$lib/i18n';
  import type { Back } from '$lib/wire';
  import { backAge } from './card-style';

  interface Props {
    back: Back;
    class?: string;
  }

  let { back, class: className = '' }: Props = $props();

  const numeral = $derived(t(`card.ageNumeral.${backAge(back)}`));
</script>

<span
  class="card-back back-{back} relative flex items-center justify-center overflow-hidden rounded-[6%/4%] {className}"
  data-back={back}
  aria-hidden="true"
>
  <svg class="relative w-[64%]" viewBox="0 0 100 64" aria-hidden="true">
    <path
      d="M50 3 C64 3 70 10 86 10 C95 10 97 20 97 32 C97 44 95 54 86 54 C70 54 64 61 50 61 C36 61 30 54 14 54 C5 54 3 44 3 32 C3 20 5 10 14 10 C30 10 36 3 50 3 Z"
      fill="var(--paper)"
      stroke="var(--gold)"
      stroke-width="3"
    />
    {#if back === 'guild'}
      <path
        d="M30 25 L36 13 L43 22 L50 9 L57 22 L64 13 L70 25 Z"
        fill="var(--back-base)"
        stroke="var(--gold-deep)"
        stroke-width="1.5"
        stroke-linejoin="round"
      />
    {/if}
    <text
      x="50"
      y={back === 'guild' ? 49 : 43}
      text-anchor="middle"
      font-family="Fraunces, ui-serif, Georgia, serif"
      font-weight="600"
      font-size={back === 'guild' ? 22 : 30}
      letter-spacing="1"
      fill="var(--back-base)">{numeral}</text
    >
  </svg>
</span>

<style>
  .card-back {
    --back-line: oklch(1 0 0 / 0.16);
    background:
      radial-gradient(circle, var(--gold) 0 1.2px, transparent 1.7px) 0 0 / 12px 12px,
      repeating-linear-gradient(45deg, var(--back-line) 0 1px, transparent 1px 8.5px),
      repeating-linear-gradient(-45deg, var(--back-line) 0 1px, transparent 1px 8.5px),
      radial-gradient(circle at 50% 50%, var(--back-glow), var(--back-base) 75%);
    box-shadow:
      inset 0 0 0 3px var(--back-base),
      inset 0 0 0 4.5px var(--gold),
      inset 0 0 0 7px var(--back-base),
      inset 0 0 0 8px color-mix(in oklch, var(--gold) 60%, transparent);
  }
  .back-age1 {
    --back-base: oklch(0.5 0.11 55);
    --back-glow: oklch(0.62 0.12 65);
  }
  .back-age2 {
    --back-base: oklch(0.42 0.1 235);
    --back-glow: oklch(0.54 0.1 215);
  }
  .back-age3 {
    --back-base: oklch(0.42 0.13 20);
    --back-glow: oklch(0.53 0.14 30);
  }
  .back-guild {
    --back-base: oklch(0.36 0.12 315);
    --back-glow: oklch(0.46 0.13 315);
    background:
      repeating-conic-gradient(oklch(1 0 0 / 0.1) 0 25%, transparent 0 50%) 0 0 / 12px 12px,
      radial-gradient(circle at 50% 50%, var(--back-glow), var(--back-base) 75%);
  }
</style>
