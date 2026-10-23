#!/usr/bin/env node
// tools/art/manifest.mjs
//
// Builds art/manifest.json from the presentation catalog
// (web/src/lib/wire/catalog.gen.ts) and the human-written art/subjects.json.
// It never overwrites art/subjects.json, and it keeps the progress
// (status, attempt, promptAmend, bytes, and the review's approvedAttempt
// and reasons) of every asset already in the manifest. Run it after a catalog change or a subjects.json edit, and
// before generate.mjs picks its next asset.
//
// With --game haftgane (lib.mjs "Games") it builds art/haftgane/manifest.json
// instead, from art/haftgane/subjects.json alone: every id there with a
// subject is an asset of that game. An id in art/haftgane/aliases.json reuses
// an existing painting and must have no subject; its target must be in its
// own game's manifest. Each game's manifest is rebuilt only from that game's
// inputs, so it never touches another game's file.
//
// An entry that is no longer in the catalog or subjects is dropped, unless
// it is approved: dropping approved art needs --drop-approved, so a rebuild
// from the wrong or a broken input stops instead.
//
// Usage:
//   node tools/art/manifest.mjs           write art/manifest.json
//   node tools/art/manifest.mjs --check   exit 1 if that would change the file
//   node tools/art/manifest.mjs --game haftgane
//   node tools/art/manifest.mjs --help    print this usage and exit
//
// Flags:
//   --game NAME       the game whose manifest to build (default duel)
//   --manifest PATH   rebuild or check this manifest file instead (a scratch
//                     copy); AJAYEB_ART_MANIFEST does the same. It must be
//                     the game's own manifest
//   --drop-approved   allow dropping approved entries that left the inputs
//   --check           write nothing; exit 1 if the file would change
//   --help            print this usage and exit
//
// See docs/art/bitmaps.md for the pipeline this is step 1 of.

import {
  readCatalog,
  STATIC_ASSETS,
  KIND_SPEC,
  makeId,
  idKind,
  loadSubjects,
  loadManifest,
  loadGameManifest,
  loadAliases,
  saveManifest,
  manifestLabel,
  displayPath,
  qualifiedId,
  setGame,
  setManifestPath,
  game,
  subjectsPath,
  printUsage,
  die,
} from './lib.mjs';

/** Duel's assets: the catalog's cards, wonders and tokens, then the
 *  static assets. Another game's: every id its subjects describe. */
async function buildAssetList(subjects) {
  if (game.name !== 'duel') {
    return Object.keys(subjects)
      .filter((id) => subjects[id]?.subject)
      .map((id) => {
        if (!/^[a-z]+\/[a-z0-9-]+$/.test(id) || !KIND_SPEC[idKind(id)]) {
          die(`${displayPath(subjectsPath)}: "${id}" is not a "<kind>/<slug>" id of a known kind`);
        }
        return { id, kind: idKind(id) };
      });
  }
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

/** Every alias must lack a subject of its own and point at an asset of its
 *  target game's manifest that is not itself an alias. */
async function checkAliases(subjects) {
  const aliases = await loadAliases();
  const manifests = new Map();
  for (const [id, target] of aliases) {
    if (subjects[id]) die(`${id} has a subject and an alias; it either has its own painting or reuses one`);
    if (target.game === game.name && aliases.has(target.id)) die(`${id} aliases ${target.id}, which is an alias itself`);
    if (!manifests.has(target.game)) manifests.set(target.game, await loadGameManifest(target.game));
    if (!manifests.get(target.game).assets?.[target.id]) {
      die(`${id} aliases ${qualifiedId(target.game, target.id)}, which is not in the ${target.game} manifest`);
    }
  }
}

async function build(dropApproved) {
  // The manifest first, so a manifest of another game stops the build
  // before any input is read.
  const existing = await loadManifest();
  const existingAssets = existing.assets || {};
  const subjects = await loadSubjects();
  const assets = await buildAssetList(subjects);
  await checkAliases(subjects);

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
      // Relative to web/static/art, so another game's starts with its name.
      out: `${game.name === 'duel' ? '' : `${game.name}/`}${spec.outDir}/${spec.sizes[0].nameOverride || slug}`,
      gen: spec.gen,
      crop: spec.crop,
      subject: subj.subject,
      accent: catalogAccent || subj.accent || 'ivory',
      status: prior.status || 'todo',
      attempt: prior.attempt || 0,
      promptAmend: prior.promptAmend || '',
      bytes: prior.bytes || {},
    };
    // Review state apply-review.mjs writes, carried forward only when it
    // exists, so an asset nobody reviewed keeps the short entry above.
    if (prior.approvedAttempt) nextAssets[id].approvedAttempt = prior.approvedAttempt;
    if (prior.reasons) nextAssets[id].reasons = prior.reasons;
  }

  if (missingSubjects.length) {
    die(
      `${displayPath(subjectsPath)} is missing an entry (or an empty "subject") for:\n  ${missingSubjects.join('\n  ')}`,
    );
  }

  // Report ids that used to be in the manifest but no longer correspond to
  // a catalog or static asset (a card renamed, a kind removed). They are
  // dropped rather than carried forward, since a stale entry would point
  // generate.mjs at an asset nothing references any more.
  const stale = Object.keys(existingAssets).filter((id) => !nextAssets[id]);
  const staleApproved = stale.filter((id) => existingAssets[id].status === 'approved');
  if (staleApproved.length && !dropApproved) {
    die(
      `this rebuild would drop ${staleApproved.length} approved asset(s) of ${manifestLabel()} that are no longer in ` +
        `the ${game.name} inputs:\n  ${staleApproved.join('\n  ')}\n` +
        `If that is meant (an item renamed or removed), pass --drop-approved.`,
    );
  }

  const sortedIds = Object.keys(nextAssets).sort();
  const sorted = {};
  for (const id of sortedIds) sorted[id] = nextAssets[id];

  // Duel's manifest predates the "game" field and keeps its bytes.
  const head = game.name === 'duel' ? {} : { game: game.name };
  return { manifest: { ...head, generatedFrom: 'tools/art/manifest.mjs', assets: sorted }, stale };
}

async function main() {
  let check = false;
  let dropApproved = false;
  const argv = process.argv.slice(2);
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--help' || a === '-h') printUsage(import.meta.url);
    else if (a === '--check') check = true;
    else if (a === '--game') setGame(argv[++i]);
    else if (a === '--manifest') setManifestPath(argv[++i]);
    else if (a === '--drop-approved') dropApproved = true;
    else die(`unknown argument "${a}" (see --help)`, 2);
  }
  const { manifest, stale } = await build(dropApproved);

  if (stale.length) {
    process.stderr.write(
      `manifest.mjs: dropping ${stale.length} stale manifest entr${stale.length === 1 ? 'y' : 'ies'} no longer in the ${game.name === 'duel' ? 'catalog or static asset list' : 'subjects'}:\n  ${stale.join('\n  ')}\n`,
    );
  }

  if (check) {
    const current = JSON.stringify((await loadManifest()), null, 2) + '\n';
    const next = JSON.stringify(manifest, null, 2) + '\n';
    if (current !== next) {
      die(`${manifestLabel()} is out of date; run node tools/art/manifest.mjs${game.name === 'duel' ? '' : ` --game ${game.name}`}`);
    }
    process.stdout.write(`manifest.mjs: ${manifestLabel()} is up to date\n`);
    return;
  }

  await saveManifest(manifest);
  process.stdout.write(`manifest.mjs: wrote ${Object.keys(manifest.assets).length} assets to ${manifestLabel()}\n`);
}

main().catch((err) => die(err.stack || String(err)));
