#!/usr/bin/env node
// tools/art/apply-review.mjs
//
// Applies a reviewer's decisions.jsonl to art/manifest.json. Each line is
// one JSON object: { id, attempt, verdict, reasons, promptAmend }.
//   verdict "approve"  -> status "approved" (process.mjs may now ship it)
//   verdict "reject"   -> status "todo" if under 3 attempts (promptAmend is
//                         appended for generate.mjs's next run), else
//                         "needs-human"
// A decision for an id whose current `attempt` does not match the
// decision's `attempt` is skipped (stale: a newer attempt exists that this
// decision never saw), and reported.
//
// Usage:
//   node tools/art/apply-review.mjs path/to/decisions.jsonl
//   node tools/art/apply-review.mjs path/to/decisions.jsonl --dry-run

import { readFile } from 'node:fs/promises';

import { loadManifest, saveManifest, die } from './lib.mjs';

const MAX_ATTEMPTS = 3;

function parseArgs(argv) {
  const args = {};
  for (const a of argv) {
    if (a === '--dry-run') args.dryRun = true;
    else if (!args.file) args.file = a;
    else die(`unknown argument "${a}"`, 2);
  }
  if (!args.file) die('usage: apply-review.mjs <decisions.jsonl> [--dry-run]', 2);
  return args;
}

function parseDecisions(text) {
  const out = [];
  const lines = text.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (!line) continue;
    let d;
    try {
      d = JSON.parse(line);
    } catch (err) {
      die(`decisions.jsonl line ${i + 1}: invalid JSON (${err.message})`);
    }
    if (!d.id || !d.verdict) die(`decisions.jsonl line ${i + 1}: needs "id" and "verdict"`);
    if (d.verdict !== 'approve' && d.verdict !== 'reject') {
      die(`decisions.jsonl line ${i + 1}: verdict must be "approve" or "reject", got "${d.verdict}"`);
    }
    out.push(d);
  }
  return out;
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const text = await readFile(args.file, 'utf8');
  const decisions = parseDecisions(text);
  const manifest = await loadManifest();

  const summary = { approved: 0, rejected: 0, needsHuman: 0, skippedStale: 0, skippedUnknown: 0 };

  for (const d of decisions) {
    const asset = manifest.assets[d.id];
    if (!asset) {
      process.stderr.write(`apply-review.mjs: "${d.id}" is not in art/manifest.json, skipping\n`);
      summary.skippedUnknown++;
      continue;
    }
    if (typeof d.attempt === 'number' && d.attempt !== asset.attempt) {
      process.stderr.write(
        `apply-review.mjs: ${d.id} decision is for attempt ${d.attempt} but the manifest is on attempt ${asset.attempt}, skipping (stale)\n`,
      );
      summary.skippedStale++;
      continue;
    }

    if (d.verdict === 'approve') {
      asset.status = 'approved';
      asset.reasons = d.reasons || [];
      summary.approved++;
    } else {
      if (asset.attempt >= MAX_ATTEMPTS) {
        asset.status = 'needs-human';
        summary.needsHuman++;
      } else {
        asset.status = 'todo';
        summary.rejected++;
      }
      if (d.promptAmend) asset.promptAmend = d.promptAmend;
      asset.reasons = d.reasons || [];
    }
  }

  process.stdout.write(
    `apply-review.mjs: approved ${summary.approved}, rejected ${summary.rejected}, needs-human ${summary.needsHuman}` +
      `${summary.skippedStale ? `, skipped ${summary.skippedStale} stale` : ''}` +
      `${summary.skippedUnknown ? `, skipped ${summary.skippedUnknown} unknown` : ''}\n`,
  );

  if (args.dryRun) {
    process.stdout.write('apply-review.mjs: --dry-run, art/manifest.json not written\n');
    return;
  }
  await saveManifest(manifest);
}

main().catch((err) => die(err.stack || String(err)));
