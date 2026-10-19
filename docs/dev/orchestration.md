# Orchestration: how the work is run

Ajayeb is built by a long-running coordinator agent that hands bounded pieces of work to other agents, verifies them independently, and commits only what passes. This page describes that mechanism. It is generic on purpose: the facts of any one machine or night live in the private run state, not here.

## Roles

- **The owner** sets direction and steers the run with a few fixed phrases (below).
- **The coordinator** is one long-lived agent session. It plans slices, launches workflows, folds results into the notes, and owns the single-writer files. Its session and recovery mechanism belong in the private run state.
- **Slice agents** each do one bounded slice, verify it and commit the owned files on their assigned branch.
- **Reviewers** are fresh agents that re-verify a slice, read its diff, and either approve it for integration or return concrete findings.
- **Workflows** are scripts that fan work out to several agents and gather the results, for example blind extractors followed by a reconciler.

## Where state lives

The repository is public, so operational state lives outside it, in a private run directory (`~/tmp/agents/ajayeb/` by default, or `AJAYEB_RUN_DIR`):

- `TASK.md`: the original request and every owner instruction, dated and verbatim. It contains only the owner's words. Coordinator prompts and progress go in checkpoints, events and `STATE.md`, never in `TASK.md`.
- `STATE.md`: the handoff state, newest entry first, stamped with the output of `date`. It records the mode, the cron job ids and when they were armed, active workflow run ids and script paths, live worktrees, the task list with status tokens, and open questions for the owner with the assumption used meanwhile.
- `NOTES.md`: iteration notes, written only by the coordinator. A curated "Learnings index" of at most 40 lines sits at the top, followed by one entry per iteration:

  ```
  ### 2026-10-02 01:14 M2.military-looting [OK] 3f2a1c9
  Summary: ...
  Key changes: ...
  Key learnings: ...
  ```

- `mode.json`: the run mode read by `tools/heavy.py`.
- Directories: `heavy/` (slot locks), `logs/` (including `heavy.jsonl`), `results/` (one `<slice>.result.md` per slice, whose first line is `<ID> DONE`, `PARTIAL` or `FAILED`), `briefs/`, `patches/`, `shots/`, `art-work/` (every art attempt, including rejects), `reports/`, `design/` (the detailed design inputs behind the plan), and `workflows/` (durable copies of the workflow scripts).

The durable, public parts live in the repo under `docs/dev/`: this page, [roadmap.md](roadmap.md) (updated only at the end of a milestone), [ram-policy.md](ram-policy.md) and [decisions.md](decisions.md). `CLAUDE.md` carries "Compact Instructions" that tell a coordinator whose context was compacted, or a fresh one, how to resume from these files without relaunching work that is still running.

## Run modes

The mode lives in `mode.json` and is set with `python3 tools/heavy.py mode ...`. The guard, not the scheduler, is the real enforcement: a scheduled prompt can fire late while the session is busy, but the guard refuses heavy work outside the allowed hours regardless.

- **continuous** (the default): work around the clock, capped by the memory guard. Only the hourly recovery ping is scheduled.
- **nightshift**: heavy work runs only inside a window, by default 00:00 to 10:00 local time. Outside it the run is in "light mode": the coordinator alone, with no workflows and no browsers. Docs, planning, reading reviews, git, and single-package test runs of kind `light` are still allowed. The guard refuses everything else with exit code 75.
- **paused**: finish the current step, checkpoint, and stop. Nothing heavy runs until the owner resumes.

## Owner phrases

Each phrase is first recorded verbatim in `TASK.md`, then acted on.

