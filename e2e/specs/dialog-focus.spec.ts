// Where the focus lands when a sheet opens (M11.dialog-title-focus): on the
// dialog's title, which shows no focus ring, never on the scrolling body.
// A reload into an open dialog is the case that drew the ring: with no
// pointer input since the load, Chrome's :focus-visible rang the body that
// showModal() focused. Tab then moves to a control, which does show it.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom, playKey } from '../fixtures/scenario';

async function quietSettings(page: Page): Promise<void> {
  await page.addInitScript(() => {
    // A stored name: a reload into a finished game skips the name prompt.
    localStorage.setItem('ajayeb:name', 'Player');
    const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
    localStorage.setItem(
      'ajayeb:settings',
      JSON.stringify({ ...settings, showOpponentMove: false, confirmMoves: 'never' })
    );
  });
}

/** The focused element, and what it looks like. */
function focused(page: Page) {
  return page.evaluate(() => {
    const el = document.activeElement as HTMLElement | null;
    const dialog = document.querySelector('dialog[open]');
    return {
      id: el?.id ?? '',
      tag: el?.tagName.toLowerCase() ?? '',
      inDialog: !!el && !!dialog?.contains(el),
      body: !!el?.matches('[data-sheet-body]'),
      tabIndex: el?.tabIndex ?? 0,
      outline: el ? getComputedStyle(el).outlineStyle : '',
      // A segmented control's radio draws its ring on the label around it.
      ring: [el, el?.closest('label')].some(
        (x) => !!x && getComputedStyle(x).outlineStyle !== 'none'
      ),
      labelledBy: dialog?.getAttribute('aria-labelledby') ?? ''
    };
  });
}

/** The dialog opened on its title, without a ring; Tab moves on to a
 *  control, with one. */
async function expectTitleFocus(page: Page, titleId: string): Promise<void> {
  await expect(page.locator('dialog[open]')).toBeVisible();
  await expect(page.locator(`dialog[open] #${titleId}`)).toBeFocused();
  expect(await focused(page)).toMatchObject({
    id: titleId,
    tag: 'h2',
    body: false,
    tabIndex: -1,
    outline: 'none',
    labelledBy: titleId
  });
  await page.keyboard.press('Tab');
  const next = await focused(page);
  expect(next.inDialog, 'Tab stays in the dialog').toBe(true);
  expect(next.id, 'Tab leaves the title').not.toBe(titleId);
  expect(next.ring, 'the next stop shows the keyboard ring').toBe(true);
}

test.describe('dialog focus on open', { tag: ['@mobile'] }, () => {
  test('a reload into a finished game opens the result on its title', async ({ page }) => {
    const problems = watchHygiene(page);
    await quietSettings(page);
    const driver = await openHotseatRoom(page, 'military-supremacy');
    await playKey(driver, 'construct:walls');
    await expect(page.locator('[data-game-over] dialog[open]')).toBeVisible();
    await page.reload();
    await new GameDriver(page).waitReady();
    await expect(page.locator('[data-game-over] dialog[open]')).toBeVisible();
    await expectTitleFocus(page, 'game-over-title');
    expect(problems).toEqual([]);
  });

  test('a reload into the starter choice opens it on its title', async ({ page }) => {
    const problems = watchHygiene(page);
    await quietSettings(page);
    // Seat 1 trails on the pawn, so this side chooses Age II's starter.
    await page.goto('/dev/new?mode=ai&seat=1&seed=7&scenario=age-transition-chooser');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await new GameDriver(page).waitReady();
    const starter = page.locator('dialog[open] [data-act="starter:0"]');
    await expect(starter).toBeVisible({ timeout: 30_000 });
    await page.reload();
    await new GameDriver(page).waitReady();
    await expect(starter).toBeVisible({ timeout: 30_000 });
    await expectTitleFocus(page, 'age-start-title');
    expect(problems).toEqual([]);
  });

  test('a tapped sheet opens on its title too; the join sheet on its code field', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto('/');
    await page.getByRole('button', { name: 'Settings', exact: true }).click();
    await expectTitleFocus(page, 'home-settings-title');
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    // Typing the code is that sheet's whole job (its field is autofocus).
    await page.locator('[data-control="home-join"]').click();
    await expect(page.locator('dialog[open] [data-code-input]')).toBeFocused();
    expect(problems).toEqual([]);
  });
});
