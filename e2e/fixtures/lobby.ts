import { expect, type Page } from '@playwright/test';
/** Production Home flow, including submission of the name before any hello. */
export async function createTableFromHome(page: Page, name = 'Host'): Promise<string> {
  await page.goto('/');
  await page.locator('[data-act="home-game:haftgane"]').click();
  await page.locator('[data-act="home-new-table"]').click();
  await page.locator('[data-new-table] [data-name-input]').fill(name);
  await page.locator('[data-act="lobby-create"]').click();
  await page.waitForURL(/\/play\/[A-Z0-9]+$/);
  await expect(page.locator('[data-lobby]')).toBeVisible();
  await lobbyReady(page);
  return new URL(page.url()).pathname.split('/').at(-1)!;
}
export async function lobbyReady(page: Page): Promise<void> {
  await expect.poll(() => page.evaluate(() => window.__ajayebTable?.ready() ?? false)).toBe(true);
}
export async function lobbyAction(page: Page, key: string): Promise<void> {
  await lobbyReady(page);
  await page.locator(`[data-act="${key}"]`).click();
  await lobbyReady(page);
}
