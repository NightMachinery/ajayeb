// Display regressions for M11.wording-consistency. No client rules or state mutation.
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { newDeviceContext } from '../fixtures/online';
import { createDevRoom, newToken } from '../fixtures/ws-game';

const plain = (s: string) => s.replace(/[\u200e\u200f\u2066-\u2069]/g, '');

for (const locale of ['en', 'fa'] as const) {
  test.describe(`consistent wording ${locale}`, { tag: '@mobile' }, () => {
    test.use({ appLocale: locale });
    test('capital winners keep both real names with hints shown or hidden, in either orientation', async ({
      page,
      isMobile
    }) => {
      await page.addInitScript(() => {
        const hidden = new URL(location.href).searchParams.get('militaryHidden') === '1';
        localStorage.setItem(
          'ajayeb:hintsHidden',
          JSON.stringify({ families: hidden ? ['military'] : [], boardMarks: false })
        );
      });
      for (const hidden of [false, true]) {
        for (const viewer of ['seat0', 'seat1', 'spectator'] as const) {
          await page.goto(
            `/dev/fixture/age-1-start-${viewer}?militaryHidden=${hidden ? '1' : '0'}`
          );
          expect(
            await page.evaluate(
              () => JSON.parse(localStorage.getItem('ajayeb:hintsHidden')!).families
            )
          ).toEqual(hidden ? ['military'] : []);
          await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
          await expect(page.locator('html')).toHaveAttribute('lang', locale);
          const track = page.locator('[data-tut="track"]').filter({ visible: true }).first();
          await expect(track).toHaveAttribute('data-left-seat', viewer === 'seat1' ? '1' : '0');
          for (const seat of [0, 1] as const) {
            const target = track.locator(`[data-capital-tip="${seat}"]`);
            if (isMobile) await target.tap();
            else await target.hover();
            const bubble = page.locator('#ajayeb-tooltip');
            await expect(bubble).toBeVisible();
            const text = plain(await bubble.innerText());
            const own = viewer === `seat${seat}`;
            const win = viewer !== 'spectator' && !own;
            const name = seat === 0 ? 'Sara' : 'Omid';
            const winner = seat === 0 ? 'Omid' : 'Sara';
            if (locale === 'en') {
              expect(text).toContain(own ? 'Your capital.' : `${name}’s capital.`);
              expect(text).toContain(win ? 'you win at once.' : `${winner} wins at once.`);
            } else {
              expect(text).toContain(own ? 'پایتخت شما.' : `پایتخت ${name}.`);
              expect(text).toContain(
                win ? 'شما بی‌درنگ برنده می‌شوید.' : `${winner} بی‌درنگ برنده می‌شود.`
              );
            }
            await page.keyboard.press('Escape');
            await page.mouse.move(0, 0);
            await expect(bubble).not.toBeVisible();
          }
        }
      }
    });

    test('real room capitals retain names when military hint details are toggled', async ({
      browser,
      baseURL,
      isMobile
    }, info) => {
      if (!baseURL) throw new Error('local server required');
      const room = await createDevRoom(baseURL, {
        mode: 'online',
        seed: '7',
        scenario: 'looting-double'
      });
      const contexts = [];
      const pages = [];
      try {
        for (const name of ['Sara', 'Omid', 'Observer']) {
          const context = await newDeviceContext(browser, info);
          contexts.push(context);
          await context.addInitScript(
            ({ name, token, locale }) => {
              localStorage.setItem('ajayeb:name', name);
              localStorage.setItem('ajayeb:token', token);
              const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
              localStorage.setItem('ajayeb:settings', JSON.stringify({ ...settings, locale }));
            },
            { name, token: newToken(), locale }
          );
          const page = await context.newPage();
          pages.push(page);
          await page.goto(new URL(room.path, baseURL).href);
          await page.waitForFunction(() => window.__ajayeb?.ready());
          await expect(page.locator('html')).toHaveAttribute('lang', locale);
        }
        for (const [viewer, page] of pages.entries()) {
          for (const hidden of [false, true]) {
            await page.evaluate((hidden) => {
              localStorage.setItem(
                'ajayeb:hintsHidden',
                JSON.stringify({ families: hidden ? ['military'] : [], boardMarks: false })
              );
            }, hidden);
            await page.reload();
            await page.waitForFunction(() => window.__ajayeb?.ready());
            await expect(page.locator('html')).toHaveAttribute('lang', locale);
            expect(
              await page.evaluate(
                () => JSON.parse(localStorage.getItem('ajayeb:hintsHidden')!).families
              )
            ).toEqual(hidden ? ['military'] : []);
            expect(
              await page.evaluate(() => window.__ajayeb?.view()?.players.every((p) => !!p.hints))
            ).toBe(true);
            const track = page.locator('[data-tut="track"]').filter({ visible: true }).first();
            await expect(track).toHaveAttribute('data-left-seat', viewer === 1 ? '1' : '0');
            for (const seat of [0, 1]) {
              const target = track.locator(`[data-capital-tip="${seat}"]`);
              if (isMobile) await target.tap();
              else await target.hover();
              const bubble = page.locator('#ajayeb-tooltip');
              await expect(bubble).toBeVisible();
              const text = plain(await bubble.innerText());
              const own = viewer === seat,
                win = viewer < 2 && !own;
              const name = seat === 0 ? 'Sara' : 'Omid',
                winner = seat === 0 ? 'Omid' : 'Sara';
              expect(text).toContain(
                locale === 'en'
                  ? own
                    ? 'Your capital.'
                    : `${name}’s capital.`
                  : own
                    ? 'پایتخت شما.'
                    : `پایتخت ${name}.`
              );
              expect(text).toContain(
                locale === 'en'
                  ? win
                    ? 'you win at once.'
                    : `${winner} wins at once.`
                  : win
                    ? 'شما بی‌درنگ برنده می‌شوید.'
                    : `${winner} بی‌درنگ برنده می‌شود.`
              );
              expect(text.includes(locale === 'en' ? 'shield' : 'سپر')).toBe(!hidden);
              await page.keyboard.press('Escape');
              await page.mouse.move(0, 0);
              await expect(bubble).not.toBeVisible();
            }
          }
        }
      } finally {
        for (const context of contexts) await context.close();
      }
    });

    test('the five long landscape prompts stay within two lines and keep full accessible instructions', async ({
      page
    }) => {
      await page.setViewportSize({ width: 740, height: 360 });
      const scenes = [
        ['pick-library-token-seat0', 'pendingChoice'],
        ['destroy-card-seat0', 'pendingChoice'],
        ['build-from-discard-seat0', 'pendingChoice'],
        ['choose-starter-seat0', 'pendingChoice'],
        ['age-1-start-seat0?link=offline', 'offline']
      ] as const;
      for (const [fixture, kind] of scenes) {
        await page.goto(`/dev/fixture/${fixture}`);
        await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
        await expect(page.locator('html')).toHaveAttribute('lang', locale);
        const bar = page.locator(`[data-turn="${kind}"]`);
        const compact = bar.locator('[data-turn-short]');
        await expect(compact).toBeVisible();
        const metrics = await compact.evaluate((el) => {
          const style = getComputedStyle(el),
            box = el.getBoundingClientRect();
          return {
            lines: box.height / parseFloat(style.lineHeight),
            width: box.width,
            scrollWidth: el.scrollWidth
          };
        });
        expect(metrics.lines).toBeLessThanOrEqual(2.05);
        expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.width + 1);
        // Full text remains in the status live region; the compact copy is visual only.
        await expect(compact).toHaveAttribute('aria-hidden', 'true');
        await expect(bar.locator('[class~=\"short:sr-only\"]')).not.toBeEmpty();
      }
    });
  });
}
