"""Tests for tools/perf_report.py, the verdict of CI's e2e-perf job.

Run: python3 -m unittest discover -s tests -v
"""

from __future__ import annotations

import base64
import importlib.util
import json
import os
import sys
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PATH = os.path.join(ROOT, "tools", "perf_report.py")


def _load():
    spec = importlib.util.spec_from_file_location("ajayeb_perf_report", PATH)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module  # dataclasses look the module up here
    spec.loader.exec_module(module)
    return module


perf_report = _load()

D = "d. no long task over 50 ms during an animation at 4x CPU"
F = "f. frame gaps during animations at 1x CPU"
OVERLAP_ERROR = "Error: long tasks during animations (worst 72 ms)\n\nexpect(received).toEqual(expected)"


def overlap(room="age change", versions=(1,), kinds=("coins", "deal"), duration=72):
    return {
        "room": room,
        "versions": list(versions),
        "kinds": list(kinds),
        "start": 2747,
        "duration": duration,
        "explained": "frame 80 ms at 2740 ms: render 70 ms (style and layout 12 ms); scripts none",
    }


def spec(project, title=D, status="passed", overlaps=None, errors=None, perf="moves 10"):
    result = {
        "status": status,
        "errors": [{"message": m} for m in (errors or [])],
        "annotations": [{"type": "perf", "description": perf}],
        "attachments": [],
    }
    if overlaps is not None:
        body = base64.b64encode(json.dumps(overlaps).encode()).decode()
        result["attachments"].append(
            {"name": "perf overlaps", "contentType": "application/json", "body": body}
        )
    return {"title": title, "tests": [{"projectName": project, "results": [result]}]}


def report(*specs):
    # Playwright nests specs in file and describe suites.
    return {"suites": [{"title": "animation.spec.ts", "suites": [{"title": "animations", "specs": list(specs)}]}]}


def clean(project, n):
    return [spec(project, overlaps=[]) for _ in range(n)]


def failed(project, *overlaps):
    return spec(project, status="failed", overlaps=list(overlaps), errors=[OVERLAP_ERROR])


def judge(rep):
    return perf_report.judge(perf_report.read_report(rep))


class Verdict(unittest.TestCase):
    def test_one_overlap_in_five_attempts_is_flaky(self):
        ok, lines = judge(report(failed("desktop", overlap()), *clean("desktop", 4), *clean("mobile", 5)))
        text = "\n".join(lines)
        self.assertTrue(ok, text)
        self.assertIn("**e2e-perf: pass**", text)
        self.assertIn("- desktop: 4 of 5 attempts with no overlapping long task", text)
        self.assertIn("median 0 ms, highest 72 ms", text)
        self.assertIn("flaky, 1 of 5 attempts: age change v1 (coins, deal), up to 72 ms; frame 80 ms", text)
        self.assertIn("- mobile: 5 of 5 attempts", text)

    def test_a_place_in_most_attempts_fails(self):
        rep = report(
            *[failed("desktop", overlap(duration=60 + i)) for i in range(3)],
            *clean("desktop", 2),
        )
        ok, lines = judge(rep)
        text = "\n".join(lines)
        self.assertFalse(ok, text)
        self.assertIn("**e2e-perf: FAIL**", text)
        self.assertIn("fails, 3 of 5 attempts: age change v1 (coins, deal), up to 62 ms", text)
        self.assertIn("median 60 ms", text)

    def test_half_of_the_attempts_is_not_most(self):
        ok, _ = judge(report(failed("desktop", overlap()), failed("desktop", overlap()), *clean("desktop", 2)))
        self.assertTrue(ok)

    def test_noise_in_a_new_place_every_attempt_passes(self):
        rep = report(*[failed("desktop", overlap(room="Age I", versions=(7 + i,), kinds=("coins",))) for i in range(5)])
        ok, lines = judge(rep)
        text = "\n".join(lines)
        self.assertTrue(ok, text)
        self.assertIn("- desktop: 0 of 5 attempts", text)
        for v in range(7, 12):
            self.assertIn(f"flaky, 1 of 5 attempts: Age I v{v} (coins)", text)

    def test_two_tasks_in_one_place_count_once_per_attempt(self):
        rep = report(failed("desktop", overlap(duration=55), overlap(duration=90)), *clean("desktop", 1))
        ok, lines = judge(rep)
        self.assertTrue(ok)
        self.assertIn("flaky, 1 of 2 attempts: age change v1 (coins, deal), up to 90 ms", "\n".join(lines))

    def test_a_single_attempt_keeps_the_strict_rule(self):
        ok, _ = judge(report(failed("desktop", overlap())))
        self.assertFalse(ok)


