# The bots

Ajayeb's bots live in `server/internal/ai`, on the server, next to the rules engine. A room with a bot seat asks the bot for a move whenever that seat must decide, exactly as it would wait for a human. This page explains how the bots choose, what each difficulty level searches, how strong the levels measured, what a move costs in time and memory, and how to rerun the measurements.

## What a bot sees

A bot never receives the real game state. It receives an `engine.Observation` of its own seat: the state with every hidden card, wonder and token replaced by "unknown", the seed removed, and the Great Library offer present only when it is the bot's own. Where a search needs a complete state, it calls `Observation.Determinize`, which draws one of the complete states that agree with everything the seat has seen, each with equal probability. Every iteration of a search draws a fresh one, so the bot plans against the whole range of deals it cannot tell apart. A fairness test plays two observations that share their public information but differ in their hidden cards and checks that the bot makes the same choice in both.

Bots are deterministic. A bot's random numbers come from its seed and the observation, never from earlier calls, so the same bot, observation and iteration budget give the same move every time, whatever the number of search workers and however the goroutines are scheduled. A wall-clock limit is the one thing that makes a bot's play depend on the machine. The room seeds its bot from the room seed, so a game against a bot can be replayed.

## The four levels

The rooms offer four levels: Random, Easy, Medium and Hard. Two kinds of search sit behind them.

**Random** picks uniformly among the legal actions.

**Greedy** looks one move ahead, plus the opponent's best reply. For each of a few determinizations (3 by default) it tries every legal action on a copy of the state. When the opponent is to move next, the action is worth the worst evaluation the opponent can reach with one reply; when Greedy keeps the move (an extra turn, a choice of its own, or choosing who starts the next age), it is worth the best of its own follow-ups. A line may add one such extra ply, and a line that stops with the same player still to move gives that player a small tempo bonus. The action's value is its mean over the determinizations. Before any of this, Greedy plays a sure win if it has one: an action that wins at once, or opens a choice of its own in which some option wins, using only public information. Greedy does a few thousand evaluations per move and needs a few fixed-size buffers.

The deterministic Greedy plays the action of highest value. It is the yardstick of the tournaments, not a level the rooms offer.

**Easy** is Greedy with softmax sampling: it plays each action with a probability proportional to `exp((value - best) / T)`, with a temperature T of 1.5 victory points. It usually plays well and sometimes errs, more often when the alternatives are close in value.

**Medium and Hard** are a tree search: single-observer information-set Monte Carlo tree search (SO-ISMCTS) with blind reveals, as described by Cowling, Powley and Whitehouse (2012). The search grows one tree over the bot's information sets. A node is a sequence of actions from the root, whatever cards those actions turned over: the tree never branches on a reveal, and since every action names its card, "take card X" exists only below the nodes where X was face up. Both players' moves share the tree, because the opponent knows almost exactly what the bot knows. One iteration works like this:

1. Determinize the observation.
2. Descend from the root using only the actions legal in that sample. Every child whose action is legal gets one more availability, and the descent follows the child with the best UCB value for the player to move, `Q/N + C * sqrt(ln(availability) / N)` with C = 0.7. A child whose action won the game outright when it was added is always taken. When some legal action has no child yet, one of them is added at random and the descent stops.
3. Roll out from there with a cheap heuristic policy, for at most 16 plies and never past the end of the age, and score the position reached.
4. Back up the reward, from 0 to 1, to every node of the path, as seen by the player who chose that node's action.

The move played is the root's most visited child. Like Greedy, the search first plays a sure win if there is one.

The **rollout policy** prices every accessible card and every open wonder once (it never lists the legal actions in the play phase, which is the hot path). With probability 0.1 it plays at random; otherwise it plays the option worth the most: a card's points, coins, shields (by how they move the pawn), science (by how many different symbols it makes), production while the age is young and chains while a follow-up card can come, minus its price in coins, plus a share of what the taken card would have been worth to the opponent. A move that wins at once is worth more than anything else, so rollouts take a supremacy when they can and deny the opponent's when that card is on offer. At the start of each rollout each player may be given a plan for the whole rollout: science (green cards are worth more) or military (red cards are worth more), each with probability 0.15. The plans make the rollouts follow the game's three roads to victory rather than one average line.

