<script lang="ts">
  // The back of a face-down card: Age I, II or III, or the guild back. A
  // procedural lattice in the back's own palette inside a gold frame, with a
  // cartouche that carries the age numeral as SVG text (never baked into
  // art, so it stays crisp and translatable). The guild back carries a crown
  // over the numeral and a checker field, so it differs from Age III by
  // shape as well as colour. Fills its container; decorative unless the
  // parent labels it. Its tooltip says which back it is (view/tips.ts
  // backTip), then says when it turns face up. Hidden candidates and costs
  // never belong in this tooltip.
  //
  // Once a back's bitmap art is approved (back/age1 and so on in the
  // generated art index), it fills the card under the same gold frame and
  // cartouche; until then, or if the file fails to load, the procedural
  // lattice shows.
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { backArtId } from '$lib/art/lookup';
  import { i18n, t } from '$lib/i18n';
  import { tooltip, type TooltipContent } from '$lib/ui/tooltip.svelte';
  import { backTip, backText } from '$lib/view/tips';
  import type { Back } from '$lib/wire';
  import { backAge } from './card-style';

  interface Props {
    back: Back;
    /** The tooltip; by default it names the back. */
    tip?: TooltipContent;
    /** A passive standalone back gets one keyboard stop. */
    focusable?: boolean;
    class?: string;
  }

  let { back, tip, focusable = false, class: className = '' }: Props = $props();

  const numeral = $derived(t(`card.ageNumeral.${backAge(back)}`));
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex (A passive card back has one keyboard stop for its explanation.) -->
<span
  class="card-back back-{back} relative flex items-center justify-center overflow-hidden rounded-[6%/4%] {className}"
  data-back={back}
  aria-hidden={focusable ? undefined : 'true'}
  role={focusable ? 'img' : undefined}
  aria-label={focusable ? backText(back) : undefined}
  tabindex={focusable ? 0 : undefined}
  {@attach tooltip({ ...(tip ?? backTip(back)), avoidAnchor: true })}
>
  <ArtImage id={backArtId(back)} feather="none" class="absolute inset-0 h-full w-full" />
  <!-- The frame sits over the art (an inset shadow alone would sit under it). -->
  <span class="back-frame pointer-events-none absolute inset-0 rounded-[inherit]"></span>
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
      direction={i18n.dir}
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
  /* Colours: the --back-* tokens (styles/tokens.css), fixed like the cards. */
  .card-back {
    background:
      radial-gradient(circle, var(--gold) 0 1.2px, transparent 1.7px) 0 0 / 12px 12px,
      repeating-linear-gradient(45deg, var(--back-line) 0 1px, transparent 1px 8.5px),
      repeating-linear-gradient(-45deg, var(--back-line) 0 1px, transparent 1px 8.5px),
      radial-gradient(circle at 50% 50%, var(--back-glow), var(--back-base) 75%);
  }
  .back-frame {
    box-shadow:
      inset 0 0 0 3px var(--back-base),
      inset 0 0 0 4.5px var(--gold),
      inset 0 0 0 7px var(--back-base),
      inset 0 0 0 8px color-mix(in oklch, var(--gold) 60%, transparent);
  }
  .back-age1 {
    --back-base: var(--back-age1-base);
    --back-glow: var(--back-age1-glow);
  }
  .back-age2 {
    --back-base: var(--back-age2-base);
    --back-glow: var(--back-age2-glow);
  }
  .back-age3 {
    --back-base: var(--back-age3-base);
    --back-glow: var(--back-age3-glow);
  }
  .back-guild {
    --back-base: var(--back-guild-base);
    --back-glow: var(--back-guild-glow);
    background:
      repeating-conic-gradient(var(--back-check) 0 25%, transparent 0 50%) 0 0 / 12px 12px,
      radial-gradient(circle at 50% 50%, var(--back-glow), var(--back-base) 75%);
  }
</style>
