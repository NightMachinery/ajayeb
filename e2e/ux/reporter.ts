import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';
import type { Reporter, TestCase, TestResult, FullResult } from '@playwright/test/reporter';
import { shotsDir, type Capture, type Budget } from './report';

const escape = (s: string) =>
  s
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
export default class ShotsReporter implements Reporter {
  captures: Capture[] = [];
  budgets: Budget[] = [];
  broken: string[] = [];
  start = Date.now();
  onBegin(): void {
    fs.mkdirSync(shotsDir, { recursive: true });
    for (const name of fs.readdirSync(shotsDir)) {
      if (/^[a-z0-9-]+__\d+x\d+__(light|dark)__(en|fa)\.png$/.test(name))
        fs.unlinkSync(path.join(shotsDir, name));
    }
  }
  onTestEnd(test: TestCase, result: TestResult): void {
    for (const a of result.attachments) {
      if (a.name === 'ux-capture' && a.body)
        this.captures.push(JSON.parse(a.body.toString()) as Capture);
      if (a.name === 'ux-budgets' && a.body)
        this.budgets.push(...(JSON.parse(a.body.toString()) as Budget[]));
    }
    if (result.status !== 'passed' && result.status !== 'skipped')
      this.broken.push(`${test.title}: ${result.error?.message ?? result.status}`);
  }
  onEnd(result: FullResult): void {
    this.captures.sort((a, b) => a.file.localeCompare(b.file));
    const failures = this.captures.flatMap((c) => c.failures.map((f) => ({ ...f, file: c.file })));
    failures.sort(
      (a, b) =>
        a.severity - b.severity ||
        a.file.localeCompare(b.file) ||
        a.message.localeCompare(b.message)
    );
    const misses = this.budgets.filter((b) => b.value === null || b.value > b.limit);
    const ranked = [
      ...failures.map((f) => ({ severity: f.severity, message: `${f.file}: ${f.message}` })),
      ...misses.map((b) => ({
        severity: 1,
        message: `BUDGET ${b.name}: ${b.value ?? 'unavailable'} ${b.unit}, limit ${b.limit}.`
      }))
    ].sort((a, b) => a.severity - b.severity || a.message.localeCompare(b.message));
    const summary =
      [
        `UX capture report: ${this.captures.length} captures, ${result.status}.`,
        `Duration: ${((Date.now() - this.start) / 1000).toFixed(1)} s.`,
        ...this.broken.map((s) => `- UNREACHED: ${s}`),
        ...ranked.map((f) => `- P${f.severity} ${f.message}`),
        ...(!ranked.length && !this.broken.length ? ['No probe failures or budget misses.'] : [])
      ].join('\n') + '\n';
    fs.writeFileSync(
      path.join(shotsDir, 'probes.json'),
      JSON.stringify(
        {
          captures: this.captures,
          budgets: this.budgets,
          durationSeconds: (Date.now() - this.start) / 1000,
          status: result.status,
          unreachable: this.broken
        },
        null,
        2
      ) + '\n'
    );
    fs.writeFileSync(path.join(shotsDir, 'summary.md'), summary);
    const screens = [...new Set(this.captures.map((c) => c.screen))];
    fs.writeFileSync(
      path.join(shotsDir, 'index.html'),
      `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Ajayeb UX captures</title><style>body{font:15px system-ui;background:#eee;color:#222;margin:1rem}nav{display:flex;gap:1rem;flex-wrap:wrap}section{border-top:2px solid #888;margin-top:2rem}.captures{display:flex;flex-wrap:wrap;align-items:start;gap:1rem}figure{margin:0;width:320px;background:white;padding:10px;overflow-wrap:anywhere}img{width:100%;height:auto}li{margin:.3em 0}pre{white-space:pre-wrap}</style><h1>Ajayeb UX captures</h1><p>${this.captures.length} captures. Probe findings are observations, not test failures.</p><nav>${screens.map((s) => `<a href="#${s}">${s}</a>`).join('')}</nav><details><summary>Budgets and run summary</summary><pre>${escape(summary)}\n${escape(JSON.stringify(this.budgets, null, 2))}</pre></details>${screens
        .map(
          (s) =>
            `<section id="${s}"><h2>${s}</h2><div class="captures">${this.captures
              .filter((c) => c.screen === s)
              .map((c) => {
                const hash = createHash('sha256')
                  .update(fs.readFileSync(path.join(shotsDir, c.file)))
                  .digest('hex')
                  .slice(0, 10);
                return `<figure><a href="${c.file}?v=${hash}"><img loading="lazy" src="${c.file}?v=${hash}" alt="${c.screen} ${c.viewport} ${c.theme} ${c.locale}"></a><figcaption><strong>${c.viewport} / ${c.theme} / ${c.locale}</strong><p>${escape(c.source)}, axe: ${escape(c.axeScope)}</p><ul>${c.failures.map((f) => `<li>P${f.severity} ${escape(f.message)}</li>`).join('') || '<li>No probe findings</li>'}</ul></figcaption></figure>`;
              })
              .join('')}</div></section>`
        )
        .join('')}</html>`
    );
  }
}
