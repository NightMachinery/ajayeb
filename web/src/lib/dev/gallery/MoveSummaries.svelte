<script lang="ts">
  // M10.S03b: the opponent move summary on hand-made histories, one button
  // per scene. Each opens the popup centred (no anchor), and any tap closes
  // it, as in a game. The histories use real card, wonder and token ids, so
  // the art and glyphs are the game's own.
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import MoveSummary from '$lib/components/game/MoveSummary.svelte';
  import { summarize } from '$lib/view/move-summary';
  import type { GameEvent, HistoryEntry } from '$lib/wire';

  type SceneKey =
    'game.gallery.summary.bot' | 'game.gallery.summary.extra' | 'game.gallery.summary.special';

  function ev(e: Partial<GameEvent> & Pick<GameEvent, 'kind'>): GameEvent {
    return { private: false, ...e };
  }

  const NAMES = ['Roshan', 'Bot (Hard)'] as const;

  const SCENES: Record<SceneKey, HistoryEntry[]> = {
    'game.gallery.summary.bot': [
      {
        ply: 21,
        seat: 1,
        events: [
          ev({ kind: 'build', seat: 1, card: 'library' }),
          ev({ kind: 'pay', seat: 1, amount: 3, toOpponent: 1 }),
          ev({ kind: 'pair', seat: 1, symbol: 'quill' })
        ]
      },
      { ply: 22, seat: 1, events: [ev({ kind: 'token', seat: 1, token: 'law' })] }
    ],
    'game.gallery.summary.extra': [
      {
        ply: 30,
        seat: 1,
        events: [
          ev({ kind: 'wonder', seat: 1, wonder: 'piraeus', card: 'baths' }),
          ev({ kind: 'pay', seat: 1, amount: 5, toOpponent: 2 }),
          ev({ kind: 'extraTurn', seat: 1 })
        ]
      },
      {
        ply: 31,
        seat: 1,
        events: [
          ev({ kind: 'build', seat: 1, card: 'barracks' }),
          ev({ kind: 'pawn', seat: 1, from: 1, to: -2 }),
          ev({ kind: 'loot', seat: 0, amount: 2, value: 2 })
        ]
      }
    ],
    'game.gallery.summary.special': [
      {
        ply: 40,
        seat: 1,
        events: [
          ev({ kind: 'wonder', seat: 1, wonder: 'statue-of-zeus', card: 'tavern' }),
          ev({ kind: 'pay', seat: 1, amount: 4, toOpponent: 0 }),
          ev({ kind: 'wonderRemoved', seat: 0, wonder: 'colossus' })
        ]
      },
      { ply: 41, seat: 1, events: [ev({ kind: 'destroy', seat: 1, card: 'clay-pool' })] },
      {
        ply: 42,
        seat: 1,
        events: [
          ev({ kind: 'discard', seat: 1, card: 'glassworks', amount: 3 }),
          ev({ kind: 'ageStart', age: 2 })
        ]
      }
    ]
  };

  const KEYS = Object.keys(SCENES) as SceneKey[];

  let shown = $state<SceneKey | null>(null);
  const summary = $derived(shown ? summarize(SCENES[shown], 1) : null);
</script>

<div class="flex flex-col gap-3" data-testid="gallery-move-summary">
  <h3 class="font-display text-lg text-fg">{t('game.gallery.summary')}</h3>
  <p class="max-w-prose text-sm text-fg-muted">{t('game.gallery.summaryIntro')}</p>
  <div class="flex flex-wrap gap-2">
    {#each KEYS as key (key)}
      <Button variant="secondary" onclick={() => (shown = key)} data-summary-scene={key}>
        {t('game.gallery.summaryOpen', { scene: t(key) })}
      </Button>
    {/each}
  </div>
</div>

<MoveSummary open={shown !== null} {summary} names={NAMES} onclose={() => (shown = null)} />