- **"switch to nightshift [HH:MM-HH:MM]"**: set the mode with `heavy.py mode nightshift --window ...`; create the recurring jobs for the start procedure (`3 0 * * *`), the last call (`12 9 * * *`) and the wind-down (`44 9 * * *`), keeping the hourly recovery ping (`17 * * * *`); record their ids in `STATE.md`; and if the current time is outside the window, run the wind-down at once. For a window other than the default, keep the same offsets: start 3 minutes after the window opens, last call 48 minutes before it closes, wind-down 16 minutes before it closes. Off-minutes such as :03 and :17 avoid the crowd of jobs scheduled on the hour.
- **"switch to continuous"** (or "nightshift off"): delete the three window jobs, set the mode to continuous, and resume.
- **"pause ajayeb"**: run the wind-down now, then set the mode to paused.
- **"resume ajayeb"**: restore the mode that was active before the pause and run the start procedure.
- **"ajayeb status"** (or "good morning"): a review rebuilt from `git log`, `heavy.jsonl`, the result files and `STATE.md`, never from memory.

## Scheduled procedures

Each scheduled prompt is self-contained, so it still works after the coordinator's context was compacted.

**Start (00:03).**

1. Read the files that the Compact Instructions list.
2. Verify a clean tree on `master`, no stray processes (`heavy.py status`, a check for leftover browsers), and a green `make verify-quick`. If verification is red, the first slice of the night is the fix.
3. Re-arm every cron job (delete the old ones, create new ones, record the ids). Recurring session jobs expire after some days, and re-arming each night avoids that.
4. Resume stopped workflows from their recorded run ids, or start the next slices from the roadmap.

**Last call (09:12).** Start no new slice or workflow. Slices are sized to finish in 45 minutes or less, so whatever is running can complete before the wind-down.

**Wind-down (09:44).**

1. Stop running workflows and record their run ids and script paths in `STATE.md`.
2. For an interrupted slice, save its diff to `patches/<slice>-<timestamp>.patch`, then restore only that slice's owned paths. Never reset the whole tree.
3. Record the live worktrees with their branches and states.
4. Run `heavy.py stop-all`.
5. Write `reports/night-<date>.md` from `git log`, `heavy.jsonl` and the result files.
6. Update `STATE.md`, push if a natural endpoint was reached, and reply in one line.

**Hourly recovery ping (minute 17).** One call gathers the facts: the mode and the time, `heavy.py status`, workflow and task status, result file modification times, and `tmux ls`. They are compared with the newest `STATE.md` entry, and the coordinator acts only on a discrepancy:

- heavy work running outside the window;
- a workflow that `STATE.md` calls running but that has finished;
- a result file not yet folded into `NOTES.md` and `STATE.md`;
- a dead tmux session that `self_host.py` should be running;
- a slice that has been silent for more than 90 minutes.

Otherwise it replies in one line. The ping is what catches a scheduled job that fired late or not at all.

The coordinator also avoids starting a large workflow when its usage window is mostly spent (above roughly 70%), because a workflow that is cut off mid-flight costs more to recover than to delay.

## Bounded slices

The practices are borrowed from gnhf (one bounded change per iteration, verified before commit, failures rolled back and noted), without its harness.

**A slice brief** names:

- the slice id and its objective;
- the files the slice owns, and the files it should read;
- the verification commands;
- an observable stop condition (for example, "these tests exist and pass");
- the path of its result file.

Every brief also says that the agent runs unattended (the brief settles the design questions), that it should read the Learnings index in `NOTES.md` first, and that it must stop every process it started before returning.

**The slice agent** works only on its owned files, runs the verification, and commits a successful slice with an explicit pathspec. It returns a structured result: success, a summary, the key changes, the key learnings, whether the loop should stop, and the files touched.

**The reviewer** is a fresh agent. It re-runs verification, checks the diff against the brief and source, and either approves it or reports findings for another implementation round. The assigned integrator rebases an approved branch onto current `master`, verifies it again and fast-forwards a clean main checkout. The brief names who owns integration; workers never merge their own work unless explicitly assigned that role.

**Review by risk.** How much review a slice gets depends on what it could break. The coordinator picks the tier when the slice reports done and says which tier it chose.

- **Full:** stored games, the engine or rules, the protocol, deploy scripts, security, and anything written by a weaker model. One independent reviewer from the other model family. Only the fix of a real blocker is reviewed again.
- **Light:** ordinary UI. One cheaper reviewer, with no second round; the integration checks confirm that its findings were fixed.
- **None:** tests, tooling, docs and chores. The coordinator reads the diff (checking that no test was weakened) and relies on the automated checks.

