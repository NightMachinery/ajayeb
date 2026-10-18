import { expect, type Page } from '@playwright/test';
import { GameDriver } from '../fixtures/driver';
import { currentView, playKey } from '../fixtures/scenario';
import { createDevRoom } from '../fixtures/ws-game';
import { openTutorialRoom, tutorialScript } from '../fixtures/tutorial';

export const screenIds = [
  'home',
  'new-game',
  'settings',
  'online-lobby',
  'join',
  'rules-game',
  'rules-page',
  'wonder-draft',
  'board-age1',
  'board-age2',
  'board-age3',
  'card-buildable',
  'card-unaffordable',
  'wonder-flow',
  'discard-pile',
  'choice-progress-token',
  'choice-great-library',
  'choice-revive',
  'choice-destroy',
  'choice-starter',
  'end-age-summary',
  'opponent-summary',
  'review-advice',
  'review-none',
  'log',
  'game-menu',
  'ending-military',
  'ending-science',
  'ending-civilian',
  'ending-tie-blue',
  'ending-draw',
  'tutorial-first',
  'tutorial-card-sheet',
  'tutorial-outro',
  'offline',
  'reconnecting',
  'spectator',
  'pass-device',
  'tooltip-board',
  'tooltip-card',
  'advisor-strength'
] as const;
export type Screen = (typeof screenIds)[number];
const visible = (page: Page, selector: string) =>
  page.locator(selector).filter({ visible: true }).first();
