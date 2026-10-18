<script lang="ts">
  // M5.S03: the board chrome gallery. Every component this slice owns, in
  // several states, built from real recorded views and state frames
  // (src/lib/wire/fixtures.gen.ts) wherever one exists -- the same fixtures
  // threats.test.ts, turn.test.ts and log-format.test.ts already use. The
  // one state no recorded game reaches (a mid-game science distinct count of
  // 5; a real game is over the moment a seat completes six) is built by
  // cloning a real view and overriding that one field, the same technique
  // turn.test.ts uses for extraTurn.
  import { t } from '$lib/i18n';
  import {
    fixtureNames,
    fixtures,
    protocolFixtureNames,
    protocolFixtures
  } from '$lib/wire/fixtures.gen';
  import type { GameEvent, State, View } from '$lib/wire';
  import { turnState, type TurnContext } from '$lib/view/turn';

  import MilitaryTrack from '$lib/components/board/MilitaryTrack.svelte';
  import LootingToken from '$lib/components/board/LootingToken.svelte';
  import ProgressToken from '$lib/components/token/ProgressToken.svelte';
  import ProgressTokenRow from '$lib/components/token/ProgressTokenRow.svelte';
  import ResourceChips from '$lib/components/city/ResourceChips.svelte';
  import ScienceTracker from '$lib/components/city/ScienceTracker.svelte';
  import WonderList from '$lib/components/wonder/WonderList.svelte';
  import WonderButton from '$lib/components/wonder/WonderButton.svelte';
  import { gameEnded } from '$lib/view/threats';
  import { buildableWonders, wonderTally } from '$lib/view/wonders';
  import CityBand from '$lib/components/city/CityBand.svelte';
  import PlayerStrip from '$lib/components/city/PlayerStrip.svelte';
  import { personaName } from '$lib/components/game/names';
  import CityPanel from '$lib/components/city/CityPanel.svelte';
  import TopBar from '$lib/components/hud/TopBar.svelte';
  import TurnBar from '$lib/components/hud/TurnBar.svelte';
  import ConnectionPill from '$lib/components/hud/ConnectionPill.svelte';
  import CitySheet from '$lib/components/sheets/CitySheet.svelte';
  import LogSheet from '$lib/components/sheets/LogSheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';

  function stateFixture(name: (typeof fixtureNames)[number]): State {
    const frame = fixtures[fixtureNames.indexOf(name)];
    if (!frame) throw new Error(`no fixture named ${name}`);
    if (frame.t !== 'state') throw new Error(`fixture ${name} is not a state frame`);
    return frame;
  }

  function viewFixture(name: (typeof protocolFixtureNames)[number]): View {
    const v = protocolFixtures[protocolFixtureNames.indexOf(name)];
    if (!v) throw new Error(`no protocol fixture named ${name}`);
    return v;
  }

  const names: readonly [string, string] = ['Sara', 'Omid'];

  function cardsLeftOf(view: View): number {
    return view.structure.filter((slot) => slot.state !== 'empty').length;
  }

  // --- Game-state scenes: the same view feeds MilitaryTrack, both players'
  // strips/panels and the TopBar/TurnBar, exactly as a real screen would. ---

  const earlyView = viewFixture('age-1-start-seat0');
  const midView = viewFixture('costs-seat0');
  const militaryDangerView = viewFixture('looting-seat0');

  // No recorded game carries a mid-game science.distinct of 5: a real game
  // ends the moment a seat completes a sixth distinct symbol, so the danger
  // threshold (5) is reachable only for the one turn before that, which none
  // of the recorded scenarios happens to freeze on. One field of a real view
  // is overridden here, the same way turn.test.ts overrides `extraTurn`.
  const scienceDangerView: View = {
    ...midView,
    players: [
      {
        ...midView.players[0],
        science: {
          distinct: 5,
          symbols: [
            { symbol: 'globe', count: 1 },
            { symbol: 'scales', count: 1 },
            { symbol: 'sundial', count: 1 },
            { symbol: 'mortar', count: 1 },
            { symbol: 'quill', count: 1 }
          ]
        }
      },
      midView.players[1]
    ]
  };

  const gameOverCivilianView = viewFixture('game-over-civilian-seat0');
  const gameOverMilitaryView = viewFixture('game-over-military-hidden-seat0');
  const gameOverScienceView = viewFixture('game-over-science-seat0');

  interface Scene {
    key: string;
    label: string;
    view: View;
  }

  const scenes: Scene[] = [
    { key: 'early', label: 'Early game: Age I, just past the wonder draft', view: earlyView },
    { key: 'mid', label: 'Mid game: Age II, cities under way', view: midView },
    {
      key: 'military-danger',
      label: "Military danger: the pawn is 6 spaces out (Omid's capital is threatened)",
      view: militaryDangerView
    },
    {
      key: 'science-danger',
      label: 'Science danger: 5 distinct symbols, one pair from winning (synthetic, see comment)',
      view: scienceDangerView
    },
    { key: 'game-over-civilian', label: 'Game over: civilian victory', view: gameOverCivilianView },
    { key: 'game-over-military', label: 'Game over: military victory', view: gameOverMilitaryView },
    { key: 'game-over-science', label: 'Game over: scientific victory', view: gameOverScienceView }
  ];

  function sceneContext(view: View): TurnContext {
    return { view, mode: 'ai', mySeats: [0], conn: 'open', pending: false, slow: false };
  }

  // --- Progress tokens: the board pick (from view.boardTokens, the public
  // row) and the Great Library pick (from pending.tokens, the three tokens
  // just drawn -- a different list, never part of boardTokens), each with
  // its real pending choice wired straight into ProgressTokenRow's
  // `pickable`. ---

  function tokenChoices(view: View): ReadonlyMap<string, string> {
    const keys: [string, string][] = [];
    for (const choice of view.choices) {
      if (choice.kind === 'token' && choice.token !== undefined) {
        keys.push([choice.token, choice.key]);
      }
    }
    return new Map(keys);
  }

  const boardPickView = viewFixture('pick-board-token-seat0');
  const libraryPickView = viewFixture('pick-library-token-seat0');

  const tokenPickScenes = [
    {
      key: 'board',
      label: 'Picking a board token (the public row)',
      tokens: boardPickView.boardTokens,
      pickable: tokenChoices(boardPickView)
    },
    {
      key: 'library',
      label: 'Picking a Great Library token (just drawn, not the board row)',
      tokens: libraryPickView.pending?.tokens ?? [],
      pickable: tokenChoices(libraryPickView)
    }
  ];

  // --- Wonder states: a built wonder with a card tucked under it, and a
  // removed (left the game, seventh-wonder) one. ---

  const wondersWithTucked = viewFixture('appian-economy-seat0').players[0].wonders;
  const wondersWithRemoved = viewFixture('seventh-wonder-seat0').players[1].wonders;
  // The appian-economy position: seat 0 is to move and may build three wonders.
  const wondersBuildable = buildableWonders(viewFixture('appian-economy-seat0'), 0);

  // --- TurnBar: every TurnState kind, from a real view wherever the state
  // only depends on the view, and a synthesized TurnContext (same shape as
  // turn.test.ts's "connection takes priority" cases) for the three that
  // depend on the connection layer, which does not exist until a later
  // slice wires Session into this. ---

  // Pulled out ahead of the array literal (rather than called inline) so
  // each `ctx` below stays a plain object literal: TurnContext's own
  // contextual type then narrows its `conn`/`mode` string literals, the
  // same way turn.test.ts's TurnContext-annotated consts do. A helper
  // function called inline would instead infer its own widened return type
  // (conn: string), which TurnContext's narrower union then rejects.
  const draftState = stateFixture('state-draft');
  const thinkingState = stateFixture('state-ai-thinking');
  const gameOverState = stateFixture('state-game-over');

  const turnRows: { label: string; ctx: TurnContext }[] = [
    {
      label: 'connecting (no view yet)',
      ctx: {
        view: null,
        mode: 'online',
        mySeats: [0],
        conn: 'connecting',
        pending: false,
        slow: false
      }
    },
    {
      label: 'reconnecting',
      ctx: {
        view: earlyView,
        mode: 'ai',
        mySeats: [0],
        conn: 'reconnecting',
        pending: false,
        slow: false
      }
    },
    {
      label: 'offline',
      ctx: {
        view: earlyView,
        mode: 'ai',
        mySeats: [0],
        conn: 'offline',
        pending: false,
        slow: false
      }
    },
    {
      label: 'sent (request in flight)',
      ctx: { view: earlyView, mode: 'ai', mySeats: [0], conn: 'open', pending: true, slow: false }
    },
    {
      label: 'sent, slow (offers Retry)',
      ctx: { view: earlyView, mode: 'ai', mySeats: [0], conn: 'open', pending: true, slow: true }
    },
    {
      label: 'your turn (ai room)',
      ctx: {
        view: viewFixture('age-1-start-seat0'),
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'hot-seat turn',
      ctx: {
        view: draftState.view,
        mode: draftState.room.mode,
        mySeats: draftState.you.seats as (0 | 1)[],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'waiting for the opponent (online)',
      ctx: {
        view: viewFixture('card-destroyed-seat0'),
        mode: 'online',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'ai is thinking',
      ctx: {
        view: thinkingState.view,
        mode: thinkingState.room.mode,
        mySeats: thinkingState.you.seats as (0 | 1)[],
        thinking: thinkingState.thinking as 0 | 1,
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'play again (extra turn)',
      ctx: {
        view: earlyView,
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false,
        playAgain: { seat: 0, wonder: 'sphinx' }
      }
    },
    {
      label: 'play again, hot-seat (extra turn)',
      ctx: {
        view: earlyView,
        mode: 'hotseat',
        mySeats: [0, 1],
        conn: 'open',
        pending: false,
        slow: false,
        playAgain: { seat: 0, wonder: 'appian-way' }
      }
    },
    {
      label: 'spectating',
      ctx: {
        view: viewFixture('age-1-start-spectator'),
        mode: 'online',
        mySeats: [],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'pending choice: choose the next starter',
      ctx: {
        view: viewFixture('choose-starter-seat0'),
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'pending choice: destroy a card',
      ctx: {
        view: viewFixture('destroy-card-seat0'),
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'pending choice: build from the discard pile',
      ctx: {
        view: viewFixture('build-from-discard-seat0'),
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'pending choice: pick a board token',
      ctx: {
        view: viewFixture('pick-board-token-seat0'),
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'pending choice: pick a Great Library token',
      ctx: {
        view: viewFixture('pick-library-token-seat0'),
        mode: 'ai',
        mySeats: [0],
        conn: 'open',
        pending: false,
        slow: false
      }
    },
    {
      label: 'game over',
      ctx: {
        view: gameOverState.view,
        mode: gameOverState.room.mode,
        mySeats: gameOverState.you.seats as (0 | 1)[],
        conn: 'open',
        pending: false,
        slow: false
      }
    }
  ];

  // --- City sheet / log sheet: opened on demand, like the main gallery's
  // own Sheet demo. The log concatenates two real state frames' events so
  // it spans more than one kind of line; GameEvent has no persistent
  // ordering of its own across updates (see log-format.ts's file comment),
  // so this is exactly the accumulation a later slice's Session will do. ---

  let citySheetOpen = $state(false);
  let logSheetOpen = $state(false);

  const logEvents: GameEvent[] = [
    ...stateFixture('state-ai-thinking').events,
    ...stateFixture('state-pending-token').events,
    ...stateFixture('state-play').events,
    ...stateFixture('state-game-over').events
  ];

  // Dev-only section captions, not shipped UI copy (this page never reaches
  // a player). Named consts rather than literal text nodes so the markup
  // stays {expression}-interpolated throughout, the same way the icon
  // gallery above shows each ICON_NAMES entry.
  const compactCaption = 'Compact variant (landscape top bar), no viewer seat';
  const lootingCaption = 'Looting tokens standalone: present and already taken';
  const medallionCaption = 'Standalone medallion states';
  const resourceCaption = 'Resource chips (mid-game seat, one choice producer)';
  const scienceCaption = 'Science tracker: none, one symbol, a pair, and the danger ring';
  const wonderCaption =
    'Wonder button (none built, one built and buildable now) and wonder lists: built with a card, buildable now, and one out of the game';
  const bandCaption = 'City bands by colour, and the ten-tick cap at higher counts';
</script>

{#snippet sceneLabel(label: string)}
  <h4 class="text-xs font-semibold tracking-wide text-fg-muted uppercase">{label}</h4>
{/snippet}

<div class="board-gallery flex flex-col gap-8">
  <section class="flex flex-col gap-3">
    {@render sceneLabel('Military track')}
    {#each scenes.slice(0, 4) as scene (scene.key)}
      <div class="flex flex-col gap-1 rounded-panel border border-rule bg-panel p-3">
        <p class="text-xs text-fg-muted">{scene.label}</p>
        <MilitaryTrack military={scene.view.military} viewerSeat={0} legend />
      </div>
    {/each}
    <div class="flex flex-col gap-1 rounded-panel border border-rule bg-panel p-3">
      <p class="text-xs text-fg-muted">
        {compactCaption}
      </p>
      <MilitaryTrack military={militaryDangerView.military} compact />
    </div>
    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-3">
      <p class="w-full text-xs text-fg-muted">
        {lootingCaption}
      </p>
      <LootingToken value={2} magnitude={3} present={true} />
      <LootingToken value={2} magnitude={3} present={false} />
      <LootingToken value={5} magnitude={6} present={true} size="sm" />
      <LootingToken value={5} magnitude={6} present={false} size="sm" />
    </div>
  </section>

  <section class="flex flex-col gap-3">
    {@render sceneLabel('Progress tokens')}
    {#each tokenPickScenes as scene (scene.key)}
      <div class="flex flex-col gap-1.5 rounded-panel border border-rule bg-panel p-3">
        <p class="text-xs text-fg-muted">{scene.label}</p>
        <ProgressTokenRow tokens={scene.tokens} pickable={scene.pickable} />
      </div>
    {/each}
    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-3">
      <p class="w-full text-xs text-fg-muted">{medallionCaption}</p>
      <ProgressToken slug="law" />
      <ProgressToken slug="masonry" taken />
      <ProgressToken slug="strategy" actionKey="token:strategy" selectable selected />
      <ProgressToken slug="theology" actionKey="token:theology" selectable={false} />
      <ProgressToken slug="agriculture" size="sm" />
    </div>
  </section>

  <section class="flex flex-col gap-3">
    {@render sceneLabel('Resources, science, wonders, city bands')}
    <div class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <p class="text-xs text-fg-muted">
        {resourceCaption}
      </p>
      <ResourceChips
        production={midView.players[0].production}
        tradePrices={midView.players[0].tradePrices}
      />
    </div>
    <div class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <p class="text-xs text-fg-muted">
        {scienceCaption}
      </p>
      <ScienceTracker science={{ distinct: 0, symbols: [] }} />
      <ScienceTracker science={{ distinct: 1, symbols: [{ symbol: 'quill', count: 1 }] }} />
      <ScienceTracker
        science={{
          distinct: 2,
          symbols: [
            { symbol: 'quill', count: 2 },
            { symbol: 'wheel', count: 1 }
          ]
        }}
      />
      <ScienceTracker science={scienceDangerView.players[0].science} />
      <div class="flex flex-wrap gap-2">
        <ScienceTracker science={{ distinct: 0, symbols: [] }} pill />
        <ScienceTracker
          science={{
            distinct: 2,
            symbols: [
              { symbol: 'quill', count: 2 },
              { symbol: 'wheel', count: 1 }
            ]
          }}
          pill
        />
        <ScienceTracker science={scienceDangerView.players[0].science} pill />
      </div>
    </div>
    <div class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <p class="text-xs text-fg-muted">
        {wonderCaption}
      </p>
      <div class="flex flex-wrap gap-2">
        <WonderButton built={0} total={4} name={names[0]} />
        <WonderButton
          {...wonderTally({ wonders: wondersWithTucked })}
          buildable={wondersBuildable.size > 0}
          name={names[0]}
        />
      </div>
      <WonderList wonders={wondersWithTucked} buildable={wondersBuildable} onselect={() => {}} />
      <WonderList wonders={wondersWithRemoved} />
    </div>
    <div class="flex flex-col gap-2 rounded-panel border border-rule bg-panel p-3">
      <p class="text-xs text-fg-muted">
        {bandCaption}
      </p>
      <div class="flex flex-wrap gap-1.5">
        <CityBand color="brown" count={2} />
        <CityBand color="grey" count={1} />
        <CityBand color="blue" count={3} />
        <CityBand color="green" count={2} />
        <CityBand color="yellow" count={4} />
        <CityBand color="red" count={1} />
        <CityBand color="purple" count={2} />
      </div>
      <div class="flex flex-wrap gap-1.5">
        <CityBand color="blue" count={0} />
        <CityBand color="blue" count={7} />
        <CityBand color="blue" count={12} />
      </div>
    </div>
  </section>

  <!-- Seat 1 is the bot (the Scholar) in the strips and the panels, so its
       avatar shows by its name, as in a vs-AI game. -->
  <section class="flex flex-col gap-3" data-gallery="strips">
    {@render sceneLabel('Player strip (phone)')}
    {#each scenes as scene (scene.key)}
      <div class="flex flex-col gap-1.5">
        <p class="text-xs text-fg-muted">{scene.label}</p>
        <PlayerStrip
          name={names[0]}
          seat={0}
          player={scene.view.players[0]}
          military={scene.view.military}
          score={scene.view.scorePreview[0]}
          ended={gameEnded(scene.view.phase)}
          wonderBuildable={buildableWonders(scene.view, 0).size > 0}
          onwonders={() => {}}
        />
        <PlayerStrip
          name={personaName('normal')}
          seat={1}
          bot="normal"
          player={scene.view.players[1]}
          military={scene.view.military}
          score={scene.view.scorePreview[1]}
          ended={gameEnded(scene.view.phase)}
          wonderBuildable={buildableWonders(scene.view, 1).size > 0}
          onwonders={() => {}}
        />
      </div>
    {/each}
  </section>

  <section class="flex flex-col gap-3" data-gallery="panels">
    {@render sceneLabel('City panel (tablet/desktop)')}
    {#each scenes as scene (scene.key)}
      <div class="flex flex-col gap-1.5">
        <p class="text-xs text-fg-muted">{scene.label}</p>
        <div class="grid grid-cols-1 gap-2 split:grid-cols-2">
          <CityPanel
            name={names[0]}
            seat={0}
            player={scene.view.players[0]}
            military={scene.view.military}
            cardTooltips={scene.view.cardTooltips}
            names={[names[0], personaName('normal')]}
            you={scene.view.viewer}
            score={scene.view.scorePreview[0]}
            ended={gameEnded(scene.view.phase)}
            buildable={buildableWonders(scene.view, 0)}
          />
          <CityPanel
            name={personaName('normal')}
            seat={1}
            bot="normal"
            player={scene.view.players[1]}
            military={scene.view.military}
            cardTooltips={scene.view.cardTooltips}
            names={[names[0], personaName('normal')]}
            you={scene.view.viewer}
            score={scene.view.scorePreview[1]}
            ended={gameEnded(scene.view.phase)}
            buildable={buildableWonders(scene.view, 1)}
          />
        </div>
      </div>
    {/each}
  </section>

  <section class="flex flex-col gap-3">
    {@render sceneLabel('Top bar + turn bar, per game-state scene')}
    {#each scenes as scene (scene.key)}
      <div class="flex flex-col overflow-hidden rounded-panel border border-rule">
        <p class="bg-panel px-3 pt-2 text-xs text-fg-muted">{scene.label}</p>
        <TopBar
          age={scene.view.age}
          cardsLeft={cardsLeftOf(scene.view)}
          canUndo
          onUndo={() => {}}
          onOpenLog={() => {}}
          onOpenSettings={() => {}}
        />
        <TurnBar state={turnState(sceneContext(scene.view))} {names} />
      </div>
    {/each}
  </section>

  <section class="flex flex-col gap-2">
    {@render sceneLabel(
      'Turn bar: every state (connection, pending choices, hot-seat, spectating)'
    )}
    {#each turnRows as row (row.label)}
      <div class="flex flex-col overflow-hidden rounded-panel border border-rule">
        <p class="bg-panel px-3 pt-2 text-xs text-fg-muted">{row.label}</p>
        <TurnBar state={turnState(row.ctx)} {names} onRetry={() => {}} />
      </div>
    {/each}
  </section>

  <section class="flex flex-col gap-2">
    {@render sceneLabel('Connection pill')}
    <div class="flex flex-wrap items-center gap-3 rounded-panel border border-rule bg-panel p-3">
      <ConnectionPill conn="connecting" />
      <ConnectionPill conn="open" />
      <ConnectionPill conn="reconnecting" />
      <ConnectionPill conn="offline" />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    {@render sceneLabel('City sheet / log sheet')}
    <div class="flex flex-wrap gap-3">
      <Button variant="secondary" onclick={() => (citySheetOpen = true)}>
        {t('board.citySheet.title')}
      </Button>
      <Button variant="secondary" onclick={() => (logSheetOpen = true)}>
        {t('board.logSheet.title')}
      </Button>
    </div>
  </section>
</div>

<CitySheet
  bind:open={citySheetOpen}
  {names}
  players={midView.players}
  cardTooltips={midView.cardTooltips}
  you={midView.viewer}
  military={midView.military}
  scores={midView.scorePreview}
/>

<LogSheet bind:open={logSheetOpen} events={logEvents} {names} />