class Errors(unittest.TestCase):
    def test_another_failure_fails(self):
        rep = report(
            spec("desktop", status="failed", overlaps=[], errors=["Error: console errors, page errors or failed requests"]),
            *clean("desktop", 4),
        )
        ok, lines = judge(rep)
        self.assertFalse(ok)
        self.assertIn("attempt 1 failed for another reason: Error: console errors", "\n".join(lines))

    def test_a_timeout_fails(self):
        ok, _ = judge(report(spec("mobile", status="timedOut", overlaps=[], errors=["Test timeout of 420000ms exceeded."])))
        self.assertFalse(ok)

    def test_an_overlap_failure_without_its_attachment_fails(self):
        ok, lines = judge(report(spec("desktop", status="failed", errors=[OVERLAP_ERROR]), *clean("desktop", 4)))
        self.assertFalse(ok)
        self.assertIn("no perf overlaps were attached", "\n".join(lines))

    def test_no_attempt_fails(self):
        self.assertFalse(judge(report())[0])
        ok, lines = judge(report(spec("mobile", status="skipped"), spec("mobile", title=F, perf="at 1x: gaps")))
        self.assertFalse(ok)
        self.assertIn("- mobile: no attempt of case d ran", "\n".join(lines))


class Report(unittest.TestCase):
    def test_case_f_reports_its_one_run(self):
        rep = report(
            *clean("desktop", 2),
            spec("desktop", title=F, perf="at 1x: frames during animations 933"),
            spec("desktop", title=F, status="skipped", perf="never measured"),
        )
        ok, lines = judge(rep)
        text = "\n".join(lines)
        self.assertTrue(ok)
        self.assertIn("  - case f: at 1x: frames during animations 933", text)
        self.assertNotIn("never measured", text)
        self.assertIn("  - attempt 2, passed: moves 10", text)

    def test_main_appends_the_summary_and_exits_with_the_verdict(self):
        with tempfile.TemporaryDirectory() as d:
            for name, rep, code in [
                ("pass", report(failed("desktop", overlap()), *clean("desktop", 4)), 0),
                ("fail", report(*[failed("desktop", overlap()) for _ in range(3)]), 1),
            ]:
                path = os.path.join(d, f"{name}.json")
                with open(path, "w", encoding="utf-8") as f:
                    json.dump(rep, f)
                summary = os.path.join(d, "summary.md")
                with open(summary, "w", encoding="utf-8") as f:
                    f.write("earlier step\n")
                with open(os.devnull, "w") as null:
                    stdout, sys.stdout = sys.stdout, null
                    try:
                        self.assertEqual(perf_report.main([path, "--summary", summary]), code, name)
                    finally:
                        sys.stdout = stdout
                with open(summary, encoding="utf-8") as f:
                    text = f.read()
                self.assertTrue(text.startswith("earlier step\n**e2e-perf: "), text)

    def test_an_unreadable_report_fails(self):
        with tempfile.TemporaryDirectory() as d:
            stderr, sys.stderr = sys.stderr, open(os.devnull, "w")
            try:
                self.assertEqual(perf_report.main([os.path.join(d, "missing.json")]), 1)
            finally:
                sys.stderr.close()
                sys.stderr = stderr


if __name__ == "__main__":
    unittest.main()
