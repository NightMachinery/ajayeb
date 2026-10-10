#!/usr/bin/env node
// tools/art/apply-review.mjs
//
// Applies a reviewer's decisions.jsonl to art/manifest.json. Each line is
// one JSON object: { id, attempt, verdict, reasons, promptAmend }.
//   verdict "approve"      -> status "approved", and `approvedAttempt` set
//                             to the decision's attempt (process.mjs ships
//                             that attempt's raw file). The attempt may be
//                             an earlier one than the latest, when the
//                             reviewer prefers it to every retry since;
//                             its raw file must exist.
//   verdict "reject"       -> status "rejected" if the asset has had fewer
//                             than --max-attempts attempts (its promptAmend
//                             is stored for generate.mjs's next run, which
//                             retries "rejected" assets), else
//                             "needs-human"
//   verdict "needs-human"  -> status "needs-human" directly, with the
//                             reasons, whatever the attempt count: for an
//                             asset the reviewer judges a prompt change
//                             cannot fix (a subject that needs rewriting,
//                             a decision only the owner can make)
// The stale-decision rule: a "reject" for an attempt other than the
// manifest's current one is skipped (a newer attempt exists that this
// decision never saw), and reported. An "approve" or "needs-human" may
// name an earlier attempt, but never one after the current one. A
// decision with no "attempt" is about the current one.
//
// Usage:
//   node tools/art/apply-review.mjs path/to/decisions.jsonl
//   node tools/art/apply-review.mjs path/to/decisions.jsonl --dry-run
//   node tools/art/apply-review.mjs decisions.jsonl --max-attempts 4
//   node tools/art/apply-review.mjs decisions.jsonl --manifest /scratch/manifest.json
//
// Flags:
//   --max-attempts N  a rejection at N or more attempts becomes
//                     "needs-human" instead of "rejected" (default 3)
//   --manifest PATH   apply to this manifest file instead of
//                     art/manifest.json (a scratch copy, for a proof that
//                     touches nothing committed); AJAYEB_ART_MANIFEST does
//                     the same
//   --dry-run         print what would change, write nothing
//   --help            print this usage and exit

import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';

import {
  loadManifest,
  saveManifest,
  setManifestPath,
  manifestLabel,
  rawAttemptPath,
  displayPath,
  printUsage,
  die,
} from './lib.mjs';

const VERDICTS = ['approve', 'reject', 'needs-human'];

function parseArgs(argv) {
  const args = { maxAttempts: 3 };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--help' || a === '-h') printUsage(import.meta.url);
    else if (a === '--dry-run') args.dryRun = true;
    else if (a === '--max-attempts') {
      args.maxAttempts = Number(argv[++i]);
      if (!Number.isInteger(args.maxAttempts) || args.maxAttempts < 1) {
        die('--max-attempts needs a whole number of 1 or more', 2);
      }
    } else if (a === '--manifest') setManifestPath(argv[++i]);
    else if (a.startsWith('-')) die(`unknown argument "${a}" (see --help)`, 2);
    else if (!args.file) args.file = a;
    else die(`unexpected extra argument "${a}" (see --help)`, 2);
  }
  if (!args.file) {
    die('usage: apply-review.mjs <decisions.jsonl> [--max-attempts N] [--manifest PATH] [--dry-run] (see --help)', 2);
  }
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
    if (!VERDICTS.includes(d.verdict)) {
      die(
        `decisions.jsonl line ${i + 1}: verdict must be one of ${VERDICTS.map((v) => `"${v}"`).join(', ')}, got "${d.verdict}"`,
      );
    }
    if (d.attempt !== undefined && (!Number.isInteger(d.attempt) || d.attempt < 1)) {
      die(`decisions.jsonl line ${i + 1}: "attempt" must be a whole number of 1 or more`);
    }
    out.push(d);
  }
  return out;
}

function note(msg) {
  process.stdout.write(`apply-review.mjs: ${msg}\n`);
}

function skip(msg) {
  process.stderr.write(`apply-review.mjs: ${msg}\n`);
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const text = await readFile(args.file, 'utf8');
  const decisions = parseDecisions(text);
  const manifest = await loadManifest();

  const summary = { approved: 0, rejected: 0, needsHuman: 0, skippedStale: 0, skippedUnknown: 0, skippedMissing: 0 };

  for (const d of decisions) {
    const asset = manifest.assets[d.id];
    if (!asset) {
      skip(`"${d.id}" is not in ${manifestLabel()}, skipping`);
      summary.skippedUnknown++;
      continue;
    }
    const attempt = d.attempt ?? asset.attempt;

    if (!attempt || attempt > asset.attempt) {
      skip(`${d.id}: the decision names attempt ${attempt}, but the manifest has ${asset.attempt} attempt(s), skipping`);
      summary.skippedStale++;
      continue;
    }
    if (d.verdict === 'reject' && attempt !== asset.attempt) {
      skip(`${d.id}: the rejection is for attempt ${attempt}, but the manifest is on attempt ${asset.attempt}, skipping (stale)`);
      summary.skippedStale++;
      continue;
    }

    if (d.verdict === 'approve') {
      const raw = rawAttemptPath(d.id, attempt);
      if (!existsSync(raw)) {
        skip(`${d.id}: the approval names attempt ${attempt}, but its raw file ${raw} is missing, skipping`);
        summary.skippedMissing++;
        continue;
      }
      asset.status = 'approved';
      asset.approvedAttempt = attempt;
      asset.reasons = d.reasons || [];
      summary.approved++;
      const earlier = attempt < asset.attempt ? `, an earlier attempt than the latest (${asset.attempt})` : '';
      note(`${d.id}: approved attempt ${attempt}${earlier}; process.mjs will ship ${displayPath(raw)}`);
    } else if (d.verdict === 'needs-human') {
      asset.status = 'needs-human';
      delete asset.approvedAttempt;
      asset.reasons = d.reasons || [];
      if (d.promptAmend) asset.promptAmend = d.promptAmend;
      summary.needsHuman++;
      const why = asset.reasons.length ? `: ${asset.reasons.join('; ')}` : '';
      note(`${d.id}: needs-human at attempt ${attempt}, set directly by the reviewer${why}`);
    } else {
      delete asset.approvedAttempt;
      if (asset.attempt >= args.maxAttempts) {
        asset.status = 'needs-human';
        summary.needsHuman++;
        note(`${d.id}: rejected at attempt ${asset.attempt}, the cap of ${args.maxAttempts}, so needs-human`);
      } else {
        asset.status = 'rejected';
        summary.rejected++;
        note(`${d.id}: rejected at attempt ${asset.attempt} of at most ${args.maxAttempts}`);
      }
      if (d.promptAmend) asset.promptAmend = d.promptAmend;
      asset.reasons = d.reasons || [];
    }
  }

  note(
    `approved ${summary.approved}, rejected ${summary.rejected}, needs-human ${summary.needsHuman}` +
      `${summary.skippedStale ? `, skipped ${summary.skippedStale} stale` : ''}` +
      `${summary.skippedUnknown ? `, skipped ${summary.skippedUnknown} unknown` : ''}` +
      `${summary.skippedMissing ? `, skipped ${summary.skippedMissing} with no raw file` : ''}`,
  );

  if (args.dryRun) {
    note(`--dry-run, ${manifestLabel()} not written`);
    return;
  }
  await saveManifest(manifest);
  note(`wrote ${manifestLabel()}`);
}

main().catch((err) => die(err.stack || String(err)));
