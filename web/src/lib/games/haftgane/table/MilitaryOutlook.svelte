<script lang="ts">
  // The military outlook against one neighbour: their shields, and beside
  // them the token the viewer would take against them if the Age ended now,
  // as the view states it (+1, +3 or +5, −1, or "=" for a tie). A sign or
  // "=" always shows, so colour is never the only cue.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    /** The view's outlook token: positive, -1, or 0 for a tie. */
    outlook: number;
    theirs: number;
    yours: number;
    name: string;
  }

  let { outlook, theirs, yours, name }: Props = $props();

  const MINUS = '−';
  const state = $derived(outlook > 0 ? 'win' : outlook < 0 ? 'lose' : 'tie');
  const token = $derived(
    outlook > 0 ? `+${formatNumber(outlook)}` : outlook < 0 ? MINUS + formatNumber(-outlook) : '='
  );
  const tip = $derived(
    state === 'win'
      ? ht('hg.screen.outlook.win', { name, you: yours, them: theirs, token: outlook })
      : state === 'lose'
        ? ht('hg.screen.outlook.lose', { name, you: yours, them: theirs })
        : ht('hg.screen.outlook.tie', { name, you: yours, them: theirs })
  );
</script>

<span
  class="outlook {state}"
  data-outlook={outlook}
  role="img"
  aria-label={tip}
  {@attach tooltip({ text: tip })}
>
  <HgGlyph name="shield" />
  <span>{formatNumber(theirs)}</span>
  <span class="token" dir="ltr">{token}</span>
</span>

<style>
  .outlook {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    height: 1.625rem;
    padding-inline: 0.35rem 0.2rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--panel);
    font-size: 0.8125rem;
    font-weight: 700;
    line-height: 1;
    white-space: nowrap;
  }
  .outlook :global(svg) {
    color: var(--cinnabar);
  }
  .token {
    display: inline-grid;
    place-items: center;
    min-width: 1.4rem;
    height: 1.15rem;
    padding-inline: 0.2rem;
    border-radius: 0.3rem;
    background: var(--panel-sunken);
  }
  .win {
    border-color: var(--afford-ok-edge);
  }
  .win .token {
    background: var(--afford-ok);
    color: var(--paper-bright);
  }
  .lose {
    border-color: var(--cinnabar);
  }
  .lose .token {
    background: var(--afford-short-bg);
    color: var(--cinnabar-ink);
  }
  .tie {
    border-style: dashed;
  }
</style>
