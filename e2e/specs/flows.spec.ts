// Every special choice the game can ask for, each played through the real
// UI: the wonder draft of an ordinary new game, then one named scenario per
// flow (server/internal/engine/scenarios_named.go), opened in a hot-seat room
// just before its trigger. For each, the spec checks that the pending panel
// appears with the options the server offers and nothing else, that the
// choice is made with the clicks a player makes, and that it lands: the
// server's next view (window.__ajayeb) shows the scenario's Expect.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import type { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { currentView, logLines, openHotseatRoom, playKey } from '../fixtures/scenario';

const host = '[data-choice-host]';

/** The open pending panel of the decider, with its kind and seat. */
async function expectPanel(page: Page, kind: string, decider: 0 | 1): Promise<void> {
  const panel = page.locator(`${host}[data-mine="true"]`);
  await expect(panel).toHaveAttribute('data-choice-host', kind);
  await expect(panel).toHaveAttribute('data-decider', String(decider));
  await expect(panel.locator('dialog[open]')).toBeVisible();
}

/** The data-pick keys of the open panel's options, in the order shown. */
async function panelPicks(page: Page): Promise<string[]> {
  return page
    .locator(`${host} dialog[open] [data-pick]`)
    .evaluateAll((els) => els.map((el) => el.getAttribute('data-pick') ?? ''));
}

/**
 * A skipped choice is explained on screen right after the move: in the move
 * toast, or (on a touch device) in the pass-device screen's summary.
 */
async function expectSkipShown(page: Page, line: string): Promise<void> {
  await expect(
    page.locator('[data-toast], [data-pass-device]').filter({ hasText: line }).first()
  ).toBeVisible();
}

/** After a move, the turn is the other seat's (past any pass-device screen). */
async function expectTurnPassedTo(driver: GameDriver, seat: 0 | 1): Promise<void> {
  const snap = await driver.nextTurn();
  expect(snap.phase).toBe('play');
  expect(snap.pending).toBeNull();
  expect(snap.toMove).toBe(seat);
  await expect(driver.page.locator(host)).toHaveCount(0);
}

