"""Drift detection uses real fixture bytes, independent of the installed skill."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / 'tools/skill_templates.py'
spec = importlib.util.spec_from_file_location('skill_templates', SCRIPT)
vendor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(vendor)


class TemplateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.skill = self.root / 'skill'
        self.templates = self.skill / vendor.TEMPLATES
        self.templates.mkdir(parents=True)
        self.manifest = self.root / 'manifest.json'
        self.entries = []
        for name, kind in [('shared.py', 'vendored'), ('integrated.py', 'adapted')]:
            (self.root / name).write_text('reviewed\n')
            (self.templates / name).write_text('reviewed\n')
            self.entries.append(dict(template=name, path=name, **{'class': kind},
                                     sha256=vendor.digest(self.root / name), reason='fixture'))
        self.save()

    def save(self):
        self.manifest.write_text(json.dumps(dict(version=1, skill_commit='a' * 40, files=self.entries)))

    def check(self, skill=None):
        return vendor.check(self.root, self.manifest, skill)[0]

    def test_offline_accepts_adaptations_and_checks_vendored_bytes(self):
        (self.root / 'integrated.py').write_text('game integration\n')
        self.assertEqual(self.check(), [])
        (self.root / 'shared.py').write_text('accidental edit\n')
        self.assertIn('vendored drift: shared.py', self.check()[0])

    def test_reports_every_changed_template_including_adapted(self):
        for name in ('shared.py', 'integrated.py'):
            (self.templates / name).write_text('upstream fix\n')
        errors = self.check(self.skill)
        self.assertEqual(sum('skill moved:' in x for x in errors), 2)
        self.assertTrue(any('skill mismatch: shared.py' in x for x in errors))

    def test_matching_local_and_new_upstream_still_requires_review(self):
        for parent in (self.root, self.templates):
            (parent / 'shared.py').write_text('new upstream\n')
        errors = self.check(self.skill)
        self.assertTrue(any('skill moved:' in x for x in errors))
        self.assertFalse(any('skill mismatch:' in x for x in errors))

    def test_missing_files_and_explicit_missing_checkout_fail(self):
        (self.root / 'integrated.py').unlink()
        (self.templates / 'shared.py').unlink()
        errors = self.check(self.skill)
        self.assertTrue(any('missing local file:' in x for x in errors))
        self.assertTrue(any('missing template:' in x for x in errors))
        with self.assertRaisesRegex(ValueError, 'directory missing'):
            self.check(self.root / 'absent')

    def test_rejects_bad_paths_hashes_classes_and_duplicate_templates(self):
        for key, value in [('path', '../outside'), ('template', '/absolute'), ('template', 'a/../b'),
                           ('sha256', 'bad'), ('class', 'unknown')]:
            with self.subTest(key=key, value=value):
                old = self.entries[0][key]
                self.entries[0][key] = value
                self.save()
                with self.assertRaises(ValueError):
                    self.check()
                self.entries[0][key] = old
        self.entries.append(dict(self.entries[0]))
        self.save()
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            self.check()

    def test_symlink_cannot_escape_root(self):
        (self.root / 'shared.py').unlink()
        (self.root / 'shared.py').symlink_to(SCRIPT)
        with self.assertRaisesRegex(ValueError, 'escapes root'):
            self.check()

    def test_offline_never_consults_default_skill(self):
        with patch.object(vendor, 'ROOT', self.root), patch.object(vendor, 'DEFAULT_SKILL', self.skill):
            (self.templates / 'integrated.py').write_text('upstream changed')
            self.assertEqual(vendor.main(['check', '--offline', '--manifest', str(self.manifest)]), 0)
            self.assertEqual(vendor.main(['check', '--manifest', str(self.manifest)]), 1)

    def test_cli_has_no_dependency_on_skill_checkout(self):
        result = subprocess.run([sys.executable, str(SCRIPT), 'check', '--help'],
                                capture_output=True, text=True)
        self.assertEqual(result.returncode, 0)
        self.assertIn('--offline', result.stdout)
