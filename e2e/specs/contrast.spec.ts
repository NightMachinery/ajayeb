// Real text contrast regressions from the M11 screen review. The UX matrix
// records axe findings for review; this smaller suite makes these failures
// block verification, including the tutorial's compact progress label.
import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { reachScreen, type Screen } from '../ux/screens';
import { settleCapture } from '../ux/settle';

const screens: readonly Screen[] = [
  'join',
  'card-unaffordable',
  'board-age3',
  'offline',
  'discard-pile',
  'tutorial-card-sheet',
  'choice-starter',
  'choice-progress-token',
  'choice-destroy'
];

for (const theme of ['light', 'dark'] as const) {
  test.describe(`text contrast in ${theme}`, () => {
    for (const screen of screens) {
      test.describe(screen, () => {
        // The danger chip and offline caption live in the desktop city panel
        // and top bar. The compact tutorial and age summary fail on phones.
        const desktop = screen === 'board-age3' || screen === 'offline';
        test.use({
          viewport: desktop ? { width: 1280, height: 800 } : { width: 360, height: 740 },
          isMobile: !desktop,
          hasTouch: !desktop
        });
        test('keeps text readable on its painted surface', async ({ page, baseURL }) => {
          if (!baseURL) throw new Error('contrast requires local baseURL');
          await page.addInitScript((theme) => {
            localStorage.setItem('ajayeb:theme', theme);
            localStorage.setItem('ajayeb:name', 'Player');
          }, theme);
          if (screen === 'choice-destroy') {
            // Its sole card is selected automatically. Disconnect through the
            // fixture transport to exercise the disabled danger commit style.
            await page.goto('/dev/fixture/destroy-card-seat0?link=offline');
            await expect(
              page.locator('[data-choice-host="destroyCard"] dialog[open]')
            ).toBeVisible();
          } else {
            await reachScreen(page, baseURL, screen);
          }
          await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
          await settleCapture(page, screen);
          const builder = new AxeBuilder({ page }).withRules(['color-contrast']);
          // The gallery outside this actual modal contains unrelated specimens.
          if (screen === 'discard-pile') builder.include('dialog[open]');
          const violations = (await builder.analyze()).violations.map((v) => ({
            id: v.id,
            targets: v.nodes.map((node) => node.target)
          }));
          expect(violations).toEqual([]);
          if (
            [
              'join',
              'card-unaffordable',
              'tutorial-card-sheet',
              'choice-progress-token',
              'choice-destroy'
            ].includes(screen)
          ) {
            // Axe exempts disabled controls. Their labels can be instructions,
            // so check the painted label/background pair too, including the
            // old whole-button opacity against the panel underneath it.
            const ratios = await page
              .locator(
                screen === 'choice-destroy'
                  ? '[data-commit] button:disabled'
                  : 'dialog[open] button:disabled'
              )
              .filter({ visible: true })
              .evaluateAll((buttons) => {
                const ctx = document.createElement('canvas').getContext('2d')!;
                const rgba = (css: string): number[] => {
                  ctx.clearRect(0, 0, 1, 1);
                  ctx.fillStyle = css;
                  ctx.fillRect(0, 0, 1, 1);
                  return Array.from(ctx.getImageData(0, 0, 1, 1).data, (n) => n / 255);
                };
                const over = (fg: number[], bg: number[], opacity = 1): number[] => {
                  const a = fg[3]! * opacity;
                  return fg
                    .slice(0, 3)
                    .map((n, i) => n * a + bg[i]! * (1 - a))
                    .concat(1);
                };
                const surface = (el: Element | null): number[] => {
                  if (!el) return [1, 1, 1, 1];
                  const bg = rgba(getComputedStyle(el).backgroundColor);
                  return bg[3] === 1 ? bg : over(bg, surface(el.parentElement));
                };
                const luminance = (rgb: number[]): number => {
                  const linear = rgb
                    .slice(0, 3)
                    .map((n) => (n <= 0.04045 ? n / 12.92 : ((n + 0.055) / 1.055) ** 2.4));
                  return linear[0]! * 0.2126 + linear[1]! * 0.7152 + linear[2]! * 0.0722;
                };
                return buttons.map((button) => {
                  const style = getComputedStyle(button);
                  const panel = surface(button.parentElement);
                  const bg = over(rgba(style.backgroundColor), panel);
                  const opacity = Number(style.opacity);
                  const fg = over(rgba(style.color), bg);
                  const values = [
                    luminance(over(fg, panel, opacity)),
                    luminance(over(bg, panel, opacity))
                  ];
                  return (Math.max(...values) + 0.05) / (Math.min(...values) + 0.05);
                });
              });
            expect(ratios.length, 'the instructional disabled button is visible').toBeGreaterThan(
              0
            );
            for (const ratio of ratios) expect(ratio).toBeGreaterThanOrEqual(4.5);
          }
        });
      });
    }
  });
}
