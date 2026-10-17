import path from 'node:path';
import type { PageProbe } from '../../tools/chrome/probe.mjs';

export const shotsDir = path.resolve(process.env.SHOTS_DIR ?? 'ux-out');
export interface Capture {
  screen: string;
  viewport: string;
  theme: string;
  locale: string;
  file: string;
  source: string;
  axeScope: string;
  probe: PageProbe;
  axe: { id: string; impact: string | null | undefined; help: string; targets: string[] }[];
  consoleErrors: string[];
  exceptions: string[];
  failedRequests: string[];
  failures: { severity: number; message: string }[];
}
export interface Budget {
  name: string;
  value: number | null;
  limit: number;
  unit: string;
  details: unknown;
}
export const axeTags = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];
export function findings(c: Omit<Capture, 'failures'>): Capture['failures'] {
  const f: Capture['failures'] = [];
  const add = (severity: number, message: string) => f.push({ severity, message });
  for (const a of c.axe)
    add(
      a.impact === 'critical' ? 0 : 1,
      `axe ${a.impact} ${a.id}: ${a.help} (${a.targets.join(', ')})`
    );
  for (const error of c.exceptions) add(0, `uncaught: ${error}`);
  for (const error of c.consoleErrors) add(1, `console: ${error}`);
  for (const error of c.failedRequests) add(1, `request: ${error}`);
  if (c.probe.hscroll) add(1, `horizontal scroll: ${c.probe.sw} > ${c.probe.viewportWidth}px`);
  for (const o of c.probe.viewportOutside)
    add(2, `outside viewport: ${o.el} (${o.left}..${o.right}px, ${o.top}..${o.bottom}px)`);
  for (const o of c.probe.covered) add(1, `covered action: ${o.el} by ${o.by}`);
  if (c.probe.tap.under24) add(2, `${c.probe.tap.under24} targets under 24px`);
  if (c.probe.tap.under44)
    add(
      3,
      `${c.probe.tap.under44} targets under 44px; smallest ${JSON.stringify(c.probe.tap.smallest)}`
    );
  return f.sort((a, b) => a.severity - b.severity || a.message.localeCompare(b.message));
}
