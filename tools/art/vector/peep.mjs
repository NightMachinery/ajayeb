#!/usr/bin/env node
// The pixel peep: crops the 12, 14 and 16 px strip out of a set's specimen
// shots and enlarges it with point filtering (ImageMagick), so every device
// pixel of the small sizes shows as a block. No browser; run shoot.mjs first.
//
//   node tools/art/vector/peep.mjs --set resources-economy-military [--iter 2]
//
// Flags:
//   --set NAME     the set whose shots to crop (required unless --dir)
//   --iter N       read and write <set>/iter-N/
//   --dir DIR      the shot directory (default <run dir>/art-work/vector/<set>)
//   --shots LIST   which shots to peep (default light@1x.png,light@2x.png,
//                  grey-light@1x.png); each needs its entry in shots.json
//   --scale PCT    enlargement for DPR 1 shots (default 800; DPR 2 gets half)
//
// Writes peep-<shot>.png next to the shots, for example peep-light@1x.png.
// The strip has paper rows (12, 14, 16 px) above night rows, one column per
// icon in sets.json order, -sm variants at 14 px and below.

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { ART_WORK, isMain, list, printHeader, shown } from './lib.mjs';

/** Crops one rect (CSS px) out of a PNG at `dpr` and enlarges it with point filtering. */
export function peep({ png, rect, dpr, scale, out }) {
  const r = (v) => Math.round(v * dpr);
  const geometry = `${r(rect.width)}x${r(rect.height)}+${r(rect.x)}+${r(rect.y)}`;
  const res = spawnSync(
    'magick',
    [png, '-crop', geometry, '+repage', '-filter', 'point', '-resize', `${scale}%`, out],
    { encoding: 'utf8' }
  );
  if (res.error)
    throw new Error(`peep: cannot run magick (${res.error.message}); install ImageMagick 7`);
  if (res.status !== 0) throw new Error(`peep: magick failed: ${res.stderr.trim()}`);
}

async function main() {
  const { values: o } = parseArgs({
    options: {
      set: { type: 'string' },
      iter: { type: 'string' },
      dir: { type: 'string' },
      shots: { type: 'string', default: 'light@1x.png,light@2x.png,grey-light@1x.png' },
      scale: { type: 'string', default: '800' },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help) {
    printHeader(import.meta.url);
    return;
  }
  if (!o.dir && !o.set) throw new Error('--set NAME or --dir DIR is required');
  let dir = o.dir ?? path.join(ART_WORK, o.set);
  if (o.iter) dir = path.join(dir, `iter-${o.iter}`);
  const metaFile = path.join(dir, 'shots.json');
  if (!fs.existsSync(metaFile)) throw new Error(`no ${shown(metaFile)}: run shoot.mjs first`);
  const meta = JSON.parse(fs.readFileSync(metaFile, 'utf8'));
  const scale = Number(o.scale);
  if (!(scale > 0)) throw new Error(`--scale: bad value ${o.scale}`);
  for (const name of list(o.shots)) {
    const shot = meta.shots[name];
    if (!shot) throw new Error(`shots.json has no ${name}`);
    const rect = shot.rects.peep;
    if (!rect) throw new Error(`${name} has no peep region`);
    const out = path.join(dir, `peep-${name}`);
    peep({ png: path.join(dir, name), rect, dpr: shot.dpr, scale: scale / shot.dpr, out });
    console.error(`peep: ${shown(out)}`);
  }
}

if (isMain(import.meta.url)) {
  main().catch((err) => {
    console.error(`${err.stack ?? err}`);
    process.exit(1);
  });
}
