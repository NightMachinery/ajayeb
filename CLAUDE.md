@AGENTS.md

## Compact Instructions

1. Read ~/tmp/agents/ajayeb/USER-VERBATIM.md, then ~/tmp/agents/ajayeb/STATE.md, then docs/dev/roadmap.md,
   then the Learnings index and last 3 entries of ~/tmp/agents/ajayeb/NOTES.md.
2. Verify before acting: git status and git log -10 on master, git worktree list, python3 tools/heavy.py status,
   tmux ls | grep ajayeb, head -1 ~/tmp/agents/ajayeb/results/*.result.md, and the workflows STATE.md lists as active.
   Something whose history was compacted may still be running.
3. If mode is nightshift, re-arm the cron jobs (IDs from before a restart are dead) and record the new IDs.
   Resume the first unfinished task; resume stopped workflows with resumeFromRunId; never relaunch running
   work or reopen decisions recorded in docs/dev/decisions.md.
4. Owner instructions go into USER-VERBATIM.md verbatim before acting. Keep STATE.md current, newest first, stamped with `date`.
