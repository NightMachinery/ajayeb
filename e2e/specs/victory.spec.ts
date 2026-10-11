// Every way a game ends, each from a named scenario (server/internal/engine/
// scenarios_named.go) played through the real UI: the scenario opens a
// hot-seat room just before its trigger, the spec plays the trigger's action
// keys with the clicks a player makes, and the result screen must name the
// victory and the winner the scenario's Expect describes and show the
// server's own score sheet (window.__ajayeb), not numbers computed here.

import { readFile } from 'node:fs/promises';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import {
  CREST_NAMES,
  currentView,
  expectResultScreen,
  openHotseatRoom,
  playKey,
  type ExpectedResult
} from '../fixtures/scenario';
import type { Flow, GameDriver } from '../fixtures/driver';

interface Case {
  scenario: string;
  title: string;
  /** The trigger, with the flow to play each key by when not the default. */
  trigger: readonly (string | [string, Flow])[];
  expected: ExpectedResult;
}

/** Opens the game menu and reports whether it offers the replay download. */
async function menuOffersReplay(driver: GameDriver): Promise<boolean> {
  await driver.visible('[data-control="menu"]').click();
  const menu = driver.visible('[data-game-menu]');
  await expect(menu.locator('[data-control="menu-home"]')).toBeVisible();
  return (await menu.locator('[data-control="menu-replay"]').count()) > 0;
}

/**
 * The finished game's replay download (docs/protocol.md, "Downloading a
 * game's log"): the game menu offers it only after the end, and the saved
 * file is the room's ajayeb-log/1 log, named after the room code.
 */
async function expectReplayDownload(page: Page, driver: GameDriver, plies: number): Promise<void> {
  await page.locator('[data-view-board]').filter({ visible: true }).first().click();
  expect(await menuOffersReplay(driver), 'the menu offers the replay once the game is over').toBe(
    true
  );
  const code = await page.evaluate(() => window.__ajayeb?.room()?.code ?? '');
  expect(code).not.toBe('');
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    driver.visible('[data-game-menu] [data-control="menu-replay"]').click()
  ]);
  expect(download.suggestedFilename()).toBe(`ajayeb-${code}.json`);
  const path = await download.path();
  const log = JSON.parse(await readFile(path, 'utf8')) as {
    format: string;
    seed: string;
    scenario: unknown;
    actions: { ply: number; seat: number; action: string }[];
    finalHash: string;
  };
  expect(log.format).toBe('ajayeb-log/1');
  expect(log.seed).toMatch(/^\d+$/);
  expect(log.scenario, "a scenario room's log carries its scenario").toBeTruthy();
  expect(log.actions).toHaveLength(plies);
  expect(log.finalHash).toMatch(/^[0-9a-f]{16}$/);
}

const CASES: readonly Case[] = [
  {
    scenario: 'military-supremacy',
    title: 'military supremacy: two shields reach the capital',
    trigger: ['construct:walls'],
    expected: { victory: 'military', winner: 0 }
  },
  {
    scenario: 'military-via-wonder',
    title: "military supremacy: a wonder's shield ends the game before its destroy choice",
    trigger: [['wonder:statue-of-zeus:rostrum', 'wonderFirst']],
    expected: { victory: 'military', winner: 0 }
  },
  {
    scenario: 'science-supremacy',
    title: 'scientific supremacy: a sixth distinct symbol from a card',
    trigger: ['construct:university'],
    expected: { victory: 'science', winner: 0 }
  },
  {
    scenario: 'law-supremacy',
    title: "scientific supremacy: Law's scales as the sixth symbol",
    trigger: ['construct:academy', 'token:law'],
    expected: { victory: 'science', winner: 0 }
  },
  {
    scenario: 'civilian-end',
    title: 'civilian victory: the last Age III card ends the game',
    trigger: ['discard:academy'],
    expected: { victory: 'civilian', winner: 0 }
  },
  {
    scenario: 'tie-blue',
    title: 'civilian victory: a tie on points broken by blue cards',
    trigger: ['discard:academy'],
    expected: { victory: 'civilian', winner: 0, tieBreak: true }
  },
  {
    scenario: 'draw',
    title: 'shared victory: equal points and equal blue points',
    trigger: ['discard:academy'],
    expected: { victory: 'shared', winner: null }
  }
];

