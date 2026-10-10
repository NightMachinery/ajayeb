#!/usr/bin/env node
// tools/art/generate.mjs
//
// Generates one bitmap attempt per asset, strictly one at a time, through
// Codex `image_gen`. For each chosen asset:
//   1. Builds the prompt from art/style.md's prefix and kind template,
//      art/manifest.json's subject/accent and (on a retry) promptAmend.
//   2. Snapshots the files under ${CODEX_HOME:-~/.codex}/generated_images/.
//   3. Runs `codex exec` (ephemeral, read-only sandbox, a scratch -C dir,
//      --enable image_generation, -o for the last message) asking it to
//      call image_gen exactly once.
//   4. Finds the newest PNG that appeared since the snapshot, verifies it
//      came back the shape asked for (same aspect ratio, at least the
//      requested size), and copies it to
//      <AJAYEB_RUN_DIR>/art-work/raw/<kind>/<slug>/<attempt>.png.
//   5. Records the attempt in art/manifest.json (status "generated",
//      attempt incremented), never overwriting an "approved" asset.
//
// The codex lane: unless it already runs inside
// `python3 tools/heavy.py run --kind codex` (AJAYEB_HEAVY_KIND=codex), the
// script re-executes itself under that wrapper, so the machine-wide
// one-codex-at-a-time rule (AGENTS.md "Assets") holds even when a caller
// forgets the wrapper. --dry-run runs no codex and skips the lane.
//
// Reference images (`-i`/`--image`) are **off by default**. M9.B3's pilot
// found a same-subject anchor makes Codex return a near-copy of the
// reference rather than a new illustration, and the style prefix alone
// already keeps the style consistent across subjects; pass --ref to opt
// back in for a cross-subject style anchor.
//
// Usage:
//   node tools/art/generate.mjs --ids card/lumber-yard
//   node tools/art/generate.mjs --kind token --limit 3
//   node tools/art/generate.mjs --ids card/lumber-yard --amend "warmer light"
//   node tools/art/generate.mjs --ids card/lumber-yard --ref
//
// Flags:
//   --ids a,b,c   generate exactly these ids (comma-separated), in order
//   --kind KIND   restrict to one kind (card|wonder|token|back|texture|
//                 hero|avatar|victory|icon) when --ids is not given
//   --limit N     generate at most N assets (default 1; this script is the
//                 "strictly one asset at a time" step, so a caller sweeping
//                 a kind still gets one codex run per asset, sequentially)
//   --status S    with --kind, only assets whose status is S (default:
//                 "todo" and "rejected", so assets a review round sent
//                 back with a promptAmend are retried too)
//   --amend TEXT  append TEXT as this run's promptAmend before generating
//                 (overrides the manifest's stored promptAmend for --ids)
//   --ref         pass art/refs/<kind>-anchor.{jpg,png} to Codex as a style
//                 anchor, when one exists (default: no reference image at
//                 all; see "Reference images" above)
//   --dry-run     print the prompt and the codex command, run nothing

import { spawn, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { readdir, stat, mkdir, copyFile, readFile } from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';

import {
  repoRoot,
  rawDir,
  loadManifest,
  saveManifest,
  idKind,
  idSlug,
  styleDocPath,
  refsDir,
  imageSize,
  die,
} from './lib.mjs';

const CODEX_BIN = process.env.CODEX_BIN || path.join(os.homedir(), '.local', 'bin', 'codex');
const CODEX_HOME = process.env.CODEX_HOME || path.join(os.homedir(), '.codex');
const GENERATED_IMAGES_DIR = path.join(CODEX_HOME, 'generated_images');
const TIMEOUT_MS = 6 * 60 * 1000;

function parseArgs(argv) {
  const args = { limit: 1, status: null };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--ids') args.ids = argv[++i].split(',').map((s) => s.trim()).filter(Boolean);
    else if (a === '--kind') args.kind = argv[++i];
    else if (a === '--limit') args.limit = Number(argv[++i]);
    else if (a === '--status') args.status = argv[++i];
    else if (a === '--amend') args.amend = argv[++i];
    else if (a === '--ref') args.ref = true;
    else if (a === '--dry-run') args.dryRun = true;
    else die(`unknown argument "${a}"`, 2);
  }
  return args;
}

