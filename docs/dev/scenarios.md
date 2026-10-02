# Named scenarios

A named scenario is a fixed starting position plus a trigger: a short script
of canonical action keys that drives one rules flow end to end, such as
building the Mausoleum with a card in the discard pile, or a pawn one space
from a capital reaching it through a wonder's shield. Each one is defined
once, in `server/internal/engine/scenarios_named.go`, and used from three
places:

- the engine's own tests (`scenarios_named_test.go`), one test per scenario,
  which build the position, run the trigger, and check the state and events
  it predicts;
- the dev endpoint described below, which Playwright and manual testing use
  to open a game already in the position a flow needs, instead of playing
  toward it move by move;
- a person reading this file, to see at a glance what each name sets up.

A scenario is data, not a saved game: `NewGameFromScenario` builds it from an
ordinary seeded deal, then rearranges the cards, wonders, tokens and pawn the
scenario names, leaving everything else (which wonder a filler slot gets,
which removed card is unnamed) random but reproducible from the seed. Running
a scenario's trigger again, from the same seed, always reaches the same
state.

## Looking one up

`engine.NamedScenario(name)` returns a fresh copy of the named scenario, or
nil if `name` is not one of the names `engine.NamedScenarioNames()` lists
(sorted, for a stable menu). Building the actual game is a separate step:

```go
sc := engine.NamedScenario("mausoleum")
s, err := engine.NewGameFromScenario(seed, engine.Options{}, sc)
```

A `Scenario` value's `Trigger` field is the action keys to run with
`engine.RunTrigger`, and `Expect` is the same flow in a sentence, for the
reader rather than for the code.

## The dev endpoint

Once the server exists (milestone M4), it exposes `POST /api/dev/rooms` only
when dev endpoints are enabled (the `--dev-endpoints` flag that
`docs/architecture.md` describes), never in a normal deployment. The request
body is JSON:

```json
{"mode": "hotseat", "seed": "1", "scenario": "mausoleum"}
```

or, to start from a sequence of actions instead of a named scenario:

```json
{"mode": "hotseat", "seed": "1", "actions": ["draft:pyramids", "draft:sphinx"]}
```

- `mode` is one of the room modes the server already supports: `ai`,
  `hotseat` or `online`.
- `seed` is the game's seed, sent as a string so a JavaScript caller does not
  lose precision on a 64-bit number.
- `scenario`, when present, names one of `engine.NamedScenarioNames()`. The
  server looks it up, builds the state with `NewGameFromScenario`, and the
  room opens at that starting position, before the scenario's `Trigger`.
  The trigger is not applied for the caller: a Playwright flow plays its
  keys itself, by clicking the matching `[data-act]` buttons, and then
  checks that the game landed where `Expect` says. That is what makes each
  named scenario a test of the real UI for its flow, not just of the engine.
- `actions`, when present instead of `scenario`, are applied to an ordinary
  `NewGame(seed)` before the room opens: each key in order, through the
  ordinary legality check, stopping with an error at the first illegal one.
  This lets a test or a manual session reach an arbitrary position without
  adding a new named scenario for it.
- Exactly one of `scenario` and `actions` is given; the response is the
  room's code and its first view, the same shape a normal room's first
  message takes.

## The scenarios

- **mausoleum**: the Mausoleum is built with a card in the discard pile,
  opening the rebuild choice; reviving that card builds it for free and ends
  the turn.
- **mausoleum-empty**: the Mausoleum is built with an empty discard pile, so
  its rebuild choice is skipped and the turn ends normally.
- **great-library**: the Great Library draws the front three tokens of a
  fixed box, with Law first; keeping Law takes it and returns the other two
  to the box.
- **zeus-destroy**: the Statue of Zeus opens its destroy choice because the
  opponent owns a brown card, and destroying it sends the card to the
  discard pile at once.
- **zeus-no-target**: the Statue of Zeus with no brown card on the opponent's
  side, so its destroy choice is skipped.
- **circus-maximus**: Circus Maximus opens its destroy choice against the
  opponent's one grey card.
- **science-pair**: a second card of a symbol the player already owns pairs
  it, opening the board token choice, without changing how many distinct
  symbols the player has.
- **science-supremacy**: a player who already shows five distinct science
  symbols builds a card with the sixth, winning the game at once.
- **law-supremacy**: a pair that adds no new distinct symbol opens the board
  token choice, and taking Law from it adds the scales as a sixth distinct
  symbol, winning the game even though no single card ever showed six.
- **military-supremacy**: a two-shield card reaches a capital that was two
  spaces away, for an immediate military victory.
- **military-via-wonder**: a wonder's shield reaches a capital one space away
  before the wonder's own special choice can open, so the choice never
  happens even though the opponent has a valid target for it.
- **looting-double**: four shields in one move (a card's three plus
  Strategy's extra shield) carry the pawn past both of a side's looting
  spaces, taking both tokens in the same move.
- **civilian-end**: discarding the last card of Age III ends the game, and
  one player's extra coins decide a civilian victory with no tie.
- **tie-blue**: both players end Age III with the same total score, and the
  tie is broken by blue-card points.
- **draw**: both players end Age III with the same total score and the same
  blue-card points too, so the game ends in a shared victory.
- **theology-chain**: a player who owns Theology gets an extra turn for
  every wonder built, even ones that grant no extra turn of their own, so two
  wonders are built back to back without the turn ever passing.
- **appian-economy**: a wonder bought entirely by trade pays its coins to an
  opponent who owns Economy before the Appian Way's own effect takes 3 coins
  from that same opponent, so the loss is never floored to zero by paying it
  first.
- **age-transition-chooser**: the last card of an age is taken with the pawn
  off centre, and the player on the pawn's weaker side chooses who starts
  the next age, not the player who just took the last card.
- **extra-turn-lost**: a wonder's extra turn is granted on an age's last
  card, so the age ends before the extra turn can be spent and it is lost.
