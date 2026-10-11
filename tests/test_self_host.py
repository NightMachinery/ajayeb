import io
import os
import shlex
import subprocess
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path
from unittest import mock

import self_host


class SelfHostTests(unittest.TestCase):
    def test_parse_url_rejects_paths(self):
        with self.assertRaises(ValueError):
            self_host.parse_site_url("https://ajayeb.example.com/room/demo")

    def test_parse_url_defaults_to_https_ajayeb(self):
        site = self_host.parse_site_url(None)
        self.assertEqual(site.scheme, "https")
        self.assertEqual(site.host, "ajayeb.pinky.lilf.ir")
        self.assertEqual(site.origin, "https://ajayeb.pinky.lilf.ir")

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
                ["caddy", "reload", "--config", str(self_host.CADDYFILE)],
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
                "user@example.host", "ajayeb", "setup", "https://game.example.test"
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
                "user@example.host", "ajayeb", "redeploy", "https://game.example.test"
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


if __name__ == "__main__":
    unittest.main()
