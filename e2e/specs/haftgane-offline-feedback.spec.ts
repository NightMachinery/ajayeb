import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { lobbyReady } from '../fixtures/lobby';
import { choose, latest, openBotTable, ready } from '../fixtures/table';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`Haftgane offline feedback ${locale}`, () => {
    test.use({ appLocale: locale });
    for (const special of [false, true]) {
      test(`saved ${special ? 'side' : 'hand'} draft explains disabled confirmation @mobile`, async ({
        page,
        context
      }) => {
        await openBotTable(page, 3, 'Player', async () => {
          await page.locator('[data-act="lobby-sides"]').selectOption(special ? 'choose' : 'day');
          await lobbyReady(page);
        });
        const frame = await latest(page);
        const choice = special
          ? frame.view.you!.choices.find((c) => c.key === 'side:day')!
          : frame.view.you!.choices.find((c) => c.kind === 'sell')!;
        if (special) {
          await page.locator('[data-side-picker] [data-pick="side:day"]').click();
          await ready(page);
        } else await choose(page, choice.key);
        const saved = (await latest(page)).pending;
        const confirmation = page.locator(
          special ? '[data-special-confirm]' : '[data-act="confirm"]'
        );
        try {
          await context.setOffline(true);
          await expect
            .poll(() => page.evaluate(() => window.__ajayebTable!.conn()))
            .not.toBe('open');
          await expect(confirmation).toBeDisabled();
          const scope = special
            ? page.locator('[data-side-picker]')
            : page.locator('[data-turn-bar]');
          await expect(scope.locator('[data-pick-connection]')).toBeVisible();
          await expect(scope.locator('[data-pick-connection]')).toContainText(
            locale === 'en'
              ? 'Your choice is kept. You can confirm when the connection returns.'
              : 'انتخاب شما محفوظ است. پس از وصل شدن دوباره می‌توانید آن را تأیید کنید.'
          );
        } finally {
          await context.setOffline(false);
        }
        await ready(page);
        expect((await latest(page)).pending).toEqual(saved);
        await expect(confirmation).toBeEnabled();
        await expect(page.locator('[data-pick-connection]')).toHaveCount(0);
        await confirmation.click();
        await ready(page);
      });
    }
  });
}
