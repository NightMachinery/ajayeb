<script lang="ts">
  // One line of the opponent move summary (MoveSummary.svelte): a move with
  // the art the board uses for it (the card's art under its colour band, the
  // wonder's art, the token's medallion, an age's card back), its text, and
  // a row of chips for what it brought: coins paid (and the trade part that
  // went to the other player), other coin changes, the conflict pawn's move,
  // looting and a science pair. Every chip and picture has its own tooltip
  // (fine pointers; on touch any tap closes the popup instead).
  //
  // A plain leaf: it shows the line it is given (lib/view/move-summary.ts,
  // numbers from the server's events) and the seat names, and decides
  // nothing.
  import Ban from '@lucide/svelte/icons/ban';
  import Flag from '@lucide/svelte/icons/flag';
  import Repeat from '@lucide/svelte/icons/repeat';
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { artId } from '$lib/art/lookup';
  import { cardBySlug } from '$lib/catalog';
  import Icon from '$lib/icons/Icon.svelte';
  import { formatSigned, t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { formatLogEvent } from '$lib/view/log-format';
  import type { SummaryLine } from '$lib/view/move-summary';
  import type { Back } from '$lib/wire';
  import CardArt from '../card/CardArt.svelte';
  import CardBack from '../card/CardBack.svelte';
  import { bandBackground } from '../card/card-style';
  import EffectGlyphs from '../card/EffectGlyphs.svelte';
  import { cardName, scienceName, wonderName } from '../card/effect-text';
  import ProgressToken from '../token/ProgressToken.svelte';
  import { tokenName } from './names';

  interface Props {
    line: SummaryLine;
    /** The seat whose moves the summary shows. */
    seat: 0 | 1;
    names: readonly [string, string];
  }

  let { line, seat, names }: Props = $props();

  const other = $derived<0 | 1>(seat === 0 ? 1 : 0);
  const card = $derived(line.card ? cardBySlug(line.card) : undefined);

  const text = $derived.by((): string => {
    const c = line.card ? cardName(line.card) : '';
    const w = line.wonder ? wonderName(line.wonder) : '';
    switch (line.kind) {
      case 'build':
        return t('game.summary.build', { card: c });
      case 'discard':
        return t('game.summary.discard', { card: c, count: line.amount ?? 0 });
      case 'wonder':
        return line.card
          ? t('game.summary.wonder', { wonder: w, card: c })
          : t('game.summary.wonderPlain', { wonder: w });
      case 'draft':
        return line.auto && line.seat !== null && line.seat !== seat
          ? t('game.summary.draftAuto', { wonder: w, name: names[line.seat] })
          : t('game.summary.draft', { wonder: w });
      case 'token': {
        const token = tokenName(line.token ?? '');
        return line.fromLibrary
          ? t('game.summary.tokenLibrary', { token })
          : t('game.summary.token', { token });
      }
      case 'destroy':
        return t('game.summary.destroy', { name: names[other], card: c });
      case 'rebuild':
        return t('game.summary.rebuild', { card: c });
      case 'starterChosen':
        return t('game.summary.starter', { name: names[line.starter ?? seat] });
      case 'resign':
        return t('game.summary.resign');
      case 'ageStart':
        return t('game.summary.ageStart', { age: line.age ?? 0 });
      case 'extraTurn':
        return t('game.summary.extraTurn');
      case 'extraTurnLost':
        return t('game.summary.extraTurnLost');
      case 'wonderRemoved':
        return t('game.summary.wonderRemoved', { name: names[line.seat ?? seat], wonder: w });
      case 'skipped': {
        const event = {
          kind: 'effectSkipped' as const,
          private: false,
          seat: line.seat ?? undefined,
          wonder: line.wonder,
          card: line.card,
          skipped: line.skipped
        };
        return formatLogEvent(event, names)?.text ?? '';
      }
      case 'gameOver':
        return line.winner === null || line.winner === undefined
          ? t('choices.over.kind.shared')
          : t('game.summary.wins', {
              name: names[line.winner],
              kind: t(`choices.over.kind.${line.victory ?? 'civilian'}`)
            });
    }
    return '';
  });

  // The pawn's move, counted toward the capital it nears (positions are
  // positive toward seat 1's capital).
  const pawn = $derived.by(() => {
    const p = line.pawn;
    if (!p || p.to === p.from) return null;
    const toward: 0 | 1 = p.to > p.from ? 1 : 0;
    return { count: Math.abs(p.to - p.from), name: names[toward] };
  });

  const back = $derived<Back | null>(
    line.age === 1 ? 'age1' : line.age === 2 ? 'age2' : line.age === 3 ? 'age3' : null
  );

  const thumb =
    'relative block h-8 w-11 shrink-0 overflow-hidden rounded-sm border border-rule-strong';
  const chip =
    'inline-flex min-h-6 items-center gap-1 rounded-full border border-rule bg-panel-sunken px-1.5 text-xs text-fg';
</script>

{#snippet coins(n: number, loss: boolean)}
  <Icon name="coin" class="text-sm text-gold [--i2:var(--gold-deep)]" />
  <span class="font-semibold tabular-nums" aria-hidden="true">{formatSigned(loss ? -n : n)}</span>
{/snippet}

<li class="flex items-start gap-2.5" data-summary-line={line.kind}>
  <span class="grid w-11 shrink-0 place-items-center">
    {#if card && (line.kind === 'build' || line.kind === 'discard' || line.kind === 'destroy' || line.kind === 'rebuild')}
      <span
        class="{thumb} {line.kind === 'destroy' ? 'opacity-60 grayscale' : ''}"
        data-card={card.slug}
        {@attach tooltip({
          text: cardName(card.slug),
          stack: [{ kind: 'card', slug: card.slug, caption: cardName(card.slug) }]
        })}
      >
        <span
          class="absolute inset-x-0 top-0 z-10 h-1.5"
          style:background={bandBackground(card.color)}
        ></span>
        <CardArt slug={card.slug} color={card.color} tier="medium" class="h-full w-full" />
      </span>
    {:else if line.wonder && (line.kind === 'wonder' || line.kind === 'draft' || line.kind === 'wonderRemoved')}
      <span
        class="{thumb} bg-card-paper {line.kind === 'wonderRemoved' ? 'opacity-60 grayscale' : ''}"
        data-wonder={line.wonder}
        {@attach tooltip({
          text:
            line.kind === 'wonderRemoved'
              ? t('game.summary.wonderRemovedTip')
              : wonderName(line.wonder)
        })}
      >
        <ArtImage
          id={artId('wonder', line.wonder)}
          size="thumb"
          feather="edge"
          class="absolute inset-0 h-full w-full"
        >
          {#snippet fallback()}
            <span class="absolute inset-0 grid place-items-center">
              <Icon name="wonder" class="text-xl text-(--lapis) [--i2:var(--gold)]" />
            </span>
          {/snippet}
        </ArtImage>
      </span>
    {:else if line.kind === 'token' && line.token}
      <ProgressToken slug={line.token} owned size="sm" />
    {:else if line.kind === 'ageStart' && back}
      <span class="block h-8 w-6 shrink-0"><CardBack {back} class="h-full w-full" /></span>
    {:else if line.kind === 'starterChosen'}
      <Icon name={line.starter === 1 ? 'crest-moon' : 'crest-sun'} class="text-2xl" />
    {:else if line.kind === 'resign'}
      <Flag class="size-5 text-cinnabar-ink" aria-hidden="true" />
    {:else if line.kind === 'extraTurn' || line.kind === 'extraTurnLost'}
      <Repeat class="size-5 text-fg-muted" aria-hidden="true" />
    {:else if line.kind === 'skipped'}
      <Ban class="size-5 text-fg-muted" aria-hidden="true" />
    {:else if line.kind === 'gameOver'}
      <Icon name="vp-laurel" class="text-2xl" />
    {/if}
  </span>

  <span class="flex min-w-0 flex-1 flex-col gap-1">
    <span class="flex min-w-0 flex-wrap items-center gap-x-1.5 text-sm leading-snug text-fg">
      <span data-summary-text>{text}</span>
      {#if card && (line.kind === 'build' || line.kind === 'rebuild')}
        <EffectGlyphs effects={card.effects} small class="text-xs" />
      {/if}
    </span>

    {#if line.paid > 0 || line.coins[seat] !== 0 || line.coins[other] !== 0 || pawn || line.loot.length > 0 || line.pairs.length > 0}
      <span class="flex flex-wrap items-center gap-1" data-summary-chips>
        {#if line.paid > 0}
          {@const paid = t('game.summary.paid', { count: line.paid })}
          <span class={chip} data-chip="paid" {@attach tooltip({ text: paid })}>
            {@render coins(line.paid, true)}
            <span class="sr-only">{paid}</span>
          </span>
          {#if line.toOpponent > 0}
            {@const trade = t('game.summary.trade', { count: line.toOpponent, name: names[other] })}
            <span class={chip} data-chip="trade" {@attach tooltip({ text: trade })}>
              <Icon name="trade" class="text-sm" />
              {@render coins(line.toOpponent, false)}
              <span class="max-w-24 truncate" dir="auto">{names[other]}</span>
              <span class="sr-only">{trade}</span>
            </span>
          {/if}
        {/if}
        {#each [seat, other] as s (s)}
          {@const delta = line.coins[s]}
          {#if delta !== 0}
            {@const coinText =
              delta > 0
                ? t('game.summary.gains', { name: names[s], count: delta })
                : t('game.summary.loses', { name: names[s], count: -delta })}
            <span class={chip} data-chip="coins" {@attach tooltip({ text: coinText })}>
              {#if s !== seat}<span class="max-w-24 truncate" dir="auto">{names[s]}</span>{/if}
              {@render coins(Math.abs(delta), delta < 0)}
              <span class="sr-only">{coinText}</span>
            </span>
          {/if}
        {/each}
        {#if pawn}
          {@const pawnText = t('game.summary.pawn', pawn)}
          <span class={chip} data-chip="pawn" {@attach tooltip({ text: pawnText })}>
            <Icon name="military-pawn" class="text-sm" />
            <span aria-hidden="true">{t('game.summary.pawnShort', pawn)}</span>
            <span class="sr-only">{pawnText}</span>
          </span>
        {/if}
        {#each line.loot as loot, i (i)}
          {@const lootText = t('game.summary.loot', { name: names[loot.seat], count: loot.amount })}
          <span class={chip} data-chip="loot" {@attach tooltip({ text: lootText })}>
            <Icon name={loot.value >= 5 ? 'looting-5' : 'looting-2'} class="text-sm" />
            <span class="max-w-24 truncate" dir="auto">{names[loot.seat]}</span>
            {@render coins(loot.amount, true)}
            <span class="sr-only">{lootText}</span>
          </span>
        {/each}
        {#each line.pairs as symbol, i (i)}
          {@const pairText = t('game.summary.pair', { symbol: scienceName(symbol) })}
          <span class={chip} data-chip="pair" {@attach tooltip({ text: pairText })}>
            <Icon name={`science-${symbol}`} class="text-sm" />
            <span aria-hidden="true">{t('game.summary.pairShort')}</span>
            <span class="sr-only">{pairText}</span>
          </span>
        {/each}
      </span>
    {/if}
  </span>
</li>