A rollout that ends the game scores 1, 0.5 or 0. Otherwise the reward is `sigmoid(Evaluate / 8)`, so an evaluation lead of 8 victory points maps to about 0.73.

**Root parallelism.** Hard runs several workers. Each grows its own tree from the same root with its own random stream, and when they stop, the statistics of the root's children are summed action by action, worker by worker in a fixed order, so the sum does not depend on scheduling.

**Memory.** Each worker's tree lives in an arena of 24-byte nodes, taken in blocks of 1024 nodes only as the tree grows and recycled between searches. A search adds at most one node per iteration, so a worker never holds more nodes than its share of the iteration budget plus one; under a time limit the cap is the time times 250,000 nodes per second, and never more than about a million nodes (24 MiB) per worker. An iteration allocates nothing. A server-wide pool (`ai.Pool`, half the CPUs by default) bounds how many searches run at once.

## The evaluation

Greedy's leaves and the rollouts' cut-offs are scored by `Evaluate`, a static evaluation in victory-point equivalents from one seat's point of view. It reads only public information, so it gives the same value on an observation as on any of its determinizations. Each feature is scored for both seats and taken as "mine minus the opponent's":

- the score so far without coins and military: blue, green, yellow and purple cards, wonders and tokens;
- coins, worth 0.5 points each early and 0.34 in Age III, where three coins make a point;
- military: the points of the pawn's zone, half a point per space from the centre, and a threat bonus of 25, 12 or 6 points when the pawn is one, two or three spaces from a capital;
- science: a convex value of the number of different symbols (0, 0, 1, 3, 6 and 12 points for zero to five), cut to a quarter when too few of the missing symbols can still be found, plus 0.75 for each symbol held once whose second card can still be had while tokens remain on the board;
- the economy, weighted by how much of the game is left: fixed production with diminishing returns (glass and papyrus a quarter more), the trading price it imposes on the opponent, flexible producers, fixed trade prices, and chain symbols whose free card can still come;
- unbuilt wonders, as a share of what each is worth, while fewer than seven wonders are built;
- the ongoing rules of the progress tokens (Strategy, Economy, Masonry, Architecture, Theology, Urbanism).

A finished game is worth 100 points to the winner (minus 100 to the loser), plus a tenth of a point per point of the final score difference. All of these numbers live in `ai.Weights`; `ajayeb-sim weights` prints them.

## The presets

Each level has two budgets, both in `server/internal/ai/levels.go`:

- The **clock budget** (`Level.Budget`) is what a live game uses. The tree search thinks for a fixed time, so a player never waits longer than that, and the bot plays as well as the server's CPU allows.
- The **fixed budget** (`Level.FixedBudget`) is the clock budget's iteration equivalent: about the number of iterations the clock budget buys on the reference machine, an Apple M2 laptop. A search under a fixed budget is deterministic, so the tournaments and tests use it, and they give the same results on any machine and at any load.

The presets, frozen on 2026-10-09:

- **Random** needs no budget.
- **Easy** is Greedy with softmax sampling at a temperature of 1.5 victory points, over 3 determinizations per move. Its clock and fixed budgets are the same.
- **Medium** is the tree search for 0.4 seconds on one worker. Its fixed budget is 40,000 iterations on one worker.
- **Hard** is the tree search for 2 seconds on up to four workers (never more than `GOMAXPROCS`). Its fixed budget is 500,000 iterations over exactly four workers, whatever the machine, because the worker count changes the trees and so the moves.

The fixed budgets come from `ajayeb-sim levels`, which runs the clock budgets on 60 positions spread over 12 Greedy self-play games and reports the iterations each search completed. On 2026-10-09, on the M2 laptop with Go limited to four threads at nice 10 and the owner's usual background load:

- Medium completed a median of 40,174 iterations per move (10th percentile 18,843, 90th percentile 80,180). The search runs faster as the game goes on: the median was about 33,000 in the wonder draft, 61,000 in Age I, 39,000 in Age II and 80,000 in Age III.
- Hard completed a median of 532,303 iterations per move (10th percentile 404,974, 90th percentile 1,014,845): about 420,000 in the draft, 770,000 in Age I, 480,000 in Age II and 1,010,000 in Age III. Four workers give about 13 times Medium's iterations, not 20, because they share the CPU with each other and with the rest of the machine.