/** Reads art/style.md and pulls out the verbatim style prefix (the
 * blockquote under "## Style prefix (verbatim)") and the kind template line
 * for one kind, so the two files never drift apart. */
async function loadStylePieces(kind) {
  const md = await readFile(styleDocPath, 'utf8');

  const prefixHeading = '## Style prefix (verbatim)';
  const prefixStart = md.indexOf(prefixHeading);
  if (prefixStart === -1) die('art/style.md has no "## Style prefix (verbatim)" section');
  const quoteLines = md
    .slice(prefixStart)
    .split('\n')
    .filter((l) => l.startsWith('> '))
    .map((l) => l.slice(2));
  const prefix = quoteLines.join(' ').replace(/\s+/g, ' ').trim();

  const templateRe = new RegExp(`\\*\\*${kind}\\*\\*:\\s*"([^"]+)"`);
  const m = templateRe.exec(md);
  if (!m) die(`art/style.md has no kind template for "${kind}"`);
  // Markdown line-wraps the template for readability and backtick-quotes
  // its placeholders; prompts are one line with no stray backticks.
  const template = m[1].replace(/\s+/g, ' ').replaceAll('`', '').trim();
  return { prefix, template };
}

function fillTemplate(template, asset) {
  return template.replaceAll('{subject}', asset.subject).replaceAll('{accent}', asset.accent);
}

async function buildPrompt(asset, amendOverride) {
  const { prefix, template } = await loadStylePieces(asset.kind);
  const body = fillTemplate(template, asset);
  const amend = amendOverride ?? asset.promptAmend;
  const amendText = amend ? ` ${amend}.` : '';
  return `${prefix} ${body}${amendText}`;
}

async function snapshotGeneratedImages() {
  try {
    const entries = await readdir(GENERATED_IMAGES_DIR, { recursive: true, withFileTypes: true });
    const files = new Map();
    for (const e of entries) {
      if (!e.isFile() || !e.name.toLowerCase().endsWith('.png')) continue;
      const full = path.join(e.parentPath ?? e.path, e.name);
      const st = await stat(full);
      files.set(full, st.mtimeMs);
    }
    return files;
  } catch (err) {
    if (err.code === 'ENOENT') return new Map();
    throw err;
  }
}

async function findNewestNewPng(before) {
  const after = await snapshotGeneratedImages();
  let newest = null;
  for (const [file, mtime] of after) {
    if (before.has(file)) continue; // existed before this run
    if (!newest || mtime > newest.mtime) newest = { file, mtime };
  }
  return newest?.file ?? null;
}

/** "landscape 1536x1024", "square 1024x1024" or "portrait 1024x1536",
 * from the manifest's per-kind generation size. */
function sizePhrase(gen) {
  const [w, h] = gen.split('x').map(Number);
  const shape = w > h ? 'landscape' : w < h ? 'portrait' : 'square';
  return `${shape} ${gen}`;
}

function codexInstruction(prompt, gen) {
  return (
    `Call the image_gen tool exactly once with this prompt (${sizePhrase(gen)}): ` +
    `${prompt} Do nothing else. Reply DONE.`
  );
}

