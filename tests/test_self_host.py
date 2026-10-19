import errno
import io
import json
import os
import shlex
import socket
import subprocess
import tempfile
import threading
import unittest
import urllib.error
from http.server import BaseHTTPRequestHandler, HTTPServer
from contextlib import redirect_stderr, redirect_stdout
from dataclasses import replace
from pathlib import Path
from unittest import mock

import self_host


class SelfHostTests(unittest.TestCase):
    def setUp(self):
        # The apply tests here drive the `caddy reload` fallback path, and no
        # test may reach a real Caddy: no admin address, no Caddy serving.
        for name, value in (("caddy_admin_address", None), ("caddy_serves_a_config", False)):
            patcher = mock.patch.object(self_host, name, return_value=value)
            patcher.start()
            self.addCleanup(patcher.stop)
        patcher = mock.patch.object(self_host, "CADDYFILE_NAMED", False)
        patcher.start()
        self.addCleanup(patcher.stop)

    def test_parse_url_rejects_paths(self):
        with self.assertRaises(ValueError):
            self_host.parse_site_url("https://ajayeb.example.com/room/demo")

    def test_parse_url_defaults_to_https_ajayeb(self):
        site = self_host.parse_site_url(None)
        self.assertEqual(site.scheme, "https")
        self.assertEqual(site.host, "ajayeb.lilf.ir")
        self.assertEqual(site.origin, "https://ajayeb.lilf.ir")

    def test_prod_caddy_block_is_single_reverse_proxy_with_redirect(self):
        site = self_host.parse_site_url("https://game.example.test")
        block = self_host.render_caddy_block(site, self_host.Mode.PROD, self_host.Ports(server=38933, web_dev=38934))
        self.assertIn("http://game.example.test", block)
        self.assertIn("redir https://game.example.test{uri} permanent", block)
        self.assertIn("reverse_proxy localhost:38933", block)
        self.assertIn("encode zstd gzip", block)
        # No try_files/static root: the Go binary serves the embedded SPA.
        self.assertNotIn("try_files", block)
        self.assertNotIn("root *", block)

    def test_dev_caddy_block_handles_backend_before_frontend_proxy(self):
        site = self_host.parse_site_url("https://game.example.test")
        block = self_host.render_caddy_block(site, self_host.Mode.DEV, self_host.Ports(server=38933, web_dev=38934))
        backend = """	handle @backend {
		reverse_proxy localhost:38933
	}"""
        frontend = """	handle {
		reverse_proxy localhost:38934
	}"""
        self.assertIn("@backend path /api /api/* /ws /health", block)
        self.assertIn(backend, block)
        self.assertIn(frontend, block)
        self.assertLess(block.index(backend), block.index(frontend))

    def test_http_caddy_block_redirects_https_to_http(self):
        site = self_host.parse_site_url("http://game.example.test:8080")
        block = self_host.render_caddy_block(site, self_host.Mode.DEV, self_host.Ports(server=38941, web_dev=38942))
        self.assertIn("https://game.example.test:8080", block)
        self.assertIn("redir http://game.example.test:8080{uri} permanent", block)
        self.assertIn("reverse_proxy localhost:38941", block)
        self.assertIn("reverse_proxy localhost:38942", block)

    def test_prod_caddy_block_with_both_schemes_serves_both_and_redirects_neither(self):
        site = self_host.parse_site_url("https://game.example.test")
        block = self_host.render_caddy_block(
            site, self_host.Mode.PROD, self_host.Ports(server=38933, web_dev=38934), both_schemes=True
        )
        self.assertIn("https://game.example.test, http://game.example.test {", block)
        self.assertNotIn("redir", block)
        self.assertEqual(block.count("reverse_proxy localhost:38933"), 1)
        self.assertTrue(block.startswith(self_host.MANAGED_START))
        self.assertTrue(block.rstrip("\n").endswith(self_host.MANAGED_END))

    def test_dev_caddy_block_with_both_schemes_keeps_backend_routing(self):
        site = self_host.parse_site_url("https://game.example.test")
        block = self_host.render_caddy_block(
            site, self_host.Mode.DEV, self_host.Ports(server=38933, web_dev=38934), both_schemes=True
        )
        self.assertIn("https://game.example.test, http://game.example.test {", block)
        self.assertIn("@backend path /api /api/* /ws /health", block)
        self.assertNotIn("redir", block)

    def test_caddyfile_block_matches_tells_the_two_renderings_apart(self):
        import tempfile

        site = self_host.parse_site_url("https://game.example.test")
        ports = self_host.Ports(server=38933, web_dev=38934)
        with tempfile.TemporaryDirectory() as tmp:
            original = self_host.CADDYFILE
            self_host.CADDYFILE = Path(tmp) / "Caddyfile"
            try:
                self_host.update_caddyfile(site, self_host.Mode.PROD, ports)
                self.assertTrue(self_host.caddyfile_block_matches(site, self_host.Mode.PROD, ports))
                self.assertFalse(self_host.caddyfile_block_matches(site, self_host.Mode.PROD, ports, True))
                self_host.update_caddyfile(site, self_host.Mode.PROD, ports, True)
                self.assertTrue(self_host.caddyfile_block_matches(site, self_host.Mode.PROD, ports, True))
                self.assertEqual(self_host.CADDYFILE.read_text().count(self_host.MANAGED_START), 1)
            finally:
                self_host.CADDYFILE = original

    def test_update_caddyfile_inserts_and_replaces_managed_block(self, *, tmp_path=None):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            caddyfile = Path(tmp) / "Caddyfile"
            caddyfile.write_text("other-site.test {\n\treverse_proxy localhost:9000\n}\n")
            orig = self_host.CADDYFILE
            self_host.CADDYFILE = caddyfile
            try:
                site = self_host.parse_site_url("https://game.example.test")
                self_host.update_caddyfile(site, self_host.Mode.PROD, self_host.Ports(server=39001, web_dev=39002))
                text = caddyfile.read_text()
                self.assertIn("other-site.test", text)
                self.assertIn(self_host.MANAGED_START, text)
                self.assertIn("reverse_proxy localhost:39001", text)
                self.assertEqual(text.count(self_host.MANAGED_START), 1)

                # A second update replaces the managed block in place rather
                # than appending a second one.
                self_host.update_caddyfile(site, self_host.Mode.PROD, self_host.Ports(server=39003, web_dev=39004))
                text2 = caddyfile.read_text()
                self.assertEqual(text2.count(self_host.MANAGED_START), 1)
                self.assertIn("reverse_proxy localhost:39003", text2)
                self.assertNotIn("39001", text2)
                self.assertIn("other-site.test", text2)
            finally:
                self_host.CADDYFILE = orig

    def test_tmux_env_args_include_existing_proxy_vars(self):
        env = {
            "ALL_PROXY": "http://127.0.0.1:9999",
            "https_proxy": "http://127.0.0.1:9998",
            "UNRELATED": "ignored",
        }
        args = self_host.tmux_env_args(env)
        self.assertEqual(
            args,
            [
                "-e",
                "ALL_PROXY=http://127.0.0.1:9999",
                "-e",
                "https_proxy=http://127.0.0.1:9998",
            ],
        )

    def test_tmux_env_args_include_ajayeb_server_for_vite_proxy(self):
        args = self_host.tmux_env_args({"AJAYEB_SERVER": "http://127.0.0.1:39041"})
        self.assertEqual(args, ["-e", "AJAYEB_SERVER=http://127.0.0.1:39041"])

    def test_node_command_uses_node_24_via_nvm(self):
        command = self_host.node_command("pnpm -C web build")
        self.assertIn("nvm-load", command)
        self.assertIn("nvm use 24", command)
        self.assertTrue(command.endswith("&& pnpm -C web build"))

    def test_pnpm_install_command_is_noninteractive_without_dedupe_by_default(self):
        command = self_host.pnpm_install_command()
        self.assertEqual(command, "CI=true pnpm install --frozen-lockfile")

    def test_pnpm_install_command_can_include_setup_dedupe(self):
        command = self_host.pnpm_install_command(dedupe=True)
        self.assertEqual(command, "CI=true pnpm install --frozen-lockfile && CI=true pnpm dedupe")

    def test_config_roundtrip(self):
        cfg = self_host.Config(
            site=self_host.parse_site_url("http://lan.example.test"),
            mode=self_host.Mode.DEV,
            ports=self_host.Ports(server=38951, web_dev=38952),
        )
        payload = cfg.to_json()
        restored = self_host.Config.from_json(payload)
        self.assertEqual(restored.site.origin, "http://lan.example.test")
        self.assertEqual(restored.mode, self_host.Mode.DEV)
        self.assertEqual(restored.ports.server, 38951)
        self.assertEqual(restored.ports.web_dev, 38952)

    def test_config_roundtrip_keeps_both_schemes(self):
        cfg = self_host.Config(
            site=self_host.parse_site_url("https://game.example.test"), mode=self_host.Mode.PROD, both_schemes=True
        )
        self.assertTrue(self_host.Config.from_json(cfg.to_json()).both_schemes)

    def test_config_roundtrip_keeps_the_caddyfile_and_omits_the_default(self):
        site = self_host.parse_site_url("https://game.example.test")
        named = self_host.Config(site=site, mode=self_host.Mode.PROD, caddyfile="sites/Caddyfile")
        self.assertEqual(self_host.Config.from_json(named.to_json()).caddyfile, "sites/Caddyfile")
        plain = self_host.Config(site=site, mode=self_host.Mode.PROD)
        self.assertNotIn("caddyfile", plain.to_json())
        self.assertIsNone(self_host.Config.from_json(plain.to_json()).caddyfile)

    def test_legacy_config_serves_one_scheme(self):
        restored = self_host.Config.from_json('{"url": "https://game.example.test", "mode": "prod"}')
        self.assertFalse(restored.both_schemes)

    def test_legacy_config_uses_default_ports(self):
        restored = self_host.Config.from_json('{"url": "http://lan.example.test", "mode": "dev"}')
        self.assertEqual(restored.ports.server, self_host.DEFAULT_SERVER_PORT)
        self.assertEqual(restored.ports.web_dev, self_host.DEFAULT_WEB_DEV_PORT)

    def test_find_free_port_skips_busy_and_reserved_ports(self):
        reserved = {39001}
        self.assertEqual(self_host.find_free_port(39001, reserved=reserved, is_free=lambda port: port == 39003), 39003)

    def test_allocate_ports_reuses_free_configured_ports(self):
        ports = self_host.allocate_ports(self_host.Ports(server=39011, web_dev=39012), is_free=lambda port: True)
        self.assertEqual(ports.server, 39011)
        self.assertEqual(ports.web_dev, 39012)

    def test_allocate_ports_reselects_busy_configured_ports(self):
        ports = self_host.allocate_ports(
            self_host.Ports(server=39021, web_dev=39022),
            is_free=lambda port: port not in {39021, 39022},
        )
        self.assertEqual(ports.server, 39023)
        self.assertEqual(ports.web_dev, 39024)

    def test_web_dev_command_includes_selected_port_and_server_origin(self):
        command, env = self_host.web_dev_command(self_host.Ports(server=39031, web_dev=39032))
        self.assertIn("--port 39032", command)
        self.assertEqual(env["AJAYEB_SERVER"], "http://127.0.0.1:39031")

    def test_server_prod_command_sets_addr_data_origin_and_trust_proxy(self):
        site = self_host.parse_site_url("https://game.example.test")
        cmd = self_host.server_prod_command(self_host.Ports(server=39051, web_dev=39052), Path("/tmp/data"), site)
        self.assertIn("--addr", cmd)
        self.assertIn("127.0.0.1:39051", cmd)
        self.assertIn("--data", cmd)
        self.assertIn("/tmp/data", cmd)
        self.assertIn("--origin", cmd)
        self.assertIn("https://game.example.test", cmd)
        self.assertIn("--trust-proxy", cmd)
        self.assertIn("127.0.0.1", cmd)

    def test_server_prod_env_sets_gomemlimit(self):
        env = self_host.server_prod_env()
        self.assertEqual(env["GOMEMLIMIT"], "96MiB")

    def test_go_build_release_command_uses_embedweb_tag_and_new_binary_path(self):
        cmd = self_host.go_build_release_command("v1.2.3-dirty")
        self.assertIn("-tags", cmd)
        self.assertIn("embedweb", cmd)
        self.assertIn("-trimpath", cmd)
        self.assertIn(str(self_host.BIN_NEW_PATH), cmd)
        self.assertIn("./cmd/ajayeb", cmd)
        self.assertTrue(any("main.version=v1.2.3-dirty" in part for part in cmd))

    def test_announce_serving_prints_site_origin(self):
        cfg = self_host.Config(
            site=self_host.parse_site_url("https://game.example.test"),
            mode=self_host.Mode.PROD,
            ports=self_host.Ports(server=39051, web_dev=39052),
        )
        out = io.StringIO()
        with redirect_stdout(out):
            self_host.announce_serving(cfg)
        self.assertEqual(out.getvalue(), "serving: https://game.example.test\n")

    def test_apply_caddyfile_safely_validates_before_reload(self):
        import tempfile

        calls: list[str] = []

        with tempfile.TemporaryDirectory() as tmp:
            caddyfile = Path(tmp) / "Caddyfile"
            config_dir = Path(tmp) / ".self-host"

            orig_caddyfile = self_host.CADDYFILE
            orig_config_dir = self_host.CONFIG_DIR
            self_host.CADDYFILE = caddyfile
            self_host.CONFIG_DIR = config_dir

            def fake_validate():
                calls.append("validate")
                return type("R", (), {"returncode": 0, "stderr": ""})()

            def fake_reload():
                calls.append("reload")

            orig_validate = self_host.validate_caddyfile
            orig_reload = self_host.reload_caddy
            self_host.validate_caddyfile = fake_validate
            self_host.reload_caddy = fake_reload
            try:
                site = self_host.parse_site_url("https://game.example.test")
                self_host.apply_caddyfile_safely(site, self_host.Mode.PROD, self_host.Ports(server=39061, web_dev=39062))
                self.assertEqual(calls, ["validate", "reload"])
                self.assertIn("reverse_proxy localhost:39061", caddyfile.read_text())
            finally:
                self_host.CADDYFILE = orig_caddyfile
                self_host.CONFIG_DIR = orig_config_dir
                self_host.validate_caddyfile = orig_validate
                self_host.reload_caddy = orig_reload

    def test_apply_caddyfile_safely_restores_backup_and_raises_on_invalid_config(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            caddyfile = Path(tmp) / "Caddyfile"
            config_dir = Path(tmp) / ".self-host"
            original_text = "keep-me.test {\n\treverse_proxy localhost:9000\n}\n"
            caddyfile.write_text(original_text)

            orig_caddyfile = self_host.CADDYFILE
            orig_config_dir = self_host.CONFIG_DIR
            self_host.CADDYFILE = caddyfile
            self_host.CONFIG_DIR = config_dir

            reload_calls: list[str] = []

            def fake_validate():
                return type("R", (), {"returncode": 1, "stderr": "boom"})()

            def fake_reload():
                reload_calls.append("reload")

            orig_validate = self_host.validate_caddyfile
            orig_reload = self_host.reload_caddy
            self_host.validate_caddyfile = fake_validate
            self_host.reload_caddy = fake_reload
            try:
                site = self_host.parse_site_url("https://game.example.test")
                with self.assertRaises(RuntimeError):
                    self_host.apply_caddyfile_safely(site, self_host.Mode.PROD, self_host.Ports(server=39071, web_dev=39072))
                self.assertEqual(reload_calls, [])
                self.assertEqual(caddyfile.read_text(), original_text)
            finally:
                self_host.CADDYFILE = orig_caddyfile
                self_host.CONFIG_DIR = orig_config_dir
                self_host.validate_caddyfile = orig_validate
                self_host.reload_caddy = orig_reload

    def test_apply_caddyfile_safely_skips_an_already_matching_caddyfile(self):
        """A `start` at boot, re-run with nothing to change, must leave
        ~/Caddyfile alone entirely: no backup, no write, no validate, no
        reload."""
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            caddyfile = Path(tmp) / "Caddyfile"
            config_dir = Path(tmp) / ".self-host"

            orig_caddyfile = self_host.CADDYFILE
            orig_config_dir = self_host.CONFIG_DIR
            self_host.CADDYFILE = caddyfile
            self_host.CONFIG_DIR = config_dir

            site = self_host.parse_site_url("https://game.example.test")
            ports = self_host.Ports(server=39081, web_dev=39082)
            self_host.update_caddyfile(site, self_host.Mode.PROD, ports)
            original_text = caddyfile.read_text()
            original_mtime_ns = caddyfile.stat().st_mtime_ns

            calls: list[str] = []
            orig_backup = self_host.backup_caddyfile
            orig_validate = self_host.validate_caddyfile
            orig_reload = self_host.reload_caddy
            self_host.backup_caddyfile = lambda: calls.append("backup")
            self_host.validate_caddyfile = lambda: calls.append("validate")
            self_host.reload_caddy = lambda: calls.append("reload")
            try:
                self_host.apply_caddyfile_safely(site, self_host.Mode.PROD, ports)
                self.assertEqual(calls, [])
                self.assertEqual(caddyfile.read_text(), original_text)
                self.assertEqual(caddyfile.stat().st_mtime_ns, original_mtime_ns)
                self.assertFalse(config_dir.exists(), "no backup file (or directory) should be created")
            finally:
                self_host.CADDYFILE = orig_caddyfile
                self_host.CONFIG_DIR = orig_config_dir
                self_host.backup_caddyfile = orig_backup
                self_host.validate_caddyfile = orig_validate
                self_host.reload_caddy = orig_reload

    def test_apply_caddyfile_safely_continues_when_reload_fails(self):
        """`caddy reload` failing (for example, Caddy is not up yet at boot)
        must not abort start/setup/redeploy: it only warns."""
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            caddyfile = Path(tmp) / "Caddyfile"
            config_dir = Path(tmp) / ".self-host"

            orig_caddyfile = self_host.CADDYFILE
            orig_config_dir = self_host.CONFIG_DIR
            self_host.CADDYFILE = caddyfile
            self_host.CONFIG_DIR = config_dir

            def fake_validate():
                return type("R", (), {"returncode": 0, "stderr": ""})()

            def fake_reload():
                raise subprocess.CalledProcessError(1, ["caddy", "reload"])

            orig_validate = self_host.validate_caddyfile
            orig_reload = self_host.reload_caddy
            self_host.validate_caddyfile = fake_validate
            self_host.reload_caddy = fake_reload
            try:
                site = self_host.parse_site_url("https://game.example.test")
                ports = self_host.Ports(server=39091, web_dev=39092)
                stderr = io.StringIO()
                with redirect_stderr(stderr):
                    self_host.apply_caddyfile_safely(site, self_host.Mode.PROD, ports)  # must not raise
                self.assertIn("reverse_proxy localhost:39091", caddyfile.read_text())
                self.assertIn("warning", stderr.getvalue().lower())
                self.assertIn("reload", stderr.getvalue().lower())
            finally:
                self_host.CADDYFILE = orig_caddyfile
                self_host.CONFIG_DIR = orig_config_dir
                self_host.validate_caddyfile = orig_validate
                self_host.reload_caddy = orig_reload

    def test_reload_caddy_only_runs_caddy_reload(self):
        """No `caddy fmt`: it would reformat the whole shared ~/Caddyfile."""
        with mock.patch("self_host.subprocess.run") as run:
            run.return_value = subprocess.CompletedProcess([], returncode=0)
            self_host.reload_caddy()
            run.assert_called_once_with(
                ["caddy", "reload", "--config", str(self_host.CADDYFILE), "--adapter", "caddyfile"],
                cwd=self_host.ROOT,
                check=True,
                text=True,
            )

    def test_archive_rooms_keeps_only_last_ten(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            config_dir = Path(tmp) / ".self-host"
            data_dir = config_dir / "data"
            backups_dir = config_dir / "backups"
            rooms_dir = data_dir / "rooms"
            rooms_dir.mkdir(parents=True)
            (rooms_dir / "ABCDEF.json").write_text("{}")

            orig_data_dir = self_host.DATA_DIR
            orig_backups_dir = self_host.BACKUPS_DIR
            self_host.DATA_DIR = data_dir
            self_host.BACKUPS_DIR = backups_dir
            try:
                for i in range(12):
                    # Distinct names so sorting is deterministic without
                    # depending on filesystem mtime resolution.
                    archive = backups_dir / f"rooms-2026010100000{i:02d}.tar.gz"
                    backups_dir.mkdir(parents=True, exist_ok=True)
                    archive.write_bytes(b"")
                self_host.prune_room_backups()
                remaining = sorted(p.name for p in backups_dir.glob("rooms-*.tar.gz"))
                self.assertEqual(len(remaining), self_host.ROOMS_BACKUP_KEEP)
                self.assertEqual(remaining[0], "rooms-202601010000002.tar.gz")
                self.assertEqual(remaining[-1], "rooms-202601010000011.tar.gz")

                archived = self_host.archive_rooms()
                self.assertIsNotNone(archived)
                self.assertTrue(archived.exists())
                remaining2 = sorted(backups_dir.glob("rooms-*.tar.gz"))
                self.assertEqual(len(remaining2), self_host.ROOMS_BACKUP_KEEP)
            finally:
                self_host.DATA_DIR = orig_data_dir
                self_host.BACKUPS_DIR = orig_backups_dir

    def test_room_backups_prune_by_age_and_count_with_clock(self):
        now = 2_000_000_000.0
        day = 24 * 60 * 60
        with tempfile.TemporaryDirectory() as tmp:
            backups = Path(tmp)
            old = backups / "rooms-old.tar.gz"
            boundary = backups / "rooms-boundary.tar.gz"
            middle = backups / "rooms-middle.tar.gz"
            fresh = backups / "rooms-fresh.tar.gz"
            unrelated = backups / "unrelated.tar.gz"
            directory = backups / "rooms-directory.tar.gz"
            directory.mkdir()
            for path, age in ((old, 31*day), (boundary, 30*day), (middle, day), (fresh, 0), (unrelated, 50*day)):
                path.write_bytes(b"backup")
                os.utime(path, (now-age, now-age))
            with mock.patch.object(self_host, "BACKUPS_DIR", backups):
                self_host.prune_room_backups(now=now)
                self.assertFalse(old.exists())
                self.assertTrue(boundary.exists())
                self_host.prune_room_backups(keep=2, now=now)
                self.assertFalse(boundary.exists())
                self.assertTrue(middle.exists())
                self.assertTrue(fresh.exists())
                self.assertTrue(unrelated.exists())
                self.assertTrue(directory.is_dir())
                self_host.prune_room_backups(now=now+31*day)
                self.assertFalse(middle.exists())
                self.assertFalse(fresh.exists())

    def test_room_backup_age_pruning_runs_without_live_rooms(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            backups = root / "backups"
            backups.mkdir()
            stale = backups / "rooms-stale.tar.gz"
            stale.write_bytes(b"old backup")
            os.utime(stale, (1, 1))
            # Only diagnostic archives remain. Do not duplicate these into a
            # deployment backup, but still expire the existing backups.
            archive = root / "data" / "rooms-archive"
            archive.mkdir(parents=True)
            record = archive / "ARCHVD.json"
            record.write_bytes(b"diagnostic record")
            with mock.patch.object(self_host, "BACKUPS_DIR", backups), mock.patch.object(self_host, "DATA_DIR", root / "data"):
                self.assertIsNone(self_host.archive_rooms())
            self.assertFalse(stale.exists())
            self.assertEqual(record.read_bytes(), b"diagnostic record")

    def test_archive_rooms_skips_when_no_rooms(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            data_dir = Path(tmp) / "data"
            backups_dir = Path(tmp) / "backups"
            orig_data_dir = self_host.DATA_DIR
            orig_backups_dir = self_host.BACKUPS_DIR
            self_host.DATA_DIR = data_dir
            self_host.BACKUPS_DIR = backups_dir
            try:
                self.assertIsNone(self_host.archive_rooms())
            finally:
                self_host.DATA_DIR = orig_data_dir
                self_host.BACKUPS_DIR = orig_backups_dir

    def test_poll_health_returns_true_when_server_responds(self):
        import http.server
        import threading

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"{}")

            def log_message(self, *args):
                pass

        server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
        port = server.server_address[1]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            self.assertTrue(self_host.poll_health(port, timeout=2.0, interval=0.05))
        finally:
            server.shutdown()
            thread.join(timeout=2)

    def test_poll_health_returns_false_when_nothing_listens(self):
        self.assertFalse(self_host.poll_health(self_host.find_free_port(45000), timeout=0.3, interval=0.05))

    def test_tmux_tail_returns_none_for_missing_session(self):
        self.assertIsNone(self_host.tmux_tail("ajayeb-self-host-test-missing-session"))

    def test_config_roundtrip_preserves_ship_target(self):
        cfg = self_host.Config(
            site=self_host.parse_site_url("https://game.example.test"),
            mode=self_host.Mode.PROD,
            ship=self_host.ShipTarget("user@example.host", "a-dir"),
        )
        restored = self_host.Config.from_json(cfg.to_json())
        self.assertEqual(restored.ship, self_host.ShipTarget("user@example.host", "a-dir"))

    def test_config_without_ship_omits_it_from_json(self):
        cfg = self_host.Config(site=self_host.parse_site_url(None), mode=self_host.Mode.PROD)
        self.assertNotIn("ship", cfg.to_json())
        self.assertIsNone(self_host.Config.from_json(cfg.to_json()).ship)

    def test_help_lists_ship_command(self):
        help_text = self_host.build_parser().format_help()
        self.assertIn("ship", help_text)

    def test_parser_parses_setup_with_prebuilt(self):
        args = self_host.build_parser().parse_args(["setup", "https://x.test", "--prebuilt", "/tmp/bin"])
        self.assertEqual(args.command, "setup")
        self.assertEqual(args.url, "https://x.test")
        self.assertEqual(args.prebuilt, "/tmp/bin")

    def test_parser_parses_redeploy_with_prebuilt_and_no_url(self):
        args = self_host.build_parser().parse_args(["redeploy", "--prebuilt", "/tmp/bin"])
        self.assertIsNone(args.url)
        self.assertEqual(args.prebuilt, "/tmp/bin")

    def test_parser_start_has_no_prebuilt_option(self):
        args = self_host.build_parser().parse_args(["start"])
        self.assertFalse(hasattr(args, "prebuilt"))

    def test_parser_both_schemes_is_three_way(self):
        parser = self_host.build_parser()
        for command in ("setup", "redeploy", "start", "dev-start", "ship"):
            self.assertIsNone(parser.parse_args([command]).both_schemes, command)
            self.assertTrue(parser.parse_args([command, "--both-schemes"]).both_schemes, command)
            self.assertFalse(parser.parse_args([command, "--no-both-schemes"]).both_schemes, command)

    def test_parser_ship_target_url_and_remote_dir_are_optional(self):
        args = self_host.build_parser().parse_args(["ship"])
        self.assertIsNone(args.target)
        self.assertIsNone(args.url)
        self.assertIsNone(args.remote_dir)

    def test_parser_parses_ship_with_all_arguments(self):
        args = self_host.build_parser().parse_args(
            ["ship", "user@example.host", "https://x.test", "--remote-dir", "custom"]
        )
        self.assertEqual(args.target, "user@example.host")
        self.assertEqual(args.url, "https://x.test")
        self.assertEqual(args.remote_dir, "custom")


def fake_adapt(text: bytes) -> dict:
    """What the fake Caddy adapts Caddyfile text to: the text itself, and two
    file_servers whose hide lists end with the stand-in the real adapter
    writes for the config file when the text comes with no file name."""
    route = {"handle": [{"handler": "file_server", "hide": ["*.bak", self_host.ADAPTED_TEXT_HIDE]}]}
    return {"apps": {"http": {"servers": {"s": {"routes": [route, json.loads(json.dumps(route))]}}}}, "text": text.decode()}


def served_from(text: str, hide_path: str = "/srv/conf/Caddyfile") -> dict:
    """The config `caddy run --config <hide_path>` serves for text."""
    return self_host.with_config_file_hidden(fake_adapt(text.encode()), hide_path)


class FakeCaddyAdmin:
    """A local HTTP server standing in for Caddy's admin API. It holds a
    running config (`running`, None for none), answers GET /config/ with it,
    POST /adapt with {"result": fake_adapt(body)} (or an error status when
    `adapt_rejects(body)` is true), and POST /load by applying the JSON body,
    or answering `load_status` with `load_body` when that is not 200.
    `drop_load` loses the reply to /load, after applying it ("after") or
    without applying it ("before"). Every request is recorded."""

    def __init__(self, running=None, load_status=200, load_body=b"", adapt_rejects=None, drop_load=None):
        self.running = running
        self.load_status, self.load_body = load_status, load_body
        self.adapt_rejects = adapt_rejects or (lambda body: False)
        self.drop_load = drop_load
        self.requests = []
        outer = self

        class Handler(BaseHTTPRequestHandler):
            def _answer(self, status, body):
                self.send_response(status)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def do_GET(self):
                outer.requests.append(("GET", self.path, dict(self.headers), b""))
                self._answer(200, json.dumps(outer.running).encode() + b"\n")

            def do_POST(self):
                data = self.rfile.read(int(self.headers.get("Content-Length", 0)))
                outer.requests.append(("POST", self.path, dict(self.headers), data))
                if self.path == "/adapt":
                    if outer.adapt_rejects(data):
                        self._answer(400, b'{"error":"adapting config: boom"}')
                    else:
                        self._answer(200, json.dumps({"result": fake_adapt(data)}).encode())
                elif self.path == "/load":
                    if outer.drop_load:
                        if outer.drop_load == "after":
                            outer.running = json.loads(data)
                        self.close_connection = True
                        return
                    if outer.load_status == 200:
                        outer.running = json.loads(data)
                    self._answer(outer.load_status, outer.load_body)
                else:
                    self._answer(404, b"")

            def log_message(self, *args):
                pass

        self.server = HTTPServer(("127.0.0.1", 0), Handler)
        self.address = f"127.0.0.1:{self.server.server_port}"
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    def paths(self):
        return [(method, path) for method, path, _headers, _body in self.requests]

    def __enter__(self):
        self.thread.start()
        return self

    def __exit__(self, *exc):
        self.server.shutdown()
        self.server.server_close()


def closed_address() -> str:
    """A 127.0.0.1 port nothing listens on."""
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return f"127.0.0.1:{sock.getsockname()[1]}"


# A proxy that would swallow every request: the admin API calls must bypass it.
DEAD_PROXY_ENV = {
    "http_proxy": "http://127.0.0.1:9",
    "HTTP_PROXY": "http://127.0.0.1:9",
    "no_proxy": "",
    "NO_PROXY": "",
}


def setUpModule():
    # No test may reach a real Caddy or read a real Caddyfile (a developer
    # machine may run one): Caddy's default admin address points at a dead
    # port, and the default Caddyfile at a path that does not exist.
    nowhere = Path(tempfile.mkdtemp()) / "no-such-Caddyfile"
    for name, value in (
        ("CADDY_DEFAULT_ADMIN", closed_address()),
        ("DEFAULT_CADDYFILE", nowhere),
        ("CADDYFILE", nowhere),
    ):
        patcher = mock.patch.object(self_host, name, value)
        patcher.start()
        unittest.addModuleCleanup(patcher.stop)


# setUp below replaces validate_caddyfile; this is the real one.
REAL_VALIDATE_CADDYFILE = self_host.validate_caddyfile


class CaddyLoadTests(unittest.TestCase):
    """The running Caddy loads the edited Caddyfile only when it serves the
    file as it stood (load_through_caddy), and a Caddyfile that does not
    exist is never created over a Caddy that serves other sites
    (refuse_to_create_caddyfile)."""

    OTHER_SITE = "other-site.test {\n\treverse_proxy localhost:9000\n}\n"

    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.tmp = Path(tmp.name)
        self.caddyfile = self.tmp / "Caddyfile"
        self.config_dir = self.tmp / ".self-host"
        for name, value in (
            ("CADDYFILE", self.caddyfile),
            ("CONFIG_DIR", self.config_dir),
            ("CADDYFILE_NAMED", False),
            ("CADDY_RECHECK_DELAY", 0.0),
        ):
            patcher = mock.patch.object(self_host, name, value)
            patcher.start()
            self.addCleanup(patcher.stop)
        self.calls = []
        validate = mock.patch.object(self_host, "validate_caddyfile", side_effect=self._validate)
        reload = mock.patch.object(self_host, "reload_caddy", side_effect=lambda: self.calls.append("reload"))
        for patcher in (validate, reload):
            patcher.start()
            self.addCleanup(patcher.stop)
        self.validate_rc = 0
        self.site = self_host.parse_site_url("https://game.example.test")
        self.ports = self_host.Ports(server=39111, web_dev=39112)

    def _validate(self):
        self.calls.append("validate")
        return subprocess.CompletedProcess([], self.validate_rc, stdout="", stderr="bad")

    def _apply(self):
        self_host.apply_caddyfile_safely(self.site, self_host.Mode.PROD, self.ports)

    def _apply_through(self, admin):
        with mock.patch.object(self_host, "caddy_admin_address", return_value=admin.address):
            self._apply()

    # The load path.

    def test_apply_loads_the_edit_as_caddy_adapts_it_with_the_file_hidden_as_before(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        with FakeCaddyAdmin(running=served_from(self.OTHER_SITE)) as admin, mock.patch.dict(os.environ, DEAD_PROXY_ENV):
            self._apply_through(admin)
        edited = self.caddyfile.read_text()
        self.assertIn("reverse_proxy localhost:39111", edited)
        self.assertIn("other-site.test", edited)
        self.assertEqual(
            admin.paths(), [("GET", "/config/"), ("POST", "/adapt"), ("POST", "/adapt"), ("POST", "/load")]
        )
        _, _, adapt_headers, before = admin.requests[1]
        self.assertEqual(adapt_headers["Content-Type"], "text/caddyfile")
        self.assertEqual(before, self.OTHER_SITE.encode())
        self.assertEqual(admin.requests[2][3], edited.encode())
        _, _, load_headers, loaded = admin.requests[3]
        self.assertEqual(load_headers["Content-Type"], "application/json")
        self.assertEqual(load_headers["Cache-Control"], "must-revalidate")
        self.assertEqual(json.loads(loaded), served_from(edited))
        self.assertNotIn(self_host.ADAPTED_TEXT_HIDE, loaded.decode())
        self.assertEqual(admin.running, served_from(edited))
        self.assertEqual(self.calls, [], "no caddy validate or caddy reload when Caddy loads the file itself")

    def test_apply_refuses_a_caddy_that_does_not_serve_the_file_as_it_stands(self):
        """A wrong path, another Caddy on the address, relative imports or
        unloaded edits all show up as a running config that differs."""
        self.caddyfile.write_text(self.OTHER_SITE)
        with FakeCaddyAdmin(running=served_from("another-file.test {\n}\n")) as admin:
            with self.assertRaises(RuntimeError) as caught:
                self._apply_through(admin)
        self.assertIn("--caddyfile", str(caught.exception))
        self.assertIn("Nothing was loaded", str(caught.exception))
        self.assertNotIn(("POST", "/load"), admin.paths())
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)
        self.assertEqual(self.calls, [])

    def test_apply_refuses_when_the_running_config_hides_two_different_config_paths(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        running = served_from(self.OTHER_SITE)
        running["apps"]["http"]["servers"]["s"]["routes"][1]["handle"][0]["hide"][1] = "/elsewhere/Caddyfile"
        with FakeCaddyAdmin(running=running) as admin:
            with self.assertRaises(RuntimeError):
                self._apply_through(admin)
        self.assertNotIn(("POST", "/load"), admin.paths())
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)

    def test_apply_restores_the_caddyfile_when_caddy_refuses_the_load(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        running = served_from(self.OTHER_SITE)
        with FakeCaddyAdmin(running=running, load_status=400, load_body=b"provision http: boom") as admin:
            with self.assertRaises(RuntimeError) as caught:
                self._apply_through(admin)
        self.assertIn("boom", str(caught.exception))
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)
        self.assertEqual(admin.running, running)
        self.assertEqual(self.calls, [])

    def test_apply_restores_the_caddyfile_when_caddy_cannot_adapt_the_edit(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        rejects_ours = lambda body: self_host.MANAGED_START.encode() in body
        with FakeCaddyAdmin(running=served_from(self.OTHER_SITE), adapt_rejects=rejects_ours) as admin:
            with self.assertRaises(RuntimeError) as caught:
                self._apply_through(admin)
        self.assertIn("boom", str(caught.exception))
        self.assertNotIn(("POST", "/load"), admin.paths())
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)

    def test_apply_accepts_a_load_whose_reply_was_lost_after_caddy_applied_it(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        with FakeCaddyAdmin(running=served_from(self.OTHER_SITE), drop_load="after") as admin:
            self._apply_through(admin)
        edited = self.caddyfile.read_text()
        self.assertIn("reverse_proxy localhost:39111", edited)
        self.assertEqual(admin.running, served_from(edited))
        self.assertEqual(admin.paths()[-1], ("GET", "/config/"))
        self.assertEqual(self.calls, [], "a running Caddy is never reported as down, and validate never runs")

    def test_apply_restores_the_caddyfile_when_a_lost_reply_left_caddy_on_its_old_config(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        running = served_from(self.OTHER_SITE)
        with FakeCaddyAdmin(running=running, drop_load="before") as admin:
            with self.assertRaises(RuntimeError) as caught:
                self._apply_through(admin)
        self.assertIn("lost", str(caught.exception))
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)
        self.assertEqual(admin.running, running)
        self.assertEqual(self.calls, [])

    def test_a_lost_load_reply_with_no_answer_to_the_recheck_restores_and_never_validates(self):
        """Whether the recheck is refused, refused with an error status or
        lost again, the outcome is unknown: restore and stop, and never
        report Caddy as down or as having kept its config."""
        real_request = self_host.caddy_admin_request
        for failure in (self_host.CaddyUnreachable, self_host.CaddyRejected, self_host.CaddyUnsure):
            self.caddyfile.write_text(self.OTHER_SITE)
            gets = []

            def request(address, path, **kwargs):
                if path == "/config/":
                    gets.append(path)
                    if len(gets) == 2:
                        raise failure("recheck failed")
                return real_request(address, path, **kwargs)

            with self.subTest(failure=failure.__name__), FakeCaddyAdmin(
                running=served_from(self.OTHER_SITE), drop_load="after"
            ) as admin, mock.patch.object(self_host, "caddy_admin_request", side_effect=request):
                with self.assertRaises(RuntimeError) as caught:
                    self._apply_through(admin)
                self.assertIn("could not be asked", str(caught.exception))
                self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)
                self.assertEqual(self.calls, [])

    def test_apply_matches_hide_lists_in_any_order(self):
        """Caddy orders the config files in a hide list differently across
        versions; the same paths in another order are the same config."""
        self.caddyfile.write_text(self.OTHER_SITE)
        running = served_from(self.OTHER_SITE)
        for route in running["apps"]["http"]["servers"]["s"]["routes"]:
            route["handle"][0]["hide"].reverse()
        with FakeCaddyAdmin(running=running) as admin:
            self._apply_through(admin)
        self.assertEqual(admin.paths()[-1], ("POST", "/load"))
        self.assertTrue(self_host.same_config(admin.running, served_from(self.caddyfile.read_text())))

    def test_apply_restores_the_caddyfile_on_an_interrupt(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        with mock.patch.object(self_host, "caddy_admin_address", return_value=closed_address()), mock.patch.object(
            self_host, "load_through_caddy", side_effect=KeyboardInterrupt
        ):
            with self.assertRaises(KeyboardInterrupt):
                self._apply()
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)

    # The preflight, before anything is stopped.

    def test_preflight_refuses_a_caddy_that_does_not_serve_the_file_while_nothing_is_stopped(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        with FakeCaddyAdmin(running=served_from("another-file.test {\n}\n")) as admin, mock.patch.object(
            self_host, "caddy_admin_address", return_value=admin.address
        ):
            with self.assertRaisesRegex(RuntimeError, "Nothing was stopped"):
                self_host.preflight_caddyfile(self.site, self_host.Mode.PROD, self.ports)
        self.assertNotIn(("POST", "/load"), admin.paths())
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)

    def test_preflight_passes_a_caddy_that_serves_the_file_and_changes_nothing(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        with FakeCaddyAdmin(running=served_from(self.OTHER_SITE)) as admin, mock.patch.object(
            self_host, "caddy_admin_address", return_value=admin.address
        ):
            self_host.preflight_caddyfile(self.site, self_host.Mode.PROD, self.ports)
        self.assertEqual(admin.paths(), [("GET", "/config/"), ("POST", "/adapt")])
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)

    def test_preflight_leaves_a_matching_block_a_new_file_and_a_stopped_caddy_alone(self):
        address = mock.Mock(return_value=closed_address())
        with mock.patch.object(self_host, "caddy_admin_address", address):
            self_host.preflight_caddyfile(self.site, self_host.Mode.PROD, self.ports)  # no file yet
            address.assert_not_called()
            self_host.update_caddyfile(self.site, self_host.Mode.PROD, self.ports)
            self_host.preflight_caddyfile(self.site, self_host.Mode.PROD, self.ports)  # block matches
            address.assert_not_called()
            self_host.preflight_caddyfile(self.site, self_host.Mode.PROD, replace(self.ports, server=39113))
            address.assert_called_once()  # block differs, but no Caddy answers: nothing raised

    def test_production_commands_retain_preflight_before_stop_and_continue_after_refusal(self):
        config = self_host.Config(site=self.site, mode=self_host.Mode.PROD, ports=self.ports)
        runs = [
            lambda: self_host.command_setup(None, "/tmp/bin"),
            lambda: self_host.command_redeploy(None, "/tmp/bin"),
            lambda: self_host.command_start(None),
        ]
        for run_command in runs:
            order = []
            failure = RuntimeError("refused. Nothing was stopped or changed")
            def preflight(*args):
                order.append("preflight")
                raise failure
            with self.subTest(command=run_command), mock.patch.multiple(
                self_host,
                resolve_config=mock.Mock(return_value=config),
                refuse_to_create_caddyfile=mock.DEFAULT,
                ensure_tools=mock.DEFAULT,
                check_prebuilt_binary=mock.DEFAULT,
            check_candidate_rooms=mock.DEFAULT,
            wait_for_port_free=mock.Mock(return_value=True),
                preflight_caddyfile=mock.Mock(side_effect=preflight),
                command_stop=mock.Mock(side_effect=lambda: order.append("stop")),
                ports_after_stop=mock.Mock(return_value=config),
                save_config=mock.DEFAULT,
                swap_in_binary=mock.DEFAULT,
                archive_rooms=mock.Mock(return_value=None),
                start_prod_with_caddy=mock.DEFAULT,
            ) as mocks, mock.patch.object(self_host, "BIN_PATH", Path(__file__)), redirect_stderr(io.StringIO()):
                run_command()
                self.assertEqual(order, ["preflight", "stop"])
                self_host.preflight_caddyfile.assert_called_once_with(config.site, self_host.Mode.PROD, config.ports, False)
                mocks["start_prod_with_caddy"].assert_called_once_with(config, failure)

    def test_apply_still_hands_caddy_a_file_that_does_not_adapt_here(self):
        """The case the load path exists for: `caddy adapt` fails in this
        shell over a placeholder, and the running Caddy still gets the file."""
        with FakeCaddyAdmin() as admin:
            original = f"{{\n\tadmin {admin.address}\n}}\n" + self.OTHER_SITE
            self.caddyfile.write_text(original)
            admin.running = served_from(original)
            failed_adapt = subprocess.CompletedProcess([], 1, stdout="", stderr="Error: password is required")
            with mock.patch("self_host.subprocess.run", return_value=failed_adapt):
                self._apply()
        self.assertEqual(admin.paths()[-1], ("POST", "/load"))
        self.assertEqual(self.calls, [])
        self.assertEqual(admin.running, served_from(self.caddyfile.read_text()))

    def test_a_new_caddyfile_is_loaded_only_into_a_caddy_with_no_config(self):
        with FakeCaddyAdmin(running=None) as admin:
            self._apply_through(admin)
        self.assertEqual(admin.paths(), [("GET", "/config/"), ("POST", "/adapt"), ("POST", "/load")])
        self.assertEqual(admin.running, served_from(self.caddyfile.read_text(), str(self.caddyfile)))
        self.caddyfile.unlink()
        with FakeCaddyAdmin(running={"apps": {"http": {}}}) as admin:
            with self.assertRaises(RuntimeError):
                self._apply_through(admin)
        self.assertNotIn(("POST", "/load"), admin.paths())
        self.assertFalse(self.caddyfile.exists())

    def test_apply_validates_when_no_caddy_answers(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        stderr = io.StringIO()
        with mock.patch.object(self_host, "caddy_admin_address", return_value=closed_address()), redirect_stderr(stderr):
            self._apply()
        self.assertEqual(self.calls, ["validate"])
        self.assertIn("reverse_proxy localhost:39111", self.caddyfile.read_text())
        self.assertIn("next start", stderr.getvalue())

    def test_apply_restores_and_hints_when_validate_fails_on_a_placeholder(self):
        original = "{$SECRET_PLACEHOLDER}\n" + self.OTHER_SITE
        self.caddyfile.write_text(original)
        self.validate_rc = 1
        with mock.patch.object(self_host, "caddy_admin_address", return_value=closed_address()), redirect_stderr(io.StringIO()):
            with self.assertRaises(RuntimeError) as caught:
                self._apply()
        self.assertEqual(self.caddyfile.read_text(), original)
        self.assertIn("{$VAR}", str(caught.exception))

    def test_apply_restores_the_caddyfile_on_an_unexpected_error_after_the_write(self):
        self.caddyfile.write_text(self.OTHER_SITE)
        with mock.patch.object(self_host, "caddy_admin_address", return_value=closed_address()), mock.patch.object(
            self_host, "load_through_caddy", side_effect=ValueError("surprise")
        ):
            with self.assertRaises(ValueError):
                self._apply()
        self.assertEqual(self.caddyfile.read_text(), self.OTHER_SITE)

    # The pieces.

    def test_admin_requests_bypass_any_proxy_and_classify_failures(self):
        with mock.patch.dict(os.environ, DEAD_PROXY_ENV):
            with FakeCaddyAdmin(running={"apps": {}}) as admin:
                self.assertEqual(self_host.caddy_admin_request(admin.address, "/config/"), {"apps": {}})
            with FakeCaddyAdmin(load_status=400, load_body=b'{"error":"boom"}') as admin:
                with self.assertRaises(self_host.CaddyRejected) as caught:
                    self_host.caddy_admin_request(admin.address, "/load", data=b"{}")
                self.assertIn("boom", str(caught.exception))
            with FakeCaddyAdmin(drop_load="before") as admin:
                with self.assertRaises(self_host.CaddyUnsure):
                    self_host.caddy_admin_request(admin.address, "/load", data=b"{}")
            with self.assertRaises(self_host.CaddyUnreachable):
                self_host.caddy_admin_request(closed_address(), "/config/")
        timed_out = mock.Mock()
        timed_out.open.side_effect = urllib.error.URLError(TimeoutError("timed out"))
        with mock.patch.object(self_host, "caddy_admin_opener", return_value=timed_out):
            with self.assertRaises(self_host.CaddyUnsure):
                self_host.caddy_admin_request("localhost:1", "/load", data=b"{}")
        # localhost can resolve to ::1 on a host without IPv6, and Python
        # reports the last address's error: still nothing listening.
        for code in (errno.EINVAL, errno.EADDRNOTAVAIL, errno.ENETUNREACH):
            unusable = mock.Mock()
            unusable.open.side_effect = urllib.error.URLError(OSError(code, os.strerror(code)))
            with self.subTest(errno=code), mock.patch.object(self_host, "caddy_admin_opener", return_value=unusable):
                with self.assertRaises(self_host.CaddyUnreachable):
                    self_host.caddy_admin_request("localhost:1", "/config/")
        dead_port = closed_address().split(":")[1]
        with self.assertRaises(self_host.CaddyUnreachable):
            self_host.caddy_admin_request(f"localhost:{dead_port}", "/config/")

    def test_hide_paths_and_putting_them_back(self):
        hidden = served_from("x", "/srv/conf/Caddyfile")
        self.assertEqual(self_host.hide_paths_for_adapted_text(fake_adapt(b"x"), hidden, set()), {"/srv/conf/Caddyfile"})
        self.assertEqual(self_host.with_config_file_hidden(fake_adapt(b"x"), "/srv/conf/Caddyfile"), hidden)
        plain = {"handler": "reverse_proxy", "hide": [self_host.ADAPTED_TEXT_HIDE]}
        self.assertEqual(self_host.with_config_file_hidden(plain, "/p"), plain, "only file_server hide lists change")

    def test_serves_a_config_only_when_a_caddy_has_one_loaded(self):
        with mock.patch.dict(os.environ, DEAD_PROXY_ENV):
            with FakeCaddyAdmin(running={"apps": {}}) as admin:
                self.assertTrue(self_host.caddy_serves_a_config(admin.address))
                self.assertEqual(admin.paths(), [("GET", "/config/")])
            for empty in (None, {}):
                with FakeCaddyAdmin(running=empty) as admin:
                    self.assertFalse(self_host.caddy_serves_a_config(admin.address))
            self.assertFalse(self_host.caddy_serves_a_config(closed_address()))

    def test_admin_address_comes_from_the_adapted_caddyfile(self):
        cases = [
            ("{}", "localhost:2019"),
            ('{"admin":{"listen":"localhost:2999"}}', "localhost:2999"),
            ('{"admin":{"listen":"tcp/127.0.0.1:2999"}}', "127.0.0.1:2999"),
            ('{"admin":{"listen":"unix//run/caddy.sock"}}', None),
            ('{"admin":{"disabled":true}}', None),
        ]
        with mock.patch.object(self_host, "CADDY_DEFAULT_ADMIN", "localhost:2019"):
            for stdout, want in cases:
                with self.subTest(stdout=stdout), mock.patch("self_host.subprocess.run") as run:
                    run.return_value = subprocess.CompletedProcess([], 0, stdout=stdout, stderr="")
                    self.assertEqual(self_host.caddy_admin_address(), want)
                    self.assertEqual(
                        run.call_args.args[0],
                        ["caddy", "adapt", "--config", str(self.caddyfile), "--adapter", "caddyfile"],
                    )

    def test_admin_address_is_read_as_text_when_the_caddyfile_does_not_adapt_here(self):
        """An unset {$VAR} can make `caddy adapt` fail in this shell while the
        running Caddy, which has it, would load the file."""
        cases = [
            ("{\n\tadmin localhost:2999\n}\n", "localhost:2999"),
            ("{\n\temail ops@example.test\n}\n" + self.OTHER_SITE, "localhost:2019"),
            ("{\n\tadmin off\n}\n", None),
        ]
        with mock.patch.object(self_host, "CADDY_DEFAULT_ADMIN", "localhost:2019"):
            for text, want in cases:
                self.caddyfile.write_text(text)
                for adapt in (
                    subprocess.CompletedProcess([], 1, stdout="", stderr="Error: bad"),
                    subprocess.CompletedProcess([], 0, stdout="not json", stderr=""),
                ):
                    with self.subTest(text=text, rc=adapt.returncode), mock.patch(
                        "self_host.subprocess.run", return_value=adapt
                    ):
                        self.assertEqual(self_host.caddy_admin_address(), want)

    def test_scanned_admin_address_reads_only_the_global_options_block(self):
        cases = [
            ("", "localhost:2019"),
            (self.OTHER_SITE, "localhost:2019"),
            ("# a comment\n{\n\tadmin localhost:2999\n}\n", "localhost:2999"),
            ("{\n\tadmin tcp/127.0.0.1:2999 {\n\t\torigins localhost\n\t}\n}\n", "127.0.0.1:2999"),
            ("{\n\tadmin {\n\t\torigins localhost\n\t}\n}\n", "localhost:2019"),
            ("{\n\tadmin unix//run/caddy.sock\n}\n", None),
            ("{\n\tadmin {$ADMIN_ADDRESS}\n}\n", None),
            ("{\n\tadmin off\n}\n", None),
            ("{\n\t# admin off\n\tdebug\n}\n", "localhost:2019"),
            ("{\n\tservers {\n\t\tadmin nope:1\n\t}\n}\n" + self.OTHER_SITE, "localhost:2019"),
            ("{\n\tdebug\n}\nsite.test {\n\tadmin nope:1\n}\n", "localhost:2019"),
        ]
        with mock.patch.object(self_host, "CADDY_DEFAULT_ADMIN", "localhost:2019"):
            for text, want in cases:
                with self.subTest(text=text):
                    self.assertEqual(self_host.scanned_admin_address(text), want)

    def test_validate_names_the_caddyfile_adapter(self):
        """A Caddyfile under another name would otherwise be read as JSON."""
        with mock.patch("self_host.subprocess.run") as run:
            run.return_value = subprocess.CompletedProcess([], 0, stdout="", stderr="")
            REAL_VALIDATE_CADDYFILE()
        self.assertEqual(
            run.call_args.args[0],
            ["caddy", "validate", "--config", str(self.caddyfile), "--adapter", "caddyfile"],
        )

    # Never creating a Caddyfile over a serving Caddy.

    def test_a_named_caddyfile_that_does_not_exist_is_never_created(self):
        admin_address, serves = mock.Mock(), mock.Mock()
        with mock.patch.object(self_host, "CADDYFILE_NAMED", True), mock.patch.object(
            self_host, "caddy_admin_address", admin_address
        ), mock.patch.object(self_host, "caddy_serves_a_config", serves):
            with self.assertRaises(RuntimeError) as caught:
                self._apply()
        self.assertIn(str(self.caddyfile), str(caught.exception))
        self.assertFalse(self.caddyfile.exists())
        self.assertFalse(self.config_dir.exists(), "no backup either")
        admin_address.assert_not_called()
        serves.assert_not_called()
        self.assertEqual(self.calls, [])

    def test_a_missing_default_caddyfile_is_not_created_over_a_serving_caddy(self):
        with FakeCaddyAdmin(running={"apps": {"http": {}}}) as admin, mock.patch.object(
            self_host, "CADDY_DEFAULT_ADMIN", admin.address
        ):
            with self.assertRaises(RuntimeError) as caught:
                self._apply()
        self.assertIn("--caddyfile", str(caught.exception))
        self.assertFalse(self.caddyfile.exists())
        self.assertEqual(self.calls, [])

    def test_a_missing_default_caddyfile_is_created_when_no_caddy_serves_a_config(self):
        with FakeCaddyAdmin(running=None) as admin, mock.patch.object(
            self_host, "CADDY_DEFAULT_ADMIN", admin.address
        ), mock.patch.object(self_host, "caddy_admin_address", return_value=None):
            self._apply()
        self.assertIn("reverse_proxy localhost:39111", self.caddyfile.read_text())
        self.assertEqual(self.calls, ["validate", "reload"])

    def test_dev_start_refuses_a_missing_named_caddyfile_before_stopping_anything(self):
        config = self_host.Config(site=self.site, mode=self_host.Mode.PROD, caddyfile=str(self.tmp / "missing"))
        commands = [lambda: self_host.command_dev_start(None)]
        reached = lambda *args, **kwargs: (_ for _ in ()).throw(AssertionError("reached past the Caddyfile check"))
        steps = ("find_go", "install_web", "build_web", "sync_webdist", "build_server_release", "check_prebuilt_binary")
        for run_command in commands:
            stop, save = mock.Mock(), mock.Mock()
            with self.subTest(command=run_command), mock.patch.multiple(
                self_host,
                resolve_config=mock.Mock(return_value=config),
                command_stop=stop,
                save_config=save,
                ensure_tools=mock.DEFAULT,
                **{step: mock.Mock(side_effect=reached) for step in steps},
            ):
                with self.assertRaisesRegex(RuntimeError, "check --caddyfile"):
                    run_command()
                stop.assert_not_called()
                save.assert_not_called()

    def test_use_caddyfile_records_whether_the_config_names_one(self):
        config = self_host.Config(site=self.site, mode=self_host.Mode.PROD)
        self_host.use_caddyfile(replace(config, caddyfile="scripts/Caddyfile"))
        self.assertTrue(self_host.CADDYFILE_NAMED)
        self_host.use_caddyfile(config)
        self.assertFalse(self_host.CADDYFILE_NAMED)
        self.assertEqual(self_host.CADDYFILE, self_host.DEFAULT_CADDYFILE)


class UnameMappingTests(unittest.TestCase):
    def test_parse_uname_maps_known_platforms(self):
        self.assertEqual(self_host.parse_uname("Linux x86_64"), ("linux", "amd64"))
        self.assertEqual(self_host.parse_uname("Linux aarch64\n"), ("linux", "arm64"))

    def test_parse_uname_refuses_unknown_platform(self):
        with self.assertRaises(ValueError):
            self_host.parse_uname("Darwin arm64")
        with self.assertRaises(ValueError):
            self_host.parse_uname("Linux armv7l")
        with self.assertRaises(ValueError):
            self_host.parse_uname("")

    def test_detect_remote_platform_runs_uname_over_ssh_and_parses_it(self):
        with mock.patch("self_host.subprocess.run") as run:
            run.return_value = subprocess.CompletedProcess([], returncode=0, stdout="Linux x86_64\n")
            goos, goarch = self_host.detect_remote_platform("user@example.host")
            self.assertEqual((goos, goarch), ("linux", "amd64"))
            run.assert_called_once_with(
                ["ssh", "user@example.host", "uname -sm"], capture_output=True, text=True, check=True
            )


class CrossCompileTests(unittest.TestCase):
    def test_go_build_cross_command_sets_tags_trimpath_output_and_version(self):
        cmd = self_host.go_build_cross_command("v9.9.9", "linux", "arm64")
        self.assertIn("-tags", cmd)
        self.assertIn("embedweb", cmd)
        self.assertIn("-trimpath", cmd)
        self.assertIn(str(self_host.dist_binary_path("linux", "arm64")), cmd)
        self.assertIn("./cmd/ajayeb", cmd)
        self.assertTrue(any("main.version=v9.9.9" in part for part in cmd))

    def test_go_build_cross_command_is_exact(self):
        with mock.patch.object(self_host, "find_go", return_value="/fake/go"):
            cmd = self_host.go_build_cross_command("v9.9.9", "linux", "amd64")
        self.assertEqual(
            cmd,
            [
                "/fake/go",
                "build",
                "-tags",
                "embedweb",
                "-trimpath",
                "-ldflags",
                "-s -w -X main.version=v9.9.9",
                "-o",
                str(self_host.DIST_DIR / "ajayeb-linux-amd64"),
                "./cmd/ajayeb",
            ],
        )

    def test_cross_build_runs_the_go_step_in_server_dir_with_the_cross_env(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            dist = Path(tmp) / "dist"
            steps: list = []

            def fake_step(cmd, *, cwd=self_host.ROOT, env=None, kind="build"):
                steps.append((cmd, cwd, env))
                if env is not None:
                    self_host.dist_binary_path("linux", "amd64").write_text("bin")

            with mock.patch.object(self_host, "DIST_DIR", dist), mock.patch.multiple(
                self_host, run_build_step=mock.DEFAULT, sync_webdist=mock.DEFAULT, find_go=mock.DEFAULT
            ) as mocks:
                mocks["run_build_step"].side_effect = fake_step
                mocks["find_go"].return_value = "/fake/go"
                out = self_host.cross_build("v1", "linux", "amd64")
                self.assertEqual(out, dist / "ajayeb-linux-amd64")
                self.assertEqual(len(steps), 3)
                self.assertIn("pnpm", steps[0][0][-1])
                self.assertIn("pnpm -C web build", steps[1][0][-1])
                go_cmd, go_cwd, go_env = steps[2]
                self.assertEqual(go_cmd[0], "/fake/go")
                self.assertEqual(go_cwd, self_host.SERVER_DIR)
                self.assertEqual(go_env, {"CGO_ENABLED": "0", "GOOS": "linux", "GOARCH": "amd64"})
                mocks["sync_webdist"].assert_called_once_with()

    def test_go_build_cross_env_disables_cgo_and_sets_target(self):
        self.assertEqual(
            self_host.go_build_cross_env("linux", "arm64"), {"CGO_ENABLED": "0", "GOOS": "linux", "GOARCH": "arm64"}
        )

    def test_heavy_wrap_passes_through_when_heavy_py_is_absent(self):
        orig = self_host.HEAVY_SCRIPT
        self_host.HEAVY_SCRIPT = Path("/nonexistent/tools/heavy.py")
        try:
            self.assertEqual(self_host.heavy_wrap(["echo", "hi"], kind="build"), ["echo", "hi"])
        finally:
            self_host.HEAVY_SCRIPT = orig

    def test_heavy_wrap_wraps_the_command_when_heavy_py_exists(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            heavy = Path(tmp) / "heavy.py"
            heavy.write_text("# fake\n")
            orig = self_host.HEAVY_SCRIPT
            self_host.HEAVY_SCRIPT = heavy
            try:
                wrapped = self_host.heavy_wrap(["echo", "hi"], kind="build", wait=42)
                self.assertEqual(
                    wrapped,
                    [self_host.sys.executable, str(heavy), "run", "--kind", "build", "--wait", "42", "--", "echo", "hi"],
                )
            finally:
                self_host.HEAVY_SCRIPT = orig

    def test_run_build_step_raises_a_clear_error_on_heavy_refusal(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            heavy = Path(tmp) / "heavy.py"
            heavy.write_text("# fake\n")
            orig = self_host.HEAVY_SCRIPT
            self_host.HEAVY_SCRIPT = heavy
            try:
                with mock.patch("self_host.subprocess.run") as run:
                    run.return_value = subprocess.CompletedProcess([], returncode=75)
                    with self.assertRaisesRegex(RuntimeError, "refused"):
                        self_host.run_build_step(["true"], kind="build")
            finally:
                self_host.HEAVY_SCRIPT = orig

    def test_run_build_step_raises_called_process_error_on_other_failures(self):
        orig = self_host.HEAVY_SCRIPT
        self_host.HEAVY_SCRIPT = Path("/nonexistent/tools/heavy.py")
        try:
            with mock.patch("self_host.subprocess.run") as run:
                run.return_value = subprocess.CompletedProcess([], returncode=1)
                with self.assertRaises(subprocess.CalledProcessError):
                    self_host.run_build_step(["false"], kind="build")
        finally:
            self_host.HEAVY_SCRIPT = orig


class RemoteCommandQuotingTests(unittest.TestCase):
    def test_remote_mkdir_command_quotes_a_remote_dir_with_a_space(self):
        cmd = self_host.remote_mkdir_command("user@example.host", "a dir")
        self.assertEqual(cmd[:2], ["ssh", "user@example.host"])
        self.assertIn(shlex.quote("a dir/.self-host/bin"), cmd[2])

    def test_rsync_self_host_command_quotes_remote_dir(self):
        cmd = self_host.rsync_self_host_command("user@example.host", "a dir")
        self.assertEqual(cmd[0], "rsync")
        self.assertEqual(cmd[-1], f"user@example.host:{shlex.quote('a dir')}/")

    def test_rsync_binary_command_quotes_remote_dir(self):
        cmd = self_host.rsync_binary_command("user@example.host", "a dir", Path("/tmp/ajayeb-linux-amd64"))
        self.assertIn(str(Path("/tmp/ajayeb-linux-amd64")), cmd)
        self.assertEqual(cmd[-1], f"user@example.host:{shlex.quote('a dir')}/{self_host.REMOTE_UPLOAD_BIN}")

    def test_remote_config_exists_command_quotes_remote_dir(self):
        cmd = self_host.remote_config_exists_command("user@example.host", "a dir")
        self.assertEqual(cmd[:2], ["ssh", "user@example.host"])
        self.assertIn(shlex.quote("a dir/.self-host/config.json"), cmd[2])

    def test_remote_deploy_command_quotes_dir_and_url_but_keeps_shell_operators(self):
        cmd = self_host.remote_deploy_command("user@example.host", "a dir", "setup", "https://ex ample.test")
        self.assertEqual(cmd[:2], ["ssh", "user@example.host"])
        remote = cmd[2]
        self.assertIn("cd " + shlex.quote("a dir") + " &&", remote)
        self.assertIn("--prebuilt " + shlex.quote(self_host.REMOTE_UPLOAD_BIN), remote)
        self.assertIn(shlex.quote("https://ex ample.test"), remote)

    def test_remote_deploy_command_omits_url_when_none(self):
        cmd = self_host.remote_deploy_command("user@example.host", "ajayeb", "redeploy", None)
        remote = cmd[2]
        self.assertTrue(remote.endswith(self_host.REMOTE_UPLOAD_BIN))
        self.assertIn(" redeploy ", remote)


    def test_remote_deploy_command_forwards_a_quoted_caddyfile_only_when_given(self):
        named = self_host.remote_deploy_command(
            "user@example.host", "ajayeb", "setup", "https://x.test", None, "my dir/Caddyfile"
        )[2]
        kept = self_host.remote_deploy_command("user@example.host", "ajayeb", "setup", "https://x.test")[2]
        self.assertIn(" --caddyfile " + shlex.quote("my dir/Caddyfile") + " ", named)
        self.assertNotIn("--caddyfile", kept)
        self.assertTrue(named.endswith(shlex.quote("https://x.test")))

    def test_remote_deploy_command_forwards_both_schemes_only_when_given(self):
        on = self_host.remote_deploy_command("user@example.host", "ajayeb", "redeploy", "https://x.test", True)[2]
        off = self_host.remote_deploy_command("user@example.host", "ajayeb", "redeploy", "https://x.test", False)[2]
        kept = self_host.remote_deploy_command("user@example.host", "ajayeb", "redeploy", "https://x.test")[2]
        self.assertIn(" --both-schemes ", on)
        self.assertIn(" --no-both-schemes ", off)
        self.assertNotIn("both-schemes", kept)
        self.assertTrue(on.endswith(shlex.quote("https://x.test")))


class DeployChoiceTests(unittest.TestCase):
    def test_determine_deploy_command_picks_setup_when_no_remote_config(self):
        with mock.patch("self_host.subprocess.run") as run:
            run.return_value = subprocess.CompletedProcess([], returncode=1)
            self.assertEqual(self_host.determine_deploy_command("user@example.host", "ajayeb"), "setup")

    def test_determine_deploy_command_picks_redeploy_when_remote_config_exists(self):
        with mock.patch("self_host.subprocess.run") as run:
            run.return_value = subprocess.CompletedProcess([], returncode=0)
            self.assertEqual(self_host.determine_deploy_command("user@example.host", "ajayeb"), "redeploy")


    def test_determine_deploy_command_raises_when_ssh_fails(self):
        with mock.patch("self_host.subprocess.run") as run:
            run.return_value = subprocess.CompletedProcess([], returncode=255, stderr="connection refused")
            with self.assertRaisesRegex(RuntimeError, "connection refused"):
                self_host.determine_deploy_command("user@example.host", "ajayeb")


class ShipTargetValidationTests(unittest.TestCase):
    def test_accepts_plain_target_and_dir(self):
        self_host.validate_ship_target("user@example.host", "apps/ajayeb")

    def test_refuses_option_like_or_unsafe_values(self):
        for target, remote_dir in [
            ("-oProxyCommand=x", "ajayeb"),
            ("user@example.host", "-rf"),
            ("user@example.host", "a dir"),
            ("user@example.host", "x;rm"),
            ("user@example.host", ""),
            ("user @example.host", "ajayeb"),
        ]:
            with self.assertRaises(RuntimeError, msg=(target, remote_dir)):
                self_host.validate_ship_target(target, remote_dir)


class PrebuiltBinaryTests(unittest.TestCase):
    def test_check_prebuilt_binary_refuses_a_missing_file(self):
        with self.assertRaises(RuntimeError):
            self_host.check_prebuilt_binary(Path("/nonexistent/ajayeb-binary-xyz"))

    def test_check_prebuilt_binary_refuses_a_non_executable_file(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "ajayeb"
            path.write_text("not a binary")
            path.chmod(0o644)
            with self.assertRaises(RuntimeError):
                self_host.check_prebuilt_binary(path)

    def test_check_prebuilt_binary_refuses_when_version_flag_fails(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "ajayeb"
            path.write_text("#!/bin/sh\nexit 1\n")
            path.chmod(0o755)
            with self.assertRaises(RuntimeError):
                self_host.check_prebuilt_binary(path)

    def test_check_prebuilt_binary_accepts_a_runnable_executable(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "ajayeb"
            path.write_text("#!/bin/sh\necho v1.2.3\n")
            path.chmod(0o755)
            self_host.check_prebuilt_binary(path)  # must not raise


class SwapInBinaryTests(unittest.TestCase):
    def _with_bin_dir(self, tmp):
        bin_dir = Path(tmp) / "bin"
        bin_dir.mkdir(parents=True)
        originals = (self_host.BIN_DIR, self_host.BIN_PATH, self_host.BIN_NEW_PATH, self_host.BIN_PREV_PATH)
        self_host.BIN_DIR = bin_dir
        self_host.BIN_PATH = bin_dir / "ajayeb"
        self_host.BIN_NEW_PATH = bin_dir / "ajayeb.new"
        self_host.BIN_PREV_PATH = bin_dir / "ajayeb.prev"
        return bin_dir, originals

    def _restore(self, originals):
        self_host.BIN_DIR, self_host.BIN_PATH, self_host.BIN_NEW_PATH, self_host.BIN_PREV_PATH = originals

    def test_swap_in_binary_backs_up_the_current_binary_and_installs_the_new_one(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            bin_dir, originals = self._with_bin_dir(tmp)
            try:
                self_host.BIN_PATH.write_text("old-binary")
                new_source = Path(tmp) / "new-binary"
                new_source.write_text("new-binary")
                self_host.swap_in_binary(new_source)
                self.assertEqual(self_host.BIN_PATH.read_text(), "new-binary")
                self.assertEqual(self_host.BIN_PREV_PATH.read_text(), "old-binary")
                self.assertTrue(os.access(self_host.BIN_PATH, os.X_OK))
                # The caller's own file (e.g. their --prebuilt path) is never touched.
                self.assertTrue(new_source.exists())
                self.assertEqual(new_source.read_text(), "new-binary")
            finally:
                self._restore(originals)

    def test_swap_in_binary_skips_the_backup_when_there_is_no_current_binary(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            bin_dir, originals = self._with_bin_dir(tmp)
            try:
                new_source = Path(tmp) / "new-binary"
                new_source.write_text("new-binary")
                self_host.swap_in_binary(new_source)
                self.assertEqual(self_host.BIN_PATH.read_text(), "new-binary")
                self.assertFalse(self_host.BIN_PREV_PATH.exists())
            finally:
                self._restore(originals)

    def test_swap_in_binary_handles_an_already_built_bin_new_path(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            bin_dir, originals = self._with_bin_dir(tmp)
            try:
                self_host.BIN_NEW_PATH.write_text("freshly-built")
                self_host.swap_in_binary(self_host.BIN_NEW_PATH)
                self.assertEqual(self_host.BIN_PATH.read_text(), "freshly-built")
            finally:
                self._restore(originals)

    def test_swap_in_binary_move_consumes_the_source_file(self):
        """setup/redeploy --prebuilt installs an uploaded binary with
        move=True, so it does not leave a second ~8.5 MB copy on disk."""
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            bin_dir, originals = self._with_bin_dir(tmp)
            try:
                self_host.BIN_PATH.write_text("old-binary")
                new_source = Path(tmp) / "new-binary"
                new_source.write_text("new-binary")
                self_host.swap_in_binary(new_source, move=True)
                self.assertEqual(self_host.BIN_PATH.read_text(), "new-binary")
                self.assertEqual(self_host.BIN_PREV_PATH.read_text(), "old-binary")
                self.assertTrue(os.access(self_host.BIN_PATH, os.X_OK))
                self.assertFalse(new_source.exists(), "the uploaded source must be consumed, not copied")
            finally:
                self._restore(originals)

    def test_swap_in_binary_move_falls_back_to_copy_across_filesystems(self):
        """A rename can't cross filesystems; move=True still ends up with no
        residual copy by copying and then removing the source."""
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            bin_dir, originals = self._with_bin_dir(tmp)
            try:
                new_source = Path(tmp) / "new-binary"
                new_source.write_text("new-binary")
                real_replace = self_host.os.replace

                def fake_replace(src, dst):
                    if str(src) == str(new_source):
                        raise OSError("Invalid cross-device link")
                    return real_replace(src, dst)

                self_host.os.replace = fake_replace
                try:
                    self_host.swap_in_binary(new_source, move=True)
                finally:
                    self_host.os.replace = real_replace
                self.assertEqual(self_host.BIN_PATH.read_text(), "new-binary")
                self.assertFalse(new_source.exists(), "the fallback must still remove the source")
            finally:
                self._restore(originals)


class SetupRedeployOrderingTests(unittest.TestCase):
    """setup/redeploy: --prebuilt skips every build step, and stop only runs
    right before the swap, after every build and check."""

    def setUp(self):
        for name, value in (("check_candidate_rooms", None), ("wait_for_port_free", True)):
            patcher = mock.patch.object(self_host, name, return_value=value)
            patcher.start()
            self.addCleanup(patcher.stop)

    def _config(self):
        return self_host.Config(
            site=self_host.parse_site_url("https://game.example.test"),
            mode=self_host.Mode.PROD,
            ports=self_host.Ports(server=1, web_dev=2),
        )

    def test_command_setup_with_prebuilt_skips_every_build_step(self):
        with mock.patch.multiple(
            self_host,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            check_prebuilt_binary=mock.DEFAULT,
            check_candidate_rooms=mock.DEFAULT,
            wait_for_port_free=mock.Mock(return_value=True),
            find_go=mock.DEFAULT,
            install_web=mock.DEFAULT,
            build_web=mock.DEFAULT,
            sync_webdist=mock.DEFAULT,
            build_server_release=mock.DEFAULT,
            save_config=mock.DEFAULT,
            command_stop=mock.DEFAULT,
            swap_in_binary=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        ) as mocks:
            mocks["resolve_config"].return_value = self._config()
            mocks["ports_after_stop"].side_effect = lambda config: config
            mocks["poll_health"].return_value = True
            self_host.command_setup("https://game.example.test", "/tmp/prebuilt-ajayeb")
            mocks["check_prebuilt_binary"].assert_called_once_with(Path("/tmp/prebuilt-ajayeb"))
            mocks["find_go"].assert_not_called()
            mocks["install_web"].assert_not_called()
            mocks["build_web"].assert_not_called()
            mocks["sync_webdist"].assert_not_called()
            mocks["build_server_release"].assert_not_called()
            # --prebuilt installs by moving the uploaded binary into place
            # rather than leaving a second copy of it behind.
            mocks["swap_in_binary"].assert_called_once_with(Path("/tmp/prebuilt-ajayeb"), move=True)

    def test_command_setup_builds_and_checks_before_stopping_the_server(self):
        calls: list = []
        with mock.patch.multiple(
            self_host,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            find_go=mock.DEFAULT,
            install_web=mock.DEFAULT,
            build_web=mock.DEFAULT,
            sync_webdist=mock.DEFAULT,
            build_server_release=mock.DEFAULT,
            save_config=mock.DEFAULT,
            command_stop=mock.DEFAULT,
            swap_in_binary=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        ) as mocks:
            mocks["resolve_config"].return_value = self._config()
            mocks["ports_after_stop"].side_effect = lambda config: config
            mocks["poll_health"].return_value = True
            mocks["build_server_release"].side_effect = lambda: calls.append("build") or Path("/tmp/new")
            mocks["command_stop"].side_effect = lambda: calls.append("stop")
            mocks["ports_after_stop"].side_effect = lambda config: calls.append("ports") or config
            mocks["save_config"].side_effect = lambda config: calls.append("save")
            mocks["swap_in_binary"].side_effect = lambda source, move=False: calls.append("swap")
            self_host.command_setup("https://game.example.test")
            self.assertEqual(calls, ["build", "stop", "ports", "save", "swap"])

    def test_command_setup_refuses_a_bad_prebuilt_before_anything_stops(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            bad = Path(tmp) / "bad-binary"
            bad.write_text("not runnable")
            bad.chmod(0o644)
            with mock.patch.multiple(
                self_host,
                resolve_config=mock.DEFAULT,
                ports_after_stop=mock.DEFAULT,
                ensure_tools=mock.DEFAULT,
                command_stop=mock.DEFAULT,
                swap_in_binary=mock.DEFAULT,
                apply_caddyfile_safely=mock.DEFAULT,
                start_server_prod=mock.DEFAULT,
            ) as mocks:
                mocks["resolve_config"].return_value = self._config()
                mocks["ports_after_stop"].side_effect = lambda config: config
                with self.assertRaises(RuntimeError):
                    self_host.command_setup("https://game.example.test", str(bad))
                mocks["command_stop"].assert_not_called()
                mocks["swap_in_binary"].assert_not_called()

    def test_command_redeploy_with_prebuilt_skips_every_build_step(self):
        with mock.patch.multiple(
            self_host,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            check_prebuilt_binary=mock.DEFAULT,
            check_candidate_rooms=mock.DEFAULT,
            wait_for_port_free=mock.Mock(return_value=True),
            find_go=mock.DEFAULT,
            archive_rooms=mock.DEFAULT,
            install_web=mock.DEFAULT,
            build_web=mock.DEFAULT,
            sync_webdist=mock.DEFAULT,
            build_server_release=mock.DEFAULT,
            save_config=mock.DEFAULT,
            command_stop=mock.DEFAULT,
            swap_in_binary=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        ) as mocks:
            mocks["resolve_config"].return_value = self._config()
            mocks["ports_after_stop"].side_effect = lambda config: config
            mocks["archive_rooms"].return_value = None
            mocks["poll_health"].return_value = True
            self_host.command_redeploy("https://game.example.test", "/tmp/prebuilt-ajayeb")
            mocks["find_go"].assert_not_called()
            mocks["install_web"].assert_not_called()
            mocks["build_web"].assert_not_called()
            mocks["sync_webdist"].assert_not_called()
            mocks["build_server_release"].assert_not_called()
            # --prebuilt installs by moving the uploaded binary into place
            # rather than leaving a second copy of it behind.
            mocks["swap_in_binary"].assert_called_once_with(Path("/tmp/prebuilt-ajayeb"), move=True)

    def test_command_redeploy_archives_and_builds_before_stopping_the_server(self):
        calls: list = []
        with mock.patch.multiple(
            self_host,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            find_go=mock.DEFAULT,
            archive_rooms=mock.DEFAULT,
            install_web=mock.DEFAULT,
            build_web=mock.DEFAULT,
            sync_webdist=mock.DEFAULT,
            build_server_release=mock.DEFAULT,
            save_config=mock.DEFAULT,
            command_stop=mock.DEFAULT,
            swap_in_binary=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        ) as mocks:
            mocks["resolve_config"].return_value = self._config()
            mocks["ports_after_stop"].side_effect = lambda config: config
            mocks["poll_health"].return_value = True
            mocks["archive_rooms"].side_effect = lambda: calls.append("archive") or None
            mocks["build_server_release"].side_effect = lambda: calls.append("build") or Path("/tmp/new")
            mocks["command_stop"].side_effect = lambda: calls.append("stop")
            mocks["ports_after_stop"].side_effect = lambda config: calls.append("ports") or config
            mocks["save_config"].side_effect = lambda config: calls.append("save")
            mocks["swap_in_binary"].side_effect = lambda source, move=False: calls.append("swap")
            self_host.command_redeploy("https://game.example.test")
            # The archive still runs with the server already stopped (so it
            # never captures a file mid-write), and still before the swap.
            self.assertEqual(calls, ["build", "stop", "ports", "save", "archive", "swap"])


class ProductionCaddyFailureTests(unittest.TestCase):
    """Production commands continue after Caddy failures, without real services."""

    COMMANDS = ("setup", "redeploy", "start")
    ORIGINAL = "other.example.test {\n\treverse_proxy localhost:38959\n}\n"

    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.root = Path(tmp.name)
        self.caddyfile = self.root / "Caddyfile"
        self.caddyfile.write_text(self.ORIGINAL)
        self.config = self_host.Config(
            site=self_host.parse_site_url("https://game.example.test"),
            mode=self_host.Mode.PROD,
            ports=self_host.Ports(server=38951, web_dev=38952),
        )
        for name, value in (
            ("DEFAULT_CADDYFILE", self.caddyfile),
            ("CADDYFILE", self.caddyfile),
            ("CADDYFILE_NAMED", False),
            ("CONFIG_DIR", self.root / "config"),
            ("BIN_PATH", Path(__file__)),
        ):
            patcher = mock.patch.object(self_host, name, value)
            patcher.start()
            self.addCleanup(patcher.stop)

    def _run(self, command, **overrides):
        patches = {
            "resolve_config": mock.Mock(return_value=self.config),
            "ensure_tools": mock.Mock(),
            "check_prebuilt_binary": mock.Mock(),
            "check_candidate_rooms": mock.Mock(),
            "wait_for_port_free": mock.Mock(return_value=True),
            "command_stop": mock.Mock(),
            "ports_after_stop": mock.Mock(return_value=replace(self.config, ports=replace(self.config.ports, server=38953))),
            "save_config": mock.Mock(),
            "swap_in_binary": mock.Mock(),
            "archive_rooms": mock.Mock(return_value=None),
            "start_server_prod": mock.Mock(),
            "poll_health": mock.Mock(return_value=True),
            "announce_serving": mock.Mock(),
            "caddy_serves_a_config": mock.Mock(return_value=False),
            "caddy_admin_address": mock.Mock(return_value="mock-admin"),
            "check_caddy_serves": mock.Mock(),
            "load_through_caddy": mock.Mock(),
            "validate_caddyfile": mock.Mock(return_value=subprocess.CompletedProcess([], 0, stderr="")),
            "reload_caddy": mock.Mock(),
        }
        patches.update(overrides)
        order = mock.Mock()
        for name in ("command_stop", "start_server_prod", "poll_health", "announce_serving"):
            order.attach_mock(patches[name], name)
        args = [command]
        if command != "start":
            args += ["--prebuilt", str(self.root / "uploaded-binary")]
        stdout, stderr = io.StringIO(), io.StringIO()
        with mock.patch.multiple(self_host, **patches), redirect_stdout(stdout), redirect_stderr(stderr):
            code = self_host.main(args)
        patches["order"] = order
        return code, stdout.getvalue(), stderr.getvalue(), patches

    def _assert_unpublished(self, result):
        code, stdout, stderr, mocks = result
        self.assertEqual(code, 1)
        mocks["command_stop"].assert_called_once()
        mocks["start_server_prod"].assert_called_once_with(replace(self.config.ports, server=38953), self.config.site)
        mocks["poll_health"].assert_called_once_with(38953)
        mocks["announce_serving"].assert_not_called()
        self.assertEqual([call[0] for call in mocks["order"].mock_calls], ["command_stop", "start_server_prod", "poll_health"])
        self.assertIn("Caddy configuration failed", stderr)
        self.assertIn("game server is running on http://127.0.0.1:38953 but is not published", stderr)
        self.assertNotIn("serving:", stdout)

    def test_missing_named_caddyfile_still_starts_each_production_command(self):
        self.config = replace(self.config, caddyfile=str(self.root / "missing"))
        for command in self.COMMANDS:
            with self.subTest(command=command):
                result = self._run(command)
                self._assert_unpublished(result)
                self.assertIn("check --caddyfile", result[2])
                result[3]["load_through_caddy"].assert_not_called()
                self.assertFalse((self.root / "missing").exists())
                self.assertEqual(self.caddyfile.read_text(), self.ORIGINAL)

    def test_missing_default_file_never_replaces_a_serving_caddy(self):
        self.caddyfile.unlink()
        for command in self.COMMANDS:
            with self.subTest(command=command):
                result = self._run(command, caddy_serves_a_config=mock.Mock(return_value=True))
                self._assert_unpublished(result)
                self.assertIn("already serves a config", result[2])
                result[3]["load_through_caddy"].assert_not_called()
                self.assertFalse(self.caddyfile.exists())

    def test_preflight_mismatch_or_adapt_refusal_still_starts_without_edit_or_load(self):
        for error in (self_host.CaddyMismatch("wrong config"), self_host.CaddyRejected("adapt refused")):
            for command in self.COMMANDS:
                with self.subTest(command=command, error=type(error).__name__):
                    result = self._run(command, check_caddy_serves=mock.Mock(side_effect=error))
                    self._assert_unpublished(result)
                    self.assertIn(str(error), result[2])
                    result[3]["load_through_caddy"].assert_not_called()
                    self.assertEqual(self.caddyfile.read_text(), self.ORIGINAL)
                    self.assertFalse((self.root / "config").exists())

    def test_refused_load_and_unknown_outcome_restore_file_and_still_start(self):
        for error in (self_host.CaddyRejected("load refused"), self_host.CaddyUnsure("outcome unknown")):
            for command in self.COMMANDS:
                with self.subTest(command=command, error=type(error).__name__):
                    result = self._run(command, load_through_caddy=mock.Mock(side_effect=error))
                    self._assert_unpublished(result)
                    result[3]["load_through_caddy"].assert_called_once()
                    result[3]["reload_caddy"].assert_not_called()
                    self.assertIn(str(error), result[2])
                    self.assertEqual(self.caddyfile.read_text(), self.ORIGINAL)

    def test_invalid_file_restores_file_and_still_starts(self):
        for command in self.COMMANDS:
            with self.subTest(command=command):
                result = self._run(
                    command,
                    caddy_admin_address=mock.Mock(return_value=None),
                    validate_caddyfile=mock.Mock(return_value=subprocess.CompletedProcess([], 1, stderr="invalid directive")),
                )
                self._assert_unpublished(result)
                self.assertIn("invalid directive", result[2])
                result[3]["reload_caddy"].assert_not_called()
                self.assertEqual(self.caddyfile.read_text(), self.ORIGINAL)

    def test_stopped_caddy_keeps_valid_edit_but_returns_nonzero(self):
        for command in self.COMMANDS:
            with self.subTest(command=command):
                self.caddyfile.write_text(self.ORIGINAL)
                result = self._run(
                    command,
                    check_caddy_serves=mock.Mock(side_effect=self_host.CaddyUnreachable("not listening")),
                    load_through_caddy=mock.Mock(side_effect=self_host.CaddyUnreachable("not listening")),
                )
                self._assert_unpublished(result)
                result[3]["validate_caddyfile"].assert_called_once()
                result[3]["reload_caddy"].assert_not_called()
                self.assertIn("reverse_proxy localhost:38953", self.caddyfile.read_text())

    def test_reload_failure_keeps_valid_edit_but_returns_nonzero(self):
        for command in self.COMMANDS:
            with self.subTest(command=command):
                self.caddyfile.write_text(self.ORIGINAL)
                result = self._run(
                    command,
                    caddy_admin_address=mock.Mock(return_value=None),
                    reload_caddy=mock.Mock(side_effect=subprocess.CalledProcessError(1, ["caddy", "reload"])),
                )
                self._assert_unpublished(result)
                self.assertIn("caddy reload failed", result[2])
                self.assertIn("reverse_proxy localhost:38953", self.caddyfile.read_text())

    def test_missing_caddy_command_still_starts(self):
        def tools(names):
            if "caddy" in names:
                raise RuntimeError("Missing required command(s): caddy")
        for command in self.COMMANDS:
            with self.subTest(command=command):
                result = self._run(command, ensure_tools=mock.Mock(side_effect=tools))
                self._assert_unpublished(result)
                self.assertIn("Missing required command(s): caddy", result[2])
                result[3]["load_through_caddy"].assert_not_called()
                self.assertEqual(self.caddyfile.read_text(), self.ORIGINAL)

    def test_normal_load_and_reload_paths_succeed(self):
        for address in ("mock-admin", None):
            for command in self.COMMANDS:
                with self.subTest(command=command, address=address):
                    self.caddyfile.write_text(self.ORIGINAL)
                    code, _, stderr, mocks = self._run(command, caddy_admin_address=mock.Mock(return_value=address))
                    self.assertEqual(code, 0, stderr)
                    mocks["start_server_prod"].assert_called_once()
                    mocks["poll_health"].assert_called_once_with(38953)
                    mocks["announce_serving"].assert_called_once()
                    if address:
                        mocks["load_through_caddy"].assert_called_once()
                        mocks["reload_caddy"].assert_not_called()
                    else:
                        mocks["validate_caddyfile"].assert_called_once()
                        mocks["reload_caddy"].assert_called_once()

    def test_unchanged_file_checks_observable_caddy_without_mutation(self):
        ports = replace(self.config.ports, server=38953)
        self.caddyfile.write_text(self_host.render_caddy_block(self.config.site, self_host.Mode.PROD, ports))
        original = self.caddyfile.read_bytes()
        for command in self.COMMANDS:
            for down in (False, True):
                with self.subTest(command=command, down=down):
                    check = mock.Mock(side_effect=self_host.CaddyUnreachable("not running") if down else None)
                    result = self._run(command, preflight_caddyfile=mock.Mock(), check_caddy_serves=check)
                    if down:
                        self._assert_unpublished(result)
                    else:
                        self.assertEqual(result[0], 0, result[2])
                    check.assert_called_once_with("mock-admin", original)
                    result[3]["load_through_caddy"].assert_not_called()
                    result[3]["validate_caddyfile"].assert_not_called()
                    result[3]["reload_caddy"].assert_not_called()
                    self.assertEqual(self.caddyfile.read_bytes(), original)
                    self.assertFalse((self.root / "config").exists())

    def test_unchanged_unobservable_caddy_is_reported_as_unverified(self):
        ports = replace(self.config.ports, server=38953)
        self.caddyfile.write_text(self_host.render_caddy_block(self.config.site, self_host.Mode.PROD, ports))
        original = self.caddyfile.read_bytes()
        for command in self.COMMANDS:
            with self.subTest(command=command):
                code, stdout, stderr, mocks = self._run(command, caddy_admin_address=mock.Mock(return_value=None))
                self.assertEqual(code, 0, stderr)
                self.assertIn("publication at https://game.example.test is unverified", stdout)
                mocks["announce_serving"].assert_not_called()
                mocks["load_through_caddy"].assert_not_called()
                mocks["reload_caddy"].assert_not_called()
                self.assertEqual(self.caddyfile.read_bytes(), original)

    def test_dev_start_keeps_warning_only_reload_behavior(self):
        self.config = replace(self.config, mode=self_host.Mode.DEV)
        with mock.patch.multiple(
            self_host,
            resolve_config=mock.Mock(return_value=self.config),
            command_stop=mock.DEFAULT,
            allocate_ports=mock.Mock(return_value=self.config.ports),
            ensure_tools=mock.DEFAULT,
            find_go=mock.DEFAULT,
            install_web=mock.DEFAULT,
            save_config=mock.DEFAULT,
            caddy_admin_address=mock.Mock(return_value=None),
            validate_caddyfile=mock.Mock(return_value=subprocess.CompletedProcess([], 0, stderr="")),
            reload_caddy=mock.Mock(side_effect=subprocess.CalledProcessError(1, ["caddy", "reload"])),
            start_server_dev=mock.DEFAULT,
            start_web_dev=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        ) as mocks, redirect_stderr(io.StringIO()) as stderr:
            self.assertEqual(self_host.main(["dev-start"]), 0)
        mocks["start_server_dev"].assert_called_once_with(self.config.ports)
        mocks["start_web_dev"].assert_called_once_with(self.config.ports)
        mocks["announce_serving"].assert_called_once()
        self.assertIn("warning: caddy reload failed", stderr.getvalue())

    def test_failed_server_start_never_claims_running_even_after_caddy_failure(self):
        for command in self.COMMANDS:
            for caddy_failed in (False, True):
                with self.subTest(command=command, caddy_failed=caddy_failed):
                    self.caddyfile.write_text(self.ORIGINAL)
                    result = self._run(
                        command,
                        start_server_prod=mock.Mock(side_effect=RuntimeError("tmux refused")),
                        load_through_caddy=mock.Mock(side_effect=self_host.CaddyRejected("load refused") if caddy_failed else None),
                    )
                    code, stdout, stderr, mocks = result
                    self.assertEqual(code, 1)
                    self.assertIn("game server startup failed on port 38953: tmux refused", stderr)
                    self.assertNotIn("game server is running", stderr + stdout)
                    mocks["poll_health"].assert_not_called()
                    mocks["announce_serving"].assert_not_called()
                    if caddy_failed:
                        self.assertIn("load refused", stderr)

    def test_failed_health_never_claims_running_even_after_caddy_failure(self):
        for command in self.COMMANDS:
            for caddy_failed in (False, True):
                with self.subTest(command=command, caddy_failed=caddy_failed):
                    self.caddyfile.write_text(self.ORIGINAL)
                    code, stdout, stderr, mocks = self._run(
                        command,
                        poll_health=mock.Mock(return_value=False),
                        load_through_caddy=mock.Mock(side_effect=self_host.CaddyRejected("load refused") if caddy_failed else None),
                    )
                    self.assertEqual(code, 1)
                    self.assertIn("game server startup could not be verified on port 38953", stderr)
                    self.assertNotIn("game server is running", stderr + stdout)
                    mocks["start_server_prod"].assert_called_once()
                    mocks["announce_serving"].assert_not_called()
                    if caddy_failed:
                        self.assertIn("load refused", stderr)

    def test_missing_tmux_is_not_a_caddy_failure_and_never_stops_or_starts(self):
        def tools(names):
            if "tmux" in names:
                raise RuntimeError("Missing required command(s): tmux")
        for command in self.COMMANDS:
            with self.subTest(command=command):
                code, stdout, stderr, mocks = self._run(command, ensure_tools=mock.Mock(side_effect=tools))
                self.assertEqual(code, 1)
                self.assertIn("Missing required command(s): tmux", stderr)
                self.assertNotIn("Caddy configuration failed", stderr)
                self.assertNotIn("game server is running", stderr + stdout)
                mocks["command_stop"].assert_not_called()
                mocks["start_server_prod"].assert_not_called()

    def test_bad_prebuilt_is_not_a_caddy_failure_and_never_stops_or_starts(self):
        for command in ("setup", "redeploy"):
            with self.subTest(command=command):
                code, stdout, stderr, mocks = self._run(command, check_prebuilt_binary=mock.Mock(side_effect=RuntimeError("invalid binary")))
                self.assertEqual(code, 1)
                self.assertIn("invalid binary", stderr)
                self.assertNotIn("Caddy configuration failed", stderr)
                self.assertNotIn("game server is running", stderr + stdout)
                mocks["command_stop"].assert_not_called()
                mocks["start_server_prod"].assert_not_called()

    def test_failed_binary_install_is_not_reported_as_caddy_failure(self):
        for command in ("setup", "redeploy"):
            with self.subTest(command=command):
                code, stdout, stderr, mocks = self._run(command, swap_in_binary=mock.Mock(side_effect=OSError("install failed")))
                self.assertEqual(code, 1)
                self.assertIn("install failed", stderr)
                self.assertNotIn("Caddy configuration failed", stderr)
                self.assertNotIn("game server is running", stderr + stdout)
                mocks["start_server_prod"].assert_not_called()


class PortsAfterStopTests(unittest.TestCase):
    """Ports are chosen after our own server stops, so a redeploy keeps its
    port instead of moving to the next one and rewriting the Caddyfile."""

    def _config(self, server=38951):
        return self_host.Config(
            site=self_host.parse_site_url("https://game.example.test"),
            mode=self_host.Mode.PROD,
            ports=self_host.Ports(server=server, web_dev=38952),
        )

    def test_wait_for_port_free_returns_once_the_old_socket_closes(self):
        answers = iter([False, False, True])
        with mock.patch.object(self_host.time, "sleep") as sleep:
            self.assertTrue(self_host.wait_for_port_free(38951, is_free=lambda port: next(answers)))
        self.assertEqual(sleep.call_count, 2)

    def test_wait_for_port_free_gives_up_after_the_timeout(self):
        with mock.patch.object(self_host.time, "sleep"):
            self.assertFalse(self_host.wait_for_port_free(38951, timeout=0.0, is_free=lambda port: False))

    def test_ports_after_stop_waits_for_the_configured_port_then_allocates(self):
        calls: list = []
        with mock.patch.multiple(self_host, wait_for_port_free=mock.DEFAULT, allocate_ports=mock.DEFAULT) as mocks:
            mocks["wait_for_port_free"].side_effect = lambda port: calls.append(("wait", port)) or True
            mocks["allocate_ports"].side_effect = lambda ports: calls.append(("allocate", ports.server)) or ports
            config = self_host.ports_after_stop(self._config(server=38953))
        self.assertEqual(calls, [("wait", 38953), ("allocate", 38953)])
        self.assertEqual(config.ports.server, 38953)

    def test_command_start_stops_before_choosing_ports(self):
        calls: list = []
        with mock.patch.multiple(
            self_host,
            command_stop=mock.DEFAULT,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            save_config=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        ) as mocks, mock.patch.object(self_host, "BIN_PATH", Path(__file__)):
            mocks["resolve_config"].return_value = self._config()
            mocks["command_stop"].side_effect = lambda: calls.append("stop")
            mocks["ports_after_stop"].side_effect = lambda config: calls.append("ports") or config
            mocks["poll_health"].return_value = True
            self_host.command_start("https://game.example.test")
        self.assertEqual(calls, ["stop", "ports"])


    def test_every_command_hands_the_stored_both_schemes_to_the_caddyfile(self):
        # A command that dropped it would render the redirect block at boot,
        # rewrite the shared Caddyfile and reload Caddy on every start.
        config = replace(self._config(), both_schemes=True)
        common = dict(
            command_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            save_config=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        )
        prod = dict(
            common,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            check_prebuilt_binary=mock.DEFAULT,
            check_candidate_rooms=mock.DEFAULT,
            wait_for_port_free=mock.Mock(return_value=True),
            swap_in_binary=mock.DEFAULT,
            archive_rooms=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
        )
        runs = [
            ("setup", prod, lambda: self_host.command_setup(None, "/tmp/bin", True)),
            ("redeploy", prod, lambda: self_host.command_redeploy(None, "/tmp/bin", True)),
            ("start", prod, lambda: self_host.command_start(None, True)),
        ]
        for name, patches, call in runs:
            with self.subTest(name), mock.patch.multiple(self_host, **patches) as mocks, mock.patch.object(
                self_host, "BIN_PATH", Path(__file__)
            ):
                mocks["resolve_config"].return_value = config
                mocks["ports_after_stop"].side_effect = lambda c: c
                mocks["archive_rooms"].return_value = None
                mocks["poll_health"].return_value = True
                call()
                self.assertTrue(mocks["resolve_config"].call_args.args[2])
                mocks["apply_caddyfile_safely"].assert_called_once_with(
                    config.site, self_host.Mode.PROD, config.ports, True
                )
        dev = dict(
            common,
            resolve_config=mock.DEFAULT,
            allocate_ports=mock.DEFAULT,
            find_go=mock.DEFAULT,
            install_web=mock.DEFAULT,
            start_server_dev=mock.DEFAULT,
            start_web_dev=mock.DEFAULT,
        )
        with mock.patch.multiple(self_host, **dev) as mocks:
            mocks["resolve_config"].return_value = replace(config, mode=self_host.Mode.DEV)
            mocks["allocate_ports"].side_effect = lambda ports: ports
            self_host.command_dev_start(None, True)
            mocks["resolve_config"].assert_called_once_with(None, self_host.Mode.DEV, True, None)
            mocks["apply_caddyfile_safely"].assert_called_once_with(config.site, self_host.Mode.DEV, config.ports, True)

    def test_main_routes_both_schemes_to_each_command(self):
        cases = [
            (["setup"], "command_setup", lambda m: m.call_args.args[2]),
            (["redeploy"], "command_redeploy", lambda m: m.call_args.args[2]),
            (["start"], "command_start", lambda m: m.call_args.args[1]),
            (["dev-start"], "command_dev_start", lambda m: m.call_args.args[1]),
            (["ship"], "command_ship", lambda m: m.call_args.args[3]),
        ]
        for argv, target, picked in cases:
            for extra, want in (([], None), (["--both-schemes"], True), (["--no-both-schemes"], False)):
                with self.subTest(argv=argv, extra=extra), mock.patch.object(self_host, target) as command:
                    self.assertEqual(self_host.main(argv + extra), 0)
                    self.assertIs(picked(command), want)

    def test_both_schemes_flags_exclude_each_other(self):
        with redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
            self_host.build_parser().parse_args(["redeploy", "--both-schemes", "--no-both-schemes"])


    def test_caddyfile_path_reads_relative_paths_from_home(self):
        home = Path.home()
        self.assertEqual(self_host.caddyfile_path(None), self_host.DEFAULT_CADDYFILE)
        self.assertEqual(self_host.caddyfile_path(""), self_host.DEFAULT_CADDYFILE)
        self.assertEqual(self_host.caddyfile_path("scripts/Caddyfile"), home / "scripts/Caddyfile")
        self.assertEqual(self_host.caddyfile_path("~/scripts/Caddyfile"), home / "scripts/Caddyfile")
        self.assertEqual(self_host.caddyfile_path("/etc/caddy/Caddyfile"), Path("/etc/caddy/Caddyfile"))

    def test_commands_edit_the_caddyfile_the_config_names(self):
        config = replace(self._config(), caddyfile="sites/Caddyfile")
        patches = dict(
            # Covered in CaddyLoadTests; here the path need not exist.
            refuse_to_create_caddyfile=mock.DEFAULT,
            preflight_caddyfile=mock.DEFAULT,
            command_stop=mock.DEFAULT,
            resolve_config=mock.DEFAULT,
            ports_after_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            save_config=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            start_server_prod=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        )
        seen = []
        original, original_named = self_host.CADDYFILE, self_host.CADDYFILE_NAMED
        try:
            with mock.patch.multiple(self_host, **patches) as mocks, mock.patch.object(
                self_host, "BIN_PATH", Path(__file__)
            ):
                mocks["resolve_config"].return_value = config
                mocks["ports_after_stop"].side_effect = lambda c: c
                mocks["poll_health"].return_value = True
                mocks["apply_caddyfile_safely"].side_effect = lambda *a: seen.append(self_host.CADDYFILE) or True
                self_host.command_start(None, None, "sites/Caddyfile")
                self.assertEqual(mocks["resolve_config"].call_args.args[3], "sites/Caddyfile")
            self.assertEqual(seen, [Path.home() / "sites/Caddyfile"])
            self.assertTrue(self_host.CADDYFILE_NAMED)
        finally:
            self_host.CADDYFILE, self_host.CADDYFILE_NAMED = original, original_named

    def test_main_routes_caddyfile_to_each_command(self):
        cases = [
            (["setup"], "command_setup", 3),
            (["redeploy"], "command_redeploy", 3),
            (["start"], "command_start", 2),
            (["dev-start"], "command_dev_start", 2),
            (["ship"], "command_ship", 4),
        ]
        for argv, target, index in cases:
            for extra, want in (([], None), (["--caddyfile", "scripts/Caddyfile"], "scripts/Caddyfile")):
                with self.subTest(argv=argv, extra=extra), mock.patch.object(self_host, target) as command:
                    self.assertEqual(self_host.main(argv + extra), 0)
                    self.assertEqual(command.call_args.args[index], want)


class ShipTargetMemoryTests(unittest.TestCase):
    def _with_config_file(self, tmp):
        config_dir = Path(tmp) / ".self-host"
        config_file = config_dir / "config.json"
        originals = (self_host.CONFIG_DIR, self_host.CONFIG_FILE)
        self_host.CONFIG_DIR, self_host.CONFIG_FILE = config_dir, config_file
        return originals

    def _restore(self, originals):
        self_host.CONFIG_DIR, self_host.CONFIG_FILE = originals

    def test_resolve_ship_target_uses_given_target_and_remembers_it(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            originals = self._with_config_file(tmp)
            try:
                resolved = self_host.resolve_ship_target("user@example.host", "custom-dir")
                self.assertEqual(resolved, self_host.ShipTarget("user@example.host", "custom-dir"))
                self_host.remember_ship_target("user@example.host", "custom-dir")

                again = self_host.resolve_ship_target(None, None)
                self.assertEqual(again, self_host.ShipTarget("user@example.host", "custom-dir"))
            finally:
                self._restore(originals)

    def test_resolve_ship_target_defaults_remote_dir_when_nothing_remembered(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            originals = self._with_config_file(tmp)
            try:
                resolved = self_host.resolve_ship_target("user@example.host", None)
                self.assertEqual(resolved, self_host.ShipTarget("user@example.host", self_host.DEFAULT_REMOTE_DIR))
            finally:
                self._restore(originals)

    def test_resolve_ship_target_raises_without_target_or_memory(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            originals = self._with_config_file(tmp)
            try:
                with self.assertRaises(RuntimeError):
                    self_host.resolve_ship_target(None, None)
            finally:
                self._restore(originals)

    def test_resolve_config_keeps_both_schemes_and_ship_unless_overridden(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            originals = self._with_config_file(tmp)
            try:
                self.assertFalse(self_host.resolve_config(None).both_schemes)
                self_host.save_config(
                    self_host.Config(
                        site=self_host.parse_site_url("https://game.example.test"),
                        mode=self_host.Mode.PROD,
                        ship=self_host.ShipTarget("user@example.host", "ajayeb"),
                        both_schemes=True,
                    )
                )
                kept = self_host.resolve_config(None)
                self.assertTrue(kept.both_schemes)
                self.assertEqual(kept.ship, self_host.ShipTarget("user@example.host", "ajayeb"))
                self.assertTrue(self_host.resolve_config("https://other.example.test").both_schemes)
                self.assertFalse(self_host.resolve_config(None, both_schemes=False).both_schemes)
                self.assertTrue(self_host.resolve_config(None, both_schemes=True).both_schemes)
            finally:
                self._restore(originals)

    def test_resolve_config_keeps_the_caddyfile_unless_overridden(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            originals = self._with_config_file(tmp)
            try:
                self.assertIsNone(self_host.resolve_config(None).caddyfile)
                self_host.save_config(
                    self_host.Config(
                        site=self_host.parse_site_url("https://game.example.test"),
                        mode=self_host.Mode.PROD,
                        caddyfile="sites/Caddyfile",
                    )
                )
                self.assertEqual(self_host.resolve_config(None).caddyfile, "sites/Caddyfile")
                self.assertEqual(self_host.resolve_config(None, caddyfile="/etc/Caddyfile").caddyfile, "/etc/Caddyfile")
                self.assertEqual(
                    self_host.resolve_config(None, both_schemes=True).caddyfile, "sites/Caddyfile"
                )
            finally:
                self._restore(originals)

    def test_remember_ship_target_preserves_existing_site_and_mode(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            originals = self._with_config_file(tmp)
            try:
                self_host.save_config(
                    self_host.Config(site=self_host.parse_site_url("https://game.example.test"), mode=self_host.Mode.DEV)
                )
                self_host.remember_ship_target("user@example.host", "ajayeb")
                reloaded = self_host.load_config()
                self.assertEqual(reloaded.site.origin, "https://game.example.test")
                self.assertEqual(reloaded.mode, self_host.Mode.DEV)
                self.assertEqual(reloaded.ship, self_host.ShipTarget("user@example.host", "ajayeb"))
            finally:
                self._restore(originals)


class PollRemoteVersionTests(unittest.TestCase):
    def test_poll_remote_version_returns_the_version_from_the_health_payload(self):
        import http.server
        import json as _json
        import threading

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(_json.dumps({"version": "v1.2.3"}).encode("utf-8"))

            def log_message(self, *args):
                pass

        server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
        port = server.server_address[1]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            version = self_host.poll_remote_version(f"http://127.0.0.1:{port}", timeout=2.0, interval=0.05)
            self.assertEqual(version, "v1.2.3")
        finally:
            server.shutdown()
            thread.join(timeout=2)

    def test_poll_remote_version_returns_none_when_nothing_listens(self):
        port = self_host.find_free_port(45100)
        self.assertIsNone(self_host.poll_remote_version(f"http://127.0.0.1:{port}", timeout=0.3, interval=0.05))


class CommandShipTests(unittest.TestCase):
    def _patched(self, **overrides):
        base = dict(
            resolve_ship_target=mock.DEFAULT,
            detect_remote_platform=mock.DEFAULT,
            git_version=mock.DEFAULT,
            cross_build=mock.DEFAULT,
            upload_to_remote=mock.DEFAULT,
            determine_deploy_command=mock.DEFAULT,
            deploy_remote=mock.DEFAULT,
            remember_ship_target=mock.DEFAULT,
            poll_remote_version=mock.DEFAULT,
        )
        return mock.patch.multiple(self_host, **base)

    def test_command_ship_warns_on_a_version_mismatch(self):
        with self._patched() as mocks:
            mocks["resolve_ship_target"].return_value = self_host.ShipTarget("user@example.host", "ajayeb")
            mocks["detect_remote_platform"].return_value = ("linux", "amd64")
            mocks["git_version"].return_value = "v1.0.0"
            mocks["cross_build"].return_value = Path("/tmp/ajayeb-linux-amd64")
            mocks["determine_deploy_command"].return_value = "setup"
            mocks["poll_remote_version"].return_value = "v0.9.0"

            stderr = io.StringIO()
            with redirect_stderr(stderr):
                self_host.command_ship(None, "https://game.example.test", None)
            self.assertIn("v0.9.0", stderr.getvalue())
            self.assertIn("v1.0.0", stderr.getvalue())
            mocks["remember_ship_target"].assert_called_once_with("user@example.host", "ajayeb")
            mocks["deploy_remote"].assert_called_once_with(
                "user@example.host", "ajayeb", "setup", "https://game.example.test", None, None
            )

    def test_command_ship_reports_matching_version(self):
        with self._patched() as mocks:
            mocks["resolve_ship_target"].return_value = self_host.ShipTarget("user@example.host", "ajayeb")
            mocks["detect_remote_platform"].return_value = ("linux", "amd64")
            mocks["git_version"].return_value = "v1.0.0"
            mocks["cross_build"].return_value = Path("/tmp/ajayeb-linux-amd64")
            mocks["determine_deploy_command"].return_value = "redeploy"
            mocks["poll_remote_version"].return_value = "v1.0.0"

            stdout = io.StringIO()
            with redirect_stdout(stdout):
                self_host.command_ship(None, "https://game.example.test", None)
            self.assertIn("v1.0.0", stdout.getvalue())

    def test_command_ship_skips_the_poll_without_a_url(self):
        with self._patched() as mocks:
            mocks["resolve_ship_target"].return_value = self_host.ShipTarget("user@example.host", "ajayeb")
            mocks["detect_remote_platform"].return_value = ("linux", "amd64")
            mocks["git_version"].return_value = "v1.0.0"
            mocks["cross_build"].return_value = Path("/tmp/ajayeb-linux-amd64")
            mocks["determine_deploy_command"].return_value = "setup"

            self_host.command_ship(None, None, None)
            mocks["poll_remote_version"].assert_not_called()

    def test_command_ship_treats_url_like_first_argument_as_the_url(self):
        """`ship URL` with a remembered target: argparse puts URL in the
        TARGET slot, so without this fix it would be used as the SSH
        destination (and fail safely with ssh exit 255) instead of reusing
        the remembered target."""
        with self._patched() as mocks:
            mocks["resolve_ship_target"].return_value = self_host.ShipTarget("user@example.host", "ajayeb")
            mocks["detect_remote_platform"].return_value = ("linux", "amd64")
            mocks["git_version"].return_value = "v1.0.0"
            mocks["cross_build"].return_value = Path("/tmp/ajayeb-linux-amd64")
            mocks["determine_deploy_command"].return_value = "redeploy"
            mocks["poll_remote_version"].return_value = "v1.0.0"

            self_host.command_ship("https://game.example.test", None, None)

            mocks["resolve_ship_target"].assert_called_once_with(None, None)
            mocks["deploy_remote"].assert_called_once_with(
                "user@example.host", "ajayeb", "redeploy", "https://game.example.test", None, None
            )
            mocks["poll_remote_version"].assert_called_once_with("https://game.example.test")

    def test_command_ship_raises_when_url_like_target_and_explicit_url_both_given(self):
        with self.assertRaises(RuntimeError):
            self_host.command_ship("https://a.test", "https://b.test", None)

    def test_command_ship_errors_clearly_when_url_like_target_and_nothing_remembered(self):
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            config_dir = Path(tmp) / ".self-host"
            config_file = config_dir / "config.json"
            orig_config_dir = self_host.CONFIG_DIR
            orig_config_file = self_host.CONFIG_FILE
            self_host.CONFIG_DIR = config_dir
            self_host.CONFIG_FILE = config_file
            try:
                with self.assertRaisesRegex(RuntimeError, "no TARGET"):
                    self_host.command_ship("https://game.example.test", None, None)
            finally:
                self_host.CONFIG_DIR = orig_config_dir
                self_host.CONFIG_FILE = orig_config_file


class DeployTargetsConfigTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.config_dir = Path(temporary.name) / ".self-host"
        self.config_dir.mkdir()
        self.path = self.config_dir / "deploy-targets.json"
        patcher = mock.patch.object(self_host, "CONFIG_DIR", self.config_dir)
        patcher.start()
        self.addCleanup(patcher.stop)
        patcher = mock.patch.dict(os.environ)
        patcher.start()
        self.addCleanup(patcher.stop)
        os.environ.pop("AJAYEB_DEPLOY_TARGETS", None)

    def valid(self, **fields):
        return {"target": "deploy@example.host", "url": "https://game.example.host", **fields}

    def test_default_path_and_environment_override(self):
        self.path.write_text(json.dumps({"targets": [self.valid()]}))
        targets = self_host.load_deploy_targets()
        self.assertEqual(targets, [self_host.DeployTarget(
            self_host.ShipTarget("deploy@example.host", "ajayeb"),
            self_host.Site("https", "game.example.host"),
        )])
        other = self.config_dir / "other.json"
        other.write_text(json.dumps({"targets": [self.valid(remote_dir="apps/game", url="http://other.example.host:8080/")]}))
        os.environ["AJAYEB_DEPLOY_TARGETS"] = str(other)
        target = self_host.load_deploy_targets()[0]
        self.assertEqual(target.destination.remote_dir, "apps/game")
        self.assertEqual(target.site.origin, "http://other.example.host:8080")

    def test_missing_config_fails_before_every_subprocess_or_build(self):
        stderr = io.StringIO()
        with mock.patch.object(self_host.subprocess, "run") as run, mock.patch.object(self_host, "cross_build") as build:
            with redirect_stderr(stderr):
                self.assertEqual(self_host.main(["ship-all"]), 1)
        run.assert_not_called()
        build.assert_not_called()
        self.assertIn(str(self.path), stderr.getvalue())
        self.assertIn('"targets"', stderr.getvalue())
        self.assertIn("example.host", stderr.getvalue())

    def test_all_invalid_config_is_checked_before_any_remote_or_build_boundary(self):
        invalid = [
            None, [], {}, {"targets": []}, {"targets": {}},
            {"targets": [None]}, {"targets": [self.valid(), {"target": "other.example.host"}]},
            {"targets": [self.valid(target=23)]}, {"targets": [self.valid(url=None)]},
            {"targets": [self.valid(remote_dir=None)]}, {"targets": [self.valid(target="")]},
            {"targets": [self.valid(target="-oProxyCommand=printf")]},
            {"targets": [self.valid(target="example.host:directory")]},
            {"targets": [self.valid(target="user@example.host;printf")]},
            {"targets": [self.valid(remote_dir="a dir")]},
            {"targets": [self.valid(remote_dir="-option")]},
            {"targets": [self.valid(url="https://secret:credential@example.host")]},
            {"targets": [self.valid(url="ftp://example.host")]},
            {"targets": [self.valid(url="https://game.example.host/path")]},
            {"targets": [self.valid(url="https://game.example.host?token=credential")]},
            {"targets": [self.valid(url="https://game.example.host:bad")]},
            {"targets": [self.valid(url="https://game.example.host:0")]},
            {"targets": [self.valid(url="https://game.example.host:")]},
            {"targets": [self.valid(url="https://game.example.host:65536")]},
            {"targets": [self.valid(url="https://game. example.host")]},
            {"targets": [self.valid(url="https://[::1]")]},
            {"targets": [self.valid(both_schemes=True)]},
            {"targets": [self.valid(), self.valid()]},
        ]
        for raw in invalid:
            with self.subTest(raw=raw):
                self.path.write_text(json.dumps(raw))
                stderr = io.StringIO()
                with mock.patch.object(self_host.subprocess, "run") as run, mock.patch.object(self_host, "cross_build") as build:
                    with redirect_stderr(stderr):
                        self.assertEqual(self_host.main(["ship-all"]), 1)
                run.assert_not_called()
                build.assert_not_called()
                self.assertIn(str(self.path), stderr.getvalue())
                self.assertIn("Expected format:", stderr.getvalue())
                self.assertNotIn("credential", stderr.getvalue().split("Expected format:")[0].replace("no credentials", ""))

    def test_bad_json_and_encoding_never_echo_config_contents(self):
        for contents in (b'{"credential": unclosed', b'\xff'):
            with self.subTest(contents=contents):
                self.path.write_bytes(contents)
                with self.assertRaisesRegex(RuntimeError, "Expected format") as caught:
                    self_host.load_deploy_targets()
                self.assertNotIn("credential", str(caught.exception))


class CommandShipAllTests(unittest.TestCase):
    def setUp(self):
        self.public_poll = self_host.poll_public_version
        self.targets = [
            self_host.DeployTarget(self_host.ShipTarget("deploy@one.example.host"), self_host.Site("https", "one.example.host")),
            self_host.DeployTarget(self_host.ShipTarget("deploy@two.example.host", "apps/game"), self_host.Site("http", "two.example.host", 8080)),
        ]
        self.output = io.StringIO()
        for redirect in (redirect_stdout(self.output), redirect_stderr(self.output)):
            redirect.__enter__()
            self.addCleanup(redirect.__exit__, None, None, None)
        patcher = mock.patch.multiple(
            self_host, load_deploy_targets=mock.DEFAULT, detect_remote_platform=mock.DEFAULT,
            git_version=mock.DEFAULT, cross_build=mock.DEFAULT, upload_to_remote=mock.DEFAULT,
            determine_deploy_command=mock.DEFAULT, deploy_remote=mock.DEFAULT,
            poll_host_version=mock.DEFAULT, poll_public_version=mock.DEFAULT,
            remember_ship_target=mock.DEFAULT,
        )
        self.mocks = patcher.start()
        self.addCleanup(patcher.stop)
        self.mocks["load_deploy_targets"].return_value = self.targets
        self.mocks["detect_remote_platform"].return_value = ("linux", "amd64")
        self.mocks["git_version"].return_value = "v-test"
        self.mocks["cross_build"].return_value = Path("artifact")
        self.mocks["determine_deploy_command"].side_effect = ["redeploy", "setup"]
        self.mocks["poll_host_version"].return_value = "v-test"
        self.mocks["poll_public_version"].return_value = "v-test"
        self.events = mock.Mock()
        for name, boundary in self.mocks.items():
            self.events.attach_mock(boundary, name)

    def test_one_build_then_sequential_deployments_and_both_health_checks(self):
        self.assertEqual(self_host.main(["ship-all"]), 0)
        self.mocks["cross_build"].assert_called_once_with("v-test", "linux", "amd64")
        calls = self.events.mock_calls
        names = [call[0] for call in calls]
        self.assertEqual(names, [
            "load_deploy_targets", "detect_remote_platform", "detect_remote_platform", "git_version", "cross_build",
            "upload_to_remote", "determine_deploy_command", "deploy_remote", "poll_host_version", "poll_public_version",
            "upload_to_remote", "determine_deploy_command", "deploy_remote", "poll_host_version", "poll_public_version",
        ])
        self.assertEqual(self.mocks["upload_to_remote"].call_args_list, [
            mock.call("deploy@one.example.host", "ajayeb", Path("artifact")),
            mock.call("deploy@two.example.host", "apps/game", Path("artifact")),
        ])
        self.assertEqual(self.mocks["deploy_remote"].call_args_list, [
            mock.call("deploy@one.example.host", "ajayeb", "redeploy", "https://one.example.host"),
            mock.call("deploy@two.example.host", "apps/game", "setup", "http://two.example.host:8080"),
        ])
        self.mocks["remember_ship_target"].assert_not_called()
        self.assertIn("target 1", self.output.getvalue())
        self.assertIn("target 2", self.output.getvalue())
        self.assertEqual(self.output.getvalue().count("remote-health=ok, local-health=ok"), 2)

    def test_remote_saved_flags_are_not_overridden(self):
        self.assertEqual(self_host.command_ship_all(), 0)
        for call in self.mocks["deploy_remote"].call_args_list:
            command = self_host.remote_deploy_command(*call.args)[-1]
            self.assertNotIn("--both-schemes", command)
            self.assertNotIn("--no-both-schemes", command)
            self.assertNotIn("--caddyfile", command)
            existing = self_host.Config(
                self_host.Site("https", "saved.example.host"), self_host.Mode.PROD,
                self_host.Ports(server=38955, web_dev=38956), both_schemes=True,
                caddyfile="config/Caddyfile",
            )
            with mock.patch.object(self_host, "load_config", return_value=existing):
                resolved = self_host.resolve_config(call.args[3], self_host.Mode.PROD)
            self.assertEqual(resolved.ports, existing.ports)
            self.assertTrue(resolved.both_schemes)
            self.assertEqual(resolved.caddyfile, existing.caddyfile)

    def test_mixed_architectures_refuse_the_entire_batch_before_building(self):
        self.mocks["detect_remote_platform"].side_effect = [("linux", "amd64"), ("linux", "arm64")]
        self.assertEqual(self_host.command_ship_all(), 1)
        self.mocks["cross_build"].assert_not_called()
        self.mocks["upload_to_remote"].assert_not_called()
        self.assertIn("same supported platform", self.output.getvalue())
        self.assertIn("target 1", self.output.getvalue())
        self.assertIn("target 2", self.output.getvalue())

    def test_preflight_failure_still_checks_and_reports_later_targets(self):
        self.mocks["detect_remote_platform"].side_effect = [RuntimeError("credential"), ("linux", "amd64")]
        self.assertEqual(self_host.command_ship_all(), 1)
        self.assertEqual(self.mocks["detect_remote_platform"].call_count, 2)
        self.mocks["cross_build"].assert_not_called()
        self.assertIn("target 2", self.output.getvalue())
        self.assertNotIn("credential", self.output.getvalue())

    def test_build_failure_reports_all_targets_and_never_uploads(self):
        self.mocks["cross_build"].side_effect = RuntimeError("credential")
        self.assertEqual(self_host.command_ship_all(), 1)
        self.mocks["upload_to_remote"].assert_not_called()
        self.assertEqual(self.output.getvalue().count("skipped: build failed"), 2)
        self.assertNotIn("credential", self.output.getvalue())

    def test_deploy_failure_still_probes_both_sides_and_ships_next_target(self):
        self.mocks["deploy_remote"].side_effect = [RuntimeError("credential"), None]
        self.assertEqual(self_host.command_ship_all(), 1)
        for name in ("upload_to_remote", "deploy_remote", "poll_host_version", "poll_public_version"):
            self.assertEqual(self.mocks[name].call_count, 2)
        self.assertIn("deploy=failed (RuntimeError)", self.output.getvalue())
        self.assertIn("target 2 (http://two.example.host:8080): deploy=ok", self.output.getvalue())
        self.assertNotIn("credential", self.output.getvalue())

    def test_remote_and_local_health_failures_continue_and_fail_overall(self):
        for boundary, failure in (("poll_host_version", None), ("poll_public_version", None),
                                  ("poll_host_version", "old-version"), ("poll_public_version", "old-version"),
                                  ("poll_host_version", RuntimeError("credential")),
                                  ("poll_public_version", TimeoutError("credential")),
                                  ("poll_public_version", self_host.InvalidHealthResponse("credential")),
                                  ("poll_public_version", self_host.HealthResponseTooLarge("credential"))):
            with self.subTest(boundary=boundary, failure=failure):
                # Reset only the changing mocks between subtests.
                self.mocks["determine_deploy_command"].side_effect = ["redeploy", "setup"]
                for name in ("poll_host_version", "poll_public_version"):
                    self.mocks[name].reset_mock(side_effect=True)
                    self.mocks[name].return_value = "v-test"
                self.mocks[boundary].side_effect = [failure, "v-test"]
                self.assertEqual(self_host.command_ship_all(), 1)
                self.assertEqual(self.mocks["poll_host_version"].call_count, 2)
                self.assertEqual(self.mocks["poll_public_version"].call_count, 2)
        self.assertNotIn("credential", self.output.getvalue())

    def test_actual_public_probe_timeout_still_checks_both_sides_and_deploys_next_target(self):
        clock = {"now": 0.0}
        processes = [mock.MagicMock(), mock.MagicMock()]
        for process in processes:
            process.__enter__.return_value = process
            process.wait.return_value = 0
            process.poll.return_value = 0
        processes[0].poll.return_value = None

        def readable(streams, _write, _errors, timeout):
            if streams == [processes[0].stdout]:
                # A stalled body consumes the entire request budget. Advance
                # the fake clock, never the real machine's clock.
                clock["now"] += timeout
                return [], [], []
            return streams, [], []

        self.mocks["poll_public_version"].side_effect = lambda url: self.public_poll(url, timeout=1)
        with mock.patch.object(self_host.subprocess, "Popen", side_effect=processes) as popen, \
             mock.patch.object(self_host.select, "select", side_effect=readable), \
             mock.patch.object(self_host.os, "read", side_effect=[b'{"version":"v-test"}', b""]), \
             mock.patch.object(self_host.time, "monotonic", side_effect=lambda: clock["now"]), \
             mock.patch.object(self_host.time, "sleep") as sleep:
            self.assertEqual(self_host.command_ship_all(), 1)
        self.assertEqual(popen.call_count, 2)
        processes[0].kill.assert_called_once_with()
        processes[0].wait.assert_called_once_with()
        sleep.assert_called_once_with(0)
        for name in ("deploy_remote", "poll_host_version", "poll_public_version"):
            self.assertEqual(self.mocks[name].call_count, 2)
        self.assertIn("target 1 (https://one.example.host): deploy=ok, remote-health=ok, local-health=failed (TimeoutError)",
                      self.output.getvalue())
        self.assertIn("target 2 (http://two.example.host:8080): deploy=ok, remote-health=ok, local-health=ok",
                      self.output.getvalue())

    def test_cli_rejects_target_and_ship_flag_arguments_before_work(self):
        self.assertIn("ship-all", self_host.build_parser().format_help())
        for args in (["ship-all", "example.host"], ["ship-all", "--both-schemes"], ["ship-all", "--remote-dir", "apps"]):
            with self.subTest(args=args), self.assertRaises(SystemExit) as caught:
                self_host.main(args)
            self.assertEqual(caught.exception.code, 2)
        self.mocks["load_deploy_targets"].assert_not_called()


class PublicHealthVersionTests(unittest.TestCase):
    def setUp(self):
        self.process = mock.MagicMock()
        self.process.__enter__.return_value = self.process
        self.process.stdout.fileno.return_value = 101
        self.process.poll.return_value = 0
        self.process.wait.return_value = 0
        self.popen = self.patch(self_host.subprocess, "Popen", return_value=self.process)
        self.select = self.patch(self_host.select, "select", side_effect=lambda streams, *_: (streams, [], []))
        self.read = self.patch(self_host.os, "read", side_effect=[b'{"version":"v-test"}', b""])
        self.patch(self_host.time, "monotonic", return_value=0)

    def patch(self, owner, name, **kwargs):
        patcher = mock.patch.object(owner, name, **kwargs)
        boundary = patcher.start()
        self.addCleanup(patcher.stop)
        return boundary

    def test_curl_and_pipe_reader_share_the_total_deadline_and_bounded_reads(self):
        self.assertEqual(self_host.public_health_version("https://game.example.host:8443/", 1), "v-test")
        cmd = self.popen.call_args.args[0]
        self.assertEqual(cmd[:2], ["curl", "--disable"])
        self.assertEqual(cmd[-1], "https://game.example.host:8443/health")
        self.assertEqual(float(cmd[cmd.index("--max-time") + 1]), 1)
        self.assertEqual(float(cmd[cmd.index("--connect-timeout") + 1]), 1)
        self.assertEqual(self.popen.call_args.kwargs, {"stdout": subprocess.PIPE, "stderr": subprocess.DEVNULL})
        self.assertTrue(all(call.args[-1] <= 1 for call in self.select.call_args_list))
        self.assertTrue(all(call.args[1] <= 4096 for call in self.read.call_args_list))
        self.process.wait.assert_any_call(timeout=1)
        self.process.kill.assert_not_called()

    def test_stalled_body_times_out_and_curl_is_killed_and_reaped(self):
        self.select.side_effect = None
        self.select.return_value = ([], [], [])
        self.process.poll.return_value = None
        with self.assertRaises(TimeoutError):
            self_host.public_health_version("https://game.example.host", 1)
        self.read.assert_not_called()
        self.process.kill.assert_called_once_with()
        self.process.wait.assert_called_once_with()
        self.process.__exit__.assert_called_once()

    def test_expired_deadline_starts_no_process(self):
        with self.assertRaises(TimeoutError):
            self_host.public_health_version("https://game.example.host", 0)
        self.popen.assert_not_called()

    def test_pipe_failure_or_interruption_kills_and_reaps_curl(self):
        self.process.poll.return_value = None
        for failure in (OSError("private"), KeyboardInterrupt()):
            with self.subTest(failure=type(failure).__name__):
                self.process.reset_mock()
                self.select.side_effect = failure
                with self.assertRaises(type(failure)):
                    self_host.public_health_version("https://game.example.host", 1)
                self.process.kill.assert_called_once_with()
                self.process.wait.assert_called_once_with()

    def test_process_exit_cannot_report_success_after_the_deadline(self):
        clock = {"now": 0.0}

        def finish(*, timeout):
            self.assertEqual(timeout, 1)
            clock["now"] = 1.1
            return 0

        # Preserve a normal cleanup wait after the delayed first wait.
        self.process.wait.side_effect = lambda **kwargs: finish(**kwargs) if kwargs else 0
        with mock.patch.object(self_host.time, "monotonic", side_effect=lambda: clock["now"]):
            with self.assertRaises(TimeoutError):
                self_host.public_health_version("https://game.example.host", 1)
        self.process.wait.assert_any_call(timeout=1)
        self.assertEqual(self.process.wait.call_args, mock.call())

    def test_trickling_body_cannot_extend_the_absolute_deadline(self):
        self.read.side_effect = [b"{", b'"']
        self.process.poll.return_value = None
        with mock.patch.object(self_host.time, "monotonic", side_effect=[0, 0.1, 0.6, 1]):
            with self.assertRaises(TimeoutError):
                self_host.public_health_version("https://game.example.host", 1)
        self.assertEqual(self.read.call_count, 2)
        self.assertEqual(self.select.call_count, 2)
        self.assertAlmostEqual(self.select.call_args_list[0].args[-1], 0.9)
        self.assertAlmostEqual(self.select.call_args_list[1].args[-1], 0.4)
        self.process.kill.assert_called_once_with()
        self.process.wait.assert_called_once_with()

    def test_oversized_body_is_rejected_without_waiting_for_eof_or_content_length(self):
        self.read.side_effect = [b"x" * 4096] * 4 + [b"x"]
        self.process.poll.return_value = None
        with self.assertRaises(self_host.HealthResponseTooLarge):
            self_host.public_health_version("https://game.example.host", 1)
        self.assertEqual([call.args[1] for call in self.read.call_args_list], [4096, 4096, 4096, 4096, 1])
        self.process.kill.assert_called_once_with()
        self.process.wait.assert_called_once_with()

    def test_exactly_the_size_limit_is_accepted(self):
        body = b'{"version":"v-test"}'
        body += b" " * (self_host.SHIP_HEALTH_MAX_BYTES - len(body))
        self.read.side_effect = [body[i:i + 4096] for i in range(0, len(body), 4096)] + [b""]
        self.assertEqual(self_host.public_health_version("https://game.example.host", 1), "v-test")
        self.assertEqual(self.read.call_args.args[1], 1)

    def test_malformed_wrong_type_and_deeply_nested_bodies_fail_safely(self):
        for body in (b"not json", b"[]", b'{"version":23}', b"{}", b'{"version":""}', b"\xff", b"[" * 1200):
            with self.subTest(body=body):
                self.read.side_effect = [body, b""]
                with self.assertRaises(self_host.InvalidHealthResponse):
                    self_host.public_health_version("https://game.example.host", 1)
        self.process.kill.assert_not_called()

    def test_curl_errors_and_wait_timeout_have_safe_failure_types_and_cleanup(self):
        for returncode, error in ((22, RuntimeError), (28, TimeoutError)):
            with self.subTest(returncode=returncode):
                self.read.side_effect = [b"", b""]
                self.process.wait.return_value = returncode
                with self.assertRaises(error):
                    self_host.public_health_version("https://game.example.host", 1)
        self.read.side_effect = [b""]
        self.process.wait.side_effect = [subprocess.TimeoutExpired("curl", 1), 0]
        self.process.poll.return_value = None
        with self.assertRaises(TimeoutError):
            self_host.public_health_version("https://game.example.host", 1)
        self.process.kill.assert_called_once_with()
        self.assertEqual(self.process.wait.call_args, mock.call())


class PollPublicVersionTests(unittest.TestCase):
    def test_retry_budget_includes_requests_and_delays_without_resetting_deadline(self):
        with mock.patch.object(self_host, "public_health_version", side_effect=[TimeoutError("private"), "v-test"]) as probe, \
             mock.patch.object(self_host.time, "monotonic", side_effect=[10, 10, 10, 15, 16, 16]), \
             mock.patch.object(self_host.time, "sleep") as sleep:
            self.assertEqual(self_host.poll_public_version("https://game.example.host", timeout=7), "v-test")
        self.assertEqual(probe.call_args_list, [mock.call("https://game.example.host", 15), mock.call("https://game.example.host", 17)])
        sleep.assert_called_once_with(1)

    def test_expired_budget_reports_the_last_failure_without_an_extra_attempt(self):
        for error in (TimeoutError("private"), self_host.InvalidHealthResponse("private"),
                      self_host.HealthResponseTooLarge("private"), OSError("private")):
            with self.subTest(error=type(error).__name__):
                with mock.patch.object(self_host, "public_health_version", side_effect=error) as probe, \
                     mock.patch.object(self_host.time, "monotonic", side_effect=[0, 0, 0, 1, 1]), \
                     mock.patch.object(self_host.time, "sleep") as sleep:
                    with self.assertRaises(type(error)):
                        self_host.poll_public_version("https://game.example.host", timeout=1)
                probe.assert_called_once_with("https://game.example.host", 1)
                sleep.assert_called_once_with(0)


class PollHostVersionTests(unittest.TestCase):
    def test_curl_resolves_the_origin_on_loopback_with_correct_scheme_port_and_sni(self):
        for scheme, port, expected in (("https", None, 443), ("http", None, 80), ("http", 8080, 8080), ("https", 8443, 8443)):
            with self.subTest(scheme=scheme, port=port):
                site = self_host.Site(scheme, "game.example.host", port)
                with mock.patch.object(self_host.subprocess, "run") as run:
                    run.return_value = subprocess.CompletedProcess([], 0, stdout='{"version":"v-test"}')
                    self.assertEqual(self_host.poll_host_version("deploy@example.host", site, timeout=2), "v-test")
                cmd = run.call_args.args[0]
                self.assertEqual(cmd[:6], ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "deploy@example.host"])
                curl = shlex.split(cmd[-1])
                self.assertEqual(curl[curl.index("--resolve") + 1], f"game.example.host:{expected}:127.0.0.1")
                self.assertEqual(curl[curl.index("--noproxy") + 1], "*")
                self.assertEqual(curl[-1], site.origin + "/health")
                self.assertLessEqual(float(curl[curl.index("--max-time") + 1]), 2)
                self.assertLessEqual(run.call_args.kwargs["timeout"], 2)

    def test_timeouts_process_failures_and_malformed_payloads_are_bounded(self):
        failures = [subprocess.TimeoutExpired("ssh", 1), OSError("unavailable"),
                    subprocess.CompletedProcess([], 22, stdout=""),
                    subprocess.CompletedProcess([], 0, stdout="not json"),
                    subprocess.CompletedProcess([], 0, stdout="[]"),
                    subprocess.CompletedProcess([], 0, stdout='{"version":23}')]
        for failure in failures:
            with self.subTest(failure=failure):
                with mock.patch.object(self_host.subprocess, "run", side_effect=[failure]) as run, \
                     mock.patch.object(self_host.time, "monotonic", side_effect=[0, 0, 1, 1]), \
                     mock.patch.object(self_host.time, "sleep"):
                    self.assertIsNone(self_host.poll_host_version("example.host", self_host.Site("https", "game.example.host"), timeout=1))
                self.assertEqual(run.call_count, 1)

    def test_failed_probe_is_retried_within_the_deadline(self):
        with mock.patch.object(self_host.subprocess, "run", side_effect=[
            subprocess.CompletedProcess([], 22, stdout=""),
            subprocess.CompletedProcess([], 0, stdout='{"version":"v-test"}'),
        ]) as run, mock.patch.object(self_host.time, "monotonic", side_effect=[0, 0, 0.2, 0.3]), \
             mock.patch.object(self_host.time, "sleep"):
            self.assertEqual(self_host.poll_host_version("example.host", self_host.Site("http", "game.example.host"), timeout=1), "v-test")
        self.assertEqual(run.call_count, 2)


class CandidateRoomCheckTests(unittest.TestCase):
    def report(self, **updates):
        report = dict(format="ajayeb-room-check/1", rules=1, rooms=2, lobby=0,
                      playing=2, finished=0, archived=0, playable=2, wouldArchive=0,
                      liveWouldArchive=0, refreshableHashes=1, schemaErrors=0, errors=0)
        report.update(updates)
        return report

    def check(self, report, code=0, force=False):
        result = subprocess.CompletedProcess([], code, stdout=json.dumps(report), stderr="private diagnostic")
        with mock.patch.object(self_host.subprocess, "run", return_value=result) as run:
            self_host.check_candidate_rooms(Path("/tmp/candidate"), force)
        return run

    def test_check_runs_read_only_candidate_mode_and_reports_refreshable_rooms(self):
        with redirect_stdout(io.StringIO()) as out:
            run = self.check(self.report())
        self.assertIn("refreshableHashes=1", out.getvalue())
        self.assertEqual(run.call_args.args[0], [str(Path("/tmp/candidate").resolve()), "--check-rooms", "--data", str(self_host.DATA_DIR)])

    def test_preflight_allows_archived_folder_counts_without_force(self):
        # The real candidate's folder scan is covered by the Go CLI test;
        # deployment must accept its archive-only report without force.
        with redirect_stdout(io.StringIO()) as out:
            self.check(self.report(rooms=4, playing=0, archived=4, playable=0, refreshableHashes=0))
        self.assertIn("archived=4", out.getvalue())

    def test_live_loss_refuses_and_force_is_explicit_in_output(self):
        report = self.report(playable=1, wouldArchive=1, liveWouldArchive=1)
        with self.assertRaisesRegex(RuntimeError, "--force-room-archive"):
            self.check(report, 3)
        with redirect_stderr(io.StringIO()) as out:
            self.check(report, 3, force=True)
        self.assertIn("accepts archiving 1", out.getvalue())

    def test_operational_schema_and_malformed_reports_never_force(self):
        for report, code in ((self.report(errors=1, schemaErrors=1), 1),
                             (self.report(), 1), ({}, 0), (self.report(errors=False), 0),
                             (self.report(liveWouldArchive=1), 0)):
            with self.subTest(report=report), self.assertRaises(RuntimeError):
                self.check(report, code, force=True)
        for exc in (PermissionError("private detail"), subprocess.TimeoutExpired("candidate", 120)):
            with mock.patch.object(self_host.subprocess, "run", side_effect=exc):
                with self.assertRaisesRegex(RuntimeError, "candidate room check failed") as ctx:
                    self_host.check_candidate_rooms(Path("/tmp/candidate"), True)
                self.assertNotIn("private detail", str(ctx.exception))

    def test_preflight_refusal_never_stops_or_swaps(self):
        for command in (self_host.command_setup, self_host.command_redeploy):
            with mock.patch.multiple(self_host, resolve_config=mock.DEFAULT, use_caddyfile=mock.DEFAULT,
                                     ensure_tools=mock.DEFAULT, check_prebuilt_binary=mock.DEFAULT,
                                     prepare_prod_caddy=mock.DEFAULT, command_stop=mock.DEFAULT,
                                     swap_in_binary=mock.DEFAULT, archive_rooms=mock.DEFAULT,
                                     check_candidate_rooms=mock.Mock(side_effect=RuntimeError("room risk"))) as mocks:
                with self.assertRaisesRegex(RuntimeError, "room risk"):
                    command(None, "/tmp/candidate")
                mocks["command_stop"].assert_not_called()
                mocks["swap_in_binary"].assert_not_called()
                mocks["archive_rooms"].assert_not_called()

    def test_second_scan_happens_after_stop_and_restarts_previous_on_failure(self):
        config = self_host.Config(site=self_host.parse_site_url("https://game.example.test"), mode=self_host.Mode.PROD)
        with tempfile.TemporaryDirectory() as tmp:
            binary = Path(tmp) / "old-binary"
            binary.write_bytes(b"old")
            calls = []
            def check(*args):
                calls.append("check")
                if len(calls) > 1:
                    raise RuntimeError("new move makes candidate incompatible")
            with mock.patch.object(self_host, "BIN_PATH", binary), mock.patch.multiple(
                self_host, check_candidate_rooms=mock.Mock(side_effect=check),
                command_stop=mock.Mock(side_effect=lambda: calls.append("stop")),
                wait_for_port_free=mock.Mock(return_value=True),
                start_server_prod=mock.Mock(side_effect=lambda *args: calls.append("restart")),
                poll_health=mock.Mock(return_value=True),
            ) as mocks:
                with self.assertRaisesRegex(RuntimeError, "new move"):
                    self_host.stop_for_candidate(config, Path("/tmp/candidate"), False)
                self.assertEqual(calls, ["check", "stop", "check", "restart"])
                self_host.start_server_prod.assert_called_once_with(config.ports, config.site)
                self.assertEqual(binary.read_bytes(), b"old")

    def test_success_checks_both_sides_of_stop_and_passes_force(self):
        config = self_host.Config(site=self_host.parse_site_url("https://game.example.test"), mode=self_host.Mode.PROD)
        events = mock.Mock()
        with mock.patch.multiple(self_host, check_candidate_rooms=mock.DEFAULT, command_stop=mock.DEFAULT,
                                 wait_for_port_free=mock.Mock(return_value=True)) as mocks:
            events.attach_mock(mocks["check_candidate_rooms"], "check")
            events.attach_mock(mocks["command_stop"], "stop")
            self_host.stop_for_candidate(config, Path("candidate"), True)
        self.assertEqual(events.mock_calls, [mock.call.check(Path("candidate"), True), mock.call.stop(),
                                            mock.call.check(Path("candidate"), True)])

    def test_override_routed_through_every_cli_and_remote_command(self):
        cases = (("setup", "command_setup"), ("redeploy", "command_redeploy"),
                 ("ship", "command_ship"), ("ship-all", "command_ship_all"))
        for command, function in cases:
            with mock.patch.object(self_host, function, return_value=0) as mocked:
                self.assertEqual(self_host.main([command, "--force-room-archive"]), 0)
                self.assertIs(mocked.call_args.args[-1], True)
        remote = self_host.remote_deploy_command("deploy@example.host", "game", "redeploy", None,
                                               force_room_archive=True)
        self.assertIn("--force-room-archive", remote[-1])
        self.assertNotIn("--force-room-archive", self_host.remote_deploy_command("deploy@example.host", "game", "setup", None)[-1])

    def test_ship_forwards_override(self):
        with CommandShipTests()._patched() as mocks:
            mocks["resolve_ship_target"].return_value = self_host.ShipTarget("deploy@example.host", "game")
            mocks["detect_remote_platform"].return_value = ("linux", "amd64")
            mocks["git_version"].return_value = "v-test"
            self_host.command_ship(None, None, None, force_room_archive=True)
            self.assertIs(mocks["deploy_remote"].call_args.kwargs["force_room_archive"], True)


    def test_ship_all_forwards_override_to_every_target(self):
        targets = [self_host.DeployTarget(self_host.ShipTarget("deploy@example.host", "game"),
                                         self_host.parse_site_url("https://game.example.test"))]
        with mock.patch.multiple(self_host, load_deploy_targets=mock.Mock(return_value=targets),
                                 detect_remote_platform=mock.Mock(return_value=("linux", "amd64")),
                                 git_version=mock.Mock(return_value="v-test"), cross_build=mock.DEFAULT,
                                 upload_to_remote=mock.DEFAULT,
                                 determine_deploy_command=mock.Mock(return_value="redeploy"),
                                 deploy_remote=mock.DEFAULT,
                                 poll_host_version=mock.Mock(return_value="v-test"),
                                 poll_public_version=mock.Mock(return_value="v-test")) as mocks:
            self.assertEqual(self_host.command_ship_all(True), 0)
            self.assertIs(mocks["deploy_remote"].call_args.kwargs["force_room_archive"], True)


if __name__ == "__main__":
    unittest.main()