The fixed budgets round these medians to 40,000 and 500,000. Since they are medians, a live Medium or Hard searches less than its fixed budget in the draft and more late in the game.

## Measured strength

The targets come from the project plan. Each is a paired tournament from base seed 1: every deal is played twice, with the seats swapped, so neither bot profits from the deal or from moving first. A shared victory counts half a win. The intervals are 95% Wilson score intervals.

All results below were measured on 2026-10-09 on the M2 laptop. Each comes with its command in the last section.

- **Greedy against Random** (target: at least 90% over 100 games). Greedy scored **100.0%** (95% interval 96.3% to 100.0%), with 20 military, 2 science and 78 civilian victories. This target passes.
- **Medium against Greedy** (target: at least 65% over 100 games). The full tournament ran locally in 25 minutes, two games at a time. Medium scored **71.0%** (95% interval 61.5% to 79.0%): Medium won 71 games (8 military, 9 science, 54 civilian) and Greedy 29 (6 military, 23 civilian). This target passes on the measured rate, but the interval's lower end is under 65%, so 100 games cannot rule out a true rate a little below the target.
- **Hard against Medium** (target: at least 58% over 200 games). The full tournament takes about 6 hours on the laptop, so it was not run locally; the nightly CI job runs it. Two smaller measurements stand in for it:
  - The **proxy**: the same 200 games with both sides at a tenth of their fixed budgets (Hard 50,000 iterations on four workers, Medium 4,000), which keeps the ratio between the budgets. Hard scored **63.7%** (95% interval 56.9% to 70.1%): Hard won 127 games (19 military, 9 science, 99 civilian), Medium 72 (6 military, 9 science, 57 civilian), and one was shared. The proxy passes on the measured rate.
  - The **full-budget sample**: the first 20 games of the full tournament (its first 10 deals), at the real fixed budgets, in 33 minutes. Hard scored **55.0%** (95% interval 34.2% to 74.2%): 11 wins (4 military, 7 civilian) to Medium's 9 (1 science, 8 civilian). On its own this sample is far too small to pass or fail the target.
  - The two agree closely where they overlap. On those same 20 games the proxy scored 12 of 20, against 11 at full budgets, and 17 of the 20 games had the same outcome at both scales. The gap between the proxy's 63.7% and the sample's 55% is therefore mostly a matter of which deals were sampled (the first 10 deals favour Medium at both scales), not of the budgets' size. Tree search gains less from each extra iteration as the budget grows, so the full tournament may still land a few points below the proxy.
  - Hard against Medium therefore passes locally on the proxy, but its full result is likely close to the 58% line. The nightly CI run of the full 200 games is the authority; if it fails, the next step is tuning Hard's search settings (the UCB constant, the rollout length) rather than the shared evaluation weights.

The Easy level has two checks of its own in the slow tests: Easy scored **95.0%** against Random (95% interval 88.8% to 97.8%; threshold 85%), and the deterministic Greedy scored **81.0%** against Easy (95% interval 72.2% to 87.5%; threshold 65%). The softmax therefore costs Easy a clear but modest amount of strength, which is what a beginner level wants.

No weights were tuned in slice S12. The evaluation weights are still the hand-set defaults of slice S10: every target passed its local measurement (Hard on the proxy), and the weights are not the lever for Hard against Medium anyway, since both levels share the same evaluation and differ only in their budgets, so better weights would strengthen both.

## Time and memory per move

A live game uses the clock budgets, so a move's thinking time is fixed by the level. Memory is mostly the search tree: 24 bytes a node, at most one node per iteration, capped per worker by the arena limit. These figures come from `ajayeb-sim levels` on 2026-10-09; the heap figures are the growth of the Go heap during a search, sampled every 100 milliseconds and at its end.

- **Random** takes microseconds and allocates nothing.
- **Easy** (and the deterministic Greedy) took 0.2 ms per move on average and 1.4 ms at most on the calibration positions, and about 0.7 ms on average over the 3,438 moves Greedy played in the Medium tournament (53 ms at most, with the machine heavily loaded). Its memory is a few fixed buffers, a few kilobytes, so the heap does not grow.
- **Medium** thinks for 0.4 seconds (402 ms median, 406 ms at most). Its tree held a median of about 0.8 to 0.9 MiB; the arena caps it at 100,001 nodes, 2.3 MiB, which the fastest late-game searches reach. The heap grew by 2.0 MiB at most.
- **Hard** thinks for 2 seconds (2.06 s median, 2.25 s at most under heavy load: the workers notice the deadline within an iteration, but a loaded machine can be slow to schedule them). Its four trees together held a median of 8 to 12 MiB; the arena caps them at 4 times 500,001 nodes, 46 MiB, and the heap grew by 24 MiB at most.