async function runCodex(prompt, gen, scratchDir, refImage) {
  await mkdir(scratchDir, { recursive: true });
  const outFile = path.join(scratchDir, 'last.txt');
  const instruction = codexInstruction(prompt, gen);

  const args = [
    'exec',
    '--ephemeral',
    '--skip-git-repo-check',
    '-s',
    'read-only',
    '-C',
    scratchDir,
    '--enable',
    'image_generation',
    '-o',
    outFile,
  ];
  // `--image=<path>` (one token), never `-i <path>` (two tokens): codex
  // exec's `-i, --image <FILE>...` is variadic, so a space-separated value
  // greedily swallows the next positional too, eating the prompt itself
  // (reproduced directly against codex exec, not just this script).
  if (refImage) args.push(`--image=${refImage}`);
  args.push(instruction);

  return new Promise((resolve, reject) => {
    const child = spawn(CODEX_BIN, args, { stdio: ['ignore', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => (stdout += d));
    child.stderr.on('data', (d) => (stderr += d));
    const timer = setTimeout(() => {
      child.kill('SIGTERM');
      reject(new Error(`codex exec timed out after ${TIMEOUT_MS / 1000}s`));
    }, TIMEOUT_MS);
    child.on('error', (err) => {
      clearTimeout(timer);
      reject(err);
    });
    child.on('close', (code) => {
      clearTimeout(timer);
      resolve({ code, stdout, stderr, outFile });
    });
  });
}

async function generateOne(asset, amendOverride, dryRun, useRef) {
  const kind = asset.kind;
  const slug = idSlug(asset.id);
  const prompt = await buildPrompt(asset, amendOverride);
  const scratchDir = path.join(os.tmpdir(), `ajayeb-art-gen-${process.pid}-${Date.now()}`);
  // Off by default (M9.B3): a same-subject reference made Codex return a
  // near-copy of the anchor rather than a new illustration in the pilot,
  // and the style prefix alone keeps the look consistent. --ref opts back
  // in for a cross-subject style anchor.
  const refImage = useRef ? await firstExistingRef(kind) : null;

  const cmdForLog = [
    CODEX_BIN,
    'exec',
    '--ephemeral',
    '--skip-git-repo-check',
    '-s',
    'read-only',
    '-C',
    scratchDir,
    '--enable',
    'image_generation',
    '-o',
    path.join(scratchDir, 'last.txt'),
    ...(refImage ? [`--image=${refImage}`] : []),
    JSON.stringify(codexInstruction(prompt, asset.gen)),
  ].join(' ');

  if (dryRun) {
    process.stdout.write(`--- ${asset.id} ---\nprompt: ${prompt}\ncommand: ${cmdForLog}\n`);
    return null;
  }

  process.stdout.write(`generate.mjs: ${asset.id} (attempt ${asset.attempt + 1})\n  prompt: ${prompt}\n`);

  const before = await snapshotGeneratedImages();
  const started = Date.now();
  const result = await runCodex(prompt, asset.gen, scratchDir, refImage);
  const tookMs = Date.now() - started;

  if (result.code !== 0) {
    process.stderr.write(`generate.mjs: codex exec exited ${result.code}\n${result.stderr}\n`);
    return { ok: false, tookMs, cmd: cmdForLog };
  }

  const newPng = await findNewestNewPng(before);
  if (!newPng) {
    process.stderr.write(
      `generate.mjs: codex exec finished but no new PNG appeared under ${GENERATED_IMAGES_DIR}\n` +
        `  codex's last message: ${await safeRead(result.outFile)}\n`,
    );
    return { ok: false, tookMs, cmd: cmdForLog };
  }

  // Verify every generation's actual pixel size before it reaches raw/
  // and the manifest. What matters is the aspect ratio and enough pixels
  // for the shipped crop: Codex image_gen returns every square request
  // (all 9 pilot tokens included) as 1254x1254, which is fine, while a
  // reference-image run in the pilot returned a 1254x1254 square for a
  // 1536x1024 landscape request, which process.mjs would crop wrongly.
  // So reject a different aspect ratio (more than 1% off) or a smaller
  // image, and accept a same-shape image that is larger.
  const [wantW, wantH] = asset.gen.split('x').map(Number);
  const got = await imageSize(newPng);
  const aspectOff = Math.abs(got.w / got.h - wantW / wantH) / (wantW / wantH);
  if (aspectOff > 0.01 || got.w < wantW || got.h < wantH) {
    process.stderr.write(
      `generate.mjs: codex image_gen returned ${got.w}x${got.h}, expected ${wantW}x${wantH} or a larger image of the same shape (${newPng}); rejecting this attempt\n`,
    );
    return { ok: false, tookMs, cmd: cmdForLog };
  }

  // The next attempt number is one past the larger of the manifest's
  // counter and the highest attempt file already on disk, so an asset
  // generated outside this script (the M9.B1 pilot) is never overwritten.
  const destDir = path.join(rawDir, kind, slug);
  const onDisk = await readdir(destDir).catch(() => []);
  const maxOnDisk = Math.max(0, ...onDisk.map((f) => /^(\d+)\.png$/.exec(f)).filter(Boolean).map((m) => Number(m[1])));
  const attempt = Math.max(asset.attempt, maxOnDisk) + 1;
  await mkdir(destDir, { recursive: true });
  const dest = path.join(destDir, `${attempt}.png`);
  await copyFile(newPng, dest);
  const bytes = (await stat(dest)).size;

  return { ok: true, dest, attempt, bytes, tookMs, prompt, cmd: cmdForLog, source: newPng };
}

async function firstExistingRef(kind) {
  // art/refs/ anchors are optional; the pilot runs without one.
  for (const ext of ['jpg', 'png']) {
    const candidate = path.join(refsDir, `${kind}-anchor.${ext}`);
    try {
      await stat(candidate);
      return candidate;
    } catch {
      // try the next extension
    }
  }
  return null;
}

/** Re-runs this script under `heavy.py run --kind codex` unless it already
 * holds the codex lane, and exits with the wrapper's status. */
function ensureCodexLane(argv) {
  if (process.env.AJAYEB_HEAVY_KIND === 'codex') return;
  const self = fileURLToPath(import.meta.url);
  const heavy = path.join(repoRoot, 'tools', 'heavy.py');
  const r = spawnSync('python3', [heavy, 'run', '--kind', 'codex', '--', process.execPath, self, ...argv], {
    stdio: 'inherit',
  });
  if (r.error) die(`could not start tools/heavy.py: ${r.error.message}`);
  process.exit(r.status ?? 1);
}

async function safeRead(file) {
  try {
    return (await readFile(file, 'utf8')).trim();
  } catch {
    return '(no output file)';
  }
}

async function main() {
  const argv = process.argv.slice(2);
  const args = parseArgs(argv);
  if (!args.dryRun) ensureCodexLane(argv);
  const manifest = await loadManifest();
  const assetsById = manifest.assets;

  let ids;
  if (args.ids) {
    ids = args.ids;
    for (const id of ids) if (!assetsById[id]) die(`"${id}" is not in art/manifest.json`);
  } else {
    ids = Object.values(assetsById)
      .filter((a) => (args.kind ? a.kind === args.kind : true))
      .filter((a) => (args.status ? a.status === args.status : a.status === 'todo' || a.status === 'rejected'))
      .slice(0, args.limit)
      .map((a) => a.id);
  }
  if (args.ids) ids = ids.slice(0, args.limit);

  if (!ids.length) {
    process.stdout.write('generate.mjs: nothing to generate (no matching assets)\n');
    return;
  }

  for (const id of ids) {
    const asset = assetsById[id];
    if (asset.status === 'approved' && !args.ids) {
      process.stdout.write(`generate.mjs: skipping ${id}, already approved\n`);
      continue;
    }
    const outcome = await generateOne(asset, args.ids ? args.amend : undefined, args.dryRun, args.ref);
    if (!outcome || args.dryRun) continue;

    if (outcome.ok) {
      asset.status = 'generated';
      asset.attempt = outcome.attempt;
      if (args.amend) asset.promptAmend = args.amend;
      await saveManifest(manifest);
      process.stdout.write(
        `generate.mjs: ${id} -> ${outcome.dest} (${outcome.bytes} bytes, ${(outcome.tookMs / 1000).toFixed(1)}s)\n`,
      );
    } else {
      process.stdout.write(`generate.mjs: ${id} failed after ${(outcome.tookMs / 1000).toFixed(1)}s; manifest left unchanged\n`);
    }
  }
}

main().catch((err) => die(err.stack || String(err)));
