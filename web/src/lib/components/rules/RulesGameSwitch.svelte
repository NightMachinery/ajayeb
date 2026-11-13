<script lang="ts">
  // The two rules pages, /rules (Duel) and /rules/haftgane, each name the
  // other here, so a reader who lands on the wrong game's rules can switch.
  // It imports nothing from $lib/games/haftgane, so the Duel page and Home
  // stay free of Haftgane's code (src/platform-imports.test.ts).
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import { t as pickerT } from '$lib/lobby/i18n/picker';

  let { current }: { current: 'duel' | 'haftgane' } = $props();

  const games = [
    { id: 'duel', name: () => pickerT('lobby.duel') },
    { id: 'haftgane', name: () => pickerT('lobby.haftgane') }
  ] as const;
</script>

<nav aria-label={t('rules.chooseGame')} class="flex flex-wrap gap-2" data-rules-game-switch>
  {#each games as game (game.id)}
    <a
      href={game.id === 'duel' ? resolve('/rules') : resolve('/rules/haftgane')}
      aria-current={game.id === current ? 'page' : undefined}
      class="inline-flex min-h-11 items-center rounded-control border px-4 text-sm font-semibold"
      class:border-accent={game.id === current}
      class:bg-accent-soft={game.id === current}
      class:text-fg={game.id === current}
      class:border-rule={game.id !== current}
      class:bg-panel={game.id !== current}
      class:text-fg-muted={game.id !== current}
      data-control="rules-game-{game.id}"
    >
      {game.name()}
    </a>
  {/each}
</nav>
