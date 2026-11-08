#!/usr/bin/env python3
"""Check reviewed skill templates without needing the skill checkout on CI."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_SKILL = Path.home() / 'code/skills/board-game-dev-skill'
TEMPLATES = Path('skills/board-game-dev-cool-breeze/templates')


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def safe_path(root: Path, value: str) -> Path:
    if not isinstance(value, str) or not value or '\\' in value:
        raise ValueError(f'invalid relative path: {value!r}')
    parts = PurePosixPath(value)
    if parts.is_absolute() or '..' in parts.parts or parts.as_posix() != value or value == '.':
        raise ValueError(f'invalid relative path: {value!r}')
    path = root / value
    if not path.resolve().is_relative_to(root.resolve()):
        raise ValueError(f'path escapes root: {value}')
    return path


def read_manifest(path: Path) -> dict:
    data = json.loads(path.read_text())
    if not isinstance(data, dict) or data.get('version') != 1:
        raise ValueError('unsupported manifest version')
    if not re.fullmatch(r'[0-9a-f]{40}', data.get('skill_commit', '')):
        raise ValueError('skill_commit must be a full git commit hash')
    if not isinstance(data.get('files'), list) or not data['files']:
        raise ValueError('manifest needs a nonempty files array')
    seen = set()
    for entry in data['files']:
        if not isinstance(entry, dict):
            raise ValueError('file entry must be an object')
        for key in ('template', 'path', 'class', 'sha256', 'reason'):
            if not isinstance(entry.get(key), str) or not entry[key]:
                raise ValueError(f'file entry needs {key}')
        if entry['class'] not in ('vendored', 'adapted'):
            raise ValueError(f"unknown class: {entry['class']}")
        if not re.fullmatch(r'[0-9a-f]{64}', entry['sha256']):
            raise ValueError(f"invalid sha256: {entry['template']}")
        if entry['template'] in seen:
            raise ValueError(f"duplicate template: {entry['template']}")
        seen.add(entry['template'])
    return data


def check(root: Path, manifest: Path, skill: Path | None = None) -> tuple[list[str], int]:
    data = read_manifest(manifest)
    errors = []
    template_root = skill / TEMPLATES if skill is not None else None
    if template_root is not None and not template_root.is_dir():
        raise ValueError(f'skill templates directory missing: {template_root}')
    vendored = 0
    for entry in data['files']:
        local = safe_path(root, entry['path'])
        # Validate template paths even offline.
        upstream = safe_path(template_root or root, entry['template'])
        if not local.is_file():
            errors.append(f"missing local file: {entry['path']}")
        elif entry['class'] == 'vendored':
            vendored += 1
            if digest(local) != entry['sha256']:
                errors.append(f"vendored drift: {entry['path']} differs from reviewed template")
        if template_root is None:
            continue
        if not upstream.is_file():
            errors.append(f"missing template: {entry['template']}")
            continue
        current = digest(upstream)
        if current != entry['sha256']:
            errors.append(f"skill moved: {entry['template']} ({entry['class']}; re-vendor or re-review)")
        if entry['class'] == 'vendored' and local.is_file() and digest(local) != current:
            errors.append(f"skill mismatch: {entry['path']} differs from current {entry['template']}")
    return errors, vendored


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['check'])
    parser.add_argument('--manifest', type=Path, default=ROOT / 'tools/skill-templates.json')
    source = parser.add_mutually_exclusive_group()
    source.add_argument('--offline', action='store_true', help='only check recorded hashes; never inspect a skill checkout')
    source.add_argument('--skill', type=Path, help='skill repository; otherwise use the default checkout if present')
    args = parser.parse_args(argv)
    skill = None if args.offline else (args.skill.expanduser() if args.skill else
                                      DEFAULT_SKILL if DEFAULT_SKILL.is_dir() else None)
    try:
        errors, count = check(ROOT, args.manifest, skill)
    except (OSError, ValueError, TypeError) as error:
        print(f'vendor-check: {error}', file=sys.stderr)
        return 1
    if errors:
        for error in errors:
            print(f'vendor-check: {error}', file=sys.stderr)
        return 1
    print(f'vendor-check: {count} vendored files match; ' +
          ('reviewed skill templates match' if skill else 'offline hashes only'))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
