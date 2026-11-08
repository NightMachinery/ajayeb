# Skill templates

Ajayeb shares tool implementations with the public board-game-dev-cool-breeze
skill. [The manifest](../../tools/skill-templates.json) records the reviewed
skill base commit, each template's SHA-256, its local path and the reason for
its classification. The hashes identify the exact reviewed bytes, including
compatibility fixes waiting to be committed upstream. The upstream MIT notice
is preserved in [skill-templates.LICENSE](../../tools/skill-templates.LICENSE).

## Vendored files

Vendored means byte-for-byte identical to the reviewed template. This includes
the heavy guard and its full tests, the shared config reader, generated-output
checker, Chrome launch/CDP/screenshot/origin helpers, vector rasterizer, and
unchanged clipboard, identity-test and tooltip-placement helpers. See the
manifest for the complete file list.

[tools/project.json](../../tools/project.json) holds Ajayeb's values: ports
38951 through 38959, the `AJAYEB_` environment prefix, `AJAYEB_RUN_DIR` and its
existing private default, admitted kinds and locks, Chrome's Playwright package
and theme options, and all nine bitmap kind specs. The art library reads those
specs directly while preserving its existing API. `BGD_PROJECT_CONFIG` selects
an alternate config file before either the Python or Node tools initialize.
`AJAYEB_HEAVY_WAIT`, `HEAVY_PRIORITY`, `HEAVY_NOTIFY` and inherited guard state
retain their existing behavior. `light` still bypasses the slot/mode gate;
refused heavy jobs still exit 75.

Chrome reads the remembered serving URL before the configured local fallback.
An explicit `--origin`, a full `--url`, or `AJAYEB_ORIGIN` can select a different
frontend. A dedicated browser profile retains the `ajayeb-chrome-` prefix.

## Adapted files

Adapted means reviewed against the template but maintained locally:

- `self_host.py` and its tests retain production `ship`, rollback, room backups,
  boot integration and longer startup health checks. The standalone hosting
  template cannot replace that deployment path.
- The Makefile retains Ajayeb's engine, code-generation, web and browser gates.
  The standalone generated-output checker is available, but the existing
  `gen-check` target continues to use its established snapshot workflow.
- The bitmap scripts retain game namespaces, cross-game aliases, generated
  catalogs, approved-attempt manifests, crop/mask and high-DPI handling, app
  icons and their PNG exception. The skill's inherited theme-pack manifests
  require a separate migration. Kind specs remain in the legacy schema in
  `project.json`; replacing a bitmap script requires adapting that schema too.
- The Chrome probe retains Haftgane's rules-drawer coverage exemption. The
  template's removal of localStorage dumps is also applied locally.
- The remaining web helpers retain typed translations, stored colour preference,
  first-paint theme setup, reconnecting sockets, game rules/diagrams, lazy rules
  overlays, game preview stacks and sheet focus/gesture behavior. The template's
  optional interactive tooltip controls have no Ajayeb caller to port.

Changes carried back to the templates make the environment namespace,
Playwright resolution and screenshot options configurable, keep the art generator
in its inherited guard namespace, and preserve input
selection and modal-dialog copying in the clipboard fallback.

## Checking and updating

Run `make vendor-check` for the offline check used by `make verify-quick` and CI.
It needs only Python's standard library and the committed manifest. Adapted
files may differ; vendored files must match their recorded hashes.

Run `python3 tools/skill_templates.py check` to also inspect the default skill
checkout when present. Use `check --skill DIR` for another checkout. Every
recorded template that changes, including an adapted one, requires review.
A vendored file that differs from the current skill template is reported too.
Use `check --offline` to explicitly ignore local skill availability.

To re-vendor:

1. Review the upstream diff and the manifest's reason for every affected file.
   Port any required compatibility fixes upstream before copying. Preserve
   current production features and environment contracts.
2. Copy vendored files unchanged. Keep project values in `tools/project.json`.
   For adapted files, port applicable fixes and retain the local integration.
3. Update the manifest's base commit and affected template hashes only after
   that review. Record any pending upstream fixes in the handoff. Never refresh
   hashes just to silence a failed check.
4. Run the full heavy tests before relying on a replacement guard, then Chrome
   tool tests, art tests and `make verify-quick`. Follow the detached heavy-job
   procedure in AGENTS.md. Run changed standalone template tests from scratch
   copies with both generic and adopter settings.