async function fixture(page: Page, name: string, query = ''): Promise<void> {
  await page.goto(`/dev/fixture/${name}${query}`);
  await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
  expect(await page.evaluate(() => !!window.__ajayeb?.view())).toBe(true);
}
async function phase(page: Page, expected: string, age?: number): Promise<void> {
  const view = await currentView(page);
  expect(view.phase).toBe(expected);
  if (age !== undefined) expect(view.age).toBe(age);
}
async function menu(page: Page, control: string): Promise<void> {
  await visible(page, '[data-control="menu"]').click();
  await visible(page, `[data-control="${control}"]`).click();
}
async function live(
  page: Page,
  origin: string,
  scenario: string,
  actions: string[] = []
): Promise<GameDriver> {
  const room = await createDevRoom(origin, { mode: 'hotseat', seed: '7', scenario, actions });
  await page.goto(room.path);
  const driver = new GameDriver(page);
  await driver.waitReady();
  return driver;
}
async function pending(page: Page, fixtureName: string, kind: string): Promise<void> {
  await fixture(page, fixtureName);
  expect((await currentView(page)).pending?.kind).toBe(kind);
  await expect(
    page.locator(`[data-choice-host="${kind}"][data-mine="true"] dialog[open]`)
  ).toBeVisible();
}
/** A screen is reached only after its state and its distinguishing UI agree. */
export async function reachScreen(page: Page, origin: string, screen: Screen): Promise<void> {
  switch (screen) {
    case 'home':
    case 'new-game':
    case 'settings':
    case 'join': {
      await page.goto('/');
      await expect(page.locator('h1')).toBeVisible();
      if (screen === 'home') return;
      if (screen === 'new-game') {
        // Stable structure, independent of translated labels.
        await page.locator('nav ul button').first().click();
        await expect(visible(page, '[data-control="ai-tutorial"]')).toBeVisible();
        await expect(page.locator('dialog[open] input[name="ai-level"]')).toHaveCount(3);
      } else if (screen === 'join') {
        await page.locator('[data-control="home-join"]').click();
        await expect(visible(page, '[data-join-sheet]')).toBeVisible();
        await visible(page, '[data-code-input]').fill('FIXTUR');
      } else {
        await page.locator('main > div > div:first-child > button').click();
        await expect(visible(page, '[data-setting="language"]')).toBeVisible();
      }
      return;
    }
    case 'online-lobby':
      await fixture(page, 'online-lobby-seat0');
      await expect(page.locator('[data-lobby="FIXTUR"] [data-qr]')).toBeVisible();
      await expect(page.locator('[data-room-code="FIXTUR"]')).toBeVisible();
      return;
    case 'rules-page':
      await page.goto('/rules');
      await expect(page.locator('[data-rules-page] #goal')).toBeVisible();
      return;
    case 'rules-game':
      await fixture(page, 'age-1-start-seat0');
      await menu(page, 'menu-rules');
      await expect(visible(page, '[data-rules-overlay]')).toBeVisible();
      return;
    case 'wonder-draft':
      await fixture(page, 'draft-round-1-seat0');
      await phase(page, 'wonderDraft', 0);
      await expect(page.locator('[data-draft][data-round="1"]')).toBeVisible();
      return;
    case 'board-age1':
    case 'board-age2':
    case 'board-age3':
      await fixture(
        page,
        {
          'board-age1': 'age-1-start-seat0',
          'board-age2': 'costs-seat0',
          'board-age3': 'looting-seat0'
        }[screen]
      );
      await phase(page, 'play', Number(screen.at(-1)));
      await expect(page.locator('.card-structure')).toBeVisible();
      return;
    case 'card-buildable':
    case 'card-unaffordable':
    case 'wonder-flow':
    case 'tooltip-card': {
      await fixture(page, 'costs-seat0');
      const view = await currentView(page);
      const card =
        screen === 'card-unaffordable'
          ? view.disabled.find((a) => a.kind === 'construct' && a.reason === 'unaffordable')?.card
          : view.choices.find((c) => c.kind === (screen === 'wonder-flow' ? 'wonder' : 'construct'))
              ?.card;
      if (!card) throw new Error(`${screen}: no matching card in costs fixture`);
      await new GameDriver(page).openCard(card);
      if (screen === 'card-unaffordable')
        await expect(visible(page, '[data-reason]')).toBeVisible();
      if (screen === 'card-buildable')
        await expect(visible(page, `dialog[open] [data-act="construct:${card}"]`)).toBeEnabled();
      if (screen === 'wonder-flow') {
        await visible(page, '[data-control="wonder-step"]').click();
        await expect(visible(page, '[data-wonder-step]')).toBeVisible();
      }
      if (screen === 'tooltip-card')
        await openTooltip(page, 'dialog[open] .card-face .band[data-tooltip]');
      return;
    }
    case 'discard-pile':
      await page.goto('/dev/gallery');
      await page.getByRole('button', { name: 'DiscardSheet', exact: true }).click();
      await expect(page.locator('dialog[open]:has([data-discard-sheet])')).toBeVisible();
      return;
    case 'choice-progress-token':
      return pending(page, 'pick-board-token-seat0', 'pickBoardToken');
    case 'choice-great-library':
      return pending(page, 'pick-library-token-seat0', 'pickLibraryToken');
    case 'choice-revive':
      return pending(page, 'build-from-discard-seat0', 'buildFromDiscard');
    case 'choice-destroy':
      return pending(page, 'destroy-card-seat0', 'destroyCard');
    case 'choice-starter':
      return pending(page, 'choose-starter-seat0', 'chooseStarter');
    case 'end-age-summary':
      await fixture(page, 'online-age-start-seat0');
      // A received starter choice now opens the retained summary itself.
      await expect(visible(page, 'dialog[open] [data-age-interstitial]')).toBeVisible();
      await expect(visible(page, '#age-start-title')).toBeVisible();
      return;
    case 'opponent-summary':
    case 'log':
    case 'game-menu':
      await fixture(page, 'state-join-history');
      if (screen === 'opponent-summary') {
        await visible(page, '[data-control="move-summary"]').click();
        await expect(visible(page, '[data-move-summary]')).toBeVisible();
        await expect(visible(page, '[data-summary-line]')).toBeVisible();
      } else if (screen === 'log') {
        await menu(page, 'menu-log');
        await expect(visible(page, 'dialog[open] [data-self-line]')).toBeVisible();
      } else {
        await visible(page, '[data-control="menu"]').click();
        await expect(page.locator('[data-game-menu]')).toBeVisible();
      }
      return;
    case 'review-advice':
    case 'review-none': {
      const room = await createDevRoom(origin, {
        mode: 'ai',
        bot: 'easy',
        seed: '7',
        scenario: screen === 'review-advice' ? 'military-supremacy' : 'looting-double',
        actions: [screen === 'review-advice' ? 'discard:walls' : 'construct:arsenal']
      });
      await page.goto(room.path);
      const driver = new GameDriver(page);
      await driver.nextTurn();
      await visible(page, '[data-control="move-summary"]').click();
      await expect(visible(page, '[data-move-summary]')).toBeVisible();
      await expect(
        visible(page, `[data-move-review="${screen === 'review-advice' ? 'better' : 'none'}"]`)
      ).toBeVisible();
      return;
    }
    case 'ending-military':
    case 'ending-science':
    case 'ending-civilian':
    case 'ending-draw':
    case 'ending-tie-blue': {
      if (screen === 'ending-tie-blue') {
        await live(page, origin, 'tie-blue', ['discard:academy']);
        await expect(visible(page, '[data-tie-break]')).toBeVisible();
      } else {
        await fixture(
          page,
          `game-over-${{ 'ending-military': 'military-hidden', 'ending-science': 'science', 'ending-civilian': 'civilian', 'ending-draw': 'shared' }[screen]}-seat0`
        );
      }
      await phase(page, 'gameOver');
      const victory = {
        'ending-military': 'military',
        'ending-science': 'science',
        'ending-civilian': 'civilian',
        'ending-draw': 'shared',
        'ending-tie-blue': 'civilian'
      }[screen];
      await expect(page.locator('[data-game-over]')).toHaveAttribute('data-victory', victory);
      await expect(visible(page, '[data-score-table]')).toBeVisible();
      return;
    }
    case 'tutorial-first':
    case 'tutorial-card-sheet':
    case 'tutorial-outro': {
      const script = await tutorialScript(origin);
      let actions: string[] = [];
      if (screen === 'tutorial-card-sheet') {
        const i = script.raw.entries.findIndex((e) => e.action?.id === 'buildFree');
        if (i < 0) throw new Error('tutorial has no buildFree');
        actions = script.keys.slice(0, i);
      } else if (screen === 'tutorial-outro') actions = script.keys;
      await openTutorialRoom(page, origin, actions);
      if (screen === 'tutorial-card-sheet') {
        for (let guard = 0; guard < 8; guard++) {
          if (await visible(page, '[data-coach-note]').count()) break;
          const next = visible(page, '[data-coach-next]');
          if (await next.count()) await next.click();
          else await visible(page, '[data-coach-target]').click();
        }
        await expect(visible(page, '[data-coach-note][data-coach-step="buildFree"]')).toBeVisible();
      } else {
        await expect(
          visible(
            page,
            `[data-coach][data-coach-step="${screen === 'tutorial-first' ? 'draftIntro' : 'tokens'}"][data-placed]`
          )
        ).toBeVisible();
      }
      return;
    }
    case 'offline':
    case 'reconnecting':
      await fixture(page, 'age-1-start-seat0', `?link=${screen}`);
      expect(await page.evaluate(() => window.__ajayeb?.conn())).toBe(screen);
      await expect(visible(page, '.connection-pill')).toBeVisible();
      return;
    case 'spectator':
      await fixture(page, 'online-watching-spectator');
      expect((await currentView(page)).viewer).toBeUndefined();
      expect((await currentView(page)).choices).toEqual([]);
      return;
    case 'pass-device': {
      const driver = await live(page, origin, 'mausoleum');
      await playKey(driver, 'discard:rostrum');
      await expect(page.locator('[data-pass-device]')).toBeVisible();
      return;
    }
    case 'tooltip-board':
      await fixture(page, 'age-1-start-seat0');
      await openTooltip(page, '[data-tut="track"] [data-capital-tip="0"]');
      return;
    case 'advisor-strength': {
      const room = await createDevRoom(origin, {
        mode: 'ai',
        bot: 'easy',
        seed: '7',
        scenario: 'mausoleum'
      });
      await page.goto(room.path);
      const driver = new GameDriver(page);
      await driver.nextTurn();
      await driver.dismissToasts();
      await menu(page, 'menu-hints');
      const sheet = page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
      await expect(sheet).toBeVisible();
      // The advisor's row, with its strength picker, is the last family.
      const picker = sheet.locator('[data-hint-family="advisor"] [data-advisor-strength]');
      if ((await picker.count()) > 0) await picker.scrollIntoViewIfNeeded();
      return;
    }
  }
}
async function openTooltip(page: Page, selector: string): Promise<void> {
  const target = visible(page, selector);
  await expect(target).toBeVisible();
  if (page.viewportSize()?.width === 1280) await target.hover();
  else await target.tap();
  await expect(page.locator('#ajayeb-tooltip')).toBeVisible();
}
