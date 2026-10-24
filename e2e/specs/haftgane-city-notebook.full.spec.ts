// The city sheet and the Seen hands sheet over the real server: one human
// and four bots, started from Home's lobby, play a few turns (building a
// wonder stage when one is offered, so the human's wonder holds a hidden
// card); then the page opens Seen hands from the hand header and every
// city from a far chip, moving round the ring tabs. Checked as it goes:
//
// - the ring tabs list the seats from the viewer round the table;
// - each city lists exactly the buildings the view gives that seat, its
//   stages built and its points by category;
// - a city other than the viewer's own shows nothing private: no card
//   element but its own buildings, no line about the cards under the
//   viewer's wonder, and no name of a card under that wonder, kept or in
//   hand (the leak oracle);
// - the viewer's own city names the cards under its wonder and its kept card;
// - the notebook shows exactly this viewer's memory: each hand and pile
//   look the view's notebook holds, the card played from each hand, the
//   statistics as the view bounds them, and no other card; a figure
//   filters the list and Show all brings it back;
// - every table frame passes the pending-ownership oracle, and no frame
//   gives a seat a private field or the page another seat's private view;
// - no console error or failed request happens.
import { readFileSync } from 'node:fs';
import { expect, type Page } from '@playwright/test';
import type { Seat } from '../../web/src/lib/wire/haftgane.gen';
import type { TableState, TableView } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import {
  choose,
  latest,
  openBotTable,
  pendingOwnership,
  ready,
  tableStates,
  traffic,
  type Traffic
} from '../fixtures/table';