test.describe('victory', { tag: ['@mobile'] }, () => {
  for (const c of CASES) {
    test(`${c.title} (${c.scenario})`, async ({ page }) => {
      const problems = watchHygiene(page);
      const driver = await openHotseatRoom(page, c.scenario);
      // One case checks the replay download, which the menu hides until the end.
      const checkReplay = c.scenario === 'military-supremacy';
      if (checkReplay) {
        expect(await menuOffersReplay(driver), 'no replay while the game runs').toBe(false);
        await page.keyboard.press('Escape');
        await expect(page.locator('[data-game-menu]')).toBeHidden();
      }

      for (const step of c.trigger) {
        const [key, flow] = typeof step === 'string' ? [step, undefined] : step;
        const before = await currentView(page);
        expect(before.phase, `the game is still on before ${key}`).not.toBe('gameOver');
        await playKey(driver, key, flow);
      }

      const end = await driver.nextTurn();
      expect(end.phase).toBe('gameOver');
      const result = await expectResultScreen(page, c.expected);
      // The scene of how it ended; a draw shows the civilian scene until a
      // draw scene of its own is approved.
      await expect(page.locator('[data-victory-scene]')).toHaveAttribute(
        'data-victory-scene',
        c.expected.victory === 'shared'
          ? /^victory\/(draw|civilian)$/
          : `victory/${c.expected.victory}`
      );
      const view = await currentView(page);

      switch (c.scenario) {
        case 'military-supremacy':
        case 'military-via-wonder':
          // The pawn stands in seat 1's capital, and the destroy choice the
          // Statue of Zeus would open never did.
          expect(view.military.position).toBeGreaterThanOrEqual(9);
          expect(view.pending).toBeUndefined();
          break;
        case 'law-supremacy':
          expect(view.players[0].tokens).toContain('law');
          break;
        case 'tie-blue':
          expect(result.scores[0].total).toBe(result.scores[1].total);
          expect(result.scores[0].blue).toBeGreaterThan(result.scores[1].blue);
          break;
        case 'draw':
          expect(result.scores[0].total).toBe(result.scores[1].total);
          expect(result.scores[0].blue).toBe(result.scores[1].blue);
          break;
        case 'civilian-end':
          expect(result.scores[0].total).toBeGreaterThan(result.scores[1].total);
          break;
      }

      // The turn bar says the game is over, and no move is offered.
      await expect(driver.visible('[data-turn]')).toHaveAttribute('data-turn', 'gameOver');
      expect(view.choices).toEqual([]);
      if (checkReplay) await expectReplayDownload(page, driver, c.trigger.length);
      expect(problems, 'console errors, page errors or failed requests').toEqual([]);
    });
  }

  test('a forfeit from the game menu, behind a confirm step (military-supremacy)', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'military-supremacy');
    const before = await currentView(page);
    const loser = before.toMove as 0 | 1;
    const winner = loser === 0 ? 1 : 0;
    const confirm = page.locator('dialog[open] [data-forfeit-confirm]');

    // The menu item opens the confirm step, which names the player to move
    // (hot-seat) and the winner; Keep playing sends nothing.
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-game-menu] [data-control="menu-forfeit"]').click();
    await expect(confirm).toBeVisible();
    await expect(confirm.locator('h2')).toHaveText(`Forfeit the game for ${CREST_NAMES[loser]}?`);
    await expect(confirm).toContainText(`${CREST_NAMES[winner]} wins at once.`);
    // It opens on the safe answer, never on Forfeit.
    await expect(confirm.locator('[data-control="forfeit-cancel"]')).toBeFocused();
    await confirm.locator('[data-control="forfeit-cancel"]').click();
    await expect(confirm).toHaveCount(0);
    const still = await currentView(page);
    expect(still.phase).toBe('play');
    expect(still.choices).toEqual(before.choices);

    // Forfeit: the other seat wins at once, and the result says who gave up.
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-game-menu] [data-control="menu-forfeit"]').click();
    await confirm.locator('[data-control="forfeit-confirm"]').click();
    const end = await driver.nextTurn();
    expect(end.phase).toBe('gameOver');
    await expectResultScreen(page, { victory: 'forfeit', winner });
    await expect(
      page.locator('dialog[open]').filter({ has: page.locator('[data-score-table]') })
    ).toContainText(`${CREST_NAMES[loser]} forfeited.`);

    // The game is over: the menu offers no forfeit any more.
    await page.locator('[data-view-board]').filter({ visible: true }).first().click();
    await driver.visible('[data-control="menu"]').click();
    await expect(driver.visible('[data-game-menu] [data-control="menu-home"]')).toBeVisible();
    await expect(page.locator('[data-game-menu] [data-control="menu-forfeit"]')).toHaveCount(0);
    expect(problems).toEqual([]);
  });
});
