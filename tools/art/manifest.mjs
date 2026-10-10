#!/usr/bin/env node
// tools/art/manifest.mjs
//
// Builds art/manifest.json from the presentation catalog
// (web/src/lib/wire/catalog.gen.ts) and the human-written art/subjects.json.
// It never overwrites art/subjects.json, and it keeps the progress
// (status, attempt, promptAmend, bytes) of every asset already in the
// manifest. Run it after a catalog change or a subjects.json edit, and
// before generate.mjs picks its next asset.
//
// Usage:
//   node tools/art/manifest.mjs           write art/manifest.json
//   node tools/art/manifest.mjs --check   exit 1 if that would change the file
//
// See docs/art/bitmaps.md for the pipeline this is step 1 of.

import {
  readCatalog,
  STATIC_ASSETS,
  KIND_SPEC,
  makeId,
  loadSubjects,
  loadManifest,
  saveManifest,
  die,
} from './lib.mjs';

async function buildAssetList() {
  const catalog = await readCatalog();
  const list = [];
  for (const { slug, data } of catalog.cards) {
    list.push({ id: makeId('card', slug), kind: 'card', catalogAccent: data.color });
  }
  for (const { slug } of catalog.wonders) {
    list.push({ id: makeId('wonder', slug), kind: 'wonder' });
  }
  for (const { slug } of catalog.tokens) {
    list.push({ id: makeId('token', slug), kind: 'token' });
  }
  for (const a of STATIC_ASSETS) {
    list.push({ id: a.id, kind: a.kind });
  }
  return list;
}

async function build() {
  const assets = await buildAssetList();
  const subjects = await loadSubjects();
  const existing = await loadManifest();
  const existingAssets = existing.assets || {};

  const missingSubjects = [];
  const nextAssets = {};

  for (const { id, kind, catalogAccent } of assets) {
    const subj = subjects[id];
    if (!subj || !subj.subject) {
      missingSubjects.push(id);
      continue;
    }
    const spec = KIND_SPEC[kind];
    if (!spec) throw new Error(`manifest.mjs: no KIND_SPEC for kind "${kind}" (${id})`);
    const slug = id.slice(id.indexOf('/') + 1);
    const prior = existingAssets[id] || {};
    nextAssets[id] = {
      id,
      kind,
      out: `${spec.outDir}/${spec.sizes[0].nameOverride || slug}`,
      gen: spec.gen,
      crop: spec.crop,
      subject: subj.subject,
      accent: catalogAccent || subj.accent || 'ivory',
      status: prior.status || 'todo',
      attempt: prior.attempt || 0,
      promptAmend: prior.promptAmend || '',
      bytes: prior.bytes || {},
    };
  }

  if (missingSubjects.length) {
    die(
      `art/subjects.json is missing an entry (or an empty "subject") for:\n  ${missingSubjects.join('\n  ')}`,
    );
  }

  // Report ids that used to be in the manifest but no longer correspond to
  // a catalog or static asset (a card renamed, a kind removed). They are
  // dropped rather than carried forward, since a stale entry would point
  // generate.mjs at an asset nothing references any more.
  const stale = Object.keys(existingAssets).filter((id) => !nextAssets[id]);

  const sortedIds = Object.keys(nextAssets).sort();
  const sorted = {};
  for (const id of sortedIds) sorted[id] = nextAssets[id];

  return { manifest: { generatedFrom: 'tools/art/manifest.mjs', assets: sorted }, stale };
}

async function main() {
  const check = process.argv.includes('--check');
  const { manifest, stale } = await build();

  if (stale.length) {
    process.stderr.write(
      `manifest.mjs: dropping ${stale.length} stale manifest entr${stale.length === 1 ? 'y' : 'ies'} no longer in the catalog or static asset list:\n  ${stale.join('\n  ')}\n`,
    );
  }

  if (check) {
    const current = JSON.stringify((await loadManifest()), null, 2) + '\n';
    const next = JSON.stringify(manifest, null, 2) + '\n';
    if (current !== next) {
      die('art/manifest.json is out of date; run node tools/art/manifest.mjs');
    }
    process.stdout.write('manifest.mjs: art/manifest.json is up to date\n');
    return;
  }

  await saveManifest(manifest);
  process.stdout.write(`manifest.mjs: wrote ${Object.keys(manifest.assets).length} assets to art/manifest.json\n`);
}

main().catch((err) => die(err.stack || String(err)));
