<script lang="ts">
  // M5.S04b: the wonder draft, every pending-choice panel, the end-of-age
  // interstitial and the game-over screen, from real recorded views
  // (src/lib/wire/fixtures.gen.ts). Each scene opens full screen over the
  // gallery, as it would over the board; ?choices=<scene> opens one
  // straight away, for screenshots. The one state no recorded view freezes
  // (a civilian tie broken on blue points) is a real game-over view with
  // its totals evened out, the way score.test.ts builds it.
  import { page } from '$app/state';
  import { t } from '$lib/i18n';
  import {
    fixtureNames,
    fixtures,
    protocolFixtureNames,
    protocolFixtures
  } from '$lib/wire/fixtures.gen';
  import type { GameEvent, View } from '$lib/wire';
  import { finalScores, scoreRows } from '$lib/view/score';
  import ChoiceHost from '$lib/components/choices/ChoiceHost.svelte';
  import AgeInterstitial from '$lib/components/screens/AgeInterstitial.svelte';
  import Draft from '$lib/components/screens/Draft.svelte';
  import GameOver from '$lib/components/screens/GameOver.svelte';
  import ScoreTable from '$lib/components/screens/ScoreTable.svelte';
  import DiscardSheet from '$lib/components/sheets/DiscardSheet.svelte';
  import TokenSheet from '$lib/components/sheets/TokenSheet.svelte';
  import X from '@lucide/svelte/icons/x';

  function viewFixture(name: (typeof protocolFixtureNames)[number]): View {
    const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
    if (!v) throw new Error(`no protocol fixture named ${name}`);
    return v;
  }

  function frameView(name: (typeof fixtureNames)[number]): View {
    const frame = fixtures[fixtureNames.indexOf(name)];
    if (frame?.t !== 'state') throw new Error(`fixture ${name} is not a state frame`);
    return frame.view;
  }

  const names: readonly [string, string] = ['Sara', 'Omid'];

  const civilian = viewFixture('game-over-civilian-seat0');
  const tieBreakView: View = civilian.result
    ? {
        ...civilian,
        result: {
          ...civilian.result,
          winner: 1,
          scores: [
            { ...civilian.result.scores[0], blue: 9, coins: 3, total: 41 },
            { ...civilian.result.scores[1], blue: 14, green: 4, total: 41 }
          ]
        }
      }
    : civilian;

  type Kind = 'draft' | 'choice' | 'over';
  interface Scene {
    id: string;
    kind: Kind;
    label: string;
    view: View;
    /** The events of the action that produced view (frames carry them; views do not). */
    events?: GameEvent[];
    /** The confirm step of an online room. */
    confirm?: boolean;
    /** The connection is down: the commit button waits for it. */
    offline?: boolean;
  }

  /** The other seat choosing who starts, as the frame after a bot's or opponent's choice carries it. */
  function starterChosenBy(view: View, seat: number): GameEvent[] {
    return [{ kind: 'starterChosen', private: false, seat, starter: view.active }];
  }

  const ageStartView = viewFixture('starter-chosen-seat1');

  const scenes: Scene[] = [
    {
      id: 'draft-r1',
      kind: 'draft',
      label: 'Draft round 1, first pick (state-draft)',
      view: frameView('state-draft')
    },
    {
      id: 'draft-r1-pick',
      kind: 'draft',
      label: 'Draft round 1, Omid picks (draft-round-1-seat1)',
      view: viewFixture('draft-round-1-seat1')
    },
    {
      id: 'draft-r1-waiting',
      kind: 'draft',
      label: 'Draft round 1, waiting (draft-round-1-seat0)',
      view: viewFixture('draft-round-1-seat0')
    },
    {
      id: 'draft-r2',
      kind: 'draft',
      label: 'Draft round 2 (draft-round-2-seat1)',
      view: viewFixture('draft-round-2-seat1')
    },
    {
      id: 'draft-r2-spectator',
      kind: 'draft',
      label: 'Draft round 2, spectator',
      view: viewFixture('draft-round-2-spectator')
    },
    {
      id: 'destroy',
      kind: 'choice',
      label: 'Destroy a card (destroy-card-seat0)',
      view: viewFixture('destroy-card-seat0')
    },
    {
      id: 'destroy-waiting',
      kind: 'choice',
      label: 'Destroy, the other player waits (destroy-card-seat1)',
      view: viewFixture('destroy-card-seat1')
    },
    {
      id: 'destroy-confirm',
      kind: 'choice',
      label: 'Destroy a card, online: tap twice to confirm (destroy-card-seat0)',
      view: viewFixture('destroy-card-seat0'),
      confirm: true
    },
    {
      id: 'destroy-offline',
      kind: 'choice',
      label: 'Destroy a card, online, the connection down (destroy-card-seat0)',
      view: viewFixture('destroy-card-seat0'),
      confirm: true,
      offline: true
    },
    {
      id: 'discard',
      kind: 'choice',
      label: 'Build from the discard pile (build-from-discard-seat0)',
      view: viewFixture('build-from-discard-seat0')
    },
    {
      id: 'board-token',
      kind: 'choice',
      label: 'Board token (pick-board-token-seat0)',
      view: viewFixture('pick-board-token-seat0')
    },
    {
      id: 'board-token-2',
      kind: 'choice',
      label: 'Board token (state-pending-token)',
      view: frameView('state-pending-token')
    },
    {
      id: 'library',
      kind: 'choice',
      label: 'Great Library (pick-library-token-seat0)',
      view: viewFixture('pick-library-token-seat0')
    },
    {
      id: 'library-waiting',
      kind: 'choice',
      label: 'Great Library, hidden (pick-library-token-seat1)',
      view: viewFixture('pick-library-token-seat1')
    },
    {
      id: 'starter',
      kind: 'choice',
      label: 'Age end and starter choice (choose-starter-seat0)',
      view: viewFixture('choose-starter-seat0')
    },
    {
      id: 'starter-confirm',
      kind: 'choice',
      label: 'Starter choice, online: tap twice to confirm (choose-starter-seat0)',
      view: viewFixture('choose-starter-seat0'),
      confirm: true
    },
    {
      id: 'starter-offline',
      kind: 'choice',
      label: 'Starter choice, online, the connection down (choose-starter-seat0)',
      view: viewFixture('choose-starter-seat0'),
      confirm: true,
      offline: true
    },
    {
      id: 'starter-waiting',
      kind: 'choice',
      label: 'Age end, the other player waits (choose-starter-seat1)',
      view: viewFixture('choose-starter-seat1')
    },
    {
      id: 'age-start',
      kind: 'choice',
      label: 'New age, starter chosen by the other player (starter-chosen-seat1)',
      view: ageStartView,
      events: starterChosenBy(ageStartView, ageStartView.viewer === 1 ? 0 : 1)
    },
    {
      id: 'over-civilian',
      kind: 'over',
      label: 'Civilian victory (game-over-civilian-seat0)',
      view: civilian
    },
    {
      id: 'over-tiebreak',
      kind: 'over',
      label: 'Civilian victory on the blue tie-break (adjusted)',
      view: tieBreakView
    },
    {
      id: 'over-shared',
      kind: 'over',
      label: 'Shared victory (game-over-shared-seat0)',
      view: viewFixture('game-over-shared-seat0')
    },
    {
      id: 'over-science',
      kind: 'over',
      label: 'Scientific supremacy (game-over-science-seat0)',
      view: viewFixture('game-over-science-seat0')
    },
    {
      id: 'over-military',
      kind: 'over',
      label: 'Military supremacy, the loser (game-over-military-hidden-seat1)',
      view: viewFixture('game-over-military-hidden-seat1')
    }
  ];

  let active = $state<string | null>(page.url.searchParams.get('choices'));
  const scene = $derived(scenes.find((s) => s.id === active) ?? null);

  // The scene's moves go nowhere; the last key sent is shown instead.
  let lastKey = $state('');
  function onact(key: string): void {
    lastKey = key;
  }

  const starterView = viewFixture('choose-starter-seat0');
  const overView = viewFixture('game-over-science-seat0');

  let tokenOpen = $state(false);
  let discardOpen = $state(false);
