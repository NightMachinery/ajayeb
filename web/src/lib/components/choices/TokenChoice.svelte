<script lang="ts">
  // A progress token to take: from the board after a science pair, or from
  // the three the Great Library drew (which only the chooser sees). Each
  // token is a row with its medallion, name and effect; select one, then
  // "Take <token>" sends its key from the view.
  //
  // Hints (M10.HS3): under each token, its outcome line (the server's
  // Choice.outcome for that key: points, coins, a win), from the page's
  // game context.
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import { sessionHintScreen } from '$lib/view/hints';
  import { outcomeContext, outcomeTerms, type OutcomeTerm } from '$lib/view/outcome';
  import OutcomeLine from '../game/OutcomeLine.svelte';
  import { tryGameContext } from './room';
  import Icon from '$lib/icons/Icon.svelte';
  import ProgressToken from '../token/ProgressToken.svelte';
  import CommitButton from './CommitButton.svelte';
  import { tokenEffect, tokenName } from './token-text';

  interface Props {
    /** Token slug to its key, from the view's choices, in the order shown. */
    keys: ReadonlyMap<string, string>;
    /** The Great Library's private offer rather than the board. */
    library?: boolean;
    canAct: boolean;
    /** Ask for a second tap before sending (the act-button rule). */
    confirm?: boolean;
    /** The connection is up. */
    connected?: boolean;
    onact: (key: string) => void;
  }

  let { keys, library = false, canAct, confirm = false, connected = true, onact }: Props = $props();

  let picked = $state<string | null>(null);
  const tokens = $derived([...keys.keys()]);
  const only = $derived(keys.size === 1 ? ([...keys.values()][0] ?? null) : null);
  const selected = $derived(picked !== null && [...keys.values()].includes(picked) ? picked : only);
  const ctx = tryGameContext();
  const screen = $derived(sessionHintScreen(ctx?.session, hintHide.value));
  const termsByKey = $derived.by((): Readonly<Record<string, OutcomeTerm[]>> => {
    const view = ctx?.session.view;
    if (!view || !screen) return {};
    const octx = outcomeContext(view, screen.families, screen.names);
    return Object.fromEntries(
      view.choices
        .filter((c) => c.kind === 'token' && c.outcome)
        .map((c) => [c.key, outcomeTerms(c.outcome, octx)])
    );
  });
  const selectedToken = $derived(
    selected === null ? null : (tokens.find((slug) => keys.get(slug) === selected) ?? null)
  );
</script>

<div class="flex flex-col gap-3" data-token-choice={library ? 'library' : 'board'}>
  <p class="text-sm text-fg">
    {library ? t('choices.library.intro') : t('choices.token.intro')}
  </p>
  {#if library}
    <p class="flex items-center gap-1.5 text-xs text-fg-muted">
      <Icon name="badge-lock" class="text-[14px]" />
      {t('choices.library.secret')}
    </p>
  {/if}
  <ul class="flex flex-col gap-2" aria-label={t('choices.token.list')}>
    {#each tokens as slug (slug)}
      {@const key = keys.get(slug) ?? ''}
      {@const isSelected = key === selected}
      <li>
        <button
          type="button"
          class="flex w-full items-center gap-3 rounded-control border-2 bg-panel px-3 py-2 text-start
            transition-colors duration-(--dur-fast) hover:bg-panel-sunken disabled:cursor-not-allowed disabled:opacity-60
            {isSelected ? 'border-accent bg-accent-soft' : 'border-rule'}"
          data-pick={key}
          data-token={slug}
          aria-pressed={isSelected}
          disabled={!canAct}
          onclick={() => (picked = key)}
        >
          <ProgressToken {slug} class="shrink-0" />
          <span class="flex min-w-0 flex-1 flex-col">
            <span class="font-display text-[15px] text-fg">{tokenName(slug)}</span>
            <span class="text-[13px] leading-snug text-fg-muted">{tokenEffect(slug)}</span>
          </span>
          {#if isSelected}
            <Icon name="badge-check" class="shrink-0 text-xl text-accent [--i2:var(--panel)]" />
          {/if}
        </button>
        <OutcomeLine terms={termsByKey[key] ?? []} actKey={key} class="mt-1 ps-3" />
      </li>
    {/each}
  </ul>
  <CommitButton
    label={selectedToken
      ? t('choices.token.confirm', { token: tokenName(selectedToken) })
      : t('choices.selectToken')}
    actKey={selected}
    disabled={!canAct}
    {confirm}
    {connected}
    {onact}
  />
</div>
