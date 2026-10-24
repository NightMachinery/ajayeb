// The table screen at five seats over the real server: one human and four
// bots play the first turns of Age I through the screen alone (a hand card,
// its card sheet, Confirm), changing a pick on the way. The bots confirm at
// once, so this seat's confirmation is always the last and resolves the
// turn; withdrawing one needs a second human (haftgane-table.full.spec.ts).
// Checked as it goes:
//
// - every table frame the page received passes the pending-ownership
//   oracle, frame by frame, after every turn (and the oracle catches a
//   contaminated frame);
// - the board shows each seat once: the viewer's strip, two neighbour
//   panels and two far chips, each with its status;
// - the hand card of the pending pick carries the drafted
//   pick tag;
// - every glyph and image on the board sits inside a tooltip anchor, or in
//   a button whose visible label already says what it is, and a trade
//   chip's tooltip names its price;
// - nothing animates with Animation off, and no console error or failed
//   request happens.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import {
  choose,
  contaminate,
  expectPending,
  latest,
  openBotTable,
  pendingOwnership,
  ready,
  tableStates,
  traffic
} from '../fixtures/table';

/** Board glyphs and images with no tooltip of their own or of an ancestor. */
async function untipped(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const root = document.querySelector('[data-table-screen]');
    if (!root) return ['no table screen'];
    const out: string[] = [];
    for (const el of root.querySelectorAll('svg, [role="img"]')) {
      if (el.parentElement?.closest('svg')) continue;
      if (!el.getClientRects().length) continue;
      if (el.closest('[data-tooltip]')) continue;
      const button = el.closest('button');
      const label = button && [...button.childNodes].some((n) => n.textContent?.trim());
      if (label) continue;
      const host = el.closest('[class]');
      out.push(`${el.tagName.toLowerCase()} in ${host?.getAttribute('class') ?? '?'}`);
    }
    return out;
  });
}

async function running(page: Page): Promise<number> {
  return page.evaluate(
    () =>
      document.getAnimations().filter((a) => {
        const timing = a.effect?.getComputedTiming();
        return (
          a.playState === 'running' && typeof timing?.duration === 'number' && timing.duration > 1
        );
      }).length
  );
}

/** Waits for the point after `ref` to be on screen. */
async function nextPoint(page: Page, ref: { id: string; revision: number }): Promise<void> {
  await expect
    .poll(async () => {
      const f = await latest(page);
      return f.point?.ref.id !== ref.id || f.point.ref.revision !== ref.revision;
    })
    .toBe(true);
}

test.describe('Haftgane table screen', { tag: ['@full', '@haftgane-table'] }, () => {
  test('five seats play their first turns through the screen', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Ana'));
    const problems = watchHygiene(page);
    const log = traffic(page);
    await openBotTable(page, 5, 'Ana');

    const start = await latest(page);
    expect(start.view.seats).toHaveLength(5);
    const me = start.you.seat!;
    for (const s of start.view.seats) {
      await expect(page.locator(`[data-seat="${s.seat}"]`)).toHaveCount(1);
    }
    await expect(page.locator(`[data-seat="${me}"]`)).toHaveAttribute('data-relation', 'self');
    await expect(page.locator('[data-neighbour]')).toHaveCount(2);
    await expect(page.locator('[data-far-count]')).toHaveAttribute('data-far-count', '2');
    expect(await untipped(page)).toEqual([]);

    // A side point, when the table lets each seat choose: its choices need
    // no hand card, so the pick list shows them at once.
    if (start.view.phase === 'sides') {
      expect(start.view.you!.choices.map((c) => c.key)).toEqual(['side:day', 'side:night']);
      await choose(page, 'side:day');
      await expectPending(page, 'side:day', false);
      await choose(page, 'side:night');
      await expectPending(page, 'side:night', false);
      await choose(page, 'confirm');
      await nextPoint(page, start.point!.ref);
      expect(pendingOwnership('the page', log)).toEqual([]);
    }

    // Three turns: pick a card through the hand, once changing the pick.
    for (let turn = 1; turn <= 3; turn++) {
      const f = await latest(page);
      expect(f.view.phase).toBe('turn');
      expect(f.view.turn).toBe(turn);
      const choices = f.view.you!.choices;
      const builds = choices.filter((c) => c.kind === 'build' || c.kind === 'sell');
      const first = builds[0]!;
      const other = builds.find((c) => c.design !== first.design) ?? first;
      if (turn === 2) {
        await choose(page, first.key);
        await expectPending(page, first.key, false);
      }
      const pick = turn === 2 ? other : first;
      await choose(page, pick.key);
      await expectPending(page, pick.key, false);
      const card = page.locator(`[data-act="hand:${pick.design}"]`).first();
      await expect(card).toHaveAttribute('data-picked', 'draft');
      // Closing the sheet leaves the server draft tagged, with no card sheet selected.
      await expect(card).toHaveAttribute('aria-pressed', 'false');
      if (turn === 1) {
        // A trade chip's tooltip on a fine pointer: hover shows the price.
        const chip = page.locator('[data-trade]').first();
        if (await chip.isVisible()) {
          await chip.hover();
          const bubble = page.locator('[data-tooltip-layer] [data-tooltip-bubble]');
          await expect(bubble).toBeVisible();
          await expect(bubble).toContainText(String(await chip.getAttribute('data-price')));
          await page.mouse.move(0, 0);
        }
        expect(await untipped(page)).toEqual([]);
      }
      await choose(page, 'confirm');
      await nextPoint(page, f.point!.ref);
      await ready(page);
      expect(await running(page)).toBe(0);
      expect(pendingOwnership('the page', log), `after turn ${turn}`).toEqual([]);
    }

    const end = await latest(page);
    expect(end.view.turn).toBe(4);

    // The oracle bites: a frame that shows a pick the page never sent, or
    // its own pick with the wrong confirmation, is reported.
    const states = tableStates(log);
    const own = states.find((i) => log[i]!.msg.pending);
    expect(own, 'a frame with the page own pending pick').toBeDefined();
    const mine = log[own!]!.msg.pending as { key: string; confirm: boolean };
    for (const canary of [
      contaminate(log, own!, { key: 'sell:no-such-card', confirm: false }),
      contaminate(log, own!, { key: mine.key, confirm: !mine.confirm }),
      contaminate(log, states.at(-1)!, { key: mine.key, confirm: true })
    ]) {
      expect(pendingOwnership('canary', canary)).not.toEqual([]);
    }
    expect(problems).toEqual([]);
  });
});
