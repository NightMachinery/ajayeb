// M7 acceptance: an online game between two browsers, played through the
// real UI from Home to the result.
//
// Each player has a browser context of their own (their own storage, so
// their own identity token and name, and their own network switch), in one
// browser. Context A creates the game from Home, passes the invite link on
// from the lobby's Copy button, and context B opens it, answers the name
// prompt and takes the other seat. Each side acts only on its own turn,
// through the confirm step online moves take. On the way:
//
// - a third context joins through the same link as a spectator, partway
//   through, and watches the moves arrive with no choices of its own;
// - B reloads its page and comes back to the same seat and position, with
//   the whole game in its log (the join frame's history);
// - A goes offline for a few seconds during B's turn: its page says it is
//   reconnecting and offers no move, B moves meanwhile, and once A is back
//   it catches up and the game goes on.
//
// Every frame each context receives is captured and handed to the leak
// check (fixtures/leak-check.ts) once the game is over and the server has
// revealed what was hidden. The game's seed and action log come from the
// server's room file. A second test opens an online room at the Great
// Library's draw, so the one secret only one seat may see is checked in the
// browser on every run, whatever the first game's deal. Its pick also checks
// that a pending choice's commit button waits for a dropped connection and
// takes the confirm step, like every other move button online.
//
// The server deals the first test's room from --seed-base (the order of the
// rooms the specs create decides which seed), so a run of the same specs
// replays the same game.

import { expect, test, type BrowserContext, type Page } from '@playwright/test';
import { GameDriver, type Snapshot } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { checkLeaks, describeLeaks } from '../fixtures/leak-check';
import {
  createOnlineGame,
  FrameLog,
  joinByLink,
  newDeviceContext,
  onlinePolicy,
  ORIGIN,
  readInvite,
  readRoomFile,
  type Crest
} from '../fixtures/online';
import { logLines, playKey } from '../fixtures/scenario';

/** How long A stays offline, at least. */
const OFFLINE_MS = 4_000;

/** Console lines a page may log while its network is switched off. */
const OFFLINE_NOISE = /WebSocket connection to .* failed|ERR_INTERNET_DISCONNECTED/;

const turnBar = (page: Page) => page.locator('[data-turn]').filter({ visible: true }).first();

async function hook<T>(page: Page, read: () => T): Promise<T> {
  return page.evaluate(read);
}

