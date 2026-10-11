// M6 acceptance: games against the bot, played through the real UI on the
// desktop and mobile projects.
//
// The first test plays a whole game against the Apprentice (easy) from Home
// to the result and back. The desktop project plays at the Sun and the
// mobile project at the Moon, so the military track is drawn both ways
// round. Midway, one undo takes back the player's move together with the
// bot's reply. The second test starts a game against the Scholar (normal),
// leaves it with the game menu and comes back through Home's Continue list.
//
// Every move is a click on the visible controls, chosen by the shared
// driver's policy (fixtures/driver.ts), and the bot's turns are waited out
// through window.__ajayeb (the page ready again with a move of ours to
// make), never with fixed sleeps. The server runs with --bot-delay 0, so the
// pacing a player sees is the client's own: the thinking frame stays up for
// the Presenter's dwell, then the bot's pick is previewed. The server deals
// each room from --seed-base, and the Apprentice is deterministic, so a run
// of the same specs replays the same games.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver, type Snapshot } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';

type Crest = 'Sun' | 'Moon';

/** Home, the vs-AI sheet, a level and a crest, Start; returns the room code. */
async function startAiGame(page: Page, level: string, crest: Crest): Promise<string> {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
  await page.getByRole('button', { name: /Play vs AI/ }).click();
  const sheet = page.locator('dialog[open]');
  await expect(sheet.getByRole('heading', { name: 'Choose your rival' })).toBeVisible();
  // Each level's choice shows its persona's avatar.
  await expect(sheet.locator('[data-avatar]')).toHaveCount(3);
  for (const level of ['easy', 'normal', 'hard']) {
    await expect(sheet.locator(`[data-avatar="${level}"]`)).toBeVisible();
  }
  await sheet.getByText(level, { exact: true }).click();
  await expect(sheet.getByRole('radio', { name: new RegExp(level) })).toBeChecked();
  await sheet.getByText(crest, { exact: true }).click();
  await expect(sheet.getByRole('radio', { name: crest, exact: true })).toBeChecked();
  await sheet.getByRole('button', { name: 'Start game' }).click();
  await page.waitForURL(/\/play\/[A-Z0-9]+$/);
  const code = new URL(page.url()).pathname.split('/').pop();
  if (!code) throw new Error(`no room code in ${page.url()}`);
  return code;
}

/** The structure as a comparable list: each slot's state and card. */
async function structure(page: Page): Promise<string[]> {
  return page.evaluate(
    () =>
      window.__ajayeb?.view()?.structure.map((s) => `${s.index}:${s.state}:${s.card ?? '-'}`) ?? []
  );
}

const keys = (snap: Snapshot) => snap.choices.map((c) => c.key);

/** The move the undo takes back: one in the middle of Age I or II, from a card. */
function undoCandidate(snap: Snapshot): boolean {
  return (
    snap.phase === 'play' &&
    snap.age <= 2 &&
    snap.ply >= 14 &&
    snap.pending === null &&
    snap.choices.some((c) => c.kind === 'construct' || c.kind === 'discard')
  );
}