The tiers came from a day on which three of eight reviews found blockers, all in a first round on a risky slice, and all three second rounds found nothing new while each cost about an hour. The rounds of the UX review loop are sized the same way: one player reviewer plus the scripted probes.

**One commit per successful slice,** with an explicit pathspec:

```sh
git add -- <paths>
git commit -m "engine: add military track and looting (M2.military-looting)" -m "Slice: M2.military-looting" \
  -m "Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>" -- <paths>
```

The `git add` is needed because `git commit -- <paths>` refuses untracked files.

**Failure handling.**

- A failed slice reports its failing checks and keeps its state on the assigned branch for correction. Preserve unrelated changes and published history. The coordinator curates the learnings in `NOTES.md`.
- An audit that needs no change reports already-done evidence rather than creating an artificial commit.
- Three consecutive failures mark the slice `blocked`, with a question for the owner in `STATE.md`. The loop moves on to other work.

**Resumability.** A slice agent first runs `git log --grep "Slice: <id>"`. If the slice is already committed, it reports done. If its owned files are dirty, it continues from that state instead of starting over.

**Reports** (morning reviews, night reports, status answers) are rebuilt from git, the logs and the result files, never from an agent's memory of what happened.

## Avoiding merge conflicts

- Some files have a single writer, the coordinator or the integrator: `go.mod`, `go.sum`, the `package.json` files, `pnpm-lock.yaml`, the `Makefile`, generated `*.gen.ts` files, `AGENTS.md`, `docs/protocol.md`, and the frozen engine types once they are frozen. Workers ask for changes to these in their result file.
- The engine has one file per rules area. The web client has one component per file, one i18n file per area and gallery fixtures for leaf components. Shared entry modules such as `web/src/lib/wire/index.ts` re-export their public types.
- At most three worktrees exist at once. Their branches are temporary: an integrator rebases each onto `master`, runs `make verify-quick` and fast-forwards, one branch at a time, so `master` stays linear.

## Workflows

- **W1 rules data (M1):** three blind extractors with different sources, a reconciler that diffs every field, three refuters per disagreement (the majority wins; anything unresolved becomes an owner question), a writer for the data, tests and rules docs, and two reviewers (completeness, and no rulebook wording).
- **W2 engine slices (M2):** a sequential loop of slice agent plus reviewer. Once the core types are frozen, independent effect families may run in up to three worktrees with disjoint file ownership.
- **W2b rules audit (M2):** six lens finders (setup and draft; costs, trading and chains; military; science and tokens; wonder effects; ages, endgame and scoring), each writing suspected divergences as failing tests. Findings are deduplicated, refuted twice each, and fixed one at a time. The loop stops after two rounds in a row with nothing new.
- **W3 AI (M3):** sequential slices; the tournament benchmark is the verification.
- **W4 server (M4):** sequential slices, then a three-lens security review (hidden-information leaks, identity and authorization, denial of service), two refuters per finding, then fixes.
- **W5 UI (M5 to M7):** the design system and shell slice runs alone; components then run in up to three worktrees, each component one file with its gallery fixture and its i18n file; one agent writes the end-to-end specs, because only one browser may run.
- **W6 vector art (M9):** a pipeline over the asset manifest with a concurrency of two or three: draft, fast render, a separate critic that looks at the image, revise, for at most a few rounds; a batch contact sheet; then an art-director pass for consistency across the whole set.
- **W7 bitmap art (M9):** not a fan-out. A script generates one asset at a time through the guard's codex lock, verifies and processes each file, and a small critique workflow approves or regenerates.
- **UX review loop (M11):** after M10 builds the game-feel features, plain-script probes and two reviewers (player and screens) check the finished game. Fixers group reproducible findings by file, capture the affected screens before and after, and run regression checks. There are no refuters or separate judge. The loop stops when a round finds only small issues that do not affect real play, with no open P0 or P1 finding (D22 in [decisions.md](decisions.md#d22-build-the-game-feel-features-first-then-one-review-loop)).

All workflow agents count against the cap on live agents in [ram-policy.md](ram-policy.md).