What a clock budget buys depends on the machine and its load. The calibration above ran twice on the same day: with a load average of about 3.5 (the run that set the fixed budgets) Medium completed a median of 40,174 iterations and Hard 532,303; with a load average of about 10, from other jobs on the laptop, Medium completed 35,441 and Hard 362,784. A busy server therefore plays a little weaker than the fixed budgets, Hard more so than Medium, since its four workers need four free cores. `ai.Pool` keeps the bots themselves from crowding the server.

The fixed budgets cost more time than the clock budgets on a loaded machine: in the tournaments, with two games at a time on four threads and other jobs running, Medium's 40,000 iterations took 0.85 seconds per move on average and Hard's 500,000 took 4.9 seconds (11.8 seconds at most).

## Rerunning the measurements

The tournaments and the calibration are subcommands of `ajayeb-sim`. Run them from `server/`; on the laptop, wrap every run in the memory guard (`python3 ../tools/heavy.py run --kind bench -- ...`) and keep `-p` at 2 or less. Build the command once with `go build -o bin/ajayeb-sim ./cmd/ajayeb-sim`.

- `bin/ajayeb-sim match -a greedy -b random -games 100 -seed 1 -min 0.90` replays the Greedy target. `-a medium -b greedy -games 100 -min 0.65` replays Medium's and `-a hard -b medium -games 200 -min 0.58` Hard's. Medium and Hard play their fixed budgets, so the results depend only on the flags. `-min` makes the command exit with status 1 when bot A scores less than that share.
- `-scale F` multiplies both sides' tree-search budgets by F, which gives the quicker proxy described above (`-scale 0.1` for the Hard proxy). Game i of a match depends only on the seed and i, so `-games 20` plays exactly the first 20 games of the 200-game match: that is the full-budget sample above. `-clock` plays the clock budgets instead of the fixed ones, so the results then depend on the machine and its load. `-v` prints one line per game, and `-logs DIR` writes every game as a log that `ajayeb-sim replay` checks.
- `-shard K/N` plays only the deals whose index is K modulo N, and `-json FILE` writes the results. `bin/ajayeb-sim tally -min R FILE...` merges the shards into one report and fails when a shard is missing. This is how the nightly CI job runs Hard against Medium, and how a long local run can be cut into pieces that release the memory guard's slot between them.
- `bin/ajayeb-sim levels` remeasures the clock budgets (iterations, time and memory per move, by age); `levels -fixed` measures the fixed budgets instead.
- The same tournaments are Go tests behind the `slow` build tag, in `server/internal/ai/tournament_test.go`: `go test -tags slow -run TestTournament -timeout 8h ./internal/ai`. The variables `AJAYEB_TOUR_SCALE` (budget factor), `AJAYEB_TOUR_GAMES` (games per tournament) and `AJAYEB_TOUR_P` (games at once, default 2) turn them into a proxy. The tests and `ajayeb-sim match` set up every game with `ai.PairedGame`, so they play identical games.
- **CI.** The nightly CI run (`.github/workflows/ci.yml`, jobs `tournament` and `tournament-verdict`) plays all three targets in full, at the fixed budgets, split by deal over ten runners, then tallies them against the targets and runs the Easy tournaments of the slow tests. Its verdicts appear on the run's summary page. Start one by hand with `gh workflow run ci.yml`. CI is the authority on the full tournaments.

**Tuning.** The evaluation's weights set the strength of every level but Random; the gap between Medium and Hard depends on their budgets and on the search's own settings (`ai.MCTSConfig`: the UCB constant and the rollouts). `bin/ajayeb-sim weights > w.json` prints the defaults as JSON; after editing a copy, `-wa w.json` (or `-wb`) gives bot A (or B) those weights for a match. Fields left out of the file keep their defaults. A sweep is a loop over such files against a fixed opponent, compared at equal seeds. The defaults in `server/internal/ai/eval.go` change only with a new set of tournaments and an update to this page.
