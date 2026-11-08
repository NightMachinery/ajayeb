"""Dev port propagation and readiness, without launching long-running services."""

import errno
import io
import shlex
import socket
import unittest
from contextlib import redirect_stdout
from unittest import mock

import self_host


class DevStartupTests(unittest.TestCase):
    def setUp(self):
        self.config = self_host.Config(
            site=self_host.parse_site_url("http://game.example.test"), mode=self_host.Mode.DEV)
        patcher = mock.patch.multiple(
            self_host,
            resolve_config=mock.Mock(return_value=self.config),
            use_caddyfile=mock.DEFAULT,
            refuse_to_create_caddyfile=mock.DEFAULT,
            command_stop=mock.DEFAULT,
            ensure_tools=mock.DEFAULT,
            find_go=mock.DEFAULT,
            install_web=mock.DEFAULT,
            save_config=mock.DEFAULT,
            apply_caddyfile_safely=mock.DEFAULT,
            tmuxnew=mock.DEFAULT,
            poll_health=mock.DEFAULT,
            start_web_dev=mock.DEFAULT,
            announce_serving=mock.DEFAULT,
        )
        self.mocks = patcher.start()
        self.mocks["poll_health"].return_value = True
        self.addCleanup(patcher.stop)

    def test_busy_default_port_reaches_air_caddy_vite_and_health_as_one_allocated_port(self):
        # If another service already owns the default, it is our occupied-port
        # fixture. Otherwise reserve it ourselves, without starting a backend.
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as occupied:
            try:
                occupied.bind(("127.0.0.1", 38951))
                occupied.listen()
            except OSError as exc:
                if exc.errno != errno.EADDRINUSE:
                    raise
            self.assertFalse(self_host.port_free(38951))
            self_host.command_dev_start(None)

        saved = self.mocks["save_config"].call_args.args[0]
        self.assertNotEqual(saved.ports.server, 38951)
        self.assertNotEqual(saved.ports.server, saved.ports.web_dev)
        self.mocks["tmuxnew"].assert_called_once_with(
            self_host.DEV_SERVER_SESSION,
            f"go tool air -- --addr 127.0.0.1:{saved.ports.server}",
            cwd=self_host.SERVER_DIR,
            env_args=self_host.tmux_env_args() + ["-e", f"GOTOOLCHAIN={self_host.GO_TOOLCHAIN}"],
        )
        # Air's -- delimiter sends --addr to the built backend, retaining
        # --dev and the other entrypoint arguments from server/.air.toml.
        command = shlex.split(self.mocks["tmuxnew"].call_args.args[1])
        self.assertEqual(command[command.index("--") + 1:],
                         ["--addr", f"127.0.0.1:{saved.ports.server}"])
        self.mocks["apply_caddyfile_safely"].assert_called_once_with(
            saved.site, self_host.Mode.DEV, saved.ports, saved.both_schemes)
        self.mocks["poll_health"].assert_called_once_with(
            saved.ports.server, timeout=90.0, progress=True)
        self.mocks["start_web_dev"].assert_called_once_with(saved.ports)
        _, vite_env = self_host.web_dev_command(saved.ports)
        self.assertEqual(vite_env["AJAYEB_SERVER"], f"http://127.0.0.1:{saved.ports.server}")
        self.mocks["announce_serving"].assert_called_once_with(saved)

    def test_health_precedes_starting_vite_and_announcing_the_site(self):
        order = mock.Mock()
        for name in ("tmuxnew", "poll_health", "start_web_dev", "announce_serving"):
            order.attach_mock(self.mocks[name], name)
        with mock.patch.object(self_host, "allocate_ports", return_value=self.config.ports):
            self_host.command_dev_start(None)
        self.assertEqual([call[0] for call in order.mock_calls],
                         ["tmuxnew", "poll_health", "start_web_dev", "announce_serving"])

    def test_failed_health_never_claims_the_site_is_ready(self):
        self.mocks["poll_health"].return_value = False
        with mock.patch.object(self_host, "allocate_ports", return_value=self.config.ports), \
             redirect_stdout(io.StringIO()) as output:
            with self.assertRaisesRegex(RuntimeError, "/health did not respond within 90s"):
                self_host.command_dev_start(None)
        self.assertEqual(output.getvalue(), "")
        self.mocks["start_web_dev"].assert_not_called()
        self.mocks["announce_serving"].assert_not_called()

    def test_health_exception_never_announces_the_site(self):
        self.mocks["poll_health"].side_effect = OSError("probe failed")
        with mock.patch.object(self_host, "allocate_ports", return_value=self.config.ports):
            with self.assertRaisesRegex(OSError, "probe failed"):
                self_host.command_dev_start(None)
        self.mocks["announce_serving"].assert_not_called()
