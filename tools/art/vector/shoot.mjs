#!/usr/bin/env node
// Builds a set's specimen (specimen.mjs) and photographs it with one
// headless Chrome: light and dark at DPR 1 and 2, then a greyscale pass and
// the three colour-vision emulations (CDP Emulation.setEmulatedVisionDeficiency)
// at DPR 1. Also writes the set's lint report. Run it under the browser lock:
//
//   python3 tools/heavy.py run --kind browser -- \
//     node tools/art/vector/shoot.mjs --set resources-economy-military [--iter 2]
//
// Flags:
//   --set NAME          a set from art/vector/sets.json (required)
//   --iter N            write into <set>/iter-N/ instead of <set>/, to keep
//                       each iteration's renders side by side
//   --out DIR           the output directory (default <run dir>/art-work/vector/<set>)
//   --cvd-themes LIST   page themes for the greyscale and colour-vision passes
//                       (default light; columns A and B already cover both surfaces)
//   --force             launch Chrome without the heavy.py browser lock
//
// Writes, in the output directory:
//   specimen.html, lint.txt
//   light@1x.png light@2x.png dark@1x.png dark@2x.png
//   grey-<theme>@1x.png deuteranopia-<theme>@1x.png protanopia-<theme>@1x.png
//   tritanopia-<theme>@1x.png
//   shots.json   per PNG: its dpr, theme, vision and the CSS-px boxes of the
//                specimen's data-rect regions (peep.mjs crops with them)

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import {
  ART_WORK,
  ensureDir,
  getSet,
  isMain,
  list,
  printHeader,
  runMain,
  setFiles,
  shown,
  withShooter
} from './lib.mjs';
import { lintFiles } from './lint.mjs';
import { buildSpecimen } from './specimen.mjs';

export const VISIONS = [
  ['grey', 'achromatopsia'],
  ['deuteranopia', 'deuteranopia'],
  ['protanopia', 'protanopia'],
  ['tritanopia', 'tritanopia']
];

async function main() {
  const { values: o } = parseArgs({
    options: {
      set: { type: 'string' },
      iter: { type: 'string' },
      out: { type: 'string' },
      'cvd-themes': { type: 'string', default: 'light' },
      force: { type: 'boolean', default: false },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help) {
    printHeader(import.meta.url);
    return;
  }
  if (!o.set) throw new Error('--set NAME is required');
  const set = getSet(o.set);
  let out = o.out ?? path.join(ART_WORK, set.name);
  if (o.iter) out = path.join(out, `iter-${o.iter}`);
  ensureDir(out);

  const html = buildSpecimen(set);
  fs.writeFileSync(path.join(out, 'specimen.html'), html);

  const files = setFiles(set);
  const { problems } = lintFiles(files.map((f) => ({ file: f.file, id: f.id, set: set.name })));
  const lintText = problems
    .map((p) => `${shown(p.file)}:${p.line}: ${p.level} ${p.rule}: ${p.message}`)
    .join('\n');
  fs.writeFileSync(path.join(out, 'lint.txt'), lintText ? `${lintText}\n` : 'clean\n');
  const errors = problems.filter((p) => p.level === 'error').length;
  console.error(`shoot: lint ${errors} errors, ${problems.length - errors} warnings (lint.txt)`);

  const plan = [];
  for (const theme of ['light', 'dark']) {
    for (const dpr of [1, 2])
      plan.push({ name: `${theme}@${dpr}x.png`, theme, dpr, vision: 'none' });
  }
  for (const theme of list(o['cvd-themes'])) {
    if (theme !== 'light' && theme !== 'dark') throw new Error(`--cvd-themes: bad theme ${theme}`);
    for (const [label, vision] of VISIONS)
      plan.push({ name: `${label}-${theme}@1x.png`, theme, dpr: 1, vision });
  }

  const meta = { set: set.name, at: new Date().toISOString(), shots: {} };
  await withShooter(
    async (shooter) => {
      await shooter.load(html);
      for (const shot of plan) {
        const file = path.join(out, shot.name);
        const m = await shooter.capture({
          out: file,
          dpr: shot.dpr,
          theme: shot.theme,
          vision: shot.vision
        });
        meta.shots[shot.name] = {
          dpr: shot.dpr,
          theme: shot.theme,
          vision: shot.vision,
          width: m.width,
          height: m.height,
          rects: m.rects
        };
        console.error(`shoot: ${shown(file)} (${m.width * shot.dpr}x${m.height * shot.dpr})`);
      }
    },
    { force: o.force }
  );
  fs.writeFileSync(path.join(out, 'shots.json'), `${JSON.stringify(meta, null, 2)}\n`);
  console.error(`shoot: ${plan.length} PNGs in ${shown(out)}`);
}

if (isMain(import.meta.url)) runMain(main);
