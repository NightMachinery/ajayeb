<script lang="ts">
  // One pill per stage of a wonder side: its numeral, then a tick when
  // built or its printed cost. The next stage is outlined when the view
  // offers a stage choice this turn. The stages come from the catalog; the
  // offer from the server's choices.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { HgStage } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import ResourceTile from './ResourceTile.svelte';

  interface Props {
    stages: readonly HgStage[];
    built: number;
    /** The next stage may be built this turn. */
    open?: boolean;
  }

  let { stages, built, open = false }: Props = $props();

  const numeral = (i: number) =>
    i === 0
      ? ht('hg.screen.stageNumeral.1')
      : i === 1
        ? ht('hg.screen.stageNumeral.2')
        : i === 2
          ? ht('hg.screen.stageNumeral.3')
          : ht('hg.screen.stageNumeral.4');

  function grouped(res: readonly string[] | undefined): { res: string; n: number }[] {
    const out: { res: string; n: number }[] = [];
    for (const r of res ?? []) {
      const last = out.at(-1);
      if (last?.res === r) last.n += 1;
      else out.push({ res: r, n: 1 });
    }
    return out;
  }
</script>

<span class="pills" data-stage-pills>
  {#each stages as st, i (i)}
    {@const state = i < built ? 'built' : i === built && open ? 'open' : 'later'}
    {@const tip = ht('hg.screen.stage.tip', { stage: numeral(i) })}
    {@const line =
      state === 'built'
        ? ht('hg.screen.stage.built')
        : state === 'open'
          ? ht('hg.screen.stage.open')
          : ht('hg.screen.stage.later')}
    <span
      class="pill {state}"
      data-stage={i + 1}
      data-stage-state={state}
      role="img"
      aria-label="{tip} {line}"
      {@attach tooltip({ text: tip, lines: [line] })}
    >
      <span class="numeral">{numeral(i)}</span>
      {#if state === 'built'}
        <HgGlyph name="badge-check" />
      {:else}
        {#each grouped(st.cost.res) as g (g.res)}
          <span class="cost"
            ><ResourceTile res={g.res} size="xs" />{#if g.n > 1}<span>{formatNumber(g.n)}</span
              >{/if}</span
          >
        {/each}
      {/if}
    </span>
  {/each}
</span>

<style>
  .pills {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.25rem;
  }
  .pill {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    height: 1.75rem;
    padding-inline: 0.4rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: 0.5rem;
    background: var(--panel);
    font-size: 0.8125rem;
    font-weight: 700;
    line-height: 1;
  }
  .numeral {
    font-family: var(--type-display);
    color: var(--fg-muted);
  }
  .built {
    background: var(--panel-sunken);
  }
  .built :global(svg) {
    color: var(--afford-ok);
  }
  .open {
    border: 2.5px dashed var(--lapis);
  }
  .cost {
    display: inline-flex;
    align-items: center;
    gap: 0.05rem;
  }
</style>