/** The English card names, read from the names table as text. */
const CARD_NAMES: ReadonlyMap<string, string> = new Map(
  [
    ...readFileSync(
      new URL('../../web/src/lib/games/haftgane/i18n/en/names.ts', import.meta.url),
      'utf8'
    ).matchAll(/'hg\.card\.([a-z0-9-]+)':\s*(['"])(.*?)\2/g)
  ].map((m) => [m[1]!, m[3]!])
);

/** The fields a public seat may carry. */
const PUBLIC_SEAT = new Set([
  'seat',
  'board',
  'side',
  'stages',
  'built',
  'coins',
  'victories',
  'defeats',
  'shields',
  'handSize',
  'hasKept',
  'relative',
  'relation',
  'last',
  'score',
  'science'
]);

/** Frames that give a seat a private field, or the page another seat's private view. */
function seatLeaks(log: readonly Traffic[]): string[] {
  const out: string[] = [];
  for (const i of tableStates(log)) {
    const f = log[i]!.msg as unknown as TableState;
    for (const s of f.view.seats) {
      const extra = Object.keys(s).filter((k) => !PUBLIC_SEAT.has(k));
      if (extra.length) out.push(`message ${i}: seat ${s.seat} carries ${extra.join(', ')}`);
    }
    if (f.view.you && f.view.you.seat !== f.you.seat)
      out.push(`message ${i}: private view of seat ${f.view.you.seat}`);
  }
  return out;
}

const MINUS = '−';
const signed = (n: number) => (n < 0 ? MINUS + String(-n) : String(n));

async function cardsIn(page: Page, selector: string): Promise<string[]> {
  return page
    .locator(`${selector} [data-card]`)
    .evaluateAll((els) => els.map((e) => e.getAttribute('data-card') ?? '').sort());
}

/** The open city sheet against the view's facts for that seat. */
async function checkCity(page: Page, view: TableView, seat: Seat): Promise<void> {
  const sheet = page.locator(`[data-city-sheet="${seat.seat}"]`);
  await expect(sheet).toBeVisible();
  const own = seat.relation === 'self';
  await expect(sheet.locator('[data-city]')).toHaveAttribute('data-city-own', String(own));
  expect(await cardsIn(page, '[data-city-sheet] [data-city-rows]')).toEqual([...seat.built].sort());
  // No card element in the sheet outside the city's own buildings.
  expect(await cardsIn(page, '[data-city-sheet]')).toEqual([...seat.built].sort());
  await expect(sheet.locator('[data-stage-built="true"]')).toHaveCount(seat.stages);
  for (const c of ['wonder', 'treasure', 'military', 'blue', 'yellow', 'green', 'purple'] as const)
    await expect(sheet.locator(`[data-score="${c}"] .value`)).toHaveText(signed(seat.score[c]));
  const you = view.you!;
  const name = (slug: string) => CARD_NAMES.get(slug) ?? slug;
  if (own) {
    if (you.tucked.length > 0) {
      const under = sheet.locator('[data-under="own"]');
      for (const slug of you.tucked) await expect(under).toContainText(name(slug));
    }
    if (you.kept) await expect(sheet.locator('[data-city-kept]')).toContainText(name(you.kept));
    return;
  }
  await expect(sheet.locator('[data-under]')).toHaveCount(0);
  await expect(sheet.locator('[data-city-kept]')).toHaveCount(seat.hasKept ? 1 : 0);
  // The leak oracle: no private card's name, unless the city shows a
  // building of that name (or one that contains it) in its own right.
  const text = (await sheet.textContent()) ?? '';
  const publicNames = seat.built.map(name);
  const privateCards = [...you.tucked, ...(you.kept ? [you.kept] : []), ...you.hand];
  for (const slug of privateCards) {
    const n = name(slug);
    if (publicNames.some((p) => p.includes(n))) continue;
    expect(text, `${slug} named in the city of seat ${seat.seat}`).not.toContain(n);
  }
}

test.describe('Haftgane city sheet and Seen hands', { tag: ['@full', '@haftgane-table'] }, () => {
  test('five seats: every city round the ring, and the notebook', async ({ page }) => {
    test.setTimeout(180_000);
    expect(CARD_NAMES.size).toBeGreaterThan(70);
    const problems = watchHygiene(page);
    const log = traffic(page);
    await openBotTable(page, 5, 'Ana');

    // The notebook starts empty.
    await page.locator('[data-act="seen-hands"]').click();
    await expect(page.locator('[data-seen-sheet] [data-seen-empty]')).toBeVisible();
    await page.locator('[data-act="seen-close"]').click();
    await expect(page.locator('[data-seen-sheet]')).toHaveCount(0);

    const start = await latest(page);
    if (start.view.phase === 'sides') {
      await choose(page, 'side:day');
      await choose(page, 'confirm');
      await expect.poll(async () => (await latest(page)).view.phase).toBe('turn');
    }

    // A few turns, building a wonder stage when one is offered.
    for (let turn = 1; turn <= 3; turn++) {
      const f = await latest(page);
      expect(f.view.phase).toBe('turn');
      const choices = f.view.you!.choices;
      const pick =
        choices.find((c) => c.kind === 'stage') ??
        choices.find((c) => c.kind === 'build') ??
        choices[0]!;
      await choose(page, pick.key);
      await choose(page, 'confirm');
      await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(f.point!.ref.id);
    }

    const view = (await latest(page)).view;
    const nb = view.you!.notebook;
    expect(nb.seen.length).toBeGreaterThan(0);

    // Seen hands: exactly this viewer's notebook.
    await page.locator('[data-act="seen-hands"]').click();
    const seen = page.locator('[data-seen-sheet]');
    await expect(seen).toBeVisible();
    for (const entry of nb.seen) {
      const item = seen.locator(`[data-seen-point="${entry.point}"][data-seen-kind="hand"]`);
      await expect(item).toBeVisible();
      expect(
        await cardsIn(page, `[data-seen-point="${entry.point}"][data-seen-kind="hand"]`)
      ).toEqual([...entry.hand].sort());
      await expect(item.locator('[data-played]')).toHaveAttribute('data-card', entry.played);
    }
    await expect(seen.locator('[data-seen-kind="hand"]')).toHaveCount(nb.seen.length);
    await expect(seen.locator('[data-seen-kind="pile"]')).toHaveCount(nb.pileLooks.length);
    const allowed = new Set([
      ...nb.seen.flatMap((e) => e.hand),
      ...nb.pileLooks.flatMap((l) => l.cards.map((c) => c.design)),
      ...nb.statistics.returning.map((r) => r.design),
      ...nb.statistics.neighbourPicks.map((p) => p.design)
    ]);
    for (const slug of await cardsIn(page, '[data-seen-sheet]')) expect(allowed).toContain(slug);
    const circulating = nb.statistics.circulating.filter((c) => c.bounds.atMost > 0);
    await expect(seen.locator('[data-circulating]')).toHaveCount(circulating.length);
    for (const c of circulating) {
      const fig = seen.locator(`[data-circulating="${c.colour}"]`);
      await expect(fig).toHaveAttribute('data-certain', String(c.bounds.certain));
      await expect(fig).toHaveAttribute('data-at-most', String(c.bounds.atMost));
    }
    // A figure filters the list to the entries holding its colour: each
    // entry left shows a card in full, the others faint.
    if (circulating.length > 0) {
      await seen.locator(`[data-circulating="${circulating[0]!.colour}"]`).click();
      await expect(seen.locator('[data-seen-filter]')).toBeVisible();
      const left = seen.locator('[data-seen-point]');
      for (let i = 0; i < (await left.count()); i++)
        await expect(left.nth(i).locator('[data-card]:not(.dim)').first()).toBeVisible();
      await seen.locator('[data-act="seen-filter-clear"]').click();
      await expect(seen.locator('[data-seen-filter]')).toHaveCount(0);
      await expect(seen.locator('[data-seen-kind="hand"]')).toHaveCount(
        nb.seen.filter((e) => e.point.startsWith(`age-${view.age}-`)).length
      );
    }
    await page.locator('[data-act="seen-close"]').click();
    await expect(seen).toHaveCount(0);

    // Every city, from a far chip round the ring tabs.
    const bySeat = new Map(view.seats.map((s) => [s.seat, s]));
    const order = [...view.seats].sort((a, b) => a.relative - b.relative);
    const far = order.find((s) => s.relation === 'other')!;
    await page.locator(`[data-act="city:${far.seat}"]`).click();
    await checkCity(page, view, far);
    const tabs = await page
      .locator('[data-city-sheet] [data-seat-tab]')
      .evaluateAll((els) => els.map((e) => Number(e.getAttribute('data-seat-tab'))));
    expect(tabs).toEqual(order.map((s) => s.seat));
    for (const seat of tabs) {
      const tab = page.locator(`[data-act="city-tab:${seat}"]`);
      await tab.click();
      await expect(tab).toHaveAttribute('aria-selected', 'true');
      await checkCity(page, view, bySeat.get(seat)!);
    }
    // The arrow keys move round the ring too.
    await page.locator(`[data-act="city-tab:${tabs.at(-1)}"]`).focus();
    await page.keyboard.press('ArrowRight');
    await expect(page.locator(`[data-act="city-tab:${tabs[0]}"]`)).toHaveAttribute(
      'aria-selected',
      'true'
    );
    await page.locator('[data-act="city-close"]').click();
    await expect(page.locator('[data-city-sheet]')).toHaveCount(0);

    // The neighbour panel and the viewer's strip open their cities too.
    for (const seat of [order.find((s) => s.relation === 'left')!, order[0]!]) {
      await page.locator(`[data-act="city:${seat.seat}"]`).first().click();
      await checkCity(page, view, seat);
      await page.locator('[data-act="city-close"]').click();
      await expect(page.locator('[data-city-sheet]')).toHaveCount(0);
    }
    await ready(page);

    expect(pendingOwnership('the page', log)).toEqual([]);
    expect(seatLeaks(log)).toEqual([]);
    expect(problems).toEqual([]);
  });
});