test.describe('flows', { tag: ['@mobile'] }, () => {
  test('the wonder draft of a new game, both rounds', async ({ page }) => {
    const problems = watchHygiene(page);
    // An ordinary start (no scenario), dealt from a fixed seed.
    const driver = await openHotseatRoom(page, null, 4242);

    const start = await currentView(page);
    expect(start.phase).toBe('wonderDraft');
    expect(start.draft?.round).toBe(1);
    const first = start.firstPlayer as 0 | 1;
    const second = first === 0 ? 1 : 0;
    // Round 1: first, second, second, and the last wonder to the first;
    // round 2: second, first, first, and the last to the second.
    const pickers: (0 | 1)[] = [first, second, second, second, first, first];
    const offered: string[] = [];

    for (let pick = 0; pick < pickers.length; pick++) {
      const snap = await driver.nextTurn();
      expect(snap.phase, `pick ${pick + 1}`).toBe('wonderDraft');
      expect(snap.toMove, `pick ${pick + 1} is ${pickers[pick]}'s`).toBe(pickers[pick]);
      const view = await currentView(page);
      const round = pick < 3 ? 1 : 2;
      expect(view.draft?.round).toBe(round);
      // The draft screen shows the round's four wonders, the taken ones
      // marked; every untaken one is a choice and nothing else is.
      const offer = view.draft?.offer ?? [];
      expect(offer).toHaveLength(4);
      if (pick === 0 || pick === 3) offered.push(...offer.map((o) => o.wonder));
      const draft = page.locator('[data-draft][data-round]');
      await expect(draft).toHaveAttribute('data-round', String(round));
      await expect(draft.locator('[data-wonder-offer]')).toHaveCount(4);
      const untaken = offer.filter((o) => o.takenBy === undefined).map((o) => `draft:${o.wonder}`);
      expect(snap.choices.map((c) => c.key).sort()).toEqual([...untaken].sort());

      const choice = snap.choices[0];
      if (!choice?.wonder) throw new Error('no draft choice');
      await driver.perform({ flow: 'draft', choice }, snap);
      await driver.waitAdvance(snap.version);

      // The wonder is the picker's, on the screen and in the server's view.
      const after = await currentView(page);
      const picker = pickers[pick] ?? 0;
      expect(after.players[picker].wonders.map((w) => w.wonder)).toContain(choice.wonder);
      if (after.phase === 'wonderDraft') {
        await expect(
          page.locator(`[data-draft-picks="${picker}"] [data-picked="${choice.wonder}"]`)
        ).toHaveCount(1);
      }
    }

    // Six picks by hand and two last wonders dealt: Age I begins with four
    // wonders each, all eight of them the ones offered.
    const snap = await driver.nextTurn();
    expect(snap.phase).toBe('play');
    expect(snap.age).toBe(1);
    expect(snap.toMove).toBe(first);
    expect(driver.stats.flows.draft).toBe(6);
    const view = await currentView(page);
    const owned = view.players.flatMap((p) => p.wonders.map((w) => w.wonder));
    expect(view.players[0].wonders).toHaveLength(4);
    expect(view.players[1].wonders).toHaveLength(4);
    expect(owned.sort()).toEqual(offered.sort());
    await expect(page.locator('[data-draft][data-round]')).toHaveCount(0);
    await expect(page.locator('.card-structure [data-card]').first()).toBeVisible();
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('a progress token from the board after a science pair (science-pair)', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'science-pair');
    const board = (await currentView(page)).boardTokens;
    expect(board.length).toBeGreaterThan(0);

    await playKey(driver, 'construct:library');
    const view = await currentView(page);
    expect(view.pending?.kind).toBe('pickBoardToken');
    expect(view.pending?.seat).toBe(0);
    // The board is untouched while the choice is open, and every token on it
    // is an option.
    expect(view.boardTokens).toEqual(board);
    expect([...(view.pending?.tokens ?? [])].sort()).toEqual([...board].sort());
    await expectPanel(page, 'pickBoardToken', 0);
    await expect(page.locator(`${host} [data-token-choice="board"]`)).toBeVisible();
    const keys = view.choices.map((c) => c.key);
    expect(keys.sort()).toEqual(board.map((t) => `token:${t}`).sort());
    expect((await panelPicks(page)).sort()).toEqual(keys.sort());
    await expect(driver.visible('[data-turn]')).toHaveAttribute('data-turn', 'pendingChoice');

    const token = board[0] ?? '';
    await playKey(driver, `token:${token}`);
    const after = await currentView(page);
    expect(after.players[0].tokens).toContain(token);
    expect(after.boardTokens).not.toContain(token);
    expect(after.boardTokens).toHaveLength(board.length - 1);
    // The token now shows in the player's own city in full colour; only a
    // token gone from the board is greyed.
    const owned = page
      .locator(`[data-seat="0"] [role="img"][data-owned]`)
      .filter({ visible: true });
    await expect(owned).toHaveCount(1);
    await expect(page.locator('[data-seat] [data-taken]')).toHaveCount(0);
    await expectTurnPassedTo(driver, 1);
    expect(problems).toEqual([]);
  });

  test('the Great Library offer shows to the decider only (great-library)', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'great-library');
    const board = (await currentView(page)).boardTokens;

    await playKey(driver, 'wonder:great-library:rostrum', 'wonderFirst');
    const view = await currentView(page);
    // The decider's own view lists the three drawn tokens, Law first, and
    // hides none of them from itself.
    expect(view.viewer).toBe(0);
    expect(view.pending?.kind).toBe('pickLibraryToken');
    expect(view.pending?.seat).toBe(0);
    expect(view.pending?.tokens).toEqual(['law', 'philosophy', 'strategy']);
    expect(view.pending?.hiddenTokens).toBe(0);
    expect(view.choices.map((c) => c.key)).toEqual([
      'token:law',
      'token:philosophy',
      'token:strategy'
    ]);
    await expectPanel(page, 'pickLibraryToken', 0);
    const panel = page.locator(`${host} [data-token-choice="library"]`);
    await expect(panel).toBeVisible();
    await expect(panel).toContainText('Only you can see these tokens.');
    await expect(panel.locator('[data-token]')).toHaveCount(3);
    expect(await panelPicks(page)).toEqual(['token:law', 'token:philosophy', 'token:strategy']);
    // The board's tokens are not part of the offer.
    expect(view.boardTokens).toEqual(board);

    await playKey(driver, 'token:law');
    const after = await currentView(page);
    expect(after.players[0].tokens).toEqual(['law']);
    expect(after.boardTokens).toEqual(board);
    // The two tokens returned to the box are hidden again from everyone:
    // the next view (seat 1's) names them nowhere, and neither does the page.
    expect(after.viewer).toBe(1);
    const json = JSON.stringify(after);
    expect(json).not.toContain('philosophy');
    expect(json).not.toContain('strategy');
    await expectTurnPassedTo(driver, 1);
    await expect(page.locator('[data-token="philosophy"], [data-token="strategy"]')).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test('reviving a card from the discard pile (mausoleum)', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'mausoleum');
    expect((await currentView(page)).discard).toEqual(['baths']);

    await playKey(driver, 'wonder:mausoleum:rostrum');
    const view = await currentView(page);
    expect(view.pending?.kind).toBe('buildFromDiscard');
    expect(view.pending?.cards).toEqual(['baths']);
    expect(view.choices.map((c) => c.key)).toEqual(['revive:baths']);
    await expectPanel(page, 'buildFromDiscard', 0);
    const panel = page.locator(`${host} [data-discard-choice]`);
    await expect(panel).toBeVisible();
    expect(await panelPicks(page)).toEqual(['revive:baths']);
    await expect(panel.locator('[data-act="revive:baths"]')).toContainText('Build Baths for free');

    const coins = view.players[0].coins;
    await playKey(driver, 'revive:baths');
    const after = await currentView(page);
    expect(after.players[0].city).toContain('baths');
    expect(after.players[0].coins).toBe(coins);
    expect(after.discard).toEqual([]);
    await expectTurnPassedTo(driver, 1);
    expect(problems).toEqual([]);
  });

  test('the Mausoleum with an empty discard pile skips its choice (mausoleum-empty)', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'mausoleum-empty');
    expect((await currentView(page)).discard).toEqual([]);

    await playKey(driver, 'wonder:mausoleum:rostrum', 'wonderFirst');
    const after = await currentView(page);
    expect(after.pending).toBeUndefined();
    expect(after.players[0].wonders.find((w) => w.wonder === 'mausoleum')?.built).toBe(true);
    const skip = 'Nothing to build for Mausoleum: the discard pile is empty.';
    await expectSkipShown(page, skip);
    await expectTurnPassedTo(driver, 1);
    expect(await logLines(driver)).toContain(skip);
    expect(problems).toEqual([]);
  });

  for (const c of [
    {
      scenario: 'zeus-destroy',
      wonder: 'statue-of-zeus',
      card: 'quarry',
      color: 'brown',
      name: 'Quarry'
    },
    {
      scenario: 'circus-maximus',
      wonder: 'circus-maximus',
      card: 'glassworks',
      color: 'grey',
      name: 'Glassworks'
    }
  ] as const) {
    test(`destroying the opponent's ${c.color} card (${c.scenario})`, async ({ page }) => {
      const problems = watchHygiene(page);
      const driver = await openHotseatRoom(page, c.scenario);
      expect((await currentView(page)).players[1].city).toEqual([c.card]);

      await playKey(driver, `wonder:${c.wonder}:rostrum`);
      const view = await currentView(page);
      expect(view.pending?.kind).toBe('destroyCard');
      expect(view.pending?.seat).toBe(0);
      expect(view.pending?.color).toBe(c.color);
      expect(view.pending?.cards).toEqual([c.card]);
      expect(view.choices.map((ch) => ch.key)).toEqual([`destroy:${c.card}`]);
      await expectPanel(page, 'destroyCard', 0);
      const panel = page.locator(`${host} [data-destroy-choice]`);
      await expect(panel).toBeVisible();
      expect(await panelPicks(page)).toEqual([`destroy:${c.card}`]);
      await expect(panel.locator(`[data-act="destroy:${c.card}"]`)).toContainText(
        `Destroy ${c.name}`
      );

      await playKey(driver, `destroy:${c.card}`);
      const after = await currentView(page);
      expect(after.players[1].city).toEqual([]);
      expect(after.discard).toContain(c.card);
      await expectTurnPassedTo(driver, 1);
      expect(problems).toEqual([]);
    });
  }

  test('the Statue of Zeus with nothing to destroy skips its choice (zeus-no-target)', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'zeus-no-target');
    expect((await currentView(page)).players[1].city).toEqual([]);

    await playKey(driver, 'wonder:statue-of-zeus:rostrum', 'wonderFirst');
    const after = await currentView(page);
    expect(after.pending).toBeUndefined();
    expect(after.players[0].wonders.find((w) => w.wonder === 'statue-of-zeus')?.built).toBe(true);
    const skip = 'Nothing to destroy for Statue of Zeus: Moon has no card of that colour.';
    await expectSkipShown(page, skip);
    await expectTurnPassedTo(driver, 1);
    expect(await logLines(driver)).toContain(skip);
    expect(problems).toEqual([]);
  });

  test("the trailing seat chooses the next age's starter (age-transition-chooser)", async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'age-transition-chooser');
    const before = await currentView(page);
    expect(before.age).toBe(1);
    expect(before.military.position).toBe(3);

    await playKey(driver, 'discard:theater');
    const snap = await driver.nextTurn();
    const view = await currentView(page);
    // Age II is dealt, and seat 1 (the pawn leans toward its capital)
    // chooses, not seat 0 who took the last card.
    expect(view.age).toBe(2);
    expect(view.pending?.kind).toBe('chooseStarter');
    expect(view.pending?.seat).toBe(1);
    expect(snap.toMove).toBe(1);
    expect(snap.choices.map((c) => c.key).sort()).toEqual(['starter:0', 'starter:1']);
    await expectPanel(page, 'chooseStarter', 1);
    const choice = page.locator(`${host} [data-starter-choice]`);
    await expect(choice).toBeVisible();
    await expect(choice.locator('[data-act]')).toHaveCount(2);
    await expect(choice).toContainText('Who starts Age 2?');
    await expect(choice).toContainText('Moon is behind on the military track, so Moon chooses.');

    await playKey(driver, 'starter:0');
    const after = await currentView(page);
    expect(after.pending).toBeUndefined();
    expect(after.phase).toBe('play');
    expect(after.age).toBe(2);
    expect(after.active).toBe(0);
    await expectTurnPassedTo(driver, 0);
    expect(await logLines(driver)).toContain('Moon chooses Sun to start the next age.');
    expect(problems).toEqual([]);
  });

  test("an extra turn on an age's last card is lost (extra-turn-lost)", async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'extra-turn-lost');

    await playKey(driver, 'wonder:sphinx:theater');
    const snap = await driver.nextTurn();
    const view = await currentView(page);
    // The Sphinx is built, its extra turn is lost with the age, and seat 0
    // (the last active player, the pawn centred) chooses Age II's starter.
    expect(view.players[0].wonders.find((w) => w.wonder === 'sphinx')?.built).toBe(true);
    expect(view.age).toBe(2);
    expect(view.extraTurn).toBe(false);
    expect(view.pending?.kind).toBe('chooseStarter');
    expect(view.pending?.seat).toBe(0);
    expect(snap.toMove).toBe(0);
    await expectPanel(page, 'chooseStarter', 0);
    await expect(page.locator(`${host} [data-starter-choice]`)).toContainText(
      'The military track is even, so Sun, who played last, chooses.'
    );

    await playKey(driver, 'starter:1');
    expect((await currentView(page)).active).toBe(1);
    await expectTurnPassedTo(driver, 1);
    const log = await logLines(driver);
    expect(log).toContain("Sun's extra turn is lost at the end of the age.");
    expect(problems).toEqual([]);
  });

  test('a wonder that grants an extra turn (appian-economy)', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, 'appian-economy');

    await playKey(driver, 'wonder:appian-way:rostrum');
    const snap = await driver.nextTurn();
    const view = await currentView(page);
    // Seat 0 plays again: the move is seat 0's once more, with no choice in
    // between, and the log says why. (The view's extraTurn flag is already
    // spent by now: it marks a turn that will be followed by another, so it
    // is not checked here.)
    expect(view.players[0].wonders.find((w) => w.wonder === 'appian-way')?.built).toBe(true);
    expect(snap.toMove).toBe(0);
    expect(snap.pending).toBeNull();
    expect(view.active).toBe(0);
    // The opponent's Economy took the 10 trade coins before the Appian Way's
    // loss of 3 (the scenario's Expect).
    expect(view.players[1].coins).toBe(7);
    await expect(page.locator(host)).toHaveCount(0);
    // The turn bar says so for the whole extra turn, naming the wonder.
    const bar = driver.visible('[data-turn]');
    await expect(bar).toHaveAttribute('data-turn', 'playAgain');
    await expect(bar).toContainText('Sun plays again (Appian Way)');
    expect(await logLines(driver)).toContain('Sun plays again.');
    await expect(driver.visible('[data-turn]')).toHaveAttribute('data-turn', 'playAgain');

    // The extra turn is an ordinary move; after it the turn passes.
    const discard = snap.choices.find((c) => c.kind === 'discard');
    if (!discard) throw new Error('no discard to spend the extra turn on');
    await playKey(driver, discard.key);
    const after = await currentView(page);
    expect(after.extraTurn).toBe(false);
    await expectTurnPassedTo(driver, 1);
    await expect(driver.visible('[data-turn]')).toHaveAttribute('data-turn', 'hotseatTurn');
    expect(problems).toEqual([]);
  });
});
