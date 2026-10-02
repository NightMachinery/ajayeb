import io
import unittest
from contextlib import redirect_stdout
from pathlib import Path

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


if __name__ == "__main__":
    unittest.main()
