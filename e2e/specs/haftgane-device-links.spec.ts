import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { newDeviceContext } from '../fixtures/online';
import { choose, latest, openBotTable, ready } from '../fixtures/table';

// Copy from the visible strip, enter with a new browser identity, share the
// private pending pick, then reset and verify the old device is detached.
test('Haftgane device links open a second device and reset it @mobile', async ({
  page,
  context,
  browser
}, testInfo) => {
  const origin = new URL(testInfo.project.use.baseURL!).origin;
  await context.grantPermissions(['clipboard-read', 'clipboard-write'], { origin });
  await openBotTable(page, 3, 'Player');
  const frame = await latest(page);
  const seat = frame.you.seat!;
  await page
    .locator(`[data-control="device-link"][data-device-link-user="${frame.you.userId}"]`)
    .click();
  const sheet = page
    .locator('dialog[open]')
    .filter({ has: page.locator('[data-device-link-sheet]') });
  await expect(sheet.locator('[data-device-link-status]')).toHaveAttribute(
    'data-device-link-status',
    'copied'
  );
  const link = await sheet.locator('[data-device-link]').inputValue();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(link);
  expect(new URL(link).hash).toMatch(/^#claim=[A-Za-z0-9_-]{22}$/);
  await testInfo.attach('device-link-sheet', {
    body: await page.screenshot({ mask: [sheet.locator('[data-device-link]')] }),
    contentType: 'image/png'
  });
  await page.keyboard.press('Escape');
  const device = await newDeviceContext(browser, testInfo);
  try {
    const phone = await device.newPage();
    await phone.goto(link);
    await ready(phone);
    const other = await latest(phone);
    expect(other.you.userId).toBe(frame.you.userId);
    expect(other.you.seat).toBe(seat);
    expect(other.view.you?.hand).toEqual(frame.view.you?.hand);
    expect(await phone.locator('[data-name-prompt]').count()).toBe(0);
    await choose(phone, other.view.you!.choices.find((c) => c.kind === 'sell')!.key);
    await expect
      .poll(async () => (await latest(page)).pending?.key)
      .toBe((await latest(phone)).pending?.key);
    await phone.reload();
    await ready(phone);
    expect((await latest(phone)).you.userId).toBe(frame.you.userId);
    await page
      .locator(`[data-control="device-link"][data-device-link-user="${frame.you.userId}"]`)
      .click();
    await sheet.locator('[data-control="device-link-reset"]').click();
    await expect(sheet.locator('[data-control="device-link-reset"]')).toHaveAttribute(
      'data-armed',
      ''
    );
    await page.waitForTimeout(400);
    await sheet.locator('[data-control="device-link-reset"]').click();
    await expect(sheet.locator('[data-device-link-status]')).toHaveAttribute(
      'data-device-link-status',
      'reset'
    );
    await expect(phone.locator('[data-room-error="bad-claim"]')).toBeVisible();
  } finally {
    await device.close();
  }
});
