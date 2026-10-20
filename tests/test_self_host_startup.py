"""Startup and shipping health budgets, with no network or real sleeps."""

import io
import subprocess
import unittest
import urllib.error
from contextlib import redirect_stderr
from unittest import mock

import self_host


class HealthPollWindowTests(unittest.TestCase):
    def setUp(self):
        self.now = 0.0
        self.sleeps = []
        self.stderr = io.StringIO()
        self.response = mock.MagicMock(status=200)
        self.response.__enter__.return_value = self.response
        self.response.read.return_value = b'{"version":"v-test"}'
        for obj, name, kwargs in (
            (self_host.time, "monotonic", {"side_effect": lambda: self.now}),
            (self_host.time, "sleep", {"side_effect": self.sleep}),
            (self_host.urllib.request, "urlopen", {"side_effect": self.unavailable}),
        ):
            patcher = mock.patch.object(obj, name, **kwargs)
            patched = patcher.start()
            self.addCleanup(patcher.stop)
            if name == "urlopen":
                self.urlopen = patched

    def sleep(self, seconds):
        self.sleeps.append(seconds)
        self.now = round(self.now + seconds, 10)

    def unavailable(self, *args, **kwargs):
        raise urllib.error.URLError("not listening yet")

    def ready_after_45_seconds(self, *args, **kwargs):
        if self.now < 45:
            return self.unavailable()
        return self.response

    def config(self):
        return self_host.Config(site=self_host.parse_site_url("https://game.example.test"),
                                mode=self_host.Mode.PROD)

    def test_production_start_can_take_45_seconds_and_reports_its_elapsed_time(self):
        self.urlopen.side_effect = self.ready_after_45_seconds
        with mock.patch.multiple(self_host, apply_caddyfile_safely=mock.Mock(return_value=True),
                                 start_server_prod=mock.DEFAULT, announce_serving=mock.DEFAULT) as mocks, \
             redirect_stderr(self.stderr):
            self_host.start_prod_with_caddy(self.config(), None)
        self.assertAlmostEqual(self.now, 45)
        mocks["start_server_prod"].assert_called_once()
        mocks["announce_serving"].assert_called_once()
        lines = self.stderr.getvalue().splitlines()
        waiting = [line for line in lines if line.startswith("waiting for game server /health")]
        self.assertGreater(len(waiting), 1)
        self.assertLess(len(waiting), 15)  # Progress is sparse, not one line per poll.
        self.assertIn("/90s)", waiting[0])
        self.assertIn("after 45.0s", lines[-1])
        self.assertTrue(all(delay == 0.3 for delay in self.sleeps))

    def test_unavailable_start_expires_at_90_seconds_without_announcing_success(self):
        def timed_out_request(_url, *, timeout):
            self.now = round(self.now + timeout, 10)
            raise TimeoutError("request timed out")
        self.urlopen.side_effect = timed_out_request
        with mock.patch.multiple(self_host, apply_caddyfile_safely=mock.Mock(return_value=True),
                                 start_server_prod=mock.DEFAULT, announce_serving=mock.DEFAULT) as mocks, \
             redirect_stderr(self.stderr):
            with self.assertRaisesRegex(RuntimeError, "/health did not respond within 90s"):
                self_host.start_prod_with_caddy(self.config(), None)
        self.assertAlmostEqual(self.now, 90)
        mocks["start_server_prod"].assert_called_once()
        mocks["announce_serving"].assert_not_called()
        self.assertNotIn("/health ready", self.stderr.getvalue())
        self.assertLess(self.urlopen.call_args.kwargs["timeout"], 1)
        self.assertTrue(all(0 <= delay <= 0.3 for delay in self.sleeps))

    def test_routine_probe_keeps_its_quiet_10_second_default(self):
        with redirect_stderr(self.stderr):
            self.assertFalse(self_host.poll_health(38953))
        self.assertAlmostEqual(self.now, 10)
        self.assertEqual(self.stderr.getvalue(), "")

    def test_all_shipping_probes_can_recover_after_the_old_30_second_window(self):
        def host_response(*args, **kwargs):
            return subprocess.CompletedProcess([], 22 if self.now < 45 else 0,
                                               stdout='{"version":"v-test"}')

        def public_response(*args, **kwargs):
            if self.now < 45:
                raise TimeoutError("not available yet")
            return "v-test"

        self.urlopen.side_effect = self.ready_after_45_seconds
        with mock.patch.object(self_host.subprocess, "run", side_effect=host_response), \
             mock.patch.object(self_host, "public_health_version", side_effect=public_response):
            for name, args in (
                ("poll_remote_version", ("https://game.example.test",)),
                ("poll_host_version", ("deploy@example.test", self.config().site)),
                ("poll_public_version", ("https://game.example.test",)),
            ):
                with self.subTest(probe=name):
                    self.now = 0.0
                    self.sleeps.clear()
                    self.assertEqual(getattr(self_host, name)(*args), "v-test")
                    self.assertAlmostEqual(self.now, 45)
                    self.assertTrue(all(delay == 1 for delay in self.sleeps))
