"""The vendored tools honor adopter settings and existing environment contracts."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ProjectToolsTests(unittest.TestCase):
    def test_configured_guard_namespace_controls_nested_state_and_defaults(self):
        with tempfile.TemporaryDirectory() as tmp:
            config = json.loads((ROOT / 'tools/project.json').read_text())
            config.update(env_prefix='EXISTING_', run_dir_env='EXISTING_RUN_DIR', run_dir='~/state/game')
            settings = Path(tmp) / 'project.json'
            settings.write_text(json.dumps(config))
            code = '''import importlib.util, json, sys
spec = importlib.util.spec_from_file_location('guard_fixture', sys.argv[1])
m = importlib.util.module_from_spec(spec); sys.modules[spec.name] = m; spec.loader.exec_module(m)
g = m.Guard(env={'HOME': sys.argv[2], 'EXISTING_MAX_SLOTS': '1'})
print(json.dumps([g.configured_run_dir, g.max_slots(), g.child_env('browser', True, {'browser'}), m.EX_TEMPFAIL]))
'''
            env = dict(os.environ, BGD_PROJECT_CONFIG=str(settings))
            result = subprocess.run([sys.executable, '-c', code, str(ROOT / 'tools/heavy.py'), tmp],
                                    env=env, capture_output=True, text=True, check=True)
            run_dir, slots, child, refused = json.loads(result.stdout)
            self.assertEqual(run_dir, str(Path(tmp) / 'state/game'))
            self.assertEqual(slots, 1)
            self.assertEqual(refused, 75)
            self.assertEqual(child['EXISTING_HEAVY_KIND'], 'browser')
            self.assertEqual(child['EXISTING_HEAVY_HELD'], '1')
            self.assertEqual(child['EXISTING_HEAVY_LOCKS'], 'browser')
            self.assertNotIn('BGD_HEAVY_HELD', child)

    def test_art_specs_and_run_directory_are_loaded_from_project_config(self):
        result = subprocess.run(['node', '--input-type=module', '-e',
            "import { project, runDir } from './tools/config.mjs'; "
            "import { KIND_SPEC, runDir as artRunDir } from './tools/art/lib.mjs'; "
            "console.log(JSON.stringify([KIND_SPEC === project.art.kinds, runDir === artRunDir, Object.keys(KIND_SPEC)]));"],
            cwd=ROOT, capture_output=True, text=True, check=True)
        same_specs, same_run_dir, kinds = json.loads(result.stdout)
        self.assertTrue(same_specs)
        self.assertTrue(same_run_dir)
        self.assertEqual(len(kinds), 9)

    def test_generated_checker_detects_additions_removals_and_changes(self):
        spec = importlib.util.spec_from_file_location('generated_fixture', ROOT / 'tools/check_generated.py')
        generated = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(generated)
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)
            original = out / 'old.txt'
            original.write_text('first')
            self.assertEqual(generated.main(['check', str(out), '--', sys.executable, '-c', 'pass']), 0)
            edit = 'from pathlib import Path; import sys; p=Path(sys.argv[1]); '
            for code in ["(p/'old.txt').write_text('changed')", "(p/'new.txt').write_text('new')", "(p/'old.txt').unlink()"]:
                self.assertEqual(generated.main(['check', str(out), '--', sys.executable, '-c', edit + code, str(out)]), 1)
            self.assertEqual(generated.main(['check', str(out), '--', sys.executable, '-c', 'raise SystemExit(7)']), 7)