</script>

<div class="flex flex-col gap-6">
  <ul class="grid gap-2 sm:grid-cols-2">
    {#each scenes as s (s.id)}
      <li>
        <button
          type="button"
          class="flex min-h-11 w-full items-center rounded-control border border-rule bg-panel px-3 py-2 text-start text-sm text-fg
            hover:bg-panel-sunken"
          data-scene={s.id}
          onclick={() => (active = s.id)}
        >
          {s.label}
        </button>
      </li>
    {/each}
  </ul>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">AgeInterstitial <!-- i18n-ok --></h3>
    <div class="max-w-md rounded-panel border border-rule bg-panel p-4">
      <AgeInterstitial
        age={1}
        {names}
        players={starterView.players}
        scores={starterView.scorePreview}
        military={starterView.military}
        you={0}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">ScoreTable <!-- i18n-ok --></h3>
    <div class="max-w-md rounded-panel border border-rule bg-panel p-4">
      <ScoreTable rows={scoreRows(finalScores(overView))} {names} winner={0} you={0} />
    </div>
  </section>

  <section class="flex flex-wrap gap-2">
    <button
      type="button"
      class="min-h-11 rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
      onclick={() => (tokenOpen = true)}
    >
      TokenSheet <!-- i18n-ok -->
    </button>
    <button
      type="button"
      class="min-h-11 rounded-control border border-rule bg-panel px-4 text-sm font-semibold text-fg"
      onclick={() => (discardOpen = true)}
    >
      DiscardSheet <!-- i18n-ok -->
    </button>
  </section>
</div>

<TokenSheet
  bind:open={tokenOpen}
  slug="theology"
  actKey="token:theology"
  canAct
  onact={(key) => (lastKey = key)}
/>
<DiscardSheet
  bind:open={discardOpen}
  cards={[
    'lumber-yard',
    'baths',
    'glassworks',
    'apothecary',
    'tavern',
    'walls',
    'library',
    'caravansery',
    'palace',
    'arsenal'
  ]}
/>

{#if scene}
  <!-- The transform makes this box the containing block of the scene's
       fixed bars, so they sit inside the preview. -->
  <div
    class="fixed inset-0 z-40 flex [transform:translateZ(0)] flex-col bg-table"
    data-scene-open={scene.id}
  >
    <div class="flex items-center justify-between gap-2 border-b border-rule bg-panel px-3 py-1">
      <span class="truncate text-xs text-fg-muted"
        >{scene.label} {lastKey ? `· ${lastKey}` : ''}</span
      >
      <button
        type="button"
        class="grid min-h-11 min-w-11 place-items-center rounded-control text-fg hover:bg-panel-sunken"
        aria-label={t('shell.close')}
        onclick={() => {
          active = null;
          lastKey = '';
        }}
      >
        <X class="size-5" aria-hidden="true" />
      </button>
    </div>
    <div class="relative min-h-0 flex-1">
      {#key scene.id}
        {#if scene.kind === 'draft'}
          <Draft view={scene.view} canAct={scene.view.choices.length > 0} {onact} {names} />
        {:else if scene.kind === 'choice'}
          <!-- The bars sit in the flow just above the turn bar in a game, so
               the scene puts them at its foot. -->
          <div class="flex h-full flex-col justify-end">
            <ChoiceHost
              view={scene.view}
              canAct={scene.view.choices.length > 0 && !scene.offline}
              {onact}
              {names}
              events={scene.events}
              confirm={scene.confirm ?? false}
              connected={!scene.offline}
            />
          </div>
        {:else}
          <GameOver
            view={scene.view}
            onrematch={() => (lastKey = 'rematch')}
            onhome={() => (lastKey = 'home')}
            {names}
          />
        {/if}
      {/key}
    </div>
  </div>
{/if}
