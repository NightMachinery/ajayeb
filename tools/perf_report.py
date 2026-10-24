#!/usr/bin/env python3
"""The verdict and run summary of CI's e2e-perf job.

The job repeats animation.spec.ts case d (no long task over 50 ms during an
animation at 4x CPU) several times per project, with no retries. Each attempt
keeps its own strict assertion and attaches the long tasks that overlapped an
animation (`perf overlaps`: the room, the frames whose animations it
overlapped, its length and what the long animation frames say it ran).

On GitHub's shared runners an attempt sometimes fails on a native stall (a
major GC, a compositor commit, renderer frame work) that lands in a different
animation each time; docs/dev/master-e2e-failures.md has the evidence. So the
job judges the attempts together: a place (a room and its frames) where a long
task overlaps an animation in more than half of a project's attempts fails the
job, and one that does so in fewer is listed as flaky. An attempt that fails
for any other reason, or a project with no attempt, also fails it. The 50 ms
limit and the per-attempt assertion are unchanged.

Usage: python3 tools/perf_report.py PERF_JSON [--summary FILE]
Prints the summary as Markdown (appending it to FILE when given) and exits 1
when the job fails, 0 when it passes.
"""

from __future__ import annotations

import argparse
import base64
import json
import statistics
import sys
from dataclasses import dataclass, field

CASE_D = "d. no long task over 50 ms during an animation"
CASE_F = "f. frame gaps during animations"
# The message of case d's overlap assertion; any other failure is an error.
OVERLAP_FAILURE = "long tasks during animations"


@dataclass
class Overlap:
    room: str
    versions: list[int]
    kinds: list[str]
    start: int
    duration: int
    explained: str

    @property
    def place(self) -> str:
        return f"{self.room} v{', v'.join(str(v) for v in self.versions)}"


@dataclass
class Attempt:
    status: str
    overlaps: list[Overlap]
    perf: list[str]
    errors: list[str]

    @property
    def worst(self) -> int:
        return max((o.duration for o in self.overlaps), default=0)

    @property
    def other_error(self) -> str | None:
        """The first failure that is not the overlap assertion, if any."""
        if self.status == "passed":
            return None
        for error in self.errors:
            if OVERLAP_FAILURE not in error:
                return error.strip().splitlines()[0] if error.strip() else self.status
        if not self.errors:
            return self.status
        if not self.overlaps:
            return "the overlap assertion failed but no perf overlaps were attached"
        return None


@dataclass
class Project:
    name: str
    attempts: list[Attempt] = field(default_factory=list)
    frame_gaps: list[str] = field(default_factory=list)

    def places(self) -> dict[str, list[Overlap]]:
        """The overlaps by place, each place counted once per attempt."""
        out: dict[str, list[Overlap]] = {}
        for attempt in self.attempts:
            seen: dict[str, Overlap] = {}
            for o in attempt.overlaps:
                if o.place not in seen or o.duration > seen[o.place].duration:
                    seen[o.place] = o
            for place, o in seen.items():
                out.setdefault(place, []).append(o)
        return out

    def reproducible(self) -> list[str]:
        """Places that overlap in more than half of the attempts."""
        n = len(self.attempts)
        return sorted(p for p, hits in self.places().items() if 2 * len(hits) > n)


def _specs(suite: dict):
    for child in suite.get("suites", []):
        yield from _specs(child)
    yield from suite.get("specs", [])


def _attachment(result: dict, name: str):
    for a in result.get("attachments", []):
        if a.get("name") == name and "body" in a:
            return json.loads(base64.b64decode(a["body"]))
    return None


def read_report(report: dict) -> dict[str, Project]:
    projects: dict[str, Project] = {}
    for suite in report.get("suites", []):
        for spec in _specs(suite):
            title = spec.get("title", "")
            if not (title.startswith(CASE_D) or title.startswith(CASE_F)):
                continue
            for test in spec.get("tests", []):
                project = projects.setdefault(test["projectName"], Project(test["projectName"]))
                if not test.get("results"):
                    continue
                result = test["results"][-1]
                notes = [a for a in result.get("annotations", []) if a.get("type") == "perf"]
                perf = [a.get("description", "") for a in notes]
                if title.startswith(CASE_F):
                    if result.get("status") != "skipped":
                        project.frame_gaps.extend(perf)
                    continue
                if result.get("status") == "skipped":
                    continue
                overlaps = [Overlap(**o) for o in (_attachment(result, "perf overlaps") or [])]
                errors = [e.get("message", "") for e in result.get("errors", [])]
                project.attempts.append(Attempt(result.get("status", ""), overlaps, perf, errors))
    return projects


def judge(projects: dict[str, Project]) -> tuple[bool, list[str]]:
    """The verdict and the summary's lines."""
    ok = True
    lines = [
        "### 4x long tasks (limit: no task over 50 ms overlapping an animation)",
        "",
        "Case d runs several times per project. A long task that overlaps an animation "
        "in the same place (room and frames) in more than half of the attempts fails "
        "this job; one that does so in fewer is runner noise, listed as flaky.",
        "",
    ]
    if not projects:
        return False, lines + ["- no case d results"]
    for name in sorted(projects):
        p = projects[name]
        n = len(p.attempts)
        if n == 0:
            ok = False
            lines.append(f"- {name}: no attempt of case d ran")
            continue
        clean = sum(1 for a in p.attempts if not a.overlaps)
        worst = [a.worst for a in p.attempts]
        lines.append(
            f"- {name}: {clean} of {n} attempts with no overlapping long task; "
            f"worst overlapping task per attempt, median {round(statistics.median(worst))} ms, "
            f"highest {max(worst)} ms"
        )
        reproducible = set(p.reproducible())
        for place, hits in sorted(p.places().items()):
            verdict = "fails" if place in reproducible else "flaky"
            longest = max(hits, key=lambda o: o.duration)
            lines.append(
                f"  - {verdict}, {len(hits)} of {n} attempts: {place} "
                f"({', '.join(longest.kinds)}), up to {longest.duration} ms; {longest.explained}"
            )
        if reproducible:
            ok = False
        for i, attempt in enumerate(p.attempts, 1):
            error = attempt.other_error
            if error is not None:
                ok = False
                lines.append(f"  - attempt {i} failed for another reason: {error}")
        for i, attempt in enumerate(p.attempts, 1):
            for note in attempt.perf:
                lines.append(f"  - attempt {i}, {attempt.status}: {note}")
        for note in p.frame_gaps:
            lines.append(f"  - case f: {note}")
    lines.insert(0, f"**e2e-perf: {'pass' if ok else 'FAIL'}**")
    lines.insert(1, "")
    return ok, lines


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("report", help="Playwright JSON report of the @perf run")
    parser.add_argument("--summary", help="append the Markdown summary to this file")
    args = parser.parse_args(argv)
    try:
        with open(args.report, encoding="utf-8") as f:
            report = json.load(f)
    except (OSError, ValueError) as e:
        print(f"perf_report: cannot read {args.report}: {e}", file=sys.stderr)
        return 1
    ok, lines = judge(read_report(report))
    text = "\n".join(lines) + "\n"
    sys.stdout.write(text)
    if args.summary:
        with open(args.summary, "a", encoding="utf-8") as f:
            f.write(text)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