test.describe('online', { tag: ['@mobile', '@full'] }, () => {
  test('a full game between two browsers, with a spectator, a reload and an offline blip', async ({
    browser,
    isMobile
  }, testInfo) => {
    test.setTimeout(600_000);
    const started = Date.now();
    const contexts: BrowserContext[] = [];
    try {
      // A creates the game from Home and reads the invite in the lobby.
      const ctxA = await newDeviceContext(browser, testInfo, { clipboard: true });
      contexts.push(ctxA);
      const pageA = await ctxA.newPage();
      const problemsA = watchHygiene(pageA);
      const logA = new FrameLog(pageA, 'A');
      const crest: Crest = isMobile ? 'Moon' : 'Sun';
      const seatA = crest === 'Sun' ? 0 : 1;
      const seatB = 1 - seatA;
      const code = await createOnlineGame(pageA, 'Roxana', crest);
      await expect(pageA.locator(`[data-lobby-seat="${seatB}"][data-open]`)).toBeVisible();
      const invite = await readInvite(pageA);
      expect(invite.code).toBe(code);

      // B opens the link, gives a name and takes the open seat.
      const ctxB = await newDeviceContext(browser, testInfo);
      contexts.push(ctxB);
      const pageB = await ctxB.newPage();
      const problemsB = watchHygiene(pageB);
      const logB = new FrameLog(pageB, 'B');
      await joinByLink(pageB, invite.link, 'Babak');

      // The lobby gives way to the draft on both sides.
      await expect(pageA.locator('[data-lobby]')).toHaveCount(0);
      const roomA = await hook(pageA, () => window.__ajayeb?.room() ?? null);
      expect(roomA?.mode).toBe('online');
      expect(roomA?.status).toBe('playing');
      expect(roomA?.seats.map((s) => s.name)).toEqual(
        seatA === 0 ? ['Roxana', 'Babak'] : ['Babak', 'Roxana']
      );
      expect(await hook(pageA, () => window.__ajayeb?.you()?.seats)).toEqual([seatA]);
      expect(await hook(pageB, () => window.__ajayeb?.you()?.seats)).toEqual([seatB]);

      const driverA = new GameDriver(pageA, { confirm: true });
      const driverB = new GameDriver(pageB, { confirm: true });
      const bySeat = seatA === 0 ? [driverA, driverB] : [driverB, driverA];
      let spectator: { page: Page; driver: GameDriver; log: FrameLog; problems: string[] } | null =
        null;
      const keysSent: string[] = [];
      const milestones: Record<string, unknown> = {};
      /** Moves the spectator watched arrive (each with no choices of its own). */
      let spectatorMoves = 0;

      const others = (mover: GameDriver) =>
        [driverA, driverB, spectator?.driver].filter(
          (d): d is GameDriver => d !== undefined && d !== mover
        );

      /** One move by the seat to move, then everyone else catches up with it. */
      async function move(mover: GameDriver): Promise<void> {
        const snap = await mover.nextTurn();
        for (const other of others(mover)) {
          expect(await hook(other.page, () => window.__ajayeb?.you()?.canAct ?? false)).toBe(false);
          expect(await hook(other.page, () => window.__ajayeb?.choices().length ?? 0)).toBe(0);
        }
        const done = await mover.play(snap, onlinePolicy);
        keysSent.push(done.choice.key);
        for (const other of others(mover)) await other.waitAdvance(snap.version);
      }

      let end: Snapshot | null = null;
      for (let guard = 0; !end; guard++) {
        if (guard > 300) throw new Error('the game did not end within 300 moves');
        await driverA.waitReady();
        const now = await driverA.snapshot();
        if (now.phase === 'gameOver') {
          end = now;
          break;
        }
        if (now.toMove !== 0 && now.toMove !== 1) throw new Error(`no seat to move at ${now.ply}`);
        const mover = bySeat[now.toMove]!;

        // A spectator joins through the same link once the draft is over.
        if (!spectator && now.phase === 'play' && now.ply >= 10) {
          const ctxS = await newDeviceContext(browser, testInfo);
          contexts.push(ctxS);
          const page = await ctxS.newPage();
          const problems = watchHygiene(page);
          const log = new FrameLog(page, 'spectator');
          await joinByLink(page, invite.link, 'Mitra');
          const driver = new GameDriver(page);
          spectator = { page, driver, log, problems };
          expect(await hook(page, () => window.__ajayeb?.you())).toEqual({
            seats: [],
            canAct: false,
            admin: false
          });
          expect(await hook(page, () => window.__ajayeb?.room()?.version)).toBe(now.version);
          await expect(turnBar(page)).toHaveAttribute('data-turn', 'spectating');
          await expect(page.locator('[data-act]')).toHaveCount(0);
          // The players see one spectator.
          for (const p of [pageA, pageB]) {
            await expect(p.locator('[data-spectators="1"]').filter({ visible: true })).toHaveCount(
              1
            );
          }
          milestones.spectatorJoinedAtPly = now.ply;
        }

        // B reloads on its own turn and comes back to the same seat and position.
        if (!milestones.reload && now.phase === 'play' && now.ply >= 18 && mover === driverB) {
          const before = await driverB.nextTurn();
          const linesBefore = await logLines(driverB);
          const historyBefore = await pageB.evaluate(() => window.__ajayeb?.history() ?? []);
          expect(historyBefore.map((e) => e.ply)).toEqual([...Array(before.ply).keys()]);
          await pageB.reload();
          await expect(pageB.locator('[data-name-prompt]')).toHaveCount(0);
          await driverB.waitTurn();
          const after = await driverB.snapshot();
          expect(after.version).toBe(before.version);
          expect(after.ply).toBe(before.ply);
          expect(after.seats).toEqual([seatB]);
          expect(after.toMove).toBe(seatB);
          expect(after.choices.map((c) => c.key)).toEqual(before.choices.map((c) => c.key));
          expect(logB.sockets).toBe(2);
          // The log after the reload is the whole game, as before it.
          expect(await pageB.evaluate(() => window.__ajayeb?.history() ?? [])).toEqual(
            historyBefore
          );
          const linesAfter = await logLines(driverB);
          expect(linesAfter).toEqual(linesBefore);
          expect(linesAfter.length).toBeGreaterThanOrEqual(before.ply);
          milestones.reload = { ply: before.ply, version: before.version };
        }

        // A goes offline during B's turn, and B moves meanwhile.
        if (!milestones.offline && now.phase === 'play' && now.age === 2 && mover === driverB) {
          const off = Date.now();
          await ctxA.setOffline(true);
          await expect(turnBar(pageA)).toHaveAttribute('data-turn', /^(reconnecting|offline)$/);
          await expect(pageA.locator('.connection-pill').filter({ visible: true })).toHaveCount(1);
          expect(await hook(pageA, () => window.__ajayeb?.ready())).toBe(false);
          const conn = await hook(pageA, () => window.__ajayeb?.conn());
          // B's page sees A's seat go away.
          await expect
            .poll(() =>
              hook(pageB, () => window.__ajayeb?.room()?.seats.map((s) => s.connected) ?? [])
            )
            .toEqual(seatA === 0 ? [false, true] : [true, false]);
          await expect(turnBar(pageB)).toHaveAttribute('data-turn', 'yourTurn');

          const snapB = await driverB.nextTurn();
          const done = await driverB.play(snapB, onlinePolicy);
          keysSent.push(done.choice.key);
          if (spectator) await spectator.driver.waitAdvance(snapB.version);
          // A, still offline, has not seen it and offers no move.
          const stale = await hook(pageA, () => window.__ajayeb?.room()?.version);
          expect(stale).toBe(snapB.version);
          await expect(pageA.locator('[data-act]:enabled')).toHaveCount(0);
          const wait = OFFLINE_MS - (Date.now() - off);
          if (wait > 0) await pageA.waitForTimeout(wait);

          await ctxA.setOffline(false);
          const back = Date.now();
          await driverA.waitAdvance(snapB.version, 30_000);
          await expect(pageA.locator('.connection-pill').filter({ visible: true })).toHaveCount(0);
          await expect
            .poll(() =>
              hook(pageB, () => window.__ajayeb?.room()?.seats.map((s) => s.connected) ?? [])
            )
            .toEqual([true, true]);
          milestones.offline = {
            ply: snapB.ply,
            conn,
            offlineMs: back - off,
            recoverMs: Date.now() - back,
            sockets: logA.sockets
          };
          continue;
        }

        await move(mover);
        if (spectator) {
          // The spectator's view moved on with no choices of its own.
          expect(await hook(spectator.page, () => window.__ajayeb?.choices().length)).toBe(0);
          spectatorMoves += 1;
        }
      }

      expect(milestones.spectatorJoinedAtPly, 'a spectator joined').toBeDefined();
      expect(milestones.reload, 'B reloaded').toBeDefined();
      expect(milestones.offline, 'A went offline in Age II').toBeDefined();
      if (!spectator) throw new Error('no spectator');

      // Everyone reached the same result, the server's.
      const result = end.result;
      expect(result, 'a finished game carries its result').toBeTruthy();
      if (!result) return;
      for (const page of [pageB, spectator.page]) {
        await page.waitForFunction(() => window.__ajayeb?.view()?.phase === 'gameOver');
        expect(await hook(page, () => window.__ajayeb?.view()?.result)).toEqual(result);
      }
      const file = readRoomFile(code);
      expect(file.status).toBe('finished');
      expect(file.actions, 'the server applied exactly the moves the two pages sent').toEqual(
        keysSent
      );
      expect(file.actions.length).toBe(end.ply);
      expect(file.result).toMatchObject({
        kind: result.victory,
        scores: result.scores.map((s) => expect.objectContaining({ total: s.total }))
      });
      if (result.winner !== undefined) expect(file.result?.winner).toBe(result.winner);

      // Both result screens say the same, each from its own side.
      const nameOf = (seat: number) => (seat === seatA ? 'Roxana' : 'Babak');
      const views: [Page, number | null][] = [
        [pageA, seatA],
        [pageB, seatB],
        [spectator.page, null]
      ];
      for (const [page, seat] of views) {
        const over = page.locator('[data-game-over]');
        await expect(over).toHaveAttribute(
          'data-victory',
          result.shared ? 'shared' : result.victory
        );
        if (result.winner !== undefined) {
          await expect(over).toHaveAttribute('data-winner', String(result.winner));
        }
        const dialog = page
          .locator('dialog[open]')
          .filter({ has: page.locator('[data-score-table]') });
        await expect(dialog).toBeVisible();
        const headline = result.shared
          ? 'Both players win'
          : result.winner === seat
            ? 'You win'
            : `${nameOf(result.winner ?? 0)} wins`;
        await expect(dialog.locator('h2')).toHaveText(headline);
      }

      // No frame told any of the three more than it was entitled to.
      const streams = [logA, logB, spectator.log];
      const report = checkLeaks(streams, { seed: file.seed });
      expect(report.leaks, describeLeaks(report)).toEqual([]);
      expect(report.problems).toEqual([]);
      expect(report.coverage.reveal).toBe(true);
      expect(report.coverage.faceDownUnknown).toBe(0);
      expect(report.coverage.seedChecked).toBe(true);
      // The spectator's frames never carried a choice.
      for (const f of spectator.log.frames) {
        const msg = JSON.parse(f.text) as { t: string; view?: { choices: unknown[] } };
        if (msg.t === 'state') expect(msg.view?.choices).toEqual([]);
      }

      const summary = {
        project: testInfo.project.name,
        crest,
        seconds: Math.round((Date.now() - started) / 1000),
        plies: end.ply,
        moves: { A: driverA.stats.actions, B: driverB.stats.actions },
        confirms: driverA.stats.confirms + driverB.stats.confirms,
        flows: {
          A: driverA.stats.flows,
          B: driverB.stats.flows
        },
        pendingKinds: { A: driverA.stats.pendingKinds, B: driverB.stats.pendingKinds },
        milestones,
        spectatorMoves,
        frames: Object.fromEntries(streams.map((s) => [s.name, s.frames.length])),
        stateFrames: report.coverage.stateFrames,
        leakCheck: {
          faceDownChecks: report.coverage.faceDownChecks,
          faceDownIdentities: report.coverage.faceDownIdentities,
          secrets: report.coverage.secrets,
          ageCards: report.coverage.ageCards,
          draftRound2: report.coverage.draftRound2,
          library: report.coverage.library,
          seedChecked: report.coverage.seedChecked
        },
        victory: result.victory,
        winner: result.winner ?? null,
        totals: result.scores.map((s) => s.total)
      };
      console.log(`online game: ${JSON.stringify(summary)}`);
      await testInfo.attach('game-summary.json', {
        body: JSON.stringify(summary, null, 2),
        contentType: 'application/json'
      });

      // Nothing went wrong on any page, beyond A's failed reconnects while offline.
      expect(problemsA.filter((p) => !OFFLINE_NOISE.test(p))).toEqual([]);
      expect(problemsB).toEqual([]);
      expect(spectator.problems).toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });

  test('the Great Library offer reaches only the decider (great-library)', async ({
    browser
  }, testInfo) => {
    test.setTimeout(120_000);
    const contexts: BrowserContext[] = [];
    try {
      // Seat 0 opens a seeded online room at the scenario; seat 1 and a
      // spectator join through its link.
      const ctxA = await newDeviceContext(browser, testInfo);
      const ctxB = await newDeviceContext(browser, testInfo);
      const ctxS = await newDeviceContext(browser, testInfo);
      contexts.push(ctxA, ctxB, ctxS);
      const [pageA, pageB, pageS] = await Promise.all([
        ctxA.newPage(),
        ctxB.newPage(),
        ctxS.newPage()
      ]);
      const problems = [watchHygiene(pageA), watchHygiene(pageB), watchHygiene(pageS)];
      const logs = [
        new FrameLog(pageA, 'seat0'),
        new FrameLog(pageB, 'seat1'),
        new FrameLog(pageS, 'spectator')
      ];
      await pageA.goto('/dev/new?mode=online&seed=7&scenario=great-library');
      await pageA.waitForURL(/\/play\/[A-Z0-9]+$/);
      await expect(pageA.locator('[data-lobby]')).toBeVisible();
      const link = new URL(new URL(pageA.url()).pathname, ORIGIN).href;
      await joinByLink(pageB, link, 'Babak');
      await joinByLink(pageS, link, 'Mitra');

      const a = new GameDriver(pageA, { confirm: true });
      const b = new GameDriver(pageB, { confirm: true });
      expect(await hook(pageA, () => window.__ajayeb?.you()?.seats)).toEqual([0]);
      expect(await hook(pageB, () => window.__ajayeb?.you()?.seats)).toEqual([1]);

      const { before } = await playKey(a, 'wonder:great-library:rostrum', 'wonderFirst');
      for (const p of [pageB, pageS]) {
        await new GameDriver(p).waitAdvance(before.version);
      }
      // The drawer sees the three tokens; seat 1 and the spectator see that
      // three were drawn, and no trace of which.
      const offer = ['law', 'philosophy', 'strategy'];
      expect(await hook(pageA, () => window.__ajayeb?.view()?.pending?.tokens)).toEqual(offer);
      for (const p of [pageB, pageS]) {
        const pending = await hook(p, () => window.__ajayeb?.view()?.pending ?? null);
        expect(pending).toMatchObject({
          kind: 'pickLibraryToken',
          seat: 0,
          tokens: [],
          hiddenTokens: 3
        });
        // The waiting side's host is only a wrapper of the peek bar, which
        // says what is happening without naming a token.
        await expect(p.locator('[data-choice-host][data-mine="false"]')).toHaveCount(1);
        await expect(p.getByText('Sun is choosing from 3 hidden tokens')).toBeVisible();
        for (const token of offer) {
          await expect(p.locator(`[data-token="${token}"]`)).toHaveCount(0);
        }
      }
      await expect(pageA.locator('[data-choice-host][data-mine="true"]')).toHaveCount(1);
      await expect(
        pageA.locator('[data-token-choice="library"] [data-token]').filter({ visible: true })
      ).toHaveCount(3);

      // The pending panel follows the card sheet's act-button rule. While
      // A's connection is down its commit button is disabled and waits for
      // the connection; once A is back, the pick takes the confirm step
      // (the driver checks the first tap only arms it).
      const commit = pageA
        .locator('[data-token-choice="library"] [data-commit] button')
        .filter({ visible: true });
      await pageA.locator('[data-token-choice="library"] [data-pick="token:law"]').click();
      await expect(commit).toHaveAttribute('data-act', 'token:law');
      await expect(commit).toBeEnabled();
      await ctxA.setOffline(true);
      await expect(turnBar(pageA)).toHaveAttribute('data-turn', /^(reconnecting|offline)$/);
      await expect(commit).toBeDisabled();
      await expect(commit).toHaveText('Waiting for connection');
      await ctxA.setOffline(false);
      await expect(commit).toBeEnabled({ timeout: 30_000 });
      await a.waitReady(30_000);
      const confirmsBefore = a.stats.confirms;
      const pick = await playKey(a, 'token:law');
      expect(a.stats.confirms).toBe(confirmsBefore + 1);
      await b.waitAdvance(pick.before.version);
      await new GameDriver(pageS).waitAdvance(pick.before.version);
      // The game goes on: one move each.
      await b.play(await b.nextTurn(), onlinePolicy);
      await a.play(await a.nextTurn(), onlinePolicy);
      for (const p of [pageB, pageS]) {
        await expect(p.locator('[data-token="philosophy"], [data-token="strategy"]')).toHaveCount(
          0
        );
      }

      const report = checkLeaks(logs);
      expect(report.leaks, describeLeaks(report)).toEqual([]);
      expect(report.problems).toEqual([]);
      expect(report.coverage.library).toEqual([expect.objectContaining({ decider: 0, offer })]);
      expect(report.coverage.library[0]?.otherFrames).toBeGreaterThanOrEqual(2);
      console.log(
        `great-library online: frames ${JSON.stringify(report.coverage.frames)}, ` +
          `library ${JSON.stringify(report.coverage.library)}`
      );
      // Nothing went wrong on any page, beyond A's failed reconnects while offline.
      const [problemsA = [], ...rest] = problems;
      expect(problemsA.filter((p) => !OFFLINE_NOISE.test(p))).toEqual([]);
      for (const p of rest) expect(p).toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });
});
