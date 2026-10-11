// Smoke test: the server is up and the home page is healthy, in both system
// colour schemes, on the desktop and mobile projects. `make e2e-smoke` runs
// it (and `verify` does, through that target).

import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';

// WCAG 2.2 AA, the level the UX plan targets.
const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];

test.describe('smoke', { tag: ['@smoke', '@mobile'] }, () => {
  test('GET /health reports ok', async ({ request }) => {
    const res = await request.get('/health');
    expect(res.ok()).toBe(true);
    expect(await res.json()).toMatchObject({ ok: true });
  });

  for (const colorScheme of ['light', 'dark'] as const) {
    test.describe(`home with a ${colorScheme} system theme`, () => {
      test.use({ colorScheme });

      test('renders, fits the viewport, logs no errors and passes axe', async ({ page }) => {
        const problems = watchHygiene(page);

        await page.goto('/');
        await expect(page).toHaveTitle(/Ajayeb/);
        await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
        // The theme bootstrap in app.html follows the system scheme by default.
        await expect(page.locator('html')).toHaveAttribute('data-theme', colorScheme);
        await page.evaluate(async () => {
          await document.fonts.ready;
        });

        const { sw, iw } = await page.evaluate(() => ({
          sw: document.documentElement.scrollWidth,
          iw: window.innerWidth
        }));
        expect(sw, `horizontal scroll: scrollWidth ${sw} > innerWidth ${iw}`).toBeLessThanOrEqual(
          iw
        );

        const axe = await new AxeBuilder({ page }).withTags(AXE_TAGS).analyze();
        const blocking = axe.violations.filter(
          (v) => v.impact === 'serious' || v.impact === 'critical'
        );
        const report = blocking
          .map(
            (v) =>
              `${v.impact} ${v.id}: ${v.help} at ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`
          )
          .join('\n');
        expect(blocking, `serious or critical axe violations:\n${report}`).toEqual([]);

        expect(problems, 'console errors, page errors or failed requests').toEqual([]);
      });
    });
  }
});