test.describe('vs AI', { tag: ['@mobile', '@full'] }, () => {
  test('a full game against the Apprentice, with an undo, from Home to the result and back', async ({
    page,
    isMobile
  }, testInfo) => {
    test.setTimeout(420_000);
    const problems = watchHygiene(page);
    const driver = new GameDriver(page);
    await driver.watchThinking();
    const started = Date.now();

    const crest: Crest = isMobile ? 'Moon' : 'Sun';
    const mySeat = crest === 'Sun' ? 0 : 1;
    const botSeat = 1 - mySeat;
    const code = await startAiGame(page, 'Apprentice', crest);

    await driver.waitReady();
    const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
    expect(room?.code).toBe(code);
    expect(room?.mode).toBe('ai');
    expect(room?.seats.find((s) => s.seat === botSeat)).toMatchObject({ kind: 'bot', bot: 'easy' });
    expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([mySeat]);

    // Play to the end, taking one move back on the way.
    let undo: {
      ply: number;
      replyPly: number;
      undoneSlots: number;
      seconds: number;
    } | null = null;
    let trackChecked = false;
    let end: Snapshot | null = null;
    while (!end) {
      const snap = await driver.nextTurn();
      if (snap.phase === 'gameOver') {
        end = snap;
        break;
      }
      expect(snap.toMove).toBe(mySeat);
      expect(snap.thinking).toBeNull();
      if (driver.stats.actions >= 300) throw new Error('the game did not end within 300 actions');

      // The track puts the player's capital at its left end: mirrored at the Moon.
      if (!trackChecked && snap.phase === 'play') {
        const track = page.locator('.military-track').filter({ visible: true }).first();
        await expect(track).toHaveAttribute('data-left-seat', String(mySeat));
        if (mySeat === 1) await expect(track).toHaveAttribute('data-mirrored', '');
        else await expect(track).not.toHaveAttribute('data-mirrored');
        trackChecked = true;
      }

      if (undo || !undoCandidate(snap)) {
        await driver.play(snap);
        continue;
      }

      // The undo: play, let the bot answer, then take both back.
      const before = await structure(page);
      await driver.play(snap);
      const replied = await driver.nextTurn();
      const afterReply = await structure(page);
      // Only a position the bot has answered in (two plies or more later,
      // still in play) is a fair test; otherwise try again at the next move.
      if (replied.phase !== 'play' || replied.ply < snap.ply + 2 || !replied.canUndo) continue;
      const taken = before.filter((slot, i) => slot !== afterReply[i]).length;
      expect(taken, 'the player and the bot each took a card').toBeGreaterThanOrEqual(2);

      const undoStarted = Date.now();
      const back = await driver.undo();
      // The bot's reply is gone with the player's move: the same position,
      // the same structure, the same choices, and the player's turn again.
      expect(back.ply).toBe(snap.ply);
      expect(back.toMove).toBe(mySeat);
      expect(back.thinking).toBeNull();
      expect(keys(back)).toEqual(keys(snap));
      expect(await structure(page)).toEqual(before);
      expect(await page.evaluate(() => window.__ajayeb?.you()?.canAct)).toBe(true);
      await expect(page.locator('[data-turn]').filter({ visible: true }).first()).toHaveAttribute(
        'data-turn',
        'yourTurn'
      );
      undo = {
        ply: snap.ply,
        replyPly: replied.ply,
        undoneSlots: taken,
        seconds: Math.round((Date.now() - undoStarted) / 100) / 10
      };
    }

    expect(undo, 'one undo was made mid-game').not.toBeNull();
    const thinking = await driver.thinkingSeen();
    expect(thinking.turnBar, 'the turn bar said the bot was thinking').toBeGreaterThan(0);
    expect(thinking.panel, "the bot's strip or panel showed it thinking").toBeGreaterThan(0);

    const { stats } = driver;
    const result = end.result;
    // The final log holds every action of the game, the undone ones excluded.
    const humanInLog = stats.actions - 1;
    const summary = {
      project: testInfo.project.name,
      crest,
      seconds: Math.round((Date.now() - started) / 1000),
      plies: end.ply,
      humanActions: stats.actions,
      botActionsInLog: end.ply - humanInLog,
      flows: stats.flows,
      pendingKinds: stats.pendingKinds,
      undo,
      thinking,
      victory: result?.victory,
      winner: result?.winner ?? null,
      totals: result?.scores.map((s) => s.total)
    };
    console.log(`vs-AI game: ${JSON.stringify(summary)}`);
    await testInfo.attach('game-summary.json', {
      body: JSON.stringify(summary, null, 2),
      contentType: 'application/json'
    });

    // The result screen shows the server's result.
    expect(result, 'a finished game carries its result').toBeTruthy();
    if (!result) return;
    const over = page.locator('[data-game-over]');
    await expect(over).toHaveAttribute('data-victory', result.shared ? 'shared' : result.victory);
    if (result.winner !== undefined) {
      await expect(over).toHaveAttribute('data-winner', String(result.winner));
    }
    const dialog = page.locator('dialog[open]').filter({ has: page.locator('[data-score-table]') });
    await expect(dialog).toBeVisible();
    const headline = result.shared
      ? 'Both players win'
      : result.winner === mySeat
        ? 'You win'
        : 'Apprentice wins';
    await expect(dialog.locator('h2')).toHaveText(headline);
    const table = dialog.locator('[data-score-table]');
    await expect(table.locator('thead')).toContainText('Apprentice');
    const categories = [
      'blue',
      'green',
      'yellow',
      'purple',
      'wonders',
      'tokens',
      'coins',
      'military',
      'total'
    ] as const;
    for (const category of categories) {
      await expect(table.locator(`[data-score-row="${category}"] td`)).toHaveText([
        String(result.scores[0][category]),
        String(result.scores[1][category])
      ]);
    }

    // Home, which no longer offers to continue the finished game.
    await dialog.locator('[data-home]').click();
    await page.waitForURL((url) => url.pathname === '/');
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Continue:/ })).toHaveCount(0);
    const saved = await page.evaluate(() => localStorage.getItem('ajayeb:rooms') ?? '[]');
    expect(saved).not.toContain(code);

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('a game against the Scholar, left with the game menu and continued from Home', async ({
    page,
    isMobile
  }) => {
    test.setTimeout(120_000);
    const problems = watchHygiene(page);
    const driver = new GameDriver(page);
    const crest: Crest = isMobile ? 'Moon' : 'Sun';
    const code = await startAiGame(page, 'Scholar', crest);

    await driver.waitReady();
    expect((await page.evaluate(() => window.__ajayeb?.room() ?? null))?.seats).toContainEqual(
      expect.objectContaining({ kind: 'bot', bot: 'normal' })
    );

    // A few moves of ours, each answered by the bot.
    for (let i = 0; i < 3; i++) await driver.play(await driver.nextTurn());
    const left = await driver.nextTurn();
    expect(left.phase).not.toBe('gameOver');
    expect(driver.stats.actions).toBe(3);

    // Leave with the game menu.
    await driver.dismissToasts();
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-game-menu] [data-control="menu-home"]').click();
    await page.waitForURL((url) => url.pathname === '/');
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();

    // Continue it from Home: the same position, version and choices.
    const entry = page.getByRole('button', { name: `Continue: vs AI · ${code}` });
    await expect(entry).toBeVisible();
    await entry.click();
    await page.waitForURL((url) => url.pathname === `/play/${code}`);
    await driver.waitTurn();
    const back = await driver.snapshot();
    expect(back.version).toBe(left.version);
    expect(back.ply).toBe(left.ply);
    expect(back.phase).toBe(left.phase);
    expect(back.toMove).toBe(left.toMove);
    expect(keys(back)).toEqual(keys(left));
    expect(back.canUndo).toBe(left.canUndo);

    // And it plays on.
    await driver.play(back);
    await driver.nextTurn();

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
